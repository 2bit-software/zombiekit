package proxy

import "time"

const defaultCallTimeout = 10 * time.Second

type ProxyConfig struct {
	ServerURL   string
	TLSCAPath   string
	APIKey      string
	CallTimeout time.Duration
	LogLevel    string
}

func (c *ProxyConfig) Validate() error {
	if c.CallTimeout == 0 {
		c.CallTimeout = defaultCallTimeout
	}
	return nil
}
