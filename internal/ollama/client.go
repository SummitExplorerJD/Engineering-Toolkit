// Package ollama implements the fixed-loopback local adviser adapter.
package ollama

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/localintelligence"
)

const (
	fixedEndpoint  = "http://127.0.0.1:11434"
	maxMessageSize = 256 << 10
	maxInventory   = 1 << 20
)

type Client struct {
	endpoint string
	http     *http.Client
	timeout  time.Duration
}

func New() *Client {
	transport := &http.Transport{
		Proxy:              nil,
		DialContext:        (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		DisableCompression: true,
	}
	return &Client{
		endpoint: fixedEndpoint,
		http: &http.Client{
			Transport:     transport,
			CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("Ollama redirects are not allowed") },
		},
		timeout: 60 * time.Second,
	}
}

func newTestClient(endpoint string, client *http.Client, timeout time.Duration) *Client {
	return &Client{endpoint: endpoint, http: client, timeout: timeout}
}

func (c *Client) Rank(ctx context.Context, model string, input localintelligence.Request) (localintelligence.ModelResponse, localintelligence.ModelEvidence, error) {
	if !allowedModel(model) {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("model tag is not allowlisted")
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var version struct {
		Version string `json:"version"`
	}
	if err := c.get(ctx, "/api/version", maxInventory, &version); err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, fmt.Errorf("inspect Ollama version: %w", err)
	}
	if strings.TrimSpace(version.Version) == "" {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("Ollama version is missing")
	}
	var tags struct {
		Models []struct {
			Name   string `json:"name"`
			Model  string `json:"model"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := c.get(ctx, "/api/tags", maxInventory, &tags); err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, fmt.Errorf("inspect Ollama tags: %w", err)
	}
	digest := ""
	matches := 0
	for _, item := range tags.Models {
		if item.Name == model || item.Model == model {
			digest = strings.TrimPrefix(strings.ToLower(item.Digest), "sha256:")
			matches++
		}
	}
	if matches != 1 || localintelligence.ValidateDigest("sha256:"+digest) != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("configured model tag did not resolve to one observed digest")
	}

	var schema any
	if err := json.Unmarshal([]byte(localintelligence.ResponseSchemaJSON), &schema); err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("internal response schema is invalid")
	}
	request := struct {
		Model     string    `json:"model"`
		Messages  []message `json:"messages"`
		Stream    bool      `json:"stream"`
		Think     bool      `json:"think"`
		Format    any       `json:"format"`
		KeepAlive int       `json:"keep_alive"`
		Options   struct {
			Temperature int `json:"temperature"`
		} `json:"options"`
	}{Model: model, Stream: false, Think: false, Format: schema, KeepAlive: 0}
	userJSON, err := json.Marshal(input)
	if err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("encode adviser input")
	}
	request.Messages = []message{{Role: "system", Content: localintelligence.SystemPrompt}, {Role: "user", Content: string(userJSON)}}
	encoded, err := json.Marshal(request)
	if err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("encode Ollama request")
	}
	if len(encoded) > maxMessageSize {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("Ollama request exceeds the 256 KiB limit")
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/chat", bytes.NewReader(encoded))
	if err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	data, err := c.do(httpRequest, maxMessageSize)
	if err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, fmt.Errorf("generate local advice: %w", err)
	}
	var outer struct {
		Message message `json:"message"`
		Done    bool    `json:"done"`
	}
	if err := json.Unmarshal(data, &outer); err != nil || !outer.Done || outer.Message.Role != "assistant" {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("Ollama returned an invalid completed chat response")
	}
	decoder := json.NewDecoder(strings.NewReader(outer.Message.Content))
	decoder.DisallowUnknownFields()
	var response localintelligence.ModelResponse
	if err := decoder.Decode(&response); err != nil {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("Ollama returned invalid structured advice")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return localintelligence.ModelResponse{}, localintelligence.ModelEvidence{}, errors.New("Ollama returned trailing structured content")
	}
	responseSum := sha256.Sum256([]byte(outer.Message.Content))
	return response, localintelligence.ModelEvidence{
		OllamaVersion: version.Version, ModelTag: model, ModelDigest: "sha256:" + digest,
		ResponseDigest: "sha256:" + hex.EncodeToString(responseSum[:]),
	}, nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *Client) get(ctx context.Context, path string, limit int64, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return err
	}
	data, err := c.do(request, limit)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return errors.New("response is not valid JSON")
	}
	return nil
}

func (c *Client) do(request *http.Request, limit int64) ([]byte, error) {
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
	}
	if response.ContentLength > limit {
		return nil, errors.New("response exceeds its size limit")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("response exceeds its size limit")
	}
	return data, nil
}

func allowedModel(model string) bool {
	return model == "phi4-mini:latest" || model == "qwen3.5:0.8b" || model == "granite4:3b"
}
