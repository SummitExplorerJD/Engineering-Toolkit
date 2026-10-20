package schemautil

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// DecodeYAML decodes one strict YAML document into out.
func DecodeYAML(data []byte, limit int, name string, out any) error {
	if len(data) > limit {
		return fmt.Errorf("%s exceeds the %d-byte limit", name, limit)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("decode %s YAML: %w", name, err)
	}
	if len(document.Content) != 1 {
		return fmt.Errorf("%s must contain exactly one YAML document", name)
	}
	if err := inspectNode(document.Content[0], name); err != nil {
		return err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("%s must contain exactly one YAML document", name)
		}
		return fmt.Errorf("decode trailing %s content: %w", name, err)
	}
	return nil
}

func inspectNode(node *yaml.Node, name string) error {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return fmt.Errorf("%s must not contain YAML aliases or anchors", name)
	}
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]struct{}, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode {
				return fmt.Errorf("%s mapping keys must be scalars", name)
			}
			if _, exists := seen[key.Value]; exists {
				return fmt.Errorf("%s contains duplicate key %q", name, key.Value)
			}
			seen[key.Value] = struct{}{}
		}
	}
	for _, child := range node.Content {
		if err := inspectNode(child, name); err != nil {
			return err
		}
	}
	return nil
}
