package profiles

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
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

func TestRecommendExpandedBuiltinGoProfiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/toolkit\n\ngo 1.27.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory, catalogFS, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	result, err := Recommend(root, inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations) != 3 {
		t.Fatalf("recommendations = %#v", result.Recommendations)
	}
	for index, want := range []string{"go-cli", "go-library", "go-service"} {
		got := result.Recommendations[index]
		if got.ID != want || !got.Compatible {
			t.Fatalf("recommendation at %d = %#v", index, got)
		}
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
