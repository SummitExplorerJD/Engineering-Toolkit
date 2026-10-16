//go:build linux

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
	stateRoot := os.Getenv("XDG_STATE_HOME")
	if stateRoot == "" {
		stateRoot = filepath.Join(home, ".local", "state")
	}
	cacheRoot := os.Getenv("XDG_CACHE_HOME")
	if cacheRoot == "" {
		cacheRoot = filepath.Join(home, ".cache")
	}
	return filepath.Join(stateRoot, "engkit"), filepath.Join(cacheRoot, "engkit"), nil
}
