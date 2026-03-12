// Package email provides an email notification plugin for Impress CMS.
// It implements the provider.NotifierProvider interface using SMTP.
// Supported: any SMTP server including Gmail, SendGrid, and self-hosted servers.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"blotting-consultancy/internal/plugin"
	"blotting-consultancy/internal/provider"
)

// Manifest describes this plugin's metadata.
var Manifest = plugin.PluginMeta{
	ID:            "eml-notifier",
	Name:          "Email Notification Plugin",
	NameZh:        "邮件通知插件",
	Version:       "1.0.0",
	Description:   "Sends email notifications on comment submission and form events via SMTP.",
	Author:        "Impress CMS",
	License:       "MIT",
	MinAppVersion: "1.0.0",
	Permissions:   []plugin.Permission{plugin.PermNetworkOutbound},
	Providers: []plugin.ProviderDecl{
		{Type: "notifier", Name: "email"},
	},
}

// Config holds the email notification plugin configuration.
type Config struct {
	// Host is the SMTP server hostname (e.g. "smtp.gmail.com").
	Host string

	// Port is the SMTP server port (e.g. 587 for STARTTLS, 465 for TLS, 25 for plain).
	Port int

	// Username is the SMTP authentication username.
	Username string

	// Password is the SMTP authentication password or app password.
	Password string

	// From is the sender email address (e.g. "no-reply@example.com").
	From string

	// FromName is the sender display name (e.g. "Impress CMS").
	FromName string

	// To is the default recipient email address. Used when the event meta has no "to" key.
	To string

	// UseTLS controls whether to use implicit TLS (port 465).
	// When false, STARTTLS is used (port 587).
	UseTLS bool

	// InsecureSkipVerify disables TLS certificate verification. Use only for local testing.
	InsecureSkipVerify bool
}

// Plugin implements provider.NotifierProvider using SMTP email.
type Plugin struct {
	config Config
}

// New creates a new email notification plugin with the provided configuration.
func New(cfg Config) (*Plugin, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("email: SMTP host is required")
	}
	if cfg.Port == 0 {
		return nil, fmt.Errorf("email: SMTP port is required")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("email: sender address (from) is required")
	}
	return &Plugin{config: cfg}, nil
}

// NewFromSettings creates a Plugin from a string settings map (used by plugin manager).
func NewFromSettings(settings map[string]string) (*Plugin, error) {
	port := 587
	if v := settings["port"]; v != "" {
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return nil, fmt.Errorf("email: invalid port %q: %w", v, err)
		}
	}
	cfg := Config{
		Host:               settings["host"],
		Port:               port,
		Username:           settings["username"],
		Password:           settings["password"],
		From:               settings["from"],
		FromName:           settings["from_name"],
		To:                 settings["to"],
		UseTLS:             settings["use_tls"] == "true",
		InsecureSkipVerify: settings["insecure_skip_verify"] == "true",
	}
	return New(cfg)
}

// Notify sends an email notification for the given event.
// The NotifyEvent.Meta may contain:
//   - "to": override the default recipient address
//   - "cc": carbon copy address
func (p *Plugin) Notify(ctx context.Context, event provider.NotifyEvent) error {
	to := p.config.To
	if override, ok := event.Meta["to"]; ok && override != "" {
		to = override
	}
	if to == "" {
		return fmt.Errorf("email: no recipient address configured")
	}

	msg := p.buildMessage(to, event)

	// Use a deadline from context if available; otherwise 30s timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}

	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	if p.config.UseTLS {
		return p.sendTLS(addr, to, msg, deadline)
	}
	return p.sendSTARTTLS(addr, to, msg)
}

// buildMessage constructs a minimal RFC 5322 email message.
func (p *Plugin) buildMessage(to string, event provider.NotifyEvent) []byte {
	fromHeader := p.config.From
	if p.config.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", p.config.FromName, p.config.From)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", to))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", event.Subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(event.Body)
	sb.WriteString("\r\n")

	return []byte(sb.String())
}

// sendSTARTTLS sends the email using STARTTLS (port 587).
// smtp.SendMail handles STARTTLS negotiation internally.
func (p *Plugin) sendSTARTTLS(addr, to string, msg []byte) error {
	var auth smtp.Auth
	if p.config.Username != "" {
		auth = smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)
	}
	return smtp.SendMail(addr, auth, p.config.From, []string{to}, msg)
}

// sendTLS sends the email using implicit TLS (port 465).
func (p *Plugin) sendTLS(addr, to string, msg []byte, deadline time.Time) error {
	tlsCfg := &tls.Config{
		ServerName:         p.config.Host,
		InsecureSkipVerify: p.config.InsecureSkipVerify, //nolint:gosec
	}

	conn, err := tls.DialWithDialer(
		&net.Dialer{Deadline: deadline},
		"tcp", addr, tlsCfg,
	)
	if err != nil {
		return fmt.Errorf("email: TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.config.Host)
	if err != nil {
		return fmt.Errorf("email: SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if p.config.Username != "" {
		auth := smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email: SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(p.config.From); err != nil {
		return fmt.Errorf("email: MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("email: RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: DATA command failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("email: writing message data failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email: closing data writer failed: %w", err)
	}

	return client.Quit()
}

// Ensure Plugin implements NotifierProvider at compile time.
var _ provider.NotifierProvider = (*Plugin)(nil)
