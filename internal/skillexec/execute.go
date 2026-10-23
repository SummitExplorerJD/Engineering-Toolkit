package skillexec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/approval"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillplan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
)

// Execute applies one freshly rebuilt and exactly approved skill mutation.
func Execute(preview skillplan.Plan, approvedPlanID string, options Options) (result Result, returnedErr error) {
	options = normalizeOptions(options)
	if err := approval.ValidateID(approvedPlanID, preview.PlanID, preview.RequiresApproval); err != nil {
		return Result{}, err
	}
	if preview.SchemaVersion != skillplan.PlanSchema || !preview.Executable {
		return Result{}, errors.New("skill mutation plan is not executable")
	}
	if err := skillplan.ValidateIdentity(preview); err != nil {
		return Result{}, err
	}
	root, err := validateWorkspace(preview.Root)
	if err != nil {
		return Result{}, err
	}
	if filepath.ToSlash(root) != preview.Root {
		return Result{}, errors.New("plan root is not the normalized workspace root")
	}
	fresh, err := rebuildPlan(preview)
	if err != nil {
		return Result{}, fmt.Errorf("rebuild skill plan: %w", err)
	}
	if fresh.PlanID != preview.PlanID {
		return Result{}, errors.New("skill state changed after planning; generate and approve a new plan")
	}
	if preview.Kind == "skill.update" || preview.Kind == "skill.remove" {
		blockers, err := providersync.OwnershipBlockers(root, preview.SkillID)
		if err != nil {
			return Result{}, fmt.Errorf("inspect provider ownership: %w", err)
		}
		if len(blockers) > 0 {
			return Result{}, fmt.Errorf("canonical skill has provider ownership; unsynchronize before mutation: %s", strings.Join(blockers, ", "))
		}
	}
	key, err := planKey(preview.PlanID)
	if err != nil {
		return Result{}, err
	}
	if err := prepareState(root); err != nil {
		return Result{}, err
	}
	tx, err := newTransaction(root, key, preview, options)
	if err != nil {
		return Result{}, err
	}
	started := false
	defer func() {
		if returnedErr != nil && started {
			_ = appendEvent(tx, "skill.execution_failed", map[string]any{"error": returnedErr.Error()})
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
	if err := appendEvent(tx, "skill.execution_started", map[string]any{"operation": preview.Kind}); err != nil {
		tx.abortBeforePrepared()
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "prepared"); err != nil {
		tx.abortBeforePrepared()
		return Result{}, err
	}
	started = true

	if preview.Kind != "skill.remove" {
		source, err := skills.SnapshotSource(preview.SourcePath)
		if err != nil {
			return Result{}, err
		}
		if source.PackageSHA256 != preview.SourceSHA256 {
			return Result{}, errors.New("source package changed after planning")
		}
		if err := copyPackage(source, tx.staging); err != nil {
			return Result{}, err
		}
		staged, err := skills.SnapshotSource(tx.staging)
		if err != nil {
			return Result{}, fmt.Errorf("validate staged package: %w", err)
		}
		if staged.PackageSHA256 != preview.SourceSHA256 {
			return Result{}, errors.New("staged package does not match approved source digest")
		}
		if err := appendEvent(tx, "skill.staged", map[string]any{"sha256": staged.PackageSHA256}); err != nil {
			return Result{}, err
		}
		if err := writeCheckpoint(tx, "staged"); err != nil {
			return Result{}, err
		}
	}
	if err := tx.fail("after_staging"); err != nil {
		return Result{}, err
	}

	fresh, err = rebuildPlan(preview)
	if err != nil || fresh.PlanID != preview.PlanID {
		return Result{}, errors.New("skill state changed before publication; recovery is required")
	}
	if err := tx.fail("before_publish"); err != nil {
		return Result{}, err
	}

	switch preview.Kind {
	case "skill.create":
		if err := publishCreate(tx); err != nil {
			return Result{}, err
		}
	case "skill.update":
		if err := publishUpdate(tx); err != nil {
			return Result{}, err
		}
	case "skill.remove":
		if err := publishRemove(tx); err != nil {
			return Result{}, err
		}
	default:
		return Result{}, fmt.Errorf("unsupported skill mutation kind %q", preview.Kind)
	}
	if err := tx.fail("after_target_publish"); err != nil {
		return Result{}, err
	}
	if err := verifyFinal(root, preview); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "skill.verification_succeeded", map[string]any{"status": "verified"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "verified"); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "skill.execution_completed", map[string]any{"status": "complete"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "complete"); err != nil {
		return Result{}, err
	}
	if err := releaseLock(tx.lock); err != nil {
		return Result{}, err
	}
	started = false
	return tx.result(preview, "complete"), nil
}

func rebuildPlan(preview skillplan.Plan) (skillplan.Plan, error) {
	switch preview.Kind {
	case "skill.create":
		return skillplan.NewCreate(preview.Root, preview.SourcePath)
	case "skill.update":
		return skillplan.NewUpdate(preview.Root, preview.SkillID, preview.SourcePath)
	case "skill.remove":
		return skillplan.NewRemove(preview.Root, preview.SkillID)
	default:
		return skillplan.Plan{}, fmt.Errorf("unsupported skill mutation kind %q", preview.Kind)
	}
}

func newTransaction(root, key string, preview skillplan.Plan, options Options) (*transaction, error) {
	engkit := filepath.Join(root, ".engkit")
	base := filepath.Join(engkit, "recovery", "skills", key)
	if _, err := os.Lstat(base); err == nil {
		return nil, errors.New("transaction state already exists; run skills recover")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	lock := filepath.Join(engkit, "recovery", "skills", "locks", preview.SkillID+".lock")
	for _, artifact := range []string{
		filepath.Join(engkit, "plans", "skills", key+".json"),
		filepath.Join(engkit, "journals", "skills", key+".jsonl"),
		filepath.Join(engkit, "checkpoints", "skills", key+".json"),
	} {
		if _, err := os.Lstat(artifact); err == nil {
			return nil, errors.New("transaction artifacts already exist; run skills recover")
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	if err := os.Mkdir(lock, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, errors.New("skill is locked; run skills recover for the interrupted plan")
		}
		return nil, fmt.Errorf("acquire skill lock: %w", err)
	}
	target := filepath.Join(root, ".agents", "skills", preview.SkillID)
	staging := filepath.Join(base, "staged", preview.SkillID)
	backup := filepath.Join(base, "backup", preview.SkillID)
	tx := &transaction{
		root: root, engkit: engkit, key: key, target: target, staging: staging,
		backup: backup, lock: lock,
		planPath:       filepath.Join(engkit, "plans", "skills", key+".json"),
		journalPath:    filepath.Join(engkit, "journals", "skills", key+".jsonl"),
		checkpointPath: filepath.Join(engkit, "checkpoints", "skills", key+".json"),
		now:            options.Now, failAt: options.FailAt,
	}
	tx.checkpoint = checkpoint{
		SchemaVersion: CheckpointSchema, PlanID: preview.PlanID, Kind: preview.Kind,
		SkillID: preview.SkillID, TargetPath: relative(root, target),
		StagingPath: relative(root, staging), BackupPath: relative(root, backup),
		PlanPath: relative(root, tx.planPath), JournalPath: relative(root, tx.journalPath),
		BeforeSHA256: beforeDigest(preview), AfterSHA256: preview.SourceSHA256,
	}
	return tx, nil
}

func (tx *transaction) start(preview skillplan.Plan, processID int) (returnedErr error) {
	base := filepath.Join(tx.engkit, "recovery", "skills", tx.key)
	defer func() {
		if returnedErr != nil {
			tx.abortBeforePrepared()
		}
	}()
	if err := os.Mkdir(base, 0o700); err != nil {
		return fmt.Errorf("create transaction directory: %w", err)
	}
	if err := os.Mkdir(filepath.Dir(tx.staging), 0o700); err != nil {
		return fmt.Errorf("create staging parent: %w", err)
	}
	if err := os.Mkdir(filepath.Dir(tx.backup), 0o700); err != nil {
		return fmt.Errorf("create backup parent: %w", err)
	}
	lockRecord := map[string]any{
		"schema_version": "engkit.skill-lock/v1alpha1",
		"plan_id":        preview.PlanID,
		"skill_id":       preview.SkillID,
		"process_id":     processID,
		"started_at":     tx.now().UTC().Format(timeFormat),
		"command":        preview.Kind,
	}
	if err := writeNewJSON(filepath.Join(tx.lock, "lock.json"), lockRecord); err != nil {
		return err
	}
	if err := writeNewJSON(tx.planPath, preview); err != nil {
		return err
	}
	return nil
}

func (tx *transaction) abortBeforePrepared() {
	_ = os.Remove(tx.checkpointPath)
	_ = os.Remove(tx.journalPath)
	_ = os.Remove(tx.planPath)
	_ = os.Remove(filepath.Dir(tx.staging))
	_ = os.Remove(filepath.Dir(tx.backup))
	_ = os.Remove(filepath.Join(tx.engkit, "recovery", "skills", tx.key))
	_ = releaseLock(tx.lock)
}

func publishCreate(tx *transaction) error {
	if err := ensureSkillParents(tx.root); err != nil {
		return err
	}
	if _, err := os.Lstat(tx.target); err == nil {
		return errors.New("create target appeared before publication")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(tx.staging, tx.target); err != nil {
		return fmt.Errorf("publish created skill: %w", err)
	}
	if err := appendEvent(tx, "skill.target_published", map[string]any{"target": relative(tx.root, tx.target)}); err != nil {
		return err
	}
	return writeCheckpoint(tx, "target_published")
}

func publishUpdate(tx *transaction) error {
	if err := os.Rename(tx.target, tx.backup); err != nil {
		return fmt.Errorf("publish skill backup: %w", err)
	}
	if err := appendEvent(tx, "skill.backup_published", map[string]any{"backup": relative(tx.root, tx.backup)}); err != nil {
		return err
	}
	if err := writeCheckpoint(tx, "backup_published"); err != nil {
		return err
	}
	if err := tx.fail("after_backup_publish"); err != nil {
		return err
	}
	if err := os.Rename(tx.staging, tx.target); err != nil {
		if restoreErr := os.Rename(tx.backup, tx.target); restoreErr == nil {
			_ = appendEvent(tx, "skill.recovery_completed", map[string]any{"status": "rolled_back"})
			_ = writeCheckpoint(tx, "rolled_back")
		}
		return fmt.Errorf("publish updated skill: %w", err)
	}
	if err := appendEvent(tx, "skill.target_published", map[string]any{"target": relative(tx.root, tx.target)}); err != nil {
		return err
	}
	return writeCheckpoint(tx, "target_published")
}

func publishRemove(tx *transaction) error {
	if err := os.Rename(tx.target, tx.backup); err != nil {
		return fmt.Errorf("publish removed skill backup: %w", err)
	}
	if err := appendEvent(tx, "skill.backup_published", map[string]any{"backup": relative(tx.root, tx.backup)}); err != nil {
		return err
	}
	if err := writeCheckpoint(tx, "backup_published"); err != nil {
		return err
	}
	if err := appendEvent(tx, "skill.target_published", map[string]any{"target_state": "missing"}); err != nil {
		return err
	}
	return writeCheckpoint(tx, "target_published")
}

func verifyFinal(root string, preview skillplan.Plan) error {
	switch preview.Kind {
	case "skill.create", "skill.update":
		snapshot, err := skills.SnapshotCanonical(root, preview.SkillID)
		if err != nil {
			return fmt.Errorf("verify published skill: %w", err)
		}
		if snapshot.PackageSHA256 != preview.SourceSHA256 {
			return errors.New("published skill does not match approved after digest")
		}
	case "skill.remove":
		target := filepath.Join(root, ".agents", "skills", preview.SkillID)
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			if err == nil {
				return errors.New("removed skill target still exists")
			}
			return err
		}
		key, _ := planKey(preview.PlanID)
		backup := filepath.Join(root, ".engkit", "recovery", "skills", key, "backup", preview.SkillID)
		snapshot, err := skills.SnapshotSource(backup)
		if err != nil {
			return fmt.Errorf("verify removed skill backup: %w", err)
		}
		if snapshot.PackageSHA256 != beforeDigest(preview) {
			return errors.New("removed skill backup does not match approved before digest")
		}
	}
	return nil
}

func ensureSkillParents(root string) error {
	for _, path := range []string{filepath.Join(root, ".agents"), filepath.Join(root, ".agents", "skills")} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(path, 0o700); err != nil {
				return fmt.Errorf("create skill parent: %w", err)
			}
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("skill parent %s must be a real directory", path)
		}
	}
	return nil
}

func beforeDigest(preview skillplan.Plan) string {
	for _, precondition := range preview.Preconditions {
		if precondition.ExpectedState == "canonical" {
			return precondition.ExpectedSHA256
		}
	}
	return ""
}

func relative(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(value)
}

func releaseLock(lock string) error {
	lockFile := filepath.Join(lock, "lock.json")
	if err := os.Remove(lockFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove skill lock record: %w", err)
	}
	if err := os.Remove(lock); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("release skill lock: %w", err)
	}
	return nil
}

func (tx *transaction) result(preview skillplan.Plan, status string) Result {
	result := Result{
		SchemaVersion: ResultSchema, Root: filepath.ToSlash(tx.root),
		PlanID: preview.PlanID, Operation: preview.Kind, SkillID: preview.SkillID,
		Status: status, PlanPath: relative(tx.root, tx.planPath),
		JournalPath:    relative(tx.root, tx.journalPath),
		CheckpointPath: relative(tx.root, tx.checkpointPath),
	}
	if preview.Kind == "skill.update" || preview.Kind == "skill.remove" {
		result.BackupPath = relative(tx.root, tx.backup)
	}
	return result
}
