package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"blotting-consultancy/internal/eventbus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_NoTargets(t *testing.T) {
	bus := eventbus.New()
	_, err := New(Config{}, bus)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one target is required")
}

func TestNew_EmptyURL(t *testing.T) {
	bus := eventbus.New()
	_, err := New(Config{
		Targets: []WebhookTarget{{URL: ""}},
	}, bus)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestNew_InvalidURLScheme(t *testing.T) {
	bus := eventbus.New()
	_, err := New(Config{
		Targets: []WebhookTarget{{URL: "ftp://invalid.example.com"}},
	}, bus)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http://")
}

func TestNew_DefaultRetryValues(t *testing.T) {
	bus := eventbus.New()
	p, err := New(Config{
		Targets: []WebhookTarget{
			{URL: "https://example.com/hook", Events: []string{"content.published"}},
		},
	}, bus)
	require.NoError(t, err)
	assert.Equal(t, 2, p.config.RetryCount)
	assert.Equal(t, 2*time.Second, p.config.RetryDelay)
}

func TestNewFromSettings_Valid(t *testing.T) {
	bus := eventbus.New()
	targets := []WebhookTarget{
		{URL: "https://hooks.slack.com/services/TOKEN", Events: []string{"content.published"}},
	}
	targetsJSON, _ := json.Marshal(targets)

	settings := map[string]string{
		"targets":     string(targetsJSON),
		"retry_count": "3",
	}
	p, err := NewFromSettings(settings, bus)
	require.NoError(t, err)
	assert.Len(t, p.config.Targets, 1)
	assert.Equal(t, 3, p.config.RetryCount)
}

func TestNewFromSettings_MissingTargets(t *testing.T) {
	bus := eventbus.New()
	_, err := NewFromSettings(map[string]string{}, bus)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'targets' setting is required")
}

func TestNewFromSettings_InvalidTargetsJSON(t *testing.T) {
	bus := eventbus.New()
	_, err := NewFromSettings(map[string]string{"targets": "not-json"}, bus)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse targets JSON")
}

func TestTargetWantsEvent_Specific(t *testing.T) {
	bus := eventbus.New()
	p, _ := New(Config{
		Targets: []WebhookTarget{
			{URL: "https://example.com", Events: []string{"content.published", "comment.created"}},
		},
	}, bus)

	target := p.config.Targets[0]
	assert.True(t, p.targetWantsEvent(target, "content.published"))
	assert.True(t, p.targetWantsEvent(target, "comment.created"))
	assert.False(t, p.targetWantsEvent(target, "content.deleted"))
}

func TestTargetWantsEvent_Wildcard(t *testing.T) {
	bus := eventbus.New()
	p, _ := New(Config{
		Targets: []WebhookTarget{
			{URL: "https://example.com", Events: []string{"*"}},
		},
	}, bus)

	target := p.config.Targets[0]
	assert.True(t, p.targetWantsEvent(target, "content.published"))
	assert.True(t, p.targetWantsEvent(target, "content.deleted"))
	assert.True(t, p.targetWantsEvent(target, "comment.created"))
	assert.True(t, p.targetWantsEvent(target, "any.custom.event"))
}

func TestDeliver_Success(t *testing.T) {
	var received WebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "content.published", r.Header.Get("X-Impress-Event"))

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	bus := eventbus.New()
	p, _ := New(Config{
		Targets: []WebhookTarget{{URL: srv.URL, Events: []string{"content.published"}}},
	}, bus)
	p.httpClient = srv.Client()

	payload := WebhookPayload{
		Event:     "content.published",
		Timestamp: time.Now().UTC(),
		Payload:   map[string]string{"slug": "my-article"},
	}

	err := p.deliver(p.config.Targets[0], payload)
	require.NoError(t, err)
	assert.Equal(t, "content.published", received.Event)
}

func TestDeliver_WithSignature(t *testing.T) {
	secret := "my-webhook-secret"
	var receivedSig string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Impress-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	bus := eventbus.New()
	p, _ := New(Config{
		Targets: []WebhookTarget{
			{URL: srv.URL, Secret: secret, Events: []string{"content.published"}},
		},
	}, bus)
	p.httpClient = srv.Client()

	payload := WebhookPayload{
		Event:     "content.published",
		Timestamp: time.Now().UTC(),
	}

	err := p.deliver(p.config.Targets[0], payload)
	require.NoError(t, err)
	assert.True(t, len(receivedSig) > 0)
	assert.Contains(t, receivedSig, "sha256=")
}

func TestDeliver_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	bus := eventbus.New()
	p, _ := New(Config{
		Targets: []WebhookTarget{{URL: srv.URL, Events: []string{"*"}}},
	}, bus)
	p.httpClient = srv.Client()

	err := p.deliver(p.config.Targets[0], WebhookPayload{Event: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestSubscribeAndHandleEvent(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	var receivedPayload WebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
		wg.Done()
	}))
	defer srv.Close()

	bus := eventbus.New()
	p, err := New(Config{
		Targets: []WebhookTarget{
			{URL: srv.URL, Events: []string{eventbus.ContentPublished}},
		},
		RetryCount: 0, // No retries in test
		RetryDelay: 0,
	}, bus)
	require.NoError(t, err)
	p.httpClient = srv.Client()

	p.Subscribe()

	bus.Publish(eventbus.Event{
		Type:    eventbus.ContentPublished,
		Payload: eventbus.ContentEventPayload{Slug: "hello-world"},
	})

	// Wait for async delivery
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.Equal(t, eventbus.ContentPublished, receivedPayload.Event)
	case <-time.After(3 * time.Second):
		t.Fatal("webhook delivery timed out")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := eventbus.New()
	p, _ := New(Config{
		Targets: []WebhookTarget{
			{URL: "https://example.com", Events: []string{eventbus.ContentPublished}},
		},
	}, bus)

	p.Subscribe()
	assert.NotEmpty(t, p.subscriptionIDs)

	p.Unsubscribe()
	assert.Empty(t, p.subscriptionIDs)
}

func TestComputeHMAC(t *testing.T) {
	// Deterministic test with known input/output
	data := []byte(`{"event":"test"}`)
	secret := "secret"
	sig := computeHMAC(data, secret)
	assert.NotEmpty(t, sig)
	assert.Len(t, sig, 64) // SHA-256 hex = 64 chars

	// Same input should produce same output
	sig2 := computeHMAC(data, secret)
	assert.Equal(t, sig, sig2)

	// Different secret should produce different signature
	sig3 := computeHMAC(data, "other-secret")
	assert.NotEqual(t, sig, sig3)
}

func TestManifest(t *testing.T) {
	err := Manifest.Validate()
	require.NoError(t, err)
}

func TestStandardEvents(t *testing.T) {
	events := standardEvents()
	assert.Contains(t, events, eventbus.ContentPublished)
	assert.Contains(t, events, eventbus.CommentCreated)
	assert.NotEmpty(t, events)
}
