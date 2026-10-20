package schemautil

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var (
	kebabPattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	snakePattern  = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	volumePattern = regexp.MustCompile(`^[A-Za-z]:`)
)

func ValidKebab(value string) bool  { return kebabPattern.MatchString(value) && !ReservedWindows(value) }
func ValidSnake(value string) bool  { return snakePattern.MatchString(value) }
func ValidDigest(value string) bool { return digestPattern.MatchString(value) }

func ValidSemver(value string) bool {
	return value != "" && !strings.HasPrefix(value, "v") && semver.IsValid("v"+value)
}

func ValidateRelative(value string) error {
	native := filepath.FromSlash(value)
	if value == "" || strings.Contains(value, `\`) || volumePattern.MatchString(value) ||
		path.IsAbs(value) || filepath.IsAbs(native) || filepath.VolumeName(native) != "" ||
		path.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return fmt.Errorf("path %q must be a normalized relative slash path", value)
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") ||
			strings.ContainsAny(component, `<>:"|?*`) {
			return fmt.Errorf("path %q is not portable to Windows", value)
		}
		for _, character := range component {
			if character < 32 {
				return fmt.Errorf("path %q is not portable to Windows", value)
			}
		}
		if ReservedWindows(component) {
			return fmt.Errorf("path %q uses a reserved Windows name", value)
		}
	}
	return nil
}

func ReservedWindows(value string) bool {
	base := strings.ToLower(strings.SplitN(value, ".", 2)[0])
	switch base {
	case "con", "prn", "aux", "nul", "clock$":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "com") || strings.HasPrefix(base, "lpt")) &&
		base[3] >= '1' && base[3] <= '9'
}
