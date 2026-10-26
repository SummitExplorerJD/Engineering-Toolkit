package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspectInitializedWorkspaceIsReadOnlyAndDeterministic(t *testing.T) {
	root := initializedWorkspace(t)
	before := treeSnapshot(t, root)
	options := Options{
		Root: root, Scope: ScopeAll, Version: "v0.3.0-test",
		LookPath: func(name string) (string, error) { return filepath.Join("tools", name), nil },
	}
	first, err := Inspect(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Inspect(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("doctor result changed between reads\nfirst: %#v\nsecond: %#v", first, second)
	}
	if !first.Summary.Healthy {
		t.Fatalf("healthy workspace result = %#v", first)
	}
	if got := treeSnapshot(t, root); !reflect.DeepEqual(before, got) {
		t.Fatalf("doctor mutated workspace\nbefore: %#v\nafter: %#v", before, got)
	}
	assertFinding(t, first, "workspace.context", StatusUnavailable)
	assertFinding(t, first, "integration.ollama.executable", StatusPass)
}

func TestInspectReportsInvalidConfiguration(t *testing.T) {
	root := initializedWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte("schema_version: engkit.config/v1alpha1\nunknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Inspect(context.Background(), Options{Root: root, Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Healthy || result.Summary.RequiredErrors != 1 {
		t.Fatalf("summary = %#v", result.Summary)
	}
	assertFinding(t, result, "workspace.config", StatusError)
}

func TestInspectReportsTransactionLock(t *testing.T) {
	root := initializedWorkspace(t)
	lock := filepath.Join(root, ".engkit", "recovery", "skills", "locks", "project-guide.lock")
	if err := os.MkdirAll(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := Inspect(context.Background(), Options{Root: root, Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Healthy {
		t.Fatal("active lock did not make required checks unhealthy")
	}
	assertFinding(t, result, "workspace.transactions", StatusError)
}

func TestInspectReportsInvalidProviderOwnership(t *testing.T) {
	root := initializedWorkspace(t)
	directory := filepath.Join(root, ".engkit", "providers", "claude")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "project-guide.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Inspect(context.Background(), Options{
		Root: root, Scope: ScopeIntegrations,
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Healthy {
		t.Fatal("invalid ownership record did not make required checks unhealthy")
	}
	assertFinding(t, result, "integration.provider-ownership.invalid", StatusError)
}

func TestInspectReportsMissingRootAsStructuredFinding(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	result, err := Inspect(context.Background(), Options{Root: root, Scope: ScopeAll, LookPath: func(string) (string, error) { return "", errors.New("missing") }})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Healthy {
		t.Fatal("missing root reported healthy")
	}
	assertFinding(t, result, "workspace.root", StatusError)
	assertFinding(t, result, "integration.ollama.executable", StatusUnavailable)
}

func TestInspectRejectsUnknownScope(t *testing.T) {
	if _, err := Inspect(context.Background(), Options{Scope: "network"}); err == nil {
		t.Fatal("unknown scope was accepted")
	}
}

func initializedWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, relative := range []string{".engkit/context", ".engkit/recovery", ".agents/skills"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(relative)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	config := "schema_version: engkit.config/v1alpha1\nworkspace:\n  source_layout: auto\n"
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertFinding(t *testing.T, result Result, code, status string) {
	t.Helper()
	for _, finding := range result.Findings {
		if finding.Code == code {
			if finding.Status != status {
				t.Fatalf("finding %s status = %s, want %s", code, finding.Status, status)
			}
			return
		}
	}
	t.Fatalf("finding %s was not returned: %#v", code, result.Findings)
}

func treeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return paths
}
