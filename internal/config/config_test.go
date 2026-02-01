package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_APIKey(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "test-api-key")
	t.Setenv("UNIFI_USERNAME", "")
	t.Setenv("UNIFI_PASSWORD", "")
	t.Setenv("UNIFI_LOG_LEVEL", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "https://192.168.1.1", cfg.Host)
	assert.Equal(t, "test-api-key", cfg.APIKey)
	assert.Equal(t, "default", cfg.Site)
	assert.True(t, cfg.VerifySSL)
	assert.True(t, cfg.UseAPIKey())
	assert.Equal(t, "error", cfg.LogLevel)
}

func TestLoad_UserPass(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "")
	t.Setenv("UNIFI_USERNAME", "admin")
	t.Setenv("UNIFI_PASSWORD", "secret")

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.UseAPIKey())
	assert.Equal(t, "admin", cfg.Username)
	assert.Equal(t, "secret", cfg.Password)
}

func TestLoad_MissingHost(t *testing.T) {
	t.Setenv("UNIFI_HOST", "")
	t.Setenv("UNIFI_API_KEY", "test-api-key")

	_, err := Load()
	assert.ErrorIs(t, err, ErrMissingHost)
}

func TestLoad_MissingCredentials(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "")
	t.Setenv("UNIFI_USERNAME", "")
	t.Setenv("UNIFI_PASSWORD", "")

	_, err := Load()
	assert.ErrorIs(t, err, ErrMissingCredentials)
}

func TestLoad_PartialUserPass(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "")
	t.Setenv("UNIFI_USERNAME", "admin")
	t.Setenv("UNIFI_PASSWORD", "")

	_, err := Load()
	assert.ErrorIs(t, err, ErrMissingCredentials)
}

func TestLoad_CustomSite(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "test-api-key")
	t.Setenv("UNIFI_SITE", "mysite")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "mysite", cfg.Site)
}

func TestLoad_VerifySSLFalse(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "test-api-key")
	t.Setenv("UNIFI_VERIFY_SSL", "false")

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.VerifySSL)
}

func TestLoad_InvalidVerifySSL(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "test-api-key")
	t.Setenv("UNIFI_VERIFY_SSL", "notabool")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UNIFI_VERIFY_SSL")
}

func TestLoad_LogLevelDefault(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "test-api-key")
	t.Setenv("UNIFI_LOG_LEVEL", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "error", cfg.LogLevel)
}

func TestLoad_LogLevelValid(t *testing.T) {
	for _, level := range []string{"disabled", "trace", "debug", "info", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			t.Setenv("UNIFI_HOST", "https://192.168.1.1")
			t.Setenv("UNIFI_API_KEY", "test-api-key")
			t.Setenv("UNIFI_LOG_LEVEL", level)

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, level, cfg.LogLevel)
		})
	}
}

func TestLoad_LogLevelCaseInsensitive(t *testing.T) {
	for _, input := range []string{"INFO", "Info", "ERROR", "Debug"} {
		t.Run(input, func(t *testing.T) {
			t.Setenv("UNIFI_HOST", "https://192.168.1.1")
			t.Setenv("UNIFI_API_KEY", "test-api-key")
			t.Setenv("UNIFI_LOG_LEVEL", input)

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, strings.ToLower(input), cfg.LogLevel)
		})
	}
}

func TestLoad_LogLevelInvalid(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "test-api-key")
	t.Setenv("UNIFI_LOG_LEVEL", "verbose")

	_, err := Load()
	assert.ErrorIs(t, err, ErrInvalidLogLevel)
	assert.Contains(t, err.Error(), "verbose")
}
