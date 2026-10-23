package providersyncexec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/approval"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
)

// Execute applies one freshly rebuilt, exactly approved provider plan.
func Execute(preview providersync.Plan, approvedPlanID string, options Options) (result Result, returnedErr error) {
	options = normalizeOptions(options)
	if err := approval.ValidateID(approvedPlanID, preview.PlanID, preview.RequiresApproval); err != nil {
		return Result{}, err
	}
	if preview.SchemaVersion != providersync.PlanSchema || !preview.Executable {
		return Result{}, errors.New("provider synchronization plan is not executable")
	}
	skillID, err := operationSkill(preview)
	if err != nil {
		return Result{}, err
	}
	root, err := validateWorkspace(preview.Root)
	if err != nil {
		return Result{}, err
	}
	if filepath.ToSlash(root) != preview.Root {
		return Result{}, errors.New("plan root is not the normalized workspace root")
	}
	fresh, prepared, err := rebuild(preview)
	if err != nil {
		return Result{}, err
	}
	if fresh.PlanID != preview.PlanID {
		return Result{}, errors.New("provider state changed after planning; generate and approve a new plan")
	}
	key, err := planKey(preview.PlanID)
	if err != nil {
		return Result{}, err
	}
	if err := prepareState(root, preview.Provider, skillID); err != nil {
		return Result{}, err
	}
	tx, err := newTransaction(root, key, preview, options)
	if err != nil {
		return Result{}, err
	}
	started := false
	defer func() {
		if returnedErr != nil && started {
			_ = appendEvent(tx, "provider_sync.execution_failed", map[string]any{"error": returnedErr.Error()})
			_ = writeCheckpoint(tx, "failed")
		}
	}()
	if err := tx.start(preview, options.ProcessID); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "approval.accepted", map[string]any{"approved_plan_id": approvedPlanID}); err != nil {
		tx.abortBeforePrepared()
		return Result{}, err
	}
	if err := appendEvent(tx, "provider_sync.execution_started", map[string]any{"operation": preview.Operation, "provider": preview.Provider, "skill_id": skillID}); err != nil {
		tx.abortBeforePrepared()
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "prepared"); err != nil {
		tx.abortBeforePrepared()
		return Result{}, err
	}
	started = true
	if preview.Operation == "synchronize" {
		if err := stagePrepared(tx, prepared); err != nil {
			return Result{}, err
		}
		if err := appendEvent(tx, "provider_sync.staged", map[string]any{"output_count": len(prepared.Files)}); err != nil {
			return Result{}, err
		}
		if err := writeCheckpoint(tx, "staged"); err != nil {
			return Result{}, err
		}
	}
	if err := probeNoReplace(tx.base, tx.link); err != nil {
		return Result{}, err
	}
	if err := tx.fail("after_staging"); err != nil {
		return Result{}, err
	}
	fresh, _, err = rebuild(preview)
	if err != nil || fresh.PlanID != preview.PlanID {
		return Result{}, errors.New("provider state changed before publication; recovery is required")
	}
	if err := publishOutputs(tx); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "provider_sync.outputs_verified", map[string]any{"output_count": len(tx.checkpoint.Outputs)}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "outputs_published"); err != nil {
		return Result{}, err
	}
	if err := tx.fail("after_outputs_publish"); err != nil {
		return Result{}, err
	}
	if err := publishOwnership(tx); err != nil {
		return Result{}, err
	}
	if err := verifyFinal(tx); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "provider_sync.verification_succeeded", map[string]any{"status": "verified"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "verified"); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "provider_sync.execution_completed", map[string]any{"status": "complete"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "complete"); err != nil {
		return Result{}, err
	}
	if err := releaseLocks(tx); err != nil {
		return Result{}, err
	}
	started = false
	return tx.result(preview, "complete"), nil
}

func rebuild(preview providersync.Plan) (providersync.Plan, providersync.Prepared, error) {
	switch preview.Operation {
	case "synchronize":
		prepared, err := providersync.Prepare(preview.Root, preview.Provider, preview.SkillID)
		return prepared.Plan, prepared, err
	case "remove":
		plan, err := providersync.NewUnsyncPlan(preview.Root, preview.Provider, preview.SkillID)
		return plan, providersync.Prepared{}, err
	default:
		return providersync.Plan{}, providersync.Prepared{}, fmt.Errorf("unsupported provider operation %q", preview.Operation)
	}
}

func newTransaction(root, key string, preview providersync.Plan, options Options) (*transaction, error) {
	engkit := filepath.Join(root, ".engkit")
	base := filepath.Join(engkit, "recovery", "provider-sync", preview.Provider, preview.SkillID, key)
	if _, err := os.Lstat(base); err == nil {
		return nil, errors.New("provider transaction state already exists; run skills recover-sync")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	planPath := filepath.Join(engkit, "plans", "provider-sync", preview.Provider, preview.SkillID, key+".json")
	journalPath := filepath.Join(engkit, "journals", "provider-sync", preview.Provider, preview.SkillID, key+".jsonl")
	checkpointPath := filepath.Join(engkit, "checkpoints", "provider-sync", preview.Provider, preview.SkillID, key+".json")
	for _, artifact := range []string{planPath, journalPath, checkpointPath} {
		if _, err := os.Lstat(artifact); err == nil {
			return nil, errors.New("provider transaction artifacts already exist; run skills recover-sync")
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	skillLock := filepath.Join(engkit, "recovery", "skills", "locks", preview.SkillID+".lock")
	if err := ensureDirectory(filepath.Dir(skillLock)); err != nil {
		return nil, err
	}
	if err := os.Mkdir(skillLock, 0o700); err != nil {
		return nil, errors.New("canonical skill is locked; recover the active skill or provider transaction")
	}
	providerLock := filepath.Join(engkit, "recovery", "provider-sync", "locks", preview.Provider, preview.SkillID+".lock")
	if err := os.Mkdir(providerLock, 0o700); err != nil {
		_ = os.Remove(skillLock)
		return nil, errors.New("provider skill is locked; run skills recover-sync for the interrupted plan")
	}
	ownership := filepath.Join(engkit, "providers", preview.Provider, preview.SkillID+".json")
	tx := &transaction{
		root: root, engkit: engkit, key: key, base: base, skillLock: skillLock, providerLock: providerLock,
		planPath: planPath, journalPath: journalPath, checkpointPath: checkpointPath,
		ownership: ownership, stagedOwnership: filepath.Join(base, "staged", "ownership.json"), backupOwnership: filepath.Join(base, "backup", "ownership.json"),
		now: options.Now, failAt: options.FailAt, link: options.Link,
	}
	tx.checkpoint = checkpoint{
		SchemaVersion: CheckpointSchema, PlanID: preview.PlanID, Operation: preview.Operation, Provider: preview.Provider, SkillID: preview.SkillID,
		PlanPath: relative(root, planPath), JournalPath: relative(root, journalPath), RecoveryPath: relative(root, base),
		OwnershipPath: relative(root, ownership), OwnershipStaging: relative(root, tx.stagedOwnership), OwnershipBackup: relative(root, tx.backupOwnership),
		Outputs: []outputState{}, CreatedDirectories: []string{},
	}
	for _, output := range preview.Outputs {
		state := outputState{Path: output.Path, Action: output.Action, BeforeSHA256: output.BeforeSHA256, AfterSHA256: output.AfterSHA256}
		if output.Action != "remove" {
			state.StagingPath = relative(root, filepath.Join(base, "staged", "outputs", filepath.FromSlash(output.Path)))
		}
		if output.Action == "replace" || output.Action == "remove" {
			state.BackupPath = relative(root, filepath.Join(base, "backup", "outputs", filepath.FromSlash(output.Path)))
		}
		tx.checkpoint.Outputs = append(tx.checkpoint.Outputs, state)
	}
	for _, observation := range preview.Observations {
		if observation.Path == relative(root, ownership) {
			tx.checkpoint.OwnershipBeforeHash = observation.SHA256
		}
	}
	return tx, nil
}

func (tx *transaction) start(preview providersync.Plan, processID int) (returnedErr error) {
	defer func() {
		if returnedErr != nil {
			tx.abortBeforePrepared()
		}
	}()
	for _, directory := range []string{tx.base, filepath.Join(tx.base, "staged"), filepath.Join(tx.base, "staged", "outputs"), filepath.Join(tx.base, "backup"), filepath.Join(tx.base, "backup", "outputs")} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return err
		}
	}
	lockRecord := map[string]any{"schema_version": LockSchema, "plan_id": preview.PlanID, "provider": preview.Provider, "skill_id": preview.SkillID, "process_id": processID, "started_at": tx.now().UTC().Format(timeFormat), "command": preview.Operation}
	if err := writeNewJSON(filepath.Join(tx.skillLock, "lock.json"), lockRecord); err != nil {
		return err
	}
	if err := writeNewJSON(filepath.Join(tx.providerLock, "lock.json"), lockRecord); err != nil {
		return err
	}
	return writeNewJSON(tx.planPath, preview)
}

func (tx *transaction) abortBeforePrepared() {
	_ = os.Remove(tx.checkpointPath)
	_ = os.Remove(tx.journalPath)
	_ = os.Remove(tx.planPath)
	_ = os.RemoveAll(tx.base)
	_ = releaseLocks(tx)
}

func stagePrepared(tx *transaction, prepared providersync.Prepared) error {
	byPath := make(map[string]providersync.MaterializedFile, len(prepared.Files))
	for _, file := range prepared.Files {
		byPath[file.Path] = file
	}
	for index := range tx.checkpoint.Outputs {
		state := &tx.checkpoint.Outputs[index]
		if state.Action == "remove" {
			continue
		}
		file, ok := byPath[state.Path]
		if !ok || file.SHA256 != state.AfterSHA256 {
			return fmt.Errorf("missing materialized provider output %s", state.Path)
		}
		staged := filepath.Join(tx.root, filepath.FromSlash(state.StagingPath))
		if err := os.MkdirAll(filepath.Dir(staged), 0o700); err != nil {
			return err
		}
		if err := writeNewSynced(staged, file.Data); err != nil {
			return err
		}
	}
	if err := writeNewJSON(tx.stagedOwnership, prepared.Ownership); err != nil {
		return err
	}
	digest, err := digestFile(tx.stagedOwnership)
	if err != nil {
		return err
	}
	tx.checkpoint.OwnershipAfterHash = digest
	return nil
}

func publishOutputs(tx *transaction) error {
	for index := range tx.checkpoint.Outputs {
		state := &tx.checkpoint.Outputs[index]
		if err := appendEvent(tx, "provider_sync.output_started", map[string]any{"path": state.Path, "action": state.Action}); err != nil {
			return err
		}
		target, err := targetPath(tx.root, state.Path)
		if err != nil {
			return err
		}
		if err := publishOutput(tx, state, target); err != nil {
			return err
		}
		state.Published = state.Action != "unchanged"
		tx.checkpoint.NextOutput = index + 1
		if err := appendEvent(tx, "provider_sync.output_published", map[string]any{"path": state.Path, "action": state.Action}); err != nil {
			return err
		}
		if err := writeCheckpoint(tx, "publishing"); err != nil {
			return err
		}
		if err := tx.fail(fmt.Sprintf("after_output_%d", index)); err != nil {
			return err
		}
	}
	return verifyOutputs(tx, true)
}

func publishOutput(tx *transaction, state *outputState, target string) error {
	switch state.Action {
	case "unchanged":
		return requireDigest(target, state.AfterSHA256)
	case "create":
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			return errors.New("provider create target appeared before publication")
		}
		if err := ensureTargetParents(tx.root, target, &tx.checkpoint.CreatedDirectories); err != nil {
			return err
		}
		return tx.link(filepath.Join(tx.root, filepath.FromSlash(state.StagingPath)), target)
	case "replace", "remove":
		if err := requireDigest(target, state.BeforeSHA256); err != nil {
			return err
		}
		backup := filepath.Join(tx.root, filepath.FromSlash(state.BackupPath))
		if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
			return err
		}
		if _, err := os.Lstat(backup); !errors.Is(err, os.ErrNotExist) {
			return errors.New("provider output backup already exists")
		}
		if err := os.Rename(target, backup); err != nil {
			return err
		}
		if state.Action == "replace" {
			if err := tx.link(filepath.Join(tx.root, filepath.FromSlash(state.StagingPath)), target); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported provider output action %q", state.Action)
	}
}

func publishOwnership(tx *transaction) error {
	if tx.checkpoint.Operation == "synchronize" {
		if tx.checkpoint.OwnershipBeforeHash != "" {
			if err := requireDigest(tx.ownership, tx.checkpoint.OwnershipBeforeHash); err != nil {
				return err
			}
			if err := os.Rename(tx.ownership, tx.backupOwnership); err != nil {
				return err
			}
		} else if _, err := os.Lstat(tx.ownership); !errors.Is(err, os.ErrNotExist) {
			return errors.New("provider ownership target appeared before publication")
		}
		if err := tx.link(tx.stagedOwnership, tx.ownership); err != nil {
			return err
		}
	} else {
		if err := requireDigest(tx.ownership, tx.checkpoint.OwnershipBeforeHash); err != nil {
			return err
		}
		if err := os.Rename(tx.ownership, tx.backupOwnership); err != nil {
			return err
		}
	}
	tx.checkpoint.OwnershipPublished = true
	if err := appendEvent(tx, "provider_sync.ownership_published", map[string]any{"ownership": relative(tx.root, tx.ownership)}); err != nil {
		return err
	}
	if err := writeCheckpoint(tx, "ownership_published"); err != nil {
		return err
	}
	return tx.fail("after_ownership_publish")
}

func verifyOutputs(tx *transaction, final bool) error {
	for _, state := range tx.checkpoint.Outputs {
		target, err := targetPath(tx.root, state.Path)
		if err != nil {
			return err
		}
		if state.Action == "remove" {
			if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("removed provider output %s still exists", state.Path)
			}
			continue
		}
		if final || state.Action != "replace" {
			if err := requireDigest(target, state.AfterSHA256); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyFinal(tx *transaction) error {
	if err := verifyOutputs(tx, true); err != nil {
		return err
	}
	if tx.checkpoint.Operation == "synchronize" {
		return requireDigest(tx.ownership, tx.checkpoint.OwnershipAfterHash)
	}
	if _, err := os.Lstat(tx.ownership); !errors.Is(err, os.ErrNotExist) {
		return errors.New("provider ownership record still exists after unsync")
	}
	return requireDigest(tx.backupOwnership, tx.checkpoint.OwnershipBeforeHash)
}

func requireDigest(path, expected string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("provider path %s is not a regular file", path)
	}
	digest, err := digestFile(path)
	if err != nil {
		return err
	}
	if digest != expected {
		return fmt.Errorf("provider path %s digest changed", path)
	}
	return nil
}

func releaseLock(path string) error {
	if err := os.Remove(filepath.Join(path, "lock.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func releaseLocks(tx *transaction) error {
	if err := releaseLock(tx.providerLock); err != nil {
		return err
	}
	return releaseLock(tx.skillLock)
}

func (tx *transaction) result(plan providersync.Plan, status string) Result {
	changed := make([]string, 0, len(tx.checkpoint.Outputs)+1)
	for _, output := range tx.checkpoint.Outputs {
		if output.Action != "unchanged" {
			changed = append(changed, output.Path)
		}
	}
	changed = append(changed, relative(tx.root, tx.ownership))
	sort.Strings(changed)
	return Result{SchemaVersion: ResultSchema, Root: filepath.ToSlash(tx.root), PlanID: plan.PlanID, Operation: plan.Operation, Provider: plan.Provider, SkillID: plan.SkillID, Status: status, ChangedPaths: changed, PlanPath: relative(tx.root, tx.planPath), JournalPath: relative(tx.root, tx.journalPath), CheckpointPath: relative(tx.root, tx.checkpointPath), RecoveryPath: relative(tx.root, tx.base)}
}
