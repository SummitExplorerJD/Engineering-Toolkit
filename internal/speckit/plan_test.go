package speckit

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewPlanIsDeterministicAndNonExecutable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Owned\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := NewPlan(root, "codex", "v0.12.11")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewPlan(root, "codex", "v0.12.11")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("plans differ:\n%#v\n%#v", first, second)
	}
	if first.Executable || !first.RequiresApproval {
		t.Fatalf("unsafe plan flags: %#v", first)
	}
	if first.Integration != "codex" || first.UpstreamVersion != "v0.12.11" {
		t.Fatalf("unexpected selection: %#v", first)
	}
	if first.Command[0] != "uvx" || first.Command[1] != "--python" || first.Command[2] != "3.11" || first.Command[len(first.Command)-1] != "ps" {
		t.Fatalf("unexpected command: %#v", first.Command)
	}
	foundIgnoreAgentTools := false
	for _, argument := range first.Command {
		if argument == "--ignore-agent-tools" {
			foundIgnoreAgentTools = true
		}
	}
	if !foundIgnoreAgentTools {
		t.Fatalf("command does not isolate provider CLI availability: %#v", first.Command)
	}
	if content, err := os.ReadFile(filepath.Join(root, "AGENTS.md")); err != nil || string(content) != "# Owned\n" {
		t.Fatalf("planning changed AGENTS.md: content=%q err=%v", content, err)
	}
}

func TestNewPlanRequiresPinnedRelease(t *testing.T) {
	for _, version := range []string{"", "main", "0.12.11", "v1"} {
		if _, err := NewPlan(t.TempDir(), "claude", version); err == nil {
			t.Fatalf("NewPlan accepted version %q", version)
		}
	}
}

func TestNewPlanReportsExistingRootsForReview(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".specify"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := NewPlan(root, "opencode", "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, impact := range got.PotentialPaths {
		if impact.Path == ".specify" {
			found = impact.State == "directory" && impact.Disposition == "review_exact_diff_before_execution"
		}
	}
	if !found {
		t.Fatalf("existing .specify was not reported for review: %#v", got.PotentialPaths)
	}
}
