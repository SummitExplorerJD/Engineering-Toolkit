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
	Schema           = "engkit.context-index/v1alpha1"
	MaxQueryBytes    = 512
	MaxResults       = 50
	MaxResponseBytes = 1 << 20
)

type Result struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Snippet string `json:"snippet"`
}

type Response struct {
	SchemaVersion string   `json:"schema_version"`
	Query         string   `json:"query"`
	Fresh         bool     `json:"fresh"`
	Results       []Result `json:"results"`
}

type Searcher interface {
	Search(context.Context, string, string) (Response, error)
}

type SearchFunc func(context.Context, string, string) (Response, error)

func (f SearchFunc) Search(ctx context.Context, root, query string) (Response, error) {
	return f(ctx, root, query)
}

// Search validates application limits and rejects stale infrastructure output.
func Search(ctx context.Context, backend Searcher, root, query string) (Response, error) {
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
	result, err := backend.Search(ctx, root, query)
	if err != nil {
		return Response{}, err
	}
	if result.SchemaVersion != Schema {
		return Response{}, errors.New("context search returned an unsupported schema")
	}
	if !result.Fresh {
		return Response{}, errors.New("context index is stale; run context rebuild")
	}
	if len(result.Results) > MaxResults {
		return Response{}, errors.New("context search exceeded the 50-result limit")
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
