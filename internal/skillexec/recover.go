package skillexec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/approval"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillplan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
)

// Recover completes or rolls back one interrupted approved transaction.
func Recover(root, skillID, approvedPlanID string, options Options) (Result, error) {
	options = normalizeOptions(options)
	if strings.TrimSpace(approvedPlanID) == "" {
		return Result{}, errors.New("approval is required; pass the complete plan ID with --approve")
	}
	key, err := planKey(approvedPlanID)
	if err != nil {
		return Result{}, err
	}
	absolute, err := validateWorkspace(root)
	if err != nil {
		return Result{}, err
	}
	engkit := filepath.Join(absolute, ".engkit")
	planPath := filepath.Join(engkit, "plans", "skills", key+".json")
	preview, err := readPlan(planPath)
	if err != nil {
		return Result{}, fmt.Errorf("load recovery plan: %w", err)
	}
	if err := approval.ValidateID(approvedPlanID, preview.PlanID, true); err != nil {
		return Result{}, err
	}
	if preview.SkillID != skillID {
		return Result{}, errors.New("approved recovery plan belongs to another skill")
	}
	checkpointPath := filepath.Join(engkit, "checkpoints", "skills", key+".json")
	saved, err := readCheckpoint(checkpointPath)
	if err != nil {
		return Result{}, fmt.Errorf("load recovery checkpoint: %w", err)
	}
	if saved.PlanID != preview.PlanID || saved.SkillID != skillID || saved.Kind != preview.Kind {
		return Result{}, errors.New("recovery checkpoint does not match the approved plan")
	}
	if saved.BeforeSHA256 != beforeDigest(preview) || saved.AfterSHA256 != preview.SourceSHA256 {
		return Result{}, errors.New("recovery checkpoint digests do not match the approved plan")
	}
	if saved.Status == "complete" || saved.Status == "rolled_back" {
		return Result{}, fmt.Errorf("skill transaction is already %s", saved.Status)
	}
	journalPath := filepath.Join(engkit, "journals", "skills", key+".jsonl")
	sequence, err := validateJournal(journalPath, preview.PlanID)
	if err != nil {
		return Result{}, err
	}
	if saved.JournalSequence > sequence {
		return Result{}, errors.New("checkpoint refers to a missing journal event")
	}

	base := filepath.Join(engkit, "recovery", "skills", key)
	tx := &transaction{
		root: absolute, engkit: engkit, key: key,
		target:   filepath.Join(absolute, ".agents", "skills", skillID),
		staging:  filepath.Join(base, "staged", skillID),
		backup:   filepath.Join(base, "backup", skillID),
		lock:     filepath.Join(engkit, "recovery", "skills", "locks", skillID+".lock"),
		planPath: planPath, journalPath: journalPath, checkpointPath: checkpointPath,
		checkpoint: saved, now: options.Now, failAt: options.FailAt,
	}
	tx.checkpoint.JournalSequence = sequence
	if err := validateCheckpointPaths(tx); err != nil {
		return Result{}, err
	}
	if err := acquireRecoveryLock(tx, preview, options.ProcessID); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "skill.recovery_started", map[string]any{"previous_status": saved.Status}); err != nil {
		return Result{}, err
	}

	status, err := recoverState(tx, preview)
	if err != nil {
		_ = appendEvent(tx, "skill.recovery_refused", map[string]any{"error": err.Error()})
		_ = writeCheckpoint(tx, "failed")
		return Result{}, err
	}
	if err := tx.fail("before_recovery_complete"); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "skill.recovery_completed", map[string]any{"status": status}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, status); err != nil {
		return Result{}, err
	}
	if err := releaseLock(tx.lock); err != nil {
		return Result{}, err
	}
	return tx.result(preview, status), nil
}

func acquireRecoveryLock(tx *transaction, preview skillplan.Plan, processID int) error {
	info, err := os.Lstat(tx.lock)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.Mkdir(tx.lock, 0o700); err != nil {
			return fmt.Errorf("acquire recovery lock: %w", err)
		}
		if err := writeNewJSON(filepath.Join(tx.lock, "lock.json"), map[string]any{
			"schema_version": "engkit.skill-lock/v1alpha1",
			"plan_id":        preview.PlanID, "skill_id": preview.SkillID,
			"process_id": processID, "started_at": tx.now().UTC().Format(timeFormat),
			"command": "skill.recover",
		}); err != nil {
			_ = os.Remove(tx.lock)
			return err
		}
		return nil
	case err != nil:
		return err
	case info.Mode()&os.ModeSymlink != 0 || !info.IsDir():
		return errors.New("skill recovery lock is not a real directory")
	}
	contents, err := os.ReadFile(filepath.Join(tx.lock, "lock.json"))
	if err != nil {
		return fmt.Errorf("read existing skill lock: %w", err)
	}
	var record struct {
		SchemaVersion string `json:"schema_version"`
		PlanID        string `json:"plan_id"`
		SkillID       string `json:"skill_id"`
		ProcessID     int    `json:"process_id"`
		StartedAt     string `json:"started_at"`
		Command       string `json:"command"`
	}
	if err := decodeStrict(contents, &record); err != nil {
		return fmt.Errorf("decode existing skill lock: %w", err)
	}
	if record.SchemaVersion != "engkit.skill-lock/v1alpha1" ||
		record.PlanID != preview.PlanID || record.SkillID != preview.SkillID {
		return errors.New("existing skill lock belongs to another transaction")
	}
	return nil
}

func validateCheckpointPaths(tx *transaction) error {
	expected := map[string]string{
		"target":  relative(tx.root, tx.target),
		"staging": relative(tx.root, tx.staging),
		"backup":  relative(tx.root, tx.backup),
		"plan":    relative(tx.root, tx.planPath),
		"journal": relative(tx.root, tx.journalPath),
	}
	actual := map[string]string{
		"target": tx.checkpoint.TargetPath, "staging": tx.checkpoint.StagingPath,
		"backup": tx.checkpoint.BackupPath, "plan": tx.checkpoint.PlanPath,
		"journal": tx.checkpoint.JournalPath,
	}
	for name, value := range expected {
		if actual[name] != value {
			return fmt.Errorf("checkpoint %s path does not match transaction layout", name)
		}
	}
	return nil
}

func recoverState(tx *transaction, preview skillplan.Plan) (string, error) {
	target := observePackage(tx.target)
	staging := observePackage(tx.staging)
	backup := observePackage(tx.backup)
	before := beforeDigest(preview)
	after := preview.SourceSHA256

	switch preview.Kind {
	case "skill.create":
		if target.valid && target.digest == after && !staging.exists && !backup.exists {
			return "complete", nil
		}
		if !target.exists && !backup.exists &&
			(!staging.exists || staging.valid && staging.digest == after) {
			if staging.exists {
				if err := removeKnownPackage(tx.staging, after); err != nil {
					return "", err
				}
			}
			return "rolled_back", nil
		}
	case "skill.update":
		if target.valid && target.digest == after && backup.valid && backup.digest == before &&
			!staging.exists {
			return "complete", nil
		}
		if !target.exists && backup.valid && backup.digest == before &&
			(!staging.exists || staging.valid && staging.digest == after) {
			if staging.exists {
				if err := removeKnownPackage(tx.staging, after); err != nil {
					return "", err
				}
			}
			if err := os.Rename(tx.backup, tx.target); err != nil {
				return "", fmt.Errorf("restore skill backup: %w", err)
			}
			restored := observePackage(tx.target)
			if !restored.valid || restored.digest != before {
				return "", errors.New("restored skill does not match approved before digest")
			}
			return "rolled_back", nil
		}
		if target.valid && target.digest == before &&
			(!backup.exists || backup.valid && backup.digest == before) &&
			(!staging.exists || staging.valid && staging.digest == after) {
			if staging.exists {
				if err := removeKnownPackage(tx.staging, after); err != nil {
					return "", err
				}
			}
			return "rolled_back", nil
		}
	case "skill.remove":
		if !target.exists && backup.valid && backup.digest == before && !staging.exists {
			return "complete", nil
		}
		if target.valid && target.digest == before &&
			(!backup.exists || backup.valid && backup.digest == before) && !staging.exists {
			return "rolled_back", nil
		}
	default:
		return "", fmt.Errorf("unsupported recovery kind %q", preview.Kind)
	}
	return "", errors.New("recovery state is ambiguous; all paths were preserved")
}

type observedPackage struct {
	exists bool
	valid  bool
	digest string
}

func observePackage(path string) observedPackage {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return observedPackage{}
	} else if err != nil {
		return observedPackage{exists: true}
	}
	snapshot, err := skills.SnapshotSource(path)
	if err != nil {
		return observedPackage{exists: true}
	}
	return observedPackage{exists: true, valid: true, digest: snapshot.PackageSHA256}
}

func removeKnownPackage(path, expectedDigest string) error {
	snapshot, err := skills.SnapshotSource(path)
	if err != nil {
		return fmt.Errorf("validate known staging package: %w", err)
	}
	if snapshot.PackageSHA256 != expectedDigest {
		return errors.New("staging package no longer matches the approved digest")
	}
	for _, file := range snapshot.Files {
		if err := os.Remove(filepath.Join(path, filepath.FromSlash(file.Path))); err != nil {
			return fmt.Errorf("remove known staging file: %w", err)
		}
	}
	for index := len(snapshot.Directories) - 1; index >= 0; index-- {
		if err := os.Remove(filepath.Join(path, filepath.FromSlash(snapshot.Directories[index]))); err != nil {
			return fmt.Errorf("remove known staging directory: %w", err)
		}
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove known staging root: %w", err)
	}
	return nil
}
