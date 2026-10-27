package workspaceconfig

import "testing"

func TestParseCompatibilityAndIntelligence(t *testing.T) {
	for _, test := range []struct {
		name    string
		data    string
		enabled bool
		model   string
	}{
		{name: "legacy", data: "schema_version: engkit.config/v1alpha1\nworkspace:\n  source_layout: auto\n"},
		{name: "disabled", data: "schema_version: engkit.config/v1alpha1\nintelligence:\n  enabled: false\n"},
		{name: "enabled", data: "schema_version: engkit.config/v1alpha1\nintelligence:\n  enabled: true\n  provider: ollama\n  model: granite4:3b\n", enabled: true, model: "granite4:3b"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse([]byte(test.data))
			if err != nil {
				t.Fatal(err)
			}
			if test.enabled != (got.Intelligence != nil && got.Intelligence.Enabled) {
				t.Fatalf("unexpected intelligence: %#v", got.Intelligence)
			}
			if got.Intelligence != nil && got.Intelligence.Model != test.model {
				t.Fatalf("model = %q", got.Intelligence.Model)
			}
		})
	}
}

func TestParseRejectsInvalidConfiguration(t *testing.T) {
	for _, data := range []string{
		"schema_version: engkit.config/v1alpha1\nunknown: true\n",
		"schema_version: engkit.config/v1alpha1\nintelligence:\n  enabled: true\n",
		"schema_version: engkit.config/v1alpha1\nintelligence:\n  enabled: true\n  provider: remote\n  model: granite4:3b\n",
		"schema_version: engkit.config/v1alpha1\nintelligence:\n  enabled: true\n  provider: ollama\n  model: qwen3.5:4b\n",
	} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatalf("Parse(%q) succeeded", data)
		}
	}
}
