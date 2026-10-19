# Installation

Engineering Toolkit `v0.1.0` is distributed as raw executables. Starting with
`v0.1.1`, each release also includes user-scoped PowerShell and POSIX bootstrap
installers. Package-manager distribution, automatic updates, self-replacement,
and production TUF activation are not included. Download release assets only
from the official
[GitHub Releases](https://github.com/SummitExplorerJD/Engineering-Toolkit/releases)
page and verify the selected release identity.

## Bootstrap installers (`v0.1.1` and later)

The bootstrap scripts require an exact version and never resolve `latest`.
They detect the operating system and architecture, download the matching
executable and `SHA256SUMS`, require exactly one checksum entry, verify SHA-256,
confirm the staged executable's version, and publish it through a recoverable
user-scoped transaction.

For Windows, download `install.ps1` from the exact release and run:

```powershell
$version = 'v0.1.1'
Invoke-WebRequest -UseBasicParsing `
    -Uri "https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/$version/install.ps1" `
    -OutFile '.\install.ps1'
.\install.ps1 -Version $version -AddToPath
```

The default destination is
`%LOCALAPPDATA%\Programs\engkit\engkit.exe`. Close and reopen the terminal after
changing the user `PATH`.

For Linux or macOS, download `install.sh` from the exact release and run:

```sh
version=v0.1.1
curl --fail --location --proto '=https' --tlsv1.2 \
  --output install.sh \
  "https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/${version}/install.sh"
sh install.sh --version "$version" --add-to-path
```

The default destination is `$HOME/.local/bin/engkit`. With `--add-to-path`, the
script adds one marked entry to `$HOME/.zprofile` for zsh or `$HOME/.profile`
for other shells. The change is idempotent and takes effect in a new shell.

An existing executable is never replaced implicitly. Upgrade with:

```powershell
.\install.ps1 -Version v0.1.1 -Upgrade
```

```sh
sh install.sh --version v0.1.1 --upgrade
```

Use `-AddToPath` or `--add-to-path` during an upgrade only when the installer
should also ensure its directory is present in the user environment.

To install from a previously downloaded and reviewed release directory that
contains the platform executables and `SHA256SUMS`, use:

```powershell
.\install.ps1 -Version v0.1.1 -AssetDirectory 'C:\path\to\release'
```

```sh
sh install.sh --version v0.1.1 --asset-directory /path/to/release
```

A local checksum detects corruption but does not independently prove who
created the local files. Preserve and verify the origin of offline assets.

The network installer uses a fixed GitHub Releases URL. In `v0.1.1`, HTTPS and
GitHub repository controls authenticate the bootstrap download; the checksum
from the same release is an integrity check, not an independent signature.

## Manual installation (`v0.1.0` and fallback)

## Choose an architecture

Use `amd64` for 64-bit Intel or AMD processors. Use `arm64` for Windows on ARM,
Linux AArch64, and Apple silicon. The release does not support 32-bit systems.

## Windows

Download `SHA256SUMS` and one of these files from the release:

- `engkit-v0.1.0-windows-amd64.exe`
- `engkit-v0.1.0-windows-arm64.exe`

Open PowerShell in the download directory and verify the selected file:

```powershell
$file = 'engkit-v0.1.0-windows-amd64.exe'
$expected = ((Select-String -Path '.\SHA256SUMS' -Pattern "  $([regex]::Escape($file))$").Line -split '\s+')[0]
$actual = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $expected) {
    throw "SHA-256 mismatch for $file"
}
```

Install the executable for the current user and add its directory to the user
`PATH`:

```powershell
$installDir = Join-Path $env:LOCALAPPDATA 'Programs\engkit'
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Copy-Item -LiteralPath $file -Destination (Join-Path $installDir 'engkit.exe')

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$pathEntries = @($userPath -split ';' | Where-Object { $_ })
if ($pathEntries -notcontains $installDir) {
    $newPath = (@($pathEntries) + $installDir) -join ';'
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
}
```

Close and reopen PowerShell, then validate the installation:

```powershell
engkit version
engkit --help
```

The Windows binaries are not Authenticode-signed in `v0.1.0`. Confirm the
checksum and release origin before responding to any operating-system warning.

## Linux

Run the following commands from a temporary download directory:

```sh
version=v0.1.0
case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

file="engkit-${version}-linux-${arch}"
base_url="https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/${version}"
curl --fail --location --remote-name "${base_url}/${file}"
curl --fail --location --remote-name "${base_url}/SHA256SUMS"
grep "  ${file}$" SHA256SUMS | sha256sum --check -

install -d "$HOME/.local/bin"
install -m 0755 "$file" "$HOME/.local/bin/engkit"
```

Add `$HOME/.local/bin` to `PATH` in the appropriate shell profile if it is not
already present. Open a new shell and validate the installation:

```sh
engkit version
engkit --help
```

## macOS

Run the following commands from a temporary download directory:

```sh
version=v0.1.0
case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

file="engkit-${version}-darwin-${arch}"
base_url="https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/${version}"
curl --fail --location --remote-name "${base_url}/${file}"
curl --fail --location --remote-name "${base_url}/SHA256SUMS"
grep "  ${file}$" SHA256SUMS | shasum -a 256 --check -

install -d "$HOME/.local/bin"
install -m 0755 "$file" "$HOME/.local/bin/engkit"
```

Add `$HOME/.local/bin` to `PATH` in the appropriate shell profile if needed,
then open a new shell and run:

```sh
engkit version
engkit --help
```

The macOS binaries are not signed or notarized in `v0.1.0`. Confirm the
checksum and release origin, and use the standard macOS security settings when
the operating system requests approval. Do not disable system-wide security
controls.

## Manual upgrade

Until production update support is activated, upgrades are manual: download
the newer executable, verify its checksum, and replace the installed
`engkit`/`engkit.exe` file. Do not rename a prerelease executable to a stable
version.

## Uninstall

Remove the installed executable:

```powershell
Remove-Item -LiteralPath (Join-Path $env:LOCALAPPDATA 'Programs\engkit\engkit.exe')
```

On Linux or macOS:

```sh
rm "$HOME/.local/bin/engkit"
```

The optional per-user update state and cache are separate from project
workspaces. Removing the executable does not delete project `.engkit`
directories or their canonical artifacts.
