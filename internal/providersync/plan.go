package providersync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"golang.org/x/mod/semver"
)

var sha256Pattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// NewPlan builds a deterministic provider synchronization preview. It reads
// canonical and destination state but never creates, replaces, or removes a
// workspace path.
func NewPlan(root, provider, id string) (Plan, error) {
	workspaceRoot, err := resolveRoot(root)
	if err != nil {
		return Plan{}, err
	}
	adapter, err := adapterContract(provider)
	if err != nil {
		return Plan{}, err
	}
	snapshots, err := selectedSnapshots(workspaceRoot, strings.TrimSpace(id))
	if err != nil {
		return Plan{}, err
	}

	plan := Plan{
		SchemaVersion: PlanSchema, Operation: "synchronize", Root: filepath.ToSlash(workspaceRoot), Provider: adapter.Provider, Adapter: adapter,
		Sources: []SourceReference{}, Observations: []Observation{}, Mappings: []Mapping{}, Outputs: []Output{}, Conflicts: []Finding{}, AffectedPaths: []string{},
		Risks:             []string{"provider discovery contracts can change after the recorded evidence date", "Codex output shares a directory with canonical and external skills", "provider runtimes retain authority over resource execution and permissions"},
		RequiredApprovals: []string{"provider-synchronization", "provider:" + adapter.Provider},
		Verification:      []string{"rebuild the plan and require the same plan_id before execution", "verify every generated file digest", "verify canonical and unowned provider files remain unchanged", "verify no Spec Kit path enters Engineering Toolkit ownership"},
		RequiresApproval:  true,
	}
	observed := map[string]Observation{}
	affected := map[string]struct{}{}

	for _, snapshot := range snapshots {
		files, mapping, source, err := renderSnapshot(snapshot, adapter)
		if err != nil {
			return Plan{}, err
		}
		fresh, err := skills.SnapshotCanonical(workspaceRoot, source.ID)
		if err != nil {
			return Plan{}, fmt.Errorf("revalidate canonical skill %s: %w", source.ID, err)
		}
		if fresh.PackageSHA256 != snapshot.PackageSHA256 {
			return Plan{}, fmt.Errorf("canonical skill %s changed while the plan was built", source.ID)
		}
		plan.Sources = append(plan.Sources, source)
		plan.Mappings = append(plan.Mappings, mapping)
		for _, finding := range mapping.Findings {
			if finding.Severity == "blocking" {
				plan.Conflicts = append(plan.Conflicts, finding)
			}
		}

		recordPath := ownershipPath(adapter.Provider, source.ID)
		record, recordObservation, recordFindings, err := readOwnership(workspaceRoot, recordPath, adapter, source.ID)
		if err != nil {
			return Plan{}, err
		}
		observed[recordObservation.Path] = recordObservation
		plan.Conflicts = append(plan.Conflicts, recordFindings...)
		affected[recordPath] = struct{}{}
		owned := ownershipMap(record)
		planned := map[string]string{}

		for _, file := range files {
			if previous, exists := planned[strings.ToLower(file.path)]; exists {
				plan.Conflicts = append(plan.Conflicts, Finding{Severity: "blocking", Code: "target.path_collision", Path: file.path, Message: "generated output collides with " + previous})
				continue
			}
			planned[strings.ToLower(file.path)] = file.path
			observation, finding, err := inspectOutput(workspaceRoot, file.path)
			if err != nil {
				return Plan{}, err
			}
			observation.Ownership = "unowned"
			prior, isOwned := owned[file.path]
			if isOwned {
				observation.Ownership = "engkit"
			}
			observed[file.path] = observation
			action, conflict := chooseAction(observation, prior, isOwned, digestBytes(file.data))
			if finding != nil {
				plan.Conflicts = append(plan.Conflicts, *finding)
				action = "conflict"
			}
			if conflict != nil {
				plan.Conflicts = append(plan.Conflicts, *conflict)
			}
			plan.Outputs = append(plan.Outputs, Output{Path: file.path, Action: action, Owner: owner(adapter.Provider, source.ID), SourcePath: file.sourcePath, Size: int64(len(file.data)), BeforeSHA256: observation.SHA256, AfterSHA256: digestBytes(file.data)})
			affected[file.path] = struct{}{}
		}

		if record != nil {
			for _, prior := range record.Outputs {
				if _, exists := planned[strings.ToLower(prior.Path)]; exists {
					continue
				}
				observation, finding, err := inspectOutput(workspaceRoot, prior.Path)
				if err != nil {
					return Plan{}, err
				}
				observation.Ownership = "engkit"
				observed[prior.Path] = observation
				action := "remove"
				if finding != nil {
					plan.Conflicts = append(plan.Conflicts, *finding)
					action = "conflict"
				} else if observation.State == "missing" {
					plan.Conflicts = append(plan.Conflicts, Finding{Severity: "blocking", Code: "target.owned_missing", Path: prior.Path, Message: "recorded provider output is missing"})
					action = "conflict"
				} else if observation.SHA256 != prior.SHA256 {
					plan.Conflicts = append(plan.Conflicts, Finding{Severity: "blocking", Code: "target.owned_changed", Path: prior.Path, Message: "recorded provider output changed outside Engineering Toolkit"})
					action = "conflict"
				}
				plan.Outputs = append(plan.Outputs, Output{Path: prior.Path, Action: action, Owner: owner(adapter.Provider, source.ID), BeforeSHA256: observation.SHA256})
				affected[prior.Path] = struct{}{}
			}
		}
	}

	for _, item := range observed {
		plan.Observations = append(plan.Observations, item)
	}
	for item := range affected {
		plan.AffectedPaths = append(plan.AffectedPaths, item)
	}
	sortPlan(&plan)
	plan.Executable = len(plan.Sources) > 0 && len(plan.Conflicts) == 0
	for _, mapping := range plan.Mappings {
		if mapping.Quality != "exact" {
			plan.Executable = false
		}
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return Plan{}, fmt.Errorf("encode provider synchronization plan: %w", err)
	}
	digest := sha256.Sum256(encoded)
	plan.PlanID = "sha256:" + hex.EncodeToString(digest[:])
	return plan, nil
}

func selectedSnapshots(root, id string) ([]skills.Snapshot, error) {
	if id != "" {
		snapshot, err := skills.SnapshotCanonical(root, id)
		if err != nil {
			return nil, err
		}
		return []skills.Snapshot{snapshot}, nil
	}
	listed, err := skills.List(root)
	if err != nil {
		return nil, err
	}
	result := []skills.Snapshot{}
	for _, candidate := range listed.Packages {
		if candidate.Classification != skills.ClassificationCanonical {
			continue
		}
		snapshot, err := skills.SnapshotCanonical(root, candidate.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	if len(result) == 0 {
		return nil, errors.New("no canonical skills selected")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Package.ID < result[j].Package.ID })
	return result, nil
}

func resolveRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("workspace root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect workspace root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("workspace root must be a real directory")
	}
	return absolute, nil
}

func readOwnership(root, relative string, adapter AdapterReference, id string) (*OwnershipRecord, Observation, []Finding, error) {
	observation, finding, err := inspectOutput(root, relative)
	if err != nil {
		return nil, Observation{}, nil, err
	}
	observation.Ownership = "engkit-state"
	if finding != nil {
		return nil, observation, []Finding{{Severity: "blocking", Code: "ownership.record_invalid", Path: relative, Message: finding.Message}}, nil
	}
	if observation.State == "missing" {
		return nil, observation, nil, nil
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return nil, observation, nil, fmt.Errorf("read provider ownership record: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var record OwnershipRecord
	if err := decoder.Decode(&record); err != nil {
		return nil, observation, []Finding{{Severity: "blocking", Code: "ownership.record_invalid", Path: relative, Message: "provider ownership record is invalid JSON"}}, nil
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, observation, []Finding{{Severity: "blocking", Code: "ownership.record_invalid", Path: relative, Message: "provider ownership record contains trailing JSON"}}, nil
	}
	if err := validateOwnership(record, adapter, id); err != nil {
		return nil, observation, []Finding{{Severity: "blocking", Code: "ownership.record_invalid", Path: relative, Message: err.Error()}}, nil
	}
	return &record, observation, nil, nil
}

func validateOwnership(record OwnershipRecord, adapter AdapterReference, id string) error {
	if record.SchemaVersion != OwnershipSchema || record.Provider != adapter.Provider || record.AdapterVersion != adapter.ContractVersion || record.SkillID != id {
		return errors.New("provider ownership record identity does not match the selected adapter and skill")
	}
	if record.SkillVersion == "" || strings.HasPrefix(record.SkillVersion, "v") || !semver.IsValid("v"+record.SkillVersion) || !sha256Pattern.MatchString(record.PackageSHA256) || !sha256Pattern.MatchString(record.SynchronizationID) {
		return errors.New("provider ownership record contains invalid source or synchronization identity")
	}
	seen := map[string]struct{}{}
	prefix := adapter.TargetRoot + "/" + id + "/"
	for _, output := range record.Outputs {
		if !strings.HasPrefix(output.Path, prefix) || !portableRelative(output.Path) || !sha256Pattern.MatchString(output.SHA256) {
			return errors.New("provider ownership record contains an invalid output")
		}
		if adapter.Provider == "codex" && output.Path != prefix+"SKILL.md" {
			return errors.New("Codex ownership can contain only the generated SKILL.md sidecar")
		}
		key := strings.ToLower(output.Path)
		if _, exists := seen[key]; exists {
			return errors.New("provider ownership record contains duplicate outputs")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func inspectOutput(root, relative string) (Observation, *Finding, error) {
	result := Observation{Path: relative, State: "missing", Ownership: "unowned"}
	if !portableRelative(relative) {
		return result, &Finding{Severity: "blocking", Code: "target.path_type", Path: relative, Message: "provider output path is not portable and relative"}, nil
	}
	parts := strings.Split(relative, "/")
	for index := 1; index < len(parts); index++ {
		parent := filepath.Join(root, filepath.FromSlash(strings.Join(parts[:index], "/")))
		info, err := os.Lstat(parent)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return result, nil, fmt.Errorf("inspect provider output ancestor %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return result, &Finding{Severity: "blocking", Code: "target.path_collision", Path: relative, Message: "provider output has a symbolic-link, junction, or non-directory ancestor"}, nil
		}
	}
	filename := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil, nil
	}
	if err != nil {
		return result, nil, fmt.Errorf("inspect provider output %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		result.State = "symlink"
		return result, &Finding{Severity: "blocking", Code: "target.path_type", Path: relative, Message: "provider output is a symbolic link or junction"}, nil
	}
	if !info.Mode().IsRegular() {
		if info.IsDir() {
			result.State = "directory"
		} else {
			result.State = "other"
		}
		return result, &Finding{Severity: "blocking", Code: "target.path_type", Path: relative, Message: "provider output is not a regular file"}, nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return result, nil, fmt.Errorf("read provider output %s: %w", relative, err)
	}
	result.State = "file"
	result.SHA256 = digestBytes(data)
	return result, nil, nil
}

func chooseAction(observation Observation, prior OwnershipOutput, owned bool, after string) (string, *Finding) {
	if !owned {
		if observation.State == "missing" {
			return "create", nil
		}
		return "conflict", &Finding{Severity: "blocking", Code: "target.unowned_existing", Path: observation.Path, Message: "provider output already exists without Engineering Toolkit ownership"}
	}
	if observation.State == "missing" {
		return "conflict", &Finding{Severity: "blocking", Code: "target.owned_missing", Path: observation.Path, Message: "recorded provider output is missing"}
	}
	if observation.SHA256 != prior.SHA256 {
		return "conflict", &Finding{Severity: "blocking", Code: "target.owned_changed", Path: observation.Path, Message: "recorded provider output changed outside Engineering Toolkit"}
	}
	if observation.SHA256 == after {
		return "unchanged", nil
	}
	return "replace", nil
}

func ownershipMap(record *OwnershipRecord) map[string]OwnershipOutput {
	result := map[string]OwnershipOutput{}
	if record == nil {
		return result
	}
	for _, output := range record.Outputs {
		result[output.Path] = output
	}
	return result
}

func ownershipPath(provider, id string) string {
	return ".engkit/providers/" + provider + "/" + id + ".json"
}

func owner(provider, id string) string {
	return "provider:" + provider + "/skill:" + id
}

func portableRelative(value string) bool {
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return false
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned != value || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") || strings.ContainsAny(component, `<>:"|?*`) || windowsReserved(component) {
			return false
		}
		for _, character := range component {
			if character < 32 {
				return false
			}
		}
	}
	return true
}

func windowsReserved(value string) bool {
	base := strings.ToLower(strings.SplitN(value, ".", 2)[0])
	switch base {
	case "con", "prn", "aux", "nul", "clock$":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "com") || strings.HasPrefix(base, "lpt")) && base[3] >= 49 && base[3] <= 57
}

func sortPlan(plan *Plan) {
	sort.Slice(plan.Sources, func(i, j int) bool { return plan.Sources[i].ID < plan.Sources[j].ID })
	sort.Slice(plan.Mappings, func(i, j int) bool { return plan.Mappings[i].SkillID < plan.Mappings[j].SkillID })
	sort.Slice(plan.Observations, func(i, j int) bool { return plan.Observations[i].Path < plan.Observations[j].Path })
	sort.Slice(plan.Outputs, func(i, j int) bool { return plan.Outputs[i].Path < plan.Outputs[j].Path })
	sort.Slice(plan.Conflicts, func(i, j int) bool {
		if plan.Conflicts[i].Code != plan.Conflicts[j].Code {
			return plan.Conflicts[i].Code < plan.Conflicts[j].Code
		}
		return plan.Conflicts[i].Path < plan.Conflicts[j].Path
	})
	sort.Strings(plan.AffectedPaths)
}
