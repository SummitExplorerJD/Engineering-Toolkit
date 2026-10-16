package sqliteindex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/execution"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

func TestRebuildSearchFreshnessAndRecovery(t *testing.T) {
	root := initializedWorkspace(t)
	ctx := context.Background()
	fixedTime := time.Date(2026, 10, 15, 14, 0, 0, 0, time.UTC)

	result, err := Rebuild(ctx, root, func() time.Time { return fixedTime })
	if err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	if result.ArtifactCount == 0 || result.RebuiltAt != fixedTime.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected rebuild result: %#v", result)
	}

	status, err := Inspect(ctx, root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if !status.Exists || !status.Fresh || status.ArtifactCount != result.ArtifactCount {
		t.Fatalf("unexpected status: %#v", status)
	}

	search, err := Search(ctx, root, "durable workspace")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(search.Results) != 1 || search.Results[0].Path != "README.md" {
		t.Fatalf("unexpected search results: %#v", search.Results)
	}
	if !search.Fresh {
		t.Fatal("newly rebuilt index must report fresh search results")
	}

	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("# Project\n\nDurable workspace changed.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err = Inspect(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if status.Fresh {
		t.Fatal("modified canonical source must make the index stale")
	}

	if _, err := Rebuild(ctx, root, time.Now); err != nil {
		t.Fatalf("second Rebuild() error = %v", err)
	}
	indexPath := filepath.Join(root, ".engkit", "context", databaseName)
	if err := os.Remove(indexPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(readme); err != nil {
		t.Fatalf("deleting derived index damaged canonical source: %v", err)
	}
	status, err = Inspect(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if status.Exists || status.Fresh {
		t.Fatalf("deleted index reported as available: %#v", status)
	}
	if _, err := Rebuild(ctx, root, time.Now); err != nil {
		t.Fatalf("recovery Rebuild() error = %v", err)
	}
}

func TestRebuildDoesNotIndexSensitiveArtifactContent(t *testing.T) {
	root := initializedWorkspace(t)
	secret := "secret-token-value"
	configPath := filepath.Join(root, ".engkit", "config.yaml")
	if err := os.WriteFile(configPath, []byte("schema_version: engkit.config/v1alpha1\ntoken: "+secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(context.Background(), root, time.Now); err != nil {
		t.Fatal(err)
	}
	results, err := Search(context.Background(), root, secret)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Results) != 0 {
		t.Fatalf("sensitive configuration content was searchable: %#v", results.Results)
	}
}

func initializedWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":    "module example.com/context-test\n\ngo 1.26.4\n",
		"README.md": "# Project\n\nA durable workspace.\n",
	}
	for relative, content := range files {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := workspace.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.NewWorkspaceInit(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := execution.Initialize(preview, preview.PlanID, time.Now); err != nil {
		t.Fatal(err)
	}
	return root
}
