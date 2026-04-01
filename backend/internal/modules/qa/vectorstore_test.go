package qa

import (
	"context"
	"math"
	"testing"

	"blotting-consultancy/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryVectorStore_StoreAndSearch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore()

	// Store some vectors
	err := store.Store(ctx, "v1", []float64{1, 0, 0}, map[string]string{"label": "x-axis"})
	require.NoError(t, err)

	err = store.Store(ctx, "v2", []float64{0, 1, 0}, map[string]string{"label": "y-axis"})
	require.NoError(t, err)

	err = store.Store(ctx, "v3", []float64{0.9, 0.1, 0}, map[string]string{"label": "near-x"})
	require.NoError(t, err)

	assert.Equal(t, 3, store.Count())

	// Search for vectors similar to x-axis
	results, err := store.Search(ctx, []float64{1, 0, 0}, 2)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// First result should be v1 (exact match)
	assert.Equal(t, "v1", results[0].ID)
	assert.InDelta(t, 1.0, results[0].Score, 0.001)
	assert.Equal(t, "x-axis", results[0].Metadata["label"])

	// Second result should be v3 (near-x)
	assert.Equal(t, "v3", results[1].ID)
	assert.True(t, results[1].Score > 0.9)
}

func TestMemoryVectorStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore()

	store.Store(ctx, "v1", []float64{1, 0}, map[string]string{})
	store.Store(ctx, "v2", []float64{0, 1}, map[string]string{})
	assert.Equal(t, 2, store.Count())

	err := store.Delete(ctx, "v1")
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())

	// Deleting non-existent key should not error
	err = store.Delete(ctx, "v999")
	require.NoError(t, err)
}

func TestMemoryVectorStore_SearchEmpty(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore()

	results, err := store.Search(ctx, []float64{1, 0}, 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestMemoryVectorStore_SearchTopKLargerThanStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore()

	store.Store(ctx, "v1", []float64{1, 0}, map[string]string{})

	results, err := store.Search(ctx, []float64{1, 0}, 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float64
		expected float64
	}{
		{"identical", []float64{1, 0}, []float64{1, 0}, 1.0},
		{"orthogonal", []float64{1, 0}, []float64{0, 1}, 0.0},
		{"opposite", []float64{1, 0}, []float64{-1, 0}, -1.0},
		{"empty", []float64{}, []float64{}, 0.0},
		{"different_lengths", []float64{1, 0}, []float64{1}, 0.0},
		{"zero_vector", []float64{0, 0}, []float64{1, 0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestMemoryVectorStore_Overwrite(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore()

	store.Store(ctx, "v1", []float64{1, 0}, map[string]string{"version": "1"})
	store.Store(ctx, "v1", []float64{0, 1}, map[string]string{"version": "2"})

	assert.Equal(t, 1, store.Count())

	results, err := store.Search(ctx, []float64{0, 1}, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "v1", results[0].ID)
	assert.InDelta(t, 1.0, results[0].Score, 0.001)
	assert.Equal(t, "2", results[0].Metadata["version"])
}

func TestStubAIProvider_Embed(t *testing.T) {
	ctx := context.Background()
	ai := service.NewStubAIProvider()

	emb1, err := ai.Embed(ctx, "hello world")
	require.NoError(t, err)
	assert.Len(t, emb1, 128)

	// Same text should produce same embedding
	emb2, err := ai.Embed(ctx, "hello world")
	require.NoError(t, err)
	assert.Equal(t, emb1, emb2)

	// Different text should produce different embedding
	emb3, err := ai.Embed(ctx, "goodbye world")
	require.NoError(t, err)
	assert.NotEqual(t, emb1, emb3)

	// Should be unit vector
	var norm float64
	for _, v := range emb1 {
		norm += v * v
	}
	assert.InDelta(t, 1.0, math.Sqrt(norm), 0.001)
}

func TestStubAIProvider_ChatComplete(t *testing.T) {
	ctx := context.Background()
	ai := service.NewStubAIProvider()

	answer, err := ai.ChatComplete(ctx, "system", "What is this?")
	require.NoError(t, err)
	assert.Contains(t, answer, "What is this?")
	assert.Contains(t, answer, "stub")
}
