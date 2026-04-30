package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/2bit-software/zombiekit/internal/config"
	"github.com/2bit-software/zombiekit/internal/database"
	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/recall"
	recallpg "github.com/2bit-software/zombiekit/internal/recall/postgres"
	"github.com/2bit-software/zombiekit/internal/server/storage"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/artifact/v1/artifactv1connect"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/config/v1/configv1connect"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/llm/v1/llmv1connect"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/profile/v1/profilev1connect"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/search/v1/searchv1connect"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/workflow/v1/workflowv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	cfg         *Config
	httpServer  *http.Server
	db          *database.PostgresPool
	recallStore recall.Storage
	embedder    *OllamaEmbedderAdapter
}

func New(cfg *Config) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &Server{cfg: cfg}, nil
}

func (s *Server) initDB(ctx context.Context) error {
	if s.cfg.PostgresURL == "" {
		logging.Logger().Warn("no postgres URL configured, database features disabled")
		return nil
	}

	storageCfg := config.StorageConfig{
		Backend:     config.BackendPostgres,
		PostgresURL: s.cfg.PostgresURL,
		MaxConns:    10,
		MinConns:    2,
	}

	db, err := database.NewPostgresPool(ctx, storageCfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	s.db = db

	if s.cfg.RunMigrations {
		logging.Logger().Info("running database migrations")
		if err := database.RunPostgresMigrations(ctx, db.Pool()); err != nil {
			db.Close()
			return fmt.Errorf("run migrations: %w", err)
		}
	}

	recallStore, err := recallpg.New(ctx, storageCfg)
	if err != nil {
		logging.Logger().Warn("recall storage not available", slog.String("error", err.Error()))
	} else {
		s.recallStore = recallStore
	}

	if s.cfg.OllamaURL != "" {
		embedder, err := NewOllamaEmbedderAdapter(s.cfg.OllamaURL, "nomic-embed-text")
		if err != nil {
			logging.Logger().Warn("embedder not available", slog.String("error", err.Error()))
		} else {
			s.embedder = embedder
		}
	}

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.initDB(ctx); err != nil {
		return err
	}
	defer func() {
		if s.db != nil {
			s.db.Close()
		}
	}()

	if err := s.buildHTTPServer(); err != nil {
		return err
	}

	return s.serveAndWait(ctx)
}

// buildHTTPServer creates the HTTP server with all routes, interceptors, and
// optional TLS configuration.
func (s *Server) buildHTTPServer() error {
	mux := http.NewServeMux()

	interceptors := connect.WithInterceptors(NewLoggingInterceptor())
	s.registerServices(mux, interceptors)
	mux.HandleFunc("/healthz", s.healthHandler)

	s.httpServer = &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if s.cfg.TLSEnabled() {
		cert, err := tls.LoadX509KeyPair(s.cfg.TLSCertPath, s.cfg.TLSKeyPath)
		if err != nil {
			return fmt.Errorf("load TLS cert: %w", err)
		}
		s.httpServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	return nil
}

// serveAndWait starts the listener, launches the serve goroutine, and blocks
// until the context is cancelled or the server returns an error.
func (s *Server) serveAndWait(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		var serveErr error
		if s.cfg.TLSEnabled() {
			logging.Logger().Info("starting server with TLS",
				slog.String("addr", s.cfg.ListenAddr))
			serveErr = s.httpServer.ServeTLS(ln, "", "")
		} else {
			logging.Logger().Info("starting server without TLS",
				slog.String("addr", s.cfg.ListenAddr))
			serveErr = s.httpServer.Serve(ln)
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logging.Logger().Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) registerServices(mux *http.ServeMux, opts ...connect.HandlerOption) {
	var profileStorage storage.ProfileStorage
	if s.db != nil {
		profileStorage = storage.NewPostgresProfileStorage(s.db.Pool())
	}
	path, handler := profilev1connect.NewProfileServiceHandler(NewProfileService(profileStorage), opts...)
	mux.Handle(path, handler)

	var initStorage storage.InitiativeStorage
	if s.db != nil {
		initStorage = storage.NewPostgresInitiativeStorage(s.db.Pool())
	}
	path, handler = workflowv1connect.NewWorkflowServiceHandler(NewWorkflowService(initStorage), opts...)
	mux.Handle(path, handler)

	var embedder Embedder
	if s.embedder != nil {
		embedder = s.embedder
	}
	path, handler = searchv1connect.NewSearchServiceHandler(NewSearchService(s.recallStore, embedder), opts...)
	mux.Handle(path, handler)

	var cfgStorage storage.ConfigStorage
	if s.db != nil {
		cfgStorage = storage.NewPostgresConfigStorage(s.db.Pool())
	}
	path, handler = configv1connect.NewConfigServiceHandler(NewConfigService(cfgStorage), opts...)
	mux.Handle(path, handler)

	path, handler = llmv1connect.NewLLMServiceHandler(&LLMService{}, opts...)
	mux.Handle(path, handler)

	var artStorage storage.ArtifactStorage
	if s.db != nil {
		artStorage = storage.NewPostgresArtifactStorage(s.db.Pool())
	}
	path, handler = artifactv1connect.NewArtifactServiceHandler(NewArtifactService(artStorage), opts...)
	mux.Handle(path, handler)
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}
