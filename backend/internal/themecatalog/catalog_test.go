package themecatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbedded(t *testing.T) {
	cat, err := LoadEmbedded()
	require.NoError(t, err)
	require.NotNil(t, cat)
	assert.Equal(t, 1, cat.SchemaVersion)
	assert.GreaterOrEqual(t, len(cat.Themes), 3)

	pf, ok := cat.FindBySlug("product-first")
	require.True(t, ok)
	assert.Equal(t, "product-first", pf.ThemeID)
	assert.True(t, pf.Official)
	assert.False(t, pf.BuiltinOnly)
	assert.NotEmpty(t, pf.Latest.UMDURL)
	assert.NoError(t, ValidateUMDURL(pf.Latest.UMDURL, DefaultUMDAllowHosts))

	cc, ok := cat.FindBySlug("corporate-classic")
	require.True(t, ok)
	assert.True(t, cc.BuiltinOnly)
	assert.Empty(t, cc.Latest.UMDURL)
}

func TestParseCatalog_RejectsBadSchema(t *testing.T) {
	_, err := ParseCatalog([]byte(`{"schemaVersion":2,"themes":[{"slug":"x","themeId":"x","name":"X","contractVersion":"1","latest":{"version":"1","umdUrl":"https://github.com/x/y/theme.umd.js"},"official":true}]}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schemaVersion")
}

func TestParseCatalog_RejectsDuplicateSlug(t *testing.T) {
	raw := `{
	  "schemaVersion": 1,
	  "themes": [
	    {"slug":"a","themeId":"a","name":"A","contractVersion":"1","latest":{"version":"1","umdUrl":"https://github.com/a/b/x.js"},"official":true},
	    {"slug":"a","themeId":"b","name":"B","contractVersion":"1","latest":{"version":"1","umdUrl":"https://github.com/a/b/y.js"},"official":true}
	  ]
	}`
	_, err := ParseCatalog([]byte(raw))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate slug")
}

func TestResolveVersion(t *testing.T) {
	cat, err := LoadEmbedded()
	require.NoError(t, err)
	pf, ok := cat.FindBySlug("product-first")
	require.True(t, ok)

	v, err := pf.ResolveVersion("")
	require.NoError(t, err)
	assert.Equal(t, "0.1.5", v.Version)

	v2, err := pf.ResolveVersion("0.1.5")
	require.NoError(t, err)
	assert.Equal(t, pf.Latest.UMDURL, v2.UMDURL)

	_, err = pf.ResolveVersion("9.9.9")
	require.Error(t, err)
}

func TestValidateEntryInstallable(t *testing.T) {
	cat, err := LoadEmbedded()
	require.NoError(t, err)
	pf, _ := cat.FindBySlug("product-first")
	ver, err := pf.ResolveVersion("latest")
	require.NoError(t, err)
	require.NoError(t, ValidateEntryInstallable(pf, ver, DefaultUMDAllowHosts))

	nonOfficial := *pf
	nonOfficial.Official = false
	require.Error(t, ValidateEntryInstallable(&nonOfficial, ver, DefaultUMDAllowHosts))

	cc, _ := cat.FindBySlug("corporate-classic")
	require.NoError(t, ValidateEntryInstallable(cc, &cc.Latest, DefaultUMDAllowHosts))
}
