package update

import (
	"errors"
	"path"
	"path/filepath"
	"strings"
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
		stateRoot = joinWindowsPath(localAppData, "EngineeringToolkit")
		return statePathsWindows(stateRoot, stateRoot)
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

func statePathsWindows(stateRoot, cacheRoot string) (StatePaths, error) {
	if !isWindowsAbs(stateRoot) || !isWindowsAbs(cacheRoot) {
		return StatePaths{}, errors.New("state and cache roots must be absolute")
	}
	return StatePaths{
		MetadataDir: joinWindowsPath(stateRoot, "update", "metadata"),
		TargetCache: joinWindowsPath(cacheRoot, "update", "targets"),
	}, nil
}

func isWindowsAbs(value string) bool {
	if len(value) >= 3 && isASCIILetter(value[0]) && value[1] == ':' && isWindowsSeparator(value[2]) {
		return true
	}
	if len(value) < 5 || !isWindowsSeparator(value[0]) || !isWindowsSeparator(value[1]) {
		return false
	}
	parts := strings.FieldsFunc(value[2:], func(r rune) bool {
		return r == '\\' || r == '/'
	})
	return len(parts) >= 2
}

func joinWindowsPath(root string, elements ...string) string {
	joined := strings.TrimRight(strings.ReplaceAll(root, "/", `\`), `\`)
	for _, element := range elements {
		element = strings.Trim(strings.ReplaceAll(element, "/", `\`), `\`)
		if element != "" {
			joined += `\` + element
		}
	}
	return joined
}

func isASCIILetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isWindowsSeparator(value byte) bool {
	return value == '\\' || value == '/'
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
