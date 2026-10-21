package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRenderProducesValidatedCanonicalPackageInMemory(t *testing.T) {
	source := []byte("# {{engkit.project_name}}\n\nReview {{engkit.review_count}} changes: {{engkit.enabled}}.\n")
	digest := sha256.Sum256(source)
	definition := `schema_version: engkit.skill-template/v1alpha1
id: release-review
version: 0.1.0
description: Release review template.
variables:
  - name: enabled
    type: boolean
    description: Whether review is enabled.
    required: true
  - name: project_name
    type: string
    description: Project name.
    required: true
  - name: review_count
    type: integer
    description: Review count.
    required: true
skill:
  id: release-review
  version: 0.1.0
  description: Reviews releases.
  instructions: instructions.md
  permissions: {}
  compatibility:
    minimum_engkit: 0.2.1
  ownership:
    owner: engineering
files:
  - source: files/instructions.md.tmpl
    destination: instructions.md
    kind: instructions
    media_type: text
    source_sha256: sha256:` + hex.EncodeToString(digest[:]) + "\n"
	filesystem := fstest.MapFS{
		"entry/template.yaml":              {Data: []byte(definition)},
		"entry/files/instructions.md.tmpl": {Data: source},
	}
	template, err := Load(filesystem, "entry/template.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := Render(filesystem, "entry", template, Origin{CatalogID: "test", CatalogVersion: "0.2.1", TemplateSHA256: "sha256:" + strings.Repeat("a", 64)}, map[string]string{"enabled": "true", "project_name": "Toolkit", "review_count": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered.Files) != 2 || rendered.Files[0].Path != "instructions.md" || rendered.Files[1].Path != "skill.yaml" {
		t.Fatalf("unexpected rendered files: %#v", rendered.Files)
	}
	if string(rendered.Files[0].Data) != "# Toolkit\n\nReview 2 changes: true.\n" {
		t.Fatalf("unexpected instructions: %q", rendered.Files[0].Data)
	}
	if !strings.HasPrefix(rendered.SHA256, "sha256:") || rendered.Manifest.Provenance == nil {
		t.Fatalf("unexpected rendered package: %#v", rendered)
	}
}

func TestResolveVariablesRejectsUnknownMissingAndInvalidValues(t *testing.T) {
	declarations := []Variable{{Name: "count", Type: "integer", Description: "Count.", Required: true}}
	for _, supplied := range []map[string]string{{}, {"unknown": "1"}, {"count": "nope"}} {
		if _, _, err := ResolveVariables(declarations, supplied); err == nil {
			t.Fatalf("ResolveVariables(%v) succeeded", supplied)
		}
	}
}
