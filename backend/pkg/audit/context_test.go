package audit

import (
	"testing"
)

func TestActorLabel_APIKey(t *testing.T) {
	m := Metadata{Actor: "alice", ActorID: 7, APIKeyID: 42}
	got := m.ActorLabel()
	want := "alice (api_key:42)"
	if got != want {
		t.Fatalf("ActorLabel=%q want %q", got, want)
	}
}

func TestActorLabel_SessionOnly(t *testing.T) {
	m := Metadata{Actor: "bob", ActorID: 3}
	if got := m.ActorLabel(); got != "bob" {
		t.Fatalf("ActorLabel=%q", got)
	}
}

func TestAddMetadata_APIKey(t *testing.T) {
	details := AddMetadata(nil, Metadata{
		Actor:    "alice",
		ActorID:  7,
		APIKeyID: 42,
		IP:       "127.0.0.1",
	})
	if details["api_key_id"] != uint(42) {
		t.Fatalf("api_key_id=%v", details["api_key_id"])
	}
	if details["auth_method"] != "api_key" {
		t.Fatalf("auth_method=%v", details["auth_method"])
	}
	if details["actor_id"] != uint(7) {
		t.Fatalf("actor_id=%v", details["actor_id"])
	}
}
