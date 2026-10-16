package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

func TestVersionJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}

	var got map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["schema_version"] != versionSchema || got["version"] != Version {
		t.Fatalf("unexpected version output: %#v", got)
	}
}

func TestWorkspaceInspectJSON(t *testing.T) {
	root := testWorkspace(t)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"workspace", "inspect", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}

	var got workspace.Inspection
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.GoModule != "example.com/project" {
		t.Fatalf("GoModule = %q", got.GoModule)
	}
}

func TestWorkspacePlanInitDoesNotWrite(t *testing.T) {
	root := testWorkspace(t)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"workspace", "plan-init", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}

	var got plan.Plan
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != "workspace.init" || !got.RequiresApproval {
		t.Fatalf("unexpected plan: %#v", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit")); !os.IsNotExist(err) {
		t.Fatalf("plan-init created .engkit or returned an unexpected error: %v", err)
	}
}

func TestWorkspaceInitRequiresAndUsesApprovedPlan(t *testing.T) {
	root := testWorkspace(t)
	var planOut, planErr bytes.Buffer
	code := Run([]string{"workspace", "plan-init", "--root", root}, &planOut, &planErr)
	if code != 0 {
		t.Fatalf("plan-init code = %d, stderr = %q", code, planErr.String())
	}
	var preview plan.Plan
	if err := json.Unmarshal(planOut.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code = Run([]string{"workspace", "init", "--root", root, "--approve", preview.PlanID}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init code = %d, stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit", "config.yaml")); err != nil {
		t.Fatalf("expected initialized config: %v", err)
	}
}

func TestContextLifecycle(t *testing.T) {
	root := testWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Project\n\nDurable context discovery.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	initializeWithCLI(t, root)

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"context", "rebuild", "--root", root}, want: `"artifact_count"`},
		{args: []string{"context", "status", "--root", root}, want: `"fresh": true`},
		{args: []string{"context", "search", "--root", root, "context discovery"}, want: `"README.md"`},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("Run(%v) output = %q, want %q", test.args, stdout.String(), test.want)
		}
	}
}

func TestAdapterAndSpecKitCommandsAreReadOnly(t *testing.T) {
	root := testWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Rules\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"adapters", "list"}, want: `"id": "codex"`},
		{args: []string{"adapters", "inspect", "--root", root, "--provider", "codex"}, want: `"configured": true`},
		{args: []string{"speckit", "plan", "--root", root, "--integration", "codex", "--version", "v0.12.11"}, want: `"executable": false`},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("Run(%v) output = %q, want %q", test.args, stdout.String(), test.want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".specify")); !os.IsNotExist(err) {
		t.Fatalf("read-only commands created .specify or returned an unexpected error: %v", err)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func testWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	content := []byte("module example.com/project\n\ngo 1.26.4\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func initializeWithCLI(t *testing.T, root string) {
	t.Helper()
	var planOut, planErr bytes.Buffer
	if code := Run([]string{"workspace", "plan-init", "--root", root}, &planOut, &planErr); code != 0 {
		t.Fatalf("plan-init code = %d, stderr = %q", code, planErr.String())
	}
	var preview plan.Plan
	if err := json.Unmarshal(planOut.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	var initOut, initErr bytes.Buffer
	if code := Run([]string{"workspace", "init", "--root", root, "--approve", preview.PlanID}, &initOut, &initErr); code != 0 {
		t.Fatalf("workspace init code = %d, stderr = %q", code, initErr.String())
	}
}
