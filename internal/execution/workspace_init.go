// Package execution applies previously approved plans.
package execution

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/approval"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	journalSchema     = "engkit.journal-event/v1alpha1"
	checkpointSchema  = "engkit.checkpoint/v1alpha1"
	configContents    = "schema_version: engkit.config/v1alpha1\nworkspace:\n  source_layout: auto\n"
	gitignoreContents = "/cache/*\n/context/*.db\n/context/*.db-*\n/recovery/*\n"
)

// Result identifies the canonical artifacts published by initialization.
type Result struct {
	SchemaVersion  string `json:"schema_version"`
	Root           string `json:"root"`
	PlanID         string `json:"plan_id"`
	PlanPath       string `json:"plan_path"`
	JournalPath    string `json:"journal_path"`
	CheckpointPath string `json:"checkpoint_path"`
	Status         string `json:"status"`
}

type journalEvent struct {
	SchemaVersion string         `json:"schema_version"`
	EventID       string         `json:"event_id"`
	PlanID        string         `json:"plan_id"`
	Sequence      int            `json:"sequence"`
	Timestamp     string         `json:"timestamp"`
	Type          string         `json:"type"`
	Payload       map[string]any `json:"payload"`
}

type checkpoint struct {
	SchemaVersion   string `json:"schema_version"`
	PlanID          string `json:"plan_id"`
	Status          string `json:"status"`
	JournalSequence int    `json:"journal_sequence"`
	UpdatedAt       string `json:"updated_at"`
}

// Initialize publishes .engkit atomically after revalidating approval and state.
func Initialize(preview plan.Plan, approvedPlanID string, now func() time.Time) (Result, error) {
	if err := approval.Validate(approvedPlanID, preview); err != nil {
		return Result{}, err
	}
	if now == nil {
		now = time.Now
	}

	fresh, err := workspace.Inspect(filepath.FromSlash(preview.Root))
	if err != nil {
		return Result{}, fmt.Errorf("reinspect workspace: %w", err)
	}
	if fresh.HasEngkit {
		return Result{}, errors.New("workspace is already initialized")
	}
	freshPlan, err := plan.NewWorkspaceInit(fresh)
	if err != nil {
		return Result{}, fmt.Errorf("rebuild workspace plan: %w", err)
	}
	if freshPlan.PlanID != preview.PlanID {
		return Result{}, errors.New("workspace changed after planning; generate and approve a new plan")
	}

	root := filepath.FromSlash(preview.Root)
	target := filepath.Join(root, ".engkit")
	staging, err := os.MkdirTemp(root, "engkit-staging-")
	if err != nil {
		return Result{}, fmt.Errorf("create initialization staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()

	for _, relative := range []string{"plans", "journals", "checkpoints", "context", "cache", "recovery"} {
		if err := os.Mkdir(filepath.Join(staging, relative), 0o700); err != nil {
			return Result{}, fmt.Errorf("create staging directory %s: %w", relative, err)
		}
	}

	if err := writeNewSynced(filepath.Join(staging, ".gitignore"), []byte(gitignoreContents)); err != nil {
		return Result{}, err
	}
	if err := writeNewSynced(filepath.Join(staging, "config.yaml"), []byte(configContents)); err != nil {
		return Result{}, err
	}
	planBytes, err := json.MarshalIndent(preview, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode approved plan: %w", err)
	}
	planBytes = append(planBytes, '\n')
	if err := writeNewSynced(filepath.Join(staging, "plans", "workspace-init.json"), planBytes); err != nil {
		return Result{}, err
	}

	timestamp := now().UTC().Format(time.RFC3339Nano)
	events := []journalEvent{
		newEvent(preview.PlanID, 1, timestamp, "approval.accepted", map[string]any{"approved_plan_id": approvedPlanID}),
		newEvent(preview.PlanID, 2, timestamp, "workspace.initialization_started", map[string]any{"operation_count": len(preview.Operations)}),
		newEvent(preview.PlanID, 3, timestamp, "workspace.initialization_completed", map[string]any{"status": "complete"}),
	}
	journalPath := filepath.Join(staging, "journals", "workspace-init.jsonl")
	if err := writeJournal(journalPath, events); err != nil {
		return Result{}, err
	}

	checkpointBytes, err := json.MarshalIndent(checkpoint{
		SchemaVersion:   checkpointSchema,
		PlanID:          preview.PlanID,
		Status:          "complete",
		JournalSequence: len(events),
		UpdatedAt:       timestamp,
	}, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode checkpoint: %w", err)
	}
	checkpointBytes = append(checkpointBytes, '\n')
	if err := writeNewSynced(filepath.Join(staging, "checkpoints", "workspace-init.json"), checkpointBytes); err != nil {
		return Result{}, err
	}

	if err := os.Rename(staging, target); err != nil {
		return Result{}, fmt.Errorf("publish initialized workspace: %w", err)
	}
	committed = true

	return Result{
		SchemaVersion:  "engkit.workspace-init-result/v1alpha1",
		Root:           preview.Root,
		PlanID:         preview.PlanID,
		PlanPath:       ".engkit/plans/workspace-init.json",
		JournalPath:    ".engkit/journals/workspace-init.jsonl",
		CheckpointPath: ".engkit/checkpoints/workspace-init.json",
		Status:         "complete",
	}, nil
}

func newEvent(planID string, sequence int, timestamp, eventType string, payload map[string]any) journalEvent {
	return journalEvent{
		SchemaVersion: journalSchema,
		EventID:       fmt.Sprintf("%s:%06d", planID, sequence),
		PlanID:        planID,
		Sequence:      sequence,
		Timestamp:     timestamp,
		Type:          eventType,
		Payload:       payload,
	}
}

func writeNewSynced(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(contents); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	remove = false
	return nil
}

func writeJournal(path string, events []journalEvent) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create journal: %w", err)
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("encode journal event: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush journal: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync journal: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close journal: %w", err)
	}
	remove = false
	return nil
}
