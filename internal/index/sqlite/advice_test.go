package sqliteindex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAdviceExcerptsRequireFreshIndexAndSelectAllowedDocumentation(t *testing.T) {
	root := initializedWorkspace(t)
	if err := os.MkdirAll(filepath.Join(root, ".agents", "context"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "context", "private.md"), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(context.Background(), root, time.Now); err != nil {
		t.Fatal(err)
	}
	excerpts, err := AdviceExcerpts(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(excerpts) != 1 || excerpts[0].Path != "README.md" {
		t.Fatalf("unexpected excerpts: %#v", excerpts)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AdviceExcerpts(context.Background(), root); err == nil {
		t.Fatal("stale index was accepted")
	}
}
