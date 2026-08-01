package themetemplates

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/model"
)

type fakeThemes struct {
	active *model.InstalledTheme
}

func (f *fakeThemes) FindActive(ctx context.Context) (*model.InstalledTheme, error) {
	return f.active, nil
}

func TestProjectSlot_ProductFirstHome(t *testing.T) {
	m := contentslots.ProductFirstManifest()
	tm := ProjectManifest(m)
	require.NotEmpty(t, tm.Templates)
	home, ok := FindBySlug(tm.Templates, "home")
	require.True(t, ok)
	assert.Equal(t, "page", home.AppliesTo)
	assert.Equal(t, "theme-page", home.Renderer)
	assert.Equal(t, "contentSlots-projection", home.Source)
	assert.NotEmpty(t, home.MediaRefPaths)
	assert.Equal(t, "product-first/home@1", home.Key)
	post, ok := FindByKey(tm.Templates, "product-first/post")
	require.True(t, ok)
	assert.Equal(t, "post", post.AppliesTo)
	assert.Equal(t, "product-first/home@1", tm.DefaultTemplates["home"])
}

func TestResolver_ProjectsActiveProductFirst(t *testing.T) {
	r := NewResolver(&fakeThemes{active: &model.InstalledTheme{
		ThemeID: builtinthemes.ProductFirst, Version: "0.1.9", IsActive: true,
	}}, contentslots.DefaultRegistry(), nil)
	res := r.ResolveActive(context.Background())
	assert.Equal(t, "projection", res.Source)
	assert.Equal(t, builtinthemes.ProductFirst, res.ActiveThemeID)
	require.NotEmpty(t, res.Templates)
	sum := Summarize(res.Templates)
	assert.NotEmpty(t, sum)
}

func TestResolver_BlogFirst(t *testing.T) {
	r := NewResolver(&fakeThemes{active: &model.InstalledTheme{
		ThemeID: builtinthemes.BlogFirst, Version: "1.0.0",
	}}, contentslots.DefaultRegistry(), nil)
	res := r.ResolveActive(context.Background())
	assert.Equal(t, "projection", res.Source)
	_, ok := FindBySlug(res.Templates, "home")
	assert.True(t, ok)
}
