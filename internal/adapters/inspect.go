package adapters

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const InspectionSchema = "engkit.adapter-inspection/v1alpha1"

// PathState describes a provider-owned or shared path without reading content.
type PathState struct {
	Path  string `json:"path"`
	State string `json:"state"`
}

// Inspection reports the selected adapter”s relevant workspace state.
type Inspection struct {
	SchemaVersion string      `json:"schema_version"`
	Root          string      `json:"root"`
	Adapter       Adapter     `json:"adapter"`
	Paths         []PathState `json:"paths"`
	Configured    bool        `json:"configured"`
}

// Inspect is read-only and never follows a provider path to classify it.
func Inspect(root, provider string) (Inspection, error) {
	adapter, err := Get(provider)
	if err != nil {
		return Inspection{}, err
	}
	absRoot, err := resolveRoot(root)
	if err != nil {
		return Inspection{}, err
	}

	unique := make(map[string]struct{})
	for _, capability := range adapter.Capabilities {
		path := inspectionPath(capability.Path)
		if path != "" {
			unique[path] = struct{}{}
		}
	}
	paths := make([]string, 0, len(unique))
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	states := make([]PathState, 0, len(paths))
	configured := false
	for _, path := range paths {
		state, err := classify(filepath.Join(absRoot, filepath.FromSlash(path)))
		if err != nil {
			return Inspection{}, err
		}
		states = append(states, PathState{Path: path, State: state})
		if state != "missing" {
			configured = true
		}
	}

	return Inspection{
		SchemaVersion: InspectionSchema,
		Root:          filepath.ToSlash(absRoot),
		Adapter:       adapter,
		Paths:         states,
		Configured:    configured,
	}, nil
}

func resolveRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("workspace root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	absRoot = filepath.Clean(absRoot)
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("inspect workspace root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace root is not a directory: %s", absRoot)
	}
	return absRoot, nil
}

func inspectionPath(path string) string {
	if index := strings.Index(path, "*"); index >= 0 {
		prefix := strings.TrimSuffix(path[:index], "/")
		if separator := strings.LastIndex(prefix, "/"); separator >= 0 {
			path = prefix[:separator]
		} else {
			path = ""
		}
	}
	return strings.TrimSuffix(path, "/")
}

func classify(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "missing", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect adapter path %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "symlink", nil
	}
	if info.IsDir() {
		return "directory", nil
	}
	if info.Mode().IsRegular() {
		return "file", nil
	}
	return "other", nil
}
