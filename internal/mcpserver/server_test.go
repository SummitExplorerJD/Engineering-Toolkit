package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextview"
)

func TestServerAdvertisesOnlyFixedReadOnlyTools(t *testing.T) {
	server := newTestServer(t, nil)
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()

	result, err := client.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || !tool.Annotations.IdempotentHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			t.Fatalf("unsafe annotations for %s: %#v", tool.Name, tool.Annotations)
		}
	}
	sort.Strings(names)
	want := []string{"catalog_inspect", "context_read", "context_search", "context_sources", "context_status", "engkit_doctor", "plan_inspect", "profiles_recommend", "skills_list", "skills_validate", "workspace_inspect", "workspace_tree"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("tools = %#v, want %#v", names, want)
	}
	caps := client.InitializeResult().Capabilities
	if caps == nil || caps.Tools == nil || caps.Tools.ListChanged || caps.Logging != nil || caps.Prompts != nil || caps.Resources != nil || caps.Completions != nil {
		t.Fatalf("capabilities = %#v", caps)
	}
}

func TestWorkspaceAndContextResultsAreStructured(t *testing.T) {
	searcher := contextsearch.SearchFunc(func(_ context.Context, root, query string) (contextsearch.Response, error) {
		return contextsearch.Response{SchemaVersion: contextsearch.Schema, Query: query, Fresh: true, Results: []contextsearch.Result{{Path: "AGENTS.md", Kind: "markdown", Snippet: root}}}, nil
	})
	server := newTestServer(t, searcher)
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()

	workspaceResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "workspace_inspect", Arguments: map[string]any{}})
	if err != nil || workspaceResult.IsError || workspaceResult.StructuredContent == nil {
		t.Fatalf("workspace_inspect result = %#v, err = %v", workspaceResult, err)
	}
	data, err := json.Marshal(workspaceResult.StructuredContent)
	if err != nil || !json.Valid(data) {
		t.Fatalf("structured result = %s, err = %v", data, err)
	}

	searchResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_search", Arguments: map[string]any{"query": "guidance"}})
	if err != nil || searchResult.IsError || searchResult.StructuredContent == nil {
		t.Fatalf("context_search result = %#v, err = %v", searchResult, err)
	}
}

func TestContextResponseModesDoNotDuplicateExcerpt(t *testing.T) {
	server := newTestServer(t, contextsearch.OptionsSearchFunc(func(_ context.Context, _, query string, options contextsearch.Options) (contextsearch.Response, error) {
		return contextsearch.Response{SchemaVersion: contextsearch.Schema, Query: query, Fresh: true, Results: []contextsearch.Result{}, AppliedLimit: options.Limit, AppliedMaxBytes: options.MaxContentBytes}, nil
	}))
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()

	compact, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_read", Arguments: map[string]any{"path": "internal/example.go", "chunk": testProjection}})
	if err != nil || compact.IsError || compact.StructuredContent == nil {
		t.Fatalf("compact result = %#v, err = %v", compact, err)
	}
	compactText := resultText(t, compact)
	if strings.Contains(compactText, "unique indexed excerpt") || len([]byte(compactText)) > MaxSummaryBytes {
		t.Fatalf("compact text duplicated content or exceeded its bound: %q", compactText)
	}

	textResult, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_read", Arguments: map[string]any{"path": "internal/example.go", "chunk": testProjection, "response_mode": "text"}})
	if err != nil || textResult.IsError || textResult.StructuredContent != nil {
		t.Fatalf("text result = %#v, err = %v", textResult, err)
	}
	if got := strings.Count(resultText(t, textResult), "unique indexed excerpt"); got != 1 {
		t.Fatalf("text response contains excerpt %d times", got)
	}
}

func TestContextErrorsAreSafeAndEncodedLimitsFailClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		reader contextview.Reader
		check  func(*testing.T, *mcp.CallToolResult, string)
	}{
		{
			name: "root is scrubbed",
			reader: contextview.ReaderFuncs{ReadFunc: func(_ context.Context, root string, _ contextview.ReadRequest) (contextview.ReadResult, error) {
				return contextview.ReadResult{}, errors.New(root + `\private failure`)
			}},
			check: func(t *testing.T, result *mcp.CallToolResult, root string) {
				if strings.Contains(resultText(t, result), root) {
					t.Fatalf("error exposed root: %q", resultText(t, result))
				}
			},
		},
		{
			name: "encoded ceiling",
			reader: contextview.ReaderFuncs{ReadFunc: func(context.Context, string, contextview.ReadRequest) (contextview.ReadResult, error) {
				return contextview.ReadResult{SchemaVersion: contextview.ReadSchema, Path: "internal/example.go", Content: strings.Repeat("x", MaxReadResponseBytes), AppliedBytes: MaxReadResponseBytes}, nil
			}},
			check: func(t *testing.T, result *mcp.CallToolResult, _ string) {
				if !strings.Contains(resultText(t, result), "protocol limit") {
					t.Fatalf("unexpected limit error: %q", resultText(t, result))
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			server, err := New(root, Dependencies{Version: "v0.4.0-test", ContextReader: test.reader})
			if err != nil {
				t.Fatal(err)
			}
			client, closeSessions := connectTestClient(t, server)
			defer closeSessions()
			result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "context_read", Arguments: map[string]any{"path": "internal/example.go", "chunk": testProjection}})
			if err != nil || !result.IsError || result.StructuredContent != nil {
				t.Fatalf("result = %#v, err = %v", result, err)
			}
			test.check(t, result, server.root)
		})
	}
}

func TestToolInputSchemasRejectUnknownAndOversizedValues(t *testing.T) {
	server := newTestServer(t, contextsearch.SearchFunc(func(context.Context, string, string) (contextsearch.Response, error) {
		return contextsearch.Response{SchemaVersion: contextsearch.Schema, Fresh: true, Results: []contextsearch.Result{}}, nil
	}))
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()

	for _, params := range []*mcp.CallToolParams{
		{Name: "workspace_inspect", Arguments: map[string]any{"unknown": true}},
		{Name: "context_search", Arguments: map[string]any{}},
		{Name: "context_search", Arguments: map[string]any{"query": "x", "limit": 0}},
		{Name: "context_sources", Arguments: map[string]any{"after": "internal/z.go"}},
		{Name: "context_read", Arguments: map[string]any{"path": "../escape", "start_line": 1, "end_line": 1}},
		{Name: "workspace_tree", Arguments: map[string]any{"max_depth": 9}},
		{Name: "skills_validate", Arguments: map[string]any{"id": "INVALID"}},
	} {
		if result, err := client.CallTool(context.Background(), params); err == nil {
			t.Fatalf("CallTool(%s) succeeded with result %#v", params.Name, result)
		}
	}
}

func TestCancellationReachesApplicationOperation(t *testing.T) {
	cancelled := make(chan struct{})
	searcher := contextsearch.SearchFunc(func(ctx context.Context, _, _ string) (contextsearch.Response, error) {
		<-ctx.Done()
		close(cancelled)
		return contextsearch.Response{}, ctx.Err()
	})
	server := newTestServer(t, searcher)
	client, closeSessions := connectTestClient(t, server)
	defer closeSessions()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _ = client.CallTool(ctx, &mcp.CallToolParams{Name: "context_search", Arguments: map[string]any{"query": "cancel"}})
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("application operation did not observe cancellation")
	}
}

func TestIOTransportOutputContainsOnlyJSONRPC(t *testing.T) {
	server := newTestServer(t, nil)
	clientToServerReader, clientToServerWriter := io.Pipe()
	serverToClientReader, serverToClientWriter := io.Pipe()
	var output bytes.Buffer
	serverTransport := &mcp.IOTransport{
		Reader:        clientToServerReader,
		Writer:        teeWriteCloser{Writer: io.MultiWriter(serverToClientWriter, &output), Closer: serverToClientWriter},
		MaxLineLength: MaxMessageBytes,
	}
	clientTransport := &mcp.IOTransport{
		Reader:        serverToClientReader,
		Writer:        clientToServerWriter,
		MaxLineLength: MaxMessageBytes,
	}

	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Run(context.Background(), serverTransport) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "engkit-test", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ListTools(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after the client closed the stream")
	}

	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte{'\n'})
	if len(lines) < 2 {
		t.Fatalf("protocol output contains %d frames, want at least 2", len(lines))
	}
	for _, line := range lines {
		if !json.Valid(line) || !bytes.Contains(line, []byte(`"jsonrpc":"2.0"`)) {
			t.Fatalf("non-JSON-RPC standard-output frame: %q", line)
		}
	}
}

type teeWriteCloser struct {
	io.Writer
	io.Closer
}

func newTestServer(t *testing.T, searcher contextsearch.Searcher) *Server {
	t.Helper()
	server, err := New(t.TempDir(), Dependencies{Version: "v0.4.0-test", ContextSearch: searcher, ContextReader: testContextReader()})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

const testProjection = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testContextReader() contextview.Reader {
	return contextview.ReaderFuncs{
		StatusFunc: func(context.Context, string) (contextview.Status, error) {
			return contextview.Status{SchemaVersion: contextview.StatusSchema, Exists: true, Supported: true, Fresh: true, ProjectionFingerprint: testProjection, SourceCount: 1, ChunkCount: 1}, nil
		},
		SourcesFunc: func(_ context.Context, _ string, options contextview.SourceOptions) (contextview.Sources, error) {
			return contextview.Sources{SchemaVersion: contextview.SourcesSchema, ProjectionFingerprint: testProjection, Items: []contextview.Source{{Path: "internal/example.go", Kind: "source", Language: "go", ChunkCount: 1}}}, nil
		},
		ReadFunc: func(context.Context, string, contextview.ReadRequest) (contextview.ReadResult, error) {
			return contextview.ReadResult{SchemaVersion: contextview.ReadSchema, Path: "internal/example.go", SourceSHA256: testProjection, ContentSHA256: testProjection, StartLine: 1, EndLine: 1, Content: "unique indexed excerpt", AppliedBytes: 22}, nil
		},
		TreeFunc: func(context.Context, string, contextview.TreeOptions) (contextview.TreeResult, error) {
			return contextview.TreeResult{SchemaVersion: contextview.TreeSchema, ProjectionFingerprint: testProjection, AppliedDepth: 3, AppliedLimit: 100, Entries: []contextview.TreeEntry{{Path: "internal", Kind: "directory"}, {Path: "internal/example.go", Kind: "source"}}}, nil
		},
	}
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("content blocks = %d, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T", result.Content[0])
	}
	return text.Text
}

func connectTestClient(t *testing.T, server *Server) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.protocolServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "engkit-test", Version: "v1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		serverSession.Close()
		t.Fatal(err)
	}
	return clientSession, func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
}
