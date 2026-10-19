#!/bin/sh

set -eu

version=''
install_dir=''
upgrade=0
add_to_path=0
asset_directory=''
profile_new=''

usage() {
    cat <<'EOF'
Usage: install.sh --version VERSION [--install-dir PATH] [--upgrade] [--add-to-path] [--asset-directory PATH]

Installs an exact Engineering Toolkit release for the current user. The
--asset-directory option installs a previously downloaded release directory;
its SHA256SUMS file must be present and is verified before execution.
EOF
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --version)
            [ "$#" -ge 2 ] || { echo 'Missing value for --version' >&2; exit 2; }
            version=$2
            shift 2
            ;;
        --install-dir)
            [ "$#" -ge 2 ] || { echo 'Missing value for --install-dir' >&2; exit 2; }
            install_dir=$2
            shift 2
            ;;
        --upgrade)
            upgrade=1
            shift
            ;;
        --add-to-path)
            add_to_path=1
            shift
            ;;
        --asset-directory)
            [ "$#" -ge 2 ] || { echo 'Missing value for --asset-directory' >&2; exit 2; }
            asset_directory=$2
            shift 2
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            usage >&2
            exit 2
            ;;
    esac
done

[ -n "$version" ] || { echo '--version is required' >&2; exit 2; }
printf '%s\n' "$version" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.[1-9][0-9]*)?$' || {
    echo "Invalid version: $version" >&2
    exit 2
}

case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) echo "Unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [ -z "$install_dir" ]; then
    : "${HOME:?HOME is unavailable; specify --install-dir explicitly}"
    install_dir=$HOME/.local/bin
fi
if [ "$add_to_path" -eq 1 ] && printf '%s' "$install_dir" | grep '[[:cntrl:]]' >/dev/null 2>&1; then
    echo 'Install directory cannot contain control characters when --add-to-path is used' >&2
    exit 1
fi

filename=engkit-$version-$os-$arch
target=$install_dir/engkit
target_exists=0
if [ -d "$target" ]; then
    echo "Installation target is a directory: $target" >&2
    exit 1
fi
if [ -f "$target" ] && [ "$upgrade" -ne 1 ]; then
    echo "Engineering Toolkit is already installed at $target; use --upgrade to replace it explicitly" >&2
    exit 1
fi
if [ -f "$target" ]; then
    target_exists=1
fi

mkdir -p "$install_dir"
transaction_directory=$(mktemp -d "$install_dir/.engkit-install.XXXXXX")
staged_binary=$transaction_directory/$filename
staged_checksums=$transaction_directory/SHA256SUMS
backup=$transaction_directory/engkit.previous
committed=0

cleanup() {
    if [ "$committed" -eq 1 ] && [ -f "$backup" ]; then
        rm -f "$backup"
    fi
    if [ -n "$profile_new" ]; then
        rm -f "$profile_new"
    fi
    if [ "$committed" -eq 0 ] && [ -f "$backup" ]; then
        printf 'Automatic rollback could not complete; the previous executable is preserved at %s\n' "$backup" >&2
    else
        rm -rf "$transaction_directory"
    fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

if [ -n "$asset_directory" ]; then
    cp "$asset_directory/$filename" "$staged_binary"
    cp "$asset_directory/SHA256SUMS" "$staged_checksums"
else
    base_url=https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/$version
    curl --fail --location --proto '=https' --tlsv1.2 --output "$staged_binary" "$base_url/$filename"
    curl --fail --location --proto '=https' --tlsv1.2 --output "$staged_checksums" "$base_url/SHA256SUMS"
fi

checksum_entries=$(
    awk -v wanted="$filename" '
        {
            name = $2
            sub(/\r$/, "", name)
        }
        name == wanted && length($1) == 64 && $1 ~ /^[0-9a-fA-F]+$/ {
            count++
            hash = tolower($1)
        }
        END {
            if (count == 1) print hash
        }
    ' "$staged_checksums"
)
[ -n "$checksum_entries" ] || {
    echo "SHA256SUMS must contain exactly one entry for $filename" >&2
    exit 1
}

if command -v sha256sum >/dev/null 2>&1; then
    actual_hash=$(sha256sum "$staged_binary" | awk '{ print tolower($1) }')
elif command -v shasum >/dev/null 2>&1; then
    actual_hash=$(shasum -a 256 "$staged_binary" | awk '{ print tolower($1) }')
else
    echo 'Neither sha256sum nor shasum is available' >&2
    exit 1
fi
[ "$actual_hash" = "$checksum_entries" ] || {
    echo "SHA-256 mismatch for $filename" >&2
    exit 1
}

chmod 0755 "$staged_binary"
reported_version=$($staged_binary version)
[ "$reported_version" = "engkit $version" ] || {
    echo "Downloaded executable reported '$reported_version', want 'engkit $version'" >&2
    exit 1
}

if [ -f "$target" ]; then
    mv "$target" "$backup"
fi
if [ -n "$asset_directory" ] && [ "${ENGKIT_INSTALL_TEST_FAIL_AFTER_BACKUP:-}" = 1 ]; then
    install_error='Injected installer failure after backup'
elif mv "$staged_binary" "$target"; then
    committed=1
    install_error=''
else
    install_error="Could not publish $target"
fi

if [ -n "$install_error" ]; then
    if [ "$target_exists" -eq 1 ]; then
        rm -f "$target"
        if [ -f "$backup" ]; then
            mv "$backup" "$target"
        fi
    fi
    echo "$install_error" >&2
    exit 1
fi

if [ "$add_to_path" -eq 1 ]; then
    shell_name=$(basename "${SHELL:-sh}")
    if [ "$shell_name" = zsh ]; then
        profile=$HOME/.zprofile
    else
        profile=$HOME/.profile
    fi
    if [ -n "$asset_directory" ] && [ -n "${ENGKIT_INSTALL_TEST_PROFILE:-}" ]; then
        profile=$ENGKIT_INSTALL_TEST_PROFILE
    fi
    marker='# Added by Engineering Toolkit installer'
    if [ -f "$profile" ] && grep -Fqx "$marker" "$profile"; then
        grep -Fq "$install_dir" "$profile" || {
            echo "Existing Engineering Toolkit PATH entry does not reference $install_dir: $profile" >&2
            exit 1
        }
    else
        profile_directory=$(dirname "$profile")
        mkdir -p "$profile_directory"
        escaped_install_dir=$(printf '%s' "$install_dir" | sed "s/'/'\\\\''/g")
        profile_new=$profile.engkit-new
        {
            if [ -f "$profile" ]; then
                cat "$profile"
                printf '\n'
            fi
            printf '%s\n' "$marker"
            printf "export PATH='%s':\"\$PATH\"\n" "$escaped_install_dir"
        } > "$profile_new"
        mv "$profile_new" "$profile"
        profile_new=''
    fi
fi

printf 'Engineering Toolkit %s installed at %s\n' "$version" "$target"
if [ "$add_to_path" -ne 1 ]; then
    printf "The installer did not modify PATH. Add '%s' explicitly if needed.\n" "$install_dir"
fi

