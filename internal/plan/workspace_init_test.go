package plan

import (
	"strings"
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

func TestWorkspaceInitPlanIsDeterministic(t *testing.T) {
	snapshot := workspace.Inspection{Root: "C:/work/project"}

	first, err := NewWorkspaceInit(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewWorkspaceInit(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	if first.PlanID != second.PlanID {
		t.Fatalf("PlanID changed: %q != %q", first.PlanID, second.PlanID)
	}
	if !strings.HasPrefix(first.PlanID, "sha256:") {
		t.Fatalf("PlanID = %q", first.PlanID)
	}
	if !first.RequiresApproval {
		t.Fatal("workspace initialization must require approval")
	}
	if len(first.Operations) != 12 {
		t.Fatalf("Operations count = %d", len(first.Operations))
	}
	if first.Operations[6].Path != ".engkit/recovery" {
		t.Fatalf("recovery operation = %#v", first.Operations[6])
	}
}

func TestValidateIdentityRejectsChangedWorkspacePlan(t *testing.T) {
	preview, err := NewWorkspaceInit(workspace.Inspection{Root: "C:/work/project"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateIdentity(preview); err != nil {
		t.Fatal(err)
	}
	preview.Kind = "changed"
	if err := ValidateIdentity(preview); err == nil {
		t.Fatal("ValidateIdentity(changed) succeeded")
	}
}

func TestWorkspaceInitPlanIsEmptyWhenInitialized(t *testing.T) {
	got, err := NewWorkspaceInit(workspace.Inspection{
		Root:      "/work/project",
		HasEngkit: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Operations == nil {
		t.Fatal("Operations must encode as an empty array, not null")
	}
	if len(got.Operations) != 0 {
		t.Fatalf("Operations count = %d", len(got.Operations))
	}
}
