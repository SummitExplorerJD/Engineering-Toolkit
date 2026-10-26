// Package planinspect safely summarizes supported persisted plan documents.
package planinspect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/composition"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillplan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/speckit"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	Schema       = "engkit.plan-inspection/v1alpha1"
	MaxPathBytes = 512
	MaxPlanBytes = 1 << 20
	maxEntries   = 100
	maxEntrySize = 1024
)

type Inspection struct {
	SchemaVersion    string   `json:"schema_version"`
	Root             string   `json:"root"`
	Path             string   `json:"path"`
	PlanSchema       string   `json:"plan_schema"`
	Operation        string   `json:"operation"`
	PlanID           string   `json:"plan_id"`
	ApprovalRequired bool     `json:"approval_required"`
	Executable       *bool    `json:"executable,omitempty"`
	IdentityValid    bool     `json:"identity_valid"`
	OperationCount   int      `json:"operation_count"`
	Verification     []string `json:"verification"`
	Recovery         []string `json:"recovery"`
}

type header struct {
	SchemaVersion string `json:"schema_version"`
}

// Inspect reads one bounded plan below .engkit/plans without modifying it.
func Inspect(root, relative string) (Inspection, error) {
	resolvedRoot, err := workspace.ValidateRoot(root)
	if err != nil {
		return Inspection{}, err
	}
	if err := validateRelative(relative); err != nil {
		return Inspection{}, err
	}
	filename := filepath.Join(filepath.FromSlash(resolvedRoot), filepath.FromSlash(relative))
	if err := rejectLinkedPath(filepath.FromSlash(resolvedRoot), filename); err != nil {
		return Inspection{}, err
	}
	info, err := os.Stat(filename)
	if err != nil {
		return Inspection{}, errors.New("plan file could not be inspected")
	}
	if !info.Mode().IsRegular() {
		return Inspection{}, errors.New("plan path must name a regular file")
	}
	if info.Size() > MaxPlanBytes {
		return Inspection{}, errors.New("plan file exceeds the 1 MiB limit")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return Inspection{}, errors.New("plan file could not be read")
	}
	var h header
	if err := json.Unmarshal(data, &h); err != nil || h.SchemaVersion == "" {
		return Inspection{}, errors.New("plan is not supported JSON")
	}

	result := Inspection{SchemaVersion: Schema, Root: resolvedRoot, Path: relative, PlanSchema: h.SchemaVersion, IdentityValid: true, Verification: []string{}, Recovery: []string{}}
	switch h.SchemaVersion {
	case plan.PlanSchema:
		var value plan.Plan
		if err := decodeStrict(data, &value); err != nil {
			return Inspection{}, err
		}
		if err := plan.ValidateIdentity(value); err != nil {
			return Inspection{}, errors.New("plan identity is invalid")
		}
		if err := requireRoot(resolvedRoot, value.Root); err != nil {
			return Inspection{}, err
		}
		result.Operation, result.PlanID, result.ApprovalRequired = value.Kind, value.PlanID, value.RequiresApproval
		result.OperationCount, result.Verification = len(value.Operations), value.Verification
	case skillplan.PlanSchema:
		var value skillplan.Plan
		if err := decodeStrict(data, &value); err != nil {
			return Inspection{}, err
		}
		if err := skillplan.ValidateIdentity(value); err != nil {
			return Inspection{}, errors.New("plan identity is invalid")
		}
		if err := requireRoot(resolvedRoot, value.Root); err != nil {
			return Inspection{}, err
		}
		result.Operation, result.PlanID, result.ApprovalRequired = value.Kind, value.PlanID, value.RequiresApproval
		result.OperationCount, result.Verification, result.Recovery = len(value.Operations), value.Verification, value.Recovery
		result.Executable = boolPointer(value.Executable)
	case composition.PlanSchema:
		var value composition.Plan
		if err := decodeStrict(data, &value); err != nil {
			return Inspection{}, err
		}
		if err := composition.ValidateIdentity(value); err != nil {
			return Inspection{}, errors.New("plan identity is invalid")
		}
		if err := requireRoot(resolvedRoot, value.TargetRoot); err != nil {
			return Inspection{}, err
		}
		result.Operation, result.PlanID, result.ApprovalRequired = value.Kind, value.PlanID, value.RequiresApproval
		result.OperationCount, result.Verification, result.Recovery = len(value.Operations), value.Verification, value.Recovery
		result.Executable = boolPointer(value.Executable)
	case providersync.PlanSchema:
		var value providersync.Plan
		if err := decodeStrict(data, &value); err != nil {
			return Inspection{}, err
		}
		if err := providersync.ValidateIdentity(value); err != nil {
			return Inspection{}, errors.New("plan identity is invalid")
		}
		if err := requireRoot(resolvedRoot, value.Root); err != nil {
			return Inspection{}, err
		}
		result.Operation, result.PlanID, result.ApprovalRequired = value.Operation, value.PlanID, value.RequiresApproval
		result.OperationCount, result.Verification = len(value.Outputs), value.Verification
		result.Executable = boolPointer(value.Executable)
	case speckit.PlanSchema:
		var value speckit.Plan
		if err := decodeStrict(data, &value); err != nil {
			return Inspection{}, err
		}
		if err := speckit.ValidateIdentity(value); err != nil {
			return Inspection{}, errors.New("plan identity is invalid")
		}
		if err := requireRoot(resolvedRoot, value.Root); err != nil {
			return Inspection{}, err
		}
		result.Operation, result.PlanID, result.ApprovalRequired = value.Kind, value.PlanID, value.RequiresApproval
		result.OperationCount, result.Verification, result.Recovery = len(value.PotentialPaths), value.RequiredReviews, value.RemovalStrategy
		result.Executable = boolPointer(value.Executable)
	default:
		return Inspection{}, errors.New("plan schema is not supported")
	}
	if err := validateEntries(result.Verification, result.Recovery); err != nil {
		return Inspection{}, err
	}
	return result, nil
}

func validateRelative(value string) error {
	if value == "" || len([]byte(value)) > MaxPathBytes || strings.Contains(value, "\\") || path.IsAbs(value) || path.Clean(value) != value || !strings.HasPrefix(value, ".engkit/plans/") || path.Ext(value) != ".json" {
		return errors.New("plan path must be a normalized .engkit/plans relative JSON path")
	}
	return nil
}

func rejectLinkedPath(root, filename string) error {
	for current := filename; !strings.EqualFold(current, root); current = filepath.Dir(current) {
		if !strings.HasPrefix(strings.ToLower(current), strings.ToLower(root+string(os.PathSeparator))) {
			return errors.New("plan path escapes the workspace")
		}
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return errors.New("plan path could not be inspected")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("plan path must not use symbolic links or junctions")
		}
	}
	return nil
}

func decodeStrict(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errors.New("plan does not match its supported schema")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("plan must contain exactly one JSON document")
	}
	return nil
}

func requireRoot(bound, recorded string) error {
	want := filepath.Clean(filepath.FromSlash(bound))
	got := filepath.Clean(filepath.FromSlash(recorded))
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("plan root does not match the bound workspace")
	}
	return nil
}

func validateEntries(groups ...[]string) error {
	for _, entries := range groups {
		if len(entries) > maxEntries {
			return errors.New("plan summary exceeds the 100-entry limit")
		}
		for _, entry := range entries {
			if len([]byte(entry)) > maxEntrySize {
				return errors.New("plan summary entry exceeds the 1024-byte limit")
			}
		}
	}
	return nil
}

func boolPointer(value bool) *bool { return &value }
