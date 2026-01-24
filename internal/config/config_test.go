package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_APIKey(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://192.168.1.1")
	t.Setenv("UNIFI_API_KEY", "test-api-key")
	t.Setenv("UNIFI_USERNAME", "")
	t.Setenv("UNIFI_PASSWORD", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "https://192.168.1.1", cfg.Host)
	assert.Equal(t, "test-api-key", cfg.APIKey)
	assert.Equal(t, "default", cfg.Site)
	assert.True(t, cfg.VerifySSL)
	assert.True(t, cfg.UseAPIKey())
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
