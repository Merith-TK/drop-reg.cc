package main

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// LoadConfig loads the configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &config, nil
}

// GetRedirectURI returns the OAuth redirect URI, auto-generating one if not set
func (c *Config) GetRedirectURI() string {
	if c.Server.RedirectURI != "" {
		return c.Server.RedirectURI
	}

	// Auto-generate redirect URI from domain
	if c.Server.Domain != "" {
		// Use HTTPS for production domains, HTTP for localhost
		if strings.Contains(c.Server.Domain, "localhost") || strings.Contains(c.Server.Domain, "127.0.0.1") {
			return fmt.Sprintf("http://%s/auth/callback", c.Server.Domain)
		} else {
			return fmt.Sprintf("https://%s/auth/callback", c.Server.Domain)
		}
	}

	// Fallback to localhost
	return "http://localhost:8080/auth/callback"
}

// GetPort returns the server port, defaulting to 8080 if not set
func (c *Config) GetPort() int64 {
	if c.Server.Port == 0 {
		return 8080
	}
	return c.Server.Port
}
