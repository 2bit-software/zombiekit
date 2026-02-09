# Technical Specification: Central ZK Server

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     cmd/zk-server/main.go                    │
│                      (entry point)                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  internal/zkserver/server.go                 │
│    ┌──────────────────────────────────────────────────────┐ │
│    │              Interceptor Chain                        │ │
│    │  rate_limit → logging → auth → recovery               │ │
│    └──────────────────────────────────────────────────────┘ │
│                              │                              │
│    ┌─────────┬─────────┬─────────┬─────────┬─────────┬────┐│
│    │Profile  │Workflow │Config   │Search   │LLM      │Art.││
│    │Service  │Service  │Service  │Service  │Service  │Svc ││
│    └─────────┴─────────┴─────────┴─────────┴─────────┴────┘│
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌──────────┐
        │PostgreSQL│   │LLM       │   │Config    │
        │pgvector  │   │Provider  │   │YAML      │
        └──────────┘   └──────────┘   └──────────┘
```

## Directory Structure

```
zombiekit/
├── cmd/
│   └── zk-server/
│       └── main.go                 # Entry point
├── internal/
│   └── zkserver/
│       ├── config.go               # ServerConfig struct + loading
│       ├── server.go               # HTTP/Connect server setup
│       ├── health.go               # gRPC health check
│       ├── interceptors/
│       │   ├── auth.go             # API key validation
│       │   ├── logging.go          # Request logging
│       │   ├── ratelimit.go        # Token bucket rate limiting
│       │   └── recovery.go         # Panic recovery
│       ├── handlers/
│       │   ├── profile.go          # ProfileServiceHandler
│       │   ├── workflow.go         # WorkflowServiceHandler
│       │   ├── config.go           # ConfigServiceHandler
│       │   ├── search.go           # SearchServiceHandler
│       │   ├── llm.go              # LLMServiceHandler
│       │   └── artifact.go         # ArtifactServiceHandler
│       ├── storage/
│       │   ├── profiles.go         # Profile database operations
│       │   ├── conversations.go    # Conversation/RAG storage
│       │   ├── initiatives.go      # Initiative CRUD
│       │   └── artifacts.go        # Artifact key-value storage
│       └── llm/
│           ├── provider.go         # Provider interface
│           ├── anthropic.go        # Anthropic SDK wrapper
│           └── ollama.go           # Ollama HTTP client
```

## Configuration

### ServerConfig Struct

```go
type ServerConfig struct {
    // Network
    ListenAddress string        `yaml:"listen_address"` // default: ":50051"

    // TLS
    TLS struct {
        CertFile string `yaml:"cert_file"`
        KeyFile  string `yaml:"key_file"`
    } `yaml:"tls"`

    // Auth
    APIKey string `yaml:"-"` // loaded from ZK_SERVER_API_KEY env

    // Shutdown
    ShutdownTimeout time.Duration `yaml:"shutdown_timeout"` // default: 30s

    // LLM
    LLM struct {
        Provider string `yaml:"provider"` // anthropic, ollama
        OllamaURL string `yaml:"ollama_url"`
    } `yaml:"llm"`

    // Rate Limiting
    RateLimit struct {
        RequestsPerMinute int `yaml:"requests_per_minute"` // default: 100
        Burst             int `yaml:"burst"`               // default: 20
    } `yaml:"rate_limit"`
}
```

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `ZK_SERVER_API_KEY` | API key for auth | (required) |
| `ZK_SERVER_TLS_CERT` | TLS cert path override | from config file |
| `ZK_SERVER_TLS_KEY` | TLS key path override | from config file |
| `ZK_LLM_PROVIDER` | LLM provider override | from config file |
| `ZK_SHUTDOWN_TIMEOUT` | Shutdown timeout | 30s |
| `BRAINS_POSTGRES_URL` | Database URL | (required) |
| `ANTHROPIC_API_KEY` | Anthropic credentials | (if using anthropic) |

### Config File Example

```yaml
listen_address: ":50051"

tls:
  cert_file: "/etc/zk-server/tls/server.crt"
  key_file: "/etc/zk-server/tls/server.key"

shutdown_timeout: 30s

llm:
  provider: "anthropic"
  ollama_url: "http://localhost:11434"

rate_limit:
  requests_per_minute: 100
  burst: 20
```

## Server Implementation

### Server Struct

```go
type Server struct {
    config     *ServerConfig
    httpServer *http.Server
    health     *health.Server
    mux        *http.ServeMux

    // Services
    profileHandler  profilev1connect.ProfileServiceHandler
    workflowHandler workflowv1connect.WorkflowServiceHandler
    configHandler   configv1connect.ConfigServiceHandler
    searchHandler   searchv1connect.SearchServiceHandler
    llmHandler      llmv1connect.LLMServiceHandler
    artifactHandler artifactv1connect.ArtifactServiceHandler
}
```

### Server Initialization

```go
func NewServer(cfg *ServerConfig, db *database.PostgresPool, llmProvider llm.Provider) (*Server, error) {
    s := &Server{config: cfg}

    // Create interceptor chain
    interceptors := connect.WithInterceptors(
        interceptors.NewRateLimitInterceptor(cfg.RateLimit),
        interceptors.NewLoggingInterceptor(),
        interceptors.NewAuthInterceptor(cfg.APIKey),
        interceptors.NewRecoveryInterceptor(),
    )

    // Create handlers
    s.profileHandler = handlers.NewProfileHandler(storage.NewProfileStore(db))
    s.workflowHandler = handlers.NewWorkflowHandler(storage.NewInitiativeStore(db))
    s.configHandler = handlers.NewConfigHandler(cfg)
    s.searchHandler = handlers.NewSearchHandler(storage.NewConversationStore(db))
    s.llmHandler = handlers.NewLLMHandler(llmProvider)
    s.artifactHandler = handlers.NewArtifactHandler(storage.NewArtifactStore(db))

    // Create mux and mount handlers
    s.mux = http.NewServeMux()

    // Health check (no auth)
    s.health = health.NewServer()
    s.mux.Handle(grpc_health_v1.NewHealthHandler(s.health))

    // Services (with interceptors)
    s.mux.Handle(profilev1connect.NewProfileServiceHandler(s.profileHandler, interceptors))
    s.mux.Handle(workflowv1connect.NewWorkflowServiceHandler(s.workflowHandler, interceptors))
    s.mux.Handle(configv1connect.NewConfigServiceHandler(s.configHandler, interceptors))
    s.mux.Handle(searchv1connect.NewSearchServiceHandler(s.searchHandler, interceptors))
    s.mux.Handle(llmv1connect.NewLLMServiceHandler(s.llmHandler, interceptors))
    s.mux.Handle(artifactv1connect.NewArtifactServiceHandler(s.artifactHandler, interceptors))

    return s, nil
}
```

### TLS Configuration

```go
func (s *Server) loadTLSConfig() (*tls.Config, error) {
    cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
    if err != nil {
        return nil, fmt.Errorf("load TLS keypair: %w", err)
    }

    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }, nil
}
```

### Graceful Shutdown

```go
func (s *Server) Shutdown(ctx context.Context) error {
    // Mark health as not serving
    s.health.Shutdown()

    // Stop accepting new connections
    return s.httpServer.Shutdown(ctx)
}
```

## Interceptor Implementations

### Auth Interceptor

```go
type AuthInterceptor struct {
    apiKey string
}

func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        // Skip health checks
        if strings.HasPrefix(req.Spec().Procedure, "/grpc.health.v1.Health/") {
            return next(ctx, req)
        }

        key := req.Header().Get("x-api-key")
        if key == "" {
            return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing api key"))
        }
        if subtle.ConstantTimeCompare([]byte(key), []byte(i.apiKey)) != 1 {
            return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid api key"))
        }

        return next(ctx, req)
    }
}

func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
    return next // Client-side, not applicable for server
}

func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
    return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
        // Same auth logic for streaming
        key := conn.RequestHeader().Get("x-api-key")
        if key == "" {
            return connect.NewError(connect.CodeUnauthenticated, errors.New("missing api key"))
        }
        if subtle.ConstantTimeCompare([]byte(key), []byte(i.apiKey)) != 1 {
            return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid api key"))
        }
        return next(ctx, conn)
    }
}
```

### Rate Limit Interceptor

```go
type RateLimitInterceptor struct {
    limiter *rate.Limiter
}

func NewRateLimitInterceptor(cfg RateLimitConfig) *RateLimitInterceptor {
    // Convert requests per minute to per second
    rps := float64(cfg.RequestsPerMinute) / 60.0
    return &RateLimitInterceptor{
        limiter: rate.NewLimiter(rate.Limit(rps), cfg.Burst),
    }
}

func (i *RateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        if !i.limiter.Allow() {
            err := connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded"))
            err.Meta().Set("retry-after", "60")
            return nil, err
        }
        return next(ctx, req)
    }
}
```

## LLM Provider Interface

```go
type Provider interface {
    Complete(ctx context.Context, req *llmv1.CompleteRequest) (*llmv1.CompleteResponse, error)
    CompleteStream(ctx context.Context, req *llmv1.CompleteStreamRequest) (<-chan *llmv1.CompleteStreamResponse, <-chan error)
}
```

### Anthropic Provider

```go
type AnthropicProvider struct {
    client *anthropic.Client
}

func (p *AnthropicProvider) CompleteStream(ctx context.Context, req *llmv1.CompleteStreamRequest) (<-chan *llmv1.CompleteStreamResponse, <-chan error) {
    responses := make(chan *llmv1.CompleteStreamResponse)
    errs := make(chan error, 1)

    go func() {
        defer close(responses)
        defer close(errs)

        stream, err := p.client.Messages.Stream(ctx, anthropic.MessageRequest{
            Model:     req.Model,
            Messages:  convertMessages(req.Messages),
            MaxTokens: int(req.MaxTokens),
        })
        if err != nil {
            errs <- err
            return
        }
        defer stream.Close()

        for {
            select {
            case <-ctx.Done():
                return // Client disconnected, clean exit
            default:
            }

            event, err := stream.Recv()
            if err == io.EOF {
                return
            }
            if err != nil {
                errs <- err
                return
            }

            responses <- &llmv1.CompleteStreamResponse{
                Delta: event.ContentBlockDelta.Delta.Text,
            }
        }
    }()

    return responses, errs
}
```

## Database Schema

### Profiles Table

```sql
CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    location VARCHAR(50) NOT NULL, -- 'global' or 'local'
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_profiles_name ON profiles(name);
```

### Initiatives Table

```sql
CREATE TABLE initiatives (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'feature', 'bug', 'refactor'
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress',
    project_path TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE initiative_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(initiative_id, name)
);
```

### Artifacts Table

```sql
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(initiative_id, key)
);
```

## Handler Implementations (Sketches)

### ProfileHandler

```go
type ProfileHandler struct {
    store storage.ProfileStore
}

func (h *ProfileHandler) ComposeProfile(ctx context.Context, req *connect.Request[profilev1.ComposeProfileRequest]) (*connect.Response[profilev1.ComposeProfileResponse], error) {
    profiles := make([]*profilev1.Profile, 0, len(req.Msg.ProfileNames))

    for _, name := range req.Msg.ProfileNames {
        p, err := h.store.Get(ctx, name)
        if err != nil {
            return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("profile %q not found", name))
        }
        profiles = append(profiles, p)
    }

    composed := composeProfiles(profiles)
    return connect.NewResponse(&profilev1.ComposeProfileResponse{
        ComposedContent: composed,
    }), nil
}

func (h *ProfileHandler) SaveProfile(ctx context.Context, req *connect.Request[profilev1.SaveProfileRequest]) (*connect.Response[profilev1.SaveProfileResponse], error) {
    // Validate YAML syntax
    if err := validateProfileSyntax(req.Msg.Content); err != nil {
        return nil, connect.NewError(connect.CodeInvalidArgument, err)
    }

    if err := h.store.Save(ctx, req.Msg.Name, req.Msg.Content, req.Msg.Location); err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }

    return connect.NewResponse(&profilev1.SaveProfileResponse{
        Success: true,
    }), nil
}

func (h *ProfileHandler) SubscribeProfileUpdates(ctx context.Context, req *connect.Request[profilev1.SubscribeProfileUpdatesRequest], stream *connect.ServerStream[profilev1.SubscribeProfileUpdatesResponse]) error {
    return connect.NewError(connect.CodeUnimplemented, errors.New("profile streaming deferred to DEV-112"))
}
```

### LLMHandler

```go
type LLMHandler struct {
    provider llm.Provider
}

func (h *LLMHandler) CompleteStream(ctx context.Context, req *connect.Request[llmv1.CompleteStreamRequest], stream *connect.ServerStream[llmv1.CompleteStreamResponse]) error {
    responses, errs := h.provider.CompleteStream(ctx, req.Msg)

    for {
        select {
        case <-ctx.Done():
            // Client disconnected - provider goroutine will clean up
            return nil
        case err := <-errs:
            if err != nil {
                return connect.NewError(connect.CodeInternal, err)
            }
        case resp, ok := <-responses:
            if !ok {
                return nil // Stream complete
            }
            if err := stream.Send(resp); err != nil {
                return err
            }
        }
    }
}
```

## Testing Strategy

### Unit Tests
- Config loading and validation
- Interceptor logic (auth, rate limit)
- Individual handler methods with mocked storage

### Integration Tests
- Server startup with valid/invalid config
- Auth rejection scenarios
- Full request round-trips through handlers
- LLM streaming with mock provider
- Graceful shutdown behavior

### Test Fixtures
- Self-signed TLS certificates for test
- Test database with known data
- Mock LLM provider returning canned responses
