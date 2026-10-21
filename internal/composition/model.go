// Package composition builds deterministic project composition plans.
package composition

import (
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/profiles"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	PlanSchema       = "engkit.composition-plan/v1alpha1"
	InspectionSchema = "engkit.project-inspection/v1alpha1"
)

type CatalogReference struct {
	ID         string `json:"id"`
	Version    string `json:"version"`
	SHA256     string `json:"sha256"`
	CLIVersion string `json:"cli_version"`
}

type ContentReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type ResolvedVariable struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type ObservedPath struct {
	Path      string `json:"path"`
	State     string `json:"state"`
	Ownership string `json:"ownership"`
	SHA256    string `json:"sha256,omitempty"`
}

type WorkspaceFacts struct {
	Mode         string         `json:"mode"`
	TargetState  string         `json:"target_state"`
	TreeSHA256   string         `json:"tree_sha256,omitempty"`
	Ecosystems   []string       `json:"ecosystems"`
	SourceLayout string         `json:"source_layout"`
	Observed     []ObservedPath `json:"observed_paths"`
}

type RenderedOutput struct {
	Path   string `json:"path"`
	Owner  string `json:"owner"`
	Kind   string `json:"kind"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// MaterializedFile is a validated rendered output kept outside the JSON plan.
type MaterializedFile struct {
	RenderedOutput
	Data []byte `json:"-"`
}

type Conflict struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type Precondition struct {
	Path          string `json:"path"`
	ExpectedState string `json:"expected_state"`
	SHA256        string `json:"sha256,omitempty"`
}

type Operation struct {
	Action      string `json:"action"`
	Path        string `json:"path"`
	Owner       string `json:"owner"`
	AfterSHA256 string `json:"after_sha256,omitempty"`
	Reason      string `json:"reason"`
}

type Plan struct {
	SchemaVersion     string             `json:"schema_version"`
	Kind              string             `json:"kind"`
	PlanID            string             `json:"plan_id"`
	TargetRoot        string             `json:"target_root"`
	Catalog           CatalogReference   `json:"catalog"`
	Profile           ContentReference   `json:"profile"`
	Templates         []ContentReference `json:"templates"`
	ResolvedVariables []ResolvedVariable `json:"resolved_variables"`
	WorkspaceFacts    WorkspaceFacts     `json:"workspace_facts"`
	RenderedOutputs   []RenderedOutput   `json:"rendered_outputs"`
	Conflicts         []Conflict         `json:"conflicts"`
	Preconditions     []Precondition     `json:"preconditions"`
	Operations        []Operation        `json:"operations"`
	RequiresApproval  bool               `json:"requires_approval"`
	Executable        bool               `json:"executable"`
	Verification      []string           `json:"verification"`
	Recovery          []string           `json:"recovery"`
}

type Inspection struct {
	SchemaVersion   string                        `json:"schema_version"`
	Workspace       workspace.Inspection          `json:"workspace"`
	Recommendations profiles.RecommendationResult `json:"profile_recommendations"`
}
