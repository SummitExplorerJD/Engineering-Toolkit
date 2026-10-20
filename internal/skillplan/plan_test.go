package skillplan

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCreatePlanIsDeterministicAndReadOnly(t *testing.T) {
	root := t.TempDir()
	source := writePackage(t, t.TempDir(), "release-review", "1.0.0", map[string]string{
		"instructions.md":     "# Release review\n",
		"references/guide.md": "Review every checksum.\n",
	})

	first, err := NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanID != second.PlanID || !strings.HasPrefix(first.PlanID, "sha256:") {
		t.Fatalf("plan is not deterministic: %q != %q", first.PlanID, second.PlanID)
	}
	if first.Kind != "skill.create" || !first.RequiresApproval || !first.Executable {
		t.Fatalf("unexpected create plan controls: %#v", first)
	}
	if !hasAction(first, "create_directory", ".agents/skills/release-review") || !hasAction(first, "write_file", ".agents/skills/release-review/skill.yaml") {
		t.Fatalf("missing create operations: %#v", first.Operations)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("create planning mutated the workspace: %v", err)
	}
}

func TestCreatePlanRejectsExistingIdentityInDifferentDirectory(t *testing.T) {
	root := t.TempDir()
	conflict := writePackage(t, filepath.Join(root, ".agents", "skills"), "other-directory", "1.0.0", map[string]string{"instructions.md": "# Existing\n"})
	manifestPath := filepath.Join(conflict, "skill.yaml")
	writeFile(t, manifestPath, strings.Replace(readFile(t, manifestPath), "id: other-directory", "id: release-review", 1))
	source := writePackage(t, t.TempDir(), "release-review", "1.0.0", map[string]string{"instructions.md": "# Proposed\n"})

	if _, err := NewCreate(root, source); err == nil {
		t.Fatal("create plan accepted a conflicting canonical identity")
	}
}

func TestUpdatePlanBindsBeforeAndAfterState(t *testing.T) {
	root := t.TempDir()
	target := writePackage(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", map[string]string{
		"instructions.md":   "# Old instructions\n",
		"references/old.md": "old\n",
	})
	source := writePackage(t, t.TempDir(), "release-review", "1.1.0", map[string]string{
		"instructions.md":   "# New instructions\n",
		"references/new.md": "new\n",
	})
	before := readFile(t, filepath.Join(target, "instructions.md"))

	got, err := NewUpdate(root, "release-review", source)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "skill.update" || got.TargetVersion != "1.0.0" || got.ProposedVersion != "1.1.0" {
		t.Fatalf("unexpected update plan: %#v", got)
	}
	if got.Root != filepath.ToSlash(root) {
		t.Fatalf("Root = %q, want %q", got.Root, filepath.ToSlash(root))
	}
	if !hasAction(got, "replace_file", ".agents/skills/release-review/instructions.md") ||
		!hasAction(got, "write_file", ".agents/skills/release-review/references/new.md") ||
		!hasAction(got, "remove_file", ".agents/skills/release-review/references/old.md") {
		t.Fatalf("missing update operations: %#v", got.Operations)
	}
	for _, operation := range got.Operations {
		if operation.Action == "replace_file" && (operation.BeforeSHA256 == "" || operation.AfterSHA256 == "") {
			t.Fatalf("replace operation is not integrity-bound: %#v", operation)
		}
	}
	if after := readFile(t, filepath.Join(target, "instructions.md")); after != before {
		t.Fatal("update planning modified the target package")
	}
}

func TestUpdatePlanRequiresHigherVersion(t *testing.T) {
	root := t.TempDir()
	writePackage(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", map[string]string{"instructions.md": "# Current\n"})
	source := writePackage(t, t.TempDir(), "release-review", "1.0.0", map[string]string{"instructions.md": "# Replacement\n"})

	if _, err := NewUpdate(root, "release-review", source); err == nil || !strings.Contains(err.Error(), "must be greater") {
		t.Fatalf("expected version refusal, got %v", err)
	}
}

func TestUpdatePlanOrdersFileDirectoryTransitions(t *testing.T) {
	root := t.TempDir()
	writePackage(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", map[string]string{
		"instructions.md":            "# Current\n",
		"assets/to-dir":              "old file\n",
		"references/to-file/old.txt": "old nested file\n",
	})
	source := writePackage(t, t.TempDir(), "release-review", "1.1.0", map[string]string{
		"instructions.md":       "# Proposed\n",
		"assets/to-dir/new.txt": "new nested file\n",
		"references/to-file":    "new file\n",
	})

	got, err := NewUpdate(root, "release-review", source)
	if err != nil {
		t.Fatal(err)
	}
	removeBlockingFile := actionIndex(got, "remove_file", ".agents/skills/release-review/assets/to-dir")
	createReplacementDirectory := actionIndex(got, "create_directory", ".agents/skills/release-review/assets/to-dir")
	removeBlockingDirectory := actionIndex(got, "remove_directory", ".agents/skills/release-review/references/to-file")
	writeReplacementFile := actionIndex(got, "write_file", ".agents/skills/release-review/references/to-file")
	if removeBlockingFile < 0 || createReplacementDirectory <= removeBlockingFile {
		t.Fatalf("file-to-directory operations are not safely ordered: %#v", got.Operations)
	}
	if removeBlockingDirectory < 0 || writeReplacementFile <= removeBlockingDirectory {
		t.Fatalf("directory-to-file operations are not safely ordered: %#v", got.Operations)
	}
}

func TestRemovePlanRejectsExternalAndPlansCanonicalInventory(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(root, ".agents", "skills", "speckit-plan")
	writeFile(t, filepath.Join(external, "SKILL.md"), "# External\n")
	if _, err := NewRemove(root, "speckit-plan"); err == nil || !strings.Contains(err.Error(), "external") {
		t.Fatalf("expected external ownership refusal, got %v", err)
	}

	writePackage(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", map[string]string{
		"instructions.md":    "# Release review\n",
		"assets/example.txt": "example\n",
	})
	got, err := NewRemove(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "skill.remove" || !got.Executable || !got.RequiresApproval {
		t.Fatalf("unexpected remove plan controls: %#v", got)
	}
	if !hasAction(got, "remove_file", ".agents/skills/release-review/skill.yaml") || !hasAction(got, "remove_directory", ".agents/skills/release-review") {
		t.Fatalf("remove plan does not cover exact inventory: %#v", got.Operations)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "release-review", "skill.yaml")); err != nil {
		t.Fatalf("remove planning mutated the package: %v", err)
	}
}

func TestValidateIdentityRejectsModifiedPlan(t *testing.T) {
	root := t.TempDir()
	source := writePackage(t, t.TempDir(), "release-review", "1.0.0", map[string]string{"instructions.md": "# Proposed\n"})
	preview, err := NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateIdentity(preview); err != nil {
		t.Fatalf("ValidateIdentity() error = %v", err)
	}
	preview.SkillID = "changed"
	if err := ValidateIdentity(preview); err == nil {
		t.Fatal("ValidateIdentity() accepted modified plan")
	}
}

func hasAction(plan Plan, action, target string) bool {
	for _, operation := range plan.Operations {
		if operation.Action == action && operation.Path == target {
			return true
		}
	}
	return false
}

func actionIndex(plan Plan, action, target string) int {
	for index, operation := range plan.Operations {
		if operation.Action == action && operation.Path == target {
			return index
		}
	}
	return -1
}

func writePackage(t *testing.T, parent, id, version string, files map[string]string) string {
	t.Helper()
	packageRoot := filepath.Join(parent, id)
	paths := make([]string, 0, len(files))
	for relative, contents := range files {
		writeFile(t, filepath.Join(packageRoot, filepath.FromSlash(relative)), contents)
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	var resources strings.Builder
	for _, relative := range paths {
		kind := "asset"
		switch {
		case relative == "instructions.md":
			kind = "instructions"
		case strings.HasPrefix(relative, "references/"):
			kind = "reference"
		case strings.HasPrefix(relative, "scripts/"):
			kind = "script"
		case strings.HasPrefix(relative, "templates/"):
			kind = "template"
		}
		resources.WriteString(fmt.Sprintf("  - path: %s\n    kind: %s\n    sha256: %s\n", relative, kind, hash(files[relative])))
	}
	manifest := fmt.Sprintf(`schema_version: engkit.skill/v1alpha1
id: %s
version: %s
description: Reviews releases.
instructions: instructions.md
permissions: {}
compatibility:
  minimum_engkit: 0.2.0
ownership:
  owner: engineering
provenance:
  source: local
  revision: %s
resources:
%s`, id, version, version, resources.String())
	writeFile(t, filepath.Join(packageRoot, "skill.yaml"), manifest)
	return packageRoot
}

func hash(contents string) string {
	digest := sha256.Sum256([]byte(contents))
	return "sha256:" + hex.EncodeToString(digest[:])
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
