package provider

import (
	"context"
)

// NotifyEvent represents a notification to be sent.
type NotifyEvent struct {
	Type    string            // e.g. "comment.new", "content.published"
	Subject string            // notification subject/title
	Body    string            // notification body
	Meta    map[string]string // additional metadata
}

// NotifierProvider defines the contract for sending notifications.
// Default implementation logs to stdout.
// Plugins can replace with email, Webhook, Slack, etc.
type NotifierProvider interface {
	// Notify sends a notification event.
	Notify(ctx context.Context, event NotifyEvent) error
}
