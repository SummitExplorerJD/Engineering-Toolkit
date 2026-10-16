//go:build darwin

package update

import (
	"os"
	"path/filepath"
)

func platformStateRoots() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(home, "Library", "Application Support", "EngineeringToolkit"),
		filepath.Join(home, "Library", "Caches", "EngineeringToolkit"), nil
}
