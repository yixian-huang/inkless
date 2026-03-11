package seo

import "strings"

// ResolveFromPath builds a PageMeta from the request path, base URL, and locale.
// It sets canonical URL, og:url, locale, and og:type based on the path pattern.
func ResolveFromPath(path, baseURL, locale string) PageMeta {
	meta := DefaultPageMeta()
	meta.Locale = locale

	canonical := strings.TrimRight(baseURL, "/") + path
	meta.CanonicalURL = canonical
	meta.OgURL = canonical

	// Blog article pages get og:type "article"
	if strings.HasPrefix(path, "/blog/") && path != "/blog/" {
		meta.OgType = "article"
	}

	return meta
}
