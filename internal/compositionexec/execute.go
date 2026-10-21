package compositionexec

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/approval"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/composition"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
)

const (
	configContents    = "schema_version: engkit.config/v1alpha1\nworkspace:\n  source_layout: auto\n"
	gitignoreContents = "/cache/*\n/context/*.db\n/context/*.db-*\n/recovery/*\n"
)

// Execute applies one freshly rebuilt and exactly approved composition plan.
func Execute(preview composition.Plan, approvedPlanID string, inventory catalog.Catalog, catalogFS fs.FS, options Options) (Result, error) {
	options = normalizeOptions(options)
	if err := approval.ValidateID(approvedPlanID, preview.PlanID, preview.RequiresApproval); err != nil {
		return Result{}, err
	}
	if preview.SchemaVersion != composition.PlanSchema || !preview.Executable {
		return Result{}, errors.New("composition plan is not executable")
	}
	for _, conflict := range preview.Conflicts {
		if conflict.Severity == "blocking" {
			return Result{}, errors.New("composition plan contains a blocking conflict")
		}
	}
	files, err := composition.Materialize(preview, inventory, catalogFS)
	if err != nil {
		return Result{}, err
	}
	key, err := planKey(preview.PlanID)
	if err != nil {
		return Result{}, err
	}
	switch preview.Kind {
	case "project.create":
		return executeCreate(preview, files, key, approvedPlanID, options)
	case "workspace.compose":
		return executeCompose(preview, files, key, approvedPlanID, options)
	default:
		return Result{}, fmt.Errorf("unsupported composition kind %q", preview.Kind)
	}
}

func executeCreate(preview composition.Plan, files []composition.MaterializedFile, key, approved string, options Options) (result Result, returnedErr error) {
	root := filepath.FromSlash(preview.TargetRoot)
	parent := filepath.Dir(root)
	if err := validateRealParent(parent); err != nil {
		return Result{}, err
	}
	name := filepath.Base(root)
	stage := filepath.Join(parent, "."+name+".engkit-create-"+key)
	backup := filepath.Join(parent, "."+name+".engkit-empty-backup-"+key)
	lock := filepath.Join(parent, "."+name+".engkit-create.lock")
	for _, candidate := range []string{stage, backup} {
		if _, err := os.Lstat(candidate); err == nil {
			return Result{}, errors.New("project creation transaction state already exists; run project recover")
		} else if !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
	}
	if err := os.Mkdir(lock, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Result{}, errors.New("project destination is locked; run project recover")
		}
		return Result{}, err
	}
	prepared := false
	defer func() {
		if returnedErr != nil && !prepared {
			abortCreateBeforePrepared(stage, lock, key)
		}
	}()
	if err := writeNewJSON(filepath.Join(lock, "lock.json"), lockRecord(preview, options, "project.create")); err != nil {
		return Result{}, err
	}
	if err := os.Mkdir(stage, 0o700); err != nil {
		return Result{}, err
	}
	for _, relative := range []string{".engkit", ".engkit/plans", ".engkit/plans/composition", ".engkit/journals", ".engkit/journals/composition", ".engkit/checkpoints", ".engkit/checkpoints/composition", ".engkit/context", ".engkit/cache", ".engkit/recovery", ".engkit/recovery/composition", ".engkit/recovery/composition/" + key} {
		if err := ensureRealDirectory(filepath.Join(stage, filepath.FromSlash(relative))); err != nil {
			return Result{}, err
		}
	}
	engkit := filepath.Join(stage, ".engkit")
	tx := &transaction{root: root, parent: parent, key: key, stage: stage, backup: backup, lock: lock, now: options.Now, failAt: options.FailAt}
	tx.planPath = filepath.Join(engkit, "plans", "composition", key+".json")
	tx.journalPath = filepath.Join(engkit, "journals", "composition", key+".jsonl")
	tx.checkpointPath = filepath.Join(engkit, "checkpoints", "composition", key+".json")
	tx.checkpoint = checkpoint{SchemaVersion: CheckpointSchema, PlanID: preview.PlanID, Kind: preview.Kind, TargetRoot: preview.TargetRoot, PlanPath: ".engkit/plans/composition/" + key + ".json", JournalPath: ".engkit/journals/composition/" + key + ".jsonl", StagingPath: filepath.ToSlash(filepath.Base(stage)), BackupPath: filepath.ToSlash(filepath.Base(backup)), LockPath: filepath.ToSlash(filepath.Base(lock)), TargetBefore: preview.WorkspaceFacts.TargetState, AppliedOutputs: []string{}, CreatedDirs: []string{}}
	defer func() {
		if returnedErr != nil && prepared {
			_ = appendEvent(tx, "composition.execution_failed", map[string]any{"error": returnedErr.Error()})
			_ = writeCheckpoint(tx, "failed")
		}
	}()
	if err := writeNewJSON(tx.planPath, preview); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "approval.accepted", map[string]any{"approved_plan_id": approved}); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.execution_started", map[string]any{"kind": preview.Kind}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "prepared"); err != nil {
		return Result{}, err
	}
	prepared = true
	if err := createPlannedDirectories(stage, preview.Operations, nil); err != nil {
		return Result{}, err
	}
	if err := writeNewSynced(filepath.Join(stage, ".engkit", "config.yaml"), []byte(configContents)); err != nil {
		return Result{}, err
	}
	if err := writeNewSynced(filepath.Join(stage, ".engkit", ".gitignore"), []byte(gitignoreContents)); err != nil {
		return Result{}, err
	}
	if err := writeMaterialized(stage, files); err != nil {
		return Result{}, err
	}
	if err := verifyOutputs(stage, files); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.staged", map[string]any{"output_count": len(files)}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "staged"); err != nil {
		return Result{}, err
	}
	if err := tx.fail("after_staging"); err != nil {
		return Result{}, err
	}
	if err := validateCreateTarget(root, preview.WorkspaceFacts.TargetState); err != nil {
		return Result{}, err
	}
	if err := tx.fail("before_publish"); err != nil {
		return Result{}, err
	}
	if preview.WorkspaceFacts.TargetState == "empty_directory" {
		if err := os.Rename(root, backup); err != nil {
			return Result{}, fmt.Errorf("retain empty destination backup: %w", err)
		}
		if err := tx.fail("after_empty_backup"); err != nil {
			return Result{}, err
		}
	}
	if err := os.Rename(stage, root); err != nil {
		if preview.WorkspaceFacts.TargetState == "empty_directory" {
			_ = os.Rename(backup, root)
		}
		return Result{}, fmt.Errorf("publish project: %w", err)
	}
	tx.stage = ""
	tx.planPath = filepath.Join(root, filepath.FromSlash(tx.checkpoint.PlanPath))
	tx.journalPath = filepath.Join(root, filepath.FromSlash(tx.checkpoint.JournalPath))
	tx.checkpointPath = filepath.Join(root, ".engkit", "checkpoints", "composition", key+".json")
	if err := appendEvent(tx, "composition.target_published", map[string]any{"target": preview.TargetRoot}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "published"); err != nil {
		return Result{}, err
	}
	if err := tx.fail("after_target_publish"); err != nil {
		return Result{}, err
	}
	if err := verifyCreatedProject(root, files); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.verification_succeeded", map[string]any{"status": "verified"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "verified"); err != nil {
		return Result{}, err
	}
	if err := tx.fail("before_complete"); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.execution_completed", map[string]any{"status": "complete"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "complete"); err != nil {
		return Result{}, err
	}
	if preview.WorkspaceFacts.TargetState == "empty_directory" {
		if err := removeExactEmptyDirectory(backup); err != nil {
			return Result{}, err
		}
	}
	if err := releaseLock(lock); err != nil {
		return Result{}, err
	}
	return resultFor(tx, preview, "complete", ""), nil
}

func executeCompose(preview composition.Plan, files []composition.MaterializedFile, key, approved string, options Options) (result Result, returnedErr error) {
	root := filepath.FromSlash(preview.TargetRoot)
	if err := validateInitialized(root); err != nil {
		return Result{}, err
	}
	engkit := filepath.Join(root, ".engkit")
	for _, relative := range []string{"plans/composition", "journals/composition", "checkpoints/composition", "recovery/composition", "recovery/composition/locks"} {
		if err := ensureDirectoryTree(engkit, relative); err != nil {
			return Result{}, err
		}
	}
	lock := filepath.Join(engkit, "recovery", "composition", "locks", "workspace.lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Result{}, errors.New("workspace composition is locked; run project recover")
		}
		return Result{}, err
	}
	base := filepath.Join(engkit, "recovery", "composition", key)
	stage := filepath.Join(base, "staged")
	if _, err := os.Lstat(base); err == nil {
		_ = os.Remove(lock)
		return Result{}, errors.New("composition transaction state already exists; run project recover")
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(lock)
		return Result{}, err
	}
	if err := writeNewJSON(filepath.Join(lock, "lock.json"), lockRecord(preview, options, "project.compose")); err != nil {
		_ = os.Remove(lock)
		return Result{}, err
	}
	if err := os.Mkdir(base, 0o700); err != nil {
		return Result{}, err
	}
	if err := os.Mkdir(stage, 0o700); err != nil {
		return Result{}, err
	}
	tx := &transaction{root: root, key: key, stage: stage, lock: lock, now: options.Now, failAt: options.FailAt}
	tx.planPath = filepath.Join(engkit, "plans", "composition", key+".json")
	tx.journalPath = filepath.Join(engkit, "journals", "composition", key+".jsonl")
	tx.checkpointPath = filepath.Join(engkit, "checkpoints", "composition", key+".json")
	tx.checkpoint = checkpoint{SchemaVersion: CheckpointSchema, PlanID: preview.PlanID, Kind: preview.Kind, TargetRoot: preview.TargetRoot, PlanPath: slashRelative(root, tx.planPath), JournalPath: slashRelative(root, tx.journalPath), StagingPath: slashRelative(root, stage), LockPath: slashRelative(root, lock), TargetBefore: preview.WorkspaceFacts.TreeSHA256, AppliedOutputs: []string{}, CreatedDirs: []string{}}
	prepared := false
	defer func() {
		if returnedErr != nil && !prepared {
			abortComposeBeforePrepared(tx, base)
		}
	}()
	if err := writeNewJSON(tx.planPath, preview); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "approval.accepted", map[string]any{"approved_plan_id": approved}); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.execution_started", map[string]any{"kind": preview.Kind}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "prepared"); err != nil {
		return Result{}, err
	}
	prepared = true
	defer func() {
		if returnedErr != nil && prepared {
			_ = appendEvent(tx, "composition.execution_failed", map[string]any{"error": returnedErr.Error()})
			_ = writeCheckpoint(tx, "failed")
		}
	}()
	if err := createPlannedDirectories(stage, preview.Operations, nil); err != nil {
		return Result{}, err
	}
	if err := writeMaterialized(stage, files); err != nil {
		return Result{}, err
	}
	if err := verifyOutputs(stage, files); err != nil {
		return Result{}, err
	}
	if err := probeNoReplace(base); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.staged", map[string]any{"output_count": len(files)}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "staged"); err != nil {
		return Result{}, err
	}
	if err := tx.fail("after_staging"); err != nil {
		return Result{}, err
	}
	if err := publishCompose(tx, preview, files); err != nil {
		return Result{}, err
	}
	if err := tx.fail("after_target_publish"); err != nil {
		return Result{}, err
	}
	if err := verifyOutputs(root, files); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.verification_succeeded", map[string]any{"status": "verified"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "verified"); err != nil {
		return Result{}, err
	}
	if err := tx.fail("before_complete"); err != nil {
		return Result{}, err
	}
	if err := appendEvent(tx, "composition.execution_completed", map[string]any{"status": "complete"}); err != nil {
		return Result{}, err
	}
	if err := writeCheckpoint(tx, "complete"); err != nil {
		return Result{}, err
	}
	if err := releaseLock(lock); err != nil {
		return Result{}, err
	}
	return resultFor(tx, preview, "complete", slashRelative(root, base)), nil
}

func publishCompose(tx *transaction, preview composition.Plan, files []composition.MaterializedFile) error {
	skillRoots := skillPublicationRoots(files)
	for _, operation := range preview.Operations {
		if operation.Action != "create_directory" {
			continue
		}
		if belongsToRoot(operation.Path, skillRoots) {
			continue
		}
		target := filepath.Join(tx.root, filepath.FromSlash(operation.Path))
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			if err == nil {
				return fmt.Errorf("planned directory %s appeared before publication", operation.Path)
			}
			return err
		}
		if err := os.Mkdir(target, 0o700); err != nil {
			return err
		}
		tx.checkpoint.CreatedDirs = append(tx.checkpoint.CreatedDirs, operation.Path)
		if err := appendEvent(tx, "composition.operation_completed", map[string]any{"action": "create_directory", "path": operation.Path}); err != nil {
			return err
		}
		if err := writeCheckpoint(tx, "publishing"); err != nil {
			return err
		}
		if err := tx.fail("after_directory_publish"); err != nil {
			return err
		}
	}
	for _, root := range skillRoots {
		target := filepath.Join(tx.root, filepath.FromSlash(root))
		stage := filepath.Join(tx.stage, filepath.FromSlash(root))
		if _, err := os.Lstat(target); errors.Is(err, os.ErrNotExist) {
			if err := os.Rename(stage, target); err != nil {
				return err
			}
			for _, file := range files {
				if file.Path == root || strings.HasPrefix(file.Path, root+"/") {
					tx.checkpoint.AppliedOutputs = append(tx.checkpoint.AppliedOutputs, file.Path)
				}
			}
			for _, operation := range preview.Operations {
				if operation.Action == "create_directory" && belongsToRoot(operation.Path, []string{root}) {
					tx.checkpoint.CreatedDirs = append(tx.checkpoint.CreatedDirs, operation.Path)
				}
			}
			if err := appendEvent(tx, "composition.operation_completed", map[string]any{"action": "publish_skill", "path": root}); err != nil {
				return err
			}
			if err := writeCheckpoint(tx, "publishing"); err != nil {
				return err
			}
			if err := tx.fail("after_output_publish"); err != nil {
				return err
			}
			continue
		} else if err != nil {
			return err
		}
		if !allOutputsMatch(target, root, files) {
			return fmt.Errorf("canonical skill target %s is not an exact approved package", root)
		}
	}
	for _, file := range files {
		if belongsToRoot(file.Path, skillRoots) {
			continue
		}
		target := filepath.Join(tx.root, filepath.FromSlash(file.Path))
		if digest, err := fileDigest(target); err == nil && digest == file.SHA256 {
			continue
		} else if err == nil || !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("planned output %s changed before publication", file.Path)
		}
		staged := filepath.Join(tx.stage, filepath.FromSlash(file.Path))
		if err := os.Link(staged, target); err != nil {
			return fmt.Errorf("publish output %s without replacement: %w", file.Path, err)
		}
		if err := os.Remove(staged); err != nil {
			return err
		}
		tx.checkpoint.AppliedOutputs = append(tx.checkpoint.AppliedOutputs, file.Path)
		if err := appendEvent(tx, "composition.operation_completed", map[string]any{"action": "write_file", "path": file.Path, "sha256": file.SHA256}); err != nil {
			return err
		}
		if err := writeCheckpoint(tx, "publishing"); err != nil {
			return err
		}
		if err := tx.fail("after_output_publish"); err != nil {
			return err
		}
	}
	if err := appendEvent(tx, "composition.target_published", map[string]any{"output_count": len(files)}); err != nil {
		return err
	}
	return writeCheckpoint(tx, "published")
}

func writeMaterialized(root string, files []composition.MaterializedFile) error {
	for _, file := range files {
		target := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := ensureDirectoryTree(root, filepath.ToSlash(filepath.Dir(file.Path))); err != nil {
			return err
		}
		if err := writeNewSynced(target, file.Data); err != nil {
			return err
		}
	}
	return nil
}

func createPlannedDirectories(root string, operations []composition.Operation, created *[]string) error {
	for _, operation := range operations {
		if operation.Action != "create_directory" {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(operation.Path))
		if err := ensureRealDirectory(full); err != nil {
			return err
		}
		if created != nil {
			*created = append(*created, operation.Path)
		}
	}
	return nil
}

func ensureDirectoryTree(root, relative string) error {
	if relative == "." || relative == "" {
		return nil
	}
	current := root
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		if part == "" || part == "." || part == ".." {
			return errors.New("invalid directory path")
		}
		current = filepath.Join(current, part)
		if err := ensureRealDirectory(current); err != nil {
			return err
		}
	}
	return nil
}

func verifyOutputs(root string, files []composition.MaterializedFile) error {
	for _, file := range files {
		digest, err := fileDigest(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil {
			return fmt.Errorf("verify output %s: %w", file.Path, err)
		}
		if digest != file.SHA256 {
			return fmt.Errorf("output %s does not match the approved digest", file.Path)
		}
	}
	for _, skillRoot := range skillPublicationRoots(files) {
		if _, err := skills.SnapshotSource(filepath.Join(root, filepath.FromSlash(skillRoot))); err != nil {
			return fmt.Errorf("validate canonical skill %s: %w", skillRoot, err)
		}
	}
	return nil
}

func verifyCreatedProject(root string, files []composition.MaterializedFile) error {
	if err := validateInitialized(root); err != nil {
		return err
	}
	if contents, err := os.ReadFile(filepath.Join(root, ".engkit", "config.yaml")); err != nil || string(contents) != configContents {
		return errors.New("created workspace config does not match the bootstrap contract")
	}
	if contents, err := os.ReadFile(filepath.Join(root, ".engkit", ".gitignore")); err != nil || string(contents) != gitignoreContents {
		return errors.New("created workspace ignore rules do not match the bootstrap contract")
	}
	return verifyOutputs(root, files)
}

func validateInitialized(root string) error {
	for _, relative := range []string{".", ".engkit"} {
		info, err := os.Lstat(filepath.Join(root, relative))
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("workspace composition requires an initialized real workspace")
		}
	}
	for _, relative := range []string{".engkit/config.yaml", ".engkit/.gitignore"} {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("initialized workspace requires regular %s", relative)
		}
	}
	return nil
}

func validateRealParent(parent string) error {
	info, err := os.Lstat(parent)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("project destination parent must be a real directory")
	}
	return nil
}

func validateCreateTarget(root, expected string) error {
	info, err := os.Lstat(root)
	if expected == "missing" {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errors.New("project destination changed after planning")
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("project destination changed after planning")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		return errors.New("project destination is no longer empty")
	}
	return nil
}

func probeNoReplace(base string) error {
	source := filepath.Join(base, ".no-replace-source")
	target := filepath.Join(base, ".no-replace-target")
	if err := writeNewSynced(source, []byte("probe")); err != nil {
		return err
	}
	defer os.Remove(source)
	defer os.Remove(target)
	if err := os.Link(source, target); err != nil {
		return fmt.Errorf("%w: %v", ErrNoReplaceUnsupported, err)
	}
	return nil
}

func skillPublicationRoots(files []composition.MaterializedFile) []string {
	set := map[string]struct{}{}
	for _, file := range files {
		if !strings.HasPrefix(file.Owner, "skill:") {
			continue
		}
		parts := strings.Split(file.Path, "/")
		if len(parts) >= 3 && parts[0] == ".agents" && parts[1] == "skills" {
			set[path.Join(parts[0], parts[1], parts[2])] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func belongsToRoot(value string, roots []string) bool {
	for _, root := range roots {
		if value == root || strings.HasPrefix(value, root+"/") {
			return true
		}
	}
	return false
}

func allOutputsMatch(target, root string, files []composition.MaterializedFile) bool {
	expectedFiles := map[string]string{}
	expectedDirs := map[string]struct{}{".": {}}
	for _, file := range files {
		if file.Path != root && !strings.HasPrefix(file.Path, root+"/") {
			continue
		}
		relative := strings.TrimPrefix(strings.TrimPrefix(file.Path, root), "/")
		expectedFiles[relative] = file.SHA256
		for parent := path.Dir(relative); parent != "."; parent = path.Dir(parent) {
			expectedDirs[parent] = struct{}{}
		}
	}
	seen := 0
	err := filepath.WalkDir(target, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(target, filename)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("link")
		}
		if entry.IsDir() {
			if _, ok := expectedDirs[relative]; !ok {
				return errors.New("unexpected directory")
			}
			return nil
		}
		expected, ok := expectedFiles[relative]
		if !ok {
			return errors.New("unexpected file")
		}
		digest, err := fileDigest(filename)
		if err != nil || digest != expected {
			return errors.New("digest mismatch")
		}
		seen++
		return nil
	})
	return err == nil && seen == len(expectedFiles)
}

func lockRecord(preview composition.Plan, options Options, command string) map[string]any {
	return map[string]any{"schema_version": LockSchema, "plan_id": preview.PlanID, "target_root": preview.TargetRoot, "process_id": options.ProcessID, "started_at": options.Now().UTC().Format(timeFormat), "command": command}
}

func releaseLock(lock string) error {
	if err := os.Remove(filepath.Join(lock, "lock.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Remove(lock)
}

func removeExactEmptyDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("empty destination backup changed; refusing cleanup")
	}
	return os.Remove(path)
}

func resultFor(tx *transaction, preview composition.Plan, status, retained string) Result {
	result := Result{SchemaVersion: ResultSchema, Root: preview.TargetRoot, PlanID: preview.PlanID, Kind: preview.Kind, Status: status, PlanPath: tx.checkpoint.PlanPath, JournalPath: tx.checkpoint.JournalPath, CheckpointPath: ".engkit/checkpoints/composition/" + tx.key + ".json", RetainedPath: retained}
	if preview.Kind == "project.create" && status == "rolled_back" {
		result.PlanPath = filepath.ToSlash(tx.planPath)
		result.JournalPath = filepath.ToSlash(tx.journalPath)
		result.CheckpointPath = filepath.ToSlash(tx.checkpointPath)
	}
	return result
}

func abortCreateBeforePrepared(stage, lock, key string) {
	engkit := filepath.Join(stage, ".engkit")
	_ = os.Remove(filepath.Join(engkit, "checkpoints", "composition", key+".json"))
	_ = os.Remove(filepath.Join(engkit, "journals", "composition", key+".jsonl"))
	_ = os.Remove(filepath.Join(engkit, "plans", "composition", key+".json"))
	for _, relative := range []string{"recovery/composition/" + key, "recovery/composition", "recovery", "cache", "context", "checkpoints/composition", "checkpoints", "journals/composition", "journals", "plans/composition", "plans"} {
		_ = os.Remove(filepath.Join(engkit, filepath.FromSlash(relative)))
	}
	_ = os.Remove(engkit)
	_ = os.Remove(stage)
	_ = os.Remove(filepath.Join(lock, "lock.json"))
	_ = os.Remove(lock)
}

func abortComposeBeforePrepared(tx *transaction, base string) {
	_ = os.Remove(tx.checkpointPath)
	_ = os.Remove(tx.journalPath)
	_ = os.Remove(tx.planPath)
	_ = os.Remove(tx.stage)
	_ = os.Remove(base)
	_ = os.Remove(filepath.Join(tx.lock, "lock.json"))
	_ = os.Remove(tx.lock)
}
