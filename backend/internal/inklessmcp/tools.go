package inklessmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yixian-huang/inkless/backend/internal/agentcli"
)

func (s *Server) registerTools(srv *mcp.Server) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(true), Title: "Read-only"}
	rw := &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(true),
		OpenWorldHint:   boolPtr(true),
		Title:           "Mutating",
	}

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_sites",
		Title:       "List fleet sites",
		Description: "List sites from the local fleet registry (no secrets). Use site_id on subsequent tools.",
		Annotations: ro,
	}, s.toolListSites)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "resolve_site",
		Title:       "Resolve site endpoint",
		Description: "Resolve baseUrl and key source for a site_id (API key is masked).",
		Annotations: ro,
	}, s.toolResolveSite)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "whoami",
		Title:       "Instance whoami",
		Description: "Call GET /admin/agent/whoami and verify baseUrl matches the resolved site.",
		Annotations: ro,
	}, s.toolWhoami)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_articles",
		Title:       "List articles",
		Description: "List admin articles. Set missing_seo=true to filter incomplete SEO fields.",
		Annotations: ro,
	}, s.toolListArticles)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_article",
		Title:       "Get article",
		Description: "GET /admin/articles/:id as JSON.",
		Annotations: ro,
	}, s.toolGetArticle)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "apply_article_patch",
		Title: "Apply article SEO/content patch",
		Description: `Merge a partial article JSON onto the current article and PUT.
Default dry_run=true returns preview_handle + merged body without writing.
To commit: dry_run=false with preview_handle from a prior dry_run, or with patch fields.`,
		Annotations: rw,
	}, s.toolApplyArticlePatch)
}

type siteArgs struct {
	SiteID string `json:"site_id" jsonschema:"Fleet site id (optional if default_site or single-site env)"`
}

type emptyArgs struct{}

type listArticlesArgs struct {
	SiteID     string `json:"site_id" jsonschema:"Fleet site id"`
	Page       int    `json:"page,omitempty" jsonschema:"Page number, default 1"`
	PageSize   int    `json:"page_size,omitempty" jsonschema:"Page size, default 20"`
	Status     string `json:"status,omitempty" jsonschema:"draft|published|scheduled"`
	Q          string `json:"q,omitempty" jsonschema:"Search query"`
	MissingSEO bool   `json:"missing_seo,omitempty" jsonschema:"If true, only articles missing SEO fields"`
}

type getArticleArgs struct {
	SiteID string `json:"site_id" jsonschema:"Fleet site id"`
	ID     uint   `json:"id" jsonschema:"Article id"`
}

type applyArticleArgs struct {
	SiteID         string         `json:"site_id" jsonschema:"Fleet site id"`
	ID             uint           `json:"id" jsonschema:"Article id"`
	Patch          map[string]any `json:"patch,omitempty" jsonschema:"Partial article fields to merge"`
	DryRun         *bool          `json:"dry_run,omitempty" jsonschema:"Default true: preview only"`
	PreviewHandle  string         `json:"preview_handle,omitempty" jsonschema:"Handle from prior dry_run to commit"`
}

func (s *Server) toolListSites(ctx context.Context, _ *mcp.CallToolRequest, _ emptyArgs) (*mcp.CallToolResult, any, error) {
	path, fleet, err := agentcli.ListSites(s.opts.FleetPath)
	if err != nil {
		return toolError(err)
	}
	sites := make([]map[string]any, 0, len(fleet.Sites))
	for id, site := range fleet.Sites {
		policy := site.PublishPolicy
		if policy == "" {
			policy = "manual"
		}
		sites = append(sites, map[string]any{
			"id":            id,
			"label":         site.Label,
			"baseUrl":       agentcli.NormalizeBaseURL(site.BaseURL),
			"publishPolicy": policy,
			"brand":         site.Brand,
			"default":       id == fleet.DefaultSite,
			"apiKeyEnv":     site.APIKeyEnv,
			// never include key material
		})
	}
	return textResult(map[string]any{
		"fleet":       path,
		"defaultSite": fleet.DefaultSite,
		"sites":       sites,
	})
}

func (s *Server) toolResolveSite(ctx context.Context, _ *mcp.CallToolRequest, args siteArgs) (*mcp.CallToolResult, any, error) {
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	return textResult(publicEP(ep))
}

func (s *Server) toolWhoami(ctx context.Context, _ *mcp.CallToolRequest, args siteArgs) (*mcp.CallToolResult, any, error) {
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	w, err := c.VerifyWhoami(cctx, ep.BaseURL)
	if err != nil {
		return toolError(err)
	}
	return textResult(map[string]any{
		"endpoint": publicEP(ep),
		"whoami":   w,
	})
}

func (s *Server) toolListArticles(ctx context.Context, _ *mcp.CallToolRequest, args listArticlesArgs) (*mcp.CallToolResult, any, error) {
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}
	res, err := c.ListArticles(cctx, agentcli.ListArticlesQuery{
		Page: args.Page, PageSize: args.PageSize, Status: args.Status, Q: args.Q,
	})
	if err != nil {
		return toolError(err)
	}
	var items []map[string]any
	if len(res.Items) > 0 {
		if err := json.Unmarshal(res.Items, &items); err != nil {
			return toolError(err)
		}
	}
	if args.MissingSEO {
		filtered := items[:0]
		for _, it := range items {
			if agentcli.ArticleMissingSEO(it) {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return textResult(map[string]any{
		"siteId":   ep.SiteID,
		"baseUrl":  ep.BaseURL,
		"total":    res.Total,
		"page":     res.Page,
		"pageSize": res.PageSize,
		"items":    items,
		"filter":   map[string]any{"missingSeo": args.MissingSEO, "status": args.Status, "q": args.Q},
	})
}

func (s *Server) toolGetArticle(ctx context.Context, _ *mcp.CallToolRequest, args getArticleArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == 0 {
		return toolError(fmt.Errorf("id is required"))
	}
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}
	art, err := c.GetArticle(cctx, args.ID)
	if err != nil {
		return toolError(err)
	}
	return textResult(map[string]any{
		"siteId":  ep.SiteID,
		"baseUrl": ep.BaseURL,
		"article": art,
	})
}

func (s *Server) toolApplyArticlePatch(ctx context.Context, _ *mcp.CallToolRequest, args applyArticleArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == 0 {
		return toolError(fmt.Errorf("id is required"))
	}
	dryRun := true
	if args.DryRun != nil {
		dryRun = *args.DryRun
	}

	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}

	var body map[string]any
	if !dryRun && args.PreviewHandle != "" {
		body, err = s.previews.Take(args.PreviewHandle, ep.SiteID, args.ID)
		if err != nil {
			return toolError(err)
		}
	} else {
		if len(args.Patch) == 0 && args.PreviewHandle == "" {
			return toolError(fmt.Errorf("patch is required for dry_run (or provide preview_handle to commit)"))
		}
		cur, err := c.GetArticle(cctx, args.ID)
		if err != nil {
			return toolError(err)
		}
		patch := args.Patch
		if patch == nil {
			patch = map[string]any{}
		}
		body = agentcli.MergeArticleUpdate(cur, patch)
	}

	if dryRun {
		handle, err := s.previews.Put(ep.SiteID, args.ID, body)
		if err != nil {
			return toolError(err)
		}
		return textResult(map[string]any{
			"dryRun":         true,
			"previewHandle":  handle,
			"siteId":         ep.SiteID,
			"articleId":      args.ID,
			"mergedBody":     body,
			"commitHint":     "Call apply_article_patch again with dry_run=false and this preview_handle to write.",
		})
	}

	res, err := c.PutArticle(cctx, args.ID, body)
	if err != nil {
		return toolError(err)
	}
	return textResult(map[string]any{
		"dryRun":    false,
		"siteId":    ep.SiteID,
		"articleId": args.ID,
		"article":   res,
	})
}
