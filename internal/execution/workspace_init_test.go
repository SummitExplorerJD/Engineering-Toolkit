package execution

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

func TestInitializeRequiresMatchingApprovalWithoutWriting(t *testing.T) {
	root, preview := previewForTest(t)
	if _, err := Initialize(preview, "sha256:wrong", time.Now); err == nil {
		t.Fatal("Initialize() expected approval error")
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit")); !os.IsNotExist(err) {
		t.Fatalf("unexpected .engkit state: %v", err)
	}
}

func TestInitializePublishesCanonicalArtifacts(t *testing.T) {
	root, preview := previewForTest(t)
	fixedTime := time.Date(2026, 10, 15, 12, 30, 0, 0, time.UTC)

	result, err := Initialize(preview, preview.PlanID, func() time.Time { return fixedTime })
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if result.Status != "complete" || result.PlanID != preview.PlanID {
		t.Fatalf("unexpected result: %#v", result)
	}

	for _, relative := range []string{
		".gitignore",
		"config.yaml",
		"plans/workspace-init.json",
		"journals/workspace-init.jsonl",
		"checkpoints/workspace-init.json",
		"context",
		"cache",
	} {
		if _, err := os.Stat(filepath.Join(root, ".engkit", filepath.FromSlash(relative))); err != nil {
			t.Errorf("expected %s: %v", relative, err)
		}
	}

	assertJournal(t, filepath.Join(root, ".engkit", "journals", "workspace-init.jsonl"), preview.PlanID)
	if _, err := Initialize(preview, preview.PlanID, time.Now); err == nil {
		t.Fatal("second Initialize() expected already-initialized error")
	}
}

func previewForTest(t *testing.T) (string, plan.Plan) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := workspace.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.NewWorkspaceInit(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return root, preview
}

func assertJournal(t *testing.T, path, planID string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	sequence := 0
	for scanner.Scan() {
		sequence++
		var event journalEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("journal line %d: %v", sequence, err)
		}
		if event.PlanID != planID || event.Sequence != sequence {
			t.Fatalf("unexpected event: %#v", event)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if sequence != 3 {
		t.Fatalf("journal event count = %d", sequence)
	}
}
