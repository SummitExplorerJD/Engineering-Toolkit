package tufclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	engupdate "github.com/SummitExplorerJD/Engineering-Toolkit/internal/update"
)

func TestOperationLockHonorsCancellation(t *testing.T) {
	if err := acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("acquire() error = %v, want context cancellation", err)
	}
}

func TestClientRefreshTargetAndDownload(t *testing.T) {
	repository := newTestRepository(t, time.Now().Add(24*time.Hour))
	root := t.TempDir()
	client, err := New(Config{
		Repository: engupdate.Repository{
			MetadataURL: repository.server.URL + "/metadata",
			TargetsURL:  repository.server.URL + "/targets",
			Root:        repository.root,
		},
		MetadataDir:  filepath.Join(root, "state", "metadata"),
		TargetCache:  filepath.Join(root, "cache", "targets"),
		UserAgent:    "engkit/v1.0.0 update-client/1",
		RoundTripper: repository.server.Client().Transport,
		Now:          func() time.Time { return time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
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
	digest := sha256.Sum256(repository.target)
	if target.SHA256 != hex.EncodeToString(digest[:]) || target.Length != int64(len(repository.target)) {
		t.Fatalf("unexpected target: %+v", target)
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
		t.Fatalf("downloaded bytes = %q", data)
	}
	info, err := os.Stat(verified.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 != 0 {
		t.Fatalf("verified target is executable: %v", info.Mode())
	}
	if _, err := client.Download(context.Background(), target, filepath.Join(root, "published")); err != nil {
		t.Fatalf("idempotent cached download failed: %v", err)
	}
}

func TestFailedRefreshPreservesLastKnownGoodMetadata(t *testing.T) {
	repository := newTestRepository(t, time.Now().Add(24*time.Hour))
	root := t.TempDir()
	metadataDir := filepath.Join(root, "state", "metadata")
	client, err := New(Config{
		Repository:  engupdate.Repository{MetadataURL: repository.server.URL + "/metadata", TargetsURL: repository.server.URL + "/targets", Root: repository.root},
		MetadataDir: metadataDir, TargetCache: filepath.Join(root, "cache"),
		RoundTripper: repository.server.Client().Transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(metadataDir, "timestamp.json"))
	if err != nil {
		t.Fatal(err)
	}
	repository.corruptTimestamp(t)
	if _, err := client.Refresh(context.Background()); err == nil {
		t.Fatal("expected refresh failure")
	}
	after, err := os.ReadFile(filepath.Join(metadataDir, "timestamp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed refresh replaced last-known-good metadata")
	}
}

func TestClientRejectsExpiredMetadata(t *testing.T) {
	repository := newTestRepository(t, time.Now().Add(-time.Hour))
	root := t.TempDir()
	client, err := New(Config{
		Repository:  engupdate.Repository{MetadataURL: repository.server.URL + "/metadata", TargetsURL: repository.server.URL + "/targets", Root: repository.root},
		MetadataDir: filepath.Join(root, "metadata"), TargetCache: filepath.Join(root, "cache"),
		RoundTripper: repository.server.Client().Transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Refresh(context.Background())
	if err == nil {
		t.Fatal("expected expired metadata failure")
	}
	var updateError *engupdate.Error
	if !errors.As(err, &updateError) || updateError.Category != engupdate.ErrorMetadataExpired {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadRejectsCacheCollision(t *testing.T) {
	repository := newTestRepository(t, time.Now().Add(24*time.Hour))
	root := t.TempDir()
	client, err := New(Config{
		Repository:  engupdate.Repository{MetadataURL: repository.server.URL + "/metadata", TargetsURL: repository.server.URL + "/targets", Root: repository.root},
		MetadataDir: filepath.Join(root, "metadata"), TargetCache: filepath.Join(root, "cache"),
		RoundTripper: repository.server.Client().Transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	target, err := client.Target(context.Background(), repository.targetPath)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "published")
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	collision := filepath.Join(cache, target.SHA256+"-"+filepath.Base(target.Path))
	if err := os.WriteFile(collision, []byte("different"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Download(context.Background(), target, cache); err == nil {
		t.Fatal("expected collision error")
	}
}

func TestRootRotationPolicy(t *testing.T) {
	tests := []struct {
		name       string
		version    int64
		signatures string
		wantError  bool
	}{
		{name: "dual threshold", version: 2, signatures: "dual"},
		{name: "insufficient threshold", version: 2, signatures: "insufficient", wantError: true},
		{name: "duplicate signature", version: 2, signatures: "duplicate", wantError: true},
		{name: "skipped version", version: 3, signatures: "dual", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newTestRepository(t, time.Now().Add(24*time.Hour))
			repository.setRootRotation(t, test.version, test.signatures)
			root := t.TempDir()
			client, err := New(Config{
				Repository:  engupdate.Repository{MetadataURL: repository.server.URL + "/metadata", TargetsURL: repository.server.URL + "/targets", Root: repository.root},
				MetadataDir: filepath.Join(root, "metadata"), TargetCache: filepath.Join(root, "cache"),
				RoundTripper: repository.server.Client().Transport,
			})
			if err != nil {
				t.Fatal(err)
			}
			state, err := client.Refresh(context.Background())
			if test.wantError {
				if err == nil {
					t.Fatal("expected root rotation failure")
				}
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
}

func TestTimestampRollbackIsRejected(t *testing.T) {
	repository := newTestRepository(t, time.Now().Add(24*time.Hour))
	repository.advanceTimestamp(t)
	root := t.TempDir()
	metadataDir := filepath.Join(root, "metadata")
	client, err := New(Config{
		Repository:  engupdate.Repository{MetadataURL: repository.server.URL + "/metadata", TargetsURL: repository.server.URL + "/targets", Root: repository.root},
		MetadataDir: metadataDir, TargetCache: filepath.Join(root, "cache"),
		RoundTripper: repository.server.Client().Transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := client.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.TimestampVersion != 2 {
		t.Fatalf("timestamp version = %d, want 2", state.TimestampVersion)
	}
	repository.rollbackTimestamp()
	_, err = client.Refresh(context.Background())
	if err == nil {
		t.Fatal("expected timestamp rollback rejection")
	}
	var updateError *engupdate.Error
	if !errors.As(err, &updateError) || updateError.Category != engupdate.ErrorRollbackDetected {
		t.Fatalf("unexpected rollback error: %v", err)
	}
}

func TestDownloadRejectsWrongTargetBytes(t *testing.T) {
	for _, test := range []struct {
		name string
		data func([]byte) []byte
	}{
		{name: "wrong hash", data: func(original []byte) []byte {
			changed := append([]byte(nil), original...)
			changed[0] ^= 0xff
			return changed
		}},
		{name: "wrong length", data: func(original []byte) []byte { return append(append([]byte(nil), original...), 'x') }},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := newTestRepository(t, time.Now().Add(24*time.Hour))
			root := t.TempDir()
			client, err := New(Config{
				Repository:  engupdate.Repository{MetadataURL: repository.server.URL + "/metadata", TargetsURL: repository.server.URL + "/targets", Root: repository.root},
				MetadataDir: filepath.Join(root, "metadata"), TargetCache: filepath.Join(root, "cache"),
				RoundTripper: repository.server.Client().Transport,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Refresh(context.Background()); err != nil {
				t.Fatal(err)
			}
			target, err := client.Target(context.Background(), repository.targetPath)
			if err != nil {
				t.Fatal(err)
			}
			repository.replaceTarget(test.data(repository.target))
			if _, err := client.Download(context.Background(), target, filepath.Join(root, "published")); err == nil {
				t.Fatal("expected target verification failure")
			}
		})
	}
}

func TestInspectTrustedStateRejectsUnsafeRepositoryFeatures(t *testing.T) {
	directory := t.TempDir()
	writeEnvelope := func(role string, consistent bool, delegations any) {
		t.Helper()
		signed := map[string]any{
			"version": 1, "expires": time.Now().Add(time.Hour).UTC(),
			"consistent_snapshot": consistent,
		}
		if delegations != nil {
			signed["delegations"] = delegations
		}
		data, err := json.Marshal(map[string]any{"signed": signed})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, role+".json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, role := range []string{"root", "timestamp", "snapshot", "targets"} {
		writeEnvelope(role, role == "root", nil)
	}
	if _, err := inspectTrustedState(directory); err != nil {
		t.Fatalf("valid top-level state rejected: %v", err)
	}
	writeEnvelope("root", false, nil)
	if _, err := inspectTrustedState(directory); err == nil {
		t.Fatal("expected consistent-snapshot rejection")
	}
	writeEnvelope("root", true, nil)
	writeEnvelope("targets", false, map[string]any{"roles": []any{map[string]any{"name": "delegated"}}})
	if _, err := inspectTrustedState(directory); err == nil {
		t.Fatal("expected delegation rejection")
	}
}
