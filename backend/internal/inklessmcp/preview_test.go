package inklessmcp

import (
	"testing"
	"time"
)

func TestPreviewStoreRoundTrip(t *testing.T) {
	s := newPreviewStore(time.Minute)
	body := map[string]any{"zhSeoTitle": "t"}
	id, err := s.Put("ops", 12, body)
	if err != nil || id == "" {
		t.Fatalf("put: %v %q", err, id)
	}
	// mutate original should not affect store
	body["zhSeoTitle"] = "changed"
	got, err := s.Take(id, "ops", 12)
	if err != nil {
		t.Fatal(err)
	}
	if got["zhSeoTitle"] != "t" {
		t.Fatalf("got=%v", got)
	}
	if _, err := s.Take(id, "ops", 12); err == nil {
		t.Fatal("expected consumed")
	}
}

func TestPreviewStoreMismatch(t *testing.T) {
	s := newPreviewStore(time.Minute)
	id, _ := s.Put("ops", 1, map[string]any{"a": 1})
	if _, err := s.Take(id, "ops", 2); err == nil {
		t.Fatal("want id mismatch")
	}
}

func TestPreviewStoreExpiry(t *testing.T) {
	s := newPreviewStore(time.Millisecond)
	id, _ := s.Put("ops", 1, map[string]any{"a": 1})
	time.Sleep(5 * time.Millisecond)
	if _, err := s.Take(id, "ops", 1); err == nil {
		t.Fatal("want expired")
	}
}
