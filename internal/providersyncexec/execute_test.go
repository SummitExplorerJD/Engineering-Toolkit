package providersyncexec

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
)

func TestExecuteSynchronizesAndUnsynchronizesExactOwnedOutputs(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")

	prepared, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Execute(prepared.Plan, prepared.Plan.PlanID, Options{ProcessID: 101, Link: testNoReplace})
	if err != nil {
		if errors.Is(err, ErrNoReplaceUnsupported) {
			t.Skipf("test filesystem does not support hard-link publication: %v", err)
		}
		t.Fatal(err)
	}
	if result.Status != "complete" || result.SkillID != "project-guide" {
		t.Fatalf("sync result = %#v", result)
	}
	for _, file := range prepared.Files {
		assertDigest(t, filepath.Join(root, filepath.FromSlash(file.Path)), file.SHA256)
	}
	ownership := filepath.Join(root, ".engkit", "providers", "claude", "project-guide.json")
	if _, err := os.Lstat(ownership); err != nil {
		t.Fatalf("ownership was not published: %v", err)
	}

	unsync, err := providersync.NewUnsyncPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	result, err = Execute(unsync, unsync.PlanID, Options{ProcessID: 102, Link: testNoReplace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" || result.Operation != "remove" {
		t.Fatalf("unsync result = %#v", result)
	}
	for _, file := range prepared.Files {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file.Path))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unsynchronized output %s remains: %v", file.Path, err)
		}
	}
	if _, err := os.Lstat(ownership); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ownership remains after unsync: %v", err)
	}
}

func TestExecuteSynchronizesUnchangedReplaceAndStaleRemoval(t *testing.T) {
	root := testWorkspace(t)
	writeCanonicalWithReference(t, root, "project-guide", "# Project guide\n", "# Reference\n")
	first, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(first.Plan, first.Plan.PlanID, Options{Link: testNoReplace}); err != nil {
		t.Fatal(err)
	}

	unchanged, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range unchanged.Plan.Outputs {
		if output.Action != "unchanged" {
			t.Fatalf("second sync output = %#v, want unchanged", output)
		}
	}
	if _, err := Execute(unchanged.Plan, unchanged.Plan.PlanID, Options{Link: testNoReplace}); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(root, ".agents", "skills", "project-guide", "references", "guide.md")); err != nil {
		t.Fatal(err)
	}
	writeCanonical(t, root, "project-guide", "# Updated project guide\n")
	updated, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]string{}
	for _, output := range updated.Plan.Outputs {
		actions[output.Path] = output.Action
	}
	if actions[".claude/skills/project-guide/SKILL.md"] != "replace" || actions[".claude/skills/project-guide/references/guide.md"] != "remove" {
		t.Fatalf("update actions = %#v", actions)
	}
	if _, err := Execute(updated.Plan, updated.Plan.PlanID, Options{Link: testNoReplace}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude", "skills", "project-guide", "references", "guide.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale derived resource remains: %v", err)
	}
}

func TestExecuteRejectsApprovalMismatchWithoutProviderWrites(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "opencode", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(prepared.Plan, "sha256:"+strings.Repeat("0", 64), Options{})
	if err == nil {
		t.Fatal("approval mismatch unexpectedly succeeded")
	}
	if _, err := os.Lstat(filepath.Join(root, ".opencode")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("approval mismatch wrote provider output: %v", err)
	}
}

func TestExecuteFailsClosedWhenNoReplaceIsUnavailable(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(prepared.Plan, prepared.Plan.PlanID, Options{Link: func(_, _ string) error { return errors.New("unsupported") }})
	if !errors.Is(err, ErrNoReplaceUnsupported) {
		t.Fatalf("Execute() error = %v, want ErrNoReplaceUnsupported", err)
	}
}

func TestRecoverRollsBackInterruptedSynchronization(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(prepared.Plan, prepared.Plan.PlanID, failAt("after_staging"))
	if errors.Is(err, ErrNoReplaceUnsupported) {
		t.Skipf("test filesystem does not support hard-link publication: %v", err)
	}
	if err == nil {
		t.Fatal("injected synchronization failure did not fail")
	}
	if !strings.Contains(err.Error(), "injected failure at after_staging") {
		t.Fatalf("synchronization failed before injection point: %v", err)
	}
	result, err := Recover(root, "claude", "project-guide", prepared.Plan.PlanID, Options{Link: testNoReplace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rolled_back" {
		t.Fatalf("recovery result = %#v", result)
	}
	for _, file := range prepared.Files {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file.Path))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("rolled-back output %s remains: %v", file.Path, err)
		}
	}
}

func TestRecoverCompletesSynchronizationPendingOwnership(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "opencode", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(prepared.Plan, prepared.Plan.PlanID, failAt("after_output_0"))
	if errors.Is(err, ErrNoReplaceUnsupported) {
		t.Skipf("test filesystem does not support hard-link publication: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "injected failure at after_output_0") {
		t.Fatalf("synchronization did not stop at ownership boundary: %v", err)
	}
	result, err := Recover(root, "opencode", "project-guide", prepared.Plan.PlanID, Options{Link: testNoReplace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("recovery result = %#v", result)
	}
}

func TestRecoverCompletesPublishedSynchronization(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "opencode", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(prepared.Plan, prepared.Plan.PlanID, failAt("after_ownership_publish"))
	if errors.Is(err, ErrNoReplaceUnsupported) {
		t.Skipf("test filesystem does not support hard-link publication: %v", err)
	}
	if err == nil {
		t.Fatal("injected synchronization failure did not fail")
	}
	if !strings.Contains(err.Error(), "injected failure at after_ownership_publish") {
		t.Fatalf("synchronization failed before injection point: %v", err)
	}
	key, err := planKey(prepared.Plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	tx := recoveryTransaction(root, "opencode", "project-guide", key, Options{})
	if err := strictJSON(tx.checkpointPath, &tx.checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := verifyOutputs(tx, true); err != nil {
		t.Fatalf("published outputs do not match checkpoint: %v", err)
	}
	if err := requireDigest(tx.ownership, tx.checkpoint.OwnershipAfterHash); err != nil {
		t.Fatalf("published ownership does not match checkpoint: %v", err)
	}
	result, err := Recover(root, "opencode", "project-guide", prepared.Plan.PlanID, Options{Link: testNoReplace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("recovery result = %#v", result)
	}
}

func TestRecoverRollsBackInterruptedUnsynchronization(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(prepared.Plan, prepared.Plan.PlanID, Options{Link: testNoReplace}); err != nil {
		if errors.Is(err, ErrNoReplaceUnsupported) {
			t.Skipf("test filesystem does not support hard-link publication: %v", err)
		}
		t.Fatal(err)
	}
	unsync, err := providersync.NewUnsyncPlan(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(unsync, unsync.PlanID, failAt("after_output_0"))
	if errors.Is(err, ErrNoReplaceUnsupported) {
		t.Skipf("test filesystem does not support hard-link publication: %v", err)
	}
	if err == nil {
		t.Fatal("injected unsynchronization failure did not fail")
	}
	if !strings.Contains(err.Error(), "injected failure at after_output_0") {
		t.Fatalf("unsynchronization failed before injection point: %v", err)
	}
	result, err := Recover(root, "claude", "project-guide", unsync.PlanID, Options{Link: testNoReplace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rolled_back" {
		t.Fatalf("recovery result = %#v", result)
	}
	for _, file := range prepared.Files {
		assertDigest(t, filepath.Join(root, filepath.FromSlash(file.Path)), file.SHA256)
	}
	if _, err := os.Lstat(filepath.Join(root, ".engkit", "providers", "claude", "project-guide.json")); err != nil {
		t.Fatalf("ownership was not restored: %v", err)
	}
}

func TestRecoverRefusesChangedStagingAndRetainsLocks(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(prepared.Plan, prepared.Plan.PlanID, failAt("after_staging"))
	if err == nil || !strings.Contains(err.Error(), "injected failure at after_staging") {
		t.Fatalf("synchronization did not stop after staging: %v", err)
	}
	key, err := planKey(prepared.Plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	tx := recoveryTransaction(root, "claude", "project-guide", key, Options{Link: testNoReplace})
	if err := strictJSON(tx.checkpointPath, &tx.checkpoint); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(root, filepath.FromSlash(tx.checkpoint.Outputs[0].StagingPath))
	if err := os.WriteFile(staged, []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Recover(root, "claude", "project-guide", prepared.Plan.PlanID, Options{Link: testNoReplace}); err == nil || !strings.Contains(err.Error(), "recovery refused") {
		t.Fatalf("Recover() error = %v, want refusal", err)
	}
	for _, lock := range []string{tx.skillLock, tx.providerLock} {
		if info, err := os.Lstat(lock); err != nil || !info.IsDir() {
			t.Fatalf("recovery refusal did not retain lock %s: %v", lock, err)
		}
	}
}

func TestValidatePersistedPlanRejectsTampering(t *testing.T) {
	root := testWorkspace(t)
	writeCanonical(t, root, "project-guide", "# Project guide\n")
	prepared, err := providersync.Prepare(root, "claude", "project-guide")
	if err != nil {
		t.Fatal(err)
	}
	prepared.Plan.Risks = append(prepared.Plan.Risks, "tampered")
	if err := validatePersistedPlan(prepared.Plan); err == nil {
		t.Fatal("tampered persisted plan unexpectedly validated")
	}
}

func failAt(point string) Options {
	return Options{ProcessID: 103, Link: testNoReplace, FailAt: func(candidate string) error {
		if candidate == point {
			return errors.New("test interruption")
		}
		return nil
	}}
}

func testNoReplace(oldname, newname string) error {
	data, err := os.ReadFile(oldname)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(newname, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	failed := true
	defer func() {
		_ = file.Close()
		if failed {
			_ = os.Remove(newname)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	failed = false
	return nil
}

func testWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".engkit"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte("schema_version: engkit.config/v1alpha1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeCanonical(t *testing.T, root, id, instructions string) {
	t.Helper()
	packageRoot := filepath.Join(root, ".agents", "skills", id)
	if err := os.MkdirAll(packageRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "instructions.md"), []byte(instructions), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(instructions))
	manifest := []byte("schema_version: engkit.skill/v1alpha1\nid: " + id + "\nversion: 0.1.0\ndescription: Safe project guidance.\ninstructions: instructions.md\npermissions: {}\ncompatibility:\n  minimum_engkit: 0.2.1\nownership:\n  owner: tests\nprovenance:\n  source: test\n  revision: fixture\nresources:\n  - path: instructions.md\n    kind: instructions\n    sha256: sha256:" + hex.EncodeToString(digest[:]) + "\n")
	if err := os.WriteFile(filepath.Join(packageRoot, "skill.yaml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeCanonicalWithReference(t *testing.T, root, id, instructions, reference string) {
	t.Helper()
	packageRoot := filepath.Join(root, ".agents", "skills", id)
	if err := os.MkdirAll(filepath.Join(packageRoot, "references"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "instructions.md"), []byte(instructions), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "references", "guide.md"), []byte(reference), 0o600); err != nil {
		t.Fatal(err)
	}
	instructionsDigest := sha256.Sum256([]byte(instructions))
	referenceDigest := sha256.Sum256([]byte(reference))
	manifest := []byte("schema_version: engkit.skill/v1alpha1\nid: " + id + "\nversion: 0.1.0\ndescription: Safe project guidance.\ninstructions: instructions.md\npermissions: {}\ncompatibility:\n  minimum_engkit: 0.2.1\nownership:\n  owner: tests\nprovenance:\n  source: test\n  revision: fixture\nresources:\n  - path: instructions.md\n    kind: instructions\n    sha256: sha256:" + hex.EncodeToString(instructionsDigest[:]) + "\n  - path: references/guide.md\n    kind: reference\n    sha256: sha256:" + hex.EncodeToString(referenceDigest[:]) + "\n")
	if err := os.WriteFile(filepath.Join(packageRoot, "skill.yaml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertDigest(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if got := "sha256:" + hex.EncodeToString(digest[:]); got != expected {
		t.Fatalf("digest for %s = %s, want %s", path, got, expected)
	}
}
