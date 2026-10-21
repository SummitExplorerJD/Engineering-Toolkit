// Package compositionexec applies and recovers approved project compositions.
package compositionexec

import (
	"errors"
	"fmt"
	"time"
)

// ErrNoReplaceUnsupported reports a filesystem that cannot atomically publish
// a staged file without replacing an existing target.
var ErrNoReplaceUnsupported = errors.New("atomic no-replace publication is unsupported")

const (
	JournalSchema    = "engkit.journal-event/v1alpha1"
	CheckpointSchema = "engkit.composition-checkpoint/v1alpha1"
	ResultSchema     = "engkit.composition-execution-result/v1alpha1"
	LockSchema       = "engkit.composition-lock/v1alpha1"
)

type Options struct {
	Now       func() time.Time
	FailAt    func(point string) error
	ProcessID int
}

type Result struct {
	SchemaVersion  string `json:"schema_version"`
	Root           string `json:"root"`
	PlanID         string `json:"plan_id"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	PlanPath       string `json:"plan_path"`
	JournalPath    string `json:"journal_path"`
	CheckpointPath string `json:"checkpoint_path"`
	RetainedPath   string `json:"retained_path,omitempty"`
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
	SchemaVersion   string   `json:"schema_version"`
	PlanID          string   `json:"plan_id"`
	Kind            string   `json:"kind"`
	TargetRoot      string   `json:"target_root"`
	Status          string   `json:"status"`
	PlanPath        string   `json:"plan_path"`
	JournalPath     string   `json:"journal_path"`
	StagingPath     string   `json:"staging_path"`
	BackupPath      string   `json:"backup_path,omitempty"`
	LockPath        string   `json:"lock_path"`
	TargetBefore    string   `json:"target_before"`
	AppliedOutputs  []string `json:"applied_outputs"`
	CreatedDirs     []string `json:"created_directories"`
	JournalSequence int      `json:"journal_sequence"`
	UpdatedAt       string   `json:"updated_at"`
}

type transaction struct {
	root, parent, key, stage, backup, lock string
	planPath, journalPath, checkpointPath  string
	checkpoint                             checkpoint
	now                                    func() time.Time
	failAt                                 func(string) error
}

func normalizeOptions(options Options) Options {
	if options.Now == nil {
		options.Now = time.Now
	}
	return options
}

func (tx *transaction) fail(point string) error {
	if tx.failAt == nil {
		return nil
	}
	if err := tx.failAt(point); err != nil {
		return fmt.Errorf("injected failure at %s: %w", point, err)
	}
	return nil
}
