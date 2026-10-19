// Package skills discovers and validates provider-neutral skill packages.
package skills

import "errors"

const (
	ManifestSchema   = "engkit.skill/v1alpha1"
	ListSchema       = "engkit.skill-list/v1alpha1"
	InspectionSchema = "engkit.skill-inspection/v1alpha1"
	ValidationSchema = "engkit.skill-validation/v1alpha1"

	ClassificationCanonical = "canonical"
	ClassificationExternal  = "external"
	ClassificationInvalid   = "invalid"
	ClassificationConflict  = "conflict"
)

// ErrValidationFailed indicates that validation completed and found invalid
// or conflicting canonical packages. The structured response remains useful.
var ErrValidationFailed = errors.New("skill validation failed")

// Manifest is the canonical, provider-neutral skill contract.
type Manifest struct {
	SchemaVersion string         `yaml:"schema_version" json:"schema_version"`
	ID            string         `yaml:"id" json:"id"`
	Version       string         `yaml:"version" json:"version"`
	Description   string         `yaml:"description" json:"description"`
	Instructions  string         `yaml:"instructions" json:"instructions"`
	Capabilities  []string       `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Inputs        []ValueSpec    `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs       []ValueSpec    `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Permissions   *Permissions   `yaml:"permissions" json:"permissions"`
	Compatibility *Compatibility `yaml:"compatibility" json:"compatibility"`
	Dependencies  []Dependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Ownership     *Ownership     `yaml:"ownership" json:"ownership"`
	Provenance    *Provenance    `yaml:"provenance" json:"provenance"`
	Resources     []Resource     `yaml:"resources" json:"resources"`
}

// ValueSpec describes one named input or output without provider semantics.
type ValueSpec struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

// Permissions declares requested capabilities. It never grants authority.
type Permissions struct {
	FilesystemRead  []string `yaml:"filesystem_read,omitempty" json:"filesystem_read,omitempty"`
	FilesystemWrite []string `yaml:"filesystem_write,omitempty" json:"filesystem_write,omitempty"`
	Network         []string `yaml:"network,omitempty" json:"network,omitempty"`
	Process         []string `yaml:"process,omitempty" json:"process,omitempty"`
}

// Compatibility declares the minimum engkit version and known provider targets.
type Compatibility struct {
	MinimumEngkit string   `yaml:"minimum_engkit" json:"minimum_engkit"`
	Providers     []string `yaml:"providers,omitempty" json:"providers,omitempty"`
}

// Dependency identifies an exact canonical skill dependency.
type Dependency struct {
	ID      string `yaml:"id" json:"id"`
	Version string `yaml:"version" json:"version"`
}

// Ownership identifies the accountable owner and optional maintainers.
type Ownership struct {
	Owner       string   `yaml:"owner" json:"owner"`
	Maintainers []string `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
}

// Provenance records where this package came from without resolving it.
type Provenance struct {
	Source   string `yaml:"source" json:"source"`
	Revision string `yaml:"revision" json:"revision"`
}

// Resource declares one integrity-protected file inside the package.
type Resource struct {
	Path   string `yaml:"path" json:"path"`
	Kind   string `yaml:"kind" json:"kind"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// Diagnostic is a deterministic validation finding.
type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

// Package reports one discovered directory under the canonical skills root.
type Package struct {
	ID             string       `json:"id"`
	Directory      string       `json:"directory"`
	Path           string       `json:"path"`
	Classification string       `json:"classification"`
	Manifest       *Manifest    `json:"manifest,omitempty"`
	Diagnostics    []Diagnostic `json:"diagnostics,omitempty"`
}

// ListResponse is a deterministic workspace skill inventory.
type ListResponse struct {
	SchemaVersion string    `json:"schema_version"`
	Root          string    `json:"root"`
	SkillsRoot    string    `json:"skills_root"`
	Packages      []Package `json:"packages"`
}

// Inspection reports one requested package.
type Inspection struct {
	SchemaVersion string  `json:"schema_version"`
	Root          string  `json:"root"`
	Package       Package `json:"package"`
}

// Validation reports all selected validation results.
type Validation struct {
	SchemaVersion string    `json:"schema_version"`
	Root          string    `json:"root"`
	Valid         bool      `json:"valid"`
	Packages      []Package `json:"packages"`
}
