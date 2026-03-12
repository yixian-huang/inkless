package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"blotting-consultancy/internal/provider"
)

func TestAnthropicProvider_Chat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "claude-sonnet-4-20250514", req.Model)
		assert.Equal(t, "You are helpful", req.System)
		assert.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "Hello from Claude"},
			},
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		})
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})

	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello from Claude", resp.Content)
	assert.Equal(t, "claude-sonnet-4-20250514", resp.Model)
	assert.Equal(t, "end_turn", resp.FinishReason)
	assert.Equal(t, 10, resp.PromptTokens)
	assert.Equal(t, 5, resp.OutputTokens)
}

func TestAnthropicProvider_Chat_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "Invalid API key",
			},
		})
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{
		APIKey:  "bad-key",
		BaseURL: ts.URL,
	})

	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.ChatMessage{{Role: "user", Content: "Hello"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestAnthropicProvider_Complete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "Completed text"},
			},
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 3},
		})
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{APIKey: "test-key", BaseURL: ts.URL})

	resp, err := p.Complete(context.Background(), provider.CompletionRequest{
		Prompt:    "Complete this:",
		MaxTokens: 100,
	})

	require.NoError(t, err)
	assert.Equal(t, "Completed text", resp.Text)
}

func TestAnthropicProvider_Summarize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Contains(t, req.Messages[0].Content, "Summarize")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "A short summary"},
			},
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 50, "output_tokens": 10},
		})
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{APIKey: "test-key", BaseURL: ts.URL})

	summary, err := p.Summarize(context.Background(), "Long text to summarize here.", 100)
	require.NoError(t, err)
	assert.Equal(t, "A short summary", summary)
}

func TestAnthropicProvider_SuggestTitles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": `["Title One", "Title Two"]`},
			},
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 20, "output_tokens": 15},
		})
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{APIKey: "test-key", BaseURL: ts.URL})

	titles, err := p.SuggestTitles(context.Background(), "Some content", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"Title One", "Title Two"}, titles)
}

func TestAnthropicProvider_SuggestTags(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": `["go", "backend"]`},
			},
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 20, "output_tokens": 10},
		})
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{APIKey: "test-key", BaseURL: ts.URL})

	tags, err := p.SuggestTags(context.Background(), "Go backend service", []string{"api"})
	require.NoError(t, err)
	assert.Equal(t, []string{"go", "backend"}, tags)
}

func TestAnthropicProvider_StreamChat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.True(t, req.Stream)

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		events := []string{
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`,
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":" world"}}`,
			`data: {"type":"message_stop"}`,
		}
		for _, event := range events {
			w.Write([]byte(event + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{APIKey: "test-key", BaseURL: ts.URL})

	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Messages: []provider.ChatMessage{{Role: "user", Content: "Hello"}},
	})
	require.NoError(t, err)

	var collected string
	for chunk := range ch {
		require.NoError(t, chunk.Err)
		collected += chunk.Content
	}
	assert.Equal(t, "Hello world", collected)
}

func TestAnthropicProvider_Name(t *testing.T) {
	p := NewAnthropicProvider(AnthropicConfig{APIKey: "test"})
	assert.Equal(t, "anthropic", p.Name())
}

func TestAnthropicProvider_SystemMessageExtraction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		json.NewDecoder(r.Body).Decode(&req)
		// System message should be extracted to the system field
		assert.Equal(t, "Be helpful", req.System)
		// Only user message should remain
		assert.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "OK"},
			},
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 2},
		})
	}))
	defer ts.Close()

	p := NewAnthropicProvider(AnthropicConfig{APIKey: "test-key", BaseURL: ts.URL})

	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "Be helpful"},
			{Role: "user", Content: "Hello"},
		},
	})
	require.NoError(t, err)
}
