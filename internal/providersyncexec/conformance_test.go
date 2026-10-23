package providersyncexec

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
)

func TestPilotLifecyclePreservesCanonicalSpecKitAndProviderNativeFiles(t *testing.T) {
	providerRoots := map[string]string{
		"codex":    ".agents/skills",
		"claude":   ".claude/skills",
		"opencode": ".opencode/skills",
	}
	for provider, providerRoot := range providerRoots {
		t.Run(provider, func(t *testing.T) {
			root := testWorkspace(t)
			writeCanonical(t, root, "project-guide", "# Project guide\n")
			canonicalManifest := filepath.Join(root, ".agents", "skills", "project-guide", "skill.yaml")
			canonicalInstructions := filepath.Join(root, ".agents", "skills", "project-guide", "instructions.md")
			specKit := filepath.Join(root, ".agents", "skills", "speckit-plan", "SKILL.md")
			providerNative := filepath.Join(root, filepath.FromSlash(providerRoot), "provider-native", "SKILL.md")
			writeSentinel(t, specKit, "official Spec Kit sentinel\n")
			writeSentinel(t, providerNative, "provider-native sentinel\n")
			before := snapshotFiles(t, canonicalManifest, canonicalInstructions, specKit, providerNative)

			prepared, err := providersync.Prepare(root, provider, "project-guide")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Execute(prepared.Plan, prepared.Plan.PlanID, Options{Link: testNoReplace}); err != nil {
				if errors.Is(err, ErrNoReplaceUnsupported) {
					t.Skipf("test filesystem does not support no-replace publication: %v", err)
				}
				t.Fatal(err)
			}
			assertSnapshot(t, before)

			unchanged, err := providersync.NewPlan(root, provider, "project-guide")
			if err != nil {
				t.Fatal(err)
			}
			if !unchanged.Executable || len(unchanged.Outputs) != 1 || unchanged.Outputs[0].Action != "unchanged" {
				t.Fatalf("unchanged plan = %#v", unchanged)
			}

			unsync, err := providersync.NewUnsyncPlan(root, provider, "project-guide")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Execute(unsync, unsync.PlanID, Options{Link: testNoReplace}); err != nil {
				t.Fatal(err)
			}
			assertSnapshot(t, before)
			for _, file := range prepared.Files {
				if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file.Path))); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("unsynchronized output remains at %s: %v", file.Path, err)
				}
			}
		})
	}
}

func writeSentinel(t *testing.T, name, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func snapshotFiles(t *testing.T, names ...string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte, len(names))
	for _, name := range names {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		result[name] = data
	}
	return result
}

func assertSnapshot(t *testing.T, snapshot map[string][]byte) {
	t.Helper()
	for name, before := range snapshot {
		after, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("preserved file %s is unavailable: %v", name, err)
		}
		if string(after) != string(before) {
			t.Fatalf("preserved file %s changed", name)
		}
	}
}
