package themecatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAllowHosts_Default(t *testing.T) {
	hosts := ParseAllowHosts("")
	assert.Equal(t, DefaultUMDAllowHosts, hosts)
}

func TestParseAllowHosts_Custom(t *testing.T) {
	hosts := ParseAllowHosts("cdn.example.com, https://github.com/foo ")
	assert.Equal(t, []string{"cdn.example.com", "github.com"}, hosts)
}

func TestValidateUMDURL_OK(t *testing.T) {
	err := ValidateUMDURL(
		"https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.5/theme.umd.js",
		DefaultUMDAllowHosts,
	)
	require.NoError(t, err)
}

func TestValidateUMDURL_RejectsHTTP(t *testing.T) {
	err := ValidateUMDURL("http://github.com/x/y/theme.umd.js", DefaultUMDAllowHosts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "https")
}

func TestValidateUMDURL_RejectsUnknownHost(t *testing.T) {
	err := ValidateUMDURL("https://evil.example/theme.umd.js", DefaultUMDAllowHosts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowlist")
}

func TestValidateUMDURL_RejectsIP(t *testing.T) {
	err := ValidateUMDURL("https://127.0.0.1/theme.umd.js", []string{"127.0.0.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IP")
}

func TestHostAllowed_Subdomain(t *testing.T) {
	assert.True(t, hostAllowed("objects.githubusercontent.com", []string{"githubusercontent.com"}))
	assert.False(t, hostAllowed("evilcom", []string{"com"})) // no suffix trick on "com" without dot rule — "com" has no dot in Contains check... actually "com" has no ".", so only exact match
	assert.False(t, hostAllowed("notcom", []string{"com"}))
}
