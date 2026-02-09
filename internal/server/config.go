package server

import "errors"

type Config struct {
	ListenAddr    string
	TLSCertPath   string
	TLSKeyPath    string
	PostgresURL   string
	OllamaURL     string
	RunMigrations bool
}

func (c *Config) Validate() error {
	if c.ListenAddr == "" {
		return errors.New("listen address is required")
	}
	if c.TLSCertPath != "" && c.TLSKeyPath == "" {
		return errors.New("tls-key is required when tls-cert is provided")
	}
	if c.TLSKeyPath != "" && c.TLSCertPath == "" {
		return errors.New("tls-cert is required when tls-key is provided")
	}
	return nil
}

func (c *Config) TLSEnabled() bool {
	return c.TLSCertPath != "" && c.TLSKeyPath != ""
}
