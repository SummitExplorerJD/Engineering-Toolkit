package catalog

const (
	Schema             = "engkit.catalog/v1alpha1"
	maxDefinitionBytes = 1 << 20
)

type Catalog struct {
	SchemaVersion string  `yaml:"schema_version" json:"schema_version"`
	ID            string  `yaml:"id" json:"id"`
	Version       string  `yaml:"version" json:"version"`
	MinimumEngkit string  `yaml:"minimum_engkit" json:"minimum_engkit"`
	Templates     []Entry `yaml:"templates" json:"templates"`
	Profiles      []Entry `yaml:"profiles" json:"profiles"`
	SHA256        string  `yaml:"-" json:"sha256"`
}

type Entry struct {
	ID      string `yaml:"id" json:"id"`
	Version string `yaml:"version" json:"version"`
	Path    string `yaml:"path" json:"path"`
	SHA256  string `yaml:"sha256" json:"sha256"`
}
