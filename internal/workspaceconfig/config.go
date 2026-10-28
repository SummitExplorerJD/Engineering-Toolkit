// Package workspaceconfig loads the bounded declarative workspace configuration.
package workspaceconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	Schema   = "engkit.config/v1alpha1"
	MaxBytes = 1 << 20
)

var allowedModels = map[string]struct{}{
	"phi4-mini:latest": {},
	"qwen3.5:0.8b":     {},
	"granite4:3b":      {},
}

type Config struct {
	SchemaVersion string        `yaml:"schema_version"`
	Workspace     *Workspace    `yaml:"workspace,omitempty"`
	Intelligence  *Intelligence `yaml:"intelligence,omitempty"`
	Context       *Context      `yaml:"context,omitempty"`
}

type Workspace struct {
	SourceLayout string `yaml:"source_layout"`
}

type Intelligence struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

// Context selects additional bounded textual source roots for the rebuildable
// context index. Omission preserves the legacy documentation-only inventory.
type Context struct {
	SourceRoots  []string `yaml:"source_roots"`
	ExcludePaths []string `yaml:"exclude_paths,omitempty"`
}

func Load(root string) (Config, error) {
	return LoadFile(filepath.Join(root, ".engkit", "config.yaml"))
}

func LoadFile(filename string) (Config, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return Config{}, fmt.Errorf("inspect configuration: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Config{}, errors.New("configuration must be a regular file")
	}
	if info.Size() > MaxBytes {
		return Config{}, errors.New("configuration exceeds the 1 MiB limit")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (Config, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil || containsAlias(&node) {
		return Config{}, errors.New("configuration is not supported YAML")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, errors.New("configuration does not match the supported schema")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("configuration must contain exactly one YAML document")
	}
	if config.SchemaVersion != Schema {
		return Config{}, errors.New("configuration uses an unsupported schema")
	}
	if config.Workspace != nil && config.Workspace.SourceLayout != "" && config.Workspace.SourceLayout != "auto" {
		return Config{}, errors.New("configuration uses an unsupported source layout")
	}
	if err := validateIntelligence(config.Intelligence); err != nil {
		return Config{}, err
	}
	if err := validateContext(config.Context); err != nil {
		return Config{}, err
	}
	return config, nil
}

func AllowedModel(model string) bool {
	_, ok := allowedModels[model]
	return ok
}

func validateIntelligence(config *Intelligence) error {
	if config == nil {
		return nil
	}
	if config.Provider != "" && config.Provider != "ollama" {
		return errors.New("intelligence provider must be ollama")
	}
	if config.Model != "" && !AllowedModel(config.Model) {
		return errors.New("intelligence model is not allowlisted")
	}
	if config.Enabled && (config.Provider != "ollama" || config.Model == "") {
		return errors.New("enabled intelligence requires provider ollama and one allowlisted model")
	}
	return nil
}

func validateContext(config *Context) error {
	if config == nil {
		return nil
	}
	if len(config.SourceRoots) == 0 || len(config.SourceRoots) > 32 {
		return errors.New("context source_roots must contain between 1 and 32 paths")
	}
	if len(config.ExcludePaths) > 128 {
		return errors.New("context exclude_paths exceeds the 128-path limit")
	}
	if err := validateContextPaths(config.SourceRoots, true); err != nil {
		return fmt.Errorf("context source_roots: %w", err)
	}
	if err := validateContextPaths(config.ExcludePaths, false); err != nil {
		return fmt.Errorf("context exclude_paths: %w", err)
	}
	return nil
}

func validateContextPaths(paths []string, allowRoot bool) error {
	seen := make(map[string]struct{}, len(paths))
	for _, value := range paths {
		if value == "" || len([]byte(value)) > 512 || !utf8.ValidString(value) || strings.ContainsAny(value, "\\\x00") {
			return errors.New("paths must contain between 1 and 512 UTF-8 bytes and use forward slashes")
		}
		if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || (len(value) >= 2 && value[1] == ':') {
			return errors.New("paths must be workspace-relative")
		}
		if path.Clean(value) != value || value == ".." || strings.HasPrefix(value, "../") {
			return errors.New("paths must be normalized and must not contain dot segments")
		}
		if value == "." && !allowRoot {
			return errors.New("the workspace root cannot be excluded")
		}
		if _, exists := seen[value]; exists {
			return errors.New("paths must be unique")
		}
		seen[value] = struct{}{}
	}
	return nil
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
