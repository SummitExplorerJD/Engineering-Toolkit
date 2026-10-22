package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogHasSelectiveProviders(t *testing.T) {
	got := List()
	if got.SchemaVersion != CatalogSchema || len(got.Adapters) != 3 {
		t.Fatalf("unexpected catalog: %#v", got)
	}
	want := []string{"claude", "codex", "opencode"}
	for i, id := range want {
		if got.Adapters[i].ID != id {
			t.Fatalf("adapter %d = %q, want %q", i, got.Adapters[i].ID, id)
		}
	}
}

func TestInspectClassifiesConfiguredPathsWithoutWriting(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Instructions\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(root, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Configured {
		t.Fatal("expected existing AGENTS.md to configure the adapter")
	}
	if got.Paths[0].Path != ".agents/skills" || got.Paths[0].State != "missing" {
		t.Fatalf("unexpected first path state: %#v", got.Paths[0])
	}
	if got.Paths[1].Path != "AGENTS.md" || got.Paths[1].State != "file" {
		t.Fatalf("unexpected second path state: %#v", got.Paths[1])
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("inspection wrote to the workspace: %v", err)
	}
}

func TestGetRejectsUnsupportedProvider(t *testing.T) {
	if _, err := Get("unknown"); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestOpenCodeCatalogDeclaresIsolatedProjectSkills(t *testing.T) {
	adapter, err := Get("opencode")
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range adapter.Capabilities {
		if capability.Name == "project_skills" {
			if capability.Support != SupportNative || capability.Path != ".opencode/skills" {
				t.Fatalf("unexpected OpenCode project skills capability: %#v", capability)
			}
			return
		}
	}
	t.Fatal("OpenCode project_skills capability is missing")
}
