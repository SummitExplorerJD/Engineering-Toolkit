// Package mcpserver exposes read-only Engineering Toolkit application use
// cases through Model Context Protocol.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextview"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/doctor"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/planinspect"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/profiles"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/schemautil"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	MaxMessageBytes         = 1 << 20
	MaxResponseBytes        = 1 << 20
	MaxStatusResponseBytes  = 16 << 10
	MaxSourcesResponseBytes = 64 << 10
	MaxSearchResponseBytes  = 32 << 10
	MaxReadResponseBytes    = 40 << 10
	MaxTreeResponseBytes    = 64 << 10
	MaxSummaryBytes         = 256
)

type Dependencies struct {
	Version       string
	ContextSearch contextsearch.Searcher
	ContextReader contextview.Reader
}

type Server struct {
	root           string
	version        string
	contextSearch  contextsearch.Searcher
	contextReader  contextview.Reader
	catalog        catalog.Catalog
	catalogFS      fs.FS
	protocolServer *mcp.Server
}

type emptyInput struct{}
type doctorInput struct {
	Scope string `json:"scope,omitempty"`
}
type queryInput struct {
	Query           string `json:"query"`
	PathPrefix      string `json:"path_prefix,omitempty"`
	Kind            string `json:"kind,omitempty"`
	Language        string `json:"language,omitempty"`
	Limit           *int   `json:"limit,omitempty"`
	MaxContentBytes *int   `json:"max_content_bytes,omitempty"`
	ResponseMode    string `json:"response_mode,omitempty"`
}
type contextStatusInput struct {
	ResponseMode string `json:"response_mode,omitempty"`
}
type contextSourcesInput struct {
	PathPrefix   string `json:"path_prefix,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Language     string `json:"language,omitempty"`
	Limit        *int   `json:"limit,omitempty"`
	After        string `json:"after,omitempty"`
	Projection   string `json:"projection,omitempty"`
	ResponseMode string `json:"response_mode,omitempty"`
}
type contextReadInput struct {
	Path         string `json:"path"`
	Chunk        string `json:"chunk,omitempty"`
	StartLine    *int   `json:"start_line,omitempty"`
	EndLine      *int   `json:"end_line,omitempty"`
	MaxBytes     *int   `json:"max_bytes,omitempty"`
	ResponseMode string `json:"response_mode,omitempty"`
}
type workspaceTreeInput struct {
	PathPrefix   string `json:"path_prefix,omitempty"`
	MaxDepth     *int   `json:"max_depth,omitempty"`
	Limit        *int   `json:"limit,omitempty"`
	After        string `json:"after,omitempty"`
	Projection   string `json:"projection,omitempty"`
	ResponseMode string `json:"response_mode,omitempty"`
}
type skillInput struct {
	ID string `json:"id,omitempty"`
}
type planInput struct {
	Path string `json:"path"`
}

var skillIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
var contextFilterPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// New validates and binds one server to one workspace root.
func New(root string, dependencies Dependencies) (*Server, error) {
	resolved, err := workspace.ValidateRoot(root)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dependencies.Version) == "" {
		return nil, errors.New("server version is required")
	}
	inventory, catalogFS, err := catalog.Builtin()
	if err != nil {
		return nil, fmt.Errorf("load packaged catalog: %w", err)
	}
	s := &Server{root: resolved, version: dependencies.Version, contextSearch: dependencies.ContextSearch, contextReader: dependencies.ContextReader, catalog: inventory, catalogFS: catalogFS}
	s.protocolServer = mcp.NewServer(&mcp.Implementation{Name: "engkit", Version: dependencies.Version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
	})
	s.registerTools()
	return s, nil
}

// Run serves one MCP session on the supplied transport.
func (s *Server) Run(ctx context.Context, transport mcp.Transport) error {
	if transport == nil {
		return errors.New("MCP transport is required")
	}
	return s.protocolServer.Run(ctx, transport)
}

// RunStdio serves one root over the process standard streams.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.Run(ctx, &mcp.StdioTransport{MaxLineLength: MaxMessageBytes})
}

func (s *Server) registerTools() {
	annotations := readOnlyAnnotations()
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "engkit_doctor", Description: "Run bounded read-only Engineering Toolkit diagnostics.", Annotations: annotations, InputSchema: schemaDoctor()}, s.doctor, validateDoctorInput)
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "workspace_inspect", Description: "Inspect deterministic workspace facts.", Annotations: annotations, InputSchema: schemaEmpty()}, s.workspaceInspect, noValidation[emptyInput])
	addContextTool(s.protocolServer, &mcp.Tool{Name: "context_status", Description: "Inspect content-free context projection freshness and counts.", Annotations: annotations, InputSchema: schemaContextStatus()}, s.contextStatusTool, validateContextStatusInput, MaxStatusResponseBytes, summarizeStatus)
	addContextTool(s.protocolServer, &mcp.Tool{Name: "context_sources", Description: "List bounded metadata for sources in a fresh context projection.", Annotations: annotations, InputSchema: schemaContextSources()}, s.contextSourcesTool, validateContextSourcesInput, MaxSourcesResponseBytes, summarizeSources)
	addContextTool(s.protocolServer, &mcp.Tool{Name: "context_search", Description: "Search a fresh optional contextual index through the application boundary.", Annotations: annotations, InputSchema: schemaQuery()}, s.contextSearchTool, validateQueryInput, MaxSearchResponseBytes, summarizeSearch)
	addContextTool(s.protocolServer, &mcp.Tool{Name: "context_read", Description: "Read one bounded excerpt already stored in a fresh context projection.", Annotations: annotations, InputSchema: schemaContextRead()}, s.contextReadTool, validateContextReadInput, MaxReadResponseBytes, summarizeRead)
	addContextTool(s.protocolServer, &mcp.Tool{Name: "workspace_tree", Description: "List a bounded hierarchy synthesized from fresh indexed source paths.", Annotations: annotations, InputSchema: schemaWorkspaceTree()}, s.workspaceTreeTool, validateWorkspaceTreeInput, MaxTreeResponseBytes, summarizeTree)
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "catalog_inspect", Description: "Inspect the embedded immutable first-party catalog.", Annotations: annotations, InputSchema: schemaEmpty()}, s.catalogInspect, noValidation[emptyInput])
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "profiles_recommend", Description: "Recommend compatible embedded project profiles deterministically.", Annotations: annotations, InputSchema: schemaEmpty()}, s.profilesRecommend, noValidation[emptyInput])
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "skills_list", Description: "List canonical skill packages without executing resources.", Annotations: annotations, InputSchema: schemaEmpty()}, s.skillsList, noValidation[emptyInput])
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "skills_validate", Description: "Validate one or all canonical skill packages without mutation.", Annotations: annotations, InputSchema: schemaSkill()}, s.skillsValidate, validateSkillInput)
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "plan_inspect", Description: "Validate and summarize one supported persisted plan without applying it.", Annotations: annotations, InputSchema: schemaPlan()}, s.planInspect, validatePlanInput)
}

func (s *Server) doctor(ctx context.Context, _ *mcp.CallToolRequest, input doctorInput) (*mcp.CallToolResult, doctor.Result, error) {
	scope := input.Scope
	if scope == "" {
		scope = doctor.ScopeAll
	}
	result, err := doctor.Inspect(ctx, doctor.Options{Root: s.root, Scope: scope, Version: s.version})
	return checked(s, result, err)
}

func (s *Server) workspaceInspect(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, workspace.Inspection, error) {
	result, err := workspace.Inspect(s.root)
	return checked(s, result, err)
}

func (s *Server) contextStatusTool(ctx context.Context, _ *mcp.CallToolRequest, _ contextStatusInput) (*mcp.CallToolResult, contextview.Status, error) {
	if s.contextReader == nil {
		return checked(s, contextview.Status{}, errors.New("context inspection is unavailable"))
	}
	result, err := s.contextReader.Status(ctx, s.root)
	return checked(s, result, err)
}

func (s *Server) contextSourcesTool(ctx context.Context, _ *mcp.CallToolRequest, input contextSourcesInput) (*mcp.CallToolResult, contextview.Sources, error) {
	if s.contextReader == nil {
		return checked(s, contextview.Sources{}, errors.New("context source inventory is unavailable"))
	}
	result, err := s.contextReader.Sources(ctx, s.root, contextview.SourceOptions{PathPrefix: input.PathPrefix, Kind: input.Kind, Language: input.Language, Limit: optionalInt(input.Limit), After: input.After, ProjectionFingerprint: input.Projection})
	return checked(s, result, err)
}

func (s *Server) contextSearchTool(ctx context.Context, _ *mcp.CallToolRequest, input queryInput) (*mcp.CallToolResult, contextsearch.Response, error) {
	result, err := contextsearch.SearchWithOptions(ctx, s.contextSearch, s.root, input.Query, contextsearch.Options{PathPrefix: input.PathPrefix, Kind: input.Kind, Language: input.Language, Limit: optionalInt(input.Limit), MaxContentBytes: optionalInt(input.MaxContentBytes)})
	return checked(s, result, err)
}

func (s *Server) contextReadTool(ctx context.Context, _ *mcp.CallToolRequest, input contextReadInput) (*mcp.CallToolResult, contextview.ReadResult, error) {
	if s.contextReader == nil {
		return checked(s, contextview.ReadResult{}, errors.New("context read is unavailable"))
	}
	result, err := s.contextReader.Read(ctx, s.root, contextview.ReadRequest{Path: input.Path, ChunkID: input.Chunk, StartLine: optionalInt(input.StartLine), EndLine: optionalInt(input.EndLine), MaxBytes: optionalInt(input.MaxBytes)})
	return checked(s, result, err)
}

func (s *Server) workspaceTreeTool(ctx context.Context, _ *mcp.CallToolRequest, input workspaceTreeInput) (*mcp.CallToolResult, contextview.TreeResult, error) {
	if s.contextReader == nil {
		return checked(s, contextview.TreeResult{}, errors.New("workspace tree is unavailable"))
	}
	result, err := s.contextReader.Tree(ctx, s.root, contextview.TreeOptions{PathPrefix: input.PathPrefix, MaxDepth: optionalInt(input.MaxDepth), Limit: optionalInt(input.Limit), After: input.After, ProjectionFingerprint: input.Projection})
	return checked(s, result, err)
}

func (s *Server) catalogInspect(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, catalog.Catalog, error) {
	return checked(s, s.catalog, nil)
}

func (s *Server) profilesRecommend(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, profiles.RecommendationResult, error) {
	result, err := profiles.Recommend(s.root, s.catalog, s.catalogFS)
	return checked(s, result, err)
}

func (s *Server) skillsList(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, skills.ListResponse, error) {
	result, err := skills.List(s.root)
	return checked(s, result, err)
}

func (s *Server) skillsValidate(_ context.Context, _ *mcp.CallToolRequest, input skillInput) (*mcp.CallToolResult, skills.Validation, error) {
	if len([]byte(input.ID)) > 64 {
		return nil, skills.Validation{}, s.safeError(errors.New("skill id exceeds the 64-byte limit"))
	}
	result, err := skills.Validate(s.root, input.ID)
	return checked(s, result, err)
}

func (s *Server) planInspect(_ context.Context, _ *mcp.CallToolRequest, input planInput) (*mcp.CallToolResult, planinspect.Inspection, error) {
	result, err := planinspect.Inspect(s.root, input.Path)
	return checked(s, result, err)
}

func checked[T any](s *Server, result T, err error) (*mcp.CallToolResult, T, error) {
	if err != nil {
		var zero T
		return nil, zero, s.safeError(err)
	}
	encoded, encodeErr := json.Marshal(result)
	if encodeErr != nil || len(encoded) > MaxResponseBytes {
		var zero T
		return nil, zero, errors.New("tool result could not be returned within the protocol limit")
	}
	return nil, result, nil
}

type typedHandler[In, Out any] func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)

func addStrictTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler typedHandler[In, Out], validate func(In) error) {
	// Populate and retain the SDK-derived output schema, then replace only the
	// handler so invalid input maps to JSON-RPC InvalidParams as contracted.
	mcp.AddTool(server, tool, mcp.ToolHandlerFor[In, Out](handler))
	server.AddTool(tool, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input In
		arguments := request.Params.Arguments
		if len(arguments) == 0 {
			arguments = json.RawMessage(`{}`)
		}
		decoder := json.NewDecoder(bytes.NewReader(arguments))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			return nil, invalidParams("arguments do not match the tool schema")
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return nil, invalidParams("arguments must contain exactly one JSON object")
		}
		if err := validate(input); err != nil {
			return nil, invalidParams(err.Error())
		}
		result, output, err := handler(ctx, request, input)
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil
		}
		encoded, err := json.Marshal(output)
		if err != nil || len(encoded) > MaxResponseBytes {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "tool result could not be returned within the protocol limit"}}}, nil
		}
		if result == nil {
			result = &mcp.CallToolResult{}
		}
		result.StructuredContent = output
		if len(result.Content) == 0 {
			result.Content = []mcp.Content{&mcp.TextContent{Text: string(encoded)}}
		}
		return result, nil
	})
}

func addContextTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler typedHandler[In, Out], validate func(In) error, maxBytes int, summarize func(Out) string) {
	// Ask the official SDK to derive and validate the typed schemas before
	// replacing only its result handler to avoid the automatic duplicate JSON
	// text fallback.
	mcp.AddTool(server, tool, mcp.ToolHandlerFor[In, Out](handler))
	server.AddTool(tool, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input In
		arguments := request.Params.Arguments
		if len(arguments) == 0 {
			arguments = json.RawMessage(`{}`)
		}
		decoder := json.NewDecoder(bytes.NewReader(arguments))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			return nil, invalidParams("arguments do not match the tool schema")
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return nil, invalidParams("arguments must contain exactly one JSON object")
		}
		if err := validate(input); err != nil {
			return nil, invalidParams(err.Error())
		}
		_, output, err := handler(ctx, request, input)
		if err != nil {
			return toolError(err.Error()), nil
		}
		encoded, err := json.Marshal(output)
		if err != nil {
			return toolError("tool result could not be encoded"), nil
		}
		var result *mcp.CallToolResult
		if contextResponseMode(input) == "text" {
			result = &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}}
		} else {
			result = &mcp.CallToolResult{
				Content:           []mcp.Content{&mcp.TextContent{Text: boundedText(summarize(output), MaxSummaryBytes)}},
				StructuredContent: output,
			}
		}
		if wire, marshalErr := json.Marshal(result); marshalErr != nil || len(wire) > maxBytes {
			return toolError("tool result could not be returned within the protocol limit"), nil
		}
		return result, nil
	})
}

func contextResponseMode(input any) string {
	switch value := input.(type) {
	case contextStatusInput:
		return value.ResponseMode
	case contextSourcesInput:
		return value.ResponseMode
	case queryInput:
		return value.ResponseMode
	case contextReadInput:
		return value.ResponseMode
	case workspaceTreeInput:
		return value.ResponseMode
	default:
		return "compact"
	}
}

func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: boundedText(message, 1024)}}}
}

func boundedText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func summarizeStatus(result contextview.Status) string {
	freshness := "unavailable"
	if result.Fresh {
		freshness = "fresh"
	} else if result.Exists && result.Supported {
		freshness = "stale"
	}
	summary := fmt.Sprintf("context %s: %d sources, %d chunks", freshness, result.SourceCount, result.ChunkCount)
	if !result.Fresh {
		summary += "; rebuild required"
	}
	return summary
}

func summarizeSources(result contextview.Sources) string {
	return fmt.Sprintf("context sources: %d returned; continuation=%s", len(result.Items), strconv.FormatBool(result.NextAfter != ""))
}

func summarizeSearch(result contextsearch.Response) string {
	return fmt.Sprintf("context search: %d results, %d snippet bytes; truncated=%s", len(result.Results), result.AppliedContentBytes, strconv.FormatBool(result.Truncated))
}

func summarizeRead(result contextview.ReadResult) string {
	return fmt.Sprintf("context read %s: lines %d-%d, %d bytes; continuation=%s", result.Path, result.StartLine, result.EndLine, result.AppliedBytes, strconv.FormatBool(result.NextStartLine > 0))
}

func summarizeTree(result contextview.TreeResult) string {
	return fmt.Sprintf("workspace tree: %d entries; continuation=%s", len(result.Entries), strconv.FormatBool(result.NextAfter != ""))
}

func invalidParams(message string) error {
	return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: message}
}

func noValidation[T any](T) error { return nil }

func validateDoctorInput(input doctorInput) error {
	if input.Scope == "" {
		return nil
	}
	for _, allowed := range []string{doctor.ScopeCore, doctor.ScopeWorkspace, doctor.ScopeIntegrations, doctor.ScopeAll} {
		if input.Scope == allowed {
			return nil
		}
	}
	return errors.New("scope is not supported")
}

func validateQueryInput(input queryInput) error {
	query := strings.TrimSpace(input.Query)
	if query == "" || len([]byte(query)) > contextsearch.MaxQueryBytes {
		return errors.New("query must contain between 1 and 512 bytes")
	}
	if err := validateOptionalContextPath(input.PathPrefix); err != nil {
		return err
	}
	if err := validateContextFilter(input.Kind, "kind"); err != nil {
		return err
	}
	if err := validateContextFilter(input.Language, "language"); err != nil {
		return err
	}
	if input.Limit != nil && (*input.Limit < 1 || *input.Limit > contextsearch.MaxResults) {
		return errors.New("limit must be between 1 and 20 when provided")
	}
	if input.MaxContentBytes != nil && (*input.MaxContentBytes < 1 || *input.MaxContentBytes > contextsearch.MaxContentBytes) {
		return errors.New("max_content_bytes must be between 1 and 16384 when provided")
	}
	return validateResponseMode(input.ResponseMode)
}

func validateContextStatusInput(input contextStatusInput) error {
	return validateResponseMode(input.ResponseMode)
}

func validateContextSourcesInput(input contextSourcesInput) error {
	if err := validateOptionalContextPath(input.PathPrefix); err != nil {
		return err
	}
	if err := validateOptionalContextPath(input.After); err != nil {
		return err
	}
	if err := validateContextFilter(input.Kind, "kind"); err != nil {
		return err
	}
	if err := validateContextFilter(input.Language, "language"); err != nil {
		return err
	}
	if input.Limit != nil && (*input.Limit < 1 || *input.Limit > contextview.MaxPage) {
		return errors.New("limit must be between 1 and 200 when provided")
	}
	if input.After != "" && !schemautil.ValidDigest(input.Projection) {
		return errors.New("projection must be the current sha256 fingerprint when after is provided")
	}
	if input.Projection != "" && !schemautil.ValidDigest(input.Projection) {
		return errors.New("projection must be a sha256 fingerprint")
	}
	return validateResponseMode(input.ResponseMode)
}

func validateContextReadInput(input contextReadInput) error {
	if err := validateRequiredContextPath(input.Path); err != nil {
		return err
	}
	byChunk := input.Chunk != ""
	byLines := input.StartLine != nil || input.EndLine != nil
	if byChunk == byLines {
		return errors.New("exactly one chunk or complete start_line and end_line selector is required")
	}
	if byChunk && !schemautil.ValidDigest(input.Chunk) {
		return errors.New("chunk must be a sha256 identity")
	}
	if byLines && (input.StartLine == nil || input.EndLine == nil || *input.StartLine < 1 || *input.EndLine < *input.StartLine || *input.EndLine-*input.StartLine+1 > contextview.MaxReadLines) {
		return errors.New("line range must contain between 1 and 400 inclusive lines")
	}
	if input.MaxBytes != nil && (*input.MaxBytes < 1 || *input.MaxBytes > contextview.MaxReadBytes) {
		return errors.New("max_bytes must be between 1 and 32768 when provided")
	}
	return validateResponseMode(input.ResponseMode)
}

func validateWorkspaceTreeInput(input workspaceTreeInput) error {
	if err := validateOptionalContextPath(input.PathPrefix); err != nil {
		return err
	}
	if err := validateOptionalContextPath(input.After); err != nil {
		return err
	}
	if input.MaxDepth != nil && (*input.MaxDepth < 1 || *input.MaxDepth > contextview.MaxTreeDepth) {
		return errors.New("max_depth must be between 1 and 8 when provided")
	}
	if input.Limit != nil && (*input.Limit < 1 || *input.Limit > contextview.MaxTreeEntries) {
		return errors.New("limit must be between 1 and 500 when provided")
	}
	if input.After != "" && !schemautil.ValidDigest(input.Projection) {
		return errors.New("projection must be the current sha256 fingerprint when after is provided")
	}
	if input.Projection != "" && !schemautil.ValidDigest(input.Projection) {
		return errors.New("projection must be a sha256 fingerprint")
	}
	return validateResponseMode(input.ResponseMode)
}

func validateResponseMode(mode string) error {
	if mode == "" || mode == "compact" || mode == "text" {
		return nil
	}
	return errors.New("response_mode must be compact or text")
}

func optionalInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func validateContextFilter(value, name string) error {
	if value == "" {
		return nil
	}
	if len([]byte(value)) > 64 || !contextFilterPattern.MatchString(value) {
		return fmt.Errorf("%s must be a normalized identifier of at most 64 bytes", name)
	}
	return nil
}

func validateOptionalContextPath(value string) error {
	if value == "" {
		return nil
	}
	return validateRequiredContextPath(value)
}

func validateRequiredContextPath(value string) error {
	if len([]byte(value)) > contextview.MaxPathBytes || schemautil.ValidateRelative(value) != nil {
		return errors.New("path must be a normalized workspace-relative slash path of at most 512 bytes")
	}
	return nil
}

func validateSkillInput(input skillInput) error {
	if input.ID == "" {
		return nil
	}
	if len([]byte(input.ID)) > 64 || !skillIDPattern.MatchString(input.ID) {
		return errors.New("id must be a normalized skill identifier of at most 64 bytes")
	}
	return nil
}

func validatePlanInput(input planInput) error {
	if input.Path == "" || len([]byte(input.Path)) > planinspect.MaxPathBytes {
		return errors.New("path must contain between 1 and 512 bytes")
	}
	return nil
}

func (s *Server) safeError(err error) error {
	message := strings.ReplaceAll(err.Error(), s.root, ".")
	message = strings.ReplaceAll(message, strings.ReplaceAll(s.root, "/", "\\"), ".")
	return errors.New(boundedText(message, 1024))
}

func readOnlyAnnotations() *mcp.ToolAnnotations {
	no := false
	return &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, DestructiveHint: &no, OpenWorldHint: &no}
}

func schemaEmpty() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":false}`)
}

func schemaDoctor() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"scope":{"type":"string","enum":["core","workspace","integrations","all"]}},"additionalProperties":false}`)
}

func schemaQuery() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","minLength":1,"maxLength":512},"path_prefix":{"type":"string","minLength":1,"maxLength":512},"kind":{"type":"string","minLength":1,"maxLength":64},"language":{"type":"string","minLength":1,"maxLength":64},"limit":{"type":"integer","minimum":1,"maximum":20},"max_content_bytes":{"type":"integer","minimum":1,"maximum":16384},"response_mode":{"type":"string","enum":["compact","text"]}},"required":["query"],"additionalProperties":false}`)
}

func schemaContextStatus() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"response_mode":{"type":"string","enum":["compact","text"]}},"additionalProperties":false}`)
}

func schemaContextSources() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path_prefix":{"type":"string","minLength":1,"maxLength":512},"kind":{"type":"string","minLength":1,"maxLength":64},"language":{"type":"string","minLength":1,"maxLength":64},"limit":{"type":"integer","minimum":1,"maximum":200},"after":{"type":"string","minLength":1,"maxLength":512},"projection":{"type":"string","pattern":"^sha256:[0-9a-f]{64}$"},"response_mode":{"type":"string","enum":["compact","text"]}},"additionalProperties":false}`)
}

func schemaContextRead() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","minLength":1,"maxLength":512},"chunk":{"type":"string","pattern":"^sha256:[0-9a-f]{64}$"},"start_line":{"type":"integer","minimum":1},"end_line":{"type":"integer","minimum":1},"max_bytes":{"type":"integer","minimum":1,"maximum":32768},"response_mode":{"type":"string","enum":["compact","text"]}},"required":["path"],"additionalProperties":false}`)
}

func schemaWorkspaceTree() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path_prefix":{"type":"string","minLength":1,"maxLength":512},"max_depth":{"type":"integer","minimum":1,"maximum":8},"limit":{"type":"integer","minimum":1,"maximum":500},"after":{"type":"string","minLength":1,"maxLength":512},"projection":{"type":"string","pattern":"^sha256:[0-9a-f]{64}$"},"response_mode":{"type":"string","enum":["compact","text"]}},"additionalProperties":false}`)
}

func schemaSkill() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","minLength":1,"maxLength":64,"pattern":"^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$"}},"additionalProperties":false}`)
}

func schemaPlan() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","minLength":1,"maxLength":512}},"required":["path"],"additionalProperties":false}`)
}
