package providersync

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/templates"
	"gopkg.in/yaml.v3"
)

func TestBuiltinPilotConformsToProviderContracts(t *testing.T) {
	root := materializeBuiltinPilot(t)
	golden, err := os.ReadFile(filepath.Join("testdata", "project-guide.SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	golden = normalizeMarkdown(golden)
	targets := map[string]string{
		"codex":    ".agents/skills/project-guide/SKILL.md",
		"claude":   ".claude/skills/project-guide/SKILL.md",
		"opencode": ".opencode/skills/project-guide/SKILL.md",
	}
	for provider, target := range targets {
		t.Run(provider, func(t *testing.T) {
			first, err := Prepare(root, provider, "project-guide")
			if err != nil {
				t.Fatal(err)
			}
			second, err := Prepare(root, provider, "project-guide")
			if err != nil {
				t.Fatal(err)
			}
			if first.Plan.PlanID != second.Plan.PlanID {
				t.Fatalf("repeated plan IDs differ: %q != %q", first.Plan.PlanID, second.Plan.PlanID)
			}
			if len(first.Files) != 1 || first.Files[0].Path != target {
				t.Fatalf("materialized files = %#v", first.Files)
			}
			if !bytes.Equal(first.Files[0].Data, golden) {
				t.Fatalf("%s pilot output does not match reviewed golden bytes", provider)
			}
			if first.Files[0].SHA256 != digestBytes(golden) {
				t.Fatalf("%s digest = %q, want %q", provider, first.Files[0].SHA256, digestBytes(golden))
			}
			if first.Plan.Adapter.ContractVersion != "v1alpha1" || first.Plan.Adapter.EvidenceDate != adapterEvidenceDate {
				t.Fatalf("adapter reference = %#v", first.Plan.Adapter)
			}
			if len(first.Plan.Mappings) != 1 || first.Plan.Mappings[0].Quality != "exact" || len(first.Plan.Mappings[0].Findings) != 1 {
				t.Fatalf("mapping = %#v", first.Plan.Mappings)
			}
			assertPortableFrontmatter(t, first.Files[0].Data)
			if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(target))); !os.IsNotExist(err) {
				t.Fatalf("preparation wrote provider output: %v", err)
			}
		})
	}
}

func TestBuiltinPilotRefusesUnownedProviderOutput(t *testing.T) {
	root := materializeBuiltinPilot(t)
	target := filepath.Join(root, ".claude", "skills", "project-guide", "SKILL.md")
	writeFile(t, target, "unowned\n")
	plan, err := NewPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Executable || len(plan.Outputs) != 1 || plan.Outputs[0].Action != "conflict" {
		t.Fatalf("unowned output plan = %#v", plan)
	}
	assertConflictCode(t, plan, "target.unowned_existing")
	if got := readFile(t, target); got != "unowned\n" {
		t.Fatalf("unowned output changed: %q", got)
	}
}

func TestBuiltinPilotRefusesLinkedAndCollidingTargets(t *testing.T) {
	t.Run("non-directory ancestor", func(t *testing.T) {
		root := materializeBuiltinPilot(t)
		writeFile(t, filepath.Join(root, ".claude"), "collision\n")
		plan, err := NewPlan(root, "claude", "project-guide")
		if err != nil {
			t.Fatal(err)
		}
		if plan.Executable || len(plan.Outputs) != 1 || plan.Outputs[0].Action != "conflict" {
			t.Fatalf("ancestor collision plan = %#v", plan)
		}
		assertConflictCode(t, plan, "target.path_collision")
	})

	t.Run("linked output", func(t *testing.T) {
		root := materializeBuiltinPilot(t)
		target := filepath.Join(root, ".opencode", "skills", "project-guide", "SKILL.md")
		linked := filepath.Join(root, "linked-skill.md")
		writeFile(t, linked, "external\n")
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(linked, target); err != nil {
			if os.Getenv("CI") != "" {
				t.Fatalf("CI host cannot create the required %s symbolic-link fixture: %v", runtime.GOOS, err)
			}
			t.Skipf("host cannot create symbolic-link fixture: %v", err)
		}
		plan, err := NewPlan(root, "opencode", "project-guide")
		if err != nil {
			t.Fatal(err)
		}
		if plan.Executable || len(plan.Outputs) != 1 || plan.Outputs[0].Action != "conflict" {
			t.Fatalf("linked output plan = %#v", plan)
		}
		assertConflictCode(t, plan, "target.path_type")
		if got := readFile(t, linked); got != "external\n" {
			t.Fatalf("linked target changed: %q", got)
		}
	})
}

func materializeBuiltinPilot(t *testing.T) string {
	t.Helper()
	inventory, catalogFS, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := catalog.Find(inventory.Templates, "project-guide", "0.1.0")
	if !ok {
		t.Fatal("builtin project-guide@0.1.0 is unavailable")
	}
	definition, err := templates.Load(catalogFS, path.Join(entry.Path, "template.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := templates.Render(catalogFS, entry.Path, definition, templates.Origin{
		CatalogID:      inventory.ID,
		CatalogVersion: inventory.Version,
		TemplateSHA256: entry.SHA256,
	}, map[string]string{"project_name": "Provider Conformance Pilot"})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	packageRoot := filepath.Join(root, ".agents", "skills", "project-guide")
	for _, file := range rendered.Files {
		writeFile(t, filepath.Join(packageRoot, filepath.FromSlash(file.Path)), string(file.Data))
	}
	return root
}

func assertPortableFrontmatter(t *testing.T, data []byte) {
	t.Helper()
	parts := bytes.SplitN(data, []byte("---\n"), 3)
	if len(parts) != 3 || len(parts[0]) != 0 {
		t.Fatal("SKILL.md does not contain leading YAML frontmatter")
	}
	var decoded map[string]any
	if err := yaml.Unmarshal(parts[1], &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded["name"] != "project-guide" || decoded["description"] != "Guides safe, repository-aware engineering work for one project." {
		t.Fatalf("frontmatter = %#v", decoded)
	}
}
