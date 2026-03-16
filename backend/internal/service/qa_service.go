package service

import (
	"context"
	"fmt"
	"strings"

	"blotting-consultancy/internal/provider"
)

const (
	// DefaultTopK is the default number of context chunks to retrieve.
	DefaultTopK = 5
	// DefaultMinScore is the minimum similarity score to include a chunk.
	DefaultMinScore = 0.3
)

// QASource represents a source reference in a Q&A answer.
type QASource struct {
	SourceID   string `json:"sourceId"`
	ChunkIndex string `json:"chunkIndex"`
	Score      float64 `json:"score"`
	Preview    string `json:"preview"`
}

// QAResult represents the result of a Q&A query.
type QAResult struct {
	Answer  string     `json:"answer"`
	Sources []QASource `json:"sources"`
}

// QAService implements a RAG (Retrieval-Augmented Generation) pipeline.
type QAService struct {
	ai          provider.AIProvider
	vectorStore provider.VectorStoreProvider
	topK        int
	minScore    float64
}

// NewQAService creates a new QAService.
func NewQAService(ai provider.AIProvider, vectorStore provider.VectorStoreProvider) *QAService {
	return &QAService{
		ai:          ai,
		vectorStore: vectorStore,
		topK:        DefaultTopK,
		minScore:    DefaultMinScore,
	}
}

// Ask processes a user question through the RAG pipeline:
// 1. Embed the question
// 2. Search for relevant content chunks
// 3. Construct a prompt with retrieved context
// 4. Generate an answer via the LLM
func (s *QAService) Ask(ctx context.Context, question string, locale string) (*QAResult, error) {
	if strings.TrimSpace(question) == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	if s.ai == nil {
		return nil, ErrAINotConfigured
	}

	// Step 1: Get embedding for the question
	queryEmbedding, err := s.ai.Embed(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("embedding question: %w", err)
	}

	// Step 2: Search for relevant chunks
	results, err := s.vectorStore.Search(ctx, queryEmbedding, s.topK)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	// Filter by minimum score
	var relevantChunks []provider.VectorResult
	for _, r := range results {
		if r.Score >= s.minScore {
			relevantChunks = append(relevantChunks, r)
		}
	}

	// Build sources list
	sources := make([]QASource, 0, len(relevantChunks))
	var contextParts []string

	for _, chunk := range relevantChunks {
		chunkText := chunk.Metadata["chunk_text"]
		sourceID := chunk.Metadata["source_id"]
		chunkIndex := chunk.Metadata["chunk_index"]

		if chunkText != "" {
			contextParts = append(contextParts, chunkText)
		}

		// Create a preview (first 200 chars)
		preview := chunkText
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}

		sources = append(sources, QASource{
			SourceID:   sourceID,
			ChunkIndex: chunkIndex,
			Score:      chunk.Score,
			Preview:    preview,
		})
	}

	// Step 3: Construct prompt with context
	systemPrompt := buildSystemPrompt(contextParts, locale)

	// Step 4: Generate answer
	answer, err := s.ai.ChatComplete(ctx, systemPrompt, question)
	if err != nil {
		return nil, fmt.Errorf("generating answer: %w", err)
	}

	return &QAResult{
		Answer:  answer,
		Sources: sources,
	}, nil
}

// buildSystemPrompt constructs the system prompt with retrieved context.
func buildSystemPrompt(contextParts []string, locale string) string {
	langInstruction := "Please respond in Chinese."
	if locale == "en" {
		langInstruction = "Please respond in English."
	}

	if len(contextParts) == 0 {
		return fmt.Sprintf(`You are a helpful assistant for Blotting Consultancy (印迹咨询).
%s
If you don't have enough information to answer the question, politely say so and suggest the user contact us directly.`, langInstruction)
	}

	contextText := strings.Join(contextParts, "\n\n---\n\n")

	return fmt.Sprintf(`You are a helpful assistant for Blotting Consultancy (印迹咨询).
%s

Use the following reference content to answer the user's question. If the answer is not in the provided content, say so honestly and suggest contacting us directly. Do not make up information.

Reference content:
---
%s
---`, langInstruction, contextText)
}
