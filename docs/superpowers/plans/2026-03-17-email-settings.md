# Email Settings Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add email configuration to system settings so contact form submissions trigger auto-reply and forwarding emails.

**Architecture:** Email config stored in `SiteConfig` (key=`"email"`). New `EmailService` handles SMTP transport and template rendering. New `email_settings` handler provides admin CRUD + test-send API. Form submission handler calls `EmailService` asynchronously after saving. Frontend adds a 3-tab admin page with CodeMirror HTML editor.

**Tech Stack:** Go/Gin/GORM (backend), React 19 + TypeScript + Tailwind + CodeMirror 6 (frontend)

**Spec:** `docs/superpowers/specs/2026-03-17-email-settings-design.md`

---

## File Map

### Backend — New
| File | Responsibility |
|------|---------------|
| `backend/internal/service/email_service.go` | SMTP transport, template rendering, config loading |
| `backend/internal/service/email_service_test.go` | Unit tests for template rendering and config parsing |
| `backend/internal/handler/email_settings/handler.go` | Admin API: GET/PUT config, POST test email |

### Backend — Modified
| File | Change |
|------|--------|
| `backend/internal/model/site_config.go` | Add `"email"` to valid keys |
| `backend/internal/handler/form_submission/handler.go` | Add `EmailService` dep, async email on submit |
| `backend/cmd/server/main.go` | Wire EmailService, register email_settings routes |

### Frontend — New
| File | Responsibility |
|------|---------------|
| `frontend/src/api/emailSettings.ts` | API client for email settings |
| `frontend/src/pages/admin/email-settings/page.tsx` | Main page with 3-tab layout |
| `frontend/src/pages/admin/email-settings/SmtpConfigTab.tsx` | SMTP config + toggles form |
| `frontend/src/pages/admin/email-settings/TemplateEditorTab.tsx` | Template editor with CodeMirror |
| `frontend/src/pages/admin/email-settings/types.ts` | TypeScript interfaces for email config |
| `frontend/src/pages/admin/email-settings/defaults.ts` | Default email templates |

### Frontend — Modified
| File | Change |
|------|--------|
| `frontend/src/router/config.tsx` | Add email-settings route |
| `frontend/src/pages/admin/components/AdminSidebar.tsx` | Add nav entry in System group |

---

## Chunk 1: Backend Core

### Task 1: Add "email" to SiteConfig Valid Keys

**Files:**
- Modify: `backend/internal/model/site_config.go`

- [ ] **Step 1: Add the constant**

In `backend/internal/model/site_config.go`, add `SiteConfigKeyEmail` alongside existing key constants:

```go
const (
	SiteConfigKeyGlobal = "global"
	SiteConfigKeyTheme  = "theme"
	SiteConfigKeyEmail  = "email"
)
```

Update the `Validate()` method to include `"email"` in the valid keys check. The current condition is:

```go
if sc.Key != SiteConfigKeyGlobal && sc.Key != SiteConfigKeyTheme {
```

Change it to:

```go
if sc.Key != SiteConfigKeyGlobal && sc.Key != SiteConfigKeyTheme && sc.Key != SiteConfigKeyEmail {
```

- [ ] **Step 2: Run existing tests**

```bash
cd backend && go test -v -race ./internal/model/...
```

Expected: PASS (existing tests still pass, new key accepted)

- [ ] **Step 3: Run go vet**

```bash
cd backend && go vet ./...
```

Expected: No issues

- [ ] **Step 4: Commit**

```bash
git add backend/internal/model/site_config.go
git commit -m "feat(email): add 'email' to SiteConfig valid keys"
```

---

### Task 2: Create EmailService — Template Rendering

**Files:**
- Create: `backend/internal/service/email_service.go`
- Create: `backend/internal/service/email_service_test.go`

- [ ] **Step 1: Define config types and EmailService struct**

Create `backend/internal/service/email_service.go`:

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// EmailConfig represents the email configuration stored in SiteConfig.
type EmailConfig struct {
	SMTP      SMTPConfig      `json:"smtp"`
	Receiver  ReceiverConfig  `json:"receiver"`
	AutoReply AutoReplyConfig `json:"autoReply"`
	Templates TemplatesConfig `json:"templates"`
}

type SMTPConfig struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	From               string `json:"from"`
	FromName           string `json:"fromName"`
	UseTLS             bool   `json:"useTLS"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

func (s SMTPConfig) IsConfigured() bool {
	return s.Host != "" && s.Port > 0 && s.From != ""
}

type ReceiverConfig struct {
	Enabled bool   `json:"enabled"`
	Email   string `json:"email"`
}

type AutoReplyConfig struct {
	Enabled bool `json:"enabled"`
}

type TemplatesConfig struct {
	AutoReply map[string]EmailTemplate `json:"autoReply"`
	Forward   map[string]EmailTemplate `json:"forward"`
}

type EmailTemplate struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type EmailService struct {
	siteConfigRepo repository.SiteConfigRepository
}

func NewEmailService(siteConfigRepo repository.SiteConfigRepository) *EmailService {
	return &EmailService{siteConfigRepo: siteConfigRepo}
}
```

- [ ] **Step 2: Implement LoadConfig**

Add to `email_service.go`:

```go
func (s *EmailService) LoadConfig(ctx context.Context) *EmailConfig {
	sc, err := s.siteConfigRepo.FindByKey(ctx, model.SiteConfigKeyEmail)
	if err != nil || sc == nil {
		return nil
	}
	if sc.PublishedConfig == nil {
		return nil
	}
	data, err := json.Marshal(sc.PublishedConfig)
	if err != nil {
		slog.Error("failed to marshal email config", "error", err)
		return nil
	}
	var cfg EmailConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		slog.Error("failed to parse email config", "error", err)
		return nil
	}
	return &cfg
}
```

- [ ] **Step 3: Implement renderTemplate**

Add to `email_service.go`:

```go
func (s *EmailService) renderTemplate(tmpl string, submission *model.FormSubmission) string {
	var dateStr string
	if submission.Locale == "en" {
		dateStr = submission.CreatedAt.Format("Jan 02, 2006 3:04 PM")
	} else {
		dateStr = submission.CreatedAt.Format("2006-01-02 15:04")
	}

	replacer := strings.NewReplacer(
		"{{name}}", submission.Name,
		"{{email}}", submission.Email,
		"{{phone}}", submission.Phone,
		"{{company}}", submission.Company,
		"{{message}}", submission.Message,
		"{{date}}", dateStr,
	)
	return replacer.Replace(tmpl)
}

func (s *EmailService) getTemplate(templates map[string]EmailTemplate, locale string) EmailTemplate {
	if t, ok := templates[locale]; ok {
		return t
	}
	if t, ok := templates["zh"]; ok {
		return t
	}
	return EmailTemplate{}
}
```

- [ ] **Step 4: Write tests for renderTemplate and getTemplate**

Create `backend/internal/service/email_service_test.go`:

```go
package service

import (
	"strings"
	"testing"
	"time"

	"blotting-consultancy/internal/model"
)

func TestRenderTemplate(t *testing.T) {
	svc := &EmailService{}
	submission := &model.FormSubmission{
		Name:    "张三",
		Email:   "zhang@example.com",
		Phone:   "13800138000",
		Company: "测试公司",
		Message: "你好，请联系我。",
		Locale:  "zh",
	}
	submission.CreatedAt = time.Date(2026, 3, 17, 14, 30, 0, 0, time.UTC)

	tmpl := "你好 {{name}}，我们收到了你在 {{date}} 的消息：{{message}}"
	result := svc.renderTemplate(tmpl, submission)

	expected := "你好 张三，我们收到了你在 2026-03-17 14:30 的消息：你好，请联系我。"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderTemplateEnglish(t *testing.T) {
	svc := &EmailService{}
	submission := &model.FormSubmission{
		Name:   "John",
		Email:  "john@example.com",
		Locale: "en",
	}
	submission.CreatedAt = time.Date(2026, 3, 17, 14, 30, 0, 0, time.UTC)

	tmpl := "Hello {{name}}, thank you for contacting us on {{date}}."
	result := svc.renderTemplate(tmpl, submission)

	expected := "Hello John, thank you for contacting us on Mar 17, 2026 2:30 PM."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderTemplateMissingVars(t *testing.T) {
	svc := &EmailService{}
	submission := &model.FormSubmission{
		Name:   "张三",
		Locale: "zh",
	}
	submission.CreatedAt = time.Now()

	tmpl := "{{name}} - {{phone}} - {{company}}"
	result := svc.renderTemplate(tmpl, submission)

	if !strings.Contains(result, "张三") {
		t.Errorf("expected name in result, got %q", result)
	}
	// Phone and company are empty strings — rendered as empty
	if strings.Contains(result, "{{phone}}") {
		t.Errorf("expected {{phone}} to be replaced, got %q", result)
	}
}

func TestGetTemplate(t *testing.T) {
	svc := &EmailService{}
	templates := map[string]EmailTemplate{
		"zh": {Subject: "中文主题", Body: "中文正文"},
		"en": {Subject: "English Subject", Body: "English body"},
	}

	// Exact match
	tmpl := svc.getTemplate(templates, "en")
	if tmpl.Subject != "English Subject" {
		t.Errorf("expected English template, got %q", tmpl.Subject)
	}

	// Fallback to zh
	tmpl = svc.getTemplate(templates, "fr")
	if tmpl.Subject != "中文主题" {
		t.Errorf("expected zh fallback, got %q", tmpl.Subject)
	}

	// Empty map
	tmpl = svc.getTemplate(map[string]EmailTemplate{}, "zh")
	if tmpl.Subject != "" {
		t.Errorf("expected empty template, got %q", tmpl.Subject)
	}
}

func TestSMTPConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  SMTPConfig
		want bool
	}{
		{"fully configured", SMTPConfig{Host: "smtp.gmail.com", Port: 587, From: "a@b.com"}, true},
		{"missing host", SMTPConfig{Port: 587, From: "a@b.com"}, false},
		{"missing port", SMTPConfig{Host: "smtp.gmail.com", From: "a@b.com"}, false},
		{"missing from", SMTPConfig{Host: "smtp.gmail.com", Port: 587}, false},
		{"all empty", SMTPConfig{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 5: Run tests**

```bash
cd backend && go test -v -race ./internal/service/... -run "TestRenderTemplate|TestGetTemplate|TestSMTPConfig"
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/email_service.go backend/internal/service/email_service_test.go
git commit -m "feat(email): add EmailService with template rendering and config types"
```

---

### Task 3: Create EmailService — SMTP Transport

**Files:**
- Modify: `backend/internal/service/email_service.go`

- [ ] **Step 1: Implement buildHTMLMessage**

Add to `email_service.go`:

```go
import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
)

func buildHTMLMessage(from, fromName, to, replyTo, subject, body string) []byte {
	var fromHeader string
	if fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", fromName, from)
	} else {
		fromHeader = from
	}

	msg := "From: " + fromHeader + "\r\n" +
		"To: " + to + "\r\n"
	if replyTo != "" {
		msg += "Reply-To: " + replyTo + "\r\n"
	}
	msg += "Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" + body

	return []byte(msg)
}
```

- [ ] **Step 2: Implement sendMail with STARTTLS and TLS support**

Add to `email_service.go`:

```go
func (s *EmailService) sendMail(cfg SMTPConfig, to, replyTo, subject, htmlBody string) error {
	msg := buildHTMLMessage(cfg.From, cfg.FromName, to, replyTo, subject, htmlBody)

	if cfg.UseTLS {
		return s.sendTLS(cfg, to, msg)
	}
	return s.sendSTARTTLS(cfg, to, msg)
}

func (s *EmailService) sendSTARTTLS(cfg SMTPConfig, to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("email: dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("email: smtp client: %w", err)
	}
	defer client.Quit()

	// STARTTLS with custom TLS config (respects InsecureSkipVerify)
	tlsConfig := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("email: starttls: %w", err)
	}

	if cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return fmt.Errorf("email: auth: %w", err)
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("email: mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("email: rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("email: write: %w", err)
	}
	return w.Close()
}

func (s *EmailService) sendTLS(cfg SMTPConfig, to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	tlsConfig := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("email: tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("email: smtp client: %w", err)
	}
	defer client.Quit()

	if cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return fmt.Errorf("email: auth: %w", err)
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("email: mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("email: rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("email: write: %w", err)
	}
	return w.Close()
}
```

- [ ] **Step 3: Implement SendAutoReply, SendForward, SendTest**

Add to `email_service.go`:

```go
func (s *EmailService) SendAutoReply(ctx context.Context, submission *model.FormSubmission, cfg *EmailConfig) error {
	tmpl := s.getTemplate(cfg.Templates.AutoReply, submission.Locale)
	if tmpl.Subject == "" && tmpl.Body == "" {
		return nil
	}
	subject := s.renderTemplate(tmpl.Subject, submission)
	body := s.renderTemplate(tmpl.Body, submission)
	return s.sendMail(cfg.SMTP, submission.Email, "", subject, body)
}

func (s *EmailService) SendForward(ctx context.Context, submission *model.FormSubmission, cfg *EmailConfig) error {
	if cfg.Receiver.Email == "" {
		return nil
	}
	tmpl := s.getTemplate(cfg.Templates.Forward, submission.Locale)
	if tmpl.Subject == "" && tmpl.Body == "" {
		return nil
	}
	subject := s.renderTemplate(tmpl.Subject, submission)
	body := s.renderTemplate(tmpl.Body, submission)
	return s.sendMail(cfg.SMTP, cfg.Receiver.Email, submission.Email, subject, body)
}

func (s *EmailService) SendTest(ctx context.Context, to string, cfg *EmailConfig) error {
	subject := "Test Email — 印迹咨询邮件配置测试"
	body := "<html><body><h2>邮件配置测试成功</h2><p>如果你收到这封邮件，说明 SMTP 配置正确。</p><p>Email configuration test successful.</p></body></html>"
	return s.sendMail(cfg.SMTP, to, "", subject, body)
}
```

- [ ] **Step 4: Add test for buildHTMLMessage**

Add to `email_service_test.go`:

```go
func TestBuildHTMLMessage(t *testing.T) {
	msg := string(buildHTMLMessage(
		"noreply@example.com", "印迹咨询",
		"user@example.com", "reply@example.com",
		"Test Subject", "<h1>Hello</h1>",
	))

	if !strings.Contains(msg, "Content-Type: text/html; charset=UTF-8") {
		t.Error("missing HTML content type")
	}
	if !strings.Contains(msg, "Reply-To: reply@example.com") {
		t.Error("missing Reply-To header")
	}
	if !strings.Contains(msg, "印迹咨询 <noreply@example.com>") {
		t.Error("missing formatted From header")
	}
	if !strings.Contains(msg, "<h1>Hello</h1>") {
		t.Error("missing body")
	}
}

func TestBuildHTMLMessageNoReplyTo(t *testing.T) {
	msg := string(buildHTMLMessage(
		"noreply@example.com", "",
		"user@example.com", "",
		"Test", "<p>body</p>",
	))

	if strings.Contains(msg, "Reply-To") {
		t.Error("should not contain Reply-To when empty")
	}
	// From should be just the email when no name
	if !strings.Contains(msg, "From: noreply@example.com\r\n") {
		t.Error("From header should be just email")
	}
}
```

- [ ] **Step 5: Run all email service tests**

```bash
cd backend && go test -v -race ./internal/service/... -run "Email|Template|SMTP|Build"
```

Expected: PASS

- [ ] **Step 6: Run go vet**

```bash
cd backend && go vet ./...
```

Expected: No issues

- [ ] **Step 7: Commit**

```bash
git add backend/internal/service/email_service.go backend/internal/service/email_service_test.go
git commit -m "feat(email): add SMTP transport with HTML and Reply-To support"
```

---

### Task 4: Create Email Settings Handler

**Files:**
- Create: `backend/internal/handler/email_settings/handler.go`
- [ ] **Step 1: Create handler with GET endpoint**

Create `backend/internal/handler/email_settings/handler.go`:

```go
package email_settings

import (
	"context"
	"net/http"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	siteConfigRepo repository.SiteConfigRepository
	emailService   *service.EmailService
}

func NewHandler(siteConfigRepo repository.SiteConfigRepository, emailService *service.EmailService) *Handler {
	return &Handler{
		siteConfigRepo: siteConfigRepo,
		emailService:   emailService,
	}
}

func (h *Handler) HandleGet(c *gin.Context) {
	sc, err := h.siteConfigRepo.FindByKey(c.Request.Context(), model.SiteConfigKeyEmail)
	if err != nil {
		// Not found or any error — return empty config (same pattern as theme handler)
		c.JSON(http.StatusOK, model.JSONMap{})
		return
	}

	config := make(model.JSONMap)
	if sc != nil && sc.PublishedConfig != nil {
		config = sc.PublishedConfig
	}

	maskPassword(config)
	c.JSON(http.StatusOK, config)
}

func maskPassword(config model.JSONMap) {
	if smtp, ok := config["smtp"].(map[string]interface{}); ok {
		if _, hasPassword := smtp["password"]; hasPassword {
			smtp["password"] = "****"
		}
	}
}
```

- [ ] **Step 2: Implement PUT endpoint (immediate publish)**

Add to handler.go:

```go
func (h *Handler) HandleUpdate(c *gin.Context) {
	var input model.JSONMap
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request body"}})
		return
	}

	ctx := c.Request.Context()

	// Preserve password if masked
	if err := h.preservePassword(ctx, input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to read existing config"}})
		return
	}

	sc, err := h.siteConfigRepo.FindByKey(ctx, model.SiteConfigKeyEmail)
	if err != nil {
		// Not found — treat as first time
		sc = nil
	}

	if sc == nil {
		// First time — create new
		sc = &model.SiteConfig{
			Key:              model.SiteConfigKeyEmail,
			DraftConfig:      input,
			DraftVersion:     1,
			PublishedConfig:  input,
			PublishedVersion: 1,
		}
		if err := h.siteConfigRepo.Upsert(ctx, sc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to save email settings"}})
			return
		}
	} else {
		// Update — immediate publish (same as theme handler)
		newVersion := sc.DraftVersion + 1
		sc.DraftConfig = input
		sc.DraftVersion = newVersion
		sc.PublishedConfig = input
		sc.PublishedVersion = newVersion
		if err := h.siteConfigRepo.Update(ctx, sc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to save email settings"}})
			return
		}
	}

	maskPassword(input)
	c.JSON(http.StatusOK, input)
}

func (h *Handler) preservePassword(ctx context.Context, input model.JSONMap) error {
	smtp, ok := input["smtp"].(map[string]interface{})
	if !ok {
		return nil
	}
	pw, ok := smtp["password"].(string)
	if !ok || pw != "****" {
		return nil
	}

	existing, err := h.siteConfigRepo.FindByKey(ctx, model.SiteConfigKeyEmail)
	if err != nil {
		return err
	}
	if existing == nil || existing.PublishedConfig == nil {
		smtp["password"] = ""
		return nil
	}

	if existingSMTP, ok := existing.PublishedConfig["smtp"].(map[string]interface{}); ok {
		if existingPW, ok := existingSMTP["password"].(string); ok {
			smtp["password"] = existingPW
		}
	}
	return nil
}
```

- [ ] **Step 3: Implement POST test endpoint**

Add to handler.go:

```go
func (h *Handler) HandleTest(c *gin.Context) {
	var input struct {
		To string `json:"to" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid email address"}})
		return
	}

	cfg := h.emailService.LoadConfig(c.Request.Context())
	if cfg == nil || !cfg.SMTP.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "SMTP is not configured"}})
		return
	}

	if err := h.emailService.SendTest(c.Request.Context(), input.To, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "发送失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试邮件已发送到 " + input.To,
	})
}
```

- [ ] **Step 4: Run go vet and build check**

```bash
cd backend && go vet ./internal/handler/email_settings/...
```

Expected: No issues

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/email_settings/
git commit -m "feat(email): add email settings admin handler (GET/PUT/test)"
```

---

### Task 5: Integrate Email Sending into Form Submission

**Files:**
- Modify: `backend/internal/handler/form_submission/handler.go`

- [ ] **Step 1: Add EmailService dependency to Handler struct**

In `backend/internal/handler/form_submission/handler.go`, modify the Handler struct and NewHandler:

```go
type Handler struct {
	repo         repository.FormSubmissionRepository
	emailService *service.EmailService
}

func NewHandler(repo repository.FormSubmissionRepository, emailService *service.EmailService) *Handler {
	return &Handler{repo: repo, emailService: emailService}
}
```

Add import: `"blotting-consultancy/internal/service"`

- [ ] **Step 2: Add async email sending to HandlePublicSubmit**

After the successful `c.JSON(http.StatusCreated, submission)` response in `HandlePublicSubmit`, add:

```go
// Async email notification — does not block the response
if h.emailService != nil {
    sub := submission // capture for goroutine
    go func() {
        bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        cfg := h.emailService.LoadConfig(bgCtx)
        if cfg == nil || !cfg.SMTP.IsConfigured() {
            return
        }
        if cfg.Receiver.Enabled {
            if err := h.emailService.SendForward(bgCtx, &sub, cfg); err != nil {
                slog.Error("forward email failed", "submissionId", sub.ID, "error", err)
            }
        }
        if cfg.AutoReply.Enabled {
            if err := h.emailService.SendAutoReply(bgCtx, &sub, cfg); err != nil {
                slog.Error("auto-reply email failed", "submissionId", sub.ID, "error", err)
            }
        }
    }()
}
```

Add imports: `"context"`, `"log/slog"`, `"time"`

- [ ] **Step 3: Run go vet**

```bash
cd backend && go vet ./internal/handler/form_submission/...
```

Expected: No issues

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/form_submission/handler.go
git commit -m "feat(email): add async email sending on form submission"
```

---

### Task 6: Wire Everything in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Create EmailService and update handler wiring**

In `backend/cmd/server/main.go`, in the handler creation section (around line 342):

1. Add import: `emailSettingsHandler "blotting-consultancy/internal/handler/email_settings"`
2. Create EmailService:
   ```go
   emailSvc := service.NewEmailService(siteConfigRepo)
   ```
3. Update form submission handler to pass emailSvc:
   ```go
   formSubmissionHandlerInst := formSubmissionHandler.NewHandler(formSubmissionRepo, emailSvc)
   ```
4. Create email settings handler:
   ```go
   emailSettingsHandlerInst := emailSettingsHandler.NewHandler(siteConfigRepo, emailSvc)
   ```

- [ ] **Step 2: Register email settings admin routes**

In the admin routes section (near the theme routes around line 656), add:

```go
// Email settings
adminGroup.GET("/email-settings", emailSettingsHandlerInst.HandleGet)
adminGroup.PUT("/email-settings", emailSettingsHandlerInst.HandleUpdate)
adminGroup.POST("/email-settings/test", emailSettingsHandlerInst.HandleTest)
```

- [ ] **Step 3: Build and verify**

```bash
cd backend && go build -o server ./cmd/server/
```

Expected: Build succeeds

- [ ] **Step 4: Run all Go tests**

```bash
cd backend && go test -v -race ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(email): wire EmailService and email settings routes in main"
```

---

## Chunk 2: Frontend

### Task 7: Add API Client and Types

**Files:**
- Create: `frontend/src/pages/admin/email-settings/types.ts`
- Create: `frontend/src/pages/admin/email-settings/defaults.ts`
- Create: `frontend/src/api/emailSettings.ts`

- [ ] **Step 1: Create TypeScript interfaces**

Create `frontend/src/pages/admin/email-settings/types.ts`:

```typescript
export interface SMTPConfig {
  host: string;
  port: number;
  username: string;
  password: string;
  from: string;
  fromName: string;
  useTLS: boolean;
  insecureSkipVerify: boolean;
}

export interface ReceiverConfig {
  enabled: boolean;
  email: string;
}

export interface AutoReplyConfig {
  enabled: boolean;
}

export interface EmailTemplate {
  subject: string;
  body: string;
}

export interface TemplatesConfig {
  autoReply: Record<string, EmailTemplate>;
  forward: Record<string, EmailTemplate>;
}

export interface EmailConfig {
  smtp: SMTPConfig;
  receiver: ReceiverConfig;
  autoReply: AutoReplyConfig;
  templates: TemplatesConfig;
}
```

- [ ] **Step 2: Create default templates**

Create `frontend/src/pages/admin/email-settings/defaults.ts`:

```typescript
import type { EmailConfig } from "./types";

const defaultAutoReplyZh = `<div style="font-family: 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1a5f8f;">亲爱的 {{name}}</h2>
  <p>感谢您联系印迹咨询，我们已收到您的消息。</p>
  <p>我们的团队将尽快审阅并回复您。</p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 20px 0;">
  <p style="color: #64748b; font-size: 14px;">您的消息内容：</p>
  <blockquote style="border-left: 3px solid #1a5f8f; padding-left: 12px; color: #475569;">{{message}}</blockquote>
  <p style="color: #94a3b8; font-size: 12px;">此邮件由系统自动发送，请勿直接回复。</p>
</div>`;

const defaultAutoReplyEn = `<div style="font-family: 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1a5f8f;">Dear {{name}}</h2>
  <p>Thank you for contacting us. We have received your message.</p>
  <p>Our team will review and respond to you as soon as possible.</p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 20px 0;">
  <p style="color: #64748b; font-size: 14px;">Your message:</p>
  <blockquote style="border-left: 3px solid #1a5f8f; padding-left: 12px; color: #475569;">{{message}}</blockquote>
  <p style="color: #94a3b8; font-size: 12px;">This is an automated message. Please do not reply directly.</p>
</div>`;

const defaultForwardZh = `<div style="font-family: 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1a5f8f;">新的联系表单提交</h2>
  <table style="width: 100%; border-collapse: collapse;">
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b; width: 80px;">姓名</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{name}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">邮箱</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{email}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">电话</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{phone}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">公司</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{company}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">时间</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{date}}</td></tr>
  </table>
  <h3 style="color: #1e293b; margin-top: 20px;">消息内容</h3>
  <div style="background: #f8fafc; padding: 16px; border-radius: 6px; white-space: pre-wrap;">{{message}}</div>
  <p style="color: #94a3b8; font-size: 12px; margin-top: 20px;">直接回复此邮件即可回复给 {{name}} ({{email}})。</p>
</div>`;

const defaultForwardEn = `<div style="font-family: 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1a5f8f;">New Contact Form Submission</h2>
  <table style="width: 100%; border-collapse: collapse;">
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b; width: 80px;">Name</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{name}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">Email</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{email}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">Phone</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{phone}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">Company</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{company}}</td></tr>
    <tr><td style="padding: 8px; border-bottom: 1px solid #e2e8f0; color: #64748b;">Date</td><td style="padding: 8px; border-bottom: 1px solid #e2e8f0;">{{date}}</td></tr>
  </table>
  <h3 style="color: #1e293b; margin-top: 20px;">Message</h3>
  <div style="background: #f8fafc; padding: 16px; border-radius: 6px; white-space: pre-wrap;">{{message}}</div>
  <p style="color: #94a3b8; font-size: 12px; margin-top: 20px;">Reply directly to this email to respond to {{name}} ({{email}}).</p>
</div>`;

export const defaultEmailConfig: EmailConfig = {
  smtp: {
    host: "",
    port: 587,
    username: "",
    password: "",
    from: "",
    fromName: "",
    useTLS: false,
    insecureSkipVerify: false,
  },
  receiver: {
    enabled: true,
    email: "",
  },
  autoReply: {
    enabled: true,
  },
  templates: {
    autoReply: {
      zh: { subject: "感谢您的联系 — {{name}}", body: defaultAutoReplyZh },
      en: { subject: "Thank you for contacting us — {{name}}", body: defaultAutoReplyEn },
    },
    forward: {
      zh: { subject: "新的联系表单：{{name}}", body: defaultForwardZh },
      en: { subject: "New contact form: {{name}}", body: defaultForwardEn },
    },
  },
};
```

- [ ] **Step 3: Create API client**

Create `frontend/src/api/emailSettings.ts`:

```typescript
import http from "./http";
import type { EmailConfig } from "@/pages/admin/email-settings/types";

export async function getEmailSettings(): Promise<EmailConfig> {
  const res = await http.get<EmailConfig>("/admin/email-settings");
  return res.data;
}

export async function updateEmailSettings(config: EmailConfig): Promise<EmailConfig> {
  const res = await http.put<EmailConfig>("/admin/email-settings", config);
  return res.data;
}

export async function sendTestEmail(to: string): Promise<{ success: boolean; message: string }> {
  const res = await http.post<{ success: boolean; message: string }>("/admin/email-settings/test", { to });
  return res.data;
}
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/admin/email-settings/types.ts frontend/src/pages/admin/email-settings/defaults.ts frontend/src/api/emailSettings.ts
git commit -m "feat(email): add frontend types, default templates, and API client"
```

---

### Task 8: Install CodeMirror Dependencies

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Install CodeMirror 6 packages**

```bash
cd /home/dev/impress && pnpm -C frontend add codemirror @codemirror/lang-html @codemirror/theme-one-dark @codemirror/view @codemirror/state
```

- [ ] **Step 2: Verify installation**

```bash
cd /home/dev/impress && pnpm -C frontend list codemirror
```

Expected: Shows installed codemirror packages

- [ ] **Step 3: Commit**

```bash
git add frontend/package.json frontend/pnpm-lock.yaml
git commit -m "chore: add CodeMirror 6 dependencies for HTML template editor"
```

---

### Task 9: Create Template Editor Tab Component

**Files:**
- Create: `frontend/src/pages/admin/email-settings/TemplateEditorTab.tsx`

- [ ] **Step 1: Create the TemplateEditorTab component**

Create `frontend/src/pages/admin/email-settings/TemplateEditorTab.tsx`:

```tsx
import { EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { basicSetup } from "codemirror";
import { html } from "@codemirror/lang-html";
import { oneDark } from "@codemirror/theme-one-dark";
import type { EmailTemplate } from "./types";

interface TemplateEditorTabProps {
  templates: Record<string, EmailTemplate>;
  onChange: (templates: Record<string, EmailTemplate>) => void;
  onSave: () => void;
}

const LOCALES = [
  { key: "zh", label: "中文" },
  { key: "en", label: "English" },
];

const VARIABLE_HINT = "可用变量: {{name}}, {{email}}, {{phone}}, {{company}}, {{message}}, {{date}}";

export default function TemplateEditorTab({ templates, onChange, onSave }: TemplateEditorTabProps) {
  const [locale, setLocale] = useState("zh");
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  const currentTemplate = templates[locale] || { subject: "", body: "" };

  const updateTemplate = useCallback(
    (field: "subject" | "body", value: string) => {
      onChange({
        ...templates,
        [locale]: { ...currentTemplate, [field]: value },
      });
    },
    [templates, locale, currentTemplate, onChange]
  );

  // Initialize CodeMirror
  useEffect(() => {
    if (!editorRef.current) return;

    const state = EditorState.create({
      doc: currentTemplate.body,
      extensions: [
        basicSetup,
        html(),
        oneDark,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            updateTemplate("body", update.state.doc.toString());
          }
        }),
      ],
    });

    const view = new EditorView({
      state,
      parent: editorRef.current,
    });

    viewRef.current = view;
    return () => view.destroy();
  }, [locale]); // Reinitialize when locale changes

  // Sync editor content when template changes externally
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (current !== currentTemplate.body) {
      view.dispatch({
        changes: { from: 0, to: current.length, insert: currentTemplate.body },
      });
    }
  }, [currentTemplate.body]);

  const [previewVisible, setPreviewVisible] = useState(false);

  return (
    <div className="space-y-4">
      {/* Locale toggle */}
      <div className="flex bg-gray-200 rounded-md w-fit overflow-hidden">
        {LOCALES.map((l) => (
          <button
            key={l.key}
            onClick={() => setLocale(l.key)}
            className={`px-4 py-1.5 text-sm font-medium transition-colors ${
              locale === l.key
                ? "bg-[#1a5f8f] text-white"
                : "text-gray-600 hover:text-gray-800"
            }`}
          >
            {l.label}
          </button>
        ))}
      </div>

      {/* Subject */}
      <div>
        <label className="block text-xs text-gray-500 mb-1">邮件主题</label>
        <input
          type="text"
          value={currentTemplate.subject}
          onChange={(e) => updateTemplate("subject", e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
        />
        <p className="text-xs text-gray-400 mt-1">{VARIABLE_HINT}</p>
      </div>

      {/* HTML Editor */}
      <div>
        <label className="block text-xs text-gray-500 mb-1">邮件正文（HTML）</label>
        <div ref={editorRef} className="border border-gray-700 rounded-md overflow-hidden min-h-[300px]" />
      </div>

      {/* Actions */}
      <div className="flex gap-3 justify-end">
        <button
          onClick={() => setPreviewVisible(true)}
          className="px-4 py-2 text-sm border border-gray-300 rounded-md text-gray-600 hover:bg-gray-50"
        >
          预览邮件
        </button>
        <button
          onClick={onSave}
          className="px-4 py-2 text-sm bg-[#1a5f8f] text-white rounded-md hover:bg-[#154d73]"
        >
          保存模板
        </button>
      </div>

      {/* Preview Modal */}
      {previewVisible && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setPreviewVisible(false)}>
          <div className="bg-white rounded-lg w-full max-w-2xl max-h-[80vh] overflow-auto" onClick={(e) => e.stopPropagation()}>
            <div className="flex justify-between items-center p-4 border-b">
              <h3 className="font-medium">邮件预览</h3>
              <button onClick={() => setPreviewVisible(false)} className="text-gray-400 hover:text-gray-600">&times;</button>
            </div>
            <div className="p-4">
              <p className="text-sm text-gray-500 mb-2">主题: {currentTemplate.subject}</p>
              <iframe
                srcDoc={currentTemplate.body}
                sandbox=""
                className="w-full h-[400px] border border-gray-200 rounded"
                title="Email Preview"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Run lint check**

```bash
pnpm lint
```

Expected: No new errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/admin/email-settings/TemplateEditorTab.tsx
git commit -m "feat(email): add TemplateEditorTab with CodeMirror HTML editor"
```

---

### Task 10: Create SMTP Config Tab Component

**Files:**
- Create: `frontend/src/pages/admin/email-settings/SmtpConfigTab.tsx`

- [ ] **Step 1: Create the SmtpConfigTab component**

Create `frontend/src/pages/admin/email-settings/SmtpConfigTab.tsx`:

```tsx
import type { EmailConfig } from "./types";

interface SmtpConfigTabProps {
  config: EmailConfig;
  onChange: (config: EmailConfig) => void;
  onSave: () => void;
  onTest: () => void;
  isSaving: boolean;
  isTesting: boolean;
}

export default function SmtpConfigTab({ config, onChange, onSave, onTest, isSaving, isTesting }: SmtpConfigTabProps) {
  const updateSMTP = (field: string, value: string | number | boolean) => {
    onChange({ ...config, smtp: { ...config.smtp, [field]: value } });
  };

  const updateReceiver = (field: string, value: string | boolean) => {
    onChange({ ...config, receiver: { ...config.receiver, [field]: value } });
  };

  const updateAutoReply = (field: string, value: boolean) => {
    onChange({ ...config, autoReply: { ...config.autoReply, [field]: value } });
  };

  return (
    <div className="space-y-4">
      {/* SMTP Server */}
      <div className="bg-white rounded-lg p-5 border border-gray-200">
        <h3 className="text-sm font-semibold text-gray-800 mb-4">SMTP 服务器</h3>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">服务器地址</label>
            <input
              type="text"
              value={config.smtp.host}
              onChange={(e) => updateSMTP("host", e.target.value)}
              placeholder="smtp.gmail.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">端口</label>
            <input
              type="number"
              value={config.smtp.port}
              onChange={(e) => updateSMTP("port", parseInt(e.target.value) || 0)}
              placeholder="587"
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">用户名</label>
            <input
              type="text"
              value={config.smtp.username}
              onChange={(e) => updateSMTP("username", e.target.value)}
              placeholder="noreply@example.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">密码</label>
            <input
              type="password"
              value={config.smtp.password}
              onChange={(e) => updateSMTP("password", e.target.value)}
              placeholder="应用专用密码"
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">发件人地址</label>
            <input
              type="email"
              value={config.smtp.from}
              onChange={(e) => updateSMTP("from", e.target.value)}
              placeholder="noreply@example.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">发件人名称</label>
            <input
              type="text"
              value={config.smtp.fromName}
              onChange={(e) => updateSMTP("fromName", e.target.value)}
              placeholder="印迹咨询"
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
            />
          </div>
        </div>
        <div className="mt-3 flex gap-5 items-center">
          <label className="text-sm text-gray-600 flex items-center gap-2">
            <input
              type="checkbox"
              checked={config.smtp.useTLS}
              onChange={(e) => updateSMTP("useTLS", e.target.checked)}
              className="rounded"
            />
            使用 TLS（端口 465）
          </label>
          <label className="text-sm text-gray-600 flex items-center gap-2">
            <input
              type="checkbox"
              checked={config.smtp.insecureSkipVerify}
              onChange={(e) => updateSMTP("insecureSkipVerify", e.target.checked)}
              className="rounded"
            />
            跳过证书验证（仅开发环境）
          </label>
        </div>
      </div>

      {/* Receiver */}
      <div className="bg-white rounded-lg p-5 border border-gray-200">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-sm font-semibold text-gray-800">转发通知</h3>
          <label className="flex items-center gap-2 cursor-pointer">
            <span className="text-xs text-gray-500">启用</span>
            <div className="relative">
              <input
                type="checkbox"
                checked={config.receiver.enabled}
                onChange={(e) => updateReceiver("enabled", e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-9 h-5 bg-gray-300 rounded-full peer-checked:bg-[#1a5f8f] transition-colors"></div>
              <div className="absolute left-0.5 top-0.5 w-4 h-4 bg-white rounded-full peer-checked:translate-x-4 transition-transform"></div>
            </div>
          </label>
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">接收邮箱</label>
          <input
            type="email"
            value={config.receiver.email}
            onChange={(e) => updateReceiver("email", e.target.value)}
            placeholder="admin@example.com"
            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#1a5f8f]"
          />
        </div>
      </div>

      {/* Auto Reply */}
      <div className="bg-white rounded-lg p-5 border border-gray-200">
        <div className="flex justify-between items-center">
          <h3 className="text-sm font-semibold text-gray-800">自动回复</h3>
          <label className="flex items-center gap-2 cursor-pointer">
            <span className="text-xs text-gray-500">启用</span>
            <div className="relative">
              <input
                type="checkbox"
                checked={config.autoReply.enabled}
                onChange={(e) => updateAutoReply("enabled", e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-9 h-5 bg-gray-300 rounded-full peer-checked:bg-[#1a5f8f] transition-colors"></div>
              <div className="absolute left-0.5 top-0.5 w-4 h-4 bg-white rounded-full peer-checked:translate-x-4 transition-transform"></div>
            </div>
          </label>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3 justify-end">
        <button
          onClick={onTest}
          disabled={isTesting}
          className="px-4 py-2 text-sm border border-gray-300 rounded-md text-gray-600 hover:bg-gray-50 disabled:opacity-50"
        >
          {isTesting ? "发送中..." : "发送测试邮件"}
        </button>
        <button
          onClick={onSave}
          disabled={isSaving}
          className="px-4 py-2 text-sm bg-[#1a5f8f] text-white rounded-md hover:bg-[#154d73] disabled:opacity-50"
        >
          {isSaving ? "保存中..." : "保存配置"}
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/admin/email-settings/SmtpConfigTab.tsx
git commit -m "feat(email): add SmtpConfigTab component"
```

---

### Task 11: Create Main Email Settings Page

**Files:**
- Create: `frontend/src/pages/admin/email-settings/page.tsx`

- [ ] **Step 1: Create the main page with 3-tab layout**

Create `frontend/src/pages/admin/email-settings/page.tsx`:

```tsx
import { getEmailSettings, updateEmailSettings, sendTestEmail } from "@/api/emailSettings";
import type { EmailConfig } from "./types";
import { defaultEmailConfig } from "./defaults";
import SmtpConfigTab from "./SmtpConfigTab";
import TemplateEditorTab from "./TemplateEditorTab";

const TABS = [
  { key: "smtp", label: "SMTP 配置" },
  { key: "autoReply", label: "自动回复模板" },
  { key: "forward", label: "转发通知模板" },
] as const;

type TabKey = (typeof TABS)[number]["key"];

export default function EmailSettingsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>("smtp");
  const [config, setConfig] = useState<EmailConfig>(defaultEmailConfig);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  useEffect(() => {
    getEmailSettings()
      .then((data) => {
        // Merge with defaults to fill any missing fields
        setConfig({
          smtp: { ...defaultEmailConfig.smtp, ...data.smtp },
          receiver: { ...defaultEmailConfig.receiver, ...data.receiver },
          autoReply: { ...defaultEmailConfig.autoReply, ...data.autoReply },
          templates: {
            autoReply: {
              ...defaultEmailConfig.templates.autoReply,
              ...data.templates?.autoReply,
            },
            forward: {
              ...defaultEmailConfig.templates.forward,
              ...data.templates?.forward,
            },
          },
        });
      })
      .catch(() => {
        // First time — use defaults
        setConfig(defaultEmailConfig);
      })
      .finally(() => setLoading(false));
  }, []);

  const handleSave = useCallback(async () => {
    setSaving(true);
    setMessage(null);
    try {
      const saved = await updateEmailSettings(config);
      setConfig((prev) => ({
        ...prev,
        smtp: { ...prev.smtp, ...saved.smtp },
        receiver: { ...prev.receiver, ...saved.receiver },
        autoReply: { ...prev.autoReply, ...saved.autoReply },
      }));
      setMessage({ type: "success", text: "保存成功" });
    } catch {
      setMessage({ type: "error", text: "保存失败，请重试" });
    } finally {
      setSaving(false);
    }
  }, [config]);

  const handleTest = useCallback(async () => {
    const to = prompt("请输入测试收件邮箱:");
    if (!to) return;
    setTesting(true);
    setMessage(null);
    try {
      const res = await sendTestEmail(to);
      setMessage({ type: res.success ? "success" : "error", text: res.message });
    } catch {
      setMessage({ type: "error", text: "测试邮件发送失败" });
    } finally {
      setTesting(false);
    }
  }, []);

  if (loading) {
    return <div className="p-6 text-gray-500">加载中...</div>;
  }

  return (
    <div className="p-6 max-w-4xl">
      <h1 className="text-xl font-semibold text-gray-800 mb-6">邮箱设置</h1>

      {/* Status message */}
      {message && (
        <div
          className={`mb-4 p-3 rounded-md text-sm ${
            message.type === "success"
              ? "bg-green-50 text-green-700 border border-green-200"
              : "bg-red-50 text-red-700 border border-red-200"
          }`}
        >
          {message.text}
        </div>
      )}

      {/* Tabs */}
      <div className="flex border-b-2 border-gray-200 mb-6">
        {TABS.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`px-5 py-3 text-sm font-medium transition-colors -mb-[2px] ${
              activeTab === tab.key
                ? "border-b-2 border-[#1a5f8f] text-[#1a5f8f]"
                : "text-gray-500 hover:text-gray-700"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === "smtp" && (
        <SmtpConfigTab
          config={config}
          onChange={setConfig}
          onSave={handleSave}
          onTest={handleTest}
          isSaving={saving}
          isTesting={testing}
        />
      )}
      {activeTab === "autoReply" && (
        <TemplateEditorTab
          templates={config.templates.autoReply}
          onChange={(autoReply) =>
            setConfig((prev) => ({
              ...prev,
              templates: { ...prev.templates, autoReply },
            }))
          }
          onSave={handleSave}
        />
      )}
      {activeTab === "forward" && (
        <TemplateEditorTab
          templates={config.templates.forward}
          onChange={(forward) =>
            setConfig((prev) => ({
              ...prev,
              templates: { ...prev.templates, forward },
            }))
          }
          onSave={handleSave}
        />
      )}
    </div>
  );
}
```

- [ ] **Step 2: Run lint and type-check**

```bash
pnpm lint && pnpm type-check
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/admin/email-settings/page.tsx
git commit -m "feat(email): add email settings page with 3-tab layout"
```

---

### Task 12: Add Route and Sidebar Navigation

**Files:**
- Modify: `frontend/src/router/config.tsx`
- Modify: `frontend/src/pages/admin/components/AdminSidebar.tsx`

- [ ] **Step 1: Add lazy import in router config**

In `frontend/src/router/config.tsx`, add a lazy import alongside the other admin pages:

```typescript
const AdminEmailSettingsPage = lazy(() => import("@/pages/admin/email-settings/page"));
```

Add a route entry in the admin children array:

```typescript
{ path: "email-settings", element: <AdminEmailSettingsPage /> },
```

- [ ] **Step 2: Add sidebar navigation entry**

In `frontend/src/pages/admin/components/AdminSidebar.tsx`, add a nav item in the "系统" (System) group, near "存储配置" (Storage):

```typescript
{ label: "邮箱设置", path: "/admin/email-settings", icon: MailIcon },
```

Use an appropriate icon. If the project uses lucide-react or heroicons, use the mail icon from that library. If using inline SVG, create a simple mail icon.

- [ ] **Step 3: Run lint and type-check**

```bash
pnpm lint && pnpm type-check
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/router/config.tsx frontend/src/pages/admin/components/AdminSidebar.tsx
git commit -m "feat(email): add email settings route and sidebar navigation"
```

---

## Chunk 3: Verification

### Task 13: Full Verification

- [ ] **Step 1: Run backend tests**

```bash
cd backend && go test -v -race ./...
```

Expected: All PASS

- [ ] **Step 2: Run backend build**

```bash
cd backend && go build -o server ./cmd/server/
```

Expected: Build succeeds

- [ ] **Step 3: Run frontend lint and type-check**

```bash
pnpm lint && pnpm type-check
```

Expected: PASS

- [ ] **Step 4: Run frontend build**

```bash
pnpm build
```

Expected: Build succeeds

- [ ] **Step 5: Start backend and test API manually**

```bash
cd /home/dev/impress/backend && PORT=8088 DB_DSN="file:./data/blotting.db?cache=shared&mode=rwc" JWT_SECRET=dev_jwt_secret_change_in_production JWT_REFRESH_SECRET=dev_jwt_refresh_secret_change_in_production ENV=development UPLOAD_DIR=./uploads FRONTEND_DIR=/home/dev/impress/frontend/out nohup ./server > /tmp/backend-dev.log 2>&1 &
```

Test the API:
```bash
# Get (should return empty/defaults)
curl --noproxy '*' -s http://127.0.0.1:8088/admin/email-settings -H "Authorization: Bearer <token>" | jq .

# PUT (save config)
curl --noproxy '*' -s -X PUT http://127.0.0.1:8088/admin/email-settings \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"smtp":{"host":"smtp.gmail.com","port":587,"username":"test","password":"secret","from":"test@example.com","fromName":"Test","useTLS":false},"receiver":{"enabled":true,"email":"admin@example.com"},"autoReply":{"enabled":true},"templates":{"autoReply":{"zh":{"subject":"感谢","body":"<p>感谢</p>"}},"forward":{"zh":{"subject":"新表单","body":"<p>表单</p>"}}}}' | jq .

# Verify password is masked in GET
curl --noproxy '*' -s http://127.0.0.1:8088/admin/email-settings -H "Authorization: Bearer <token>" | jq '.smtp.password'
# Expected: "****"
```
