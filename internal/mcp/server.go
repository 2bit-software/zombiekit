// Package mcp implements the Model Context Protocol server for brains.
// It exposes the stickymemory and code-reasoning tools via the MCP protocol.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/mcp/tools/codereasoning"
	profiletool "github.com/zombiekit/brains/internal/mcp/tools/profile"
	"github.com/zombiekit/brains/internal/mcp/tools/stickymemory"
	"github.com/zombiekit/brains/internal/memory"
)

// Server is the MCP protocol server that exposes tools.
type Server struct {
	mcpServer      *server.MCPServer
	storage        memory.Storage
	stickyMemory   *stickymemory.Tool
	codeReasoning  *codereasoning.Tool
	sessionManager *codereasoning.SessionManager
	profileTool    *profiletool.Tool
	config         *config.Config
}

// NewServer creates a new MCP server with the given storage backend and configuration.
// If cfg is nil, all tools are enabled by default.
func NewServer(storage memory.Storage, cfg *config.Config) *Server {
	if cfg == nil {
		cfg = config.NewDefaultConfig()
	}

	mcpServer := server.NewMCPServer(
		"brains",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	sessionManager := codereasoning.NewSessionManager()
	stickyMemoryTool := stickymemory.NewTool(storage)
	codeReasoningTool := codereasoning.NewTool(sessionManager)
	profTool := profiletool.NewTool()

	s := &Server{
		mcpServer:      mcpServer,
		storage:        storage,
		stickyMemory:   stickyMemoryTool,
		codeReasoning:  codeReasoningTool,
		sessionManager: sessionManager,
		profileTool:    profTool,
		config:         cfg,
	}

	// Register tools (filtered by config)
	s.registerTools()

	return s
}

// registerTools registers all MCP tools with the server.
// Tools are only registered if enabled in the configuration.
func (s *Server) registerTools() {
	// Register stickymemory tool
	if s.config.IsToolEnabled("stickymemory") {
		stickyDef := s.stickyMemory.Definition()
		stickyTool := mcp.NewTool(stickyDef.Name,
			mcp.WithDescription(stickyDef.Description),
			mcp.WithString("operation",
				mcp.Required(),
				mcp.Description("The operation to perform"),
				mcp.Enum("get", "set", "list", "delete", "search", "clear"),
			),
			mcp.WithString("name",
				mcp.Description("The name/key of the memory item"),
			),
			mcp.WithString("content",
				mcp.Description("The content to store (required for 'set' operation)"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of items to return (for 'list' and 'search' operations)"),
			),
		)
		s.mcpServer.AddTool(stickyTool, s.handleStickyMemory)
	}

	// Register code-reasoning tool
	if s.config.IsToolEnabled("code-reasoning") {
		reasoningDef := s.codeReasoning.Definition()
		reasoningTool := mcp.NewTool(reasoningDef.Name,
			mcp.WithDescription(reasoningDef.Description),
			mcp.WithString("thought",
				mcp.Required(),
				mcp.Description("Your current reasoning step"),
			),
			mcp.WithNumber("thought_number",
				mcp.Required(),
				mcp.Description("Current number in sequence (1-indexed)"),
			),
			mcp.WithNumber("total_thoughts",
				mcp.Required(),
				mcp.Description("Estimated final count"),
			),
			mcp.WithBoolean("next_thought_needed",
				mcp.Required(),
				mcp.Description("Set to FALSE ONLY when completely done"),
			),
			mcp.WithBoolean("is_revision",
				mcp.Description("When correcting earlier thinking"),
			),
			mcp.WithNumber("revises_thought",
				mcp.Description("Which thought to revise"),
			),
			mcp.WithNumber("branch_from_thought",
				mcp.Description("When exploring alternative approaches"),
			),
			mcp.WithString("branch_id",
				mcp.Description("Branch identifier"),
			),
		)
		s.mcpServer.AddTool(reasoningTool, s.handleCodeReasoning)
	}

	// Register profile tools
	s.registerProfileTools()
}

// handleStickyMemory handles stickymemory tool calls.
func (s *Server) handleStickyMemory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	result, err := s.stickyMemory.Execute(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// handleCodeReasoning handles code-reasoning tool calls.
func (s *Server) handleCodeReasoning(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	// Use a default session ID (in real usage, this would come from connection context)
	sessionID := "default"

	result, err := s.codeReasoning.Execute(ctx, sessionID, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// MCPServer returns the underlying mcp-go server for transport configuration.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// ServeStdio starts the server using stdio transport.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// ServeSSE starts the server using SSE transport on the given address.
func (s *Server) ServeSSE(addr string) *server.SSEServer {
	return server.NewSSEServer(s.mcpServer, server.WithBaseURL(fmt.Sprintf("http://localhost%s", addr)))
}

// Close cleans up server resources.
func (s *Server) Close() error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}

// registerProfileTools registers the profile MCP tools.
// Tools are only registered if enabled in the configuration.
func (s *Server) registerProfileTools() {
	// profile-compose
	if s.config.IsToolEnabled("profile-compose") {
		composeTool := mcp.NewTool("profile-compose",
			mcp.WithDescription("Compose one or more profiles into merged prompt content. Profiles are resolved from local (.brains/profiles/) and global (~/.brains/profiles/) directories with local taking precedence."),
			mcp.WithArray("profiles",
				mcp.Required(),
				mcp.Description("List of profile names to compose"),
				mcp.Items(map[string]interface{}{"type": "string"}),
			),
			mcp.WithString("working_directory",
				mcp.Description("Working directory for profile resolution (defaults to CWD)"),
			),
		)
		s.mcpServer.AddTool(composeTool, s.handleProfileCompose)
	}

	// profile-list
	if s.config.IsToolEnabled("profile-list") {
		listTool := mcp.NewTool("profile-list",
			mcp.WithDescription("List all available profiles from local and global .brains/profiles/ directories."),
			mcp.WithString("working_directory",
				mcp.Description("Working directory for profile resolution (defaults to CWD)"),
			),
		)
		s.mcpServer.AddTool(listTool, s.handleProfileList)
	}
}

// handleProfileCompose handles profile-compose tool calls.
func (s *Server) handleProfileCompose(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	result, err := s.profileTool.HandleCompose(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// handleProfileList handles profile-list tool calls.
func (s *Server) handleProfileList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	result, err := s.profileTool.HandleList(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

