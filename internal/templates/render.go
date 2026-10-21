package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"gopkg.in/yaml.v3"
)

// Origin binds rendered bytes to the exact packaged catalog entry.
type Origin struct {
	CatalogID      string
	CatalogVersion string
	TemplateSHA256 string
}

// ResolvedValue is a typed, canonical template input.
type ResolvedValue struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// RenderedFile is one immutable in-memory output.
type RenderedFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Data   []byte `json:"-"`
}

// RenderedPackage is a complete canonical skill package held in memory.
type RenderedPackage struct {
	ID        string          `json:"id"`
	Version   string          `json:"version"`
	SHA256    string          `json:"sha256"`
	Variables []ResolvedValue `json:"variables"`
	Manifest  skills.Manifest `json:"manifest"`
	Files     []RenderedFile  `json:"files"`
}

// ResolveVariables parses explicit CLI values and applies declared defaults.
func ResolveVariables(declarations []Variable, supplied map[string]string) ([]ResolvedValue, map[string]any, error) {
	declared := make(map[string]Variable, len(declarations))
	for _, variable := range declarations {
		declared[variable.Name] = variable
	}
	for name := range supplied {
		if _, exists := declared[name]; !exists {
			return nil, nil, fmt.Errorf("unknown variable %q", name)
		}
	}
	resolved := make([]ResolvedValue, 0, len(declarations))
	values := make(map[string]any, len(declarations))
	for _, variable := range declarations {
		var value any
		if raw, exists := supplied[variable.Name]; exists {
			parsed, err := parseValue(variable.Type, raw)
			if err != nil {
				return nil, nil, fmt.Errorf("variable %q: %w", variable.Name, err)
			}
			value = parsed
		} else if variable.Default != nil {
			value = variable.Default
		} else if variable.Required {
			return nil, nil, fmt.Errorf("required variable %q is missing", variable.Name)
		} else {
			continue
		}
		if err := validateValue(variable, value, false); err != nil {
			return nil, nil, err
		}
		if len(variable.AllowedValues) != 0 && !containsValue(variable.AllowedValues, value) {
			return nil, nil, fmt.Errorf("variable %q value is not allowed", variable.Name)
		}
		values[variable.Name] = value
		resolved = append(resolved, ResolvedValue{Name: variable.Name, Type: variable.Type, Value: value})
	}
	return resolved, values, nil
}

func parseValue(kind, raw string) (any, error) {
	switch kind {
	case "string":
		return raw, nil
	case "integer":
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("must be a base-10 integer")
		}
		return value, nil
	case "boolean":
		if raw == "true" {
			return true, nil
		}
		if raw == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("must be true or false")
	default:
		return nil, fmt.Errorf("uses unsupported type %q", kind)
	}
}

func containsValue(allowed []any, value any) bool {
	needle := canonicalValue(value)
	for _, candidate := range allowed {
		if fmt.Sprintf("%T", candidate) == fmt.Sprintf("%T", value) && canonicalValue(candidate) == needle {
			return true
		}
	}
	return false
}

// Render produces and validates a complete canonical package without writing.
func Render(fsys fs.FS, root string, template Template, origin Origin, supplied map[string]string) (RenderedPackage, error) {
	resolved, values, err := ResolveVariables(template.Variables, supplied)
	if err != nil {
		return RenderedPackage{}, err
	}
	files := make([]RenderedFile, 0, len(template.Files)+1)
	resources := make([]skills.Resource, 0, len(template.Files))
	memory := make([]skills.MemoryFile, 0, len(template.Files))
	for _, declaration := range template.Files {
		data, err := fs.ReadFile(fsys, path.Join(root, declaration.Source))
		if err != nil {
			return RenderedPackage{}, fmt.Errorf("read template source %q: %w", declaration.Source, err)
		}
		if declaration.MediaType == "text" {
			data, err = substitute(data, values)
			if err != nil {
				return RenderedPackage{}, fmt.Errorf("render %q: %w", declaration.Source, err)
			}
		}
		digest := byteDigest(data)
		files = append(files, RenderedFile{Path: declaration.Destination, Kind: declaration.Kind, Size: int64(len(data)), SHA256: digest, Data: data})
		resources = append(resources, skills.Resource{Path: declaration.Destination, Kind: declaration.Kind, SHA256: digest})
		memory = append(memory, skills.MemoryFile{Path: declaration.Destination, Data: data})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	sort.Slice(resources, func(i, j int) bool { return resources[i].Path < resources[j].Path })
	manifest := skills.Manifest{
		SchemaVersion: skills.ManifestSchema, ID: template.Skill.ID, Version: template.Skill.Version,
		Description: template.Skill.Description, Instructions: template.Skill.Instructions,
		Capabilities: template.Skill.Capabilities, Inputs: template.Skill.Inputs, Outputs: template.Skill.Outputs,
		Permissions: template.Skill.Permissions, Compatibility: template.Skill.Compatibility,
		Dependencies: template.Skill.Dependencies, Ownership: template.Skill.Ownership,
		Provenance: &skills.Provenance{
			Source:   fmt.Sprintf("catalog:%s/%s", origin.CatalogID, template.ID),
			Revision: fmt.Sprintf("%s/%s/%s", origin.CatalogVersion, template.Version, origin.TemplateSHA256),
		},
		Resources: resources,
	}
	if err := skills.ValidateMemory(manifest, memory); err != nil {
		return RenderedPackage{}, err
	}
	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return RenderedPackage{}, fmt.Errorf("encode rendered skill manifest: %w", err)
	}
	files = append(files, RenderedFile{Path: "skill.yaml", Kind: "manifest", Size: int64(len(manifestData)), SHA256: byteDigest(manifestData), Data: manifestData})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return RenderedPackage{ID: manifest.ID, Version: manifest.Version, SHA256: fileSetDigest(files), Variables: resolved, Manifest: manifest, Files: files}, nil
}

func substitute(data []byte, values map[string]any) ([]byte, error) {
	var missing string
	result := placeholderPattern.ReplaceAllStringFunc(string(data), func(token string) string {
		match := placeholderPattern.FindStringSubmatch(token)
		value, exists := values[match[1]]
		if !exists {
			missing = match[1]
			return token
		}
		return canonicalValue(value)
	})
	if missing != "" {
		return nil, fmt.Errorf("variable %q has no resolved value", missing)
	}
	return []byte(result), nil
}

// RenderText applies the same single-pass substitution used by skill resources.
func RenderText(data []byte, resolved []ResolvedValue) ([]byte, error) {
	values := make(map[string]any, len(resolved))
	for _, value := range resolved {
		values[value.Name] = value.Value
	}
	return substitute(data, values)
}

func canonicalValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	default:
		integer, _ := integerValue(value)
		return strconv.FormatInt(integer, 10)
	}
}

// ValueText returns the canonical substitution form for a resolved value.
func ValueText(value any) string { return canonicalValue(value) }

// InputText validates a typed binding and returns its canonical CLI form.
func InputText(variable Variable, value any) (string, error) {
	if !matchesType(variable.Type, value) {
		return "", fmt.Errorf("value does not match type %q", variable.Type)
	}
	if err := validateValue(variable, value, false); err != nil {
		return "", err
	}
	if len(variable.AllowedValues) != 0 && !containsValue(variable.AllowedValues, value) {
		return "", fmt.Errorf("value is not allowed")
	}
	return canonicalValue(value), nil
}

func byteDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func fileSetDigest(files []RenderedFile) string {
	hash := sha256.New()
	for _, file := range files {
		_, _ = hash.Write([]byte(file.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(strings.TrimPrefix(file.SHA256, "sha256:")))
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}
