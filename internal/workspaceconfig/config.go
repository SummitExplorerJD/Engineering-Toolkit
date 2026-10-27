// Package workspaceconfig loads the bounded declarative workspace configuration.
package workspaceconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
}

type Workspace struct {
	SourceLayout string `yaml:"source_layout"`
}

type Intelligence struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
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
