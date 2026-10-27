package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillplan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

func TestVersionJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}

	var got map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["schema_version"] != versionSchema || got["version"] != Version {
		t.Fatalf("unexpected version output: %#v", got)
	}
}

func TestDoctorEmitsStructuredResultAndHealthExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--scope", "core"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	var healthy struct {
		SchemaVersion string `json:"schema_version"`
		Summary       struct {
			Healthy bool `json:"healthy"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &healthy); err != nil {
		t.Fatal(err)
	}
	if healthy.SchemaVersion != "engkit.doctor-result/v1alpha1" || !healthy.Summary.Healthy {
		t.Fatalf("doctor result = %#v", healthy)
	}

	stdout.Reset()
	stderr.Reset()
	missing := filepath.Join(t.TempDir(), "missing")
	code = Run([]string{"doctor", "--root", missing, "--scope", "workspace"}, &stdout, &stderr)
	if code != 1 || stderr.Len() != 0 {
		t.Fatalf("unhealthy Run() code = %d, stderr = %q", code, stderr.String())
	}
	var unhealthy struct {
		Summary struct {
			Healthy        bool `json:"healthy"`
			RequiredErrors int  `json:"required_errors"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &unhealthy); err != nil {
		t.Fatal(err)
	}
	if unhealthy.Summary.Healthy || unhealthy.Summary.RequiredErrors == 0 {
		t.Fatalf("unhealthy doctor result = %#v", unhealthy)
	}
}

func TestScopedHelpSucceeds(t *testing.T) {
	for _, args := range [][]string{
		{"doctor", "--help"},
		{"init", "--help"},
		{"init", "plan", "--help"},
		{"init", "apply", "--help"},
		{"mcp", "--help"},
		{"mcp", "serve", "--help"},
		{"skills", "--help"},
		{"skills", "list", "--help"},
		{"workspace", "help"},
		{"templates", "scaffold", "-h"},
		{"profiles", "--help"},
		{"project", "create", "--help"},
		{"project", "recover", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Usage:") || stderr.Len() != 0 {
			t.Fatalf("Run(%v) stdout = %q, stderr = %q", args, stdout.String(), stderr.String())
		}
	}
}

func TestMCPServeRequiresExplicitTransportAndRoot(t *testing.T) {
	for _, args := range [][]string{
		{"mcp"},
		{"mcp", "unknown"},
		{"mcp", "serve"},
		{"mcp", "serve", "--transport", "http", "--root", "."},
		{"mcp", "serve", "--transport", "stdio"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 2 {
			t.Fatalf("Run(%v) code = %d, stdout = %q, stderr = %q", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestUnifiedInitHelpExplainsSelectionWithoutDiscovery(t *testing.T) {
	for _, args := range [][]string{{"init", "--help"}, {"init", "plan", "--help"}, {"init", "apply", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Selection:") || !strings.Contains(stdout.String(), "specialized plan unchanged") {
			t.Fatalf("Run(%v) output = %q", args, stdout.String())
		}
	}
}

func TestBuiltinCatalogCommandsAreOfflineAndDeterministic(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"catalog", "inspect"}, want: `"schema_version": "engkit.catalog/v1alpha1"`},
		{args: []string{"templates", "list"}, want: `"schema_version": "engkit.template-list/v1alpha1"`},
		{args: []string{"templates", "inspect", "--id", "project-guide", "--version", "0.1.0"}, want: `"id": "project-guide"`},
		{args: []string{"profiles", "list"}, want: `"schema_version": "engkit.profile-list/v1alpha1"`},
		{args: []string{"profiles", "inspect", "--id", "go-service", "--version", "0.1.0"}, want: `"id": "go-service"`},
	} {
		var first, second, stderr bytes.Buffer
		if code := Run(test.args, &first, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		stderr.Reset()
		if code := Run(test.args, &second, &stderr); code != 0 {
			t.Fatalf("second Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if first.String() != second.String() || !strings.Contains(first.String(), test.want) {
			t.Fatalf("Run(%v) output = %q, second = %q", test.args, first.String(), second.String())
		}
	}
}

func TestTemplateScaffoldCLI(t *testing.T) {
	parent, err := os.MkdirTemp(".", "engkit-cli-template-test-")
	if err != nil {
		t.Fatal(err)
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	output := filepath.Join(parent, "release-review")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"templates", "scaffold", "--output", output, "--id", "release-review", "--version", "0.1.0"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version": "engkit.template-scaffold-result/v1alpha1"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(output, "template.yaml")); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"templates", "scaffold", "--output", output, "--id", "release-review", "--version", "0.1.0"}, &stdout, &stderr); code != 1 {
		t.Fatalf("second Run() code = %d, stderr = %q", code, stderr.String())
	}
}

func TestWorkspaceInspectJSON(t *testing.T) {
	root := testWorkspace(t)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"workspace", "inspect", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}

	var got workspace.Inspection
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.GoModule != "example.com/project" {
		t.Fatalf("GoModule = %q", got.GoModule)
	}
}

func TestProjectInspectionAndRecommendationsAreReadOnly(t *testing.T) {
	root := testWorkspace(t)
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"project", "inspect", "--root", root}, want: `"schema_version": "engkit.project-inspection/v1alpha1"`},
		{args: []string{"profiles", "recommend", "--root", root}, want: `"schema_version": "engkit.profile-recommendation/v1alpha1"`},
		{args: []string{"project", "--help"}, want: "engkit project plan-create"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("Run(%v) output = %q, want %q", test.args, stdout.String(), test.want)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("read-only project commands wrote .agents: %v", err)
	}
}

func TestProjectPlanningRequiresExactPackagedProfileWithoutWriting(t *testing.T) {
	parent, err := os.MkdirTemp(".", "engkit-cli-project-test-")
	if err != nil {
		t.Fatal(err)
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	target := filepath.Join(parent, "new-project")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"project", "plan-create", "--destination", target, "--profile", "go-service@0.1.0", "--set", "project_name=Toolkit"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var preview struct {
		PlanID     string `json:"plan_id"`
		Executable bool   `json:"executable"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.PlanID == "" || !preview.Executable {
		t.Fatalf("unexpected plan: %#v", preview)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("planning wrote target: %v", err)
	}
}

func TestProjectCreateInstallsBuiltinPilotProfile(t *testing.T) {
	parent, err := os.MkdirTemp(".", "engkit-cli-project-create-test-")
	if err != nil {
		t.Fatal(err)
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	target := filepath.Join(parent, "pilot")
	arguments := []string{"--destination", target, "--profile", "go-service@0.1.0", "--set", "project_name=Pilot"}
	var planOutput, stderr bytes.Buffer
	if code := Run(append([]string{"project", "plan-create"}, arguments...), &planOutput, &stderr); code != 0 {
		t.Fatalf("plan code = %d, stderr = %q", code, stderr.String())
	}
	var preview struct {
		PlanID string `json:"plan_id"`
	}
	if err := json.Unmarshal(planOutput.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	var createOutput bytes.Buffer
	stderr.Reset()
	createArguments := append([]string{"project", "create"}, arguments...)
	createArguments = append(createArguments, "--approve", preview.PlanID)
	if code := Run(createArguments, &createOutput, &stderr); code != 0 {
		t.Fatalf("create code = %d, stderr = %q", code, stderr.String())
	}
	for _, relative := range []string{"AGENTS.md", ".agents/context/project-overview.md", ".agents/conventions/go.md", ".agents/skills/project-guide/skill.yaml", ".agents/workflows/verify.md", ".engkit/config.yaml", "src"} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("missing %s: %v", relative, err)
		}
	}
	var validationOutput bytes.Buffer
	stderr.Reset()
	if code := Run([]string{"skills", "validate", "--root", target, "--id", "project-guide"}, &validationOutput, &stderr); code != 0 {
		t.Fatalf("validate code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(validationOutput.String(), `"valid": true`) {
		t.Fatalf("unexpected validation output: %s", validationOutput.String())
	}
}

func TestUnifiedInitPlansEqualSpecializedPlans(t *testing.T) {
	parent := t.TempDir()
	createTarget := filepath.Join(parent, "new-project")
	workspaceTarget := testWorkspace(t)
	composeTarget := testWorkspace(t)
	initializeWithCLI(t, composeTarget)

	tests := []struct {
		name        string
		unified     []string
		specialized []string
	}{
		{
			name:        "new project",
			unified:     []string{"init", "plan", "--target", createTarget, "--profile", "go-service@0.1.0", "--set", "project_name=Unified"},
			specialized: []string{"project", "plan-create", "--destination", createTarget, "--profile", "go-service@0.1.0", "--set", "project_name=Unified"},
		},
		{
			name:        "workspace initialization",
			unified:     []string{"init", "plan", "--target", workspaceTarget},
			specialized: []string{"workspace", "plan-init", "--root", workspaceTarget},
		},
		{
			name:        "workspace composition",
			unified:     []string{"init", "plan", "--target", composeTarget, "--profile", "go-service@0.1.0", "--set", "project_name=Unified"},
			specialized: []string{"project", "plan-compose", "--root", composeTarget, "--profile", "go-service@0.1.0", "--set", "project_name=Unified"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var unified, specialized, unifiedErr, specializedErr bytes.Buffer
			if code := Run(test.unified, &unified, &unifiedErr); code != 0 {
				t.Fatalf("Run(%v) code = %d, stderr = %q", test.unified, code, unifiedErr.String())
			}
			if code := Run(test.specialized, &specialized, &specializedErr); code != 0 {
				t.Fatalf("Run(%v) code = %d, stderr = %q", test.specialized, code, specializedErr.String())
			}
			if unified.String() != specialized.String() {
				t.Fatalf("unified plan differs from specialized plan\nunified: %s\nspecialized: %s", unified.String(), specialized.String())
			}
		})
	}
	if _, err := os.Lstat(createTarget); !os.IsNotExist(err) {
		t.Fatalf("unified planning created target: %v", err)
	}
}

func TestUnifiedInitApplyUsesExactApproval(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "pilot")
	arguments := []string{"--target", target, "--profile", "go-service@0.1.0", "--set", "project_name=Pilot"}
	var planOutput, stderr bytes.Buffer
	if code := Run(append([]string{"init", "plan"}, arguments...), &planOutput, &stderr); code != 0 {
		t.Fatalf("plan code = %d, stderr = %q", code, stderr.String())
	}
	var preview struct {
		PlanID string `json:"plan_id"`
	}
	if err := json.Unmarshal(planOutput.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	stderr.Reset()
	wrong := append(append([]string{"init", "apply"}, arguments...), "--approve", "sha256:"+strings.Repeat("0", 64))
	if code := Run(wrong, &output, &stderr); code != 1 {
		t.Fatalf("wrong approval code = %d, stdout = %q, stderr = %q", code, output.String(), stderr.String())
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("rejected apply created target: %v", err)
	}

	output.Reset()
	stderr.Reset()
	approved := append(append([]string{"init", "apply"}, arguments...), "--approve", preview.PlanID)
	if code := Run(approved, &output, &stderr); code != 0 {
		t.Fatalf("approved apply code = %d, stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(target, ".engkit", "config.yaml")); err != nil {
		t.Fatalf("approved apply did not create initialized project: %v", err)
	}
}

func TestUnifiedInitApplyRejectsChangedClassification(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "changed")
	arguments := []string{"--target", target, "--profile", "go-service@0.1.0", "--set", "project_name=Changed"}
	var planOutput, stderr bytes.Buffer
	if code := Run(append([]string{"init", "plan"}, arguments...), &planOutput, &stderr); code != 0 {
		t.Fatalf("plan code = %d, stderr = %q", code, stderr.String())
	}
	var preview struct {
		PlanID string `json:"plan_id"`
	}
	if err := json.Unmarshal(planOutput.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("# Changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	apply := append(append([]string{"init", "apply"}, arguments...), "--approve", preview.PlanID)
	if code := Run(apply, &output, &stderr); code != 1 {
		t.Fatalf("changed classification code = %d, stdout = %q, stderr = %q", code, output.String(), stderr.String())
	}
	if _, err := os.Lstat(filepath.Join(target, ".engkit")); !os.IsNotExist(err) {
		t.Fatalf("rejected changed classification wrote .engkit: %v", err)
	}
}

func TestUnifiedInitRefusesCompoundAndInvalidInput(t *testing.T) {
	root := testWorkspace(t)
	for _, test := range []struct {
		args []string
		want int
	}{
		{args: []string{"init", "plan", "--target", root, "--profile", "go-service@0.1.0", "--set", "project_name=Blocked"}, want: 1},
		{args: []string{"init", "plan", "--target", root, "--set", "project_name=Blocked"}, want: 2},
		{args: []string{"init", "apply", "--target", root}, want: 2},
		{args: []string{"init", "plan"}, want: 2},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != test.want {
			t.Fatalf("Run(%v) code = %d, want %d, stdout = %q, stderr = %q", test.args, code, test.want, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Lstat(filepath.Join(root, ".engkit")); !os.IsNotExist(err) {
		t.Fatalf("refused unified init wrote .engkit: %v", err)
	}
}

func TestWorkspacePlanInitDoesNotWrite(t *testing.T) {
	root := testWorkspace(t)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"workspace", "plan-init", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}

	var got plan.Plan
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != "workspace.init" || !got.RequiresApproval {
		t.Fatalf("unexpected plan: %#v", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit")); !os.IsNotExist(err) {
		t.Fatalf("plan-init created .engkit or returned an unexpected error: %v", err)
	}
}

func TestWorkspaceInitRequiresAndUsesApprovedPlan(t *testing.T) {
	root := testWorkspace(t)
	var planOut, planErr bytes.Buffer
	code := Run([]string{"workspace", "plan-init", "--root", root}, &planOut, &planErr)
	if code != 0 {
		t.Fatalf("plan-init code = %d, stderr = %q", code, planErr.String())
	}
	var preview plan.Plan
	if err := json.Unmarshal(planOut.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code = Run([]string{"workspace", "init", "--root", root, "--approve", preview.PlanID}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init code = %d, stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".engkit", "config.yaml")); err != nil {
		t.Fatalf("expected initialized config: %v", err)
	}
}

func TestContextLifecycle(t *testing.T) {
	root := testWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Project\n\nDurable context discovery.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	initializeWithCLI(t, root)

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"context", "rebuild", "--root", root}, want: `"artifact_count"`},
		{args: []string{"context", "status", "--root", root}, want: `"fresh": true`},
		{args: []string{"context", "search", "--root", root, "context discovery"}, want: `"README.md"`},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("Run(%v) output = %q, want %q", test.args, stdout.String(), test.want)
		}
	}
}

func TestAdviseProfilesUsesDeterministicFallbackWhenDisabled(t *testing.T) {
	root := testWorkspace(t)
	initializeWithCLI(t, root)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"advise", "profiles", "--root", root, "--goal", "Select a compatible profile."}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("advise code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version": "engkit.advice-result/v1alpha1"`) || !strings.Contains(stdout.String(), `"mode": "deterministic"`) || !strings.Contains(stdout.String(), `"go-service@0.1.0"`) {
		t.Fatalf("unexpected advice: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"model"`) {
		t.Fatalf("disabled advice reported model evidence: %s", stdout.String())
	}
}

func TestAdapterAndSpecKitCommandsAreReadOnly(t *testing.T) {
	root := testWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Rules\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"adapters", "list"}, want: `"id": "codex"`},
		{args: []string{"adapters", "inspect", "--root", root, "--provider", "codex"}, want: `"configured": true`},
		{args: []string{"speckit", "plan", "--root", root, "--integration", "codex", "--version", "v0.12.11"}, want: `"executable": false`},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("Run(%v) output = %q, want %q", test.args, stdout.String(), test.want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".specify")); !os.IsNotExist(err) {
		t.Fatalf("read-only commands created .specify or returned an unexpected error: %v", err)
	}
}

func TestSkillCommandsClassifyExternalPackagesWithoutWriting(t *testing.T) {
	root := testWorkspace(t)
	skillRoot := filepath.Join(root, ".agents", "skills", "speckit-plan")
	if err := os.MkdirAll(skillRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillRoot, "SKILL.md")
	contents := []byte("# External Spec Kit skill\n")
	if err := os.WriteFile(skillPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"skills", "list", "--root", root}, want: `"classification": "external"`},
		{args: []string{"skills", "inspect", "--root", root, "--id", "speckit-plan"}, want: `"directory": "speckit-plan"`},
		{args: []string{"skills", "validate", "--root", root}, want: `"valid": true`},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("Run(%v) output = %q, want %q", test.args, stdout.String(), test.want)
		}
	}
	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, contents) {
		t.Fatal("read-only skill commands modified an external skill")
	}
}

func TestSkillValidateUsesFailureExitCode(t *testing.T) {
	root := testWorkspace(t)
	invalidRoot := filepath.Join(root, ".agents", "skills", "invalid-skill")
	if err := os.MkdirAll(invalidRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invalidRoot, "skill.yaml"), []byte("schema_version: unsupported\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"skills", "validate", "--root", root, "--id", "invalid-skill"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), `"valid": false`) || !strings.Contains(stderr.String(), "skill validation failed") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestSkillMutationPlanCommandsAreReadOnly(t *testing.T) {
	root := testWorkspace(t)
	sourceParent := t.TempDir()
	source := writeCLISkill(t, sourceParent, "release-review", "1.0.0", "# Proposed\n")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"skills", "plan-create", "--root", root, "--source", source}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"kind": "skill.create"`) || !strings.Contains(stdout.String(), `"executable": true`) {
		t.Fatalf("plan-create code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("plan-create mutated workspace: %v", err)
	}

	writeCLISkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Current\n")
	source = writeCLISkill(t, t.TempDir(), "release-review", "1.1.0", "# Proposed update\n")
	for _, test := range []struct {
		args []string
		kind string
	}{
		{args: []string{"skills", "plan-update", "--root", root, "--id", "release-review", "--source", source}, kind: "skill.update"},
		{args: []string{"skills", "plan-remove", "--root", root, "--id", "release-review"}, kind: "skill.remove"},
	} {
		stdout.Reset()
		stderr.Reset()
		code = Run(test.args, &stdout, &stderr)
		if code != 0 || !strings.Contains(stdout.String(), `"kind": "`+test.kind+`"`) {
			t.Fatalf("Run(%v) code = %d, stdout = %q, stderr = %q", test.args, code, stdout.String(), stderr.String())
		}
	}
	if got, err := os.ReadFile(filepath.Join(root, ".agents", "skills", "release-review", "instructions.md")); err != nil || string(got) != "# Current\n" {
		t.Fatalf("planning modified target instructions: %q, %v", got, err)
	}
}

func TestProviderSynchronizationPlanCLIIsReadOnly(t *testing.T) {
	root := testWorkspace(t)
	writeCLISkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Review\n")

	var first, second, stderr bytes.Buffer
	args := []string{"skills", "plan-sync", "--root", root, "--provider", "opencode", "--id", "release-review"}
	if code := Run(args, &first, &stderr); code != 0 {
		t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	stderr.Reset()
	if code := Run(args, &second, &stderr); code != 0 {
		t.Fatalf("second Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	if first.String() != second.String() || !strings.Contains(first.String(), `"schema_version": "engkit.provider-sync-plan/v1alpha1"`) || !strings.Contains(first.String(), `"executable": true`) {
		t.Fatalf("unexpected provider plan output: first=%q second=%q", first.String(), second.String())
	}
	if _, err := os.Lstat(filepath.Join(root, ".opencode")); !os.IsNotExist(err) {
		t.Fatalf("provider planning wrote .opencode: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".engkit", "providers")); !os.IsNotExist(err) {
		t.Fatalf("provider planning wrote ownership state: %v", err)
	}
}

func TestProviderSynchronizationMutationCLI(t *testing.T) {
	root := testWorkspace(t)
	initializeWithCLI(t, root)
	writeCLISkill(t, filepath.Join(root, ".agents", "skills"), "release-review", "1.0.0", "# Review\n")

	syncPlan := providerPlanFromCLI(t, []string{"skills", "plan-sync", "--root", root, "--provider", "claude", "--id", "release-review"})
	runCLI(t, []string{"skills", "sync", "--root", root, "--provider", "claude", "--id", "release-review", "--approve", syncPlan.PlanID}, `"status": "complete"`)
	if _, err := os.Lstat(filepath.Join(root, ".claude", "skills", "release-review", "SKILL.md")); err != nil {
		t.Fatalf("sync output is unavailable: %v", err)
	}

	unsyncPlan := providerPlanFromCLI(t, []string{"skills", "plan-unsync", "--root", root, "--provider", "claude", "--id", "release-review"})
	runCLI(t, []string{"skills", "unsync", "--root", root, "--provider", "claude", "--id", "release-review", "--approve", unsyncPlan.PlanID}, `"status": "complete"`)
	if _, err := os.Lstat(filepath.Join(root, ".claude", "skills", "release-review", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("unsync output remains: %v", err)
	}
}

func TestSkillMutationCLI(t *testing.T) {
	root := testWorkspace(t)
	initializeWithCLI(t, root)
	source := writeCLISkill(t, t.TempDir(), "release-review", "1.0.0", "# Proposed\n")

	create := skillPlanFromCLI(t, []string{"skills", "plan-create", "--root", root, "--source", source})
	runCLI(t, []string{"skills", "create", "--root", root, "--source", source, "--approve", create.PlanID}, `"status": "complete"`)

	source = writeCLISkill(t, t.TempDir(), "release-review", "1.1.0", "# Updated\n")
	update := skillPlanFromCLI(t, []string{"skills", "plan-update", "--root", root, "--id", "release-review", "--source", source})
	runCLI(t, []string{"skills", "update", "--root", root, "--id", "release-review", "--source", source, "--approve", update.PlanID}, `"status": "complete"`)

	remove := skillPlanFromCLI(t, []string{"skills", "plan-remove", "--root", root, "--id", "release-review"})
	runCLI(t, []string{"skills", "remove", "--root", root, "--id", "release-review", "--approve", remove.PlanID}, `"status": "complete"`)
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "release-review")); !os.IsNotExist(err) {
		t.Fatalf("skills remove target state = %v", err)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func testWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	content := []byte("module example.com/project\n\ngo 1.26.4\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func initializeWithCLI(t *testing.T, root string) {
	t.Helper()
	var planOut, planErr bytes.Buffer
	if code := Run([]string{"workspace", "plan-init", "--root", root}, &planOut, &planErr); code != 0 {
		t.Fatalf("plan-init code = %d, stderr = %q", code, planErr.String())
	}
	var preview plan.Plan
	if err := json.Unmarshal(planOut.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	var initOut, initErr bytes.Buffer
	if code := Run([]string{"workspace", "init", "--root", root, "--approve", preview.PlanID}, &initOut, &initErr); code != 0 {
		t.Fatalf("workspace init code = %d, stderr = %q", code, initErr.String())
	}
}

func skillPlanFromCLI(t *testing.T, args []string) skillplan.Plan {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	var preview skillplan.Plan
	if err := json.Unmarshal(stdout.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	return preview
}

func providerPlanFromCLI(t *testing.T, args []string) providersync.Plan {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	var preview providersync.Plan
	if err := json.Unmarshal(stdout.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	return preview
}

func runCLI(t *testing.T, args []string, want string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 0 {
		if strings.Contains(stderr.String(), "atomic no-replace publication is unsupported") {
			t.Skipf("test filesystem does not support hard-link publication: %s", stderr.String())
		}
		t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), want) {
		t.Fatalf("Run(%v) output = %q, want %q", args, stdout.String(), want)
	}
}

func writeCLISkill(t *testing.T, parent, id, version, instructions string) string {
	t.Helper()
	packageRoot := filepath.Join(parent, id)
	if err := os.MkdirAll(packageRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "instructions.md"), []byte(instructions), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(instructions))
	manifest := fmt.Sprintf(`schema_version: engkit.skill/v1alpha1
id: %s
version: %s
description: Reviews releases.
instructions: instructions.md
permissions: {}
compatibility:
  minimum_engkit: 0.2.0
ownership:
  owner: engineering
provenance:
  source: local
  revision: %s
resources:
  - path: instructions.md
    kind: instructions
    sha256: sha256:%s
`, id, version, version, hex.EncodeToString(digest[:]))
	if err := os.WriteFile(filepath.Join(packageRoot, "skill.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return packageRoot
}
