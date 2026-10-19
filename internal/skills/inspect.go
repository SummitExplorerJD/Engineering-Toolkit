package skills

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxManifestBytes = 1 << 20

// List discovers canonical and external skill directories without mutation.
func List(root string) (ListResponse, error) {
	absRoot, err := resolveRoot(root)
	if err != nil {
		return ListResponse{}, err
	}
	packages, err := discover(absRoot)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{
		SchemaVersion: ListSchema,
		Root:          filepath.ToSlash(absRoot),
		SkillsRoot:    ".agents/skills",
		Packages:      packages,
	}, nil
}

// Inspect returns one package selected by its safe directory identifier.
func Inspect(root, id string) (Inspection, error) {
	if !validID(id) {
		return Inspection{}, errors.New("skill id must be a normalized lowercase hyphenated identifier")
	}
	listed, err := List(root)
	if err != nil {
		return Inspection{}, err
	}
	for _, candidate := range listed.Packages {
		if candidate.Directory == id {
			return Inspection{SchemaVersion: InspectionSchema, Root: listed.Root, Package: candidate}, nil
		}
	}
	return Inspection{}, fmt.Errorf("skill %q was not found", id)
}

// Validate returns structured findings for one package or the complete root.
func Validate(root, id string) (Validation, error) {
	listed, err := List(root)
	if err != nil {
		return Validation{}, err
	}
	selected := listed.Packages
	if id != "" {
		if !validID(id) {
			return Validation{}, errors.New("skill id must be a normalized lowercase hyphenated identifier")
		}
		selected = nil
		for _, candidate := range listed.Packages {
			if candidate.Directory == id {
				selected = append(selected, candidate)
				break
			}
		}
		if len(selected) == 0 {
			return Validation{}, fmt.Errorf("skill %q was not found", id)
		}
	}

	valid := true
	for _, candidate := range selected {
		if candidate.Classification == ClassificationInvalid || candidate.Classification == ClassificationConflict {
			valid = false
			break
		}
	}
	return Validation{
		SchemaVersion: ValidationSchema,
		Root:          listed.Root,
		Valid:         valid,
		Packages:      selected,
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

func discover(root string) ([]Package, error) {
	skillsRoot := filepath.Join(root, ".agents", "skills")
	info, err := os.Lstat(skillsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return []Package{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect skills root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("skills root must not be a symbolic link")
	}
	if !info.IsDir() {
		return nil, errors.New(".agents/skills is not a directory")
	}

	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return nil, fmt.Errorf("read skills root: %w", err)
	}
	packages := make([]Package, 0, len(entries))
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect skill entry %s: %w", entry.Name(), err)
		}
		if !entryInfo.IsDir() && entryInfo.Mode()&os.ModeSymlink == 0 {
			continue
		}
		packages = append(packages, inspectPackage(skillsRoot, entry.Name(), entryInfo))
	}
	markConflicts(packages)
	return packages, nil
}

func inspectPackage(skillsRoot, directory string, info os.FileInfo) Package {
	relative := filepath.ToSlash(filepath.Join(".agents", "skills", directory))
	result := Package{ID: directory, Directory: directory, Path: relative, Classification: ClassificationExternal}
	if info.Mode()&os.ModeSymlink != 0 {
		result.Classification = ClassificationInvalid
		result.Diagnostics = []Diagnostic{diagnostic("package.symlink", relative, "skill package directories must not be symbolic links")}
		return result
	}

	packageRoot := filepath.Join(skillsRoot, directory)
	manifestPath := filepath.Join(packageRoot, "skill.yaml")
	manifestInfo, err := os.Lstat(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return result
	}
	if err != nil {
		result.Classification = ClassificationInvalid
		result.Diagnostics = []Diagnostic{diagnostic("manifest.inspect", filepath.ToSlash(manifestPath), err.Error())}
		return result
	}
	result.Classification = ClassificationInvalid
	if manifestInfo.Mode()&os.ModeSymlink != 0 || !manifestInfo.Mode().IsRegular() {
		result.Diagnostics = []Diagnostic{diagnostic("manifest.type", relative+"/skill.yaml", "skill.yaml must be a regular file and not a symbolic link")}
		return result
	}

	manifest, findings := readManifest(manifestPath)
	if manifest != nil {
		result.ID = manifest.ID
		result.Manifest = manifest
		findings = append(findings, validateManifest(*manifest, directory)...)
		findings = append(findings, validateResources(packageRoot, *manifest)...)
	}
	sortDiagnostics(findings)
	result.Diagnostics = findings
	if len(findings) == 0 {
		result.Classification = ClassificationCanonical
	}
	return result
}

func readManifest(filename string) (*Manifest, []Diagnostic) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, []Diagnostic{diagnostic("manifest.open", filepath.ToSlash(filename), err.Error())}
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return nil, []Diagnostic{diagnostic("manifest.read", filepath.ToSlash(filename), err.Error())}
	}
	if len(data) > maxManifestBytes {
		return nil, []Diagnostic{diagnostic("manifest.size", "skill.yaml", "manifest exceeds the 1 MiB limit")}
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, []Diagnostic{diagnostic("manifest.yaml", "skill.yaml", err.Error())}
	}
	if containsAlias(&document) {
		return nil, []Diagnostic{diagnostic("manifest.alias", "skill.yaml", "YAML aliases are not allowed")}
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, []Diagnostic{diagnostic("manifest.decode", "skill.yaml", err.Error())}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, []Diagnostic{diagnostic("manifest.documents", "skill.yaml", "exactly one YAML document is allowed")}
		}
		return nil, []Diagnostic{diagnostic("manifest.decode", "skill.yaml", err.Error())}
	}
	return &manifest, nil
}

func containsAlias(node *yaml.Node) bool {
	if node.Kind == yaml.AliasNode {
		return true
	}
	for _, child := range node.Content {
		if containsAlias(child) {
			return true
		}
	}
	return false
}

func markConflicts(packages []Package) {
	byID := make(map[string][]int)
	for index := range packages {
		if packages[index].Manifest != nil && validID(packages[index].Manifest.ID) {
			byID[packages[index].Manifest.ID] = append(byID[packages[index].Manifest.ID], index)
		}
	}
	for id, indexes := range byID {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			packages[index].Classification = ClassificationConflict
			packages[index].Diagnostics = append(packages[index].Diagnostics, diagnostic("identity.duplicate", packages[index].Path, fmt.Sprintf("canonical identity %q is declared by multiple directories", id)))
			sortDiagnostics(packages[index].Diagnostics)
		}
	}
}

func diagnostic(code, path, message string) Diagnostic {
	return Diagnostic{Severity: "error", Code: code, Path: path, Message: message}
}

func sortDiagnostics(items []Diagnostic) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path != items[j].Path {
			return items[i].Path < items[j].Path
		}
		if items[i].Code != items[j].Code {
			return items[i].Code < items[j].Code
		}
		return items[i].Message < items[j].Message
	})
}
