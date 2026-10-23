package providersyncexec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/approval"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
)

// Recover completes or rolls back one interrupted provider transaction from
// durable local evidence. It does not render provider content.
func Recover(root, provider, skillID, approvedPlanID string, options Options) (Result, error) {
	options = normalizeOptions(options)
	if err := approval.ValidateID(approvedPlanID, approvedPlanID, true); err != nil {
		return Result{}, err
	}
	workspaceRoot, err := validateWorkspace(root)
	if err != nil {
		return Result{}, err
	}
	key, err := planKey(approvedPlanID)
	if err != nil {
		return Result{}, err
	}
	tx := recoveryTransaction(workspaceRoot, provider, skillID, key, options)
	var plan providersync.Plan
	if err := strictJSON(tx.planPath, &plan); err != nil {
		return Result{}, fmt.Errorf("load provider synchronization plan: %w", err)
	}
	if plan.PlanID != approvedPlanID || plan.Provider != provider || plan.SkillID != skillID || plan.Root != filepath.ToSlash(workspaceRoot) {
		return Result{}, errors.New("persisted provider plan identity does not match recovery request")
	}
	if err := validatePersistedPlan(plan); err != nil {
		return Result{}, err
	}
	if err := strictJSON(tx.checkpointPath, &tx.checkpoint); err != nil {
		return Result{}, fmt.Errorf("load provider checkpoint: %w", err)
	}
	if tx.checkpoint.SchemaVersion != CheckpointSchema || tx.checkpoint.PlanID != plan.PlanID || tx.checkpoint.Provider != provider || tx.checkpoint.SkillID != skillID || tx.checkpoint.Operation != plan.Operation {
		return Result{}, errors.New("provider checkpoint identity does not match persisted plan")
	}
	if err := validateCheckpointLayout(tx, plan); err != nil {
		return Result{}, err
	}
	if err := validateJournal(tx.journalPath, plan.PlanID, tx.checkpoint.JournalSequence); err != nil {
		return Result{}, err
	}
	if err := validateRecoveryLocks(tx, plan); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "provider_sync.recovery_started", map[string]any{"status": tx.checkpoint.Status}); err != nil {
		return Result{}, err
	}
	if err := validateStoredEvidence(tx); err != nil {
		return refuseRecovery(tx, err)
	}
	if finalStateMatches(tx) {
		return finishRecovery(tx, plan, "complete")
	}
	completed, err := completePendingSync(tx)
	if err != nil {
		return refuseRecovery(tx, err)
	}
	if completed {
		return finishRecovery(tx, plan, "complete")
	}
	if err := rollback(tx); err != nil {
		return refuseRecovery(tx, err)
	}
	if err := appendEvent(tx, "provider_sync.recovery_completed", map[string]any{"status": "rolled_back"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "rolled_back"); err != nil {
		return Result{}, err
	}
	if err := releaseLocks(tx); err != nil {
		return Result{}, err
	}
	return tx.result(plan, "rolled_back"), nil
}

func refuseRecovery(tx *transaction, reason error) (Result, error) {
	_ = appendEvent(tx, "provider_sync.recovery_refused", map[string]any{"error": reason.Error()})
	_ = writeCheckpoint(tx, "failed")
	return Result{}, fmt.Errorf("provider recovery refused: %w", reason)
}

func validatePersistedPlan(plan providersync.Plan) error {
	if plan.SchemaVersion != providersync.PlanSchema || !plan.Executable || plan.RequiresApproval == false {
		return errors.New("persisted provider plan is not an executable approved-plan shape")
	}
	copy := plan
	copy.PlanID = ""
	data, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	if digestBytes(data) != plan.PlanID {
		return errors.New("persisted provider plan digest does not match its plan ID")
	}
	return nil
}

func validateCheckpointLayout(tx *transaction, plan providersync.Plan) error {
	if tx.checkpoint.PlanPath != relative(tx.root, tx.planPath) || tx.checkpoint.JournalPath != relative(tx.root, tx.journalPath) || tx.checkpoint.RecoveryPath != relative(tx.root, tx.base) || tx.checkpoint.OwnershipPath != relative(tx.root, tx.ownership) || tx.checkpoint.OwnershipStaging != relative(tx.root, tx.stagedOwnership) || tx.checkpoint.OwnershipBackup != relative(tx.root, tx.backupOwnership) {
		return errors.New("provider checkpoint paths do not match the transaction identity")
	}
	if tx.checkpoint.NextOutput < 0 || tx.checkpoint.NextOutput > len(plan.Outputs) || len(tx.checkpoint.Outputs) != len(plan.Outputs) {
		return errors.New("provider checkpoint output progress is invalid")
	}
	for index, output := range plan.Outputs {
		state := tx.checkpoint.Outputs[index]
		if state.Path != output.Path || state.Action != output.Action || state.BeforeSHA256 != output.BeforeSHA256 || state.AfterSHA256 != output.AfterSHA256 {
			return errors.New("provider checkpoint outputs do not match the persisted plan")
		}
		expectedStaging := ""
		if output.Action != "remove" {
			expectedStaging = relative(tx.root, filepath.Join(tx.base, "staged", "outputs", filepath.FromSlash(output.Path)))
		}
		expectedBackup := ""
		if output.Action == "replace" || output.Action == "remove" {
			expectedBackup = relative(tx.root, filepath.Join(tx.base, "backup", "outputs", filepath.FromSlash(output.Path)))
		}
		if state.StagingPath != expectedStaging || state.BackupPath != expectedBackup {
			return errors.New("provider checkpoint staging or backup paths are invalid")
		}
	}
	for _, directory := range tx.checkpoint.CreatedDirectories {
		candidate, err := targetPath(tx.root, directory)
		if err != nil || candidate == tx.root {
			return errors.New("provider checkpoint contains an invalid created directory")
		}
		ownedAncestor := false
		for _, output := range plan.Outputs {
			target, targetErr := targetPath(tx.root, output.Path)
			if targetErr != nil {
				return targetErr
			}
			rel, relErr := filepath.Rel(candidate, target)
			if relErr == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				ownedAncestor = true
				break
			}
		}
		if !ownedAncestor {
			return errors.New("provider checkpoint created directory is outside planned output ancestry")
		}
	}
	return nil
}

func validateStoredEvidence(tx *transaction) error {
	for _, state := range tx.checkpoint.Outputs {
		if state.StagingPath != "" {
			exists, digest, err := regularState(filepath.Join(tx.root, filepath.FromSlash(state.StagingPath)))
			if err != nil {
				return err
			}
			if exists && digest != state.AfterSHA256 {
				return fmt.Errorf("staged provider output %s changed", state.Path)
			}
		}
		if state.BackupPath != "" {
			exists, digest, err := regularState(filepath.Join(tx.root, filepath.FromSlash(state.BackupPath)))
			if err != nil {
				return err
			}
			if exists && digest != state.BeforeSHA256 {
				return fmt.Errorf("provider output backup %s changed", state.Path)
			}
		}
	}
	if exists, digest, err := regularState(tx.stagedOwnership); err != nil {
		return err
	} else if exists && (tx.checkpoint.OwnershipAfterHash == "" || digest != tx.checkpoint.OwnershipAfterHash) {
		return errors.New("staged provider ownership changed")
	}
	if exists, digest, err := regularState(tx.backupOwnership); err != nil {
		return err
	} else if exists && (tx.checkpoint.OwnershipBeforeHash == "" || digest != tx.checkpoint.OwnershipBeforeHash) {
		return errors.New("provider ownership backup changed")
	}
	return nil
}

func finishRecovery(tx *transaction, plan providersync.Plan, status string) (Result, error) {
	if err := appendEvent(tx, "provider_sync.recovery_completed", map[string]any{"status": status}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, status); err != nil {
		return Result{}, err
	}
	if err := releaseLocks(tx); err != nil {
		return Result{}, err
	}
	return tx.result(plan, status), nil
}

func completePendingSync(tx *transaction) (bool, error) {
	if tx.checkpoint.Operation != "synchronize" || verifyOutputs(tx, true) != nil {
		return false, nil
	}
	staged, stagedDigest, err := regularState(tx.stagedOwnership)
	if err != nil {
		return false, err
	}
	if !staged {
		return false, nil
	}
	if stagedDigest != tx.checkpoint.OwnershipAfterHash {
		return false, errors.New("staged provider ownership changed before recovery")
	}
	live, liveDigest, err := regularState(tx.ownership)
	if err != nil {
		return false, err
	}
	backup, _, err := regularState(tx.backupOwnership)
	if err != nil {
		return false, err
	}
	if backup {
		return false, errors.New("unexpected provider ownership backup blocks completion")
	}
	if tx.checkpoint.OwnershipBeforeHash == "" {
		if live {
			return false, errors.New("unexpected live provider ownership blocks completion")
		}
	} else if !live || liveDigest != tx.checkpoint.OwnershipBeforeHash {
		return false, errors.New("prior provider ownership changed before completion")
	}
	if err := publishOwnership(tx); err != nil {
		return false, err
	}
	if err := verifyFinal(tx); err != nil {
		return false, err
	}
	return true, nil
}

func recoveryTransaction(root, provider, skillID, key string, options Options) *transaction {
	engkit := filepath.Join(root, ".engkit")
	base := filepath.Join(engkit, "recovery", "provider-sync", provider, skillID, key)
	return &transaction{
		root: root, engkit: engkit, key: key, base: base,
		skillLock:       filepath.Join(engkit, "recovery", "skills", "locks", skillID+".lock"),
		providerLock:    filepath.Join(engkit, "recovery", "provider-sync", "locks", provider, skillID+".lock"),
		planPath:        filepath.Join(engkit, "plans", "provider-sync", provider, skillID, key+".json"),
		journalPath:     filepath.Join(engkit, "journals", "provider-sync", provider, skillID, key+".jsonl"),
		checkpointPath:  filepath.Join(engkit, "checkpoints", "provider-sync", provider, skillID, key+".json"),
		ownership:       filepath.Join(engkit, "providers", provider, skillID+".json"),
		stagedOwnership: filepath.Join(base, "staged", "ownership.json"), backupOwnership: filepath.Join(base, "backup", "ownership.json"),
		now: options.Now, failAt: options.FailAt, link: options.Link,
	}
}

func validateRecoveryLocks(tx *transaction, plan providersync.Plan) error {
	for _, path := range []string{tx.skillLock, tx.providerLock} {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("provider recovery lock %s is missing or invalid", path)
		}
		var record struct {
			SchemaVersion string `json:"schema_version"`
			PlanID        string `json:"plan_id"`
			Provider      string `json:"provider"`
			SkillID       string `json:"skill_id"`
			ProcessID     int    `json:"process_id"`
			StartedAt     string `json:"started_at"`
			Command       string `json:"command"`
		}
		if err := strictJSON(filepath.Join(path, "lock.json"), &record); err != nil || record.SchemaVersion != LockSchema || record.PlanID != plan.PlanID || record.Provider != plan.Provider || record.SkillID != plan.SkillID {
			return fmt.Errorf("provider recovery lock %s does not match persisted plan", path)
		}
	}
	return nil
}

func finalStateMatches(tx *transaction) bool {
	if verifyOutputs(tx, true) != nil {
		return false
	}
	if tx.checkpoint.Operation == "synchronize" {
		if requireDigest(tx.ownership, tx.checkpoint.OwnershipAfterHash) != nil {
			return false
		}
		if tx.checkpoint.OwnershipBeforeHash != "" && requireDigest(tx.backupOwnership, tx.checkpoint.OwnershipBeforeHash) != nil {
			return false
		}
		for _, state := range tx.checkpoint.Outputs {
			if state.Action == "replace" && requireDigest(filepath.Join(tx.root, filepath.FromSlash(state.BackupPath)), state.BeforeSHA256) != nil {
				return false
			}
		}
		return true
	}
	if _, err := os.Lstat(tx.ownership); !errors.Is(err, os.ErrNotExist) {
		return false
	}
	if requireDigest(tx.backupOwnership, tx.checkpoint.OwnershipBeforeHash) != nil {
		return false
	}
	for _, state := range tx.checkpoint.Outputs {
		if requireDigest(filepath.Join(tx.root, filepath.FromSlash(state.BackupPath)), state.BeforeSHA256) != nil {
			return false
		}
	}
	return true
}

func rollback(tx *transaction) error {
	if tx.checkpoint.Operation == "synchronize" {
		if err := rollbackSyncOwnership(tx); err != nil {
			return err
		}
	}
	for index := len(tx.checkpoint.Outputs) - 1; index >= 0; index-- {
		if err := rollbackOutput(tx, tx.checkpoint.Outputs[index]); err != nil {
			return err
		}
	}
	if tx.checkpoint.Operation == "remove" {
		if err := restoreOwnership(tx); err != nil {
			return err
		}
	}
	for index := len(tx.checkpoint.CreatedDirectories) - 1; index >= 0; index-- {
		_ = os.Remove(filepath.Join(tx.root, filepath.FromSlash(tx.checkpoint.CreatedDirectories[index])))
	}
	return verifyBeforeState(tx)
}

func rollbackSyncOwnership(tx *transaction) error {
	exists, digest, err := regularState(tx.ownership)
	if err != nil {
		return err
	}
	if exists {
		if digest == tx.checkpoint.OwnershipBeforeHash {
			backup, _, backupErr := regularState(tx.backupOwnership)
			if backupErr != nil {
				return backupErr
			}
			if !backup {
				return nil
			}
			return errors.New("unexpected ownership backup prevents rollback")
		}
		if digest != tx.checkpoint.OwnershipAfterHash {
			return errors.New("live provider ownership is neither approved before nor after state")
		}
		if err := os.Remove(tx.ownership); err != nil {
			return err
		}
	}
	if tx.checkpoint.OwnershipBeforeHash != "" {
		return restoreOwnership(tx)
	}
	if backup, _, err := regularState(tx.backupOwnership); err != nil || backup {
		return errors.New("unexpected ownership backup prevents rollback")
	}
	return nil
}

func restoreOwnership(tx *transaction) error {
	live, liveDigest, err := regularState(tx.ownership)
	if err != nil {
		return err
	}
	backup, backupDigest, err := regularState(tx.backupOwnership)
	if err != nil {
		return err
	}
	if live {
		if liveDigest == tx.checkpoint.OwnershipBeforeHash && !backup {
			return nil
		}
		return errors.New("live provider ownership blocks exact restoration")
	}
	if !backup || backupDigest != tx.checkpoint.OwnershipBeforeHash {
		return errors.New("exact provider ownership backup is unavailable")
	}
	return os.Rename(tx.backupOwnership, tx.ownership)
}

func rollbackOutput(tx *transaction, state outputState) error {
	target, err := targetPath(tx.root, state.Path)
	if err != nil {
		return err
	}
	live, liveDigest, err := regularState(target)
	if err != nil {
		return err
	}
	var backup bool
	var backupDigest string
	var backupPath string
	if state.BackupPath != "" {
		backupPath = filepath.Join(tx.root, filepath.FromSlash(state.BackupPath))
		backup, backupDigest, err = regularState(backupPath)
		if err != nil {
			return err
		}
	}
	switch state.Action {
	case "unchanged":
		if !live || liveDigest != state.AfterSHA256 {
			return fmt.Errorf("unchanged output %s no longer matches", state.Path)
		}
		return nil
	case "create":
		if !live {
			return nil
		}
		if liveDigest != state.AfterSHA256 {
			return fmt.Errorf("created output %s changed before rollback", state.Path)
		}
		return os.Remove(target)
	case "replace", "remove":
		if live && liveDigest == state.BeforeSHA256 && !backup {
			return nil
		}
		if !backup || backupDigest != state.BeforeSHA256 {
			return fmt.Errorf("exact backup for %s is unavailable", state.Path)
		}
		if live {
			if state.Action != "replace" || liveDigest != state.AfterSHA256 {
				return fmt.Errorf("live output %s blocks rollback", state.Path)
			}
			if err := os.Remove(target); err != nil {
				return err
			}
		}
		return os.Rename(backupPath, target)
	default:
		return fmt.Errorf("unsupported rollback action %q", state.Action)
	}
}

func verifyBeforeState(tx *transaction) error {
	for _, state := range tx.checkpoint.Outputs {
		target, err := targetPath(tx.root, state.Path)
		if err != nil {
			return err
		}
		if state.Action == "create" {
			if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("created output %s remains after rollback", state.Path)
			}
			continue
		}
		expected := state.BeforeSHA256
		if state.Action == "unchanged" {
			expected = state.AfterSHA256
		}
		if err := requireDigest(target, expected); err != nil {
			return err
		}
	}
	if tx.checkpoint.OwnershipBeforeHash == "" {
		if _, err := os.Lstat(tx.ownership); !errors.Is(err, os.ErrNotExist) {
			return errors.New("provider ownership remains after rollback")
		}
		return nil
	}
	return requireDigest(tx.ownership, tx.checkpoint.OwnershipBeforeHash)
}

func regularState(path string) (bool, string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, "", fmt.Errorf("recovery path %s is not a regular file", path)
	}
	digest, err := digestFile(path)
	return true, digest, err
}
