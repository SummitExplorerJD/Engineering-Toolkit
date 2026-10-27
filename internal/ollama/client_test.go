package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/localintelligence"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonResponse(status int, value string, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(value)), ContentLength: int64(len(value))}
}

func TestClientUsesBoundedContract(t *testing.T) {
	digest := strings.Repeat("a", 64)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/api/version":
			return jsonResponse(200, `{"version":"0.40.4"}`, nil), nil
		case "/api/tags":
			return jsonResponse(200, `{"models":[{"name":"granite4:3b","digest":"`+digest+`"}]}`, nil), nil
		case "/api/chat":
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["stream"] != false || body["think"] != false || body["keep_alive"].(float64) != 0 || body["tools"] != nil {
				t.Errorf("unexpected request: %#v", body)
			}
			content := `{"rankings":[{"candidate_id":"a@1.0.0","score":90,"reason":"best"}],"uncertainty":"","warnings":[]}`
			encoded, _ := json.Marshal(map[string]any{"message": map[string]any{"role": "assistant", "content": content}, "done": true})
			return jsonResponse(200, string(encoded), nil), nil
		default:
			return jsonResponse(404, `{}`, nil), nil
		}
	})
	client := newTestClient("http://127.0.0.1:11434", &http.Client{Transport: transport}, time.Second)
	input := localintelligence.Request{ContractVersion: localintelligence.ContractVersion, Goal: "choose", Candidates: []localintelligence.Candidate{{ID: "a@1.0.0", DeterministicRank: 1}}}
	response, evidence, err := client.Rank(context.Background(), "granite4:3b", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Rankings) != 1 || evidence.ModelDigest != "sha256:"+digest || evidence.OllamaVersion != "0.40.4" {
		t.Fatalf("unexpected response: %#v %#v", response, evidence)
	}
}

func TestClientRejectsRedirectAndTimeout(t *testing.T) {
	redirectTransport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		header := make(http.Header)
		header.Set("Location", "http://127.0.0.1:11434/elsewhere")
		return jsonResponse(http.StatusFound, `{}`, header), nil
	})
	redirectClient := &http.Client{Transport: redirectTransport, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }}
	client := newTestClient("http://127.0.0.1:11434", redirectClient, time.Second)
	if _, _, err := client.Rank(context.Background(), "granite4:3b", localintelligence.Request{}); err == nil {
		t.Fatal("redirect was accepted")
	}

	slowTransport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		select {
		case <-request.Context().Done():
			return nil, request.Context().Err()
		case <-time.After(50 * time.Millisecond):
			return jsonResponse(200, `{}`, nil), nil
		}
	})
	client = newTestClient("http://127.0.0.1:11434", &http.Client{Transport: slowTransport}, 5*time.Millisecond)
	if _, _, err := client.Rank(context.Background(), "granite4:3b", localintelligence.Request{}); err == nil {
		t.Fatal("timeout was accepted")
	}
}

func TestProductionClientDisablesProxyAndRedirects(t *testing.T) {
	client := New()
	transport, ok := client.http.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || client.http.CheckRedirect == nil {
		t.Fatalf("unsafe production HTTP client: %#v", client.http)
	}
}

func TestClientRejectsOversizedInventory(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, strings.Repeat("x", maxInventory+1), nil), nil
	})
	client := newTestClient("http://127.0.0.1:11434", &http.Client{Transport: transport}, time.Second)
	if _, _, err := client.Rank(context.Background(), "granite4:3b", localintelligence.Request{}); err == nil {
		t.Fatal("oversized inventory was accepted")
	}
}
