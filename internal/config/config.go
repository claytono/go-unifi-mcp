package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	ErrMissingHost        = errors.New("UNIFI_HOST environment variable is required")
	ErrMissingCredentials = errors.New("either UNIFI_API_KEY or both UNIFI_USERNAME and UNIFI_PASSWORD must be set")
	ErrInvalidLogLevel    = errors.New("UNIFI_LOG_LEVEL must be one of: disabled, trace, debug, info, warn, error")
)

var validLogLevels = map[string]bool{
	"disabled": true,
	"trace":    true,
	"debug":    true,
	"info":     true,
	"warn":     true,
	"error":    true,
}

// Config holds the MCP server configuration.
type Config struct {
	Host      string // UNIFI_HOST - UniFi controller URL
	APIKey    string // UNIFI_API_KEY - API key auth (preferred)
	Username  string // UNIFI_USERNAME - username/password auth
	Password  string // UNIFI_PASSWORD - username/password auth
	Site      string // UNIFI_SITE - site name (default: "default")
	VerifySSL bool   // UNIFI_VERIFY_SSL - verify SSL certs (default: true)
	LogLevel  string // UNIFI_LOG_LEVEL - go-unifi log level (default: "error")
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Host:      os.Getenv("UNIFI_HOST"),
		APIKey:    os.Getenv("UNIFI_API_KEY"),
		Username:  os.Getenv("UNIFI_USERNAME"),
		Password:  os.Getenv("UNIFI_PASSWORD"),
		Site:      os.Getenv("UNIFI_SITE"),
		VerifySSL: true,
	}

	// Parse UNIFI_VERIFY_SSL
	if v := os.Getenv("UNIFI_VERIFY_SSL"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.New("UNIFI_VERIFY_SSL must be a boolean (true/false)")
		}
		cfg.VerifySSL = parsed
	}

	// Parse UNIFI_LOG_LEVEL
	if v := os.Getenv("UNIFI_LOG_LEVEL"); v != "" {
		v = strings.ToLower(v)
		if !validLogLevels[v] {
			return nil, fmt.Errorf("%w: got %q", ErrInvalidLogLevel, v)
		}
		cfg.LogLevel = v
	} else {
		cfg.LogLevel = "error"
	}

	// Set default site
	if cfg.Site == "" {
		cfg.Site = "default"
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks required configuration.
func (c *Config) Validate() error {
	if c.Host == "" {
		return ErrMissingHost
	}

	if !c.UseAPIKey() && !c.UseUserPass() {
		return ErrMissingCredentials
	}

	return nil
}

// UseAPIKey returns true if API key auth should be used.
func (c *Config) UseAPIKey() bool {
	return c.APIKey != ""
}

// UseUserPass returns true if username/password auth should be used.
func (c *Config) UseUserPass() bool {
	return c.Username != "" && c.Password != ""
}
