package comment

import (
	"context"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

const (
	AuthorRoleGuest  = "guest"
	AuthorRoleAuthor = "author"
)

func authorNameFromGlobalConfig(cfg model.JSONMap) string {
	if cfg == nil {
		return ""
	}
	if author, ok := cfg["author"].(map[string]interface{}); ok {
		if name, ok := author["name"].(string); ok && name != "" {
			return name
		}
	}
	if identity, ok := cfg["identity"].(map[string]interface{}); ok {
		if name, ok := identity["name"].(map[string]interface{}); ok {
			if zh, ok := name["zh"].(string); ok && zh != "" {
				return zh
			}
			if en, ok := name["en"].(string); ok && en != "" {
				return en
			}
		}
	}
	return ""
}

func resolveSiteAuthorName(
	ctx context.Context,
	siteCfg repository.SiteConfigRepository,
	contentDoc repository.ContentDocumentRepository,
) string {
	if siteCfg != nil {
		sc, err := siteCfg.FindByKey(ctx, model.SiteConfigKeyGlobal)
		if err == nil && sc != nil && sc.ID != 0 && sc.PublishedConfig != nil {
			if name := authorNameFromGlobalConfig(sc.PublishedConfig); name != "" {
				return name
			}
		}
	}
	if contentDoc != nil {
		doc, err := contentDoc.FindByPageKey(ctx, model.PageKeyGlobal)
		if err == nil && doc != nil && doc.PublishedConfig != nil {
			if name := authorNameFromGlobalConfig(doc.PublishedConfig); name != "" {
				return name
			}
		}
	}
	return "Author"
}
