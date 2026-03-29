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

// WithRateLimitThreshold sets the low-water mark for pre-emptive rate limit
// delay. When X-RateLimit-Remaining drops below this value, the client sleeps
// before the next request. Default: 10.
func WithRateLimitThreshold(n int) Option {
	return func(c *httpClient) { c.rateLimitThreshold = n }
}
