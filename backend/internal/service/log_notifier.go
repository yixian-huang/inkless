package service

import (
	"context"
	"log"

	"blotting-consultancy/internal/provider"
)

// LogNotifier implements provider.NotifierProvider by logging notifications.
// This is the default implementation; plugins can replace with email, Webhook, etc.
type LogNotifier struct{}

// NewLogNotifier creates a new log-based notifier.
func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

// Notify logs the notification event.
func (n *LogNotifier) Notify(ctx context.Context, event provider.NotifyEvent) error {
	log.Printf("[NOTIFY] type=%s subject=%q body_len=%d meta=%v",
		event.Type, event.Subject, len(event.Body), event.Meta)
	return nil
}
