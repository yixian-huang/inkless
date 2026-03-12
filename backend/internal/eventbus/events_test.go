package eventbus_test

import (
	"testing"

	"blotting-consultancy/internal/eventbus"
)

func TestContentEventTypes(t *testing.T) {
	events := []string{
		eventbus.ContentCreated,
		eventbus.ContentUpdated,
		eventbus.ContentPublished,
		eventbus.ContentDeleted,
		eventbus.CommentCreated,
		eventbus.CommentApproved,
		eventbus.CommentDeleted,
	}
	for _, e := range events {
		if e == "" {
			t.Error("event type constant should not be empty")
		}
	}
}

func TestContentEventPayload(t *testing.T) {
	payload := eventbus.ContentEventPayload{
		ContentType: "article",
		ContentID:   1,
		Slug:        "test-article",
		Locale:      "zh",
		Title:       "Test",
		Action:      eventbus.ContentPublished,
	}
	if payload.ContentType != "article" {
		t.Errorf("unexpected content type: %s", payload.ContentType)
	}
	if payload.ContentID != 1 {
		t.Errorf("unexpected content ID: %d", payload.ContentID)
	}
	if payload.Action != eventbus.ContentPublished {
		t.Errorf("unexpected action: %s", payload.Action)
	}
}

func TestCommentEventPayload(t *testing.T) {
	payload := eventbus.CommentEventPayload{
		CommentID:   5,
		ContentType: "article",
		ContentID:   10,
		AuthorName:  "Alice",
		Action:      eventbus.CommentCreated,
	}
	if payload.CommentID != 5 {
		t.Errorf("unexpected comment ID: %d", payload.CommentID)
	}
	if payload.AuthorName != "Alice" {
		t.Errorf("unexpected author name: %s", payload.AuthorName)
	}
}
