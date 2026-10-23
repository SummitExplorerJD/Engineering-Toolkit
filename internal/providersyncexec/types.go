// Package providersyncexec applies and recovers exactly approved provider
// synchronization plans without invoking provider runtimes.
package providersyncexec

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
)

var ErrNoReplaceUnsupported = errors.New("atomic no-replace publication is unsupported")

const (
	JournalSchema    = "engkit.journal-event/v1alpha1"
	CheckpointSchema = "engkit.provider-sync-checkpoint/v1alpha1"
	ResultSchema     = "engkit.provider-sync-execution-result/v1alpha1"
	LockSchema       = "engkit.provider-sync-lock/v1alpha1"
)

type Options struct {
	Now       func() time.Time
	FailAt    func(point string) error
	ProcessID int
	Link      func(oldname, newname string) error
}

type Result struct {
	SchemaVersion  string   `json:"schema_version"`
	Root           string   `json:"root"`
	PlanID         string   `json:"plan_id"`
	Operation      string   `json:"operation"`
	Provider       string   `json:"provider"`
	SkillID        string   `json:"skill_id"`
	Status         string   `json:"status"`
	ChangedPaths   []string `json:"changed_paths"`
	PlanPath       string   `json:"plan_path"`
	JournalPath    string   `json:"journal_path"`
	CheckpointPath string   `json:"checkpoint_path"`
	RecoveryPath   string   `json:"recovery_path"`
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

type outputState struct {
	Path         string `json:"path"`
	Action       string `json:"action"`
	BeforeSHA256 string `json:"before_sha256,omitempty"`
	AfterSHA256  string `json:"after_sha256,omitempty"`
	StagingPath  string `json:"staging_path,omitempty"`
	BackupPath   string `json:"backup_path,omitempty"`
	Published    bool   `json:"published"`
}

type checkpoint struct {
	SchemaVersion       string        `json:"schema_version"`
	PlanID              string        `json:"plan_id"`
	Operation           string        `json:"operation"`
	Provider            string        `json:"provider"`
	SkillID             string        `json:"skill_id"`
	Status              string        `json:"status"`
	PlanPath            string        `json:"plan_path"`
	JournalPath         string        `json:"journal_path"`
	RecoveryPath        string        `json:"recovery_path"`
	OwnershipPath       string        `json:"ownership_path"`
	OwnershipStaging    string        `json:"ownership_staging_path,omitempty"`
	OwnershipBackup     string        `json:"ownership_backup_path"`
	OwnershipBeforeHash string        `json:"ownership_before_sha256,omitempty"`
	OwnershipAfterHash  string        `json:"ownership_after_sha256,omitempty"`
	OwnershipPublished  bool          `json:"ownership_published"`
	NextOutput          int           `json:"next_output"`
	Outputs             []outputState `json:"outputs"`
	CreatedDirectories  []string      `json:"created_directories"`
	JournalSequence     int           `json:"journal_sequence"`
	UpdatedAt           string        `json:"updated_at"`
}

type transaction struct {
	root, engkit, key, base, skillLock, providerLock string
	planPath, journalPath, checkpointPath            string
	ownership, stagedOwnership, backupOwnership      string
	checkpoint                                       checkpoint
	now                                              func() time.Time
	failAt                                           func(string) error
	link                                             func(string, string) error
}

func normalizeOptions(options Options) Options {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Link == nil {
		options.Link = os.Link
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

func operationSkill(plan providersync.Plan) (string, error) {
	if plan.SkillID == "" {
		return "", fmt.Errorf("provider plan must select exactly one skill")
	}
	return plan.SkillID, nil
}
