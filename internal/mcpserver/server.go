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
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/doctor"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/planinspect"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/profiles"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const (
	MaxMessageBytes  = 1 << 20
	MaxResponseBytes = 1 << 20
)

type Dependencies struct {
	Version       string
	ContextSearch contextsearch.Searcher
}

type Server struct {
	root           string
	version        string
	contextSearch  contextsearch.Searcher
	catalog        catalog.Catalog
	catalogFS      fs.FS
	protocolServer *mcp.Server
}

type emptyInput struct{}
type doctorInput struct {
	Scope string `json:"scope,omitempty"`
}
type queryInput struct {
	Query string `json:"query"`
}
type skillInput struct {
	ID string `json:"id,omitempty"`
}
type planInput struct {
	Path string `json:"path"`
}

var skillIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

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
	s := &Server{root: resolved, version: dependencies.Version, contextSearch: dependencies.ContextSearch, catalog: inventory, catalogFS: catalogFS}
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
	addStrictTool(s.protocolServer, &mcp.Tool{Name: "context_search", Description: "Search a fresh optional contextual index through the application boundary.", Annotations: annotations, InputSchema: schemaQuery()}, s.contextSearchTool, validateQueryInput)
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

func (s *Server) contextSearchTool(ctx context.Context, _ *mcp.CallToolRequest, input queryInput) (*mcp.CallToolResult, contextsearch.Response, error) {
	result, err := contextsearch.Search(ctx, s.contextSearch, s.root, input.Query)
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
	if len(message) > 1024 {
		message = message[:1024]
	}
	return errors.New(message)
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
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","minLength":1,"maxLength":512}},"required":["query"],"additionalProperties":false}`)
}

func schemaSkill() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","minLength":1,"maxLength":64,"pattern":"^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$"}},"additionalProperties":false}`)
}

func schemaPlan() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","minLength":1,"maxLength":512}},"required":["path"],"additionalProperties":false}`)
}
