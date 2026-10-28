// Package sqliteindex maintains an optional, rebuildable projection of canonical workspace artifacts.
package sqliteindex

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	sqlite3 "github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextview"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	indexSchema      = "engkit.context-index/v1alpha1"
	databaseName     = "index.db"
	maxDatabaseBytes = 512 << 20
)

func init() {
	sqlite3.AutoExtension(fts5.Register)
}

// RebuildResult describes a successfully published derived index.
type RebuildResult struct {
	SchemaVersion     string `json:"schema_version"`
	DatabasePath      string `json:"database_path"`
	ArtifactCount     int    `json:"artifact_count"`
	SourceCount       int    `json:"source_count"`
	SourceBytes       int64  `json:"source_bytes"`
	ChunkCount        int    `json:"chunk_count"`
	ExcludedCount     int    `json:"excluded_count"`
	SourceFingerprint string `json:"source_fingerprint"`
	RebuiltAt         string `json:"rebuilt_at"`
}

// Status reports whether the derived index matches current canonical sources.
type Status = contextview.Status

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
	inventory, err := collectSourceInventory(resolvedRoot)
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
	if err := populate(ctx, db, inventory, rebuiltAt); err != nil {
		_ = db.Close()
		return RebuildResult{}, err
	}
	var integrity string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		_ = db.Close()
		if err != nil {
			return RebuildResult{}, fmt.Errorf("verify rebuilt context index: %w", err)
		}
		return RebuildResult{}, errors.New("rebuilt context index failed integrity verification")
	}
	if err := db.Close(); err != nil {
		return RebuildResult{}, fmt.Errorf("close rebuilt context index: %w", err)
	}
	if info, err := os.Stat(tempPath); err != nil {
		return RebuildResult{}, fmt.Errorf("inspect rebuilt context index: %w", err)
	} else if info.Size() > maxDatabaseBytes {
		return RebuildResult{}, errors.New("rebuilt context index exceeds the 512 MiB limit")
	}

	if err := replaceDatabase(tempPath, databasePath); err != nil {
		return RebuildResult{}, err
	}
	published = true
	return RebuildResult{
		SchemaVersion:     indexSchema,
		DatabasePath:      ".engkit/context/" + databaseName,
		ArtifactCount:     len(inventory.Sources),
		SourceCount:       len(inventory.Sources),
		SourceBytes:       inventory.SourceBytes,
		ChunkCount:        inventory.ChunkCount,
		ExcludedCount:     exclusionTotal(inventory.Exclusions),
		SourceFingerprint: inventory.Fingerprint,
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
		SchemaVersion: contextview.StatusSchema,
		DatabasePath:  ".engkit/context/" + databaseName,
	}
	current, collectErr := collectSourceInventory(resolvedRoot)
	if collectErr != nil {
		return Status{}, collectErr
	}
	status.ConfiguredRootCount = current.ConfiguredRoots
	status.CurrentFingerprint = current.Fingerprint
	if _, err := os.Stat(databasePath); errors.Is(err, os.ErrNotExist) {
		status.Reason = "missing"
		return status, nil
	} else if err != nil {
		return Status{}, fmt.Errorf("inspect context index: %w", err)
	}
	status.Exists = true
	if info, err := os.Stat(databasePath); err == nil {
		status.IndexBytes = info.Size()
	}

	db, err := openDatabase(databasePath, true)
	if err != nil {
		return Status{}, err
	}
	defer db.Close()
	var userVersion int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&userVersion); err != nil {
		return Status{}, fmt.Errorf("read context index version: %w", err)
	}
	if userVersion != 2 {
		status.Reason = "unsupported_schema"
		return status, nil
	}
	status.Supported = true

	metadata, err := readMetadata(ctx, db)
	if err != nil {
		return Status{}, err
	}
	count, err := strconv.Atoi(metadata["artifact_count"])
	if err != nil {
		return Status{}, fmt.Errorf("invalid context index artifact count: %w", err)
	}
	status.ArtifactCount = count
	status.SourceCount, _ = strconv.Atoi(metadata["source_count"])
	status.ChunkCount, _ = strconv.Atoi(metadata["chunk_count"])
	status.ExcludedCount, _ = strconv.Atoi(metadata["excluded_count"])
	status.SourceBytes, _ = strconv.ParseInt(metadata["source_bytes"], 10, 64)
	status.PolicyVersion = metadata["policy_version"]
	status.SourceFingerprint = metadata["source_fingerprint"]
	status.ProjectionFingerprint = metadata["source_fingerprint"]
	status.RebuiltAt = metadata["rebuilt_at"]
	status.Fresh = metadata["schema_version"] == indexSchema && metadata["policy_version"] == policyVersion && status.SourceFingerprint == current.Fingerprint
	if !status.Fresh {
		status.Reason = "stale"
	}
	return status, nil
}

// Search queries indexed content with compact token-efficient defaults.
func Search(ctx context.Context, root, query string) (SearchResponse, error) {
	return SearchWithOptions(ctx, root, query, contextsearch.Options{})
}

// SearchWithOptions applies bounded filters without exposing FTS syntax.
func SearchWithOptions(ctx context.Context, root, query string, options contextsearch.Options) (SearchResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResponse{}, errors.New("search query is required")
	}
	if len([]byte(query)) > contextsearch.MaxQueryBytes {
		return SearchResponse{}, errors.New("search query exceeds the 512-byte limit")
	}
	if options.Limit == 0 {
		options.Limit = contextsearch.DefaultResults
	}
	if options.Limit < 1 || options.Limit > contextsearch.MaxResults {
		return SearchResponse{}, errors.New("search result limit must be between 1 and 20")
	}
	if options.MaxContentBytes == 0 {
		options.MaxContentBytes = contextsearch.DefaultContentBytes
	}
	if options.MaxContentBytes < 1 || options.MaxContentBytes > contextsearch.MaxContentBytes {
		return SearchResponse{}, errors.New("search content budget must be between 1 and 16384 bytes")
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
	if !status.Supported {
		return SearchResponse{}, errors.New("context index uses an unsupported schema; run context rebuild")
	}
	if !status.Fresh {
		return SearchResponse{}, errors.New("context index is stale; run context rebuild")
	}

	db, err := openDatabase(databasePath, true)
	if err != nil {
		return SearchResponse{}, err
	}
	defer db.Close()

	statement := `SELECT s.path, s.kind, s.language, c.chunk_id, c.start_line,
		c.end_line, c.content_sha256, c.content
		FROM chunk_search
		JOIN chunks c ON c.rowid = chunk_search.rowid
		JOIN sources s ON s.path = c.source_path
		WHERE chunk_search MATCH ?`
	arguments := []any{literalFTSQuery(query)}
	if options.PathPrefix != "" {
		statement += ` AND (s.path = ? OR s.path LIKE ? ESCAPE '\')`
		arguments = append(arguments, options.PathPrefix, escapeLike(options.PathPrefix)+"/%")
	}
	if options.Kind != "" {
		statement += ` AND s.kind = ?`
		arguments = append(arguments, options.Kind)
	}
	if options.Language != "" {
		statement += ` AND s.language = ?`
		arguments = append(arguments, options.Language)
	}
	statement += ` ORDER BY bm25(chunk_search), s.path, c.ordinal LIMIT ?`
	arguments = append(arguments, options.Limit+1)
	rows, err := db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("search context index: %w", err)
	}
	defer rows.Close()

	results := make([]SearchResult, 0)
	truncated := false
	for rows.Next() {
		var item SearchResult
		var content string
		if err := rows.Scan(&item.Path, &item.Kind, &item.Language, &item.ChunkID, &item.StartLine, &item.EndLine, &item.ContentSHA256, &content); err != nil {
			return SearchResponse{}, fmt.Errorf("read context search result: %w", err)
		}
		if len(results) == options.Limit {
			truncated = true
			break
		}
		remaining := options.MaxContentBytes
		for _, existing := range results {
			remaining -= len(existing.Snippet)
		}
		if remaining <= 0 {
			truncated = true
			break
		}
		limit := contextsearch.DefaultSnippetBytes
		if remaining < limit {
			limit = remaining
		}
		item.Snippet, item.Truncated = compactSnippet(content, query, limit)
		truncated = truncated || item.Truncated
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return SearchResponse{}, fmt.Errorf("iterate context search results: %w", err)
	}
	applied := 0
	for _, item := range results {
		applied += len(item.Snippet)
	}
	return SearchResponse{SchemaVersion: indexSchema, Query: query, Fresh: status.Fresh, Results: results, AppliedLimit: options.Limit, AppliedMaxBytes: options.MaxContentBytes, AppliedContentBytes: applied, Truncated: truncated}, nil
}

// ListSources returns content-free indexed source metadata.
func ListSources(ctx context.Context, root string, options contextview.SourceOptions) (contextview.Sources, error) {
	status, err := Inspect(ctx, root)
	if err != nil {
		return contextview.Sources{}, err
	}
	if !status.Exists || !status.Supported || !status.Fresh {
		return contextview.Sources{}, errors.New("a fresh supported context index is required; run context rebuild")
	}
	if options.Limit == 0 {
		options.Limit = contextview.DefaultPage
	}
	if options.Limit < 1 || options.Limit > contextview.MaxPage {
		return contextview.Sources{}, errors.New("source limit must be between 1 and 200")
	}
	for _, value := range []string{options.PathPrefix, options.After} {
		if err := validateRelativeContextPath(value, true); err != nil {
			return contextview.Sources{}, err
		}
	}
	if options.After != "" && options.ProjectionFingerprint != status.ProjectionFingerprint {
		return contextview.Sources{}, errors.New("source continuation requires the current projection fingerprint")
	}
	_, databasePath, err := validatedPaths(root)
	if err != nil {
		return contextview.Sources{}, err
	}
	db, err := openDatabase(databasePath, true)
	if err != nil {
		return contextview.Sources{}, err
	}
	defer db.Close()
	statement := `SELECT path, kind, language, size_bytes, source_sha256,
		normalized_sha256, chunk_count FROM sources WHERE 1 = 1`
	arguments := make([]any, 0, 8)
	if options.PathPrefix != "" {
		statement += ` AND (path = ? OR path LIKE ? ESCAPE '\')`
		arguments = append(arguments, options.PathPrefix, escapeLike(options.PathPrefix)+"/%")
	}
	if options.Kind != "" {
		statement += ` AND kind = ?`
		arguments = append(arguments, options.Kind)
	}
	if options.Language != "" {
		statement += ` AND language = ?`
		arguments = append(arguments, options.Language)
	}
	if options.After != "" {
		statement += ` AND path > ?`
		arguments = append(arguments, options.After)
	}
	statement += ` ORDER BY path LIMIT ?`
	arguments = append(arguments, options.Limit+1)
	rows, err := db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return contextview.Sources{}, fmt.Errorf("list context sources: %w", err)
	}
	defer rows.Close()
	result := contextview.Sources{SchemaVersion: contextview.SourcesSchema, ProjectionFingerprint: status.ProjectionFingerprint, AppliedLimit: options.Limit, Items: []contextview.Source{}}
	for rows.Next() {
		var item contextview.Source
		var normalized sql.NullString
		if err := rows.Scan(&item.Path, &item.Kind, &item.Language, &item.SizeBytes, &item.SourceSHA256, &normalized, &item.ChunkCount); err != nil {
			return contextview.Sources{}, fmt.Errorf("read context source: %w", err)
		}
		item.NormalizedSHA256 = normalized.String
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return contextview.Sources{}, fmt.Errorf("iterate context sources: %w", err)
	}
	if len(result.Items) > options.Limit {
		result.Items = result.Items[:options.Limit]
		result.Truncated = true
		result.NextAfter = result.Items[len(result.Items)-1].Path
	}
	return result, nil
}

// ReadIndexed returns only content already stored in a fresh projection.
func ReadIndexed(ctx context.Context, root string, request contextview.ReadRequest) (contextview.ReadResult, error) {
	if err := validateRelativeContextPath(request.Path, false); err != nil {
		return contextview.ReadResult{}, err
	}
	byChunk := request.ChunkID != ""
	byLines := request.StartLine > 0 || request.EndLine > 0
	if byChunk == byLines {
		return contextview.ReadResult{}, errors.New("context read requires exactly one chunk id or line range")
	}
	if byLines && (request.StartLine < 1 || request.EndLine < request.StartLine || request.EndLine-request.StartLine+1 > contextview.MaxReadLines) {
		return contextview.ReadResult{}, errors.New("context read line range must contain between 1 and 400 lines")
	}
	if request.MaxBytes == 0 {
		request.MaxBytes = contextview.DefaultRead
	}
	if request.MaxBytes < 1 || request.MaxBytes > contextview.MaxReadBytes {
		return contextview.ReadResult{}, errors.New("context read byte budget must be between 1 and 32768")
	}
	status, err := Inspect(ctx, root)
	if err != nil {
		return contextview.ReadResult{}, err
	}
	if !status.Exists || !status.Supported || !status.Fresh {
		return contextview.ReadResult{}, errors.New("a fresh supported context index is required; run context rebuild")
	}
	_, databasePath, err := validatedPaths(root)
	if err != nil {
		return contextview.ReadResult{}, err
	}
	db, err := openDatabase(databasePath, true)
	if err != nil {
		return contextview.ReadResult{}, err
	}
	defer db.Close()
	var sourceDigest string
	if err := db.QueryRowContext(ctx, `SELECT source_sha256 FROM sources WHERE path = ?`, request.Path).Scan(&sourceDigest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contextview.ReadResult{}, errors.New("context source is not indexed")
		}
		return contextview.ReadResult{}, fmt.Errorf("read context source identity: %w", err)
	}
	var content string
	startLine, endLine := request.StartLine, request.EndLine
	if byChunk {
		if err := db.QueryRowContext(ctx, `SELECT start_line, end_line, content FROM chunks WHERE source_path = ? AND chunk_id = ?`, request.Path, request.ChunkID).Scan(&startLine, &endLine, &content); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return contextview.ReadResult{}, errors.New("context chunk is not indexed for the requested path")
			}
			return contextview.ReadResult{}, fmt.Errorf("read context chunk: %w", err)
		}
	} else {
		rows, err := db.QueryContext(ctx, `SELECT start_line, end_line, content FROM chunks
			WHERE source_path = ? AND end_line >= ? AND start_line <= ? ORDER BY ordinal`, request.Path, request.StartLine, request.EndLine)
		if err != nil {
			return contextview.ReadResult{}, fmt.Errorf("read context line range: %w", err)
		}
		defer rows.Close()
		var builder strings.Builder
		for rows.Next() {
			var chunkStart, chunkEnd int
			var chunkContent string
			if err := rows.Scan(&chunkStart, &chunkEnd, &chunkContent); err != nil {
				return contextview.ReadResult{}, fmt.Errorf("scan context line range: %w", err)
			}
			builder.WriteString(extractLineRange(chunkContent, chunkStart, request.StartLine, request.EndLine))
		}
		if err := rows.Err(); err != nil {
			return contextview.ReadResult{}, fmt.Errorf("iterate context line range: %w", err)
		}
		content = builder.String()
	}
	if content == "" {
		return contextview.ReadResult{}, errors.New("context read selected no indexed content")
	}
	bounded, truncated := boundedUTF8Prefix(content, request.MaxBytes)
	sum := sha256.Sum256([]byte(bounded))
	result := contextview.ReadResult{SchemaVersion: contextview.ReadSchema, Path: request.Path, SourceSHA256: sourceDigest, ContentSHA256: "sha256:" + hex.EncodeToString(sum[:]), StartLine: startLine, EndLine: endLine, Content: bounded, AppliedMaxBytes: request.MaxBytes, AppliedBytes: len(bounded), Truncated: truncated}
	if truncated {
		result.NextStartLine = startLine + strings.Count(bounded, "\n")
	}
	return result, nil
}

// Tree synthesizes a bounded hierarchy only from paths already present in a
// fresh source projection.
func Tree(ctx context.Context, root string, options contextview.TreeOptions) (contextview.TreeResult, error) {
	if options.MaxDepth == 0 {
		options.MaxDepth = contextview.DefaultTreeDepth
	}
	if options.MaxDepth < 1 || options.MaxDepth > contextview.MaxTreeDepth {
		return contextview.TreeResult{}, errors.New("workspace tree depth must be between 1 and 8")
	}
	if options.Limit == 0 {
		options.Limit = contextview.DefaultTreeEntries
	}
	if options.Limit < 1 || options.Limit > contextview.MaxTreeEntries {
		return contextview.TreeResult{}, errors.New("workspace tree limit must be between 1 and 500")
	}
	for _, value := range []string{options.PathPrefix, options.After} {
		if err := validateRelativeContextPath(value, true); err != nil {
			return contextview.TreeResult{}, err
		}
	}
	status, err := Inspect(ctx, root)
	if err != nil {
		return contextview.TreeResult{}, err
	}
	if !status.Exists || !status.Supported || !status.Fresh {
		return contextview.TreeResult{}, errors.New("a fresh supported context index is required; run context rebuild")
	}
	if options.ProjectionFingerprint != "" && options.ProjectionFingerprint != status.ProjectionFingerprint {
		return contextview.TreeResult{}, errors.New("workspace tree requires the current projection fingerprint")
	}
	if options.After != "" && options.ProjectionFingerprint == "" {
		return contextview.TreeResult{}, errors.New("workspace tree continuation requires the current projection fingerprint")
	}
	_, databasePath, err := validatedPaths(root)
	if err != nil {
		return contextview.TreeResult{}, err
	}
	db, err := openDatabase(databasePath, true)
	if err != nil {
		return contextview.TreeResult{}, err
	}
	defer db.Close()

	statement := `SELECT path FROM sources`
	arguments := make([]any, 0, 2)
	if options.PathPrefix != "" {
		statement += ` WHERE path = ? OR path LIKE ? ESCAPE '\'`
		arguments = append(arguments, options.PathPrefix, escapeLike(options.PathPrefix)+"/%")
	}
	statement += ` ORDER BY path`
	rows, err := db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return contextview.TreeResult{}, fmt.Errorf("list indexed paths for workspace tree: %w", err)
	}
	defer rows.Close()

	entries := make(map[string]string)
	for rows.Next() {
		var indexedPath string
		if err := rows.Scan(&indexedPath); err != nil {
			return contextview.TreeResult{}, fmt.Errorf("read indexed path for workspace tree: %w", err)
		}
		relative := indexedPath
		if options.PathPrefix != "" {
			if indexedPath == options.PathPrefix {
				relative = ""
			} else {
				relative = strings.TrimPrefix(indexedPath, options.PathPrefix+"/")
			}
		}
		if relative == "" {
			entries[indexedPath] = "source"
			continue
		}
		parts := strings.Split(relative, "/")
		directoryCount := len(parts) - 1
		if directoryCount > options.MaxDepth {
			directoryCount = options.MaxDepth
		}
		for depth := 1; depth <= directoryCount; depth++ {
			path := strings.Join(parts[:depth], "/")
			if options.PathPrefix != "" {
				path = options.PathPrefix + "/" + path
			}
			entries[path] = "directory"
		}
		if len(parts) <= options.MaxDepth {
			entries[indexedPath] = "source"
		}
	}
	if err := rows.Err(); err != nil {
		return contextview.TreeResult{}, fmt.Errorf("iterate indexed paths for workspace tree: %w", err)
	}

	paths := make([]string, 0, len(entries))
	for path := range entries {
		if options.After == "" || path > options.After {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	result := contextview.TreeResult{
		SchemaVersion:         contextview.TreeSchema,
		ProjectionFingerprint: status.ProjectionFingerprint,
		PathPrefix:            options.PathPrefix,
		AppliedDepth:          options.MaxDepth,
		AppliedLimit:          options.Limit,
		Entries:               []contextview.TreeEntry{},
	}
	if len(paths) > options.Limit {
		paths = paths[:options.Limit]
		result.Truncated = true
		result.NextAfter = paths[len(paths)-1]
	}
	for _, path := range paths {
		result.Entries = append(result.Entries, contextview.TreeEntry{Path: path, Kind: entries[path]})
	}
	return result, nil
}

func populate(ctx context.Context, db *sql.DB, inventory sourceInventory, rebuiltAt string) error {
	statements := []string{
		`PRAGMA journal_mode = DELETE`,
		`PRAGMA synchronous = FULL`,
		`PRAGMA foreign_keys = ON`,
		`PRAGMA user_version = 2`,
		`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL) STRICT`,
		`CREATE TABLE sources (
			path TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			language TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			modified_ns INTEGER NOT NULL,
			source_sha256 TEXT NOT NULL,
			normalized_sha256 TEXT,
			chunk_count INTEGER NOT NULL
		) STRICT`,
		`CREATE TABLE chunks (
			rowid INTEGER PRIMARY KEY,
			chunk_id TEXT NOT NULL UNIQUE,
			source_path TEXT NOT NULL REFERENCES sources(path) ON DELETE CASCADE,
			ordinal INTEGER NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL,
			content_sha256 TEXT NOT NULL,
			content TEXT NOT NULL,
			UNIQUE(source_path, ordinal)
		) STRICT`,
		`CREATE TABLE exclusion_summary (reason TEXT PRIMARY KEY, count INTEGER NOT NULL) STRICT`,
		`CREATE INDEX sources_kind_language_path_idx ON sources(kind, language, path)`,
		`CREATE INDEX chunks_source_ordinal_idx ON chunks(source_path, ordinal)`,
		`CREATE VIRTUAL TABLE chunk_search USING fts5(content, content='chunks', content_rowid='rowid', tokenize='unicode61')`,
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

	insertSource, err := tx.PrepareContext(ctx, `INSERT INTO sources(
		path, kind, language, size_bytes, modified_ns, source_sha256,
		normalized_sha256, chunk_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare context source insertion: %w", err)
	}
	defer insertSource.Close()
	insertChunk, err := tx.PrepareContext(ctx, `INSERT INTO chunks(
		chunk_id, source_path, ordinal, start_line, end_line, content_sha256, content)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare context chunk insertion: %w", err)
	}
	defer insertChunk.Close()
	for _, item := range inventory.Sources {
		var normalized any
		if item.NormalizedSHA256 != "" {
			normalized = item.NormalizedSHA256
		}
		if _, err := insertSource.ExecContext(ctx, item.Path, item.Kind, item.Language, item.SizeBytes, item.ModifiedNS, item.SHA256, normalized, len(item.Chunks)); err != nil {
			return fmt.Errorf("insert context source %s: %w", item.Path, err)
		}
		for _, chunk := range item.Chunks {
			result, err := insertChunk.ExecContext(ctx, chunk.ID, item.Path, chunk.Ordinal, chunk.StartLine, chunk.EndLine, chunk.ContentSHA256, chunk.Content)
			if err != nil {
				return fmt.Errorf("insert context chunk %s: %w", item.Path, err)
			}
			rowID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("identify context chunk %s: %w", item.Path, err)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO chunk_search(rowid, content) VALUES (?, ?)`, rowID, chunk.Content); err != nil {
				return fmt.Errorf("index context chunk %s: %w", item.Path, err)
			}
		}
	}

	metadata := map[string]string{
		"schema_version":       indexSchema,
		"policy_version":       policyVersion,
		"configuration_sha256": inventory.ConfigSHA256,
		"source_fingerprint":   inventory.Fingerprint,
		"rebuilt_at":           rebuiltAt,
		"artifact_count":       strconv.Itoa(len(inventory.Sources)),
		"source_count":         strconv.Itoa(len(inventory.Sources)),
		"source_bytes":         strconv.FormatInt(inventory.SourceBytes, 10),
		"chunk_count":          strconv.Itoa(inventory.ChunkCount),
		"excluded_count":       strconv.Itoa(exclusionTotal(inventory.Exclusions)),
	}
	for key, value := range metadata {
		if _, err := tx.ExecContext(ctx, `INSERT INTO metadata(key, value) VALUES (?, ?)`, key, value); err != nil {
			return fmt.Errorf("insert context metadata %s: %w", key, err)
		}
	}
	reasons := make([]string, 0, len(inventory.Exclusions))
	for reason := range inventory.Exclusions {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		if _, err := tx.ExecContext(ctx, `INSERT INTO exclusion_summary(reason, count) VALUES (?, ?)`, reason, inventory.Exclusions[reason]); err != nil {
			return fmt.Errorf("insert context exclusion summary %s: %w", reason, err)
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
	validatedRoot, err := workspace.ValidateRoot(root)
	if err != nil {
		return "", "", err
	}
	snapshot, err := workspace.Inspect(validatedRoot)
	if err != nil {
		return "", "", err
	}
	if !snapshot.HasEngkit {
		return "", "", errors.New("workspace is not initialized; run workspace plan-init and workspace init first")
	}
	resolvedRoot := filepath.FromSlash(snapshot.Root)
	return resolvedRoot, filepath.Join(resolvedRoot, ".engkit", "context", databaseName), nil
}

func exclusionTotal(exclusions map[string]int) int {
	total := 0
	for _, count := range exclusions {
		total += count
	}
	return total
}

func validateRelativeContextPath(value string, allowEmpty bool) error {
	if value == "" && allowEmpty {
		return nil
	}
	if value == "" || len([]byte(value)) > contextview.MaxPathBytes || strings.ContainsAny(value, "\\\x00") || filepath.IsAbs(value) {
		return errors.New("context path must be a normalized workspace-relative path of at most 512 bytes")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean != value || value == "." || value == ".." || strings.HasPrefix(value, "../") || (len(value) >= 2 && value[1] == ':') {
		return errors.New("context path must be a normalized workspace-relative path of at most 512 bytes")
	}
	return nil
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func compactSnippet(content, query string, limit int) (string, bool) {
	if limit <= 0 {
		return "", content != ""
	}
	lower := strings.ToLower(content)
	start := 0
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if index := strings.Index(lower, term); index >= 0 {
			start = index - limit/4
			if start < 0 {
				start = 0
			}
			break
		}
	}
	for start > 0 && (content[start]&0xc0) == 0x80 {
		start--
	}
	value, tailTruncated := boundedUTF8Prefix(content[start:], limit)
	return value, start > 0 || tailTruncated
}

func extractLineRange(content string, chunkStart, requestedStart, requestedEnd int) string {
	if chunkStart == requestedStart && requestedEnd >= chunkStart+strings.Count(content, "\n") {
		return content
	}
	lines := strings.SplitAfter(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var builder strings.Builder
	for index, line := range lines {
		lineNumber := chunkStart + index
		if lineNumber >= requestedStart && lineNumber <= requestedEnd {
			builder.WriteString(line)
		}
	}
	return builder.String()
}
