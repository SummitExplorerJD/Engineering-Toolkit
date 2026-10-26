package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
