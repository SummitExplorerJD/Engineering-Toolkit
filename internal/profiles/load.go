package profiles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/schemautil"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/templates"
)

var placeholderPattern = regexp.MustCompile(`\{\{engkit\.([a-z][a-z0-9]*(?:_[a-z0-9]+)*)\}\}`)

func Load(fsys fs.FS, definition string) (Profile, error) {
	if path.Base(definition) != "profile.yaml" {
		return Profile{}, fmt.Errorf("profile definition must be named profile.yaml")
	}
	data, err := fs.ReadFile(fsys, definition)
	if err != nil {
		return Profile{}, fmt.Errorf("read profile definition: %w", err)
	}
	var profile Profile
	if err := schemautil.DecodeYAML(data, maxProfileBytes, definition, &profile); err != nil {
		return Profile{}, err
	}
	if err := Validate(fsys, path.Dir(definition), profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func Validate(fsys fs.FS, root string, profile Profile) error {
	if profile.SchemaVersion != Schema {
		return fmt.Errorf("schema_version must be %q", Schema)
	}
	if !schemautil.ValidKebab(profile.ID) || !schemautil.ValidSemver(profile.Version) {
		return fmt.Errorf("profile identity or version is invalid")
	}
	if strings.TrimSpace(profile.Description) != profile.Description || profile.Description == "" {
		return fmt.Errorf("profile description must be normalized and non-empty")
	}
	variables, err := templates.ValidateVariables(profile.Variables)
	if err != nil {
		return fmt.Errorf("profile variables: %w", err)
	}
	if err := validateCompatibility(profile.Compatibility); err != nil {
		return err
	}
	if profile.SourceLayout.Policy != "preserve" && profile.SourceLayout.Policy != "ecosystem-default" && profile.SourceLayout.Policy != "src" {
		return fmt.Errorf("unsupported source_layout policy %q", profile.SourceLayout.Policy)
	}
	seenSkills := make(map[string]struct{}, len(profile.Skills))
	for _, selection := range profile.Skills {
		if !schemautil.ValidKebab(selection.TemplateID) || !schemautil.ValidSemver(selection.TemplateVersion) {
			return fmt.Errorf("skill template selection is invalid")
		}
		key := selection.TemplateID + "@" + selection.TemplateVersion
		if _, exists := seenSkills[key]; exists {
			return fmt.Errorf("duplicate skill template selection %q", key)
		}
		seenSkills[key] = struct{}{}
		for name, binding := range selection.Bindings {
			if !schemautil.ValidSnake(name) {
				return fmt.Errorf("binding name %q is invalid", name)
			}
			if (binding.Variable == "") == (binding.Literal == nil) {
				return fmt.Errorf("binding %q must define exactly one of variable or literal", name)
			}
			if binding.Variable != "" {
				if _, exists := variables[binding.Variable]; !exists {
					return fmt.Errorf("binding %q references unknown variable %q", name, binding.Variable)
				}
			}
		}
	}
	if err := validateDirectories(profile.WorkspaceDirectories); err != nil {
		return err
	}
	if err := validateAssets(fsys, root, profile.WorkspaceAssets, variables); err != nil {
		return err
	}
	seenRequests := make(map[string]struct{}, len(profile.IntegrationRequests))
	for _, request := range profile.IntegrationRequests {
		if !schemautil.ValidKebab(request.Provider) || !schemautil.ValidKebab(request.Capability) {
			return fmt.Errorf("integration request is invalid")
		}
		key := request.Provider + "/" + request.Capability
		if _, exists := seenRequests[key]; exists {
			return fmt.Errorf("duplicate integration request %q", key)
		}
		seenRequests[key] = struct{}{}
	}
	return nil
}

func validateCompatibility(value Compatibility) error {
	seen := make(map[string]struct{})
	for _, ecosystem := range value.Ecosystems {
		if !schemautil.ValidKebab(ecosystem) {
			return fmt.Errorf("ecosystem %q is invalid", ecosystem)
		}
		if _, exists := seen[ecosystem]; exists {
			return fmt.Errorf("ecosystem %q is duplicated", ecosystem)
		}
		seen[ecosystem] = struct{}{}
	}
	seen = make(map[string]struct{})
	for _, mode := range value.Modes {
		if mode != "new-project" && mode != "existing-workspace" {
			return fmt.Errorf("compatibility mode %q is invalid", mode)
		}
		if _, exists := seen[mode]; exists {
			return fmt.Errorf("compatibility mode %q is duplicated", mode)
		}
		seen[mode] = struct{}{}
	}
	if len(value.Modes) == 0 {
		return fmt.Errorf("compatibility modes must not be empty")
	}
	return nil
}

func validateDirectories(values []string) error {
	allowed := map[string]struct{}{".agents/context": {}, ".agents/conventions": {}, ".agents/skills": {}, ".agents/workflows": {}}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("workspace directory %q is not allowed", value)
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("workspace directory %q is duplicated", value)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateAssets(fsys fs.FS, root string, assets []Asset, variables map[string]struct{}) error {
	seen := make(map[string]struct{}, len(assets))
	seenSources := make(map[string]struct{}, len(assets))
	destinations := make([]string, 0, len(assets))
	for _, asset := range assets {
		if err := schemautil.ValidateRelative(asset.Source); err != nil || !strings.HasPrefix(asset.Source, "files/") {
			return fmt.Errorf("asset source %q is invalid", asset.Source)
		}
		if err := schemautil.ValidateRelative(asset.Destination); err != nil {
			return err
		}
		if !validAssetDestination(asset.Kind, asset.Destination) {
			return fmt.Errorf("asset kind %q does not match destination %q", asset.Kind, asset.Destination)
		}
		key := strings.ToLower(asset.Destination)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate or case-colliding asset destination %q", asset.Destination)
		}
		seen[key] = struct{}{}
		destinations = append(destinations, key)
		sourceKey := strings.ToLower(asset.Source)
		if _, exists := seenSources[sourceKey]; exists {
			return fmt.Errorf("duplicate or case-colliding asset source %q", asset.Source)
		}
		seenSources[sourceKey] = struct{}{}
		if asset.MediaType != "text" && asset.MediaType != "binary" {
			return fmt.Errorf("asset %q has invalid media_type", asset.Source)
		}
		if !schemautil.ValidDigest(asset.SourceSHA256) {
			return fmt.Errorf("asset %q has invalid source_sha256", asset.Source)
		}
		contents, err := fs.ReadFile(fsys, path.Join(root, asset.Source))
		if err != nil {
			return fmt.Errorf("read asset %q: %w", asset.Source, err)
		}
		digest := sha256.Sum256(contents)
		if "sha256:"+hex.EncodeToString(digest[:]) != asset.SourceSHA256 {
			return fmt.Errorf("asset %q digest mismatch", asset.Source)
		}
		if asset.MediaType == "text" {
			if !utf8.Valid(contents) {
				return fmt.Errorf("asset %q is not valid UTF-8", asset.Source)
			}
			for _, match := range placeholderPattern.FindAllSubmatch(contents, -1) {
				if _, exists := variables[string(match[1])]; !exists {
					return fmt.Errorf("asset %q uses undeclared variable %q", asset.Source, match[1])
				}
			}
			if strings.Contains(string(placeholderPattern.ReplaceAll(contents, nil)), "{{engkit.") {
				return fmt.Errorf("asset %q contains a malformed placeholder", asset.Source)
			}
		} else if placeholderPattern.Match(contents) {
			return fmt.Errorf("binary asset %q contains a placeholder", asset.Source)
		}
	}
	for _, destination := range destinations {
		for parent := path.Dir(destination); parent != "."; parent = path.Dir(parent) {
			if _, exists := seen[parent]; exists {
				return fmt.Errorf("asset destinations %q and %q have a file/directory collision", parent, destination)
			}
		}
	}
	inventory, err := schemautil.RegularInventory(fsys, root)
	if err != nil {
		return fmt.Errorf("inspect profile inventory: %w", err)
	}
	if len(inventory) != len(seenSources)+1 {
		return fmt.Errorf("profile file inventory does not match declared assets")
	}
	for _, relative := range inventory {
		if relative == "profile.yaml" {
			continue
		}
		if _, exists := seenSources[strings.ToLower(relative)]; !exists {
			return fmt.Errorf("profile asset source %q is undeclared", relative)
		}
	}
	return nil
}

func validAssetDestination(kind, destination string) bool {
	switch kind {
	case "agent-instructions":
		return destination == "AGENTS.md"
	case "context":
		return strings.HasPrefix(destination, ".agents/context/")
	case "convention":
		return strings.HasPrefix(destination, ".agents/conventions/")
	case "workflow":
		return strings.HasPrefix(destination, ".agents/workflows/")
	default:
		return false
	}
}

func SortedReferences(items []Reference) []Reference {
	result := append([]Reference(nil), items...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].Version < result[j].Version
	})
	return result
}
