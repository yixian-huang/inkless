package builtinthemes

const (
	CorporateClassic = "corporate-classic"
	BlogFirst        = "blog-first"
	ProductFirst     = "product-first"
	MinimalStarter   = "minimal-starter"
	EditorialFirm    = "editorial-firm"
)

// DefaultFallbackThemeID is used when activation fails or no theme is set.
const DefaultFallbackThemeID = CorporateClassic

// BlankSiteDefaultThemeID is activated for fresh blank-site seeds.
const BlankSiteDefaultThemeID = BlogFirst
