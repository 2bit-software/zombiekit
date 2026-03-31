package github

import (
	"net/http"
	"time"
)

// Option configures an httpClient.
type Option func(*httpClient)

// WithEndpoint overrides the default GitHub API base URL.
func WithEndpoint(url string) Option {
	return func(c *httpClient) { c.endpoint = url }
}

// WithHTTPClient overrides the default HTTP client.
// When set, the rate-limiter transport is NOT applied.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *httpClient) {
		c.customHTTPClient = hc
	}
}

// WithRetryTiming overrides retry backoff timing (for tests).
func WithRetryTiming(base, maxJitter time.Duration) Option {
	return func(c *httpClient) {
		c.retryBase = base
		c.retryMaxJitter = maxJitter
	}
}
