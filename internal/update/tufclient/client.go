package tufclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	engupdate "github.com/SummitExplorerJD/Engineering-Toolkit/internal/update"
	"github.com/theupdateframework/go-tuf/v2/metadata"
	"github.com/theupdateframework/go-tuf/v2/metadata/config"
	"github.com/theupdateframework/go-tuf/v2/metadata/updater"
)

var operationSlot = make(chan struct{}, 1)

type Config struct {
	Repository   engupdate.Repository
	MetadataDir  string
	TargetCache  string
	UserAgent    string
	RoundTripper http.RoundTripper
	Now          func() time.Time
}

type Client struct {
	config    Config
	transport *Transport
	fetcher   *contextFetcher
	mu        sync.RWMutex
	updater   *updater.Updater
	targets   map[string]*metadata.TargetFiles
	state     engupdate.TrustedState
}

type contextFetcher struct {
	transport *Transport
	mu        sync.RWMutex
	ctx       context.Context
}

func (f *contextFetcher) setContext(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ctx = ctx
}

func (f *contextFetcher) DownloadFile(rawURL string, maxLength int64, _ time.Duration) ([]byte, error) {
	f.mu.RLock()
	ctx := f.ctx
	f.mu.RUnlock()
	if ctx == nil {
		ctx = context.Background()
	}
	return f.transport.Download(ctx, rawURL, maxLength)
}

func New(clientConfig Config) (*Client, error) {
	if err := engupdate.ValidateRepository(clientConfig.Repository); err != nil {
		return nil, &engupdate.Error{Category: engupdate.ErrorUpdateNotConfigured, Op: "validate repository", Err: err}
	}
	if !filepath.IsAbs(clientConfig.MetadataDir) || !filepath.IsAbs(clientConfig.TargetCache) {
		return nil, &engupdate.Error{Category: engupdate.ErrorUpdateNotConfigured, Op: "validate paths", Err: errors.New("metadata and target cache paths must be absolute")}
	}
	options := []TransportOption{}
	if clientConfig.RoundTripper != nil {
		options = append(options, WithRoundTripper(clientConfig.RoundTripper))
	}
	transport, err := NewTransport(clientConfig.Repository.MetadataURL, clientConfig.Repository.TargetsURL, clientConfig.UserAgent, options...)
	if err != nil {
		return nil, &engupdate.Error{Category: engupdate.ErrorUpdateNotConfigured, Op: "configure transport", Err: err}
	}
	if clientConfig.Now == nil {
		clientConfig.Now = time.Now
	}
	fetcher := &contextFetcher{transport: transport}
	return &Client{
		config: clientConfig, transport: transport, fetcher: fetcher,
		targets: make(map[string]*metadata.TargetFiles),
	}, nil
}

func (c *Client) Refresh(ctx context.Context) (engupdate.TrustedState, error) {
	if err := acquire(ctx); err != nil {
		return engupdate.TrustedState{}, classified("acquire update lock", err)
	}
	defer release()
	c.fetcher.setContext(ctx)

	parent := filepath.Dir(c.config.MetadataDir)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return engupdate.TrustedState{}, cacheError("create metadata parent", err)
	}
	stage, err := os.MkdirTemp(parent, ".metadata-stage-")
	if err != nil {
		return engupdate.TrustedState{}, cacheError("create metadata stage", err)
	}
	stageOwned := true
	defer func() {
		if stageOwned {
			_ = os.RemoveAll(stage)
		}
	}()
	if err := copyMetadata(c.config.MetadataDir, stage); err != nil {
		return engupdate.TrustedState{}, cacheError("stage trusted metadata", err)
	}
	if err := os.MkdirAll(c.config.TargetCache, 0700); err != nil {
		return engupdate.TrustedState{}, cacheError("create target cache", err)
	}

	updaterConfig, err := config.New(c.config.Repository.MetadataURL, c.config.Repository.Root)
	if err != nil {
		return engupdate.TrustedState{}, classified("configure updater", err)
	}
	updaterConfig.Fetcher = c.fetcher
	updaterConfig.LocalMetadataDir = stage
	updaterConfig.LocalTargetsDir = c.config.TargetCache
	updaterConfig.RemoteTargetsURL = c.config.Repository.TargetsURL
	updaterConfig.MaxRootRotations = MaxRootRotations
	updaterConfig.MaxDelegations = 0
	updaterConfig.RootMaxLength = RootMaxLength
	updaterConfig.TimestampMaxLength = TimestampMaxLength
	updaterConfig.SnapshotMaxLength = SnapshotMaxLength
	updaterConfig.TargetsMaxLength = TargetsMaxLength
	updaterConfig.PrefixTargetsWithHash = true

	instance, err := updater.New(updaterConfig)
	if err != nil {
		return engupdate.TrustedState{}, classified("initialize updater", err)
	}
	if err := instance.Refresh(); err != nil {
		return engupdate.TrustedState{}, classified("refresh metadata", err)
	}
	state, err := inspectTrustedState(stage)
	if err != nil {
		return engupdate.TrustedState{}, classified("inspect trusted metadata", err)
	}
	if err := publishMetadata(stage, c.config.MetadataDir); err != nil {
		return engupdate.TrustedState{}, cacheError("publish trusted metadata", err)
	}
	stageOwned = false

	c.mu.Lock()
	c.updater = instance
	c.targets = make(map[string]*metadata.TargetFiles)
	c.state = state
	c.mu.Unlock()
	return state, nil
}

func (c *Client) Target(ctx context.Context, targetPath string) (engupdate.Target, error) {
	if err := acquire(ctx); err != nil {
		return engupdate.Target{}, classified("acquire update lock", err)
	}
	defer release()
	c.fetcher.setContext(ctx)
	if err := validateTargetPath(targetPath); err != nil {
		return engupdate.Target{}, &engupdate.Error{Category: engupdate.ErrorTargetInvalid, Op: "validate target path", Err: err}
	}
	c.mu.RLock()
	instance := c.updater
	c.mu.RUnlock()
	if instance == nil {
		return engupdate.Target{}, &engupdate.Error{Category: engupdate.ErrorUpdateNotConfigured, Op: "target lookup", Err: errors.New("metadata has not been refreshed")}
	}
	info, err := instance.GetTargetInfo(targetPath)
	if err != nil {
		return engupdate.Target{}, classified("lookup target", err)
	}
	if info == nil {
		return engupdate.Target{}, &engupdate.Error{Category: engupdate.ErrorTargetInvalid, Op: "lookup target", Err: errors.New("target is not present in trusted metadata")}
	}
	target, err := convertTarget(info)
	if err != nil {
		return engupdate.Target{}, &engupdate.Error{Category: engupdate.ErrorTargetInvalid, Op: "validate signed target", Err: err}
	}
	if err := engupdate.ValidateTarget(target, target.Channel, target.Version, target.Platform); err != nil {
		return engupdate.Target{}, &engupdate.Error{Category: engupdate.ErrorTargetIncompatible, Op: "validate target policy", Err: err}
	}
	c.mu.Lock()
	c.targets[target.Path] = info
	c.mu.Unlock()
	return target, nil
}

func (c *Client) Download(ctx context.Context, target engupdate.Target, cacheDir string) (engupdate.VerifiedFile, error) {
	if err := acquire(ctx); err != nil {
		return engupdate.VerifiedFile{}, classified("acquire update lock", err)
	}
	defer release()
	c.fetcher.setContext(ctx)
	if !filepath.IsAbs(cacheDir) {
		return engupdate.VerifiedFile{}, cacheError("validate cache path", errors.New("cache path must be absolute"))
	}
	if err := engupdate.ValidateTarget(target, target.Channel, target.Version, target.Platform); err != nil {
		return engupdate.VerifiedFile{}, &engupdate.Error{Category: engupdate.ErrorTargetInvalid, Op: "validate target", Err: err}
	}
	c.mu.RLock()
	instance := c.updater
	info := c.targets[target.Path]
	c.mu.RUnlock()
	if instance == nil || info == nil {
		return engupdate.VerifiedFile{}, &engupdate.Error{Category: engupdate.ErrorUpdateNotConfigured, Op: "download target", Err: errors.New("target must be resolved from refreshed metadata before download")}
	}
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return engupdate.VerifiedFile{}, cacheError("create target cache", err)
	}
	finalPath := filepath.Join(cacheDir, target.SHA256+"-"+filepath.Base(target.Path))
	if ok, err := verifiedExisting(finalPath, target); err != nil {
		return engupdate.VerifiedFile{}, cacheError("check cached target", err)
	} else if ok {
		return c.verifiedFile(finalPath, target), nil
	}
	temporary, err := os.CreateTemp(cacheDir, ".target-partial-")
	if err != nil {
		return engupdate.VerifiedFile{}, cacheError("create target stage", err)
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return engupdate.VerifiedFile{}, cacheError("close target stage", err)
	}
	defer os.Remove(temporaryPath)
	if err := os.Chmod(temporaryPath, 0600); err != nil {
		return engupdate.VerifiedFile{}, cacheError("protect target stage", err)
	}
	_, data, err := instance.DownloadTarget(info, temporaryPath, "")
	if err != nil {
		return engupdate.VerifiedFile{}, classified("download target", err)
	}
	if int64(len(data)) != target.Length || int64(len(data)) > TargetMaxLength {
		return engupdate.VerifiedFile{}, &engupdate.Error{Category: engupdate.ErrorTargetInvalid, Op: "verify target length", Err: errors.New("downloaded target length mismatch")}
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != target.SHA256 {
		return engupdate.VerifiedFile{}, &engupdate.Error{Category: engupdate.ErrorTargetInvalid, Op: "verify target digest", Err: errors.New("downloaded target digest mismatch")}
	}
	if err := syncFile(temporaryPath); err != nil {
		return engupdate.VerifiedFile{}, cacheError("synchronize target", err)
	}
	if ok, err := verifiedExisting(finalPath, target); err != nil {
		return engupdate.VerifiedFile{}, cacheError("check publication collision", err)
	} else if ok {
		return c.verifiedFile(finalPath, target), nil
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		if ok, verifyErr := verifiedExisting(finalPath, target); verifyErr == nil && ok {
			return c.verifiedFile(finalPath, target), nil
		}
		return engupdate.VerifiedFile{}, cacheError("publish verified target", err)
	}
	if err := os.Chmod(finalPath, 0600); err != nil {
		return engupdate.VerifiedFile{}, cacheError("protect verified target", err)
	}
	return c.verifiedFile(finalPath, target), nil
}

func (c *Client) verifiedFile(path string, target engupdate.Target) engupdate.VerifiedFile {
	return engupdate.VerifiedFile{Path: path, Length: target.Length, SHA256: target.SHA256, VerifiedAt: c.config.Now().UTC()}
}

func acquire(ctx context.Context) error {
	select {
	case operationSlot <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func release() { <-operationSlot }

func convertTarget(info *metadata.TargetFiles) (engupdate.Target, error) {
	sha, ok := info.Hashes["sha256"]
	if !ok || len(sha) != sha256.Size {
		return engupdate.Target{}, errors.New("trusted target must contain exactly one usable SHA-256 digest")
	}
	if info.Custom == nil {
		return engupdate.Target{}, errors.New("signed target custom metadata is required")
	}
	var custom struct {
		Schema  string `json:"schema"`
		Version string `json:"version"`
		Channel string `json:"channel"`
		OS      string `json:"os"`
		Arch    string `json:"arch"`
		Format  string `json:"format"`
	}
	if err := json.Unmarshal(*info.Custom, &custom); err != nil {
		return engupdate.Target{}, err
	}
	if custom.Schema != engupdate.TargetSchema {
		return engupdate.Target{}, fmt.Errorf("unsupported target schema %q", custom.Schema)
	}
	var customFields map[string]any
	if err := json.Unmarshal(*info.Custom, &customFields); err != nil {
		return engupdate.Target{}, err
	}
	return engupdate.Target{
		Version: custom.Version, Channel: custom.Channel,
		Platform: engupdate.Platform{OS: custom.OS, Arch: custom.Arch},
		Path:     info.Path, Length: info.Length, SHA256: hex.EncodeToString(sha),
		Format: custom.Format, Custom: customFields,
	}, nil
}

type metadataEnvelope struct {
	Signed struct {
		Version            int64           `json:"version"`
		Expires            time.Time       `json:"expires"`
		ConsistentSnapshot bool            `json:"consistent_snapshot"`
		Delegations        json.RawMessage `json:"delegations"`
	} `json:"signed"`
}

func inspectTrustedState(directory string) (engupdate.TrustedState, error) {
	roles := make(map[string]metadataEnvelope, 4)
	for _, role := range []string{"root", "timestamp", "snapshot", "targets"} {
		data, err := os.ReadFile(filepath.Join(directory, role+".json"))
		if err != nil {
			return engupdate.TrustedState{}, err
		}
		var envelope metadataEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			return engupdate.TrustedState{}, err
		}
		if envelope.Signed.Version < 1 {
			return engupdate.TrustedState{}, fmt.Errorf("%s metadata has invalid version", role)
		}
		roles[role] = envelope
	}
	if !roles["root"].Signed.ConsistentSnapshot {
		return engupdate.TrustedState{}, errors.New("consistent snapshots are required")
	}
	delegations := roles["targets"].Signed.Delegations
	if len(delegations) != 0 && string(delegations) != "null" && string(delegations) != "{}" {
		return engupdate.TrustedState{}, errors.New("delegated targets are not supported")
	}
	return engupdate.TrustedState{
		RootVersion:      roles["root"].Signed.Version,
		TimestampVersion: roles["timestamp"].Signed.Version,
		SnapshotVersion:  roles["snapshot"].Signed.Version,
		TargetsVersion:   roles["targets"].Signed.Version,
		TimestampExpires: roles["timestamp"].Signed.Expires,
		SnapshotExpires:  roles["snapshot"].Signed.Expires,
		TargetsExpires:   roles["targets"].Signed.Expires,
	}, nil
}

func copyMetadata(source, destination string) error {
	entries, err := os.ReadDir(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		input, err := os.Open(filepath.Join(source, entry.Name()))
		if err != nil {
			return err
		}
		output, err := os.OpenFile(filepath.Join(destination, entry.Name()), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, io.LimitReader(input, TargetsMaxLength+1))
		closeErr := output.Close()
		_ = input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func publishMetadata(stage, destination string) error {
	parent := filepath.Dir(destination)
	backup, err := os.MkdirTemp(parent, ".metadata-backup-")
	if err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}
	hadDestination := false
	if _, err := os.Stat(destination); err == nil {
		hadDestination = true
		if err := os.Rename(destination, backup); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(stage, destination); err != nil {
		if hadDestination {
			_ = os.Rename(backup, destination)
		}
		return err
	}
	if hadDestination {
		if err := os.RemoveAll(backup); err != nil {
			return err
		}
	}
	return nil
}

func validateTargetPath(targetPath string) error {
	if targetPath == "" || filepath.IsAbs(targetPath) || strings.HasPrefix(targetPath, "/") || strings.Contains(targetPath, "\\") {
		return errors.New("target path must be a non-empty relative slash path")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(targetPath)))
	if clean != targetPath || strings.HasPrefix(clean, "../") || clean == ".." {
		return errors.New("target path is not canonical")
	}
	return nil
}

func verifiedExisting(path string, target engupdate.Target) (bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || info.Size() != target.Length || info.Mode().Perm()&0111 != 0 {
		return false, errors.New("cache collision with incompatible file")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, TargetMaxLength+1)); err != nil {
		return false, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != target.SHA256 {
		return false, errors.New("cache collision with different digest")
	}
	return true, nil
}

func syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func cacheError(op string, err error) error {
	return &engupdate.Error{Category: engupdate.ErrorCacheFailed, Op: op, Err: err}
}

func classified(op string, err error) error {
	category := engupdate.ErrorTrustFailed
	message := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		category = engupdate.ErrorTransportFailed
	case strings.Contains(message, "expired"):
		category = engupdate.ErrorMetadataExpired
	case strings.Contains(message, "rollback") ||
		strings.Contains(message, "version") && (strings.Contains(message, "lower") || strings.Contains(message, "must be >=") || strings.Contains(message, "bad version")):
		category = engupdate.ErrorRollbackDetected
	case strings.Contains(message, "download") || strings.Contains(message, "http") || strings.Contains(message, "connection"):
		category = engupdate.ErrorTransportFailed
	}
	return &engupdate.Error{Category: category, Op: op, Err: err}
}
