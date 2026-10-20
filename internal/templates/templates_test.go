package templates

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestScaffoldCreatesLoadableTemplateAndRefusesOverwrite(t *testing.T) {
	output := filepath.Join(localTempDir(t), "release-review")
	result, err := Scaffold(output, "release-review", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != ScaffoldSchema || len(result.Created) != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, err := Load(os.DirFS(output), "template.yaml"); err != nil {
		t.Fatalf("load scaffold: %v", err)
	}
	if _, err := Scaffold(output, "release-review", "0.1.0"); err == nil {
		t.Fatal("second Scaffold succeeded")
	}
}

func TestScaffoldRollsBackPublishFailure(t *testing.T) {
	parent := localTempDir(t)
	output := filepath.Join(parent, "release-review")
	if _, err := scaffold(output, "release-review", "0.1.0", func(_, _ string) error { return errors.New("injected publish failure") }); err == nil {
		t.Fatal("Scaffold succeeded")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("output exists after failure: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(parent, "engkit-template-staging-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("staging remains: %v, %v", matches, err)
	}
}

func localTempDir(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp(".", "engkit-template-test-")
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(absolute) })
	return absolute
}

func TestLoadRejectsUnknownPlaceholderAndDigestMismatch(t *testing.T) {
	validDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	definition := `schema_version: engkit.skill-template/v1alpha1
id: release-review
version: 0.1.0
description: Release review template.
skill:
  id: release-review
  version: 0.1.0
  description: Release review skill.
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
    source_sha256: ` + validDigest + "\n"
	filesystem := fstest.MapFS{
		"template.yaml":              {Data: []byte(definition)},
		"files/instructions.md.tmpl": {Data: []byte("{{engkit.unknown}}")},
	}
	if _, err := Load(filesystem, "template.yaml"); err == nil {
		t.Fatal("Load succeeded")
	}
}

func TestLoadRejectsSymlinkSource(t *testing.T) {
	filesystem := fstest.MapFS{"template.yaml": {Data: []byte("schema_version: engkit.skill-template/v1alpha1\n")}, "files/link": {Mode: fs.ModeSymlink}}
	if _, err := Load(filesystem, "template.yaml"); err == nil {
		t.Fatal("Load succeeded")
	}
}
