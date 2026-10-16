package update

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"golang.org/x/mod/semver"
)

var supportedPlatforms = map[Platform]struct{}{
	{OS: "windows", Arch: "amd64"}: {},
	{OS: "windows", Arch: "arm64"}: {},
	{OS: "linux", Arch: "amd64"}:   {},
	{OS: "linux", Arch: "arm64"}:   {},
	{OS: "darwin", Arch: "amd64"}:  {},
	{OS: "darwin", Arch: "arm64"}:  {},
}

func ValidateRepository(repository Repository) error {
	if len(repository.Root) == 0 {
		return errors.New("trusted root is required")
	}
	if len(repository.Root) > 512*1024 {
		return errors.New("trusted root exceeds 512 KiB")
	}
	for name, raw := range map[string]string{"metadata": repository.MetadataURL, "targets": repository.TargetsURL} {
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("%s URL: %w", name, err)
		}
		if u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
			return fmt.Errorf("%s URL must be an absolute HTTPS URL without user information or fragment", name)
		}
		if err := validateEscapedPath(u); err != nil {
			return fmt.Errorf("%s URL: %w", name, err)
		}
	}
	return nil
}

func ValidatePlatform(platform Platform) error {
	if _, ok := supportedPlatforms[platform]; !ok {
		return fmt.Errorf("unsupported platform %s/%s", platform.OS, platform.Arch)
	}
	return nil
}

func ValidateStableVersion(version string) error {
	if !semver.IsValid(version) || semver.Canonical(version) != version {
		return fmt.Errorf("version %q is not canonical Go semantic versioning", version)
	}
	if semver.Prerelease(version) != "" || semver.Build(version) != "" {
		return fmt.Errorf("stable version %q must not contain pre-release or build metadata", version)
	}
	return nil
}

func CompareCandidate(current, candidate string) error {
	if err := ValidateStableVersion(current); err != nil {
		return fmt.Errorf("current version: %w", err)
	}
	if err := ValidateStableVersion(candidate); err != nil {
		return fmt.Errorf("candidate version: %w", err)
	}
	if semver.Compare(candidate, current) < 0 {
		return fmt.Errorf("candidate %s is older than current %s", candidate, current)
	}
	return nil
}

func TargetPath(channel, version string, platform Platform) (string, error) {
	if channel != StableChannel {
		return "", fmt.Errorf("unsupported channel %q", channel)
	}
	if err := ValidateStableVersion(version); err != nil {
		return "", err
	}
	if err := ValidatePlatform(platform); err != nil {
		return "", err
	}
	leaf := "engkit"
	if platform.OS == "windows" {
		leaf += ".exe"
	}
	return path.Join("releases", channel, version, platform.OS+"-"+platform.Arch, leaf), nil
}

func ValidateTarget(target Target, channel, version string, platform Platform) error {
	expected, err := TargetPath(channel, version, platform)
	if err != nil {
		return err
	}
	if target.Path != expected || target.Channel != channel || target.Version != version || target.Platform != platform {
		return errors.New("signed target metadata does not match requested release")
	}
	if target.Format != ExecutableFormatRaw {
		return fmt.Errorf("unsupported target format %q", target.Format)
	}
	if target.Length <= 0 || target.Length > 128*1024*1024 {
		return fmt.Errorf("target length %d is outside the allowed range", target.Length)
	}
	digest, err := hex.DecodeString(target.SHA256)
	if err != nil || len(digest) != 32 || target.SHA256 != strings.ToLower(target.SHA256) {
		return errors.New("target SHA-256 must be 64 lowercase hexadecimal characters")
	}
	return nil
}

func validateEscapedPath(u *url.URL) error {
	escaped := u.EscapedPath()
	decoded, err := url.PathUnescape(escaped)
	if err != nil {
		return errors.New("invalid escaped path")
	}
	if strings.Contains(decoded, "\\") || strings.ContainsRune(decoded, '\x00') {
		return errors.New("ambiguous path")
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return errors.New("path traversal is not allowed")
		}
	}
	if decoded != "" && path.Clean(decoded) != decoded && path.Clean(decoded)+"/" != decoded {
		return errors.New("path is not canonical")
	}
	return nil
}
