package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yixian-huang/inkless/backend/internal/agentcli"
)

func articlesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "articles",
		Short:        "Remote article Admin API (fleet / single-site)",
		SilenceUsage: true,
	}
	cmd.AddCommand(articlesListCmd())
	cmd.AddCommand(articlesGetCmd())
	cmd.AddCommand(articlesApplyCmd())
	return cmd
}

func articlesListCmd() *cobra.Command {
	var f remoteFlags
	var page, pageSize int
	var status, q string
	var missingSEO bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List articles via GET /admin/articles",
		RunE: func(cmd *cobra.Command, args []string) error {
			ep, err := f.resolve()
			if err != nil {
				return err
			}
			if !f.jsonOut {
				printEndpointSummary(ep)
			}
			ctx, cancel := f.context()
			defer cancel()
			c := f.client(ep)
			if err := f.verifyIfNeeded(ctx, ep, c); err != nil {
				return err
			}

			res, err := c.ListArticles(ctx, agentcli.ListArticlesQuery{
				Page: page, PageSize: pageSize, Status: status, Q: q,
			})
			if err != nil {
				return err
			}
			var items []map[string]any
			if len(res.Items) > 0 {
				if err := json.Unmarshal(res.Items, &items); err != nil {
					return err
				}
			}
			if missingSEO {
				filtered := items[:0]
				for _, it := range items {
					if agentcli.ArticleMissingSEO(it) {
						filtered = append(filtered, it)
					}
				}
				items = filtered
			}
			if f.jsonOut {
				return f.printJSON(map[string]any{
					"siteId":   ep.SiteID,
					"baseUrl":  ep.BaseURL,
					"total":    res.Total,
					"page":     res.Page,
					"pageSize": res.PageSize,
					"items":    items,
					"filter":   map[string]any{"missingSeo": missingSEO, "status": status, "q": q},
				})
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tSLUG\tZH_TITLE\tSEO_GAP")
			for _, it := range items {
				gap := ""
				if agentcli.ArticleMissingSEO(it) {
					gap = "yes"
				}
				fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%s\n",
					it["id"], it["status"], it["slug"], truncate(fmt.Sprint(it["zhTitle"]), 40), gap)
			}
			_ = w.Flush()
			fmt.Fprintf(os.Stderr, "shown=%d total=%d page=%d\n", len(items), res.Total, res.Page)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	cmd.Flags().StringVar(&status, "status", "", "Status filter: draft|published|scheduled")
	cmd.Flags().StringVar(&q, "q", "", "Search query")
	cmd.Flags().BoolVar(&missingSEO, "missing-seo", false, "Only show items missing SEO title/meta")
	return cmd
}

func articlesGetCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get article JSON via GET /admin/articles/:id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseUintArg(args[0])
			if err != nil {
				return err
			}
			ep, err := f.resolve()
			if err != nil {
				return err
			}
			if !f.jsonOut {
				printEndpointSummary(ep)
			}
			ctx, cancel := f.context()
			defer cancel()
			c := f.client(ep)
			if err := f.verifyIfNeeded(ctx, ep, c); err != nil {
				return err
			}
			art, err := c.GetArticle(ctx, id)
			if err != nil {
				return err
			}
			return f.printJSON(art)
		},
	}
	f.addTo(cmd)
	return cmd
}

func articlesApplyCmd() *cobra.Command {
	var f remoteFlags
	var fromFile string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply <id>",
		Short: "GET article, merge JSON patch from file, PUT /admin/articles/:id",
		Long: `Merge a partial JSON object onto the current article and PUT the result.

Example patch file:
  {"zhSeoTitle":"…","zhMetaDescription":"…","enSeoTitle":"…","enMetaDescription":"…"}

Uses MergeArticleUpdate so unspecified writable fields keep current values.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseUintArg(args[0])
			if err != nil {
				return err
			}
			if strings.TrimSpace(fromFile) == "" {
				return fmt.Errorf("--from-file is required")
			}
			raw, err := os.ReadFile(fromFile)
			if err != nil {
				return err
			}
			var patch map[string]any
			if err := json.Unmarshal(raw, &patch); err != nil {
				return fmt.Errorf("parse patch: %w", err)
			}
			ep, err := f.resolve()
			if err != nil {
				return err
			}
			if !f.jsonOut {
				printEndpointSummary(ep)
			}
			ctx, cancel := f.context()
			defer cancel()
			c := f.client(ep)
			if err := f.verifyIfNeeded(ctx, ep, c); err != nil {
				return err
			}
			cur, err := c.GetArticle(ctx, id)
			if err != nil {
				return err
			}
			body := agentcli.MergeArticleUpdate(cur, patch)
			if dryRun {
				out := map[string]any{"dryRun": true, "id": id, "siteId": ep.SiteID, "body": body}
				return f.printJSON(out)
			}
			res, err := c.PutArticle(ctx, id, body)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			fmt.Printf("updated article %d on site %s\n", id, ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON patch file (partial article fields)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print merged PUT body without writing")
	return cmd
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
