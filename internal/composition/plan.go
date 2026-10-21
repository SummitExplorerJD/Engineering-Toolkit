package composition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/profiles"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/templates"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

type plannedFile struct {
	RenderedOutput
	data []byte
}

type digestBody struct {
	SchemaVersion     string             `json:"schema_version"`
	Kind              string             `json:"kind"`
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

func NewCreate(destination, profileID, profileVersion string, supplied map[string]string, cliVersion string, inventory catalog.Catalog, catalogFS fs.FS) (Plan, error) {
	return build("project.create", destination, profileID, profileVersion, supplied, cliVersion, inventory, catalogFS)
}

func NewCompose(root, profileID, profileVersion string, supplied map[string]string, cliVersion string, inventory catalog.Catalog, catalogFS fs.FS) (Plan, error) {
	return build("workspace.compose", root, profileID, profileVersion, supplied, cliVersion, inventory, catalogFS)
}

func Inspect(root string, inventory catalog.Catalog, catalogFS fs.FS) (Inspection, error) {
	snapshot, err := workspace.Inspect(root)
	if err != nil {
		return Inspection{}, err
	}
	recommendations, err := profiles.Recommend(root, inventory, catalogFS)
	if err != nil {
		return Inspection{}, err
	}
	return Inspection{SchemaVersion: InspectionSchema, Workspace: snapshot, Recommendations: recommendations}, nil
}

func build(kind, target, profileID, profileVersion string, supplied map[string]string, cliVersion string, inventory catalog.Catalog, catalogFS fs.FS) (Plan, error) {
	mode := "new-project"
	if kind == "workspace.compose" {
		mode = "existing-workspace"
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve target root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	if err := rejectLinkedAncestors(absolute); err != nil {
		return Plan{}, err
	}

	entry, ok := catalog.Find(inventory.Profiles, profileID, profileVersion)
	if !ok {
		return Plan{}, fmt.Errorf("profile %s@%s is not in the packaged catalog", profileID, profileVersion)
	}
	profile, err := profiles.Load(catalogFS, path.Join(entry.Path, "profile.yaml"))
	if err != nil {
		return Plan{}, err
	}
	if profile.ID != entry.ID || profile.Version != entry.Version {
		return Plan{}, fmt.Errorf("profile identity does not match its catalog entry")
	}
	if !stringContains(profile.Compatibility.Modes, mode) {
		return Plan{}, fmt.Errorf("profile %s@%s does not support %s mode", profileID, profileVersion, mode)
	}

	profileResolved, profileValues, err := templates.ResolveVariables(profile.Variables, supplied)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve profile variables: %w", err)
	}
	resolved := make([]ResolvedVariable, 0, len(profileResolved))
	for _, value := range profileResolved {
		resolved = append(resolved, ResolvedVariable{Scope: "profile", Name: value.Name, Type: value.Type, Value: value.Value})
	}

	outputs := []plannedFile{}
	templateRefs := []ContentReference{}
	for _, selection := range profile.Skills {
		templateEntry, exists := catalog.Find(inventory.Templates, selection.TemplateID, selection.TemplateVersion)
		if !exists {
			return Plan{}, fmt.Errorf("template %s@%s selected by profile is not in the packaged catalog", selection.TemplateID, selection.TemplateVersion)
		}
		definition, err := templates.Load(catalogFS, path.Join(templateEntry.Path, "template.yaml"))
		if err != nil {
			return Plan{}, err
		}
		if definition.ID != templateEntry.ID || definition.Version != templateEntry.Version {
			return Plan{}, fmt.Errorf("template identity does not match its catalog entry")
		}
		bound, err := bindTemplate(definition, selection, profile.Variables, profileValues)
		if err != nil {
			return Plan{}, fmt.Errorf("bind template %s@%s: %w", selection.TemplateID, selection.TemplateVersion, err)
		}
		rendered, err := templates.Render(catalogFS, templateEntry.Path, definition, templates.Origin{CatalogID: inventory.ID, CatalogVersion: inventory.Version, TemplateSHA256: templateEntry.SHA256}, bound)
		if err != nil {
			return Plan{}, fmt.Errorf("render template %s@%s: %w", selection.TemplateID, selection.TemplateVersion, err)
		}
		scope := "template:" + selection.TemplateID + "@" + selection.TemplateVersion
		for _, value := range rendered.Variables {
			resolved = append(resolved, ResolvedVariable{Scope: scope, Name: value.Name, Type: value.Type, Value: value.Value})
		}
		owner := "skill:" + rendered.ID + "@" + rendered.Version
		for _, file := range rendered.Files {
			outputPath := path.Join(".agents/skills", rendered.ID, file.Path)
			outputs = append(outputs, plannedFile{RenderedOutput: RenderedOutput{Path: outputPath, Owner: owner, Kind: file.Kind, Size: file.Size, SHA256: file.SHA256}, data: file.Data})
		}
		templateRefs = append(templateRefs, ContentReference{ID: templateEntry.ID, Version: templateEntry.Version, SHA256: templateEntry.SHA256})
	}

	for _, asset := range profile.WorkspaceAssets {
		data, err := fs.ReadFile(catalogFS, path.Join(entry.Path, asset.Source))
		if err != nil {
			return Plan{}, fmt.Errorf("read profile asset %q: %w", asset.Source, err)
		}
		if asset.MediaType == "text" {
			data, err = templates.RenderText(data, profileResolved)
			if err != nil {
				return Plan{}, fmt.Errorf("render profile asset %q: %w", asset.Source, err)
			}
		}
		outputs = append(outputs, plannedFile{RenderedOutput: RenderedOutput{Path: asset.Destination, Owner: "profile:" + profile.ID + "@" + profile.Version, Kind: asset.Kind, Size: int64(len(data)), SHA256: bytesDigest(data)}, data: data})
	}

	sort.Slice(templateRefs, func(i, j int) bool {
		if templateRefs[i].ID != templateRefs[j].ID {
			return templateRefs[i].ID < templateRefs[j].ID
		}
		return templateRefs[i].Version < templateRefs[j].Version
	})
	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].Scope != resolved[j].Scope {
			return resolved[i].Scope < resolved[j].Scope
		}
		return resolved[i].Name < resolved[j].Name
	})
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	if err := validateOutputCollisions(outputs); err != nil {
		return Plan{}, err
	}

	facts, conflicts, preconditions, operations, err := observeAndPlan(kind, absolute, profile, outputs)
	if err != nil {
		return Plan{}, err
	}
	for _, request := range profile.IntegrationRequests {
		conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "integration.unsupported", Message: fmt.Sprintf("integration %s/%s is outside the composition planner", request.Provider, request.Capability)})
	}
	sortConflicts(conflicts)
	renderedOutputs := make([]RenderedOutput, len(outputs))
	for index := range outputs {
		renderedOutputs[index] = outputs[index].RenderedOutput
	}
	body := digestBody{
		SchemaVersion: PlanSchema, Kind: kind, TargetRoot: filepath.ToSlash(absolute),
		Catalog: CatalogReference{ID: inventory.ID, Version: inventory.Version, SHA256: inventory.SHA256, CLIVersion: cliVersion},
		Profile: ContentReference{ID: entry.ID, Version: entry.Version, SHA256: entry.SHA256}, Templates: templateRefs,
		ResolvedVariables: resolved, WorkspaceFacts: facts, RenderedOutputs: renderedOutputs,
		Conflicts: conflicts, Preconditions: preconditions, Operations: operations,
		RequiresApproval: true, Executable: false,
		Verification: []string{"rebuild the plan and require the same plan_id before execution", "validate every canonical skill and rendered file digest", "preserve every unowned workspace path", "inspect the final changed-file list"},
		Recovery:     []string{"future execution must use same-filesystem staging, an append-only JSONL journal, and an atomic JSON checkpoint", "future recovery must fail closed when observed ownership or content is ambiguous"},
	}
	return finalize(body)
}

func bindTemplate(definition templates.Template, selection profiles.SkillSelection, profileVariables []templates.Variable, profileValues map[string]any) (map[string]string, error) {
	declared := make(map[string]templates.Variable, len(definition.Variables))
	for _, variable := range definition.Variables {
		declared[variable.Name] = variable
	}
	for name := range selection.Bindings {
		if _, exists := declared[name]; !exists {
			return nil, fmt.Errorf("binding %q is not declared by the template", name)
		}
	}
	profileDeclared := make(map[string]templates.Variable, len(profileVariables))
	for _, variable := range profileVariables {
		profileDeclared[variable.Name] = variable
	}
	bound := make(map[string]string, len(selection.Bindings))
	for name, binding := range selection.Bindings {
		target := declared[name]
		if binding.Variable != "" {
			value, exists := profileValues[binding.Variable]
			if !exists {
				return nil, fmt.Errorf("profile variable %q has no resolved value", binding.Variable)
			}
			source := profileDeclared[binding.Variable]
			if source.Type != target.Type {
				return nil, fmt.Errorf("profile variable %q type %q does not match template variable %q type %q", binding.Variable, source.Type, name, target.Type)
			}
			text, err := templates.InputText(target, value)
			if err != nil {
				return nil, fmt.Errorf("binding %q: %w", name, err)
			}
			bound[name] = text
		} else {
			text, err := templates.InputText(target, binding.Literal)
			if err != nil {
				return nil, fmt.Errorf("binding %q: %w", name, err)
			}
			bound[name] = text
		}
	}
	return bound, nil
}

func observeAndPlan(kind, root string, profile profiles.Profile, outputs []plannedFile) (WorkspaceFacts, []Conflict, []Precondition, []Operation, error) {
	state, observed, treeDigest, ecosystems, err := inspectTarget(kind, root)
	if err != nil {
		return WorkspaceFacts{}, nil, nil, nil, err
	}
	facts := WorkspaceFacts{Mode: map[string]string{"project.create": "new-project", "workspace.compose": "existing-workspace"}[kind], TargetState: state, TreeSHA256: treeDigest, Ecosystems: ecosystems, SourceLayout: profile.SourceLayout.Policy, Observed: observed}
	conflicts := []Conflict{}
	if kind == "project.create" && state == "nonempty_directory" {
		conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "target.nonempty", Message: "project creation destination must be absent or empty"})
	}
	if kind == "workspace.compose" && state != "directory" {
		return WorkspaceFacts{}, nil, nil, nil, fmt.Errorf("workspace composition target must be an existing directory")
	}
	if len(profile.Compatibility.Ecosystems) != 0 && kind == "workspace.compose" && len(stringIntersection(profile.Compatibility.Ecosystems, ecosystems)) == 0 {
		conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "ecosystem.incompatible", Message: "selected profile does not match observed ecosystem evidence"})
	}
	if profile.SourceLayout.Policy == "src" && kind == "workspace.compose" && !stringContains(ecosystems, "go") {
		conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "source_layout.incompatible", Path: "src", Message: "src layout requires compatible observed ecosystem evidence"})
	}
	if profile.SourceLayout.Policy == "src" && kind == "project.create" && !stringContains(profile.Compatibility.Ecosystems, "go") {
		conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "source_layout.incompatible", Path: "src", Message: "src layout requires a supported ecosystem declared by the selected profile"})
	}
	preconditions := []Precondition{{Path: ".", ExpectedState: state, SHA256: treeDigest}}
	directories := directorySet(profile, outputs)
	operations := []Operation{}
	for _, directory := range directories {
		full := filepath.Join(root, filepath.FromSlash(directory))
		info, err := os.Lstat(full)
		switch {
		case errors.Is(err, os.ErrNotExist):
			preconditions = append(preconditions, Precondition{Path: directory, ExpectedState: "missing"})
			operations = append(operations, Operation{Action: "create_directory", Path: directory, Owner: "profile:" + profile.ID + "@" + profile.Version, Reason: "prepare the selected profile layout"})
		case err != nil:
			return WorkspaceFacts{}, nil, nil, nil, fmt.Errorf("inspect planned directory %q: %w", directory, err)
		case info.Mode()&os.ModeSymlink != 0 || !info.IsDir():
			conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "path.directory_collision", Path: directory, Message: "planned directory is blocked by a non-directory or link"})
		default:
			preconditions = append(preconditions, Precondition{Path: directory, ExpectedState: "directory"})
		}
	}
	for _, output := range outputs {
		full := filepath.Join(root, filepath.FromSlash(output.Path))
		info, err := os.Lstat(full)
		switch {
		case errors.Is(err, os.ErrNotExist):
			preconditions = append(preconditions, Precondition{Path: output.Path, ExpectedState: "missing"})
			operations = append(operations, Operation{Action: "write_file", Path: output.Path, Owner: output.Owner, AfterSHA256: output.SHA256, Reason: "publish an integrity-bound rendered output"})
		case err != nil:
			return WorkspaceFacts{}, nil, nil, nil, fmt.Errorf("inspect planned output %q: %w", output.Path, err)
		case info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular():
			conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "path.file_collision", Path: output.Path, Message: "planned file is blocked by a non-regular file or link"})
		default:
			digest, err := diskFileDigest(full)
			if err != nil {
				return WorkspaceFacts{}, nil, nil, nil, err
			}
			preconditions = append(preconditions, Precondition{Path: output.Path, ExpectedState: "regular_file", SHA256: digest})
			if digest == output.SHA256 {
				conflicts = append(conflicts, Conflict{Severity: "informational", Code: "path.already_matches", Path: output.Path, Message: "existing file already matches the rendered output"})
			} else {
				conflicts = append(conflicts, Conflict{Severity: "blocking", Code: "path.unowned_collision", Path: output.Path, Message: "existing file is not owned by this composition plan"})
			}
		}
	}
	sort.Slice(preconditions, func(i, j int) bool { return preconditions[i].Path < preconditions[j].Path })
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Action != operations[j].Action {
			return operations[i].Action < operations[j].Action
		}
		return operations[i].Path < operations[j].Path
	})
	return facts, conflicts, preconditions, operations, nil
}

func inspectTarget(kind, root string) (string, []ObservedPath, string, []string, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		if kind == "workspace.compose" {
			return "missing", []ObservedPath{}, "", []string{}, nil
		}
		return "missing", []ObservedPath{}, "", []string{}, nil
	}
	if err != nil {
		return "", nil, "", nil, fmt.Errorf("inspect target root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", nil, "", nil, fmt.Errorf("target root must be a directory and not a symbolic link")
	}
	observed := []ObservedPath{}
	ecosystems := []string{}
	err = filepath.WalkDir(root, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filename == root {
			return nil
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("target contains symbolic link %q", relative)
		}
		if entry.IsDir() {
			observed = append(observed, ObservedPath{Path: relative, State: "directory", Ownership: "unowned"})
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("target contains special file %q", relative)
		}
		digest, err := diskFileDigest(filename)
		if err != nil {
			return err
		}
		observed = append(observed, ObservedPath{Path: relative, State: "regular_file", Ownership: "unowned", SHA256: digest})
		if relative == "go.mod" {
			ecosystems = append(ecosystems, "go")
		}
		return nil
	})
	if err != nil {
		return "", nil, "", nil, fmt.Errorf("inventory target: %w", err)
	}
	sort.Slice(observed, func(i, j int) bool { return observed[i].Path < observed[j].Path })
	state := "directory"
	if kind == "project.create" {
		if len(observed) == 0 {
			state = "empty_directory"
		} else {
			state = "nonempty_directory"
		}
	}
	return state, observed, observedDigest(observed), ecosystems, nil
}

func rejectLinkedAncestors(target string) error {
	for candidate := filepath.Clean(target); ; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("target root must not use symbolic-link or junction ancestors")
			}
			if !info.IsDir() {
				return fmt.Errorf("target ancestor %q is not a directory", candidate)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect target ancestor %q: %w", candidate, err)
		}
		parent := filepath.Dir(candidate)
		if strings.EqualFold(parent, candidate) {
			return nil
		}
	}
}

func directorySet(profile profiles.Profile, outputs []plannedFile) []string {
	set := map[string]struct{}{}
	for _, directory := range profile.WorkspaceDirectories {
		addDirectoryParents(set, directory)
	}
	if profile.SourceLayout.Policy == "src" {
		addDirectoryParents(set, "src")
	}
	for _, output := range outputs {
		parent := path.Dir(output.Path)
		if parent != "." {
			addDirectoryParents(set, parent)
		}
	}
	result := make([]string, 0, len(set))
	for directory := range set {
		result = append(result, directory)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := strings.Count(result[i], "/"), strings.Count(result[j], "/")
		if left != right {
			return left < right
		}
		return result[i] < result[j]
	})
	return result
}

func addDirectoryParents(set map[string]struct{}, directory string) {
	parts := strings.Split(directory, "/")
	for index := range parts {
		set[strings.Join(parts[:index+1], "/")] = struct{}{}
	}
}

func validateOutputCollisions(outputs []plannedFile) error {
	seen := map[string]string{}
	paths := make([]string, 0, len(outputs))
	for _, output := range outputs {
		key := strings.ToLower(output.Path)
		if previous, exists := seen[key]; exists {
			return fmt.Errorf("rendered outputs %q and %q collide", previous, output.Path)
		}
		seen[key] = output.Path
		paths = append(paths, key)
	}
	for _, outputPath := range paths {
		for parent := path.Dir(outputPath); parent != "."; parent = path.Dir(parent) {
			if previous, exists := seen[parent]; exists {
				return fmt.Errorf("rendered outputs %q and %q have a file/directory collision", previous, seen[outputPath])
			}
		}
	}
	return nil
}

func finalize(body digestBody) (Plan, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return Plan{}, fmt.Errorf("encode composition plan digest body: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return Plan{SchemaVersion: body.SchemaVersion, Kind: body.Kind, PlanID: "sha256:" + hex.EncodeToString(digest[:]), TargetRoot: body.TargetRoot, Catalog: body.Catalog, Profile: body.Profile, Templates: body.Templates, ResolvedVariables: body.ResolvedVariables, WorkspaceFacts: body.WorkspaceFacts, RenderedOutputs: body.RenderedOutputs, Conflicts: body.Conflicts, Preconditions: body.Preconditions, Operations: body.Operations, RequiresApproval: body.RequiresApproval, Executable: body.Executable, Verification: body.Verification, Recovery: body.Recovery}, nil
}

func bytesDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func diskFileDigest(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filename, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func observedDigest(items []ObservedPath) string {
	hash := sha256.New()
	for _, item := range items {
		_, _ = hash.Write([]byte(item.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(item.State))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(item.SHA256))
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func sortConflicts(values []Conflict) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Severity != values[j].Severity {
			return values[i].Severity < values[j].Severity
		}
		if values[i].Path != values[j].Path {
			return values[i].Path < values[j].Path
		}
		return values[i].Code < values[j].Code
	})
}
func stringContains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func stringIntersection(left, right []string) []string {
	result := []string{}
	for _, value := range left {
		if stringContains(right, value) {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
