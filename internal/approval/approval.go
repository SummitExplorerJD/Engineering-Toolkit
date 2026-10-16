// Package approval binds explicit user approval to an immutable plan digest.
package approval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
)

// Validate confirms that an approval applies to exactly one plan.
func Validate(provided string, preview plan.Plan) error {
	if !preview.RequiresApproval {
		return nil
	}
	provided = strings.TrimSpace(provided)
	if provided == "" {
		return errors.New("approval is required; pass the complete plan ID with --approve")
	}
	if provided != preview.PlanID {
		return fmt.Errorf("approval does not match plan %s", preview.PlanID)
	}
	return nil
}
