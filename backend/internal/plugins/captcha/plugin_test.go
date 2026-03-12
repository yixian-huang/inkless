package captcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_MissingProvider(t *testing.T) {
	_, err := New(Config{SecretKey: "secret"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is required")
}

func TestNew_MissingSecretKey(t *testing.T) {
	_, err := New(Config{Provider: ProviderReCAPTCHAV2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret key is required")
}

func TestNew_ReCAPTCHAV2_Valid(t *testing.T) {
	p, err := New(Config{Provider: ProviderReCAPTCHAV2, SecretKey: "6Le..."})
	require.NoError(t, err)
	assert.Equal(t, reCAPTCHAVerifyURL, p.verifyURL)
	assert.Equal(t, 0.0, p.config.MinScore) // Not applicable to v2
}

func TestNew_ReCAPTCHAV3_DefaultMinScore(t *testing.T) {
	p, err := New(Config{Provider: ProviderReCAPTCHAV3, SecretKey: "6Le..."})
	require.NoError(t, err)
	assert.Equal(t, 0.5, p.config.MinScore)
}

func TestNew_HCaptcha_VerifyURL(t *testing.T) {
	p, err := New(Config{Provider: ProviderHCaptcha, SecretKey: "hcaptcha-secret"})
	require.NoError(t, err)
	assert.Equal(t, hCaptchaVerifyURL, p.verifyURL)
}

func TestNewFromSettings_Valid(t *testing.T) {
	settings := map[string]string{
		"provider":   "recaptcha_v3",
		"secret_key": "my-secret",
		"min_score":  "0.7",
	}
	p, err := NewFromSettings(settings)
	require.NoError(t, err)
	assert.Equal(t, ProviderReCAPTCHAV3, p.config.Provider)
	assert.InDelta(t, 0.7, p.config.MinScore, 0.001)
}

func TestNewFromSettings_InvalidMinScore(t *testing.T) {
	settings := map[string]string{
		"provider":   "recaptcha_v3",
		"secret_key": "my-secret",
		"min_score":  "not-a-float",
	}
	_, err := NewFromSettings(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min_score")
}

func TestVerify_EmptyToken(t *testing.T) {
	p, _ := New(Config{Provider: ProviderReCAPTCHAV2, SecretKey: "secret"})
	err := p.Verify(context.Background(), "", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

func TestVerify_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		resp := verifyResponse{
			Success:     true,
			ChallengeTS: "2024-01-01T00:00:00Z",
			Hostname:    "example.com",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := New(Config{Provider: ProviderReCAPTCHAV2, SecretKey: "secret"})
	p.verifyURL = srv.URL
	p.httpClient = srv.Client()

	err := p.Verify(context.Background(), "valid-token", "1.2.3.4")
	require.NoError(t, err)
}

func TestVerify_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := verifyResponse{
			Success:    false,
			ErrorCodes: []string{"invalid-input-response"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := New(Config{Provider: ProviderReCAPTCHAV2, SecretKey: "secret"})
	p.verifyURL = srv.URL
	p.httpClient = srv.Client()

	err := p.Verify(context.Background(), "bad-token", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid-input-response")
}

func TestVerify_ReCAPTCHAV3_ScorePass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := verifyResponse{
			Success: true,
			Score:   0.9,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := New(Config{Provider: ProviderReCAPTCHAV3, SecretKey: "secret", MinScore: 0.5})
	p.verifyURL = srv.URL
	p.httpClient = srv.Client()

	err := p.Verify(context.Background(), "token", "")
	require.NoError(t, err)
}

func TestVerify_ReCAPTCHAV3_ScoreFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := verifyResponse{
			Success: true,
			Score:   0.1, // Very low score - likely a bot
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := New(Config{Provider: ProviderReCAPTCHAV3, SecretKey: "secret", MinScore: 0.5})
	p.verifyURL = srv.URL
	p.httpClient = srv.Client()

	err := p.Verify(context.Background(), "bot-token", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "score")
	assert.Contains(t, err.Error(), "bot suspected")
}

func TestVerify_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	p, _ := New(Config{Provider: ProviderReCAPTCHAV2, SecretKey: "secret"})
	p.verifyURL = srv.URL
	p.httpClient = srv.Client()

	err := p.Verify(context.Background(), "token", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestVerify_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	p, _ := New(Config{Provider: ProviderReCAPTCHAV2, SecretKey: "secret"})
	p.verifyURL = srv.URL
	p.httpClient = srv.Client()

	err := p.Verify(context.Background(), "token", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse verify response")
}

func TestManifest(t *testing.T) {
	err := Manifest.Validate()
	require.NoError(t, err)
}
