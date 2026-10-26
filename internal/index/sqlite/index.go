// Package sqliteindex maintains an optional, rebuildable projection of canonical workspace artifacts.
package sqliteindex

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	indexSchema  = "engkit.context-index/v1alpha1"
	databaseName = "index.db"
)

// RebuildResult describes a successfully published derived index.
type RebuildResult struct {
	SchemaVersion     string `json:"schema_version"`
	DatabasePath      string `json:"database_path"`
	ArtifactCount     int    `json:"artifact_count"`
	SourceFingerprint string `json:"source_fingerprint"`
	RebuiltAt         string `json:"rebuilt_at"`
}

// Status reports whether the derived index matches current canonical sources.
type Status struct {
	SchemaVersion      string `json:"schema_version"`
	DatabasePath       string `json:"database_path"`
	Exists             bool   `json:"exists"`
	Fresh              bool   `json:"fresh"`
	ArtifactCount      int    `json:"artifact_count"`
	SourceFingerprint  string `json:"source_fingerprint,omitempty"`
	CurrentFingerprint string `json:"current_fingerprint,omitempty"`
	RebuiltAt          string `json:"rebuilt_at,omitempty"`
}

// SearchResult contains documentation-only indexed content.
type SearchResult = contextsearch.Result

// SearchResponse is the stable JSON contract returned by context search.
type SearchResponse = contextsearch.Response

// Rebuild creates a new database from canonical files and replaces the prior projection.
func Rebuild(ctx context.Context, root string, now func() time.Time) (RebuildResult, error) {
	resolvedRoot, databasePath, err := validatedPaths(root)
	if err != nil {
		return RebuildResult{}, err
	}
	artifacts, fingerprint, err := collectArtifacts(resolvedRoot)
	if err != nil {
		return RebuildResult{}, err
	}
	if now == nil {
		now = time.Now
	}
	rebuiltAt := now().UTC().Format(time.RFC3339Nano)

	contextDir := filepath.Dir(databasePath)
	if err := os.MkdirAll(contextDir, 0o700); err != nil {
		return RebuildResult{}, fmt.Errorf("create context index directory: %w", err)
	}
	tempFile, err := os.CreateTemp(contextDir, "index-build-*.db")
	if err != nil {
		return RebuildResult{}, fmt.Errorf("create context index staging file: %w", err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return RebuildResult{}, fmt.Errorf("close context index staging file: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.Remove(tempPath)
		}
		_ = os.Remove(tempPath + "-journal")
		_ = os.Remove(tempPath + "-wal")
		_ = os.Remove(tempPath + "-shm")
	}()

	db, err := openDatabase(tempPath, false)
	if err != nil {
		return RebuildResult{}, err
	}
	if err := populate(ctx, db, artifacts, fingerprint, rebuiltAt); err != nil {
		_ = db.Close()
		return RebuildResult{}, err
	}
	if err := db.Close(); err != nil {
		return RebuildResult{}, fmt.Errorf("close rebuilt context index: %w", err)
	}

	if err := replaceDatabase(tempPath, databasePath); err != nil {
		return RebuildResult{}, err
	}
	published = true
	return RebuildResult{
		SchemaVersion:     indexSchema,
		DatabasePath:      ".engkit/context/" + databaseName,
		ArtifactCount:     len(artifacts),
		SourceFingerprint: fingerprint,
		RebuiltAt:         rebuiltAt,
	}, nil
}

// Inspect reports index freshness without changing the workspace.
func Inspect(ctx context.Context, root string) (Status, error) {
	resolvedRoot, databasePath, err := validatedPaths(root)
	if err != nil {
		return Status{}, err
	}
	status := Status{
		SchemaVersion: indexSchema,
		DatabasePath:  ".engkit/context/" + databaseName,
	}
	if _, err := os.Stat(databasePath); errors.Is(err, os.ErrNotExist) {
		_, current, collectErr := collectArtifacts(resolvedRoot)
		if collectErr != nil {
			return Status{}, collectErr
		}
		status.CurrentFingerprint = current
		return status, nil
	} else if err != nil {
		return Status{}, fmt.Errorf("inspect context index: %w", err)
	}

	db, err := openDatabase(databasePath, true)
	if err != nil {
		return Status{}, err
	}
	defer db.Close()

	metadata, err := readMetadata(ctx, db)
	if err != nil {
		return Status{}, err
	}
	count, err := strconv.Atoi(metadata["artifact_count"])
	if err != nil {
		return Status{}, fmt.Errorf("invalid context index artifact count: %w", err)
	}
	_, current, err := collectArtifacts(resolvedRoot)
	if err != nil {
		return Status{}, err
	}
	status.Exists = true
	status.ArtifactCount = count
	status.SourceFingerprint = metadata["source_fingerprint"]
	status.CurrentFingerprint = current
	status.RebuiltAt = metadata["rebuilt_at"]
	status.Fresh = metadata["schema_version"] == indexSchema && status.SourceFingerprint == current
	return status, nil
}

// Search queries indexed Markdown content. Non-document artifacts never expose content.
func Search(ctx context.Context, root, query string) (SearchResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResponse{}, errors.New("search query is required")
	}
	_, databasePath, err := validatedPaths(root)
	if err != nil {
		return SearchResponse{}, err
	}
	if _, err := os.Stat(databasePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SearchResponse{}, errors.New("context index does not exist; run context rebuild")
		}
		return SearchResponse{}, fmt.Errorf("inspect context index: %w", err)
	}
	status, err := Inspect(ctx, root)
	if err != nil {
		return SearchResponse{}, err
	}

	db, err := openDatabase(databasePath, true)
	if err != nil {
		return SearchResponse{}, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT path, kind, content
		FROM artifacts
		WHERE content IS NOT NULL AND instr(lower(content), lower(?)) > 0
		ORDER BY path
		LIMIT 50`, query)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("search context index: %w", err)
	}
	defer rows.Close()

	results := make([]SearchResult, 0)
	for rows.Next() {
		var path, kind, content string
		if err := rows.Scan(&path, &kind, &content); err != nil {
			return SearchResponse{}, fmt.Errorf("read context search result: %w", err)
		}
		results = append(results, SearchResult{Path: path, Kind: kind, Snippet: snippet(content)})
	}
	if err := rows.Err(); err != nil {
		return SearchResponse{}, fmt.Errorf("iterate context search results: %w", err)
	}
	return SearchResponse{SchemaVersion: indexSchema, Query: query, Fresh: status.Fresh, Results: results}, nil
}

func populate(ctx context.Context, db *sql.DB, artifacts []artifact, fingerprint, rebuiltAt string) error {
	statements := []string{
		`PRAGMA journal_mode = DELETE`,
		`PRAGMA synchronous = FULL`,
		`PRAGMA foreign_keys = ON`,
		`PRAGMA user_version = 1`,
		`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL) STRICT`,
		`CREATE TABLE artifacts (
			path TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			modified_ns INTEGER NOT NULL,
			sha256 TEXT NOT NULL,
			content TEXT
		) STRICT`,
		`CREATE INDEX artifacts_kind_idx ON artifacts(kind)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize context index schema: %w", err)
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin context index transaction: %w", err)
	}
	defer tx.Rollback()

	insertArtifact, err := tx.PrepareContext(ctx, `
		INSERT INTO artifacts(path, kind, size_bytes, modified_ns, sha256, content)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare context artifact insertion: %w", err)
	}
	defer insertArtifact.Close()
	for _, item := range artifacts {
		if _, err := insertArtifact.ExecContext(ctx, item.Path, item.Kind, item.SizeBytes, item.ModifiedNS, item.SHA256, item.Content); err != nil {
			return fmt.Errorf("insert context artifact %s: %w", item.Path, err)
		}
	}

	metadata := map[string]string{
		"schema_version":     indexSchema,
		"source_fingerprint": fingerprint,
		"rebuilt_at":         rebuiltAt,
		"artifact_count":     strconv.Itoa(len(artifacts)),
	}
	for key, value := range metadata {
		if _, err := tx.ExecContext(ctx, `INSERT INTO metadata(key, value) VALUES (?, ?)`, key, value); err != nil {
			return fmt.Errorf("insert context metadata %s: %w", key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit context index: %w", err)
	}
	return nil
}

func openDatabase(path string, readOnly bool) (*sql.DB, error) {
	dsn := path
	if readOnly {
		dsn = readOnlyURI(path)
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open context index: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to context index: %w", err)
	}
	return db, nil
}

func readOnlyURI(path string) string {
	slashPath := filepath.ToSlash(path)
	u := url.URL{Scheme: "file", Opaque: slashPath}
	query := u.Query()
	query.Set("mode", "ro")
	u.RawQuery = query.Encode()
	return u.String()
}

func readMetadata(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT key, value FROM metadata`)
	if err != nil {
		return nil, fmt.Errorf("read context index metadata: %w", err)
	}
	defer rows.Close()
	metadata := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan context index metadata: %w", err)
		}
		metadata[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate context index metadata: %w", err)
	}
	return metadata, nil
}

func replaceDatabase(staged, target string) error {
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(staged, target); err != nil {
			return fmt.Errorf("publish context index: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect existing context index: %w", err)
	}

	backupFile, err := os.CreateTemp(filepath.Dir(target), "index-backup-*.db")
	if err != nil {
		return fmt.Errorf("reserve context index backup path: %w", err)
	}
	backup := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		_ = os.Remove(backup)
		return fmt.Errorf("close context index backup placeholder: %w", err)
	}
	if err := os.Remove(backup); err != nil {
		return fmt.Errorf("prepare context index backup path: %w", err)
	}

	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("back up current context index: %w", err)
	}
	if err := os.Rename(staged, target); err != nil {
		if rollbackErr := os.Rename(backup, target); rollbackErr != nil {
			return fmt.Errorf("publish context index: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("publish context index: %w", err)
	}
	if err := os.Remove(backup); err != nil {
		return fmt.Errorf("remove replaced context index backup: %w", err)
	}
	return nil
}

func validatedPaths(root string) (string, string, error) {
	snapshot, err := workspace.Inspect(root)
	if err != nil {
		return "", "", err
	}
	if !snapshot.HasEngkit {
		return "", "", errors.New("workspace is not initialized; run workspace plan-init and workspace init first")
	}
	resolvedRoot := filepath.FromSlash(snapshot.Root)
	return resolvedRoot, filepath.Join(resolvedRoot, ".engkit", "context", databaseName), nil
}

func snippet(content string) string {
	fields := strings.FieldsFunc(content, unicode.IsSpace)
	text := strings.Join(fields, " ")
	runes := []rune(text)
	if len(runes) > 240 {
		return string(runes[:240]) + "…"
	}
	return text
}
