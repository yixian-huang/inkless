// Package captcha provides a CAPTCHA verification plugin for Impress CMS.
// It implements the provider.CaptchaProvider interface supporting reCAPTCHA v2/v3 and hCaptcha.
package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"blotting-consultancy/internal/plugin"
	"blotting-consultancy/internal/provider"
)

// Manifest describes this plugin's metadata.
var Manifest = plugin.PluginMeta{
	ID:            "cap-captcha",
	Name:          "CAPTCHA Plugin",
	NameZh:        "验证码插件",
	Version:       "1.0.0",
	Description:   "Verifies reCAPTCHA v2/v3 or hCaptcha tokens for comments and contact forms.",
	Author:        "Impress CMS",
	License:       "MIT",
	MinAppVersion: "1.0.0",
	Permissions:   []plugin.Permission{plugin.PermNetworkOutbound},
	Providers: []plugin.ProviderDecl{
		{Type: "captcha", Name: "captcha"},
	},
}

// Provider identifies the CAPTCHA service.
type Provider string

const (
	// ProviderReCAPTCHAV2 uses Google reCAPTCHA v2 ("I'm not a robot" checkbox).
	ProviderReCAPTCHAV2 Provider = "recaptcha_v2"

	// ProviderReCAPTCHAV3 uses Google reCAPTCHA v3 (invisible, score-based).
	ProviderReCAPTCHAV3 Provider = "recaptcha_v3"

	// ProviderHCaptcha uses hCaptcha.
	ProviderHCaptcha Provider = "hcaptcha"
)

// reCAPTCHA v2/v3 verification endpoint.
const reCAPTCHAVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

// hCaptcha verification endpoint.
const hCaptchaVerifyURL = "https://hcaptcha.com/siteverify"

// Config holds the CAPTCHA plugin configuration.
type Config struct {
	// Provider selects the CAPTCHA service (recaptcha_v2, recaptcha_v3, hcaptcha).
	Provider Provider

	// SecretKey is the server-side secret key from the CAPTCHA provider dashboard.
	SecretKey string

	// MinScore is the minimum acceptable score for reCAPTCHA v3 (0.0–1.0).
	// Requests with a score below this threshold are rejected. Default: 0.5.
	MinScore float64
}

// verifyResponse is the common response format for reCAPTCHA and hCaptcha.
type verifyResponse struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score"`       // reCAPTCHA v3 only
	Action      string   `json:"action"`      // reCAPTCHA v3 only
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

// Plugin implements provider.CaptchaProvider.
type Plugin struct {
	config     Config
	httpClient *http.Client
	verifyURL  string
}

// New creates a new CAPTCHA plugin with the provided configuration.
func New(cfg Config) (*Plugin, error) {
	if cfg.Provider == "" {
		return nil, fmt.Errorf("captcha: provider is required (recaptcha_v2, recaptcha_v3, hcaptcha)")
	}
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("captcha: secret key is required")
	}

	verifyURL := reCAPTCHAVerifyURL
	if cfg.Provider == ProviderHCaptcha {
		verifyURL = hCaptchaVerifyURL
	}

	minScore := cfg.MinScore
	if minScore == 0 && cfg.Provider == ProviderReCAPTCHAV3 {
		minScore = 0.5 // sensible default for v3
	}
	cfg.MinScore = minScore

	return &Plugin{
		config:    cfg,
		verifyURL: verifyURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// NewFromSettings creates a Plugin from a string settings map (used by plugin manager).
func NewFromSettings(settings map[string]string) (*Plugin, error) {
	minScore := 0.5
	if v := settings["min_score"]; v != "" {
		if _, err := fmt.Sscanf(v, "%f", &minScore); err != nil {
			return nil, fmt.Errorf("captcha: invalid min_score %q: %w", v, err)
		}
	}
	cfg := Config{
		Provider:  Provider(settings["provider"]),
		SecretKey: settings["secret_key"],
		MinScore:  minScore,
	}
	return New(cfg)
}

// Verify validates a CAPTCHA response token against the configured provider.
// Returns nil if the token is valid, or an error describing the failure.
func (p *Plugin) Verify(ctx context.Context, token, remoteIP string) error {
	if token == "" {
		return fmt.Errorf("captcha: token is required")
	}

	formData := url.Values{
		"secret":   {p.config.SecretKey},
		"response": {token},
	}
	if remoteIP != "" {
		formData.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.verifyURL,
		strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("captcha: failed to create verify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("captcha: verify request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("captcha: failed to read verify response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("captcha: verify returned HTTP %d", resp.StatusCode)
	}

	var result verifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("captcha: failed to parse verify response: %w", err)
	}

	if !result.Success {
		codes := strings.Join(result.ErrorCodes, ", ")
		if codes == "" {
			codes = "unknown error"
		}
		return fmt.Errorf("captcha: verification failed: %s", codes)
	}

	// For reCAPTCHA v3, also check the score
	if p.config.Provider == ProviderReCAPTCHAV3 {
		if result.Score < p.config.MinScore {
			return fmt.Errorf("captcha: reCAPTCHA v3 score %.2f is below minimum %.2f (bot suspected)",
				result.Score, p.config.MinScore)
		}
	}

	return nil
}

// Ensure Plugin implements CaptchaProvider at compile time.
var _ provider.CaptchaProvider = (*Plugin)(nil)
