// Package doctor performs deterministic, read-only health checks.
package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	sqliteindex "github.com/SummitExplorerJD/Engineering-Toolkit/internal/index/sqlite"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	SchemaVersion = "engkit.doctor-result/v1alpha1"

	ScopeCore         = "core"
	ScopeWorkspace    = "workspace"
	ScopeIntegrations = "integrations"
	ScopeAll          = "all"

	StatusPass        = "pass"
	StatusWarning     = "warning"
	StatusError       = "error"
	StatusUnavailable = "unavailable"

	maxConfigurationBytes = 1 << 20
	maxOwnershipBytes     = 1 << 20
)

// ErrUnhealthy indicates that the complete JSON result contains one or more
// required error findings. The result remains the authoritative diagnostic.
var ErrUnhealthy = errors.New("required doctor checks failed")

// Options selects the bounded diagnostic surface.
type Options struct {
	Root     string
	Scope    string
	Version  string
	LookPath func(string) (string, error)
}

// Evidence is one deterministic, secret-safe observed fact.
type Evidence struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Finding is one stable diagnostic result.
type Finding struct {
	Code        string     `json:"code"`
	Scope       string     `json:"scope"`
	Status      string     `json:"status"`
	Required    bool       `json:"required"`
	Summary     string     `json:"summary"`
	Path        string     `json:"path,omitempty"`
	Evidence    []Evidence `json:"evidence,omitempty"`
	Remediation string     `json:"remediation,omitempty"`
}

// Summary contains stable counts and the overall required-check outcome.
type Summary struct {
	Pass           int  `json:"pass"`
	Warning        int  `json:"warning"`
	Error          int  `json:"error"`
	Unavailable    int  `json:"unavailable"`
	RequiredErrors int  `json:"required_errors"`
	Healthy        bool `json:"healthy"`
}

// Result is the versioned machine-readable doctor response.
type Result struct {
	SchemaVersion string    `json:"schema_version"`
	Root          string    `json:"root,omitempty"`
	Scope         string    `json:"scope"`
	Findings      []Finding `json:"findings"`
	Summary       Summary   `json:"summary"`
}

type workspaceState struct {
	inspection workspace.Inspection
	available  bool
}

// Inspect runs selected probes without mutating the workspace or invoking an
// integration runtime.
func Inspect(ctx context.Context, options Options) (Result, error) {
	scope := strings.ToLower(strings.TrimSpace(options.Scope))
	if scope == "" {
		scope = ScopeAll
	}
	if !validScope(scope) {
		return Result{}, errors.New("doctor scope must be one of core, workspace, integrations, or all")
	}
	root := strings.TrimSpace(options.Root)
	if root == "" {
		root = "."
	}
	result := Result{SchemaVersion: SchemaVersion, Scope: scope, Findings: []Finding{}}

	if includes(scope, ScopeCore) {
		result.Findings = append(result.Findings, inspectCore(options.Version)...)
	}

	var state workspaceState
	if includes(scope, ScopeWorkspace) || includes(scope, ScopeIntegrations) {
		result.Root = filepath.ToSlash(root)
		state = inspectRoot(root)
		if state.available {
			result.Root = state.inspection.Root
		} else {
			result.Findings = append(result.Findings, finding(
				"workspace.root", ScopeWorkspace, StatusError, true,
				"Workspace root could not be inspected.", "", nil,
				"Select an existing readable directory with --root.",
			))
		}
	}

	if includes(scope, ScopeWorkspace) {
		if state.available {
			result.Findings = append(result.Findings, inspectWorkspace(ctx, state.inspection)...)
		} else {
			result.Findings = append(result.Findings, skippedWorkspaceFindings()...)
		}
	}
	if includes(scope, ScopeIntegrations) {
		result.Findings = append(result.Findings, inspectIntegrations(state, options.LookPath)...)
	}

	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].Scope != result.Findings[j].Scope {
			return result.Findings[i].Scope < result.Findings[j].Scope
		}
		if result.Findings[i].Code != result.Findings[j].Code {
			return result.Findings[i].Code < result.Findings[j].Code
		}
		return result.Findings[i].Path < result.Findings[j].Path
	})
	result.Summary = summarize(result.Findings)
	return result, nil
}

func validScope(scope string) bool {
	return scope == ScopeCore || scope == ScopeWorkspace || scope == ScopeIntegrations || scope == ScopeAll
}

func includes(selected, candidate string) bool {
	return selected == ScopeAll || selected == candidate
}

func inspectCore(version string) []Finding {
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	findings := []Finding{finding(
		"core.runtime", ScopeCore, StatusPass, true,
		"Engineering Toolkit runtime information is available.", "", []Evidence{
			{Name: "engkit_version", Value: version},
			{Name: "go_version", Value: runtime.Version()},
			{Name: "platform", Value: runtime.GOOS + "/" + runtime.GOARCH},
		}, "",
	)}
	loaded, _, err := catalog.Builtin()
	if err != nil {
		findings = append(findings, finding("core.catalog", ScopeCore, StatusError, true, "The packaged first-party catalog is invalid.", "", nil, "Reinstall a verified Engineering Toolkit release."))
		return findings
	}
	findings = append(findings, finding(
		"core.catalog", ScopeCore, StatusPass, true,
		"The packaged first-party catalog passed integrity validation.", "", []Evidence{
			{Name: "catalog", Value: loaded.ID + "@" + loaded.Version},
			{Name: "profiles", Value: fmt.Sprint(len(loaded.Profiles))},
			{Name: "templates", Value: fmt.Sprint(len(loaded.Templates))},
		}, "",
	))
	return findings
}

func inspectRoot(root string) workspaceState {
	inspection, err := workspace.Inspect(root)
	if err != nil {
		return workspaceState{}
	}
	return workspaceState{inspection: inspection, available: true}
}

func inspectWorkspace(ctx context.Context, inspection workspace.Inspection) []Finding {
	findings := []Finding{finding("workspace.root", ScopeWorkspace, StatusPass, true, "Workspace root is a readable directory.", "", []Evidence{{Name: "root", Value: inspection.Root}}, "")}
	if !inspection.HasEngkit {
		findings = append(findings, finding("workspace.initialized", ScopeWorkspace, StatusWarning, false, "The directory is not initialized as an Engineering Toolkit workspace.", ".engkit", nil, "Review engkit workspace plan-init before approving initialization."))
		findings = append(findings, skippedWorkspaceFindings()...)
		validation, err := skills.Validate(filepath.FromSlash(inspection.Root), "")
		findings = append(findings, skillFinding(validation, err))
		return findings
	}

	findings = append(findings, finding("workspace.initialized", ScopeWorkspace, StatusPass, true, "The workspace contains Engineering Toolkit state.", ".engkit", nil, ""))
	findings = append(findings, inspectConfiguration(filepath.FromSlash(inspection.Root)))
	validation, err := skills.Validate(filepath.FromSlash(inspection.Root), "")
	findings = append(findings, skillFinding(validation, err))
	findings = append(findings, inspectContext(ctx, filepath.FromSlash(inspection.Root)))
	findings = append(findings, inspectLocks(filepath.FromSlash(inspection.Root)))
	return findings
}

func skippedWorkspaceFindings() []Finding {
	return []Finding{
		finding("workspace.config", ScopeWorkspace, StatusUnavailable, false, "Workspace configuration was not checked.", ".engkit/config.yaml", nil, "Initialize or select a valid Engineering Toolkit workspace."),
		finding("workspace.context", ScopeWorkspace, StatusUnavailable, false, "The optional contextual index was not checked.", ".engkit/context/index.db", nil, "Initialize or select a valid Engineering Toolkit workspace."),
		finding("workspace.transactions", ScopeWorkspace, StatusUnavailable, false, "Transaction locks were not checked.", ".engkit/recovery", nil, "Initialize or select a valid Engineering Toolkit workspace."),
	}
}

type configDocument struct {
	SchemaVersion string           `yaml:"schema_version"`
	Workspace     *workspaceConfig `yaml:"workspace,omitempty"`
}

type workspaceConfig struct {
	SourceLayout string `yaml:"source_layout"`
}

func inspectConfiguration(root string) Finding {
	relative := ".engkit/config.yaml"
	filename := filepath.Join(root, ".engkit", "config.yaml")
	info, err := os.Lstat(filename)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace configuration is missing or is not a regular file.", relative, nil, "Restore a reviewed regular .engkit/config.yaml file.")
	}
	if info.Size() > maxConfigurationBytes {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace configuration exceeds the 1 MiB limit.", relative, nil, "Reduce the configuration to the supported bounded schema.")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace configuration could not be read.", relative, nil, "Check file permissions and restore the configuration from reviewed source.")
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil || yamlContainsAlias(&node) {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace configuration is not supported YAML.", relative, nil, "Remove YAML aliases and fix the document syntax.")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var document configDocument
	if err := decoder.Decode(&document); err != nil {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace configuration does not match the supported schema.", relative, nil, "Remove unknown fields or migrate the configuration to engkit.config/v1alpha1.")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace configuration must contain exactly one YAML document.", relative, nil, "Keep one engkit.config/v1alpha1 document.")
	}
	if document.SchemaVersion != "engkit.config/v1alpha1" {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace configuration uses an unsupported schema.", relative, nil, "Migrate the configuration to engkit.config/v1alpha1.")
	}
	if document.Workspace != nil && document.Workspace.SourceLayout != "" && document.Workspace.SourceLayout != "auto" {
		return finding("workspace.config", ScopeWorkspace, StatusError, true, "Workspace source layout is unsupported.", relative, nil, "Use workspace.source_layout: auto in the current schema.")
	}
	return finding("workspace.config", ScopeWorkspace, StatusPass, true, "Workspace configuration passed schema validation.", relative, []Evidence{{Name: "schema_version", Value: document.SchemaVersion}}, "")
}

func yamlContainsAlias(node *yaml.Node) bool {
	if node.Kind == yaml.AliasNode {
		return true
	}
	for _, child := range node.Content {
		if yamlContainsAlias(child) {
			return true
		}
	}
	return false
}

func skillFinding(validation skills.Validation, err error) Finding {
	if err != nil {
		return finding("workspace.skills", ScopeWorkspace, StatusError, true, "Canonical skills could not be inspected.", ".agents/skills", nil, "Resolve the reported .agents/skills path or permission problem.")
	}
	if !validation.Valid {
		return finding("workspace.skills", ScopeWorkspace, StatusError, true, "One or more canonical skill packages are invalid or conflicting.", ".agents/skills", []Evidence{{Name: "packages", Value: fmt.Sprint(len(validation.Packages))}}, "Run engkit skills validate and resolve every reported diagnostic.")
	}
	return finding("workspace.skills", ScopeWorkspace, StatusPass, true, "Canonical skill packages passed validation.", ".agents/skills", []Evidence{{Name: "packages", Value: fmt.Sprint(len(validation.Packages))}}, "")
}

func inspectContext(ctx context.Context, root string) Finding {
	status, err := sqliteindex.Inspect(ctx, root)
	if err != nil {
		return finding("workspace.context", ScopeWorkspace, StatusWarning, false, "The optional contextual index could not be inspected.", ".engkit/context/index.db", nil, "Review the index state and run engkit context rebuild if canonical files are valid.")
	}
	if !status.Exists {
		return finding("workspace.context", ScopeWorkspace, StatusUnavailable, false, "The optional contextual index has not been built.", status.DatabasePath, nil, "Run engkit context rebuild when indexed search is desired.")
	}
	evidence := []Evidence{{Name: "artifacts", Value: fmt.Sprint(status.ArtifactCount)}, {Name: "rebuilt_at", Value: status.RebuiltAt}}
	if !status.Fresh {
		return finding("workspace.context", ScopeWorkspace, StatusWarning, false, "The optional contextual index is stale.", status.DatabasePath, evidence, "Run engkit context rebuild before relying on indexed search.")
	}
	return finding("workspace.context", ScopeWorkspace, StatusPass, false, "The optional contextual index matches canonical sources.", status.DatabasePath, evidence, "")
}

func inspectLocks(root string) Finding {
	paths, err := transactionLocks(root)
	if err != nil {
		return finding("workspace.transactions", ScopeWorkspace, StatusError, true, "Transaction recovery state could not be inspected safely.", ".engkit/recovery", nil, "Inspect linked or unreadable recovery paths manually; do not delete transaction evidence.")
	}
	if len(paths) != 0 {
		evidence := make([]Evidence, 0, len(paths))
		for _, path := range paths {
			evidence = append(evidence, Evidence{Name: "lock", Value: path})
		}
		return finding("workspace.transactions", ScopeWorkspace, StatusError, true, "One or more transaction locks require review or explicit recovery.", ".engkit/recovery", evidence, "Use the matching engkit recovery command after reviewing the persisted plan and evidence.")
	}
	return finding("workspace.transactions", ScopeWorkspace, StatusPass, true, "No active Engineering Toolkit transaction lock was found.", ".engkit/recovery", nil, "")
}

func transactionLocks(root string) ([]string, error) {
	locks := []string{}
	recovery := filepath.Join(root, ".engkit", "recovery")
	if err := walkRealTree(recovery, func(path string, entry os.DirEntry) error {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".lock") {
			locks = append(locks, relativePath(root, path))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	parent := filepath.Dir(root)
	createLock := filepath.Join(parent, "."+filepath.Base(root)+".engkit-create.lock")
	if info, err := os.Lstat(createLock); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, errors.New("project creation lock is not a real directory")
		}
		locks = append(locks, filepath.ToSlash(createLock))
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	sort.Strings(locks)
	return locks, nil
}

func walkRealTree(root string, visit func(string, os.DirEntry) error) error {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("diagnostic root is not a real directory")
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entryInfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("linked diagnostic path %s is not allowed", path)
		}
		return visit(path, entry)
	})
}

func inspectIntegrations(state workspaceState, lookPath func(string) (string, error)) []Finding {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	findings := []Finding{}
	if path, err := lookPath("ollama"); err == nil {
		findings = append(findings, finding("integration.ollama.executable", ScopeIntegrations, StatusPass, false, "Ollama is available in PATH.", "", []Evidence{{Name: "executable", Value: filepath.ToSlash(path)}}, ""))
	} else {
		findings = append(findings, finding("integration.ollama.executable", ScopeIntegrations, StatusUnavailable, false, "Ollama is not available in PATH.", "", nil, "Install Ollama independently or add the existing executable to PATH before enabling local intelligence."))
	}
	findings = append(findings, finding("integration.mcp.server", ScopeIntegrations, StatusPass, false, "The read-only MCP stdio server is available.", "", []Evidence{{Name: "transport", Value: "stdio"}, {Name: "authority", Value: "read-only"}}, ""))
	if !state.available || !state.inspection.HasEngkit {
		findings = append(findings, finding("integration.provider-ownership", ScopeIntegrations, StatusUnavailable, false, "Provider ownership records were not checked.", ".engkit/providers", nil, "Select an initialized Engineering Toolkit workspace."))
		return findings
	}
	findings = append(findings, inspectProviderOwnership(filepath.FromSlash(state.inspection.Root))...)
	return findings
}

func inspectProviderOwnership(root string) []Finding {
	providerRoot := filepath.Join(root, ".engkit", "providers")
	records := 0
	invalid := []Finding{}
	err := walkRealTree(providerRoot, func(filename string, entry os.DirEntry) error {
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) != ".json" {
			invalid = append(invalid, invalidOwnership(root, filename, "Provider ownership state contains an unexpected file."))
			return nil
		}
		records++
		if err := validateOwnership(root, providerRoot, filename); err != nil {
			invalid = append(invalid, invalidOwnership(root, filename, "Provider ownership record is invalid."))
		}
		return nil
	})
	if err != nil {
		return []Finding{finding("integration.provider-ownership", ScopeIntegrations, StatusError, true, "Provider ownership state could not be inspected safely.", ".engkit/providers", nil, "Resolve linked, special, or unreadable ownership paths without adopting unknown files.")}
	}
	if len(invalid) != 0 {
		return invalid
	}
	return []Finding{finding("integration.provider-ownership", ScopeIntegrations, StatusPass, true, "Provider ownership records passed structural validation.", ".engkit/providers", []Evidence{{Name: "records", Value: fmt.Sprint(records)}}, "")}
}

func invalidOwnership(root, filename, summary string) Finding {
	return finding("integration.provider-ownership.invalid", ScopeIntegrations, StatusError, true, summary, relativePath(root, filename), nil, "Review the record and matching provider output; use explicit recovery or unsync rather than deleting evidence blindly.")
}

func validateOwnership(root, providerRoot, filename string) error {
	info, err := os.Lstat(filename)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maxOwnershipBytes {
		return errors.New("ownership record must be a bounded regular file")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record providersync.OwnershipRecord
	if err := decoder.Decode(&record); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("ownership record contains trailing content")
	}
	relative, err := filepath.Rel(providerRoot, filename)
	if err != nil {
		return err
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) != 2 || record.SchemaVersion != providersync.OwnershipSchema || record.Provider != parts[0] || record.SkillID != strings.TrimSuffix(parts[1], ".json") || record.AdapterVersion == "" || record.SkillVersion == "" || !validSHA256(record.PackageSHA256) || !validSHA256(record.SynchronizationID) || len(record.Outputs) == 0 {
		return errors.New("ownership record identity is inconsistent")
	}
	seen := map[string]bool{}
	for _, output := range record.Outputs {
		if output.Path == "" || strings.Contains(output.Path, "\\") || filepath.IsAbs(output.Path) || !validSHA256(output.SHA256) || seen[output.Path] {
			return errors.New("ownership output is invalid")
		}
		target := filepath.Clean(filepath.Join(root, filepath.FromSlash(output.Path)))
		targetRelative, err := filepath.Rel(root, target)
		if err != nil || targetRelative == ".." || strings.HasPrefix(targetRelative, ".."+string(filepath.Separator)) {
			return errors.New("ownership output escapes the workspace")
		}
		seen[output.Path] = true
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range strings.TrimPrefix(value, "sha256:") {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func relativePath(root, filename string) string {
	relative, err := filepath.Rel(root, filename)
	if err != nil {
		return filepath.ToSlash(filename)
	}
	return filepath.ToSlash(relative)
}

func finding(code, scope, status string, required bool, summary, path string, evidence []Evidence, remediation string) Finding {
	return Finding{Code: code, Scope: scope, Status: status, Required: required, Summary: summary, Path: path, Evidence: evidence, Remediation: remediation}
}

func summarize(findings []Finding) Summary {
	summary := Summary{Healthy: true}
	for _, finding := range findings {
		switch finding.Status {
		case StatusPass:
			summary.Pass++
		case StatusWarning:
			summary.Warning++
		case StatusError:
			summary.Error++
			if finding.Required {
				summary.RequiredErrors++
			}
		case StatusUnavailable:
			summary.Unavailable++
		}
	}
	summary.Healthy = summary.RequiredErrors == 0
	return summary
}
