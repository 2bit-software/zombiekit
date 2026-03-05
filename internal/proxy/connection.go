package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/zombiekit/brains/gen/zombiekit/brains/profile/v1/profilev1connect"
	"github.com/zombiekit/brains/gen/zombiekit/brains/search/v1/searchv1connect"
)

const healthCacheTTL = 5 * time.Second

type Connection struct {
	baseURL   string
	profiles  profilev1connect.ProfileServiceClient
	search    searchv1connect.SearchServiceClient
	mu        sync.Mutex
	lastCheck time.Time
	lastOK    bool
	lastErr   string
}

func NewConnection(cfg *ProxyConfig) (*Connection, error) {
	if cfg.ServerURL == "" {
		return nil, nil
	}

	httpClient, err := buildHTTPClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	return &Connection{
		baseURL:  cfg.ServerURL,
		profiles: profilev1connect.NewProfileServiceClient(httpClient, cfg.ServerURL),
		search:   searchv1connect.NewSearchServiceClient(httpClient, cfg.ServerURL),
	}, nil
}

func (c *Connection) IsConfigured() bool {
	return c != nil && c.baseURL != ""
}

func (c *Connection) Profiles() profilev1connect.ProfileServiceClient {
	return c.profiles
}

func (c *Connection) Search() searchv1connect.SearchServiceClient {
	return c.search
}

func (c *Connection) HealthCheck(ctx context.Context) (bool, string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.lastCheck) < healthCacheTTL {
		return c.lastOK, c.lastErr
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		c.lastCheck = time.Now()
		c.lastOK = false
		c.lastErr = err.Error()
		return false, c.lastErr
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.lastCheck = time.Now()
		c.lastOK = false
		c.lastErr = err.Error()
		return false, c.lastErr
	}
	resp.Body.Close()

	c.lastCheck = time.Now()
	c.lastOK = resp.StatusCode == http.StatusOK
	if !c.lastOK {
		c.lastErr = fmt.Sprintf("health check returned %d", resp.StatusCode)
	} else {
		c.lastErr = ""
	}
	return c.lastOK, c.lastErr
}

func (c *Connection) ServerURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

func (c *Connection) LastCheck() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastCheck
}

func buildHTTPClient(cfg *ProxyConfig) (*http.Client, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: cfg.CallTimeout,
		}).DialContext,
	}

	if cfg.TLSCAPath != "" {
		caCert, err := os.ReadFile(cfg.TLSCAPath)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("invalid CA cert at %s", cfg.TLSCAPath)
		}
		transport.TLSClientConfig = &tls.Config{
			RootCAs: pool,
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.CallTimeout,
	}, nil
}
