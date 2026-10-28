// Package contextsearch defines the application-owned bounded context-search
// use case. Storage implementations remain behind the Searcher interface.
package contextsearch

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

const (
	Schema              = "engkit.context-index/v1alpha1"
	MaxQueryBytes       = 512
	MaxResults          = 20
	DefaultResults      = 5
	DefaultSnippetBytes = 512
	MaxSnippetBytes     = 1024
	DefaultContentBytes = 4 << 10
	MaxContentBytes     = 16 << 10
	MaxResponseBytes    = 1 << 20
)

type Result struct {
	Path          string `json:"path"`
	Kind          string `json:"kind"`
	Language      string `json:"language,omitempty"`
	ChunkID       string `json:"chunk_id,omitempty"`
	StartLine     int    `json:"start_line,omitempty"`
	EndLine       int    `json:"end_line,omitempty"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
	Snippet       string `json:"snippet"`
	Truncated     bool   `json:"truncated,omitempty"`
}

type Response struct {
	SchemaVersion       string   `json:"schema_version"`
	Query               string   `json:"query"`
	Fresh               bool     `json:"fresh"`
	Results             []Result `json:"results"`
	AppliedLimit        int      `json:"applied_limit,omitempty"`
	AppliedMaxBytes     int      `json:"applied_max_content_bytes,omitempty"`
	AppliedContentBytes int      `json:"applied_content_bytes,omitempty"`
	Truncated           bool     `json:"truncated,omitempty"`
}

type Options struct {
	PathPrefix      string
	Kind            string
	Language        string
	Limit           int
	MaxContentBytes int
}

type Searcher interface {
	Search(context.Context, string, string) (Response, error)
}

type OptionsSearcher interface {
	SearchWithOptions(context.Context, string, string, Options) (Response, error)
}

type SearchFunc func(context.Context, string, string) (Response, error)

func (f SearchFunc) Search(ctx context.Context, root, query string) (Response, error) {
	return f(ctx, root, query)
}

type OptionsSearchFunc func(context.Context, string, string, Options) (Response, error)

func (f OptionsSearchFunc) Search(ctx context.Context, root, query string) (Response, error) {
	return f(ctx, root, query, Options{})
}

func (f OptionsSearchFunc) SearchWithOptions(ctx context.Context, root, query string, options Options) (Response, error) {
	return f(ctx, root, query, options)
}

// Search validates application limits and rejects stale infrastructure output.
func Search(ctx context.Context, backend Searcher, root, query string) (Response, error) {
	return SearchWithOptions(ctx, backend, root, query, Options{})
}

// SearchWithOptions validates bounded filters and rejects stale infrastructure
// output.
func SearchWithOptions(ctx context.Context, backend Searcher, root, query string, options Options) (Response, error) {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return Response{}, errors.New("search query is required")
	}
	if len([]byte(query)) > MaxQueryBytes {
		return Response{}, errors.New("search query exceeds the 512-byte limit")
	}
	if backend == nil {
		return Response{}, errors.New("context search is unavailable")
	}
	if options.Limit == 0 {
		options.Limit = DefaultResults
	}
	if options.Limit < 1 || options.Limit > MaxResults {
		return Response{}, errors.New("context search limit must be between 1 and 20")
	}
	if options.MaxContentBytes == 0 {
		options.MaxContentBytes = DefaultContentBytes
	}
	if options.MaxContentBytes < 1 || options.MaxContentBytes > MaxContentBytes {
		return Response{}, errors.New("context search content budget must be between 1 and 16384 bytes")
	}
	var result Response
	var err error
	if withOptions, ok := backend.(OptionsSearcher); ok {
		result, err = withOptions.SearchWithOptions(ctx, root, query, options)
	} else if options.PathPrefix == "" && options.Kind == "" && options.Language == "" && options.Limit == DefaultResults && options.MaxContentBytes == DefaultContentBytes {
		result, err = backend.Search(ctx, root, query)
	} else {
		return Response{}, errors.New("context search filters are unavailable")
	}
	if err != nil {
		return Response{}, err
	}
	if result.SchemaVersion != Schema {
		return Response{}, errors.New("context search returned an unsupported schema")
	}
	if !result.Fresh {
		return Response{}, errors.New("context index is stale; run context rebuild")
	}
	if len(result.Results) > options.Limit {
		return Response{}, errors.New("context search exceeded the applied result limit")
	}
	contentBytes := 0
	for _, item := range result.Results {
		if len([]byte(item.Snippet)) > MaxSnippetBytes {
			return Response{}, errors.New("context search exceeded the per-snippet limit")
		}
		contentBytes += len([]byte(item.Snippet))
	}
	if contentBytes > options.MaxContentBytes {
		return Response{}, errors.New("context search exceeded the applied content budget")
	}
	if result.AppliedLimit == 0 {
		result.AppliedLimit = options.Limit
	}
	if result.AppliedMaxBytes == 0 {
		result.AppliedMaxBytes = options.MaxContentBytes
	}
	result.AppliedContentBytes = contentBytes
	if result.AppliedLimit != options.Limit || result.AppliedMaxBytes != options.MaxContentBytes {
		return Response{}, errors.New("context search reported inconsistent applied limits")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return Response{}, errors.New("context search result could not be encoded")
	}
	if len(encoded) > MaxResponseBytes {
		return Response{}, errors.New("context search result exceeds the 1 MiB limit")
	}
	return result, nil
}
