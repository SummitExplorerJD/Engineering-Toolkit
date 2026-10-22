// Package providersync builds deterministic, read-only plans for rendering
// canonical skills into explicitly selected provider formats.
package providersync

const (
	PlanSchema      = "engkit.provider-sync-plan/v1alpha1"
	OwnershipSchema = "engkit.provider-ownership/v1alpha1"
)

type AdapterReference struct {
	Provider        string `json:"provider"`
	ContractVersion string `json:"contract_version"`
	EvidenceDate    string `json:"evidence_date"`
	TargetRoot      string `json:"target_root"`
}

type SourceReference struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	PackageSHA256  string `json:"package_sha256"`
	ManifestSHA256 string `json:"manifest_sha256"`
}

type Observation struct {
	Path      string `json:"path"`
	State     string `json:"state"`
	SHA256    string `json:"sha256,omitempty"`
	Ownership string `json:"ownership"`
}

type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type Mapping struct {
	SkillID  string    `json:"skill_id"`
	Quality  string    `json:"quality"`
	Findings []Finding `json:"findings"`
}

type Output struct {
	Path         string `json:"path"`
	Action       string `json:"action"`
	Owner        string `json:"owner"`
	SourcePath   string `json:"source_path,omitempty"`
	Size         int64  `json:"size,omitempty"`
	BeforeSHA256 string `json:"before_sha256,omitempty"`
	AfterSHA256  string `json:"after_sha256,omitempty"`
}

type Plan struct {
	SchemaVersion     string            `json:"schema_version"`
	PlanID            string            `json:"plan_id,omitempty"`
	Operation         string            `json:"operation"`
	Root              string            `json:"root"`
	Provider          string            `json:"provider"`
	Adapter           AdapterReference  `json:"adapter"`
	Sources           []SourceReference `json:"sources"`
	Observations      []Observation     `json:"observations"`
	Mappings          []Mapping         `json:"mappings"`
	Outputs           []Output          `json:"outputs"`
	Conflicts         []Finding         `json:"conflicts"`
	AffectedPaths     []string          `json:"affected_paths"`
	Risks             []string          `json:"risks"`
	RequiredApprovals []string          `json:"required_approvals"`
	Verification      []string          `json:"verification"`
	RequiresApproval  bool              `json:"requires_approval"`
	Executable        bool              `json:"executable"`
}

type OwnershipOutput struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type OwnershipRecord struct {
	SchemaVersion     string            `json:"schema_version"`
	Provider          string            `json:"provider"`
	AdapterVersion    string            `json:"adapter_contract_version"`
	SkillID           string            `json:"skill_id"`
	SkillVersion      string            `json:"skill_version"`
	PackageSHA256     string            `json:"package_sha256"`
	SynchronizationID string            `json:"synchronization_plan_id"`
	Outputs           []OwnershipOutput `json:"outputs"`
}

type renderedFile struct {
	path       string
	sourcePath string
	data       []byte
}
