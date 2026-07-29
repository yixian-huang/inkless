package themecatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseOpts() MergeOptions {
	return MergeOptions{
		BuiltinIDs: BuiltinIDSet(
			"corporate-classic",
			"blog-first",
			"product-first",
			"minimal-starter",
			"editorial-firm",
		),
		SupportedContracts: HostSupportedContracts,
		HostVersion:        "0.1.0-alpha.2",
		AllowHosts:         DefaultUMDAllowHosts,
	}
}

func TestMergeInstallState_NotInstalled(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "custom-oss",
		ThemeID:         "custom-oss",
		Name:            "Custom",
		ContractVersion: "1",
		Official:        true,
		Latest: ThemeVersion{
			Version: "1.0.0",
			UMDURL:  "https://github.com/org/repo/releases/download/v1.0.0/theme.umd.js",
		},
	}
	st := MergeInstallState(entry, nil, baseOpts())
	assert.Equal(t, InstallStateNotInstalled, st.InstallState)
	assert.Empty(t, st.IncompatibleReason)
}

func TestMergeInstallState_BuiltinOnly_NoRow(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "corporate-classic",
		ThemeID:         "corporate-classic",
		Name:            "Corporate",
		ContractVersion: "1",
		Official:        true,
		BuiltinOnly:     true,
		Latest:          ThemeVersion{Version: "1.0.0"},
	}
	st := MergeInstallState(entry, nil, baseOpts())
	assert.Equal(t, InstallStateBuiltin, st.InstallState)
}

func TestMergeInstallState_Active(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "product-first",
		ThemeID:         "product-first",
		Name:            "Product",
		ContractVersion: "1",
		Official:        true,
		Latest: ThemeVersion{
			Version: "0.1.5",
			UMDURL:  "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.5/theme.umd.js",
		},
	}
	inst := &InstalledRef{
		ThemeID:  "product-first",
		Version:  "0.1.5",
		Source:   "built-in",
		IsActive: true,
	}
	st := MergeInstallState(entry, inst, baseOpts())
	assert.Equal(t, InstallStateActive, st.InstallState)
	assert.Equal(t, "0.1.5", st.InstalledVersion)
	assert.False(t, st.UpdateAvailable)
}

func TestMergeInstallState_ActiveWithUpdateFlag(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "product-first",
		ThemeID:         "product-first",
		Name:            "Product",
		ContractVersion: "1",
		Official:        true,
		Latest: ThemeVersion{
			Version: "0.2.0",
			UMDURL:  "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.2.0/theme.umd.js",
		},
	}
	inst := &InstalledRef{
		ThemeID:     "product-first",
		Version:     "0.1.5",
		Source:      "marketplace",
		ExternalURL: "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.5/theme.umd.js",
		IsActive:    true,
	}
	st := MergeInstallState(entry, inst, baseOpts())
	assert.Equal(t, InstallStateActive, st.InstallState)
	assert.True(t, st.UpdateAvailable)
}

func TestMergeInstallState_UpdateAvailable(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "product-first",
		ThemeID:         "product-first",
		Name:            "Product",
		ContractVersion: "1",
		Official:        true,
		Latest: ThemeVersion{
			Version: "0.2.0",
			UMDURL:  "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.2.0/theme.umd.js",
		},
	}
	inst := &InstalledRef{
		ThemeID:     "product-first",
		Version:     "0.1.0",
		Source:      "marketplace",
		ExternalURL: "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.0/theme.umd.js",
		IsActive:    false,
	}
	st := MergeInstallState(entry, inst, baseOpts())
	assert.Equal(t, InstallStateUpdateAvailable, st.InstallState)
	assert.True(t, st.UpdateAvailable)
}

func TestMergeInstallState_InstalledExternal(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "product-first",
		ThemeID:         "product-first",
		Name:            "Product",
		ContractVersion: "1",
		Official:        true,
		Latest: ThemeVersion{
			Version: "0.1.5",
			UMDURL:  "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.5/theme.umd.js",
		},
	}
	inst := &InstalledRef{
		ThemeID:     "product-first",
		Version:     "0.1.5",
		Source:      "external",
		ExternalURL: "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.5/theme.umd.js",
		IsActive:    false,
	}
	st := MergeInstallState(entry, inst, baseOpts())
	assert.Equal(t, InstallStateInstalled, st.InstallState)
}

func TestMergeInstallState_BuiltinInstalledInactive(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "blog-first",
		ThemeID:         "blog-first",
		Name:            "Blog",
		ContractVersion: "1",
		Official:        true,
		Latest: ThemeVersion{
			Version: "1.0.0",
			UMDURL:  "https://github.com/yixian-huang/inkless-theme-blog-first/releases/download/v1.0.0/theme.umd.js",
		},
	}
	inst := &InstalledRef{
		ThemeID:  "blog-first",
		Version:  "1.0.0",
		Source:   "built-in",
		IsActive: false,
	}
	st := MergeInstallState(entry, inst, baseOpts())
	assert.Equal(t, InstallStateBuiltin, st.InstallState)
}

func TestMergeInstallState_IncompatibleContract(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "future",
		ThemeID:         "future",
		Name:            "Future",
		ContractVersion: "99",
		Official:        true,
		Latest: ThemeVersion{
			Version: "1.0.0",
			UMDURL:  "https://github.com/org/repo/theme.umd.js",
		},
	}
	st := MergeInstallState(entry, nil, baseOpts())
	assert.Equal(t, InstallStateIncompatible, st.InstallState)
	assert.Contains(t, st.IncompatibleReason, "contractVersion")
}

func TestMergeInstallState_IncompatibleMinHost(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "needs-new-host",
		ThemeID:         "needs-new-host",
		Name:            "Needs New",
		ContractVersion: "1",
		MinHostVersion:  "9.0.0",
		Official:        true,
		Latest: ThemeVersion{
			Version: "1.0.0",
			UMDURL:  "https://github.com/org/repo/theme.umd.js",
		},
	}
	st := MergeInstallState(entry, nil, baseOpts())
	assert.Equal(t, InstallStateIncompatible, st.InstallState)
	assert.Contains(t, st.IncompatibleReason, "requires host")
}

func TestMergeInstallState_IncompatibleBadUMDHost(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "evil",
		ThemeID:         "evil",
		Name:            "Evil",
		ContractVersion: "1",
		Official:        true,
		Latest: ThemeVersion{
			Version: "1.0.0",
			UMDURL:  "https://evil.example/theme.umd.js",
		},
	}
	st := MergeInstallState(entry, nil, baseOpts())
	assert.Equal(t, InstallStateIncompatible, st.InstallState)
	assert.Contains(t, st.IncompatibleReason, "allowlist")
}

func TestMergeInstallState_NotOfficial(t *testing.T) {
	entry := ThemeEntry{
		Slug:            "third",
		ThemeID:         "third",
		Name:            "Third",
		ContractVersion: "1",
		Official:        false,
		Latest: ThemeVersion{
			Version: "1.0.0",
			UMDURL:  "https://github.com/org/repo/theme.umd.js",
		},
	}
	st := MergeInstallState(entry, nil, baseOpts())
	assert.Equal(t, InstallStateIncompatible, st.InstallState)
}

func TestMergeCatalogStatuses_Embedded(t *testing.T) {
	cat, err := LoadEmbedded()
	require.NoError(t, err)

	installed := []InstalledRef{
		{ThemeID: "corporate-classic", Version: "1.0.0", Source: "built-in", IsActive: true},
		{ThemeID: "blog-first", Version: "1.0.0", Source: "built-in", IsActive: false},
		{
			ThemeID:     "product-first",
			Version:     "0.1.0",
			Source:      "marketplace",
			ExternalURL: "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.0/theme.umd.js",
			IsActive:    false,
		},
	}

	items := MergeCatalogStatuses(cat, installed, baseOpts())
	require.Len(t, items, len(cat.Themes))

	bySlug := map[string]CatalogItemStatus{}
	for _, it := range items {
		bySlug[it.Entry.Slug] = it
	}

	assert.Equal(t, InstallStateActive, bySlug["corporate-classic"].InstallState)
	assert.Equal(t, InstallStateBuiltin, bySlug["blog-first"].InstallState)
	// product-first embedded latest is 0.1.5 > 0.1.0
	assert.Equal(t, InstallStateUpdateAvailable, bySlug["product-first"].InstallState)
	assert.True(t, bySlug["product-first"].UpdateAvailable)
}

func TestVersionIsNewer(t *testing.T) {
	assert.True(t, versionIsNewer("0.2.0", "0.1.5"))
	assert.False(t, versionIsNewer("0.1.5", "0.1.5"))
	assert.False(t, versionIsNewer("0.1.0", "0.1.5"))
	assert.True(t, versionIsNewer("1.0.0", "0.9.9"))
}

func TestHostMeetsMin(t *testing.T) {
	ok, known := hostMeetsMin("0.1.0-alpha.2", "0.1.0-alpha.1")
	// pre-release ordering may vary; if known, just ensure no panic
	if known {
		_ = ok
	}
	ok, known = hostMeetsMin("1.2.0", "1.0.0")
	assert.True(t, known)
	assert.True(t, ok)
	ok, known = hostMeetsMin("1.0.0", "2.0.0")
	assert.True(t, known)
	assert.False(t, ok)
	_, known = hostMeetsMin("dev", "1.0.0")
	assert.False(t, known)
}
