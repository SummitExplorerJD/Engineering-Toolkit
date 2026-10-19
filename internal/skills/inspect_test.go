package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListClassifiesCanonicalAndExternalPackages(t *testing.T) {
	root := t.TempDir()
	writeCanonicalPackage(t, root, "release-review", "# Release review\n", "")
	writeFile(t, filepath.Join(root, ".agents", "skills", "speckit-plan", "SKILL.md"), "# External\n")

	got, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != ListSchema || len(got.Packages) != 2 {
		t.Fatalf("unexpected list response: %#v", got)
	}
	if got.Packages[0].Classification != ClassificationCanonical || got.Packages[0].ID != "release-review" {
		t.Fatalf("canonical package = %#v", got.Packages[0])
	}
	if got.Packages[1].Classification != ClassificationExternal || got.Packages[1].Directory != "speckit-plan" {
		t.Fatalf("external package = %#v", got.Packages[1])
	}
}

func TestInspectAndValidateDoNotModifyPackage(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeCanonicalPackage(t, root, "release-review", "# Release review\n", "")
	beforeManifest := readFile(t, manifestPath)
	beforeInstructions := readFile(t, filepath.Join(filepath.Dir(manifestPath), "instructions.md"))

	inspection, err := Inspect(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Package.Manifest == nil || inspection.Package.Manifest.ID != "release-review" {
		t.Fatalf("unexpected inspection: %#v", inspection)
	}
	validation, err := Validate(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	if !validation.Valid {
		t.Fatalf("validation failed: %#v", validation)
	}
	if got := readFile(t, manifestPath); got != beforeManifest {
		t.Fatal("inspection or validation modified skill.yaml")
	}
	if got := readFile(t, filepath.Join(filepath.Dir(manifestPath), "instructions.md")); got != beforeInstructions {
		t.Fatal("inspection or validation modified instructions.md")
	}
}

func TestValidateRejectsUnknownFieldsTraversalAndUndeclaredFiles(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeCanonicalPackage(t, root, "unsafe-skill", "# Unsafe\n", "unknown_field: true\n")
	writeFile(t, filepath.Join(filepath.Dir(manifestPath), "extra.txt"), "not declared\n")

	got, err := Validate(root, "unsafe-skill")
	if err != nil {
		t.Fatal(err)
	}
	if got.Valid || got.Packages[0].Classification != ClassificationInvalid {
		t.Fatalf("unexpected validation: %#v", got)
	}
	if !hasDiagnostic(got.Packages[0], "manifest.decode") {
		t.Fatalf("expected strict unknown-field diagnostic: %#v", got.Packages[0].Diagnostics)
	}

	writeCanonicalPackage(t, root, "unsafe-skill", "# Unsafe\n", "", resourceBlock("../escape.md", "reference", digest("missing")))
	got, err = Validate(root, "unsafe-skill")
	if err != nil {
		t.Fatal(err)
	}
	if got.Valid || !hasDiagnostic(got.Packages[0], "resource.path") || !hasDiagnostic(got.Packages[0], "resource.undeclared") {
		t.Fatalf("expected path and undeclared-resource diagnostics: %#v", got.Packages[0].Diagnostics)
	}
}

func TestValidateRejectsIntegrityMismatch(t *testing.T) {
	root := t.TempDir()
	writeCanonicalPackage(t, root, "release-review", "# Release review\n", "", resourceBlock("instructions.md", "instructions", digest("different")))

	got, err := Validate(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	if got.Valid || !hasDiagnostic(got.Packages[0], "resource.integrity") {
		t.Fatalf("expected integrity failure: %#v", got)
	}
}

func TestValidateRejectsPlatformIndependentAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	writeCanonicalPackage(t, root, "unsafe-skill", "# Unsafe\n", "", resourceBlock("C:/escape.md", "reference", digest("missing")))

	got, err := Validate(root, "unsafe-skill")
	if err != nil {
		t.Fatal(err)
	}
	if got.Valid || !hasDiagnostic(got.Packages[0], "resource.path") {
		t.Fatalf("expected Windows-volume path failure on every platform: %#v", got)
	}
}

func TestValidateRejectsUnknownEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeCanonicalPackage(t, root, "release-review", "# Release review\n", "")
	if err := os.Mkdir(filepath.Join(filepath.Dir(manifestPath), "unknown"), 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := Validate(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	if got.Valid || !hasDiagnostic(got.Packages[0], "resource.directory") {
		t.Fatalf("expected unknown-directory failure: %#v", got)
	}
}

func TestListMarksDuplicateCanonicalIdentityAsConflict(t *testing.T) {
	root := t.TempDir()
	first := writeCanonicalPackage(t, root, "first-skill", "# First\n", "")
	second := writeCanonicalPackage(t, root, "second-skill", "# Second\n", "")
	writeFile(t, first, strings.Replace(readFile(t, first), "id: first-skill", "id: shared-skill", 1))
	writeFile(t, second, strings.Replace(readFile(t, second), "id: second-skill", "id: shared-skill", 1))

	got, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Packages) != 2 {
		t.Fatalf("packages = %d", len(got.Packages))
	}
	for _, pkg := range got.Packages {
		if pkg.Classification != ClassificationConflict || !hasDiagnostic(pkg, "identity.duplicate") {
			t.Fatalf("expected conflict classification: %#v", pkg)
		}
	}
}

func TestInspectRejectsUnsafeID(t *testing.T) {
	root := t.TempDir()
	if _, err := Inspect(root, "../escape"); err == nil {
		t.Fatal("Inspect accepted a traversal identifier")
	}
}

func writeCanonicalPackage(t *testing.T, root, id, instructions, extra string, resourceOverrides ...string) string {
	t.Helper()
	packageRoot := filepath.Join(root, ".agents", "skills", id)
	writeFile(t, filepath.Join(packageRoot, "instructions.md"), instructions)
	resources := resourceBlock("instructions.md", "instructions", digest(instructions))
	if len(resourceOverrides) > 0 {
		resources = strings.Join(resourceOverrides, "")
	}
	manifest := fmt.Sprintf(`schema_version: engkit.skill/v1alpha1
id: %s
version: 0.1.0
description: Reviews a release deterministically.
instructions: instructions.md
capabilities:
  - release.review
inputs:
  - name: release
    type: string
    description: Release identifier.
    required: true
outputs:
  - name: report
    type: object
    description: Review report.
permissions: {}
compatibility:
  minimum_engkit: 0.2.0
  providers:
    - codex
ownership:
  owner: engineering
provenance:
  source: local
  revision: initial
resources:
%s%s`, id, resources, extra)
	manifestPath := filepath.Join(packageRoot, "skill.yaml")
	writeFile(t, manifestPath, manifest)
	return manifestPath
}

func resourceBlock(resourcePath, kind, hash string) string {
	return fmt.Sprintf("  - path: %s\n    kind: %s\n    sha256: %s\n", resourcePath, kind, hash)
}

func digest(contents string) string {
	sum := sha256.Sum256([]byte(contents))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeFile(t *testing.T, filename, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, filename string) string {
	t.Helper()
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}

func hasDiagnostic(pkg Package, code string) bool {
	for _, finding := range pkg.Diagnostics {
		if finding.Code == code {
			return true
		}
	}
	return false
}
