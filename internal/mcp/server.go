// Package mcp implements the Model Context Protocol server for brains.
// It exposes the stickymemory and code-reasoning tools via the MCP protocol.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/2bit-software/zombiekit/internal/config"
	internalgit "github.com/2bit-software/zombiekit/internal/git"
	"github.com/2bit-software/zombiekit/internal/mcp/tools/codereasoning"
	ghprtool "github.com/2bit-software/zombiekit/internal/mcp/tools/ghpr"
	gittool "github.com/2bit-software/zombiekit/internal/mcp/tools/git"
	initiativetool "github.com/2bit-software/zombiekit/internal/mcp/tools/initiative"
	profiletool "github.com/2bit-software/zombiekit/internal/mcp/tools/profile"
	recalltool "github.com/2bit-software/zombiekit/internal/mcp/tools/recall"
	"github.com/2bit-software/zombiekit/internal/mcp/tools/stickymemory"
	workflowtool "github.com/2bit-software/zombiekit/internal/mcp/tools/workflow"
	"github.com/2bit-software/zombiekit/internal/memory"
	"github.com/2bit-software/zombiekit/internal/recall"
)

// Server is the MCP protocol server that exposes tools.
type Server struct {
	mcpServer      *server.MCPServer
	storage        memory.Storage
	recallStorage  recall.Storage
	stickyMemory   *stickymemory.Tool
	codeReasoning  *codereasoning.Tool
	sessionManager *codereasoning.SessionManager
	profileTool    *profiletool.Tool
	workflowTool   *workflowtool.Tool
	initiativeTool *initiativetool.Tool
	recallTool     *recalltool.Tool
	gitTool        *gittool.Tool
	ghPRTool       *ghprtool.Tool
	config         *config.Config
}

// NewServer creates a new MCP server with the given storage backend and configuration.
// If cfg is nil, all tools are enabled by default.
// recallStorage may be nil if recall features are not needed.
// workDir is the working directory for git operations (empty string disables git tools).
func NewServer(storage memory.Storage, recallStorage recall.Storage, cfg *config.Config, workDir ...string) *Server {
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
	wfTool := workflowtool.NewTool()
	initiativeToolInst := initiativetool.NewTool()

	var recallToolInst *recalltool.Tool
	if recallStorage != nil {
		recallToolInst = recalltool.NewTool(recallStorage)
	}

	// Create git tools if a working directory is provided
	var gitToolInst *gittool.Tool
	var ghPRToolInst *ghprtool.Tool
	gitWorkDir := ""
	if len(workDir) > 0 && workDir[0] != "" {
		gitWorkDir = workDir[0]
	}
	if gitWorkDir != "" {
		if runner, err := internalgit.NewRunner(gitWorkDir); err == nil {
			gitToolInst = gittool.NewTool(runner)
		}
		if prTool, err := ghprtool.NewTool(gitWorkDir); err == nil {
			ghPRToolInst = prTool
		}
	}

	s := &Server{
		mcpServer:      mcpServer,
		storage:        storage,
		recallStorage:  recallStorage,
		stickyMemory:   stickyMemoryTool,
		codeReasoning:  codeReasoningTool,
		sessionManager: sessionManager,
		profileTool:    profTool,
		workflowTool:   wfTool,
		initiativeTool: initiativeToolInst,
		recallTool:     recallToolInst,
		gitTool:        gitToolInst,
		ghPRTool:       ghPRToolInst,
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

	// Register workflow tool
	s.registerWorkflowTool()

	// Register initiative tool
	s.registerInitiativeTool()

	// Register recall tools
	s.registerRecallTools()

	// Register git tools
	s.registerGitTool()

	// Register gh-pr tool
	s.registerGHPRTool()
}

// handleStickyMemory handles stickymemory tool calls.
func (s *Server) handleStickyMemory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
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
	args, ok := req.Params.Arguments.(map[string]any)
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
				mcp.Items(map[string]any{"type": "string"}),
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

	// profile-save (renamed from profile-write to distinguish from CLI workflow)
	if s.config.IsToolEnabled("profile-save") {
		saveTool := mcp.NewTool("profile-save",
			mcp.WithDescription("Save a profile to disk at the specified location. Creates the directory if needed."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Profile name (will be used as filename)"),
			),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("Full profile content including frontmatter"),
			),
			mcp.WithString("location",
				mcp.Required(),
				mcp.Description("'local' (.brains/profiles/) or 'global' (~/.brains/profiles/)"),
			),
			mcp.WithBoolean("overwrite",
				mcp.Description("Allow overwriting existing profile (default: false)"),
			),
			mcp.WithString("working_directory",
				mcp.Description("Working directory for local profile resolution (defaults to CWD)"),
			),
		)
		s.mcpServer.AddTool(saveTool, s.handleProfileSave)
	}
}

// handleProfileCompose handles profile-compose tool calls.
func (s *Server) handleProfileCompose(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
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
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	result, err := s.profileTool.HandleList(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// handleProfileSave handles profile-save tool calls.
func (s *Server) handleProfileSave(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	result, err := s.profileTool.HandleSave(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// registerInitiativeTool registers the initiative MCP tool.
// The initiative tool is only registered if enabled in the configuration.
func (s *Server) registerInitiativeTool() {
	if !s.config.IsToolEnabled("initiative") {
		return
	}

	initDef := s.initiativeTool.Definition()
	initMCPTool := mcp.NewTool(initDef.Name,
		mcp.WithDescription(initDef.Description),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("The lifecycle action to perform"),
			mcp.Enum("create", "status", "complete", "list"),
		),
		mcp.WithString("dir",
			mcp.Required(),
			mcp.Description("Working directory containing the .brains folder"),
		),
		mcp.WithString("type",
			mcp.Description("Required for create: Type of initiative (feature, bug, refactor)"),
		),
		mcp.WithString("name",
			mcp.Description("Required for create: Name/slug for the initiative (e.g., 'user-auth')"),
		),
		mcp.WithString("description",
			mcp.Description("Optional for create: Description of the initiative"),
		),
	)
	s.mcpServer.AddTool(initMCPTool, s.handleInitiative)
}

// handleInitiative handles initiative tool calls.
func (s *Server) handleInitiative(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	result, err := s.initiativeTool.Execute(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// registerWorkflowTool registers the workflow-compose MCP tool.
func (s *Server) registerWorkflowTool() {
	if !s.config.IsToolEnabled("workflow-compose") {
		return
	}

	wfDef := s.workflowTool.Definition()
	wfMCPTool := mcp.NewTool(wfDef.Name,
		mcp.WithDescription(wfDef.Description),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Workflow name to load"),
		),
		mcp.WithString("working_directory",
			mcp.Description("Working directory for resolution"),
		),
	)
	s.mcpServer.AddTool(wfMCPTool, s.handleWorkflow)
}

// handleWorkflow handles workflow-compose tool calls.
func (s *Server) handleWorkflow(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	result, err := s.workflowTool.HandleCompose(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// registerRecallTools registers the recall MCP tools.
func (s *Server) registerRecallTools() {
	// Skip if recall storage is not available
	if s.recallTool == nil {
		return
	}

	// Register recall-list-conversations
	if s.config.IsToolEnabled("recall-list-conversations") {
		listTool := mcp.NewTool("recall-list-conversations",
			mcp.WithDescription("List conversation summaries with pagination. Returns conversations ordered by last activity (most recent first)."),
			mcp.WithNumber("page",
				mcp.Description("Page number (1-indexed). Defaults to 1."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Items per page. Defaults to 20, maximum 100."),
			),
			mcp.WithString("project",
				mcp.Description("Filter by project path prefix (e.g., '/Users/me/project'). Empty returns all."),
			),
		)
		s.mcpServer.AddTool(listTool, s.handleRecallListConversations)
	}

	// Register recall-read-conversation
	if s.config.IsToolEnabled("recall-read-conversation") {
		readTool := mcp.NewTool("recall-read-conversation",
			mcp.WithDescription("Read conversation chunks with pagination. Returns chunks in chronological order (oldest first)."),
			mcp.WithString("conversation_id",
				mcp.Required(),
				mcp.Description("Conversation UUID to read"),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number (1-indexed). Defaults to 1."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Items per page. Defaults to 20, maximum 100."),
			),
		)
		s.mcpServer.AddTool(readTool, s.handleRecallReadConversation)
	}
}

// handleRecallListConversations handles recall-list-conversations tool calls.
func (s *Server) handleRecallListConversations(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	result, err := s.recallTool.ListConversations(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// handleRecallReadConversation handles recall-read-conversation tool calls.
func (s *Server) handleRecallReadConversation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	result, err := s.recallTool.ReadConversation(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// registerGitTool registers the git MCP tool.
func (s *Server) registerGitTool() {
	if s.gitTool == nil || !s.config.IsToolEnabled("git") {
		return
	}

	gitDef := s.gitTool.Definition()
	gitMCPTool := mcp.NewTool(gitDef.Name,
		mcp.WithDescription(gitDef.Description),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Git operation to perform"),
			mcp.Enum("status", "log", "diff", "stage", "commit", "push"),
		),
		mcp.WithString("base",
			mcp.Description("Base ref for log/diff (default: main)"),
		),
		mcp.WithString("range",
			mcp.Description("Git revision range for log (e.g., 'main..HEAD')"),
		),
		mcp.WithNumber("count",
			mcp.Description("Number of log entries (default: 10)"),
		),
		mcp.WithString("scope",
			mcp.Description("Diff scope: all, staged, unstaged (default: all)"),
		),
		mcp.WithBoolean("stat_only",
			mcp.Description("Return only file stat, not full diff content"),
		),
		mcp.WithString("paths",
			mcp.Description("Comma-separated file paths to limit diff"),
		),
		mcp.WithString("files",
			mcp.Description("Comma-separated file paths to stage (required for stage action)"),
		),
		mcp.WithString("message",
			mcp.Description("Commit message (required for commit action)"),
		),
		mcp.WithBoolean("set_upstream",
			mcp.Description("Set upstream tracking on push (default: false)"),
		),
		mcp.WithString("remote",
			mcp.Description("Remote name for push (default: origin)"),
		),
		mcp.WithString("directory",
			mcp.Description("Working directory for git operations. When omitted, uses the server's default working directory."),
		),
	)
	s.mcpServer.AddTool(gitMCPTool, s.handleGit)
}

// handleGit handles git tool calls.
func (s *Server) handleGit(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	result, err := s.gitTool.Execute(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

// registerGHPRTool registers the gh-pr MCP tool.
func (s *Server) registerGHPRTool() {
	if s.ghPRTool == nil || !s.config.IsToolEnabled("gh-pr") {
		return
	}

	prDef := s.ghPRTool.Definition()
	prMCPTool := mcp.NewTool(prDef.Name,
		mcp.WithDescription(prDef.Description),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("PR operation to perform"),
			mcp.Enum("view", "create", "comment"),
		),
		mcp.WithString("title",
			mcp.Description("PR title (required for create)"),
		),
		mcp.WithString("body",
			mcp.Description("PR body or comment text (required for create/comment)"),
		),
		mcp.WithString("base",
			mcp.Description("Base branch for PR (default: main)"),
		),
		mcp.WithBoolean("draft",
			mcp.Description("Create PR as draft (default: false)"),
		),
		mcp.WithNumber("pr_number",
			mcp.Description("PR number (required for comment)"),
		),
	)
	s.mcpServer.AddTool(prMCPTool, s.handleGHPR)
}

// handleGHPR handles gh-pr tool calls.
func (s *Server) handleGHPR(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	result, err := s.ghPRTool.Execute(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}
