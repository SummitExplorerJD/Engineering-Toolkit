package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRoot(t *testing.T) {
	root := t.TempDir()
	got, err := ValidateRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(root)
	if got != filepath.ToSlash(want) {
		t.Fatalf("ValidateRoot() = %q, want %q", got, filepath.ToSlash(want))
	}

	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateRoot(file); err == nil {
		t.Fatal("ValidateRoot(file) succeeded")
	}
}

func TestValidateRootRejectsLinkedRoot(t *testing.T) {
	real := t.TempDir()
	linked := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(real, linked); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := ValidateRoot(linked); err == nil {
		t.Fatal("ValidateRoot(link) succeeded")
	}
}
