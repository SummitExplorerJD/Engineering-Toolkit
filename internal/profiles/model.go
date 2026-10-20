package profiles

import "github.com/SummitExplorerJD/Engineering-Toolkit/internal/templates"

const (
	Schema           = "engkit.project-profile/v1alpha1"
	ListSchema       = "engkit.profile-list/v1alpha1"
	InspectionSchema = "engkit.profile-inspection/v1alpha1"
	maxProfileBytes  = 1 << 20
)

type Profile struct {
	SchemaVersion        string               `yaml:"schema_version" json:"schema_version"`
	ID                   string               `yaml:"id" json:"id"`
	Version              string               `yaml:"version" json:"version"`
	Description          string               `yaml:"description" json:"description"`
	Compatibility        Compatibility        `yaml:"compatibility" json:"compatibility"`
	Variables            []templates.Variable `yaml:"variables,omitempty" json:"variables,omitempty"`
	Skills               []SkillSelection     `yaml:"skills,omitempty" json:"skills,omitempty"`
	WorkspaceDirectories []string             `yaml:"workspace_directories,omitempty" json:"workspace_directories,omitempty"`
	WorkspaceAssets      []Asset              `yaml:"workspace_assets,omitempty" json:"workspace_assets,omitempty"`
	SourceLayout         SourceLayout         `yaml:"source_layout" json:"source_layout"`
	IntegrationRequests  []IntegrationRequest `yaml:"integration_requests,omitempty" json:"integration_requests,omitempty"`
}

type Compatibility struct {
	Ecosystems []string `yaml:"ecosystems" json:"ecosystems"`
	Modes      []string `yaml:"modes" json:"modes"`
}

type SkillSelection struct {
	TemplateID      string             `yaml:"template_id" json:"template_id"`
	TemplateVersion string             `yaml:"template_version" json:"template_version"`
	Bindings        map[string]Binding `yaml:"bindings,omitempty" json:"bindings,omitempty"`
}

type Binding struct {
	Variable string `yaml:"variable,omitempty" json:"variable,omitempty"`
	Literal  any    `yaml:"literal,omitempty" json:"literal,omitempty"`
}

type Asset struct {
	Source       string `yaml:"source" json:"source"`
	Destination  string `yaml:"destination" json:"destination"`
	Kind         string `yaml:"kind" json:"kind"`
	MediaType    string `yaml:"media_type" json:"media_type"`
	SourceSHA256 string `yaml:"source_sha256" json:"source_sha256"`
}

type SourceLayout struct {
	Policy string `yaml:"policy" json:"policy"`
}

type IntegrationRequest struct {
	Provider   string `yaml:"provider" json:"provider"`
	Capability string `yaml:"capability" json:"capability"`
}

type Reference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type ListResult struct {
	SchemaVersion  string      `json:"schema_version"`
	CatalogID      string      `json:"catalog_id"`
	CatalogVersion string      `json:"catalog_version"`
	Profiles       []Reference `json:"profiles"`
}

type Inspection struct {
	SchemaVersion   string  `json:"schema_version"`
	CatalogID       string  `json:"catalog_id"`
	CatalogVersion  string  `json:"catalog_version"`
	Profile         Profile `json:"profile"`
	DirectorySHA256 string  `json:"directory_sha256"`
}
