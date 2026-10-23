package providersync

import (
	"errors"
	"fmt"
	"sort"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
)

// Prepare rebuilds one synchronization plan and binds it to the exact bytes
// and ownership record that a future approved executor may stage.
func Prepare(root, provider, id string) (Prepared, error) {
	if id == "" {
		return Prepared{}, errors.New("preparation requires exactly one canonical skill identifier")
	}
	plan, err := NewPlan(root, provider, id)
	if err != nil {
		return Prepared{}, err
	}
	if len(plan.Sources) != 1 {
		return Prepared{}, errors.New("preparation requires exactly one canonical skill")
	}
	snapshot, err := skills.SnapshotCanonical(root, id)
	if err != nil {
		return Prepared{}, err
	}
	files, _, source, err := renderSnapshot(snapshot, plan.Adapter)
	if err != nil {
		return Prepared{}, err
	}
	if source.PackageSHA256 != plan.Sources[0].PackageSHA256 {
		return Prepared{}, errors.New("canonical skill changed while provider bytes were materialized")
	}
	byPath := make(map[string]Output, len(plan.Outputs))
	for _, output := range plan.Outputs {
		byPath[output.Path] = output
	}
	materialized := make([]MaterializedFile, 0, len(files))
	for _, file := range files {
		output, ok := byPath[file.path]
		if !ok || output.Action == "remove" || output.AfterSHA256 != digestBytes(file.data) {
			return Prepared{}, fmt.Errorf("materialized output %s does not match its plan", file.path)
		}
		materialized = append(materialized, MaterializedFile{Path: file.path, SourcePath: file.sourcePath, SHA256: output.AfterSHA256, Data: append([]byte(nil), file.data...)})
	}
	sort.Slice(materialized, func(i, j int) bool { return materialized[i].Path < materialized[j].Path })
	owned := make([]OwnershipOutput, 0, len(materialized))
	for _, file := range materialized {
		owned = append(owned, OwnershipOutput{Path: file.Path, SHA256: file.SHA256})
	}
	record := OwnershipRecord{
		SchemaVersion: OwnershipSchema, Provider: plan.Provider, AdapterVersion: plan.Adapter.ContractVersion,
		SkillID: source.ID, SkillVersion: source.Version, PackageSHA256: source.PackageSHA256,
		SynchronizationID: plan.PlanID, Outputs: owned,
	}
	if err := validateOwnership(record, plan.Adapter, source.ID); err != nil {
		return Prepared{}, fmt.Errorf("build provider ownership record: %w", err)
	}
	return Prepared{Plan: plan, Files: materialized, Ownership: record}, nil
}
