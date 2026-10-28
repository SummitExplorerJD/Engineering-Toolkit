// Package contextview defines application-owned read models for the optional
// rebuildable context projection.
package contextview

import "context"

const (
	StatusSchema       = "engkit.context-status/v1alpha1"
	SourcesSchema      = "engkit.context-sources/v1alpha1"
	ReadSchema         = "engkit.context-read/v1alpha1"
	TreeSchema         = "engkit.workspace-tree/v1alpha1"
	MaxPathBytes       = 512
	MaxPage            = 200
	DefaultPage        = 50
	MaxReadBytes       = 32 << 10
	DefaultRead        = 8 << 10
	MaxReadLines       = 400
	MaxTreeEntries     = 500
	DefaultTreeEntries = 100
	MaxTreeDepth       = 8
	DefaultTreeDepth   = 3
)

type Status struct {
	SchemaVersion         string `json:"schema_version"`
	DatabasePath          string `json:"database_path,omitempty"`
	Exists                bool   `json:"exists"`
	Supported             bool   `json:"supported"`
	Fresh                 bool   `json:"fresh"`
	Reason                string `json:"reason,omitempty"`
	PolicyVersion         string `json:"policy_version,omitempty"`
	ProjectionFingerprint string `json:"projection_fingerprint,omitempty"`
	SourceFingerprint     string `json:"source_fingerprint,omitempty"`
	CurrentFingerprint    string `json:"current_fingerprint,omitempty"`
	ConfiguredRootCount   int    `json:"configured_root_count"`
	ArtifactCount         int    `json:"artifact_count"`
	SourceCount           int    `json:"source_count"`
	SourceBytes           int64  `json:"source_bytes"`
	ChunkCount            int    `json:"chunk_count"`
	ExcludedCount         int    `json:"excluded_count"`
	IndexBytes            int64  `json:"index_bytes"`
	RebuiltAt             string `json:"rebuilt_at,omitempty"`
}

type SourceOptions struct {
	PathPrefix            string
	Kind                  string
	Language              string
	Limit                 int
	After                 string
	ProjectionFingerprint string
}

type Source struct {
	Path             string `json:"path"`
	Kind             string `json:"kind"`
	Language         string `json:"language"`
	SizeBytes        int64  `json:"size_bytes"`
	SourceSHA256     string `json:"source_sha256"`
	NormalizedSHA256 string `json:"normalized_sha256,omitempty"`
	ChunkCount       int    `json:"chunk_count"`
}

type Sources struct {
	SchemaVersion         string   `json:"schema_version"`
	ProjectionFingerprint string   `json:"projection_fingerprint"`
	AppliedLimit          int      `json:"applied_limit"`
	Items                 []Source `json:"items"`
	Truncated             bool     `json:"truncated"`
	NextAfter             string   `json:"next_after,omitempty"`
}

type ReadRequest struct {
	Path      string
	ChunkID   string
	StartLine int
	EndLine   int
	MaxBytes  int
}

type ReadResult struct {
	SchemaVersion   string `json:"schema_version"`
	Path            string `json:"path"`
	SourceSHA256    string `json:"source_sha256"`
	ContentSHA256   string `json:"content_sha256"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	Content         string `json:"content"`
	AppliedMaxBytes int    `json:"applied_max_bytes"`
	AppliedBytes    int    `json:"applied_bytes"`
	Truncated       bool   `json:"truncated"`
	NextStartLine   int    `json:"next_start_line,omitempty"`
}

type TreeOptions struct {
	PathPrefix            string
	MaxDepth              int
	Limit                 int
	After                 string
	ProjectionFingerprint string
}

type TreeEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type TreeResult struct {
	SchemaVersion         string      `json:"schema_version"`
	ProjectionFingerprint string      `json:"projection_fingerprint"`
	PathPrefix            string      `json:"path_prefix,omitempty"`
	AppliedDepth          int         `json:"applied_depth"`
	AppliedLimit          int         `json:"applied_limit"`
	Entries               []TreeEntry `json:"entries"`
	Truncated             bool        `json:"truncated"`
	NextAfter             string      `json:"next_after,omitempty"`
}

// Reader is the storage-independent application boundary used by future CLI
// and MCP presentation adapters.
type Reader interface {
	Status(context.Context, string) (Status, error)
	Sources(context.Context, string, SourceOptions) (Sources, error)
	Read(context.Context, string, ReadRequest) (ReadResult, error)
	Tree(context.Context, string, TreeOptions) (TreeResult, error)
}

// ReaderFuncs adapts application functions without exposing infrastructure to
// presentation layers.
type ReaderFuncs struct {
	StatusFunc  func(context.Context, string) (Status, error)
	SourcesFunc func(context.Context, string, SourceOptions) (Sources, error)
	ReadFunc    func(context.Context, string, ReadRequest) (ReadResult, error)
	TreeFunc    func(context.Context, string, TreeOptions) (TreeResult, error)
}

func (r ReaderFuncs) Status(ctx context.Context, root string) (Status, error) {
	return r.StatusFunc(ctx, root)
}

func (r ReaderFuncs) Sources(ctx context.Context, root string, options SourceOptions) (Sources, error) {
	return r.SourcesFunc(ctx, root, options)
}

func (r ReaderFuncs) Read(ctx context.Context, root string, request ReadRequest) (ReadResult, error) {
	return r.ReadFunc(ctx, root, request)
}

func (r ReaderFuncs) Tree(ctx context.Context, root string, options TreeOptions) (TreeResult, error) {
	return r.TreeFunc(ctx, root, options)
}
