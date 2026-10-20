package profiles

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"testing/fstest"
)

func TestLoadValidProfile(t *testing.T) {
	asset := []byte("# {{engkit.project_name}}\n")
	digest := sha256.Sum256(asset)
	definition := `schema_version: engkit.project-profile/v1alpha1
id: go-service
version: 0.1.0
description: Go service workspace.
compatibility:
  ecosystems: [go]
  modes: [new-project, existing-workspace]
variables:
  - name: project_name
    type: string
    description: Project name.
    required: true
skills: []
workspace_directories: [.agents/context, .agents/conventions, .agents/skills, .agents/workflows]
workspace_assets:
  - source: files/AGENTS.md.tmpl
    destination: AGENTS.md
    kind: agent-instructions
    media_type: text
    source_sha256: sha256:` + hex.EncodeToString(digest[:]) + `
source_layout:
  policy: src
integration_requests: []
`
	filesystem := fstest.MapFS{"profile.yaml": {Data: []byte(definition)}, "files/AGENTS.md.tmpl": {Data: asset}}
	got, err := Load(filesystem, "profile.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "go-service" || got.SourceLayout.Policy != "src" {
		t.Fatalf("unexpected profile: %#v", got)
	}
}

func TestLoadRejectsUnsafeOrAmbiguousProfile(t *testing.T) {
	definitions := []string{
		"schema_version: engkit.project-profile/v1alpha1\nid: test\nversion: 0.1.0\ndescription: Test.\ncompatibility: {ecosystems: [], modes: []}\nsource_layout: {policy: preserve}\nunknown: true\n",
		"schema_version: engkit.project-profile/v1alpha1\nid: test\nversion: 0.1.0\ndescription: Test.\ncompatibility: {ecosystems: [], modes: [new-project]}\nworkspace_directories: [../escape]\nsource_layout: {policy: preserve}\n",
	}
	for _, definition := range definitions {
		if _, err := Load(fstest.MapFS{"profile.yaml": {Data: []byte(definition)}}, "profile.yaml"); err == nil {
			t.Fatalf("Load(%q) succeeded", definition)
		}
	}
}
