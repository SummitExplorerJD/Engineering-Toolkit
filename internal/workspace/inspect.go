package workspace

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const inspectionSchema = "engkit.workspace-inspection/v1alpha1"

// Inspection is a read-only snapshot of workspace facts used by planning.
type Inspection struct {
	SchemaVersion string   `json:"schema_version"`
	Root          string   `json:"root"`
	GoModule      string   `json:"go_module,omitempty"`
	HasGit        bool     `json:"has_git"`
	HasAgents     bool     `json:"has_agents"`
	HasEngkit     bool     `json:"has_engkit"`
	HasSpecKit    bool     `json:"has_spec_kit"`
	Entries       []string `json:"entries"`
}

// Inspect resolves root and reports facts without modifying the filesystem.
func Inspect(root string) (Inspection, error) {
	if strings.TrimSpace(root) == "" {
		return Inspection{}, errors.New("workspace root is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Inspection{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	absRoot = filepath.Clean(absRoot)

	info, err := os.Stat(absRoot)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect workspace root: %w", err)
	}
	if !info.IsDir() {
		return Inspection{}, fmt.Errorf("workspace root is not a directory: %s", absRoot)
	}

	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return Inspection{}, fmt.Errorf("read workspace root: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	module, err := readModulePath(filepath.Join(absRoot, "go.mod"))
	if err != nil {
		return Inspection{}, err
	}
	hasGit, err := probePath(filepath.Join(absRoot, ".git"))
	if err != nil {
		return Inspection{}, err
	}
	hasAgents, err := probePath(filepath.Join(absRoot, ".agents"))
	if err != nil {
		return Inspection{}, err
	}
	hasEngkit, err := probePath(filepath.Join(absRoot, ".engkit"))
	if err != nil {
		return Inspection{}, err
	}
	hasSpecKit, err := probePath(filepath.Join(absRoot, ".specify"))
	if err != nil {
		return Inspection{}, err
	}

	return Inspection{
		SchemaVersion: inspectionSchema,
		Root:          filepath.ToSlash(absRoot),
		GoModule:      module,
		HasGit:        hasGit,
		HasAgents:     hasAgents,
		HasEngkit:     hasEngkit,
		HasSpecKit:    hasSpecKit,
		Entries:       names,
	}, nil
}

func probePath(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("inspect path %s: %w", path, err)
}

func readModulePath(path string) (string, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if module == "" {
				return "", errors.New("go.mod has an empty module directive")
			}
			return module, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	return "", nil
}
