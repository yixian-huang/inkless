package contentslots

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/model"
)

type fakeThemes struct {
	active *model.InstalledTheme
	err    error
}

func (f *fakeThemes) FindActive(ctx context.Context) (*model.InstalledTheme, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.active, nil
}

func TestResolver_ProductFirstFromRegistry(t *testing.T) {
	r := NewResolver(&fakeThemes{active: &model.InstalledTheme{
		ThemeID: builtinthemes.ProductFirst,
		Version: "0.1.9",
		IsActive: true,
	}}, DefaultRegistry())

	res := r.ResolveActive(context.Background())
	assert.Equal(t, "theme", res.Source)
	assert.Equal(t, builtinthemes.ProductFirst, res.ActiveThemeID)
	require.NotEmpty(t, res.Slots)
	assert.Equal(t, "home", res.Slots[0].PageKey)

	_, slot, ok := r.ResolveSlot(context.Background(), "home")
	require.True(t, ok)
	assert.Equal(t, "product-first/home@1", slot.SchemaID)
}

func TestResolver_HostFallbackNoSlots(t *testing.T) {
	r := NewResolver(&fakeThemes{active: &model.InstalledTheme{
		ThemeID: builtinthemes.BlogFirst,
		Version: "1.0.0",
	}}, DefaultRegistry())

	res := r.ResolveActive(context.Background())
	assert.Equal(t, "host-fallback", res.Source)
	assert.Empty(t, res.Slots)
}

func TestResolver_ConfigOverride(t *testing.T) {
	r := NewResolver(&fakeThemes{active: &model.InstalledTheme{
		ThemeID: "custom-theme",
		Version: "2.0.0",
		Config: model.JSONMap{
			"contentSlots": []any{
				map[string]any{
					"pageKey":  "home",
					"schemaId": "custom/home@1",
					"stringPaths": []any{"foo"},
				},
			},
		},
	}}, DefaultRegistry())

	res := r.ResolveActive(context.Background())
	assert.Equal(t, "theme", res.Source)
	require.Len(t, res.Slots, 1)
	assert.Equal(t, "custom/home@1", res.Slots[0].SchemaID)
}

func TestResolver_NoActive(t *testing.T) {
	r := NewResolver(&fakeThemes{err: errors.New("not found")}, DefaultRegistry())
	res := r.ResolveActive(context.Background())
	assert.Equal(t, "none", res.Source)
}
