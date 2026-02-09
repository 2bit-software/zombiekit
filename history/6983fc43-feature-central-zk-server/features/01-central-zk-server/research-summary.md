# Research Summary: Central ZK Server

## Codebase Patterns

### Entry Point Pattern
- Minimal `main.go` in `/cmd/` directory
- Initialize embedded filesystems in `init()`
- Create app instance, call `Run()`, handle fatal error

### Configuration System
- **Storage Config**: `LoadStorageConfigFromEnv()` - PostgreSQL/SQLite with env vars
- **Tool Config**: TOML-based tool enable/disable flags
- **Startup Config**: YAML-based service configuration

### Database Layer
- `database.PostgresPool` wraps `pgxpool` for connection management
- Fail-fast connectivity check via `pool.Ping(ctx)`
- Connection pool settings: MaxConnLifetime 1h, MaxConnIdleTime 30m

### gRPC/Connect Setup
- Uses **ConnectRPC** (not raw gRPC) - `connectrpc.com/connect v1.19.1`
- Proto definitions in `/proto/zombiekit/brains/*/v1/`
- Generated code in `/gen/zombiekit/brains/*/v1/`
- Services defined: Profile, Workflow, Config, Search, LLM, Artifact

### Graceful Shutdown
- `shutdown.Manager` handles SIGINT/SIGTERM
- First signal: graceful shutdown with configurable timeout
- Second signal or timeout: force exit
- Uses `errgroup.WithContext` for concurrent service coordination

### Logging
- Singleton logger: `logging.InitLogger(level, jsonOutput, writer)`
- Initialize at entrypoint before goroutines
- For MCP tools: dependency-inject logger to stderr (not stdout)

## gRPC Server Best Practices

### TLS Configuration
- Use `credentials.NewTLS()` with `tls.Config`
- Set `MinVersion: tls.VersionTLS12` minimum
- For service-to-service: mTLS with `ClientAuth: tls.RequireAndVerifyClientCert`
- Load certs from file, consider rotation strategy

### Graceful Shutdown
```go
// Two-phase shutdown pattern
go func() {
    server.GracefulStop()  // Phase 1: Wait for RPCs
    close(stopped)
}()
select {
case <-stopped:
case <-time.After(timeout):
    server.Stop()  // Phase 2: Force stop
}
```

### Health Checks
- Use `google.golang.org/grpc/health` package
- Register `healthgrpc.RegisterHealthServer(server, healthServer)`
- Set `NOT_SERVING` before graceful shutdown
- Kubernetes supports native gRPC health probes (GA in 1.27+)

### Interceptors (Order Matters)
1. Rate limiting (reject early)
2. Tracing (start span)
3. Logging (request details)
4. Auth (validate token)
5. Recovery (convert panics - LAST)

### Keepalives
- Server: Set `MaxConnectionAge` for connection refresh
- Coordinate client `Time` >= server `MinTime`
- AWS ALB has 350s idle timeout - configure accordingly

### Streaming
- Batch messages for 1.75x performance
- Check context cancellation for graceful disconnect
- Use `io.EOF` to detect clean client close

## Key Files for Reference

| Component | Path |
|-----------|------|
| Entry Point | `/cmd/brains/main.go` |
| Config | `/internal/config/` |
| Database | `/internal/database/postgres.go` |
| Shutdown | `/internal/shutdown/manager.go` |
| Web Server | `/internal/web/server.go` |
| Proto | `/proto/zombiekit/brains/*/v1/` |
| Generated | `/gen/zombiekit/brains/*/v1/` |

## Implications for ZK Server

1. **Use ConnectRPC** - already set up, generated handlers exist
2. **Reuse storage config** - `LoadStorageConfigFromEnv()` handles PostgreSQL
3. **Copy shutdown pattern** - existing `shutdown.Manager` works for gRPC
4. **New config struct needed** - listen address, TLS paths, LLM provider settings
5. **Service handlers** - implement interfaces from generated `*connect.go` files
