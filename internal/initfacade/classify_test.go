package initfacade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifySelectsOneLifecycle(t *testing.T) {
	parent := t.TempDir()
	absent := filepath.Join(parent, "absent")
	empty := filepath.Join(parent, "empty")
	if err := os.Mkdir(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	nonempty := filepath.Join(parent, "nonempty")
	if err := os.Mkdir(nonempty, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonempty, "README.md"), []byte("# Existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	initialized := filepath.Join(parent, "initialized")
	writeInitialized(t, initialized, "schema_version: engkit.config/v1alpha1\nworkspace:\n  source_layout: auto\n")

	tests := []struct {
		name        string
		target      string
		profile     bool
		lifecycle   Lifecycle
		wantErrPart string
	}{
		{name: "absent profile", target: absent, profile: true, lifecycle: ProjectCreate},
		{name: "absent no profile", target: absent, wantErrPart: "requires an exact --profile"},
		{name: "empty profile", target: empty, profile: true, lifecycle: ProjectCreate},
		{name: "empty no profile", target: empty, lifecycle: WorkspaceInitialize},
		{name: "nonempty no profile", target: nonempty, lifecycle: WorkspaceInitialize},
		{name: "nonempty profile", target: nonempty, profile: true, wantErrPart: "separate workspace initialization"},
		{name: "initialized profile", target: initialized, profile: true, lifecycle: WorkspaceCompose},
		{name: "initialized no profile", target: initialized, wantErrPart: "already initialized"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Classify(test.target, test.profile)
			if test.wantErrPart != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
					t.Fatalf("Classify() error = %v, want containing %q", err, test.wantErrPart)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Lifecycle != test.lifecycle || got.Target == "" {
				t.Fatalf("Classify() = %#v", got)
			}
		})
	}
}

func TestClassifyRejectsPartialInvalidAndFileTargets(t *testing.T) {
	parent := t.TempDir()
	partial := filepath.Join(parent, "partial")
	if err := os.MkdirAll(filepath.Join(partial, ".engkit"), 0o700); err != nil {
		t.Fatal(err)
	}
	invalid := filepath.Join(parent, "invalid")
	writeInitialized(t, invalid, "schema_version: unsupported\n")
	file := filepath.Join(parent, "file")
	if err := os.WriteFile(file, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{partial, invalid, file} {
		if got, err := Classify(target, true); err == nil {
			t.Fatalf("Classify(%q) = %#v, want error", target, got)
		}
	}
}

func TestClassifyRejectsLinkedTarget(t *testing.T) {
	parent := t.TempDir()
	real := filepath.Join(parent, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(parent, "linked")
	if err := os.Symlink(real, linked); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if got, err := Classify(linked, true); err == nil {
		t.Fatalf("Classify() = %#v, want linked-target error", got)
	}
}

func writeInitialized(t *testing.T, root, config string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".engkit"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".engkit", ".gitignore"), []byte("/cache/*\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
