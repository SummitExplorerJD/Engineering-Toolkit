package approval

import (
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
)

func TestValidateRequiresExactPlanID(t *testing.T) {
	preview := plan.Plan{PlanID: "sha256:expected", RequiresApproval: true}

	for _, candidate := range []string{"", "sha256:other"} {
		if err := Validate(candidate, preview); err == nil {
			t.Fatalf("Validate(%q) expected an error", candidate)
		}
	}
	if err := Validate("sha256:expected", preview); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
