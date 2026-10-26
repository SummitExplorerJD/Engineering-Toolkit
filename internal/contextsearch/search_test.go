package contextsearch

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSearchValidation(t *testing.T) {
	backend := SearchFunc(func(_ context.Context, _, query string) (Response, error) {
		return Response{SchemaVersion: Schema, Query: query, Fresh: true, Results: []Result{}}, nil
	})
	if _, err := Search(context.Background(), backend, "root", "  test  "); err != nil {
		t.Fatal(err)
	}
	if _, err := Search(context.Background(), backend, "root", " "); err == nil {
		t.Fatal("empty query succeeded")
	}
	if _, err := Search(context.Background(), backend, "root", strings.Repeat("x", MaxQueryBytes+1)); err == nil {
		t.Fatal("oversized query succeeded")
	}
}

func TestSearchRejectsStaleAndBackendFailure(t *testing.T) {
	stale := SearchFunc(func(context.Context, string, string) (Response, error) {
		return Response{SchemaVersion: Schema, Fresh: false}, nil
	})
	if _, err := Search(context.Background(), stale, "root", "test"); err == nil {
		t.Fatal("stale result succeeded")
	}
	want := errors.New("failed")
	failing := SearchFunc(func(context.Context, string, string) (Response, error) {
		return Response{}, want
	})
	if _, err := Search(context.Background(), failing, "root", "test"); !errors.Is(err, want) {
		t.Fatalf("Search() error = %v", err)
	}
}
