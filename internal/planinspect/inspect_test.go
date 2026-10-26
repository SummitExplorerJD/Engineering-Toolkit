package planinspect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

func TestInspectWorkspacePlan(t *testing.T) {
	root := t.TempDir()
	resolved, err := workspace.ValidateRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.NewWorkspaceInit(workspace.Inspection{Root: resolved})
	if err != nil {
		t.Fatal(err)
	}
	writePlan(t, root, ".engkit/plans/workspace-init.json", preview)
	result, err := Inspect(root, ".engkit/plans/workspace-init.json")
	if err != nil {
		t.Fatal(err)
	}
	if result.PlanID != preview.PlanID || !result.IdentityValid || result.Executable != nil {
		t.Fatalf("Inspect() = %#v", result)
	}
}

func TestInspectRejectsChangedAndEscapingPlans(t *testing.T) {
	root := t.TempDir()
	resolved, err := workspace.ValidateRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.NewWorkspaceInit(workspace.Inspection{Root: resolved})
	if err != nil {
		t.Fatal(err)
	}
	preview.Kind = "changed"
	writePlan(t, root, ".engkit/plans/changed.json", preview)
	if _, err := Inspect(root, ".engkit/plans/changed.json"); err == nil {
		t.Fatal("changed plan succeeded")
	}
	if _, err := Inspect(root, "../outside.json"); err == nil {
		t.Fatal("escaping path succeeded")
	}
}

func TestInspectRejectsUnknownFieldsAndLinkedPlans(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".engkit", "plans", "unknown.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":"engkit.plan/v1alpha1","unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(root, ".engkit/plans/unknown.json"); err == nil {
		t.Fatal("unknown field succeeded")
	}

	target := filepath.Join(root, "target.json")
	if err := os.WriteFile(target, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, ".engkit", "plans", "linked.json")
	if err := os.Symlink(target, linked); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := Inspect(root, ".engkit/plans/linked.json"); err == nil {
		t.Fatal("linked plan succeeded")
	}
}

func writePlan(t *testing.T, root, relative string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
