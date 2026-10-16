//go:build windows

package update

import (
	"errors"
	"os"
	"path/filepath"
)

func platformStateRoots() (string, string, error) {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		return "", "", errors.New("LOCALAPPDATA is required")
	}
	root = filepath.Join(root, "EngineeringToolkit")
	return root, root, nil
}
