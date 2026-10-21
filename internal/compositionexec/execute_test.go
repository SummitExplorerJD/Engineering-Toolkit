package compositionexec

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/composition"
)

func TestExecuteCreatePublishesInitializedProject(t *testing.T) {
	inventory, catalogFS := executionCatalog(t)
	parent := localParent(t)
	target := filepath.Join(parent, "service")
	preview := createPlan(t, target, inventory, catalogFS)
	result, err := Execute(preview, preview.PlanID, inventory, catalogFS, Options{ProcessID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("status = %q", result.Status)
	}
	for _, relative := range []string{"AGENTS.md", ".agents/skills/project-guide/skill.yaml", ".engkit/config.yaml", ".engkit/plans/composition"} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("missing %s: %v", relative, err)
		}
	}
	if _, err := os.Stat(filepath.Join(parent, ".service.engkit-create.lock")); !os.IsNotExist(err) {
		t.Fatalf("lock state = %v", err)
	}
}

func TestExecuteCreatePublishesIntoEmptyDestination(t *testing.T) {
	inventory, catalogFS := executionCatalog(t)
	parent := localParent(t)
	target := filepath.Join(parent, "service")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	preview := createPlan(t, target, inventory, catalogFS)
	result, err := Execute(preview, preview.PlanID, inventory, catalogFS, Options{ProcessID: 9})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("status = %q", result.Status)
	}
	if _, err := os.Stat(filepath.Join(target, "AGENTS.md")); err != nil {
		t.Fatalf("missing AGENTS.md: %v", err)
	}
	key, err := planKey(preview.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(parent, ".service.engkit-empty-backup-"+key)
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("empty destination backup state = %v", err)
	}
}

func TestExecuteComposePublishesAdditiveOutputs(t *testing.T) {
	inventory, catalogFS := executionCatalog(t)
	root := initializedWorkspace(t)
	preview, err := composition.NewCompose(root, "go-service", "0.1.0", map[string]string{"project_name": "Service"}, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Execute(preview, preview.PlanID, inventory, catalogFS, Options{ProcessID: 8})
	if errors.Is(err, ErrNoReplaceUnsupported) {
		t.Skipf("test filesystem does not support hard-link publication: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" {
		t.Fatalf("status = %q", result.Status)
	}
	if got, err := os.ReadFile(filepath.Join(root, "AGENTS.md")); err != nil || string(got) != "# Service instructions\n" {
		t.Fatalf("AGENTS.md = %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "go.mod")); err != nil || string(got) != "module example.com/service\n\ngo 1.26.4\n" {
		t.Fatalf("unowned go.mod changed: %q, %v", got, err)
	}
}

func TestExecuteRejectsApprovalBeforeWriting(t *testing.T) {
	inventory, catalogFS := executionCatalog(t)
	parent := localParent(t)
	target := filepath.Join(parent, "service")
	preview := createPlan(t, target, inventory, catalogFS)
	if _, err := Execute(preview, "sha256:"+string(make([]byte, 64)), inventory, catalogFS, Options{}); err == nil {
		t.Fatal("mismatched approval succeeded")
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("approval failure wrote %d entries", len(entries))
	}
}

func TestRecoverCreateBeforeAndAfterPublication(t *testing.T) {
	inventory, catalogFS := executionCatalog(t)
	for _, test := range []struct {
		name       string
		point      string
		wantStatus string
	}{
		{name: "staged", point: "after_staging", wantStatus: "rolled_back"},
		{name: "published", point: "after_target_publish", wantStatus: "complete"},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent := localParent(t)
			target := filepath.Join(parent, "service")
			preview := createPlan(t, target, inventory, catalogFS)
			_, err := Execute(preview, preview.PlanID, inventory, catalogFS, Options{FailAt: failOnce(test.point)})
			if err == nil {
				t.Fatal("injected failure did not stop execution")
			}
			result, err := Recover(target, preview.PlanID, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != test.wantStatus {
				t.Fatalf("status = %q, want %q", result.Status, test.wantStatus)
			}
		})
	}
}

func TestRecoverCreateRefusesChangedStaging(t *testing.T) {
	inventory, catalogFS := executionCatalog(t)
	parent := localParent(t)
	target := filepath.Join(parent, "service")
	preview := createPlan(t, target, inventory, catalogFS)
	if _, err := Execute(preview, preview.PlanID, inventory, catalogFS, Options{FailAt: failOnce("after_staging")}); err == nil {
		t.Fatal("injected failure did not stop execution")
	}
	key, err := planKey(preview.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(parent, ".service.engkit-create-"+key)
	if err := os.WriteFile(filepath.Join(stage, "AGENTS.md"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Recover(target, preview.PlanID, Options{}); err == nil {
		t.Fatal("recovery accepted changed staging")
	}
	if _, err := os.Stat(filepath.Join(parent, ".service.engkit-create.lock")); err != nil {
		t.Fatalf("recovery refusal did not preserve the lock: %v", err)
	}
}

func TestRecoverComposeRefusesChangedAppliedOutput(t *testing.T) {
	inventory, catalogFS := executionCatalog(t)
	root := initializedWorkspace(t)
	preview, err := composition.NewCompose(root, "go-service", "0.1.0", map[string]string{"project_name": "Service"}, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(preview, preview.PlanID, inventory, catalogFS, Options{FailAt: failOnce("after_output_publish")})
	if errors.Is(err, ErrNoReplaceUnsupported) {
		t.Skipf("test filesystem does not support hard-link publication: %v", err)
	}
	if err == nil {
		t.Fatal("injected failure did not stop execution")
	}
	applied := filepath.Join(root, ".agents", "skills", "project-guide", "instructions.md")
	if _, statErr := os.Stat(applied); statErr == nil {
		if writeErr := os.WriteFile(applied, []byte("changed\n"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if _, err := Recover(root, preview.PlanID, Options{}); err == nil {
		t.Fatal("recovery accepted changed applied output")
	}
}

func failOnce(wanted string) func(string) error {
	fired := false
	return func(point string) error {
		if point == wanted && !fired {
			fired = true
			return errors.New("stop")
		}
		return nil
	}
}

func createPlan(t *testing.T, target string, inventory catalog.Catalog, catalogFS fs.FS) composition.Plan {
	t.Helper()
	preview, err := composition.NewCreate(target, "go-service", "0.1.0", map[string]string{"project_name": "Service"}, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Executable {
		t.Fatalf("plan is not executable: %#v", preview.Conflicts)
	}
	return preview
}

func initializedWorkspace(t *testing.T) string {
	t.Helper()
	root := filepath.Join(localParent(t), "workspace")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/service\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".engkit"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte(configContents), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".engkit", ".gitignore"), []byte(gitignoreContents), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func executionCatalog(t *testing.T) (catalog.Catalog, fs.FS) {
	t.Helper()
	templateSource := []byte("# {{engkit.project_name}}\n")
	templateSourceDigest := sha256.Sum256(templateSource)
	templateDefinition := fmt.Sprintf(`schema_version: engkit.skill-template/v1alpha1
id: project-guide
version: 0.1.0
description: Project guide template.
variables:
  - name: project_name
    type: string
    description: Project name.
    required: true
skill:
  id: project-guide
  version: 0.1.0
  description: Guides project work.
  instructions: instructions.md
  permissions: {}
  compatibility:
    minimum_engkit: 0.2.1
  ownership:
    owner: engineering
files:
  - source: files/instructions.md.tmpl
    destination: instructions.md
    kind: instructions
    media_type: text
    source_sha256: sha256:%s
`, hex.EncodeToString(templateSourceDigest[:]))
	asset := []byte("# {{engkit.project_name}} instructions\n")
	assetDigest := sha256.Sum256(asset)
	profileDefinition := fmt.Sprintf(`schema_version: engkit.project-profile/v1alpha1
id: go-service
version: 0.1.0
description: Go service workspace.
compatibility:
  ecosystems: [go]
  modes: [new-project, existing-workspace]
variables:
  - name: project_name
    type: string
    description: Project name.
    required: true
skills:
  - template_id: project-guide
    template_version: 0.1.0
    bindings:
      project_name:
        variable: project_name
workspace_directories: [.agents/context, .agents/conventions, .agents/skills, .agents/workflows]
workspace_assets:
  - source: files/AGENTS.md.tmpl
    destination: AGENTS.md
    kind: agent-instructions
    media_type: text
    source_sha256: sha256:%s
source_layout:
  policy: src
integration_requests: []
`, hex.EncodeToString(assetDigest[:]))
	filesystem := fstest.MapFS{
		"templates/project-guide/0.1.0/template.yaml":              {Data: []byte(templateDefinition)},
		"templates/project-guide/0.1.0/files/instructions.md.tmpl": {Data: templateSource},
		"profiles/go-service/0.1.0/profile.yaml":                   {Data: []byte(profileDefinition)},
		"profiles/go-service/0.1.0/files/AGENTS.md.tmpl":           {Data: asset},
	}
	templateDigest, err := catalog.DirectoryDigest(filesystem, "templates/project-guide/0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	profileDigest, err := catalog.DirectoryDigest(filesystem, "profiles/go-service/0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	catalogDefinition := fmt.Sprintf(`schema_version: engkit.catalog/v1alpha1
id: test-catalog
version: 0.2.1
minimum_engkit: 0.2.1
templates:
  - id: project-guide
    version: 0.1.0
    path: templates/project-guide/0.1.0
    sha256: %s
profiles:
  - id: go-service
    version: 0.1.0
    path: profiles/go-service/0.1.0
    sha256: %s
`, templateDigest, profileDigest)
	filesystem["catalog.yaml"] = &fstest.MapFile{Data: []byte(catalogDefinition)}
	inventory, err := catalog.Load(filesystem, "catalog.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return inventory, filesystem
}

func localParent(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp(".", "engkit-compositionexec-test-")
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(absolute) })
	return absolute
}
