package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const planSchema = "engkit.plan/v1alpha1"

// Operation describes one proposed filesystem mutation. It is data only.
type Operation struct {
	Action string `json:"action"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Plan is an immutable preview contract. Execution is intentionally separate.
type Plan struct {
	SchemaVersion    string      `json:"schema_version"`
	Kind             string      `json:"kind"`
	PlanID           string      `json:"plan_id"`
	Root             string      `json:"root"`
	RequiresApproval bool        `json:"requires_approval"`
	Operations       []Operation `json:"operations"`
	Verification     []string    `json:"verification"`
}

type digestBody struct {
	SchemaVersion    string      `json:"schema_version"`
	Kind             string      `json:"kind"`
	Root             string      `json:"root"`
	RequiresApproval bool        `json:"requires_approval"`
	Operations       []Operation `json:"operations"`
	Verification     []string    `json:"verification"`
}

// NewWorkspaceInit creates a deterministic preview and never writes to disk.
func NewWorkspaceInit(snapshot workspace.Inspection) (Plan, error) {
	body := digestBody{
		SchemaVersion:    planSchema,
		Kind:             "workspace.init",
		Root:             snapshot.Root,
		RequiresApproval: true,
		Operations:       workspaceInitOperations(snapshot.HasEngkit),
		Verification: []string{
			"confirm all created paths remain within the workspace root",
			"validate generated artifacts against their versioned contracts",
			"inspect the final changed-file list",
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return Plan{}, fmt.Errorf("encode plan digest body: %w", err)
	}
	digest := sha256.Sum256(encoded)

	return Plan{
		SchemaVersion:    body.SchemaVersion,
		Kind:             body.Kind,
		PlanID:           "sha256:" + hex.EncodeToString(digest[:]),
		Root:             body.Root,
		RequiresApproval: body.RequiresApproval,
		Operations:       body.Operations,
		Verification:     body.Verification,
	}, nil
}

func workspaceInitOperations(alreadyInitialized bool) []Operation {
	if alreadyInitialized {
		return []Operation{}
	}

	paths := []string{
		".engkit",
		".engkit/plans",
		".engkit/journals",
		".engkit/checkpoints",
		".engkit/context",
		".engkit/cache",
	}
	operations := make([]Operation, 0, len(paths))
	for _, path := range paths {
		operations = append(operations, Operation{
			Action: "create_directory",
			Path:   filepath.ToSlash(path),
			Reason: "prepare the versioned Engineering Toolkit workspace layout",
		})
	}
	files := []struct {
		path   string
		reason string
	}{
		{path: ".engkit/.gitignore", reason: "exclude rebuildable databases and caches from version control"},
		{path: ".engkit/config.yaml", reason: "create the declarative workspace configuration"},
		{path: ".engkit/plans/workspace-init.json", reason: "preserve the approved initialization plan"},
		{path: ".engkit/journals/workspace-init.jsonl", reason: "record append-only initialization evidence"},
		{path: ".engkit/checkpoints/workspace-init.json", reason: "record resumable initialization state"},
	}
	for _, file := range files {
		operations = append(operations, Operation{
			Action: "write_file",
			Path:   file.path,
			Reason: file.reason,
		})
	}
	return operations
}
