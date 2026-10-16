package update

import (
	"errors"
	"path"
	"path/filepath"
)

type StatePaths struct {
	MetadataDir string
	TargetCache string
}

func DefaultStatePaths() (StatePaths, error) {
	stateRoot, cacheRoot, err := platformStateRoots()
	if err != nil {
		return StatePaths{}, err
	}
	return statePaths(stateRoot, cacheRoot)
}

func StatePathsFor(goos, home, localAppData, xdgState, xdgCache string) (StatePaths, error) {
	var stateRoot, cacheRoot string
	switch goos {
	case "windows":
		if localAppData == "" {
			return StatePaths{}, errors.New("LOCALAPPDATA is required")
		}
		stateRoot = filepath.Join(localAppData, "EngineeringToolkit")
		cacheRoot = stateRoot
	case "darwin":
		if home == "" {
			return StatePaths{}, errors.New("home directory is required")
		}
		stateRoot = path.Join(home, "Library", "Application Support", "EngineeringToolkit")
		cacheRoot = path.Join(home, "Library", "Caches", "EngineeringToolkit")
		return statePathsSlash(stateRoot, cacheRoot)
	case "linux":
		if home == "" && (xdgState == "" || xdgCache == "") {
			return StatePaths{}, errors.New("home directory is required when XDG roots are unset")
		}
		if xdgState == "" {
			xdgState = path.Join(home, ".local", "state")
		}
		if xdgCache == "" {
			xdgCache = path.Join(home, ".cache")
		}
		stateRoot = path.Join(xdgState, "engkit")
		cacheRoot = path.Join(xdgCache, "engkit")
		return statePathsSlash(stateRoot, cacheRoot)
	default:
		return StatePaths{}, errors.New("unsupported operating system")
	}
	return statePaths(stateRoot, cacheRoot)
}

func statePathsSlash(stateRoot, cacheRoot string) (StatePaths, error) {
	if !path.IsAbs(stateRoot) || !path.IsAbs(cacheRoot) {
		return StatePaths{}, errors.New("state and cache roots must be absolute")
	}
	return StatePaths{
		MetadataDir: path.Join(stateRoot, "update", "metadata"),
		TargetCache: path.Join(cacheRoot, "update", "targets"),
	}, nil
}

func statePaths(stateRoot, cacheRoot string) (StatePaths, error) {
	if !filepath.IsAbs(stateRoot) || !filepath.IsAbs(cacheRoot) {
		return StatePaths{}, errors.New("state and cache roots must be absolute")
	}
	return StatePaths{
		MetadataDir: filepath.Join(stateRoot, "update", "metadata"),
		TargetCache: filepath.Join(cacheRoot, "update", "targets"),
	}, nil
}
