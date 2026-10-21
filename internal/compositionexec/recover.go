package compositionexec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/approval"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/composition"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
)

// Recover completes or rolls back one interrupted approved composition.
func Recover(root, approvedPlanID string, options Options) (Result, error) {
	options = normalizeOptions(options)
	if strings.TrimSpace(approvedPlanID) == "" {
		return Result{}, errors.New("approval is required; pass the complete plan ID with --approve")
	}
	key, err := planKey(approvedPlanID)
	if err != nil {
		return Result{}, err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	absolute = filepath.Clean(absolute)
	name := filepath.Base(absolute)
	parent := filepath.Dir(absolute)
	createStage := filepath.Join(parent, "."+name+".engkit-create-"+key)
	rootPlan := filepath.Join(absolute, ".engkit", "plans", "composition", key+".json")
	stagePlan := filepath.Join(createStage, ".engkit", "plans", "composition", key+".json")
	rootExists := isRegular(rootPlan)
	stageExists := isRegular(stagePlan)
	if rootExists && stageExists {
		return Result{}, errors.New("composition recovery has ambiguous evidence roots; all paths were preserved")
	}
	evidenceRoot := absolute
	planPath := rootPlan
	if stageExists {
		evidenceRoot = createStage
		planPath = stagePlan
	} else if !rootExists {
		return Result{}, errors.New("composition recovery plan was not found")
	}
	preview, err := readPlan(planPath)
	if err != nil {
		return Result{}, fmt.Errorf("load composition recovery plan: %w", err)
	}
	if err := approval.ValidateID(approvedPlanID, preview.PlanID, true); err != nil {
		return Result{}, err
	}
	if filepath.FromSlash(preview.TargetRoot) != absolute {
		return Result{}, errors.New("approved recovery plan belongs to another target")
	}
	checkpointPath := filepath.Join(evidenceRoot, ".engkit", "checkpoints", "composition", key+".json")
	saved, err := readCheckpoint(checkpointPath)
	if err != nil {
		return Result{}, err
	}
	if saved.PlanID != preview.PlanID || saved.Kind != preview.Kind || saved.TargetRoot != preview.TargetRoot {
		return Result{}, errors.New("composition checkpoint does not match the approved plan")
	}
	if saved.Status == "complete" || saved.Status == "rolled_back" {
		return Result{}, fmt.Errorf("composition transaction is already %s", saved.Status)
	}
	journalPath := filepath.Join(evidenceRoot, ".engkit", "journals", "composition", key+".jsonl")
	sequence, err := validateJournal(journalPath, preview.PlanID)
	if err != nil {
		return Result{}, err
	}
	if saved.JournalSequence > sequence {
		return Result{}, errors.New("composition checkpoint refers to a missing journal event")
	}
	saved.JournalSequence = sequence
	tx := &transaction{root: absolute, parent: parent, key: key, stage: createStage, planPath: planPath, journalPath: journalPath, checkpointPath: checkpointPath, checkpoint: saved, now: options.Now, failAt: options.FailAt}
	if preview.Kind == "project.create" {
		tx.backup = filepath.Join(parent, "."+name+".engkit-empty-backup-"+key)
		tx.lock = filepath.Join(parent, "."+name+".engkit-create.lock")
	} else if preview.Kind == "workspace.compose" {
		tx.stage = filepath.Join(absolute, ".engkit", "recovery", "composition", key, "staged")
		tx.lock = filepath.Join(absolute, ".engkit", "recovery", "composition", "locks", "workspace.lock")
	} else {
		return Result{}, fmt.Errorf("unsupported composition recovery kind %q", preview.Kind)
	}
	if err := validateCheckpointLayout(tx, preview); err != nil {
		return Result{}, err
	}
	if err := acquireRecoveryLock(tx, preview, options); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.recovery_started", map[string]any{"previous_status": saved.Status}); err != nil {
		return Result{}, err
	}
	var status string
	if preview.Kind == "project.create" {
		status, err = recoverCreate(tx, preview)
	} else {
		status, err = recoverCompose(tx, preview)
	}
	if err != nil {
		_ = appendEvent(tx, "composition.recovery_refused", map[string]any{"error": err.Error()})
		_ = writeCheckpoint(tx, "failed")
		return Result{}, err
	}
	if err := tx.fail("before_recovery_complete"); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.recovery_completed", map[string]any{"status": status}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, status); err != nil {
		return Result{}, err
	}
	if err := releaseLock(tx.lock); err != nil {
		return Result{}, err
	}
	retained := ""
	if status == "rolled_back" || preview.Kind == "workspace.compose" {
		retained = filepath.ToSlash(tx.stage)
	}
	return resultFor(tx, preview, status, retained), nil
}

func recoverCreate(tx *transaction, preview composition.Plan) (string, error) {
	targetComplete := verifyPlannedOutputs(tx.root, preview.RenderedOutputs) == nil && validateInitialized(tx.root) == nil
	stagePresent := !isMissing(tx.stage)
	stageComplete := verifyPlannedOutputs(tx.stage, preview.RenderedOutputs) == nil && validateInitialized(tx.stage) == nil
	rootMissing := isMissing(tx.root)
	rootEmpty := isEmptyRealDirectory(tx.root)
	stageMissing := isMissing(tx.stage)
	if targetComplete && stageMissing {
		tx.planPath = filepath.Join(tx.root, filepath.FromSlash(tx.checkpoint.PlanPath))
		tx.journalPath = filepath.Join(tx.root, filepath.FromSlash(tx.checkpoint.JournalPath))
		tx.checkpointPath = filepath.Join(tx.root, ".engkit", "checkpoints", "composition", tx.key+".json")
		if isEmptyRealDirectory(tx.backup) {
			if err := os.Remove(tx.backup); err != nil {
				return "", err
			}
		} else if !isMissing(tx.backup) {
			return "", errors.New("empty destination backup changed; recovery refused")
		}
		return "complete", nil
	}
	if stagePresent && stageComplete && (rootMissing || rootEmpty) {
		if !isMissing(tx.backup) {
			if !isEmptyRealDirectory(tx.backup) || !rootMissing {
				return "", errors.New("empty destination backup state is ambiguous")
			}
			if err := os.Rename(tx.backup, tx.root); err != nil {
				return "", fmt.Errorf("restore empty destination: %w", err)
			}
		}
		return "rolled_back", nil
	}
	return "", errors.New("project creation recovery state is ambiguous; all paths were preserved")
}

func recoverCompose(tx *transaction, preview composition.Plan) (string, error) {
	if verifyPlannedOutputs(tx.root, preview.RenderedOutputs) == nil {
		return "complete", nil
	}
	byPath := map[string]composition.RenderedOutput{}
	for _, output := range preview.RenderedOutputs {
		byPath[output.Path] = output
	}
	allowedDirs := map[string]struct{}{}
	for _, operation := range preview.Operations {
		if operation.Action == "create_directory" {
			allowedDirs[operation.Path] = struct{}{}
		}
	}
	for _, relative := range tx.checkpoint.CreatedDirs {
		if _, ok := allowedDirs[relative]; !ok {
			return "", fmt.Errorf("checkpoint directory %s is not in the approved plan", relative)
		}
	}
	for _, relative := range tx.checkpoint.AppliedOutputs {
		output, ok := byPath[relative]
		if !ok {
			return "", fmt.Errorf("checkpoint output %s is not in the approved plan", relative)
		}
		digest, err := fileDigest(filepath.Join(tx.root, filepath.FromSlash(relative)))
		if err != nil || digest != output.SHA256 {
			return "", fmt.Errorf("applied output %s changed; recovery refused", relative)
		}
	}
	for index := len(tx.checkpoint.AppliedOutputs) - 1; index >= 0; index-- {
		relative := tx.checkpoint.AppliedOutputs[index]
		if err := os.Remove(filepath.Join(tx.root, filepath.FromSlash(relative))); err != nil {
			return "", fmt.Errorf("remove exact applied output %s: %w", relative, err)
		}
	}
	directories := append([]string(nil), tx.checkpoint.CreatedDirs...)
	sort.SliceStable(directories, func(i, j int) bool {
		left, right := strings.Count(directories[i], "/"), strings.Count(directories[j], "/")
		if left != right {
			return left > right
		}
		return directories[i] > directories[j]
	})
	for _, relative := range directories {
		err := os.Remove(filepath.Join(tx.root, filepath.FromSlash(relative)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("remove transaction-created directory %s: %w", relative, err)
		}
	}
	return "rolled_back", nil
}

func acquireRecoveryLock(tx *transaction, preview composition.Plan, options Options) error {
	info, err := os.Lstat(tx.lock)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(tx.lock, 0o700); err != nil {
			return err
		}
		if err := writeNewJSON(filepath.Join(tx.lock, "lock.json"), lockRecord(preview, options, "project.recover")); err != nil {
			_ = os.Remove(tx.lock)
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("composition recovery lock is not a real directory")
	}
	contents, err := os.ReadFile(filepath.Join(tx.lock, "lock.json"))
	if err != nil {
		return err
	}
	var record struct {
		SchemaVersion string `json:"schema_version"`
		PlanID        string `json:"plan_id"`
		TargetRoot    string `json:"target_root"`
		ProcessID     int    `json:"process_id"`
		StartedAt     string `json:"started_at"`
		Command       string `json:"command"`
	}
	if err := decodeStrict(contents, &record); err != nil {
		return err
	}
	if record.SchemaVersion != LockSchema || record.PlanID != preview.PlanID || record.TargetRoot != preview.TargetRoot {
		return errors.New("existing composition lock belongs to another transaction")
	}
	return nil
}

func verifyPlannedOutputs(root string, outputs []composition.RenderedOutput) error {
	for _, output := range outputs {
		digest, err := fileDigest(filepath.Join(root, filepath.FromSlash(output.Path)))
		if err != nil || digest != output.SHA256 {
			return fmt.Errorf("output %s does not match", output.Path)
		}
	}
	seenSkills := map[string]struct{}{}
	for _, output := range outputs {
		if !strings.HasPrefix(output.Owner, "skill:") {
			continue
		}
		parts := strings.Split(output.Path, "/")
		if len(parts) >= 3 {
			seenSkills[strings.Join(parts[:3], "/")] = struct{}{}
		}
	}
	for skillRoot := range seenSkills {
		if _, err := skills.SnapshotSource(filepath.Join(root, filepath.FromSlash(skillRoot))); err != nil {
			return err
		}
	}
	return nil
}

func validateCheckpointLayout(tx *transaction, preview composition.Plan) error {
	expectedPlan := ".engkit/plans/composition/" + tx.key + ".json"
	expectedJournal := ".engkit/journals/composition/" + tx.key + ".jsonl"
	if tx.checkpoint.PlanPath != expectedPlan || tx.checkpoint.JournalPath != expectedJournal {
		return errors.New("composition checkpoint evidence paths do not match the transaction layout")
	}
	if preview.Kind == "project.create" {
		if tx.checkpoint.StagingPath != filepath.ToSlash(filepath.Base(tx.stage)) || tx.checkpoint.BackupPath != filepath.ToSlash(filepath.Base(tx.backup)) || tx.checkpoint.LockPath != filepath.ToSlash(filepath.Base(tx.lock)) || tx.checkpoint.TargetBefore != preview.WorkspaceFacts.TargetState {
			return errors.New("project creation checkpoint paths do not match the transaction layout")
		}
		return nil
	}
	if tx.checkpoint.StagingPath != slashRelative(tx.root, tx.stage) || tx.checkpoint.LockPath != slashRelative(tx.root, tx.lock) || tx.checkpoint.BackupPath != "" || tx.checkpoint.TargetBefore != preview.WorkspaceFacts.TreeSHA256 {
		return errors.New("workspace composition checkpoint paths do not match the transaction layout")
	}
	return nil
}

func isRegular(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular()
}

func isMissing(path string) bool {
	_, err := os.Lstat(path)
	return errors.Is(err, os.ErrNotExist)
}

func isEmptyRealDirectory(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) == 0
}
