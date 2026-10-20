package skillexec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillplan"
)

func TestExecuteRequiresExactApprovalWithoutWriting(t *testing.T) {
	root := initializedWorkspace(t)
	source := writeSkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(preview, "sha256:wrong", testOptions()); err == nil {
		t.Fatal("Execute() expected approval error")
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit", "plans", "skills")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("approval failure wrote execution state: %v", err)
	}
}

func TestExecuteRejectsChangedSourceWithoutWriting(t *testing.T) {
	root := initializedWorkspace(t)
	parent := t.TempDir()
	source := writeSkill(t, parent, "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	writeSkill(t, parent, "release-review", "1.0.0", "# Changed\n")
	if _, err := Execute(preview, preview.PlanID, testOptions()); err == nil ||
		!strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit", "plans", "skills")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("state-change refusal wrote execution state: %v", err)
	}
}

func TestExecuteCreatePublishesArtifacts(t *testing.T) {
	root := initializedWorkspace(t)
	source := writeSkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Execute(preview, preview.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" || result.BackupPath != "" {
		t.Fatalf("unexpected result: %#v", result)
	}
	assertSkillVersion(t, root, "release-review", "1.0.0")
	assertArtifacts(t, root, preview, "complete")
	if _, err := os.Stat(filepath.Join(root, ".engkit", "recovery", "skills", "locks", "release-review.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed execution retained lock: %v", err)
	}
}

func TestExecuteUpdateAndRemoveRetainBackups(t *testing.T) {
	root := initializedWorkspace(t)
	writeSkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Current\n")
	source := writeSkill(t, t.TempDir(), "release-review", "1.1.0", "# Updated\n")
	update, err := skillplan.NewUpdate(root, "release-review", source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Execute(update, update.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	assertSkillVersion(t, root, "release-review", "1.1.0")
	assertBackupVersion(t, root, result.BackupPath, "1.0.0")

	remove, err := skillplan.NewRemove(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	result, err = Execute(remove, remove.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "release-review")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove left target: %v", err)
	}
	assertBackupVersion(t, root, result.BackupPath, "1.1.0")
}

func TestRecoverRestoresInterruptedUpdate(t *testing.T) {
	root := initializedWorkspace(t)
	writeSkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Current\n")
	source := writeSkill(t, t.TempDir(), "release-review", "1.1.0", "# Updated\n")
	preview, err := skillplan.NewUpdate(root, "release-review", source)
	if err != nil {
		t.Fatal(err)
	}
	options := testOptions()
	options.FailAt = func(point string) error {
		if point == "after_backup_publish" {
			return errors.New("stop")
		}
		return nil
	}
	if _, err := Execute(preview, preview.PlanID, options); err == nil {
		t.Fatal("Execute() expected injected failure")
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "release-review")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("interrupted update target state = %v", err)
	}
	result, err := Recover(root, "release-review", preview.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rolled_back" {
		t.Fatalf("recovery status = %q", result.Status)
	}
	assertSkillVersion(t, root, "release-review", "1.0.0")
}

func TestRecoverCompletesPublishedCreate(t *testing.T) {
	root := initializedWorkspace(t)
	source := writeSkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	options := testOptions()
	options.FailAt = func(point string) error {
		if point == "after_target_publish" {
			return errors.New("stop")
		}
		return nil
	}
	if _, err := Execute(preview, preview.PlanID, options); err == nil {
		t.Fatal("Execute() expected injected failure")
	}
	result, err := Recover(root, "release-review", preview.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("recovery status = %q", result.Status)
	}
	assertSkillVersion(t, root, "release-review", "1.0.0")
}

func TestRecoverRollsBackStagedCreate(t *testing.T) {
	root := initializedWorkspace(t)
	source := writeSkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	executeWithFailure(t, preview, "after_staging")

	result, err := Recover(root, "release-review", preview.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rolled_back" {
		t.Fatalf("recovery status = %q", result.Status)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "release-review")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rolled-back create target state = %v", err)
	}
}

func TestRecoverCompletesPublishedUpdate(t *testing.T) {
	root := initializedWorkspace(t)
	writeSkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Current\n")
	source := writeSkill(t, t.TempDir(), "release-review", "1.1.0", "# Updated\n")
	preview, err := skillplan.NewUpdate(root, "release-review", source)
	if err != nil {
		t.Fatal(err)
	}
	executeWithFailure(t, preview, "after_target_publish")

	result, err := Recover(root, "release-review", preview.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("recovery status = %q", result.Status)
	}
	assertSkillVersion(t, root, "release-review", "1.1.0")
	assertBackupVersion(t, root, result.BackupPath, "1.0.0")
}

func TestRecoverCompletesPublishedRemove(t *testing.T) {
	root := initializedWorkspace(t)
	writeSkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Current\n")
	preview, err := skillplan.NewRemove(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	executeWithFailure(t, preview, "after_target_publish")

	result, err := Recover(root, "release-review", preview.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("recovery status = %q", result.Status)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "release-review")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovered remove target state = %v", err)
	}
	assertBackupVersion(t, root, result.BackupPath, "1.0.0")
}

func TestRecoverRollsBackUnpublishedRemove(t *testing.T) {
	root := initializedWorkspace(t)
	writeSkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Current\n")
	preview, err := skillplan.NewRemove(root, "release-review")
	if err != nil {
		t.Fatal(err)
	}
	executeWithFailure(t, preview, "before_publish")

	result, err := Recover(root, "release-review", preview.PlanID, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rolled_back" {
		t.Fatalf("recovery status = %q", result.Status)
	}
	assertSkillVersion(t, root, "release-review", "1.0.0")
}

func TestRecoverRefusesAmbiguousTarget(t *testing.T) {
	root := initializedWorkspace(t)
	source := writeSkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	options := testOptions()
	options.FailAt = func(point string) error {
		if point == "after_staging" {
			return errors.New("stop")
		}
		return nil
	}
	if _, err := Execute(preview, preview.PlanID, options); err == nil {
		t.Fatal("Execute() expected injected failure")
	}
	writeSkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "9.0.0", "# Unexpected\n")
	if _, err := Recover(root, "release-review", preview.PlanID, testOptions()); err == nil ||
		!strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Recover() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit", "recovery", "skills", "locks", "release-review.lock")); err != nil {
		t.Fatalf("refused recovery removed lock: %v", err)
	}
	assertSkillVersion(t, root, "release-review", "9.0.0")
}

func TestRecoverRejectsTruncatedJournal(t *testing.T) {
	root := initializedWorkspace(t)
	source := writeSkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	options := testOptions()
	options.FailAt = func(point string) error {
		if point == "after_staging" {
			return errors.New("stop")
		}
		return nil
	}
	if _, err := Execute(preview, preview.PlanID, options); err == nil {
		t.Fatal("Execute() expected injected failure")
	}
	key, err := planKey(preview.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	journal := filepath.Join(root, ".engkit", "journals", "skills", key+".jsonl")
	file, err := os.OpenFile(journal, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("{"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Recover(root, "release-review", preview.PlanID, testOptions()); err == nil ||
		!strings.Contains(err.Error(), "truncated") {
		t.Fatalf("Recover() error = %v", err)
	}
}

func TestExecuteRefusesExistingLock(t *testing.T) {
	root := initializedWorkspace(t)
	source := writeSkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")
	preview, err := skillplan.NewCreate(root, source)
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, ".engkit", "recovery", "skills", "locks", "release-review.lock")
	if err := os.MkdirAll(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(preview, preview.PlanID, testOptions()); err == nil ||
		!strings.Contains(err.Error(), "locked") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func initializedWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	engkit := filepath.Join(root, ".engkit")
	for _, relative := range []string{"plans", "journals", "checkpoints", "context", "cache", "recovery"} {
		if err := os.MkdirAll(filepath.Join(engkit, relative), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(engkit, "config.yaml"), []byte("schema_version: engkit.config/v1alpha1\nworkspace:\n  source_layout: auto\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(engkit, ".gitignore"), []byte("/cache/*\n/context/*.db\n/context/*.db-*\n/recovery/*\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeSkill(t *testing.T, parent, id, version, instructions string) string {
	t.Helper()
	root := filepath.Join(parent, id)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "instructions.md"), []byte(instructions), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(instructions))
	manifest := fmt.Sprintf(`schema_version: engkit.skill/v1alpha1
id: %s
version: %s
description: Reviews releases.
instructions: instructions.md
permissions: {}
compatibility:
  minimum_engkit: 0.2.0
ownership:
  owner: engineering
provenance:
  source: local
  revision: %s
resources:
  - path: instructions.md
    kind: instructions
    sha256: sha256:%s
`, id, version, version, hex.EncodeToString(digest[:]))
	if err := os.WriteFile(filepath.Join(root, "skill.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertSkillVersion(t *testing.T, root, id, version string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, ".agents", "skills", id, "skill.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "version: "+version+"\n") {
		t.Fatalf("skill manifest does not contain version %s", version)
	}
}

func assertBackupVersion(t *testing.T, root, relativePath, version string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath), "skill.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "version: "+version+"\n") {
		t.Fatalf("backup manifest does not contain version %s", version)
	}
}

func assertArtifacts(t *testing.T, root string, preview skillplan.Plan, status string) {
	t.Helper()
	key, err := planKey(preview.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(root, ".engkit", "plans", "skills", key+".json"),
		filepath.Join(root, ".engkit", "journals", "skills", key+".jsonl"),
		filepath.Join(root, ".engkit", "checkpoints", "skills", key+".json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing artifact %s: %v", path, err)
		}
	}
	contents, err := os.ReadFile(filepath.Join(root, ".engkit", "checkpoints", "skills", key+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved checkpoint
	if err := json.Unmarshal(contents, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Status != status {
		t.Fatalf("checkpoint status = %q", saved.Status)
	}
}

func executeWithFailure(t *testing.T, preview skillplan.Plan, failurePoint string) {
	t.Helper()
	options := testOptions()
	options.FailAt = func(point string) error {
		if point == failurePoint {
			return errors.New("stop")
		}
		return nil
	}
	if _, err := Execute(preview, preview.PlanID, options); err == nil {
		t.Fatalf("Execute() expected injected failure at %s", failurePoint)
	}
}

func testOptions() Options {
	return Options{
		Now: func() time.Time {
			return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
		},
		ProcessID: 42,
	}
}
