package tufclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	engupdate "github.com/SummitExplorerJD/Engineering-Toolkit/internal/update"
)

// TestStagingRehearsal exercises the complete pre-production trust flow with
// disposable keys and targets. It never contacts a production repository and
// never persists private key material outside the test process.
func TestStagingRehearsal(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	t.Run("supported target inventory", func(t *testing.T) {
		tests := []struct {
			platform engupdate.Platform
			want     string
		}{
			{engupdate.Platform{OS: "windows", Arch: "amd64"}, "releases/stable/v1.2.3/windows-amd64/engkit.exe"},
			{engupdate.Platform{OS: "windows", Arch: "arm64"}, "releases/stable/v1.2.3/windows-arm64/engkit.exe"},
			{engupdate.Platform{OS: "linux", Arch: "amd64"}, "releases/stable/v1.2.3/linux-amd64/engkit"},
			{engupdate.Platform{OS: "linux", Arch: "arm64"}, "releases/stable/v1.2.3/linux-arm64/engkit"},
			{engupdate.Platform{OS: "darwin", Arch: "amd64"}, "releases/stable/v1.2.3/darwin-amd64/engkit"},
			{engupdate.Platform{OS: "darwin", Arch: "arm64"}, "releases/stable/v1.2.3/darwin-arm64/engkit"},
		}
		for _, test := range tests {
			got, err := engupdate.TargetPath(engupdate.StableChannel, "v1.2.3", test.platform)
			if err != nil {
				t.Fatalf("TargetPath(%+v): %v", test.platform, err)
			}
			if got != test.want {
				t.Fatalf("TargetPath(%+v) = %q, want %q", test.platform, got, test.want)
			}
		}
	})

	t.Run("verified refresh and non-executable download", func(t *testing.T) {
		repository := newTestRepository(t, now.Add(24*time.Hour))
		root := t.TempDir()
		client := rehearsalClient(t, repository, root, now)

		state, err := client.Refresh(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if state.RootVersion != 1 || state.TimestampVersion != 1 || state.SnapshotVersion != 1 || state.TargetsVersion != 1 {
			t.Fatalf("unexpected trusted state: %+v", state)
		}

		target, err := client.Target(context.Background(), repository.targetPath)
		if err != nil {
			t.Fatal(err)
		}
		verified, err := client.Download(context.Background(), target, filepath.Join(root, "published"))
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(verified.Path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, repository.target) {
			t.Fatal("verified cache does not contain the signed target bytes")
		}
		info, err := os.Stat(verified.Path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0111 != 0 {
			t.Fatalf("verified staging target is executable: %v", info.Mode())
		}
		digest := sha256.Sum256(repository.root)
		t.Logf("staging bootstrap root sha256=%s", hex.EncodeToString(digest[:]))
		t.Logf("verified target path=%s sha256=%s", target.Path, target.SHA256)
	})

	t.Run("invalid signature preserves last known good metadata", func(t *testing.T) {
		repository := newTestRepository(t, now.Add(24*time.Hour))
		root := t.TempDir()
		metadataDir := filepath.Join(root, "state", "metadata")
		client := rehearsalClient(t, repository, root, now)
		if _, err := client.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(filepath.Join(metadataDir, "timestamp.json"))
		if err != nil {
			t.Fatal(err)
		}
		repository.corruptTimestamp(t)
		_, err = client.Refresh(context.Background())
		assertUpdateError(t, err, engupdate.ErrorTrustFailed)
		after, readErr := os.ReadFile(filepath.Join(metadataDir, "timestamp.json"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !bytes.Equal(before, after) {
			t.Fatal("invalid metadata replaced the last-known-good timestamp")
		}
	})

	t.Run("tampered target is rejected without publication", func(t *testing.T) {
		repository := newTestRepository(t, now.Add(24*time.Hour))
		root := t.TempDir()
		client := rehearsalClient(t, repository, root, now)
		if _, err := client.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
		target, err := client.Target(context.Background(), repository.targetPath)
		if err != nil {
			t.Fatal(err)
		}
		tampered := append([]byte(nil), repository.target...)
		tampered[0] ^= 0xff
		repository.replaceTarget(tampered)
		cache := filepath.Join(root, "published")
		_, err = client.Download(context.Background(), target, cache)
		assertUpdateError(t, err, engupdate.ErrorTrustFailed)
		entries, readErr := os.ReadDir(cache)
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			t.Fatal(readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("tampered target left published files: %v", entries)
		}
	})

	t.Run("expired metadata fails closed", func(t *testing.T) {
		repository := newTestRepository(t, now.Add(-time.Minute))
		client := rehearsalClient(t, repository, t.TempDir(), now)
		_, err := client.Refresh(context.Background())
		assertUpdateError(t, err, engupdate.ErrorMetadataExpired)
	})

	t.Run("timestamp rollback fails closed", func(t *testing.T) {
		repository := newTestRepository(t, now.Add(24*time.Hour))
		repository.advanceTimestamp(t)
		client := rehearsalClient(t, repository, t.TempDir(), now)
		state, err := client.Refresh(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if state.TimestampVersion != 2 {
			t.Fatalf("timestamp version = %d, want 2", state.TimestampVersion)
		}
		repository.rollbackTimestamp()
		_, err = client.Refresh(context.Background())
		assertUpdateError(t, err, engupdate.ErrorRollbackDetected)
	})

	t.Run("root rotation requires old and new thresholds", func(t *testing.T) {
		for _, test := range []struct {
			name       string
			signatures string
			wantError  bool
		}{
			{name: "dual threshold accepted", signatures: "dual"},
			{name: "insufficient threshold rejected", signatures: "insufficient", wantError: true},
		} {
			t.Run(test.name, func(t *testing.T) {
				repository := newTestRepository(t, now.Add(24*time.Hour))
				repository.setRootRotation(t, 2, test.signatures)
				client := rehearsalClient(t, repository, t.TempDir(), now)
				state, err := client.Refresh(context.Background())
				if test.wantError {
					assertUpdateError(t, err, engupdate.ErrorTrustFailed)
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if state.RootVersion != 2 {
					t.Fatalf("root version = %d, want 2", state.RootVersion)
				}
			})
		}
	})
}

func rehearsalClient(t *testing.T, repository *testRepository, root string, now time.Time) *Client {
	t.Helper()
	client, err := New(Config{
		Repository: engupdate.Repository{
			MetadataURL: repository.server.URL + "/metadata",
			TargetsURL:  repository.server.URL + "/targets",
			Root:        repository.root,
		},
		MetadataDir:  filepath.Join(root, "state", "metadata"),
		TargetCache:  filepath.Join(root, "cache", "targets"),
		UserAgent:    "engkit/staging-rehearsal update-client/1",
		RoundTripper: repository.server.Client().Transport,
		Now:          func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertUpdateError(t *testing.T, err error, want engupdate.ErrorCategory) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error", want)
	}
	var updateError *engupdate.Error
	if !errors.As(err, &updateError) {
		t.Fatalf("error %v is not an update error", err)
	}
	if updateError.Category != want {
		t.Fatalf("error category = %s, want %s: %v", updateError.Category, want, err)
	}
}
