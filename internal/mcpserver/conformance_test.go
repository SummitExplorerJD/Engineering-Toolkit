package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextview"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/execution"
	sqliteindex "github.com/SummitExplorerJD/Engineering-Toolkit/internal/index/sqlite"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/planinspect"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

func TestWorkspaceToolMatchesApplicationResult(t *testing.T) {
	server := newTestServer(t, nil)
	want, err := workspace.Inspect(server.root)
	if err != nil {
		t.Fatal(err)
	}
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()
	result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "workspace_inspect", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	var got workspace.Inspection
	decodeStructured(t, result.StructuredContent, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MCP result = %#v, application result = %#v", got, want)
	}
}

func TestPlanInspectMatchesApplicationResultAndRejectsChangedPlan(t *testing.T) {
	server := newTestServer(t, nil)
	preview, err := plan.NewWorkspaceInit(workspace.Inspection{Root: server.root})
	if err != nil {
		t.Fatal(err)
	}
	relative := ".engkit/plans/workspace-init.json"
	filename := filepath.Join(filepath.FromSlash(server.root), filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(preview)
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		t.Fatal(err)
	}
	want, err := planinspect.Inspect(server.root, relative)
	if err != nil {
		t.Fatal(err)
	}

	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()
	result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "plan_inspect", Arguments: map[string]any{"path": relative}})
	if err != nil {
		t.Fatal(err)
	}
	var got planinspect.Inspection
	decodeStructured(t, result.StructuredContent, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MCP result = %#v, application result = %#v", got, want)
	}

	preview.Kind = "changed"
	data, _ = json.Marshal(preview)
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		t.Fatal(err)
	}
	failed, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "plan_inspect", Arguments: map[string]any{"path": relative}})
	if err != nil {
		t.Fatal(err)
	}
	if !failed.IsError || failed.StructuredContent != nil {
		t.Fatalf("changed plan result = %#v", failed)
	}
}

func TestContextToolsMatchApplicationResults(t *testing.T) {
	searcher := contextsearch.OptionsSearchFunc(func(_ context.Context, _, query string, options contextsearch.Options) (contextsearch.Response, error) {
		return contextsearch.Response{SchemaVersion: contextsearch.Schema, Query: query, Fresh: true, Results: []contextsearch.Result{{Path: "internal/example.go", Kind: "source", Language: "go", ChunkID: testProjection, Snippet: "needle"}}, AppliedLimit: options.Limit, AppliedMaxBytes: options.MaxContentBytes, AppliedContentBytes: 6}, nil
	})
	server := newTestServer(t, searcher)
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()

	statusResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_status", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	wantStatus, _ := server.contextReader.Status(context.Background(), server.root)
	var gotStatus contextview.Status
	decodeStructured(t, statusResult.StructuredContent, &gotStatus)
	if !reflect.DeepEqual(gotStatus, wantStatus) {
		t.Fatalf("status = %#v, want %#v", gotStatus, wantStatus)
	}

	sourcesResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_sources", Arguments: map[string]any{"path_prefix": "internal", "limit": 10}})
	if err != nil {
		t.Fatal(err)
	}
	var gotSources contextview.Sources
	decodeStructured(t, sourcesResult.StructuredContent, &gotSources)
	if gotSources.SchemaVersion != contextview.SourcesSchema || len(gotSources.Items) != 1 {
		t.Fatalf("sources = %#v", gotSources)
	}

	searchResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_search", Arguments: map[string]any{"query": "needle", "path_prefix": "internal", "kind": "source", "language": "go", "limit": 2, "max_content_bytes": 256}})
	if err != nil {
		t.Fatal(err)
	}
	var gotSearch contextsearch.Response
	decodeStructured(t, searchResult.StructuredContent, &gotSearch)
	if gotSearch.AppliedLimit != 2 || gotSearch.AppliedMaxBytes != 256 || len(gotSearch.Results) != 1 {
		t.Fatalf("search = %#v", gotSearch)
	}

	treeResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "workspace_tree", Arguments: map[string]any{"max_depth": 2, "limit": 10}})
	if err != nil {
		t.Fatal(err)
	}
	var gotTree contextview.TreeResult
	decodeStructured(t, treeResult.StructuredContent, &gotTree)
	if gotTree.SchemaVersion != contextview.TreeSchema || len(gotTree.Entries) != 2 {
		t.Fatalf("tree = %#v", gotTree)
	}

	textStatus, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_status", Arguments: map[string]any{"response_mode": "text"}})
	if err != nil {
		t.Fatal(err)
	}
	var decodedText contextview.Status
	if err := json.Unmarshal([]byte(resultText(t, textStatus)), &decodedText); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decodedText, wantStatus) || textStatus.StructuredContent != nil {
		t.Fatalf("text status = %#v", textStatus)
	}
}

func TestContextToolsUseSyntheticSQLiteProjection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/mcp-context\n\ngo 1.27.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Synthetic\n"), 0o600); err != nil {
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
	if _, err := execution.Initialize(preview, preview.PlanID, time.Now); err != nil {
		t.Fatal(err)
	}
	config := "schema_version: engkit.config/v1alpha1\nworkspace:\n  source_layout: auto\ncontext:\n  source_roots: [.]\n  exclude_paths: [private]\n"
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	for relative, content := range map[string]string{
		"internal/service.go": "package internal\n\nfunc MCPNeedle() string { return \"bounded\" }\n",
		"private/secret.go":   "package private\n\nfunc MCPNeedle() string { return \"excluded\" }\n",
	} {
		filename := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sqliteindex.Rebuild(context.Background(), root, time.Now); err != nil {
		t.Fatal(err)
	}
	server, err := New(root, Dependencies{
		Version:       "v0.4.0-test",
		ContextSearch: contextsearch.OptionsSearchFunc(sqliteindex.SearchWithOptions),
		ContextReader: contextview.ReaderFuncs{StatusFunc: sqliteindex.Inspect, SourcesFunc: sqliteindex.ListSources, ReadFunc: sqliteindex.ReadIndexed, TreeFunc: sqliteindex.Tree},
	})
	if err != nil {
		t.Fatal(err)
	}
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()

	searchResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_search", Arguments: map[string]any{"query": "MCPNeedle", "path_prefix": "internal"}})
	if err != nil || searchResult.IsError {
		t.Fatalf("search result = %#v, err = %v", searchResult, err)
	}
	var search contextsearch.Response
	decodeStructured(t, searchResult.StructuredContent, &search)
	if len(search.Results) != 1 || search.Results[0].Path != "internal/service.go" {
		t.Fatalf("search = %#v", search)
	}

	readResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_read", Arguments: map[string]any{"path": search.Results[0].Path, "chunk": search.Results[0].ChunkID}})
	if err != nil || readResult.IsError {
		t.Fatalf("read result = %#v, err = %v", readResult, err)
	}
	var read contextview.ReadResult
	decodeStructured(t, readResult.StructuredContent, &read)
	if read.Content == "" || read.Path != "internal/service.go" {
		t.Fatalf("read = %#v", read)
	}

	treeResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "workspace_tree", Arguments: map[string]any{"limit": 500, "max_depth": 8}})
	if err != nil || treeResult.IsError {
		t.Fatalf("tree result = %#v, err = %v", treeResult, err)
	}
	var tree contextview.TreeResult
	decodeStructured(t, treeResult.StructuredContent, &tree)
	for _, entry := range tree.Entries {
		if entry.Path == "private" || entry.Path == "private/secret.go" {
			t.Fatalf("tree exposed excluded path: %#v", entry)
		}
	}
}

func decodeStructured(t *testing.T, value, destination any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		t.Fatal(err)
	}
}
