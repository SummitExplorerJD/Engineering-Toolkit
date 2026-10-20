package schemautil

import "testing"

func TestDecodeYAMLRejectsAmbiguousDocuments(t *testing.T) {
	type value struct {
		Name string `yaml:"name"`
	}
	for _, input := range []string{
		"name: first\nname: second\n",
		"name: &value first\n",
		"name: first\n---\nname: second\n",
		"unknown: value\n",
	} {
		var got value
		if err := DecodeYAML([]byte(input), 1024, "test.yaml", &got); err == nil {
			t.Fatalf("DecodeYAML(%q) succeeded", input)
		}
	}
}

func TestValidateRelative(t *testing.T) {
	for _, value := range []string{"instructions.md", "references/design.md", ".agents/context/initial.md"} {
		if err := ValidateRelative(value); err != nil {
			t.Fatalf("ValidateRelative(%q): %v", value, err)
		}
	}
	for _, value := range []string{"../escape", `bad\path`, "CON.txt", "trailing. "} {
		if err := ValidateRelative(value); err == nil {
			t.Fatalf("ValidateRelative(%q) succeeded", value)
		}
	}
}
