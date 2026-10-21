package catalog

import (
	"testing"
	"testing/fstest"
)

func TestBuiltinCatalogContainsValidatedPilotContent(t *testing.T) {
	got, _, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != Schema || got.ID != "engineering-toolkit" || len(got.Templates) != 1 || len(got.Profiles) != 1 {
		t.Fatalf("unexpected built-in catalog: %#v", got)
	}
	if got.Templates[0].ID != "project-guide" || got.Templates[0].Version != "0.1.0" {
		t.Fatalf("unexpected pilot template: %#v", got.Templates[0])
	}
	if got.Profiles[0].ID != "go-service" || got.Profiles[0].Version != "0.1.0" {
		t.Fatalf("unexpected pilot profile: %#v", got.Profiles[0])
	}
}

func TestLoadRejectsUnknownFieldAndUnsortedEntries(t *testing.T) {
	for _, definition := range []string{
		"schema_version: engkit.catalog/v1alpha1\nid: test\nversion: 0.1.0\nminimum_engkit: 0.2.1\ntemplates: []\nprofiles: []\nunknown: true\n",
		"schema_version: engkit.catalog/v1alpha1\nid: test\nversion: 0.1.0\nminimum_engkit: 0.2.1\ntemplates:\n  - {id: zed, version: 0.1.0, path: templates/zed/0.1.0, sha256: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa}\n  - {id: alpha, version: 0.1.0, path: templates/alpha/0.1.0, sha256: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb}\nprofiles: []\n",
	} {
		filesystem := fstest.MapFS{"catalog.yaml": {Data: []byte(definition)}}
		if _, err := Load(filesystem, "catalog.yaml"); err == nil {
			t.Fatalf("Load(%q) succeeded", definition)
		}
	}
}

func TestDirectoryDigestIsDeterministic(t *testing.T) {
	filesystem := fstest.MapFS{
		"entry/a.txt":        {Data: []byte("a")},
		"entry/nested/b.txt": {Data: []byte("b")},
	}
	first, err := DirectoryDigest(filesystem, "entry")
	if err != nil {
		t.Fatal(err)
	}
	second, err := DirectoryDigest(filesystem, "entry")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("digest changed: %q != %q", first, second)
	}
}
