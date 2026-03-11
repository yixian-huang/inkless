package sitemap

import (
	"encoding/xml"
	"net/http"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler handles sitemap generation
type Handler struct {
	contentDocRepo repository.ContentDocumentRepository
	articleRepo    repository.ArticleRepository
	baseURL        string
}

// NewHandler creates a new sitemap handler
func NewHandler(contentDocRepo repository.ContentDocumentRepository, articleRepo repository.ArticleRepository, baseURL string) *Handler {
	return &Handler{
		contentDocRepo: contentDocRepo,
		articleRepo:    articleRepo,
		baseURL:        baseURL,
	}
}

// urlset is the root element of a sitemap XML
type urlset struct {
	XMLName xml.Name  `xml:"urlset"`
	XMLNS   string    `xml:"xmlns,attr"`
	XHTMLns string    `xml:"xmlns:xhtml,attr"`
	URLs    []siteURL `xml:"url"`
}

type siteURL struct {
	Loc        string      `xml:"loc"`
	LastMod    string      `xml:"lastmod,omitempty"`
	ChangeFreq string      `xml:"changefreq,omitempty"`
	Priority   string      `xml:"priority,omitempty"`
	Links      []xhtmlLink `xml:"xhtml:link"`
}

type xhtmlLink struct {
	Rel      string `xml:"rel,attr"`
	Hreflang string `xml:"hreflang,attr"`
	Href     string `xml:"href,attr"`
}

// pageKeyToPath maps page keys to URL paths
var pageKeyToPath = map[model.PageKey]string{
	model.PageKeyHome:         "/",
	model.PageKeyAbout:        "/about",
	model.PageKeyAdvantages:   "/advantages",
	model.PageKeyCoreServices: "/core-services",
	model.PageKeyCases:        "/cases",
	model.PageKeyExperts:      "/experts",
	model.PageKeyContact:      "/contact",
}

// pageKeyPriority maps page keys to sitemap priority values
var pageKeyPriority = map[model.PageKey]string{
	model.PageKeyHome: "1.0",
}

// pageKeyChangeFreq maps page keys to sitemap changefreq values
var pageKeyChangeFreq = map[model.PageKey]string{
	model.PageKeyHome: "daily",
}

const defaultPagePriority = "0.6"
const defaultPageChangeFreq = "monthly"
const articlePriority = "0.8"
const articleChangeFreq = "weekly"

// GetSitemap generates and returns an XML sitemap
// GET /sitemap.xml
func (h *Handler) GetSitemap(c *gin.Context) {
	docs, err := h.contentDocRepo.List(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate sitemap")
		return
	}

	set := urlset{
		XMLNS:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		XHTMLns: "http://www.w3.org/1999/xhtml",
	}

	for _, doc := range docs {
		// Only include pages that have been published
		if doc.PublishedVersion == 0 {
			continue
		}

		path, ok := pageKeyToPath[doc.PageKey]
		if !ok {
			// Skip non-page documents (e.g., "global" config)
			continue
		}

		loc := h.baseURL + path

		priority := defaultPagePriority
		if p, ok := pageKeyPriority[doc.PageKey]; ok {
			priority = p
		}

		changeFreq := defaultPageChangeFreq
		if cf, ok := pageKeyChangeFreq[doc.PageKey]; ok {
			changeFreq = cf
		}

		u := siteURL{
			Loc:        loc,
			LastMod:    doc.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: changeFreq,
			Priority:   priority,
			Links: []xhtmlLink{
				{
					Rel:      "alternate",
					Hreflang: "zh",
					Href:     h.baseURL + path + "?locale=zh",
				},
				{
					Rel:      "alternate",
					Hreflang: "en",
					Href:     h.baseURL + path + "?locale=en",
				},
			},
		}

		set.URLs = append(set.URLs, u)
	}

	// Add published articles to sitemap
	if h.articleRepo != nil {
		articles, _, err := h.articleRepo.ListPublished(c.Request.Context(), 0, 1000, "", "")
		if err == nil {
			for _, article := range articles {
				path := "/articles/" + article.Slug
				loc := h.baseURL + path

				u := siteURL{
					Loc:        loc,
					LastMod:    article.UpdatedAt.Format("2006-01-02"),
					ChangeFreq: articleChangeFreq,
					Priority:   articlePriority,
					Links: []xhtmlLink{
						{
							Rel:      "alternate",
							Hreflang: "zh",
							Href:     h.baseURL + path + "?locale=zh",
						},
						{
							Rel:      "alternate",
							Hreflang: "en",
							Href:     h.baseURL + path + "?locale=en",
						},
					},
				}

				set.URLs = append(set.URLs, u)
			}
		}
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Writer.WriteString(xml.Header)
	enc := xml.NewEncoder(c.Writer)
	enc.Indent("", "  ")
	if err := enc.Encode(set); err != nil {
		// Headers already sent, just log
		return
	}
}
