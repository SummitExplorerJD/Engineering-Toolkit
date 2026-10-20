// Package skillexec applies and recovers approved canonical skill mutations.
package skillexec

import (
	"fmt"
	"time"
)

const (
	JournalSchema    = "engkit.journal-event/v1alpha1"
	CheckpointSchema = "engkit.skill-checkpoint/v1alpha1"
	ResultSchema     = "engkit.skill-execution-result/v1alpha1"
)

// Options provides deterministic time and test-only failure injection.
type Options struct {
	Now       func() time.Time
	FailAt    func(point string) error
	ProcessID int
}

// Result identifies durable artifacts for one mutation or recovery.
type Result struct {
	SchemaVersion  string `json:"schema_version"`
	Root           string `json:"root"`
	PlanID         string `json:"plan_id"`
	Operation      string `json:"operation"`
	SkillID        string `json:"skill_id"`
	Status         string `json:"status"`
	PlanPath       string `json:"plan_path"`
	JournalPath    string `json:"journal_path"`
	CheckpointPath string `json:"checkpoint_path"`
	BackupPath     string `json:"backup_path,omitempty"`
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
	Kind            string `json:"kind"`
	SkillID         string `json:"skill_id"`
	Status          string `json:"status"`
	TargetPath      string `json:"target_path"`
	StagingPath     string `json:"staging_path,omitempty"`
	BackupPath      string `json:"backup_path,omitempty"`
	PlanPath        string `json:"plan_path"`
	JournalPath     string `json:"journal_path"`
	BeforeSHA256    string `json:"before_sha256,omitempty"`
	AfterSHA256     string `json:"after_sha256,omitempty"`
	JournalSequence int    `json:"journal_sequence"`
	UpdatedAt       string `json:"updated_at"`
}

type transaction struct {
	root, engkit, key, target, staging, backup, lock string
	planPath, journalPath, checkpointPath            string
	checkpoint                                       checkpoint
	now                                              func() time.Time
	failAt                                           func(string) error
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

func normalizeOptions(options Options) Options {
	if options.Now == nil {
		options.Now = time.Now
	}
	return options
}
