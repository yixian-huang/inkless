package qa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkText_Short(t *testing.T) {
	chunks := ChunkText("Hello world", 2000, 200)
	assert.Len(t, chunks, 1)
	assert.Equal(t, "Hello world", chunks[0])
}

func TestChunkText_Empty(t *testing.T) {
	chunks := ChunkText("", 2000, 200)
	assert.Nil(t, chunks)

	chunks = ChunkText("   ", 2000, 200)
	assert.Nil(t, chunks)
}

func TestChunkText_LongText(t *testing.T) {
	// Create a text that's longer than chunk size
	text := ""
	for i := 0; i < 100; i++ {
		text += "This is sentence number. "
	}

	chunks := ChunkText(text, 100, 20)
	assert.True(t, len(chunks) > 1, "expected multiple chunks, got %d", len(chunks))

	// All chunks should be non-empty
	for i, c := range chunks {
		assert.NotEmpty(t, c, "chunk %d is empty", i)
	}
}

func TestChunkText_ChineseText(t *testing.T) {
	text := ""
	for i := 0; i < 50; i++ {
		text += "这是一个测试句子。"
	}

	chunks := ChunkText(text, 100, 20)
	assert.True(t, len(chunks) > 1)

	for _, c := range chunks {
		assert.NotEmpty(t, c)
	}
}

func TestChunkText_ParagraphBreaks(t *testing.T) {
	text := "First paragraph with enough content to fill the space.\n\n" +
		"Second paragraph with more content to test paragraph breaking.\n\n" +
		"Third paragraph that adds even more text to ensure chunking happens."

	chunks := ChunkText(text, 60, 10)
	assert.True(t, len(chunks) >= 2)
}
