package tufclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestTransportDownloadsWithLimitsAndUserAgent(t *testing.T) {
	var userAgent string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.UserAgent()
		_, _ = w.Write([]byte("verified"))
	}))
	defer server.Close()
	transport, err := NewTransport(server.URL+"/metadata", server.URL+"/targets", "engkit/v1.0.0 update-client/1", WithRoundTripper(server.Client().Transport))
	if err != nil {
		t.Fatal(err)
	}
	data, err := transport.Download(context.Background(), server.URL+"/metadata/timestamp.json", 64)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "verified" || userAgent != "engkit/v1.0.0 update-client/1" {
		t.Fatalf("unexpected response %q or user agent %q", data, userAgent)
	}
	if _, err := transport.Download(context.Background(), server.URL+"/metadata/timestamp.json", 2); err == nil {
		t.Fatal("expected size-limit error")
	}
}

func TestTransportRetriesOnlyTransientResponses(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	transport, err := NewTransport(server.URL, server.URL, "engkit/test", WithRoundTripper(server.Client().Transport))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.Download(context.Background(), server.URL+"/timestamp.json", 64); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}

	calls.Store(0)
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer bad.Close()
	transport, err = NewTransport(bad.URL, bad.URL, "engkit/test", WithRoundTripper(bad.Client().Transport))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.Download(context.Background(), bad.URL+"/timestamp.json", 64); err == nil {
		t.Fatal("expected HTTP error")
	}
	if calls.Load() != 1 {
		t.Fatalf("non-transient response retried %d times", calls.Load())
	}
}

func TestTransportRejectsUnsafeURLs(t *testing.T) {
	unsafe := []string{
		"http://example.com/metadata",
		"https://user@example.com/metadata",
		"https://example.com/%2e%2e/metadata",
		"https://example.com/a%5cb",
		"https://example.com/metadata#fragment",
	}
	for _, raw := range unsafe {
		t.Run(strings.ReplaceAll(raw, "/", "_"), func(t *testing.T) {
			if _, err := NewTransport(raw, "https://example.com/targets", "engkit/test"); err == nil {
				t.Fatalf("accepted unsafe URL %q", raw)
			}
		})
	}
}

func TestTransportRejectsCrossHostRedirect(t *testing.T) {
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("unexpected"))
	}))
	defer destination.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL+"/target", http.StatusFound)
	}))
	defer source.Close()
	transport, err := NewTransport(source.URL, source.URL, "engkit/test", WithRoundTripper(source.Client().Transport))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.Download(context.Background(), source.URL+"/target", 64); err == nil {
		t.Fatal("expected cross-host redirect rejection")
	}
}

func TestTransportBoundsSameHostRedirects(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step := 0
		_, _ = fmt.Sscanf(r.URL.Path, "/%d", &step)
		http.Redirect(w, r, fmt.Sprintf("%s/%d", server.URL, step+1), http.StatusFound)
	}))
	defer server.Close()
	transport, err := NewTransport(server.URL, server.URL, "engkit/test", WithRoundTripper(server.Client().Transport))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.Download(context.Background(), server.URL+"/0", 64); err == nil {
		t.Fatal("expected redirect-limit rejection")
	}
}
