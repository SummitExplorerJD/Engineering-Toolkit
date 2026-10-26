package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateRoot resolves an existing workspace directory and rejects linked
// path components. It is intended for long-lived root-bound processes.
func ValidateRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("workspace root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	abs = filepath.Clean(abs)
	for candidate := abs; ; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if err != nil {
			return "", fmt.Errorf("inspect workspace root ancestry: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("workspace root must not use symbolic-link or junction ancestors")
		}
		if !info.IsDir() {
			return "", errors.New("workspace root ancestry must contain directories only")
		}
		parent := filepath.Dir(candidate)
		if strings.EqualFold(parent, candidate) {
			break
		}
	}
	return filepath.ToSlash(abs), nil
}
