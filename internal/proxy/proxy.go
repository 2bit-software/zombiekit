package proxy

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zombiekit/brains/internal/proxy/handlers/hybrid"
	"github.com/zombiekit/brains/internal/proxy/handlers/local"
	"github.com/zombiekit/brains/internal/proxy/handlers/remote"
)

type Proxy struct {
	mcpServer  *server.MCPServer
	router     *Router
	connection *Connection
	logger     *slog.Logger
}

func NewProxy(cfg *ProxyConfig, logger *slog.Logger) (*Proxy, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	conn, err := NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	mcpSrv := server.NewMCPServer(
		"brains-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	p := &Proxy{
		mcpServer:  mcpSrv,
		router:     NewRouter(),
		connection: conn,
		logger:     logger,
	}

	p.registerToolSchemas()
	p.registerHandlers()

	return p, nil
}

func (p *Proxy) registerHandlers() {
	// Local handlers
	p.router.Register("code-reasoning", local.NewCodeReasoningHandler())
	p.router.Register("workflow-compose", local.NewWorkflowHandler())
	p.router.Register("initiative", local.NewInitiativeHandler())
	p.router.Register("profile-save", local.NewProfileSaveHandler())
	p.router.Register("brains-connection-status", local.NewConnectionStatusHandler(p.connection))

	// Remote handlers
	p.router.Register("recall-list-conversations", remote.NewRecallListHandler(p.connection))
	p.router.Register("recall-read-conversation", remote.NewRecallReadHandler(p.connection))

	// Hybrid handlers
	p.router.Register("profile-compose", hybrid.NewProfileComposeHandler(p.connection))
	p.router.Register("profile-list", hybrid.NewProfileListHandler(p.connection))
}

func (p *Proxy) Router() *Router {
	return p.router
}

func (p *Proxy) Connection() *Connection {
	return p.connection
}

func (p *Proxy) ServeStdio() error {
	return server.ServeStdio(p.mcpServer)
}

func (p *Proxy) handleTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}

	result, err := p.router.Dispatch(ctx, req.Params.Name, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (p *Proxy) registerToolSchemas() {
	p.registerCodeReasoning()
	p.registerWorkflowCompose()
	p.registerInitiative()
	p.registerProfileSave()
	p.registerConnectionStatus()
	p.registerRecallListConversations()
	p.registerRecallReadConversation()
	p.registerProfileCompose()
	p.registerProfileList()
}

func (p *Proxy) registerCodeReasoning() {
	tool := mcp.NewTool("code-reasoning",
		mcp.WithDescription("A tool for structured, step-by-step reasoning about code problems."),
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
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerWorkflowCompose() {
	tool := mcp.NewTool("workflow-compose",
		mcp.WithDescription("Load a workflow by name. Workflows are entry points for starting work."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Workflow name to load"),
		),
		mcp.WithString("working_directory",
			mcp.Description("Working directory for resolution"),
		),
	)
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerInitiative() {
	tool := mcp.NewTool("initiative",
		mcp.WithDescription("Manage development initiatives - create, track status, complete, and list."),
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
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerProfileSave() {
	tool := mcp.NewTool("profile-save",
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
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerConnectionStatus() {
	tool := mcp.NewTool("brains-connection-status",
		mcp.WithDescription("Check connectivity to the central ZK server. Returns connection status, server URL, and any errors."),
	)
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerRecallListConversations() {
	tool := mcp.NewTool("recall-list-conversations",
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
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerRecallReadConversation() {
	tool := mcp.NewTool("recall-read-conversation",
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
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerProfileCompose() {
	tool := mcp.NewTool("profile-compose",
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
	p.mcpServer.AddTool(tool, p.handleTool)
}

func (p *Proxy) registerProfileList() {
	tool := mcp.NewTool("profile-list",
		mcp.WithDescription("List all available profiles from local and global .brains/profiles/ directories."),
		mcp.WithString("working_directory",
			mcp.Description("Working directory for profile resolution (defaults to CWD)"),
		),
	)
	p.mcpServer.AddTool(tool, p.handleTool)
}
