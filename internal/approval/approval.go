// Package approval binds explicit user approval to an immutable plan digest.
package approval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
)

// Validate confirms that an approval applies to exactly one workspace plan.
func Validate(provided string, preview plan.Plan) error {
	return ValidateID(provided, preview.PlanID, preview.RequiresApproval)
}

// ValidateID confirms that approval applies to one immutable plan identity.
func ValidateID(provided, expected string, required bool) error {
	if !required {
		return nil
	}
	provided = strings.TrimSpace(provided)
	if provided == "" {
		return errors.New("approval is required; pass the complete plan ID with --approve")
	}
	if provided != expected {
		return fmt.Errorf("approval does not match plan %s", expected)
	}
	return nil
}
