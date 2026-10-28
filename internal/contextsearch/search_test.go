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

func TestSearchWithOptionsAppliesBoundsAndDelegates(t *testing.T) {
	want := Options{PathPrefix: "internal", Kind: "source", Language: "go", Limit: 2, MaxContentBytes: 256}
	backend := OptionsSearchFunc(func(_ context.Context, _, query string, got Options) (Response, error) {
		if got != want {
			t.Fatalf("options = %#v, want %#v", got, want)
		}
		return Response{SchemaVersion: Schema, Query: query, Fresh: true, Results: []Result{}, AppliedLimit: got.Limit, AppliedMaxBytes: got.MaxContentBytes}, nil
	})
	if _, err := SearchWithOptions(context.Background(), backend, "root", "needle", want); err != nil {
		t.Fatal(err)
	}
	for _, options := range []Options{{Limit: MaxResults + 1}, {MaxContentBytes: MaxContentBytes + 1}} {
		if _, err := SearchWithOptions(context.Background(), backend, "root", "needle", options); err == nil {
			t.Fatalf("options %#v succeeded", options)
		}
	}
}
