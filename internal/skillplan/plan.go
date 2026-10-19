// Package skillplan creates immutable, non-executable skill mutation plans.
package skillplan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"golang.org/x/mod/semver"
)

const PlanSchema = "engkit.skill-mutation-plan/v1alpha1"

// Precondition binds a plan to one observed path state or package digest.
type Precondition struct {
	Path           string `json:"path"`
	ExpectedState  string `json:"expected_state"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
}

// Operation describes one proposed filesystem mutation. It is data only.
type Operation struct {
	Action       string `json:"action"`
	Path         string `json:"path"`
	SourcePath   string `json:"source_path,omitempty"`
	BeforeSHA256 string `json:"before_sha256,omitempty"`
	AfterSHA256  string `json:"after_sha256,omitempty"`
	Reason       string `json:"reason"`
}

// Plan is an immutable preview. This package intentionally provides no Apply.
type Plan struct {
	SchemaVersion    string         `json:"schema_version"`
	Kind             string         `json:"kind"`
	PlanID           string         `json:"plan_id"`
	Root             string         `json:"root"`
	SkillID          string         `json:"skill_id"`
	SourcePath       string         `json:"source_path,omitempty"`
	SourceSHA256     string         `json:"source_sha256,omitempty"`
	TargetVersion    string         `json:"target_version,omitempty"`
	ProposedVersion  string         `json:"proposed_version,omitempty"`
	RequiresApproval bool           `json:"requires_approval"`
	Executable       bool           `json:"executable"`
	Preconditions    []Precondition `json:"preconditions"`
	Operations       []Operation    `json:"operations"`
	Verification     []string       `json:"verification"`
	Recovery         []string       `json:"recovery"`
}

type digestBody struct {
	SchemaVersion    string         `json:"schema_version"`
	Kind             string         `json:"kind"`
	Root             string         `json:"root"`
	SkillID          string         `json:"skill_id"`
	SourcePath       string         `json:"source_path,omitempty"`
	SourceSHA256     string         `json:"source_sha256,omitempty"`
	TargetVersion    string         `json:"target_version,omitempty"`
	ProposedVersion  string         `json:"proposed_version,omitempty"`
	RequiresApproval bool           `json:"requires_approval"`
	Executable       bool           `json:"executable"`
	Preconditions    []Precondition `json:"preconditions"`
	Operations       []Operation    `json:"operations"`
	Verification     []string       `json:"verification"`
	Recovery         []string       `json:"recovery"`
}

// NewCreate previews creation from a complete, valid local source package.
func NewCreate(root, source string) (Plan, error) {
	sourceSnapshot, err := skills.SnapshotSource(source)
	if err != nil {
		return Plan{}, err
	}
	id := sourceSnapshot.Package.Manifest.ID
	listed, err := skills.List(root)
	if err != nil {
		return Plan{}, err
	}
	for _, candidate := range listed.Packages {
		if candidate.Directory == id || candidate.ID == id {
			return Plan{}, fmt.Errorf("target skill %q already exists as %s state", id, candidate.Classification)
		}
	}

	packagePath := path.Join(".agents/skills", id)
	targetPath := filepath.Join(filepath.FromSlash(listed.Root), filepath.FromSlash(packagePath))
	if _, err := os.Lstat(targetPath); err == nil {
		return Plan{}, fmt.Errorf("target skill path %q already exists", packagePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Plan{}, fmt.Errorf("inspect target skill path: %w", err)
	}
	preconditions, parentOperations, err := createParentPlan(filepath.FromSlash(listed.Root))
	if err != nil {
		return Plan{}, err
	}
	preconditions = append(preconditions,
		Precondition{Path: packagePath, ExpectedState: "missing"},
		Precondition{Path: sourceSnapshot.PackageRoot, ExpectedState: "canonical_source", ExpectedSHA256: sourceSnapshot.PackageSHA256},
	)
	operations := append([]Operation{}, parentOperations...)
	operations = append(operations, Operation{Action: "create_directory", Path: packagePath, Reason: "create the canonical skill package root"})
	for _, directory := range sourceSnapshot.Directories {
		operations = append(operations, Operation{Action: "create_directory", Path: path.Join(packagePath, directory), Reason: "reproduce the validated source package layout"})
	}
	for _, file := range sourceSnapshot.Files {
		operations = append(operations, Operation{
			Action: "write_file", Path: path.Join(packagePath, file.Path),
			SourcePath: sourceFilePath(sourceSnapshot.PackageRoot, file.Path), AfterSHA256: file.SHA256,
			Reason: "copy an integrity-bound file from the validated source package",
		})
	}
	return finalize(digestBody{
		SchemaVersion: PlanSchema, Kind: "skill.create", Root: listed.Root, SkillID: id,
		SourcePath: sourceSnapshot.PackageRoot, SourceSHA256: sourceSnapshot.PackageSHA256,
		ProposedVersion: sourceSnapshot.Package.Manifest.Version, RequiresApproval: true, Executable: false,
		Preconditions: preconditions, Operations: operations,
		Verification: standardVerification(),
		Recovery: []string{
			"remove only paths created by this plan whose content still matches the planned after_sha256",
			"preserve pre-existing parent directories and every path outside the canonical package root",
		},
	})
}

// NewUpdate previews replacement of one canonical package with a newer source.
func NewUpdate(root, id, source string) (Plan, error) {
	sourceSnapshot, err := skills.SnapshotSource(source)
	if err != nil {
		return Plan{}, err
	}
	if sourceSnapshot.Package.Manifest.ID != id {
		return Plan{}, fmt.Errorf("source skill id %q does not match requested id %q", sourceSnapshot.Package.Manifest.ID, id)
	}
	targetSnapshot, err := skills.SnapshotCanonical(root, id)
	if err != nil {
		return Plan{}, err
	}
	currentVersion := targetSnapshot.Package.Manifest.Version
	proposedVersion := sourceSnapshot.Package.Manifest.Version
	if semver.Compare("v"+proposedVersion, "v"+currentVersion) <= 0 {
		return Plan{}, fmt.Errorf("proposed version %s must be greater than current version %s", proposedVersion, currentVersion)
	}

	packagePath := path.Join(".agents/skills", id)
	operations := updateOperations(packagePath, targetSnapshot, sourceSnapshot)
	return finalize(digestBody{
		SchemaVersion: PlanSchema, Kind: "skill.update", Root: workspaceRoot(targetSnapshot.PackageRoot), SkillID: id,
		SourcePath: sourceSnapshot.PackageRoot, SourceSHA256: sourceSnapshot.PackageSHA256,
		TargetVersion: currentVersion, ProposedVersion: proposedVersion, RequiresApproval: true, Executable: false,
		Preconditions: []Precondition{
			{Path: packagePath, ExpectedState: "canonical", ExpectedSHA256: targetSnapshot.PackageSHA256},
			{Path: sourceSnapshot.PackageRoot, ExpectedState: "canonical_source", ExpectedSHA256: sourceSnapshot.PackageSHA256},
		},
		Operations: operations, Verification: standardVerification(),
		Recovery: []string{
			"capture checkpoint backups for every replaced or removed file before execution",
			"restore only from those backups when the interrupted target still matches recorded journal evidence",
		},
	})
}

// NewRemove previews removal of exactly one canonical package.
func NewRemove(root, id string) (Plan, error) {
	targetSnapshot, err := skills.SnapshotCanonical(root, id)
	if err != nil {
		return Plan{}, err
	}
	packagePath := path.Join(".agents/skills", id)
	operations := make([]Operation, 0, len(targetSnapshot.Files)+len(targetSnapshot.Directories)+1)
	for _, file := range targetSnapshot.Files {
		operations = append(operations, Operation{Action: "remove_file", Path: path.Join(packagePath, file.Path), BeforeSHA256: file.SHA256, Reason: "remove an integrity-bound canonical skill file"})
	}
	for _, directory := range deepestFirst(targetSnapshot.Directories) {
		operations = append(operations, Operation{Action: "remove_directory", Path: path.Join(packagePath, directory), Reason: "remove an empty canonical skill directory"})
	}
	operations = append(operations, Operation{Action: "remove_directory", Path: packagePath, Reason: "remove the empty canonical skill package root"})
	return finalize(digestBody{
		SchemaVersion: PlanSchema, Kind: "skill.remove", Root: workspaceRoot(targetSnapshot.PackageRoot), SkillID: id,
		TargetVersion: targetSnapshot.Package.Manifest.Version, RequiresApproval: true, Executable: false,
		Preconditions: []Precondition{{Path: packagePath, ExpectedState: "canonical", ExpectedSHA256: targetSnapshot.PackageSHA256}},
		Operations:    operations, Verification: standardVerification(),
		Recovery: []string{
			"capture checkpoint backups for every package file before execution",
			"restore the complete package only when its recorded precondition digest is still authoritative",
		},
	})
}

func createParentPlan(root string) ([]Precondition, []Operation, error) {
	preconditions := []Precondition{}
	operations := []Operation{}
	for _, relative := range []string{".agents", ".agents/skills"} {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
		switch {
		case errors.Is(err, os.ErrNotExist):
			preconditions = append(preconditions, Precondition{Path: relative, ExpectedState: "missing"})
			operations = append(operations, Operation{Action: "create_directory", Path: relative, Reason: "prepare the canonical agent asset layout"})
		case err != nil:
			return nil, nil, fmt.Errorf("inspect parent path %s: %w", relative, err)
		case info.Mode()&os.ModeSymlink != 0:
			return nil, nil, fmt.Errorf("parent path %s must not be a symbolic link", relative)
		case !info.IsDir():
			return nil, nil, fmt.Errorf("parent path %s is not a directory", relative)
		default:
			preconditions = append(preconditions, Precondition{Path: relative, ExpectedState: "directory"})
		}
	}
	return preconditions, operations, nil
}

func updateOperations(packagePath string, target, source skills.Snapshot) []Operation {
	targetFiles := fileMap(target.Files)
	sourceFiles := fileMap(source.Files)
	targetDirectories := stringSet(target.Directories)
	sourceDirectories := stringSet(source.Directories)
	removedFiles := make(map[string]struct{})
	removedDirectories := make(map[string]struct{})
	operations := []Operation{}

	for _, file := range target.Files {
		if sourceDirectories[file.Path] || beneathAnyFile(file.Path, sourceFiles) {
			operations = append(operations, removeFileOperation(packagePath, file))
			removedFiles[file.Path] = struct{}{}
		}
	}
	for _, directory := range deepestFirst(target.Directories) {
		if _, blocked := sourceFiles[directory]; blocked || beneathAnyFile(directory, sourceFiles) {
			operations = append(operations, Operation{Action: "remove_directory", Path: path.Join(packagePath, directory), Reason: "remove a directory that blocks a planned file"})
			removedDirectories[directory] = struct{}{}
		}
	}
	for _, directory := range source.Directories {
		_, wasRemoved := removedDirectories[directory]
		if wasRemoved || !targetDirectories[directory] {
			operations = append(operations, Operation{Action: "create_directory", Path: path.Join(packagePath, directory), Reason: "create a directory required by the proposed package"})
		}
	}
	for _, file := range source.Files {
		targetFile, exists := targetFiles[file.Path]
		action := "write_file"
		before := ""
		if exists {
			if targetFile.SHA256 == file.SHA256 {
				continue
			}
			action = "replace_file"
			before = targetFile.SHA256
		}
		operations = append(operations, Operation{
			Action: action, Path: path.Join(packagePath, file.Path), SourcePath: sourceFilePath(source.PackageRoot, file.Path),
			BeforeSHA256: before, AfterSHA256: file.SHA256, Reason: "apply an integrity-bound file from the proposed package",
		})
	}
	for _, file := range target.Files {
		if _, retained := sourceFiles[file.Path]; retained {
			continue
		}
		if _, alreadyRemoved := removedFiles[file.Path]; alreadyRemoved {
			continue
		}
		operations = append(operations, removeFileOperation(packagePath, file))
	}
	for _, directory := range deepestFirst(target.Directories) {
		if sourceDirectories[directory] {
			continue
		}
		if _, alreadyRemoved := removedDirectories[directory]; alreadyRemoved {
			continue
		}
		operations = append(operations, Operation{Action: "remove_directory", Path: path.Join(packagePath, directory), Reason: "remove a directory absent from the proposed package"})
	}
	return operations
}

func removeFileOperation(packagePath string, file skills.FileSnapshot) Operation {
	return Operation{Action: "remove_file", Path: path.Join(packagePath, file.Path), BeforeSHA256: file.SHA256, Reason: "remove a file absent from or blocking the proposed package"}
}

func beneathAnyFile(candidate string, files map[string]skills.FileSnapshot) bool {
	for filename := range files {
		if strings.HasPrefix(candidate, filename+"/") {
			return true
		}
	}
	return false
}

func fileMap(files []skills.FileSnapshot) map[string]skills.FileSnapshot {
	result := make(map[string]skills.FileSnapshot, len(files))
	for _, file := range files {
		result[file.Path] = file
	}
	return result
}

func stringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func deepestFirst(values []string) []string {
	result := append([]string(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		leftDepth := strings.Count(result[i], "/")
		rightDepth := strings.Count(result[j], "/")
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return result[i] < result[j]
	})
	return result
}

func workspaceRoot(packageRoot string) string {
	root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Clean(packageRoot))))
	return filepath.ToSlash(root)
}

func sourceFilePath(packageRoot, relative string) string {
	return filepath.ToSlash(filepath.Join(filepath.FromSlash(packageRoot), filepath.FromSlash(relative)))
}

func standardVerification() []string {
	return []string{
		"rebuild the plan and require the same plan_id immediately before execution",
		"keep every mutation within the canonical package root",
		"validate the resulting package against engkit.skill/v1alpha1",
		"verify every resulting file digest against the approved operation",
		"inspect the final changed-file list",
	}
}

func finalize(body digestBody) (Plan, error) {
	if body.Preconditions == nil {
		body.Preconditions = []Precondition{}
	}
	if body.Operations == nil {
		body.Operations = []Operation{}
	}
	if body.Verification == nil {
		body.Verification = []string{}
	}
	if body.Recovery == nil {
		body.Recovery = []string{}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Plan{}, fmt.Errorf("encode skill plan digest body: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return Plan{
		SchemaVersion: body.SchemaVersion, Kind: body.Kind,
		PlanID: "sha256:" + hex.EncodeToString(digest[:]), Root: body.Root, SkillID: body.SkillID,
		SourcePath: body.SourcePath, SourceSHA256: body.SourceSHA256,
		TargetVersion: body.TargetVersion, ProposedVersion: body.ProposedVersion,
		RequiresApproval: body.RequiresApproval, Executable: body.Executable,
		Preconditions: body.Preconditions, Operations: body.Operations,
		Verification: body.Verification, Recovery: body.Recovery,
	}, nil
}
