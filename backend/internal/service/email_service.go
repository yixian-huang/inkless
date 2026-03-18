package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// ---------- Config types ----------

// EmailConfig is the top-level email configuration stored in SiteConfig key "email".
type EmailConfig struct {
	SMTP      SMTPConfig                `json:"smtp"`
	Receivers ReceiverConfig            `json:"receivers"`
	AutoReply AutoReplyConfig           `json:"autoReply"`
	Templates map[string]TemplatesConfig `json:"templates"`
}

// SMTPConfig holds SMTP server settings.
type SMTPConfig struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	FromAddress        string `json:"fromAddress"`
	FromName           string `json:"fromName"`
	UseTLS             bool   `json:"useTLS"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

// IsConfigured returns true when the minimum SMTP fields are present.
func (s *SMTPConfig) IsConfigured() bool {
	return s.Host != "" && s.Port > 0 && s.Username != "" && s.Password != "" && s.FromAddress != ""
}

// ReceiverConfig lists forwarding recipients.
type ReceiverConfig struct {
	To []string `json:"to"`
}

// AutoReplyConfig controls the auto-reply feature.
type AutoReplyConfig struct {
	Enabled bool `json:"enabled"`
}

// TemplatesConfig groups templates by purpose for a single locale.
type TemplatesConfig struct {
	Forward   EmailTemplate `json:"forward"`
	AutoReply EmailTemplate `json:"autoReply"`
}

// EmailTemplate holds the subject and HTML body template.
type EmailTemplate struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// ---------- Service ----------

// EmailService handles loading email config and rendering/sending emails.
type EmailService struct {
	siteConfigRepo repository.SiteConfigRepository
}

// NewEmailService creates a new EmailService.
func NewEmailService(repo repository.SiteConfigRepository) *EmailService {
	return &EmailService{siteConfigRepo: repo}
}

// LoadConfig reads the "email" SiteConfig and returns the parsed EmailConfig, or nil.
func (s *EmailService) LoadConfig(ctx context.Context) *EmailConfig {
	doc, err := s.siteConfigRepo.FindByKey(ctx, model.SiteConfigKeyEmail)
	if err != nil {
		slog.Warn("email config not found", "error", err)
		return nil
	}
	if len(doc.PublishedConfig) == 0 {
		return nil
	}
	raw, err := json.Marshal(doc.PublishedConfig)
	if err != nil {
		slog.Error("failed to marshal email config", "error", err)
		return nil
	}
	var cfg EmailConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		slog.Error("failed to unmarshal email config", "error", err)
		return nil
	}
	return &cfg
}

// renderTemplate replaces placeholders in tmpl with values from submission.
func (s *EmailService) renderTemplate(tmpl string, submission *model.FormSubmission) string {
	locale := submission.Locale
	if locale == "" {
		locale = "zh"
	}

	var dateStr string
	if locale == "en" {
		dateStr = submission.CreatedAt.Format("Jan 02, 2006 3:04 PM")
	} else {
		dateStr = submission.CreatedAt.Format("2006-01-02 15:04")
	}

	r := strings.NewReplacer(
		"{{name}}", submission.Name,
		"{{email}}", submission.Email,
		"{{phone}}", submission.Phone,
		"{{company}}", submission.Company,
		"{{message}}", submission.Message,
		"{{date}}", dateStr,
	)
	return r.Replace(tmpl)
}

// getTemplate returns the template for the given locale, falling back to "zh".
func (s *EmailService) getTemplate(templates map[string]EmailTemplate, locale string) EmailTemplate {
	if t, ok := templates[locale]; ok {
		return t
	}
	if t, ok := templates["zh"]; ok {
		return t
	}
	return EmailTemplate{}
}
