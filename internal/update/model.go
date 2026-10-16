package update

import (
	"context"
	"fmt"
	"time"
)

const (
	StableChannel       = "stable"
	TargetSchema        = "engkit.release-target/v1alpha1"
	CheckSchema         = "engkit.update-check/v1alpha1"
	DownloadSchema      = "engkit.update-download/v1alpha1"
	ExecutableFormatRaw = "executable"
)

type Repository struct {
	MetadataURL string
	TargetsURL  string
	Root        []byte
}

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type TrustedState struct {
	RootVersion      int64     `json:"root_version"`
	TimestampVersion int64     `json:"timestamp_version"`
	SnapshotVersion  int64     `json:"snapshot_version"`
	TargetsVersion   int64     `json:"targets_version"`
	TimestampExpires time.Time `json:"timestamp_expires"`
	SnapshotExpires  time.Time `json:"snapshot_expires"`
	TargetsExpires   time.Time `json:"targets_expires"`
}

type Target struct {
	Version  string         `json:"version"`
	Channel  string         `json:"channel"`
	Platform Platform       `json:"platform"`
	Path     string         `json:"target_path"`
	Length   int64          `json:"length"`
	SHA256   string         `json:"sha256"`
	Format   string         `json:"format"`
	Custom   map[string]any `json:"-"`
}

type VerifiedFile struct {
	Path       string    `json:"path"`
	Length     int64     `json:"length"`
	SHA256     string    `json:"sha256"`
	VerifiedAt time.Time `json:"verified_at"`
}

type Client interface {
	Refresh(context.Context) (TrustedState, error)
	Target(context.Context, string) (Target, error)
	Download(context.Context, Target, string) (VerifiedFile, error)
}

type ErrorCategory string

const (
	ErrorUpdateNotConfigured ErrorCategory = "update_not_configured"
	ErrorTransportFailed     ErrorCategory = "transport_failed"
	ErrorTrustFailed         ErrorCategory = "trust_failed"
	ErrorMetadataExpired     ErrorCategory = "metadata_expired"
	ErrorRollbackDetected    ErrorCategory = "rollback_detected"
	ErrorTargetIncompatible  ErrorCategory = "target_incompatible"
	ErrorTargetInvalid       ErrorCategory = "target_invalid"
	ErrorCacheFailed         ErrorCategory = "cache_failed"
)

type Error struct {
	Category ErrorCategory
	Op       string
	Err      error
}

func (e *Error) Error() string {
	if e.Op == "" {
		return fmt.Sprintf("%s: %v", e.Category, e.Err)
	}
	return fmt.Sprintf("%s: %s: %v", e.Category, e.Op, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }
