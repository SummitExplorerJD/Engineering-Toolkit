package composition

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/profiles"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/templates"
)

// Materialize rebuilds a plan from current state and returns its exact output bytes.
func Materialize(preview Plan, inventory catalog.Catalog, catalogFS fs.FS) ([]MaterializedFile, error) {
	if err := ValidateIdentity(preview); err != nil {
		return nil, err
	}
	if preview.Catalog.ID != inventory.ID || preview.Catalog.Version != inventory.Version || preview.Catalog.SHA256 != inventory.SHA256 {
		return nil, errors.New("composition plan catalog does not match the packaged catalog")
	}
	entry, ok := catalog.Find(inventory.Profiles, preview.Profile.ID, preview.Profile.Version)
	if !ok || entry.SHA256 != preview.Profile.SHA256 {
		return nil, errors.New("composition plan profile does not match the packaged catalog")
	}
	profile, err := profiles.Load(catalogFS, path.Join(entry.Path, "profile.yaml"))
	if err != nil {
		return nil, err
	}
	supplied, err := profileInputs(profile, preview.ResolvedVariables)
	if err != nil {
		return nil, err
	}
	var fresh Plan
	switch preview.Kind {
	case "project.create":
		fresh, err = NewCreate(preview.TargetRoot, preview.Profile.ID, preview.Profile.Version, supplied, preview.Catalog.CLIVersion, inventory, catalogFS)
	case "workspace.compose":
		fresh, err = NewCompose(preview.TargetRoot, preview.Profile.ID, preview.Profile.Version, supplied, preview.Catalog.CLIVersion, inventory, catalogFS)
	default:
		return nil, fmt.Errorf("unsupported composition kind %q", preview.Kind)
	}
	if err != nil {
		return nil, fmt.Errorf("rebuild composition plan: %w", err)
	}
	if fresh.PlanID != preview.PlanID {
		return nil, errors.New("composition state changed after planning; generate and approve a new plan")
	}

	profileResolved, profileValues, err := templates.ResolveVariables(profile.Variables, supplied)
	if err != nil {
		return nil, err
	}
	files := []MaterializedFile{}
	for _, selection := range profile.Skills {
		templateEntry, exists := catalog.Find(inventory.Templates, selection.TemplateID, selection.TemplateVersion)
		if !exists {
			return nil, fmt.Errorf("template %s@%s is not in the packaged catalog", selection.TemplateID, selection.TemplateVersion)
		}
		definition, err := templates.Load(catalogFS, path.Join(templateEntry.Path, "template.yaml"))
		if err != nil {
			return nil, err
		}
		bound, err := bindTemplate(definition, selection, profile.Variables, profileValues)
		if err != nil {
			return nil, err
		}
		rendered, err := templates.Render(catalogFS, templateEntry.Path, definition, templates.Origin{CatalogID: inventory.ID, CatalogVersion: inventory.Version, TemplateSHA256: templateEntry.SHA256}, bound)
		if err != nil {
			return nil, err
		}
		owner := "skill:" + rendered.ID + "@" + rendered.Version
		for _, file := range rendered.Files {
			files = append(files, MaterializedFile{RenderedOutput: RenderedOutput{Path: path.Join(".agents/skills", rendered.ID, file.Path), Owner: owner, Kind: file.Kind, Size: file.Size, SHA256: file.SHA256}, Data: append([]byte(nil), file.Data...)})
		}
	}
	for _, asset := range profile.WorkspaceAssets {
		data, err := fs.ReadFile(catalogFS, path.Join(entry.Path, asset.Source))
		if err != nil {
			return nil, err
		}
		if asset.MediaType == "text" {
			data, err = templates.RenderText(data, profileResolved)
			if err != nil {
				return nil, err
			}
		}
		files = append(files, MaterializedFile{RenderedOutput: RenderedOutput{Path: asset.Destination, Owner: "profile:" + profile.ID + "@" + profile.Version, Kind: asset.Kind, Size: int64(len(data)), SHA256: bytesDigest(data)}, Data: append([]byte(nil), data...)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	if len(files) != len(preview.RenderedOutputs) {
		return nil, errors.New("materialized output count does not match the approved plan")
	}
	for index := range files {
		if files[index].RenderedOutput != preview.RenderedOutputs[index] {
			return nil, fmt.Errorf("materialized output %q does not match the approved plan", files[index].Path)
		}
	}
	return files, nil
}

func profileInputs(profile profiles.Profile, resolved []ResolvedVariable) (map[string]string, error) {
	byName := map[string]ResolvedVariable{}
	for _, value := range resolved {
		if value.Scope == "profile" {
			byName[value.Name] = value
		}
	}
	result := make(map[string]string, len(profile.Variables))
	for _, variable := range profile.Variables {
		value, ok := byName[variable.Name]
		if !ok {
			return nil, fmt.Errorf("composition plan is missing resolved profile variable %q", variable.Name)
		}
		text, err := templates.InputText(variable, value.Value)
		if err != nil {
			return nil, fmt.Errorf("restore profile variable %q: %w", variable.Name, err)
		}
		result[variable.Name] = text
	}
	return result, nil
}
