package analytics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_MissingProvider(t *testing.T) {
	_, err := New(Config{MeasurementID: "G-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is required")
}

func TestNew_UnknownProvider(t *testing.T) {
	_, err := New(Config{Provider: "unknown", SiteID: "123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestNew_GA4_MissingMeasurementID(t *testing.T) {
	_, err := New(Config{Provider: ProviderGA4})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "measurement_id is required")
}

func TestNew_GA4_Valid(t *testing.T) {
	p, err := New(Config{Provider: ProviderGA4, MeasurementID: "G-ABCDEF1234"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNew_Baidu_MissingSiteID(t *testing.T) {
	_, err := New(Config{Provider: ProviderBaidu})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "site_id is required")
}

func TestNew_Baidu_Valid(t *testing.T) {
	p, err := New(Config{Provider: ProviderBaidu, SiteID: "abc123def456"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNew_Umami_MissingSiteID(t *testing.T) {
	_, err := New(Config{Provider: ProviderUmami, UmamiScriptURL: "https://umami.example.com/script.js"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "website_id")
}

func TestNew_Umami_MissingScriptURL(t *testing.T) {
	_, err := New(Config{Provider: ProviderUmami, SiteID: "website-id-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script_url is required")
}

func TestNew_Umami_Valid(t *testing.T) {
	p, err := New(Config{
		Provider:       ProviderUmami,
		SiteID:         "website-id-abc",
		UmamiScriptURL: "https://umami.example.com/script.js",
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewFromSettings_GA4(t *testing.T) {
	settings := map[string]string{
		"provider":       "ga4",
		"measurement_id": "G-TEST12345",
		"anonymize_ip":   "true",
	}
	p, err := NewFromSettings(settings)
	require.NoError(t, err)
	assert.Equal(t, ProviderGA4, p.config.Provider)
	assert.Equal(t, "G-TEST12345", p.config.MeasurementID)
}

func TestNewFromSettings_AnonymizeIPDefault(t *testing.T) {
	settings := map[string]string{
		"provider":       "ga4",
		"measurement_id": "G-TEST",
	}
	p, err := NewFromSettings(settings)
	require.NoError(t, err)
	assert.True(t, p.config.AnonymizeIP) // default true
}

func TestNewFromSettings_AnonymizeIPFalse(t *testing.T) {
	settings := map[string]string{
		"provider":       "ga4",
		"measurement_id": "G-TEST",
		"anonymize_ip":   "false",
	}
	p, err := NewFromSettings(settings)
	require.NoError(t, err)
	assert.False(t, p.config.AnonymizeIP)
}

func TestHeadScript_GA4_ContainsExpectedParts(t *testing.T) {
	p, _ := New(Config{Provider: ProviderGA4, MeasurementID: "G-TEST123456"})
	script := string(p.HeadScript())

	assert.Contains(t, script, "G-TEST123456")
	assert.Contains(t, script, "googletagmanager.com")
	assert.Contains(t, script, "gtag")
	assert.Contains(t, script, "anonymize_ip")
}

func TestHeadScript_GA4_AnonymizeIPFalse(t *testing.T) {
	p, _ := New(Config{Provider: ProviderGA4, MeasurementID: "G-TEST", AnonymizeIP: false})
	script := string(p.HeadScript())
	assert.Contains(t, script, "'anonymize_ip': false")
}

func TestHeadScript_GA4_AnonymizeIPTrue(t *testing.T) {
	p, _ := New(Config{Provider: ProviderGA4, MeasurementID: "G-TEST", AnonymizeIP: true})
	script := string(p.HeadScript())
	assert.Contains(t, script, "'anonymize_ip': true")
}

func TestHeadScript_Baidu_ContainsExpectedParts(t *testing.T) {
	p, _ := New(Config{Provider: ProviderBaidu, SiteID: "abc123"})
	script := string(p.HeadScript())

	assert.Contains(t, script, "hm.baidu.com")
	assert.Contains(t, script, "abc123")
	assert.Contains(t, script, "_hmt")
}

func TestHeadScript_Umami_ContainsExpectedParts(t *testing.T) {
	p, _ := New(Config{
		Provider:       ProviderUmami,
		SiteID:         "my-website-id",
		UmamiScriptURL: "https://analytics.my-domain.com/umami.js",
	})
	script := string(p.HeadScript())

	assert.Contains(t, script, "analytics.my-domain.com/umami.js")
	assert.Contains(t, script, "my-website-id")
	assert.Contains(t, script, "data-website-id")
}

func TestInjectIntoHTML_GA4(t *testing.T) {
	p, _ := New(Config{Provider: ProviderGA4, MeasurementID: "G-XYZ789"})

	html := `<!DOCTYPE html><html><head><title>Test</title></head><body>Hello</body></html>`
	result := p.InjectIntoHTML(html)

	assert.Contains(t, result, "G-XYZ789")
	assert.Contains(t, result, "googletagmanager.com")
	// Script should appear before </head>
	headIdx := strings.Index(result, "</head>")
	scriptIdx := strings.Index(result, "googletagmanager.com")
	assert.Greater(t, headIdx, scriptIdx, "script should appear before </head>")
}

func TestInjectIntoHTML_NoHeadTag_ReturnsOriginal(t *testing.T) {
	p, _ := New(Config{Provider: ProviderGA4, MeasurementID: "G-TEST"})

	html := `<html><body>No head tag here</body></html>`
	result := p.InjectIntoHTML(html)
	assert.Equal(t, html, result)
}

func TestDescription_GA4(t *testing.T) {
	p, _ := New(Config{Provider: ProviderGA4, MeasurementID: "G-ABC"})
	desc := p.Description()
	assert.Contains(t, desc, "Google Analytics 4")
	assert.Contains(t, desc, "G-ABC")
}

func TestDescription_Baidu(t *testing.T) {
	p, _ := New(Config{Provider: ProviderBaidu, SiteID: "site123"})
	desc := p.Description()
	assert.Contains(t, desc, "Baidu")
	assert.Contains(t, desc, "site123")
}

func TestDescription_Umami(t *testing.T) {
	p, _ := New(Config{
		Provider:       ProviderUmami,
		SiteID:         "wid-abc",
		UmamiScriptURL: "https://umami.example.com/script.js",
	})
	desc := p.Description()
	assert.Contains(t, desc, "Umami")
	assert.Contains(t, desc, "wid-abc")
}

func TestManifest(t *testing.T) {
	err := Manifest.Validate()
	require.NoError(t, err)
}
