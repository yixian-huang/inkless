// Package webhook provides a Webhook plugin for Impress CMS.
// It subscribes to EventBus events and pushes notifications to user-configured HTTP endpoints.
// Useful for integrating with Slack, Discord, CI/CD pipelines, or custom automation.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"blotting-consultancy/internal/eventbus"
	"blotting-consultancy/internal/plugin"
)

// Manifest describes this plugin's metadata.
var Manifest = plugin.PluginMeta{
	ID:            "wbh-webhook",
	Name:          "Webhook Plugin",
	NameZh:        "Webhook 推送插件",
	Version:       "1.0.0",
	Description:   "Subscribes to CMS events and POSTs JSON payloads to configured webhook URLs.",
	Author:        "Impress CMS",
	License:       "MIT",
	MinAppVersion: "1.0.0",
	Permissions:   []plugin.Permission{plugin.PermEventSubscribe, plugin.PermNetworkOutbound},
}

// WebhookTarget describes a single webhook endpoint configuration.
type WebhookTarget struct {
	// URL is the HTTP(S) endpoint to POST events to.
	URL string

	// Secret is an optional HMAC-SHA256 signing secret.
	// When set, the plugin adds an "X-Impress-Signature" header to requests.
	Secret string

	// Events is the list of event types to subscribe to (e.g. "content.published").
	// Use ["*"] to subscribe to all events.
	Events []string

	// Timeout is the HTTP request timeout for this target (default: 10s).
	Timeout time.Duration
}

// Config holds the Webhook plugin configuration.
type Config struct {
	// Targets is the list of webhook endpoints.
	Targets []WebhookTarget

	// RetryCount is the number of retry attempts on failure (default: 2).
	RetryCount int

	// RetryDelay is the delay between retry attempts (default: 2s).
	RetryDelay time.Duration
}

// WebhookPayload is the JSON body sent to webhook endpoints.
type WebhookPayload struct {
	Event     string      `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Plugin subscribes to EventBus events and forwards them to configured webhook endpoints.
type Plugin struct {
	config         Config
	bus            eventbus.EventBus
	subscriptionIDs map[string]uint64 // eventType -> subscriptionID
	httpClient     *http.Client
}

// New creates a new Webhook plugin with the provided configuration.
func New(cfg Config, bus eventbus.EventBus) (*Plugin, error) {
	if len(cfg.Targets) == 0 {
		return nil, fmt.Errorf("webhook: at least one target is required")
	}
	for i, t := range cfg.Targets {
		if t.URL == "" {
			return nil, fmt.Errorf("webhook: target[%d] URL is required", i)
		}
		if !strings.HasPrefix(t.URL, "http://") && !strings.HasPrefix(t.URL, "https://") {
			return nil, fmt.Errorf("webhook: target[%d] URL must start with http:// or https://", i)
		}
	}
	if cfg.RetryCount == 0 {
		cfg.RetryCount = 2
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 2 * time.Second
	}

	return &Plugin{
		config:          cfg,
		bus:             bus,
		subscriptionIDs: make(map[string]uint64),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// NewFromSettings creates a Plugin from a string settings map and EventBus.
// The settings are expected to have targets as JSON under the "targets" key.
func NewFromSettings(settings map[string]string, bus eventbus.EventBus) (*Plugin, error) {
	targetsJSON := settings["targets"]
	if targetsJSON == "" {
		return nil, fmt.Errorf("webhook: 'targets' setting is required (JSON array)")
	}

	var targets []WebhookTarget
	if err := json.Unmarshal([]byte(targetsJSON), &targets); err != nil {
		return nil, fmt.Errorf("webhook: failed to parse targets JSON: %w", err)
	}

	cfg := Config{Targets: targets}

	if v := settings["retry_count"]; v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			cfg.RetryCount = n
		}
	}

	return New(cfg, bus)
}

// Subscribe registers event handlers on the EventBus for all configured targets.
// Call this during plugin initialization.
func (p *Plugin) Subscribe() {
	// Collect all unique event types across all targets
	eventSet := make(map[string]struct{})
	for _, target := range p.config.Targets {
		for _, ev := range target.Events {
			if ev == "*" {
				// Subscribe to known standard events
				for _, known := range standardEvents() {
					eventSet[known] = struct{}{}
				}
			} else {
				eventSet[ev] = struct{}{}
			}
		}
	}

	for eventType := range eventSet {
		et := eventType // capture loop variable
		id := p.bus.Subscribe(et, eventbus.AsyncHandler(func(ev eventbus.Event) {
			p.handleEvent(ev)
		}))
		p.subscriptionIDs[et] = id
	}
}

// Unsubscribe removes all event subscriptions. Call during plugin shutdown.
func (p *Plugin) Unsubscribe() {
	for eventType, id := range p.subscriptionIDs {
		p.bus.Unsubscribe(eventType, id)
	}
	p.subscriptionIDs = make(map[string]uint64)
}

// handleEvent processes a single bus event and dispatches it to matching targets.
func (p *Plugin) handleEvent(ev eventbus.Event) {
	payload := WebhookPayload{
		Event:     ev.Type,
		Timestamp: time.Now().UTC(),
		Payload:   ev.Payload,
	}

	for _, target := range p.config.Targets {
		if !p.targetWantsEvent(target, ev.Type) {
			continue
		}
		// Send in the background per target to avoid blocking
		t := target
		go p.deliverWithRetry(t, payload)
	}
}

// targetWantsEvent returns true if the target is subscribed to the given event type.
func (p *Plugin) targetWantsEvent(target WebhookTarget, eventType string) bool {
	for _, ev := range target.Events {
		if ev == "*" || ev == eventType {
			return true
		}
	}
	return false
}

// deliverWithRetry attempts to deliver the webhook payload to the target,
// retrying on failure up to RetryCount times.
func (p *Plugin) deliverWithRetry(target WebhookTarget, payload WebhookPayload) {
	for attempt := 0; attempt <= p.config.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(p.config.RetryDelay)
		}

		if err := p.deliver(target, payload); err == nil {
			return // success
		}
		// On final failure, we log but don't panic - webhook delivery is best-effort
	}
}

// deliver sends a single webhook delivery attempt.
func (p *Plugin) deliver(target WebhookTarget, payload WebhookPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: failed to marshal payload: %w", err)
	}

	timeout := target.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("webhook: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Impress-CMS-Webhook/1.0")
	req.Header.Set("X-Impress-Event", payload.Event)
	req.Header.Set("X-Impress-Timestamp", payload.Timestamp.Format(time.RFC3339))

	if target.Secret != "" {
		sig := computeHMAC(data, target.Secret)
		req.Header.Set("X-Impress-Signature", "sha256="+sig)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: delivery to %s failed: %w", target.URL, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: target %s returned HTTP %d", target.URL, resp.StatusCode)
	}
	return nil
}

// computeHMAC computes HMAC-SHA256 of data using secret, returning hex string.
func computeHMAC(data []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// standardEvents returns the well-known CMS event types.
func standardEvents() []string {
	return []string{
		eventbus.ContentCreated,
		eventbus.ContentUpdated,
		eventbus.ContentPublished,
		eventbus.ContentDeleted,
		eventbus.CommentCreated,
		eventbus.CommentApproved,
		eventbus.CommentDeleted,
	}
}
