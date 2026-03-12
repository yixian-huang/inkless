package service_test

import (
	"context"
	"testing"

	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/service"
)

// Verify LogNotifier implements NotifierProvider at compile time.
var _ provider.NotifierProvider = (*service.LogNotifier)(nil)

func TestLogNotifierImplementsInterface(t *testing.T) {
	var n provider.NotifierProvider = service.NewLogNotifier()
	err := n.Notify(context.Background(), provider.NotifyEvent{
		Type:    "test",
		Subject: "Test Notification",
		Body:    "This is a test",
	})
	if err != nil {
		t.Errorf("Notify should not fail: %v", err)
	}
}

func TestLogNotifierWithMeta(t *testing.T) {
	notifier := service.NewLogNotifier()
	err := notifier.Notify(context.Background(), provider.NotifyEvent{
		Type:    "comment.new",
		Subject: "New Comment",
		Body:    "Someone commented on your article",
		Meta: map[string]string{
			"articleId": "42",
			"author":    "Alice",
		},
	})
	if err != nil {
		t.Errorf("Notify with meta should not fail: %v", err)
	}
}
