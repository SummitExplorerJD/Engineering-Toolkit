package speckit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/adapters"
)

const PlanSchema = "engkit.speckit-plan/v1alpha1"

var releasePattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

// PathImpact is a conservative root-level preview. A release-derived exact
// inventory is still required before execution.
type PathImpact struct {
	Path        string `json:"path"`
	Ownership   string `json:"ownership"`
	State       string `json:"state"`
	Disposition string `json:"disposition"`
}

// Plan is immutable preparation data. This package intentionally has no Apply.
type Plan struct {
	SchemaVersion    string       `json:"schema_version"`
	Kind             string       `json:"kind"`
	PlanID           string       `json:"plan_id"`
	Root             string       `json:"root"`
	Integration      string       `json:"integration"`
	UpstreamVersion  string       `json:"upstream_version"`
	Source           string       `json:"source"`
	Command          []string     `json:"command"`
	PotentialPaths   []PathImpact `json:"potential_paths"`
	RequiredReviews  []string     `json:"required_reviews"`
	RemovalStrategy  []string     `json:"removal_strategy"`
	RequiresApproval bool         `json:"requires_approval"`
	Executable       bool         `json:"executable"`
}

type digestBody struct {
	SchemaVersion    string       `json:"schema_version"`
	Kind             string       `json:"kind"`
	Root             string       `json:"root"`
	Integration      string       `json:"integration"`
	UpstreamVersion  string       `json:"upstream_version"`
	Source           string       `json:"source"`
	Command          []string     `json:"command"`
	PotentialPaths   []PathImpact `json:"potential_paths"`
	RequiredReviews  []string     `json:"required_reviews"`
	RemovalStrategy  []string     `json:"removal_strategy"`
	RequiresApproval bool         `json:"requires_approval"`
	Executable       bool         `json:"executable"`
}

// NewPlan creates a deterministic, non-executable Spec Kit preparation plan.
func NewPlan(root, provider, version string) (Plan, error) {
	version = strings.TrimSpace(version)
	if !releasePattern.MatchString(version) {
		return Plan{}, errors.New("upstream version must be an explicit release tag such as v0.12.11")
	}
	adapter, err := adapters.Get(provider)
	if err != nil {
		return Plan{}, err
	}
	absRoot, err := resolveRoot(root)
	if err != nil {
		return Plan{}, err
	}

	paths := []struct {
		path      string
		ownership string
	}{
		{path: ".specify", ownership: "spec_kit"},
		{path: "specs", ownership: "shared"},
		{path: adapter.SpecKitArtifactRoot, ownership: "provider"},
		{path: "AGENTS.md", ownership: "engkit"},
	}
	seen := make(map[string]struct{})
	impacts := make([]PathImpact, 0, len(paths))
	for _, candidate := range paths {
		if _, ok := seen[candidate.path]; ok {
			continue
		}
		seen[candidate.path] = struct{}{}
		state, err := classify(filepath.Join(absRoot, filepath.FromSlash(candidate.path)))
		if err != nil {
			return Plan{}, err
		}
		disposition := "create_if_absent"
		if candidate.path == "AGENTS.md" {
			disposition = "preserve_byte_for_byte"
		} else if state != "missing" {
			disposition = "review_exact_diff_before_execution"
		}
		impacts = append(impacts, PathImpact{Path: candidate.path, Ownership: candidate.ownership, State: state, Disposition: disposition})
	}
	sort.Slice(impacts, func(i, j int) bool { return impacts[i].Path < impacts[j].Path })

	source := "git+https://github.com/github/spec-kit.git@" + version
	body := digestBody{
		SchemaVersion:   PlanSchema,
		Kind:            "speckit.prepare",
		Root:            filepath.ToSlash(absRoot),
		Integration:     adapter.SpecKitIntegration,
		UpstreamVersion: version,
		Source:          source,
		Command: []string{
			"uvx", "--python", "3.11", "--from", source, "specify", "init", "--here", "--force",
			"--non-interactive", "--ignore-agent-tools", "--integration", adapter.SpecKitIntegration, "--script", "ps",
		},
		PotentialPaths: impacts,
		RequiredReviews: []string{
			"verify the pinned release tag and installation provenance",
			"verify Python 3.11 or newer is available to uvx",
			"confirm --ignore-agent-tools is appropriate because the agent runs outside the initialization environment",
			"run initialization in an isolated temporary copy to capture the exact generated-file inventory",
			"compare every generated file with repository-owned and provider-owned paths",
			"inspect generated PowerShell scripts before running them",
			"obtain a new approval bound to the exact generated inventory and diff",
		},
		RemovalStrategy: []string{
			"record the exact files created by the pinned release",
			"remove only files proven to be owned by that Spec Kit installation",
			"preserve pre-existing files and shared provider directories",
		},
		RequiresApproval: true,
		Executable:       false,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Plan{}, fmt.Errorf("encode Spec Kit plan digest body: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return Plan{
		SchemaVersion: body.SchemaVersion, Kind: body.Kind,
		PlanID: "sha256:" + hex.EncodeToString(digest[:]), Root: body.Root,
		Integration: body.Integration, UpstreamVersion: body.UpstreamVersion,
		Source: body.Source, Command: body.Command, PotentialPaths: body.PotentialPaths,
		RequiredReviews: body.RequiredReviews, RemovalStrategy: body.RemovalStrategy,
		RequiresApproval: body.RequiresApproval, Executable: body.Executable,
	}, nil
}

// ValidateIdentity confirms that the plan ID matches its canonical body.
func ValidateIdentity(value Plan) error {
	body := digestBody{
		SchemaVersion: value.SchemaVersion, Kind: value.Kind, Root: value.Root,
		Integration: value.Integration, UpstreamVersion: value.UpstreamVersion,
		Source: value.Source, Command: value.Command, PotentialPaths: value.PotentialPaths,
		RequiredReviews: value.RequiredReviews, RemovalStrategy: value.RemovalStrategy,
		RequiresApproval: value.RequiresApproval, Executable: value.Executable,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode Spec Kit plan digest body: %w", err)
	}
	digest := sha256.Sum256(encoded)
	if value.PlanID != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("Spec Kit plan ID does not match its canonical body")
	}
	return nil
}

func resolveRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("workspace root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	absRoot = filepath.Clean(absRoot)
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("inspect workspace root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace root is not a directory: %s", absRoot)
	}
	return absRoot, nil
}

func classify(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "missing", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect Spec Kit path %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "symlink", nil
	}
	if info.IsDir() {
		return "directory", nil
	}
	if info.Mode().IsRegular() {
		return "file", nil
	}
	return "other", nil
}
