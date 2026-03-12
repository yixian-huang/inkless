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

func TestOpenAIProvider_Chat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req openAIChatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "gpt-4o-mini", req.Model)
		assert.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]string{"content": "Hello from OpenAI"},
					"finish_reason": "stop",
				},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
			},
		})
	}))
	defer ts.Close()

	p := NewOpenAIProvider(OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})

	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.ChatMessage{{Role: "user", Content: "Hello"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello from OpenAI", resp.Content)
	assert.Equal(t, "gpt-4o-mini", resp.Model)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.PromptTokens)
	assert.Equal(t, 5, resp.OutputTokens)
}

func TestOpenAIProvider_Chat_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid API key",
				"type":    "authentication_error",
			},
		})
	}))
	defer ts.Close()

	p := NewOpenAIProvider(OpenAIConfig{
		APIKey:  "bad-key",
		BaseURL: ts.URL,
	})

	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.ChatMessage{{Role: "user", Content: "Hello"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestOpenAIProvider_Complete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]string{"content": "Completed text"},
					"finish_reason": "stop",
				},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]int{"prompt_tokens": 5, "completion_tokens": 3},
		})
	}))
	defer ts.Close()

	p := NewOpenAIProvider(OpenAIConfig{APIKey: "test-key", BaseURL: ts.URL})

	resp, err := p.Complete(context.Background(), provider.CompletionRequest{
		Prompt:    "Complete this:",
		MaxTokens: 100,
	})

	require.NoError(t, err)
	assert.Equal(t, "Completed text", resp.Text)
}

func TestOpenAIProvider_Summarize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Contains(t, req.Messages[0].Content, "Summarize")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]string{"content": "A short summary"},
					"finish_reason": "stop",
				},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]int{"prompt_tokens": 50, "completion_tokens": 10},
		})
	}))
	defer ts.Close()

	p := NewOpenAIProvider(OpenAIConfig{APIKey: "test-key", BaseURL: ts.URL})

	summary, err := p.Summarize(context.Background(), "Long text to summarize here.", 100)
	require.NoError(t, err)
	assert.Equal(t, "A short summary", summary)
}

func TestOpenAIProvider_SuggestTitles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]string{"content": `["Title One", "Title Two"]`},
					"finish_reason": "stop",
				},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 15},
		})
	}))
	defer ts.Close()

	p := NewOpenAIProvider(OpenAIConfig{APIKey: "test-key", BaseURL: ts.URL})

	titles, err := p.SuggestTitles(context.Background(), "Some content", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"Title One", "Title Two"}, titles)
}

func TestOpenAIProvider_SuggestTags(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]string{"content": `["go", "backend"]`},
					"finish_reason": "stop",
				},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 10},
		})
	}))
	defer ts.Close()

	p := NewOpenAIProvider(OpenAIConfig{APIKey: "test-key", BaseURL: ts.URL})

	tags, err := p.SuggestTags(context.Background(), "Go backend service", []string{"api"})
	require.NoError(t, err)
	assert.Equal(t, []string{"go", "backend"}, tags)
}

func TestOpenAIProvider_StreamChat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.True(t, req.Stream)

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"choices":[{"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: {"choices":[{"delta":{"content":""},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}
		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer ts.Close()

	p := NewOpenAIProvider(OpenAIConfig{APIKey: "test-key", BaseURL: ts.URL})

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

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider(OpenAIConfig{APIKey: "test"})
	assert.Equal(t, "openai", p.Name())
}

func TestParseJSONStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		hasErr   bool
	}{
		{
			name:     "plain array",
			input:    `["a", "b", "c"]`,
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with code fences",
			input:    "```json\n[\"a\", \"b\"]\n```",
			expected: []string{"a", "b"},
		},
		{
			name:   "invalid json",
			input:  "not json",
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJSONStringArray(tt.input)
			if tt.hasErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
