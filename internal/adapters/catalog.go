package adapters

import (
	"errors"
	"sort"
	"strings"
)

const CatalogSchema = "engkit.adapter-catalog/v1alpha1"

// Support describes how an adapter exposes a capability without implying
// feature parity between providers.
type Support string

const (
	SupportNative      Support = "native"
	SupportViaSpecKit  Support = "via_spec_kit"
	SupportRuntimeOnly Support = "runtime_only"
	SupportUnsupported Support = "unsupported"
)

// Capability records one provider-specific integration boundary.
type Capability struct {
	Name    string  `json:"name"`
	Support Support `json:"support"`
	Path    string  `json:"path,omitempty"`
	Notes   string  `json:"notes"`
}

// Adapter is immutable provider metadata used for inspection and planning.
type Adapter struct {
	ID                  string       `json:"id"`
	DisplayName         string       `json:"display_name"`
	SpecKitIntegration  string       `json:"spec_kit_integration"`
	SpecKitArtifactRoot string       `json:"spec_kit_artifact_root"`
	Capabilities        []Capability `json:"capabilities"`
}

// Catalog is versioned machine-readable adapter metadata.
type Catalog struct {
	SchemaVersion string    `json:"schema_version"`
	Adapters      []Adapter `json:"adapters"`
}

var catalog = []Adapter{
	{
		ID:                  "claude",
		DisplayName:         "Claude Code",
		SpecKitIntegration:  "claude",
		SpecKitArtifactRoot: ".claude/skills",
		Capabilities: []Capability{
			{Name: "project_instructions", Support: SupportNative, Path: "CLAUDE.md", Notes: "Claude Code reads a repository-root CLAUDE.md as project memory."},
			{Name: "project_skills", Support: SupportNative, Path: ".claude/skills", Notes: "Project skills are directories containing SKILL.md."},
			{Name: "spec_kit_workflows", Support: SupportViaSpecKit, Path: ".claude/skills/speckit-*/SKILL.md", Notes: "Official Spec Kit uses its Claude integration key and skills layout."},
			{Name: "approval_policy", Support: SupportRuntimeOnly, Notes: "Runtime permissions remain provider configuration, not an engkit-generated repository policy."},
		},
	},
	{
		ID:                  "codex",
		DisplayName:         "Codex",
		SpecKitIntegration:  "codex",
		SpecKitArtifactRoot: ".agents/skills",
		Capabilities: []Capability{
			{Name: "project_instructions", Support: SupportNative, Path: "AGENTS.md", Notes: "Codex discovers repository instructions from root to working directory."},
			{Name: "project_skills", Support: SupportNative, Path: ".agents/skills", Notes: "Repository-local skills use SKILL.md with progressively loaded resources."},
			{Name: "spec_kit_workflows", Support: SupportViaSpecKit, Path: ".agents/skills/speckit-*/SKILL.md", Notes: "Official Spec Kit installs Codex workflow skills into the shared skills root."},
			{Name: "approval_policy", Support: SupportRuntimeOnly, Notes: "Sandbox and approval behavior remain Codex runtime policy."},
		},
	},
	{
		ID:                  "opencode",
		DisplayName:         "OpenCode",
		SpecKitIntegration:  "opencode",
		SpecKitArtifactRoot: ".opencode/commands",
		Capabilities: []Capability{
			{Name: "project_instructions", Support: SupportNative, Path: "AGENTS.md", Notes: "OpenCode V2 discovers AGENTS.md and does not use CLAUDE.md as a fallback."},
			{Name: "project_skills", Support: SupportNative, Path: ".opencode/skills", Notes: "Native project skills use SKILL.md; compatibility discovery of .agents and .claude does not transfer ownership."},
			{Name: "custom_commands", Support: SupportNative, Path: ".opencode/commands", Notes: "Project command definitions are Markdown files under the provider directory."},
			{Name: "spec_kit_workflows", Support: SupportViaSpecKit, Path: ".opencode/commands", Notes: "The exact generated inventory must be confirmed against the pinned Spec Kit release."},
			{Name: "approval_policy", Support: SupportRuntimeOnly, Path: "opencode.json", Notes: "OpenCode tool permissions can allow, deny, or ask; engkit does not generate them in this milestone."},
		},
	},
}

// List returns a stable copy of the supported adapter catalog.
func List() Catalog {
	items := make([]Adapter, len(catalog))
	for i, item := range catalog {
		items[i] = clone(item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return Catalog{SchemaVersion: CatalogSchema, Adapters: items}
}

// Get resolves one explicitly selected provider.
func Get(id string) (Adapter, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return Adapter{}, errors.New("provider is required")
	}
	for _, item := range catalog {
		if item.ID == id {
			return clone(item), nil
		}
	}
	return Adapter{}, errors.New("unsupported provider: " + id)
}

func clone(adapter Adapter) Adapter {
	adapter.Capabilities = append([]Capability(nil), adapter.Capabilities...)
	return adapter
}
