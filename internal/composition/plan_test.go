package composition

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/profiles"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/templates"
)

func TestNewCreateIsDeterministicAndDoesNotWrite(t *testing.T) {
	inventory, catalogFS := testCatalog(t)
	parent := localParent(t)
	target := filepath.Join(parent, "new-project")
	values := map[string]string{"project_name": "Toolkit"}
	first, err := NewCreate(target, "go-service", "0.1.0", values, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCreate(target, "go-service", "0.1.0", values, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanID != second.PlanID || !first.Executable || !first.RequiresApproval {
		t.Fatalf("unexpected plans: %#v %#v", first, second)
	}
	if len(first.RenderedOutputs) != 3 || !hasOperation(first.Operations, "create_directory", "src") {
		t.Fatalf("unexpected plan outputs or operations: %#v", first)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("planning wrote target: %v", err)
	}
}

func TestNewComposeInventoriesWorkspaceWithoutWriting(t *testing.T) {
	inventory, catalogFS := testCatalog(t)
	root := filepath.Join(localParent(t), "workspace")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	goMod := []byte("module example.com/service\n\ngo 1.26.4\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), goMod, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".engkit"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".engkit", "config.yaml"), []byte("schema_version: engkit.config/v1alpha1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".engkit", ".gitignore"), []byte("/recovery/*\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := NewCompose(root, "go-service", "0.1.0", map[string]string{"project_name": "Service"}, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	if !stringContains(preview.WorkspaceFacts.Ecosystems, "go") || hasBlocking(preview.Conflicts) {
		t.Fatalf("unexpected workspace facts or conflicts: %#v", preview)
	}
	got, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(goMod) {
		t.Fatal("planning modified go.mod")
	}
	if _, err := os.Lstat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("planning wrote .agents: %v", err)
	}
}

func TestNewCreateReportsNonemptyDestinationConflict(t *testing.T) {
	inventory, catalogFS := testCatalog(t)
	root := filepath.Join(localParent(t), "occupied")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := NewCreate(root, "go-service", "0.1.0", map[string]string{"project_name": "Service"}, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	if !hasConflict(preview.Conflicts, "target.nonempty") {
		t.Fatalf("missing target conflict: %#v", preview.Conflicts)
	}
}

func TestNewComposeRequiresInitializedWorkspace(t *testing.T) {
	inventory, catalogFS := testCatalog(t)
	root := filepath.Join(localParent(t), "workspace")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/service\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := NewCompose(root, "go-service", "0.1.0", map[string]string{"project_name": "Service"}, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Executable || !hasConflict(preview.Conflicts, "workspace.uninitialized") {
		t.Fatalf("unexpected uninitialized plan: %#v", preview)
	}
}

func TestNewComposeBlocksPartialCanonicalSkill(t *testing.T) {
	inventory, catalogFS := testCatalog(t)
	root := filepath.Join(localParent(t), "workspace")
	if err := os.MkdirAll(filepath.Join(root, ".engkit"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string]string{"config.yaml": "schema_version: engkit.config/v1alpha1\n", ".gitignore": "/recovery/*\n"} {
		if err := os.WriteFile(filepath.Join(root, ".engkit", name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/service\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	skillRoot := filepath.Join(root, ".agents", "skills", "project-guide")
	if err := os.MkdirAll(skillRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "instructions.md"), []byte("# Service\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := NewCompose(root, "go-service", "0.1.0", map[string]string{"project_name": "Service"}, "0.2.1", inventory, catalogFS)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Executable || !hasConflict(preview.Conflicts, "skill.target_not_absent") {
		t.Fatalf("partial canonical skill was not blocked: %#v", preview.Conflicts)
	}
}

func TestBindTemplateRejectsLiteralAndReferenceTypeCoercion(t *testing.T) {
	definition := templates.Template{Variables: []templates.Variable{{Name: "count", Type: "integer", Description: "Count.", Required: true}}}
	if _, err := bindTemplate(definition, profiles.SkillSelection{Bindings: map[string]profiles.Binding{"count": {Literal: "2"}}}, nil, nil); err == nil {
		t.Fatal("string literal was coerced to integer")
	}
	profileVariables := []templates.Variable{{Name: "source", Type: "string", Description: "Source.", Required: true}}
	selection := profiles.SkillSelection{Bindings: map[string]profiles.Binding{"count": {Variable: "source"}}}
	if _, err := bindTemplate(definition, selection, profileVariables, map[string]any{"source": "2"}); err == nil {
		t.Fatal("string profile variable was coerced to integer")
	}
}

func testCatalog(t *testing.T) (catalog.Catalog, fs.FS) {
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
	directory, err := os.MkdirTemp(".", "engkit-composition-test-")
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

func hasOperation(values []Operation, action, path string) bool {
	for _, value := range values {
		if value.Action == action && value.Path == path {
			return true
		}
	}
	return false
}
func hasConflict(values []Conflict, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
func hasBlocking(values []Conflict) bool {
	for _, value := range values {
		if value.Severity == "blocking" {
			return true
		}
	}
	return false
}
