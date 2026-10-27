// Package initfacade classifies one explicit initialization target.
package initfacade

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspaceconfig"
)

// Lifecycle identifies the one existing operation selected by unified init.
type Lifecycle string

const (
	WorkspaceInitialize Lifecycle = "workspace.initialize"
	ProjectCreate       Lifecycle = "project.create"
	WorkspaceCompose    Lifecycle = "workspace.compose"
)

// Classification is a read-only decision over one resolved target.
type Classification struct {
	Target    string
	Lifecycle Lifecycle
}

// Classify selects exactly one existing lifecycle without modifying the target.
func Classify(target string, hasProfile bool) (Classification, error) {
	if strings.TrimSpace(target) == "" {
		return Classification{}, errors.New("init target is required")
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		return Classification{}, fmt.Errorf("resolve init target: %w", err)
	}
	absolute = filepath.Clean(absolute)
	if err := rejectLinkedAncestors(absolute); err != nil {
		return Classification{}, err
	}

	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) {
		if !hasProfile {
			return Classification{}, errors.New("absent init target requires an exact --profile")
		}
		return Classification{Target: filepath.ToSlash(absolute), Lifecycle: ProjectCreate}, nil
	}
	if err != nil {
		return Classification{}, fmt.Errorf("inspect init target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Classification{}, errors.New("init target must not be a symbolic link or junction")
	}
	if !info.IsDir() {
		return Classification{}, errors.New("init target must be a directory or an absent path")
	}

	entries, err := os.ReadDir(absolute)
	if err != nil {
		return Classification{}, fmt.Errorf("read init target: %w", err)
	}
	initialized, err := inspectInitializedState(absolute)
	if err != nil {
		return Classification{}, err
	}
	if initialized {
		if !hasProfile {
			return Classification{}, errors.New("init target is already initialized; use workspace inspect or project inspect")
		}
		return Classification{Target: filepath.ToSlash(absolute), Lifecycle: WorkspaceCompose}, nil
	}

	if hasProfile {
		if len(entries) != 0 {
			return Classification{}, errors.New("non-empty uninitialized target requires separate workspace initialization and project composition approvals")
		}
		return Classification{Target: filepath.ToSlash(absolute), Lifecycle: ProjectCreate}, nil
	}
	return Classification{Target: filepath.ToSlash(absolute), Lifecycle: WorkspaceInitialize}, nil
}

func inspectInitializedState(root string) (bool, error) {
	engkitPath := filepath.Join(root, ".engkit")
	info, err := os.Lstat(engkitPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect .engkit state: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false, errors.New("init target contains partial or unsafe .engkit state")
	}
	for _, relative := range []string{"config.yaml", ".gitignore"} {
		path := filepath.Join(engkitPath, relative)
		entry, err := os.Lstat(path)
		if err != nil || entry.Mode()&os.ModeSymlink != 0 || !entry.Mode().IsRegular() {
			return false, fmt.Errorf("init target contains partial or unsafe .engkit state at .engkit/%s", relative)
		}
	}
	if err := validateConfiguration(filepath.Join(engkitPath, "config.yaml")); err != nil {
		return false, fmt.Errorf("init target contains invalid .engkit configuration: %w", err)
	}
	return true, nil
}

func validateConfiguration(path string) error {
	_, err := workspaceconfig.LoadFile(path)
	return err
}

func rejectLinkedAncestors(target string) error {
	for candidate := filepath.Clean(target); ; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return errors.New("init target must not use symbolic-link or junction ancestors")
			}
			if !info.IsDir() {
				return fmt.Errorf("init target ancestor %q is not a directory", candidate)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect init target ancestor %q: %w", candidate, err)
		}
		parent := filepath.Dir(candidate)
		if strings.EqualFold(parent, candidate) {
			return nil
		}
	}
}
