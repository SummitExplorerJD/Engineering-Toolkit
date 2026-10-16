package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectReportsWorkspaceFacts(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/tool\n\ngo 1.26.4\n")
	mustMkdir(t, filepath.Join(root, ".git"))
	mustMkdir(t, filepath.Join(root, ".agents"))

	got, err := Inspect(root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if got.SchemaVersion != inspectionSchema {
		t.Fatalf("SchemaVersion = %q", got.SchemaVersion)
	}
	if got.GoModule != "example.com/tool" {
		t.Fatalf("GoModule = %q", got.GoModule)
	}
	if !got.HasGit || !got.HasAgents {
		t.Fatalf("expected Git and agents detection: %#v", got)
	}
	if got.HasEngkit || got.HasSpecKit {
		t.Fatalf("unexpected managed state detection: %#v", got)
	}
}

func TestInspectRejectsFileRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "file")
	mustWrite(t, root, "not a directory")

	if _, err := Inspect(root); err == nil {
		t.Fatal("Inspect() expected an error")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
}
