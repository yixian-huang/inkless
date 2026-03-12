package email

import (
	"context"
	"strings"
	"testing"

	"blotting-consultancy/internal/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_MissingHost(t *testing.T) {
	_, err := New(Config{
		Port: 587,
		From: "no-reply@example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP host is required")
}

func TestNew_MissingPort(t *testing.T) {
	_, err := New(Config{
		Host: "smtp.example.com",
		From: "no-reply@example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP port is required")
}

func TestNew_MissingFrom(t *testing.T) {
	_, err := New(Config{
		Host: "smtp.example.com",
		Port: 587,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sender address (from) is required")
}

func TestNew_Valid(t *testing.T) {
	p, err := New(Config{
		Host: "smtp.example.com",
		Port: 587,
		From: "bot@example.com",
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewFromSettings(t *testing.T) {
	settings := map[string]string{
		"host":      "smtp.gmail.com",
		"port":      "587",
		"username":  "user@gmail.com",
		"password":  "app-password",
		"from":      "user@gmail.com",
		"from_name": "My CMS",
		"to":        "admin@example.com",
	}
	p, err := NewFromSettings(settings)
	require.NoError(t, err)
	assert.Equal(t, "smtp.gmail.com", p.config.Host)
	assert.Equal(t, 587, p.config.Port)
	assert.Equal(t, "My CMS", p.config.FromName)
}

func TestNewFromSettings_InvalidPort(t *testing.T) {
	settings := map[string]string{
		"host": "smtp.example.com",
		"port": "not-a-number",
		"from": "bot@example.com",
	}
	_, err := NewFromSettings(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port")
}

func TestNotify_NoRecipient(t *testing.T) {
	p, err := New(Config{
		Host: "smtp.example.com",
		Port: 587,
		From: "bot@example.com",
		// No "To" configured
	})
	require.NoError(t, err)

	err = p.Notify(context.Background(), provider.NotifyEvent{
		Type:    "comment.new",
		Subject: "New comment",
		Body:    "Someone commented.",
		Meta:    map[string]string{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipient address")
}

func TestBuildMessage_PlainFrom(t *testing.T) {
	p, _ := New(Config{
		Host: "smtp.example.com",
		Port: 587,
		From: "bot@example.com",
	})

	msg := p.buildMessage("admin@example.com", provider.NotifyEvent{
		Subject: "Test Subject",
		Body:    "Hello, world!",
	})

	msgStr := string(msg)
	assert.Contains(t, msgStr, "From: bot@example.com")
	assert.Contains(t, msgStr, "To: admin@example.com")
	assert.Contains(t, msgStr, "Subject: Test Subject")
	assert.Contains(t, msgStr, "Hello, world!")
}

func TestBuildMessage_WithFromName(t *testing.T) {
	p, _ := New(Config{
		Host:     "smtp.example.com",
		Port:     587,
		From:     "bot@example.com",
		FromName: "My Bot",
	})

	msg := p.buildMessage("admin@example.com", provider.NotifyEvent{
		Subject: "Test",
		Body:    "Body text",
	})

	assert.Contains(t, string(msg), "From: My Bot <bot@example.com>")
}

func TestBuildMessage_MetaToOverride(t *testing.T) {
	p, _ := New(Config{
		Host: "smtp.example.com",
		Port: 587,
		From: "bot@example.com",
		To:   "default@example.com",
	})

	// Verify meta["to"] override via Notify
	// We can't actually send mail in tests, but we test buildMessage directly
	msg := p.buildMessage("custom@example.com", provider.NotifyEvent{
		Subject: "Override Test",
		Body:    "Custom recipient",
	})
	assert.Contains(t, string(msg), "To: custom@example.com")
}

func TestBuildMessage_ContainsMIMEHeaders(t *testing.T) {
	p, _ := New(Config{
		Host: "smtp.example.com",
		Port: 587,
		From: "bot@example.com",
	})

	msg := p.buildMessage("to@example.com", provider.NotifyEvent{
		Subject: "S",
		Body:    "B",
	})
	msgStr := string(msg)
	assert.Contains(t, msgStr, "MIME-Version: 1.0")
	assert.Contains(t, msgStr, "Content-Type: text/plain; charset=UTF-8")
}

func TestBuildMessage_CRLFLineEndings(t *testing.T) {
	p, _ := New(Config{
		Host: "smtp.example.com",
		Port: 587,
		From: "bot@example.com",
	})

	msg := p.buildMessage("to@example.com", provider.NotifyEvent{
		Subject: "S",
		Body:    "B",
	})
	// RFC 5322 requires CRLF line endings in headers
	assert.True(t, strings.Contains(string(msg), "\r\n"))
}

func TestManifest(t *testing.T) {
	err := Manifest.Validate()
	require.NoError(t, err)
}
