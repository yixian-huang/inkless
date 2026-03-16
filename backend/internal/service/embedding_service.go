package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode/utf8"

	"blotting-consultancy/internal/provider"
)

const (
	// DefaultChunkSize is the approximate number of characters per chunk (~500 tokens).
	DefaultChunkSize = 2000
	// DefaultChunkOverlap is the number of overlapping characters between chunks.
	DefaultChunkOverlap = 200
)

// EmbeddingService handles text chunking and vector storage for RAG indexing.
type EmbeddingService struct {
	ai           provider.AIProvider
	vectorStore  provider.VectorStoreProvider
	chunkSize    int
	chunkOverlap int
}

// NewEmbeddingService creates a new EmbeddingService.
func NewEmbeddingService(ai provider.AIProvider, vectorStore provider.VectorStoreProvider) *EmbeddingService {
	return &EmbeddingService{
		ai:           ai,
		vectorStore:  vectorStore,
		chunkSize:    DefaultChunkSize,
		chunkOverlap: DefaultChunkOverlap,
	}
}

// IndexContent chunks the given text, generates embeddings, and stores them.
// sourceID is a unique identifier for the content source (e.g., "article:42").
// metadata is additional key-value pairs stored alongside each chunk.
func (s *EmbeddingService) IndexContent(ctx context.Context, sourceID string, text string, metadata map[string]string) (int, error) {
	if s.ai == nil {
		return 0, ErrAINotConfigured
	}

	chunks := ChunkText(text, s.chunkSize, s.chunkOverlap)
	if len(chunks) == 0 {
		return 0, nil
	}

	indexed := 0
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		embedding, err := s.ai.Embed(ctx, chunk)
		if err != nil {
			return indexed, fmt.Errorf("embedding chunk %d: %w", i, err)
		}

		chunkID := chunkID(sourceID, i)
		chunkMeta := make(map[string]string, len(metadata)+2)
		for k, v := range metadata {
			chunkMeta[k] = v
		}
		chunkMeta["source_id"] = sourceID
		chunkMeta["chunk_index"] = fmt.Sprintf("%d", i)
		chunkMeta["chunk_text"] = chunk

		if err := s.vectorStore.Store(ctx, chunkID, embedding, chunkMeta); err != nil {
			return indexed, fmt.Errorf("storing chunk %d: %w", i, err)
		}
		indexed++
	}

	return indexed, nil
}

// DeleteContent removes all indexed chunks for a given source ID.
// Since we use deterministic IDs, we try deleting chunk IDs up to a reasonable limit.
func (s *EmbeddingService) DeleteContent(ctx context.Context, sourceID string) error {
	// Try deleting chunks with indices 0..999 (well beyond typical)
	for i := 0; i < 1000; i++ {
		_ = s.vectorStore.Delete(ctx, chunkID(sourceID, i))
	}
	return nil
}

// chunkID generates a deterministic ID for a chunk.
func chunkID(sourceID string, index int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s::%d", sourceID, index)))
	return fmt.Sprintf("%x", h[:8])
}

// ChunkText splits text into chunks of approximately chunkSize characters
// with overlap. It tries to break on paragraph or sentence boundaries.
func ChunkText(text string, chunkSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	charCount := utf8.RuneCountInString(text)
	if charCount <= chunkSize {
		return []string{text}
	}

	runes := []rune(text)
	var chunks []string
	start := 0

	for start < len(runes) {
		end := start + chunkSize
		if end >= len(runes) {
			end = len(runes)
		}

		// Try to find a good break point (paragraph break, then sentence end)
		if end < len(runes) {
			breakAt := findBreakPoint(runes, start, end)
			if breakAt > start {
				end = breakAt
			}
		}

		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		// Advance: move forward by at least 1 rune to guarantee progress
		nextStart := end - overlap
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
	}

	return chunks
}

// findBreakPoint looks for a natural break point (double newline, then period)
// in the range [start, end] of runes, searching backwards from end.
func findBreakPoint(runes []rune, start, end int) int {
	// First try to find a paragraph break (double newline)
	for i := end - 1; i > start+end/2; i-- {
		if i > 0 && runes[i] == '\n' && runes[i-1] == '\n' {
			return i + 1
		}
	}

	// Then try sentence boundaries
	for i := end - 1; i > start+end/2; i-- {
		r := runes[i]
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			return i + 1
		}
		// Chinese sentence endings
		if r == '\u3002' || r == '\uff01' || r == '\uff1f' {
			return i + 1
		}
	}

	return end
}
