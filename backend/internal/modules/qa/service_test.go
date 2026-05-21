package qa

import (
	"context"
	"testing"

	"blotting-consultancy/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQAService_AskEmpty(t *testing.T) {
	ai := service.NewStubAIProvider()
	vs := NewMemoryVectorStore()
	svc := NewQAService(ai, vs)

	_, err := svc.Ask(context.Background(), "", "zh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestQAService_AskWithNoContent(t *testing.T) {
	ai := service.NewStubAIProvider()
	vs := NewMemoryVectorStore()
	svc := NewQAService(ai, vs)

	result, err := svc.Ask(context.Background(), "What is this company?", "zh")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Answer)
	assert.Empty(t, result.Sources) // no content indexed
}

func TestQAService_AskWithIndexedContent(t *testing.T) {
	ctx := context.Background()
	ai := service.NewStubAIProvider()
	vs := NewMemoryVectorStore()

	// Index some content
	embSvc := NewEmbeddingService(ai, vs)
	count, err := embSvc.IndexContent(ctx, "test:1", "We provide consulting services for businesses.", map[string]string{
		"type": "content",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Now query
	svc := NewQAService(ai, vs)
	result, err := svc.Ask(ctx, "What services do you provide?", "en")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Answer)
	// With the stub AI, the embedding similarity may or may not find relevant chunks
	// depending on the hash-based pseudo-embedding, so we just check structure
}

func TestQAService_EndToEnd(t *testing.T) {
	ctx := context.Background()
	ai := service.NewStubAIProvider()
	vs := NewMemoryVectorStore()

	// Index multiple pieces of content
	embSvc := NewEmbeddingService(ai, vs)
	embSvc.IndexContent(ctx, "about:1", "Our company was founded in 2020. We are experts in digital transformation.", map[string]string{"type": "page"})
	embSvc.IndexContent(ctx, "services:1", "We offer cloud migration, AI consulting, and data analytics.", map[string]string{"type": "page"})

	assert.True(t, vs.Count() >= 2, "expected at least 2 vectors stored")

	svc := NewQAService(ai, vs)
	result, err := svc.Ask(ctx, "When was the company founded?", "en")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Answer)
}

func TestBuildSystemPrompt(t *testing.T) {
	// No context
	prompt := buildSystemPrompt(nil, "zh")
	assert.Contains(t, prompt, "Chinese")
	assert.Contains(t, prompt, "helpful assistant")

	// With context, English locale
	prompt = buildSystemPrompt([]string{"chunk1", "chunk2"}, "en")
	assert.Contains(t, prompt, "English")
	assert.Contains(t, prompt, "chunk1")
	assert.Contains(t, prompt, "chunk2")
}
