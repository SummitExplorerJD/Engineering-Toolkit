package templates

import "github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"

const (
	Schema             = "engkit.skill-template/v1alpha1"
	ListSchema         = "engkit.template-list/v1alpha1"
	InspectionSchema   = "engkit.template-inspection/v1alpha1"
	ScaffoldSchema     = "engkit.template-scaffold-result/v1alpha1"
	maxDefinitionBytes = 1 << 20
)

type Template struct {
	SchemaVersion string     `yaml:"schema_version" json:"schema_version"`
	ID            string     `yaml:"id" json:"id"`
	Version       string     `yaml:"version" json:"version"`
	Description   string     `yaml:"description" json:"description"`
	Variables     []Variable `yaml:"variables,omitempty" json:"variables,omitempty"`
	Skill         SkillSpec  `yaml:"skill" json:"skill"`
	Files         []File     `yaml:"files" json:"files"`
}

type Variable struct {
	Name          string `yaml:"name" json:"name"`
	Type          string `yaml:"type" json:"type"`
	Description   string `yaml:"description" json:"description"`
	Required      bool   `yaml:"required" json:"required"`
	Default       any    `yaml:"default,omitempty" json:"default,omitempty"`
	MinLength     *int   `yaml:"min_length,omitempty" json:"min_length,omitempty"`
	MaxLength     *int   `yaml:"max_length,omitempty" json:"max_length,omitempty"`
	Pattern       string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	AllowedValues []any  `yaml:"allowed_values,omitempty" json:"allowed_values,omitempty"`
	Minimum       *int64 `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	Maximum       *int64 `yaml:"maximum,omitempty" json:"maximum,omitempty"`
}

type SkillSpec struct {
	ID            string                `yaml:"id" json:"id"`
	Version       string                `yaml:"version" json:"version"`
	Description   string                `yaml:"description" json:"description"`
	Instructions  string                `yaml:"instructions" json:"instructions"`
	Capabilities  []string              `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Inputs        []skills.ValueSpec    `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs       []skills.ValueSpec    `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Permissions   *skills.Permissions   `yaml:"permissions" json:"permissions"`
	Compatibility *skills.Compatibility `yaml:"compatibility" json:"compatibility"`
	Dependencies  []skills.Dependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Ownership     *skills.Ownership     `yaml:"ownership" json:"ownership"`
}

type File struct {
	Source       string `yaml:"source" json:"source"`
	Destination  string `yaml:"destination" json:"destination"`
	Kind         string `yaml:"kind" json:"kind"`
	MediaType    string `yaml:"media_type" json:"media_type"`
	SourceSHA256 string `yaml:"source_sha256" json:"source_sha256"`
}

type Inspection struct {
	SchemaVersion   string   `json:"schema_version"`
	CatalogID       string   `json:"catalog_id"`
	CatalogVersion  string   `json:"catalog_version"`
	Template        Template `json:"template"`
	DirectorySHA256 string   `json:"directory_sha256"`
}

type ListResult struct {
	SchemaVersion  string      `json:"schema_version"`
	CatalogID      string      `json:"catalog_id"`
	CatalogVersion string      `json:"catalog_version"`
	Templates      []Reference `json:"templates"`
}

type Reference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type ScaffoldResult struct {
	SchemaVersion string   `json:"schema_version"`
	Output        string   `json:"output"`
	TemplateID    string   `json:"template_id"`
	Version       string   `json:"version"`
	Created       []string `json:"created"`
}
