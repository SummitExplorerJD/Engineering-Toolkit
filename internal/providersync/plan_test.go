package providersync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPlanIsDeterministicSelectiveAndReadOnly(t *testing.T) {
	root := t.TempDir()
	writeCanonical(t, root, "project-guide", "Guides safe project work.", "# Project guide\n\nInspect before changing files.\n", "", nil)
	writeFile(t, filepath.Join(root, ".agents", "skills", "speckit-plan", "SKILL.md"), "# External Spec Kit skill\n")

	targets := map[string]string{
		"codex":    ".agents/skills/project-guide/SKILL.md",
		"claude":   ".claude/skills/project-guide/SKILL.md",
		"opencode": ".opencode/skills/project-guide/SKILL.md",
	}
	for provider, target := range targets {
		t.Run(provider, func(t *testing.T) {
			first, err := NewPlan(root, provider, "project-guide")
			if err != nil {
				t.Fatal(err)
			}
			second, err := NewPlan(root, provider, "project-guide")
			if err != nil {
				t.Fatal(err)
			}
			if first.PlanID != second.PlanID || !strings.HasPrefix(first.PlanID, "sha256:") {
				t.Fatalf("plan is not deterministic: %q != %q", first.PlanID, second.PlanID)
			}
			if !first.Executable || !first.RequiresApproval || len(first.Outputs) != 1 {
				t.Fatalf("unexpected plan: %#v", first)
			}
			if first.Outputs[0].Path != target || first.Outputs[0].Action != "create" {
				t.Fatalf("unexpected output: %#v", first.Outputs[0])
			}
			if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(target))); !os.IsNotExist(err) {
				t.Fatalf("planning wrote provider output: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(root, ".engkit", "providers", provider, "project-guide.json")); !os.IsNotExist(err) {
				t.Fatalf("planning wrote ownership state: %v", err)
			}
		})
	}
	if got := readFile(t, filepath.Join(root, ".agents", "skills", "speckit-plan", "SKILL.md")); got != "# External Spec Kit skill\n" {
		t.Fatalf("planning changed external Spec Kit content: %q", got)
	}
}

func TestNewPlanCopiesPassiveResourcesOnlyForIsolatedProviders(t *testing.T) {
	root := t.TempDir()
	resources := map[string]resourceInput{"references/guide.md": {kind: "reference", data: "# Guide\n"}}
	writeCanonical(t, root, "review", "Review changes.", "# Review\n", "", resources)

	codex, err := NewPlan(root, "codex", "review")
	if err != nil {
		t.Fatal(err)
	}
	if len(codex.Outputs) != 1 {
		t.Fatalf("Codex should reference canonical resources in place: %#v", codex.Outputs)
	}
	for _, provider := range []string{"claude", "opencode"} {
		plan, err := NewPlan(root, provider, "review")
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Outputs) != 2 || plan.Outputs[0].Path == plan.Outputs[1].Path {
			t.Fatalf("%s passive resource outputs = %#v", provider, plan.Outputs)
		}
	}
}

func TestNewPlanReportsUnsupportedAndLossyMappings(t *testing.T) {
	root := t.TempDir()
	resources := map[string]resourceInput{"scripts/run.sh": {kind: "script", data: "#!/bin/sh\n"}}
	writeCanonical(t, root, "unsafe", "Unsafe example.", "# Unsafe\n", "  network:\n    - example.com\n", resources)

	plan, err := NewPlan(root, "claude", "unsafe")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Executable || plan.Mappings[0].Quality != "unsupported" {
		t.Fatalf("unsafe mapping unexpectedly executable: %#v", plan.Mappings[0])
	}
	assertConflictCode(t, plan, "mapping.permissions_unmapped")
	assertConflictCode(t, plan, "mapping.script_unsupported")
}

func TestNewPlanRefusesUnownedAndChangedOwnedOutputs(t *testing.T) {
	root := t.TempDir()
	writeCanonical(t, root, "project-guide", "Guides safe project work.", "# Project guide\n", "", nil)
	target := filepath.Join(root, ".claude", "skills", "project-guide", "SKILL.md")
	generated := "---\nname: project-guide\ndescription: Guides safe project work.\n---\n\n# Project guide\n"
	writeFile(t, target, generated)

	unowned, err := NewPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if unowned.Executable || unowned.Outputs[0].Action != "conflict" {
		t.Fatalf("unowned output unexpectedly accepted: %#v", unowned.Outputs)
	}
	assertConflictCode(t, unowned, "target.unowned_existing")

	record := OwnershipRecord{SchemaVersion: OwnershipSchema, Provider: "claude", AdapterVersion: "v1alpha1", SkillID: "project-guide", SkillVersion: "0.1.0", PackageSHA256: unowned.Sources[0].PackageSHA256, SynchronizationID: "sha256:" + strings.Repeat("a", 64), Outputs: []OwnershipOutput{{Path: ".claude/skills/project-guide/SKILL.md", SHA256: digestBytes([]byte(generated))}}}
	writeJSON(t, filepath.Join(root, ".engkit", "providers", "claude", "project-guide.json"), record)
	owned, err := NewPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if !owned.Executable || owned.Outputs[0].Action != "unchanged" {
		t.Fatalf("exact owned output not accepted: %#v conflicts=%#v", owned.Outputs, owned.Conflicts)
	}

	writeFile(t, target, generated+"changed\n")
	changed, err := NewPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if changed.Executable || changed.Outputs[0].Action != "conflict" {
		t.Fatalf("changed owned output unexpectedly accepted: %#v", changed.Outputs)
	}
	assertConflictCode(t, changed, "target.owned_changed")
}

func TestNewPlanAcceptsOwnedCodexSidecarWithoutChangingSourceDigest(t *testing.T) {
	root := t.TempDir()
	writeCanonical(t, root, "project-guide", "Guides safe project work.", "# Project guide\n", "", nil)

	first, err := NewPlan(root, "codex", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	target := ".agents/skills/project-guide/SKILL.md"
	generated := "---\nname: project-guide\ndescription: Guides safe project work.\n---\n\n# Project guide\n"
	writeFile(t, filepath.Join(root, filepath.FromSlash(target)), generated)
	record := OwnershipRecord{
		SchemaVersion: OwnershipSchema, Provider: "codex", AdapterVersion: "v1alpha1",
		SkillID: "project-guide", SkillVersion: "0.1.0", PackageSHA256: first.Sources[0].PackageSHA256,
		SynchronizationID: first.PlanID, Outputs: []OwnershipOutput{{Path: target, SHA256: digestBytes([]byte(generated))}},
	}
	writeJSON(t, filepath.Join(root, ".engkit", "providers", "codex", "project-guide.json"), record)

	second, err := NewPlan(root, "codex", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if !second.Executable || second.Sources[0].PackageSHA256 != first.Sources[0].PackageSHA256 || second.Outputs[0].Action != "unchanged" {
		t.Fatalf("owned Codex sidecar changed canonical planning: %#v", second)
	}
}

func TestNewPlanRejectsCodexOwnershipOfCanonicalFiles(t *testing.T) {
	root := t.TempDir()
	writeCanonical(t, root, "project-guide", "Guides safe project work.", "# Project guide\n", "", nil)
	first, err := NewPlan(root, "codex", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	record := OwnershipRecord{
		SchemaVersion: OwnershipSchema, Provider: "codex", AdapterVersion: "v1alpha1",
		SkillID: "project-guide", SkillVersion: "0.1.0", PackageSHA256: first.Sources[0].PackageSHA256,
		SynchronizationID: first.PlanID,
		Outputs:           []OwnershipOutput{{Path: ".agents/skills/project-guide/skill.yaml", SHA256: first.Sources[0].ManifestSHA256}},
	}
	writeJSON(t, filepath.Join(root, ".engkit", "providers", "codex", "project-guide.json"), record)

	plan, err := NewPlan(root, "codex", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Executable {
		t.Fatalf("invalid Codex ownership record produced an executable plan: %#v", plan)
	}
	assertConflictCode(t, plan, "ownership.record_invalid")
	for _, output := range plan.Outputs {
		if output.Path == ".agents/skills/project-guide/skill.yaml" {
			t.Fatalf("planner proposed a canonical manifest action: %#v", output)
		}
	}
}

func TestNewPlanAllCanonicalSkillsAreSorted(t *testing.T) {
	root := t.TempDir()
	writeCanonical(t, root, "zeta", "Zeta.", "# Zeta\n", "", nil)
	writeCanonical(t, root, "alpha", "Alpha.", "# Alpha\n", "", nil)
	plan, err := NewPlan(root, "opencode", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Sources) != 2 || plan.Sources[0].ID != "alpha" || plan.Sources[1].ID != "zeta" {
		t.Fatalf("sources are not sorted: %#v", plan.Sources)
	}
	if plan.Executable {
		t.Fatal("multi-skill plan unexpectedly executable")
	}
	assertConflictCode(t, plan, "execution.multi_skill_unsupported")
}

func TestPrepareBindsExactBytesAndOwnership(t *testing.T) {
	root := t.TempDir()
	writeCanonical(t, root, "project-guide", "Guides safe project work.", "# Project guide\n", "", nil)
	prepared, err := Prepare(root, "opencode", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Files) != 1 || prepared.Files[0].SHA256 != digestBytes(prepared.Files[0].Data) {
		t.Fatalf("unexpected materialization: %#v", prepared.Files)
	}
	if prepared.Ownership.SynchronizationID != prepared.Plan.PlanID || len(prepared.Ownership.Outputs) != 1 || prepared.Ownership.Outputs[0].SHA256 != prepared.Files[0].SHA256 {
		t.Fatalf("ownership does not bind materialization: %#v", prepared.Ownership)
	}
}

func TestNewUnsyncPlanUsesOnlyExactOwnership(t *testing.T) {
	root := t.TempDir()
	writeCanonical(t, root, "project-guide", "Guides safe project work.", "# Project guide\n", "", nil)
	prepared, err := Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range prepared.Files {
		writeFile(t, filepath.Join(root, filepath.FromSlash(file.Path)), string(file.Data))
	}
	writeJSON(t, filepath.Join(root, ".engkit", "providers", "claude", "project-guide.json"), prepared.Ownership)

	plan, err := NewUnsyncPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Executable || plan.Operation != "remove" || len(plan.Sources) != 0 || len(plan.Outputs) != len(prepared.Files) {
		t.Fatalf("unexpected unsync plan: %#v", plan)
	}
	for _, output := range plan.Outputs {
		if output.Action != "remove" {
			t.Fatalf("unsync output = %#v", output)
		}
	}

	writeFile(t, filepath.Join(root, filepath.FromSlash(prepared.Files[0].Path)), "changed\n")
	changed, err := NewUnsyncPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if changed.Executable {
		t.Fatal("changed owned output produced executable unsync plan")
	}
	assertConflictCode(t, changed, "target.owned_changed")
}

func TestOwnershipBlockersFailClosed(t *testing.T) {
	root := t.TempDir()
	blockers, err := OwnershipBlockers(root, "project-guide")
	if err != nil || len(blockers) != 0 {
		t.Fatalf("empty blockers = %#v, %v", blockers, err)
	}
	writeFile(t, filepath.Join(root, ".engkit", "providers", "codex", "project-guide.json"), "malformed")
	blockers, err = OwnershipBlockers(root, "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if len(blockers) != 1 || blockers[0] != ".engkit/providers/codex/project-guide.json" {
		t.Fatalf("blockers = %#v", blockers)
	}
}

type resourceInput struct {
	kind string
	data string
}

func writeCanonical(t *testing.T, root, id, description, instructions, permissionYAML string, resources map[string]resourceInput) {
	t.Helper()
	packageRoot := filepath.Join(root, ".agents", "skills", id)
	writeFile(t, filepath.Join(packageRoot, "instructions.md"), instructions)
	resourceYAML := "  - path: instructions.md\n    kind: instructions\n    sha256: " + digestString(instructions) + "\n"
	paths := make([]string, 0, len(resources))
	for relative := range resources {
		paths = append(paths, relative)
	}
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j] < paths[i] {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
	for _, relative := range paths {
		resource := resources[relative]
		writeFile(t, filepath.Join(packageRoot, filepath.FromSlash(relative)), resource.data)
		resourceYAML += "  - path: " + relative + "\n    kind: " + resource.kind + "\n    sha256: " + digestString(resource.data) + "\n"
	}
	permissionsBlock := "permissions: {}\n"
	if permissionYAML != "" {
		permissionsBlock = "permissions:\n" + permissionYAML
	}
	manifest := "schema_version: engkit.skill/v1alpha1\nid: " + id + "\nversion: 0.1.0\ndescription: " + description + "\ninstructions: instructions.md\ncapabilities:\n  - project-guidance\n" + permissionsBlock + "compatibility:\n  minimum_engkit: 0.2.1\nownership:\n  owner: tests\nprovenance:\n  source: test\n  revision: fixture\nresources:\n" + resourceYAML
	writeFile(t, filepath.Join(packageRoot, "skill.yaml"), manifest)
}

func writeFile(t *testing.T, filename, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, filename string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filename, string(data)+"\n")
}

func readFile(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func digestString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func assertConflictCode(t *testing.T, plan Plan, code string) {
	t.Helper()
	for _, conflict := range plan.Conflicts {
		if conflict.Code == code {
			return
		}
	}
	t.Fatalf("conflict %q not found in %#v", code, plan.Conflicts)
}
