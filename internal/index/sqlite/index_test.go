package sqliteindex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextview"
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
	planDir := filepath.Join(root, ".engkit", "plans", "test")
	if err := os.MkdirAll(planDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planDir, "sensitive.json"), []byte(`{"token":"`+secret+`"}`), 0o600); err != nil {
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

func TestConfiguredSourceSearchInventoryAndRead(t *testing.T) {
	root := initializedWorkspace(t)
	config := "schema_version: engkit.config/v1alpha1\ncontext:\n  source_roots: ['.']\n  exclude_paths: [private]\n"
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	for relative, content := range map[string]string{
		"internal/service.go":  "package internal\n\nfunc TokenEfficientNeedle() string { return \"bounded\" }\n",
		"private/hidden.go":    "package private\nfunc TokenEfficientNeedle() {}\n",
		"vendor/dependency.go": "package dependency\nfunc TokenEfficientNeedle() {}\n",
		".env":                 "TokenEfficientNeedle=secret\n",
	} {
		filename := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := Rebuild(context.Background(), root, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChunkCount == 0 || result.ExcludedCount < 3 {
		t.Fatalf("unexpected rebuild result: %#v", result)
	}
	search, err := Search(context.Background(), root, "TokenEfficientNeedle")
	if err != nil {
		t.Fatal(err)
	}
	if len(search.Results) != 1 || search.Results[0].Path != "internal/service.go" || search.Results[0].ChunkID == "" {
		t.Fatalf("unexpected search: %#v", search)
	}
	sources, err := ListSources(context.Background(), root, contextview.SourceOptions{PathPrefix: "internal", Language: "go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sources.Items) != 1 || sources.Items[0].Path != "internal/service.go" {
		t.Fatalf("unexpected sources: %#v", sources)
	}
	read, err := ReadIndexed(context.Background(), root, contextview.ReadRequest{Path: "internal/service.go", ChunkID: search.Results[0].ChunkID})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read.Content, "TokenEfficientNeedle") || read.AppliedBytes > contextview.DefaultRead {
		t.Fatalf("unexpected read: %#v", read)
	}
	tree, err := Tree(context.Background(), root, contextview.TreeOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Entries) != 1 || !tree.Truncated || tree.NextAfter == "" {
		t.Fatalf("unexpected first tree page: %#v", tree)
	}
	next, err := Tree(context.Background(), root, contextview.TreeOptions{Limit: 10, After: tree.NextAfter, ProjectionFingerprint: tree.ProjectionFingerprint})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Entries) == 0 {
		t.Fatalf("empty second tree page: %#v", next)
	}
	for _, entry := range append(tree.Entries, next.Entries...) {
		if strings.HasPrefix(entry.Path, "private") || strings.HasPrefix(entry.Path, "vendor") || entry.Path == ".env" {
			t.Fatalf("tree exposed excluded path: %#v", entry)
		}
	}
	if _, err := Tree(context.Background(), root, contextview.TreeOptions{After: tree.NextAfter, ProjectionFingerprint: "sha256:" + strings.Repeat("0", 64)}); err == nil {
		t.Fatal("tree accepted stale projection continuation")
	}
}

func TestSourceChunksAreBoundedAndStable(t *testing.T) {
	source := sourceRecord{Path: "internal/large.go", SHA256: "sha256:source"}
	text := strings.Repeat("é", maxChunkBytes) + "\n" + strings.Repeat("line\n", maxChunkLines+1)
	first := splitSourceChunks(source, text)
	second := splitSourceChunks(source, text)
	if len(first) < 3 || len(first) != len(second) {
		t.Fatalf("unexpected chunks: %d and %d", len(first), len(second))
	}
	for index := range first {
		if len(first[index].Content) > maxChunkBytes || first[index].ID != second[index].ID {
			t.Fatalf("chunk %d is not bounded and stable: %#v", index, first[index])
		}
	}
}

func TestSearchUsesBoundedContentBudget(t *testing.T) {
	root := initializedWorkspace(t)
	config := "schema_version: engkit.config/v1alpha1\ncontext:\n  source_roots: [src]\n"
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	baseline := strings.Repeat("irrelevant context line\n", 900) + "func BoundedBudgetNeedle() {}\n"
	if err := os.WriteFile(filepath.Join(root, "src", "large.go"), []byte(baseline), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(context.Background(), root, time.Now); err != nil {
		t.Fatal(err)
	}
	result, err := SearchWithOptions(context.Background(), root, "BoundedBudgetNeedle", contextsearch.Options{MaxContentBytes: 512})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 || result.AppliedContentBytes > 512 {
		t.Fatalf("unexpected bounded result: %#v", result)
	}
	if result.AppliedContentBytes*10 >= len(baseline) {
		t.Fatalf("staged retrieval did not reduce content sufficiently: result=%d baseline=%d", result.AppliedContentBytes, len(baseline))
	}
	t.Logf("staged retrieval bytes=%d baseline bytes=%d", result.AppliedContentBytes, len(baseline))
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
