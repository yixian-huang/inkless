package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yixian-huang/inkless/backend/internal/pagepresets"
)

func pagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "pages",
		Short:        "Remote page Admin API (fleet / single-site)",
		SilenceUsage: true,
	}
	cmd.AddCommand(pagesListCmd())
	cmd.AddCommand(pagesGetCmd())
	cmd.AddCommand(pagesGetDraftCmd())
	cmd.AddCommand(pagesPutDraftCmd())
	cmd.AddCommand(pagesPublishCmd())
	cmd.AddCommand(pagesPresetsCmd())
	cmd.AddCommand(pagesCreateCmd())
	cmd.AddCommand(pagesApplyPresetCmd())
	return cmd
}

func pagesListCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pages via GET /admin/pages",
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
			itemsRaw, err := c.ListPages(ctx)
			if err != nil {
				return err
			}
			var items []map[string]any
			if err := json.Unmarshal(itemsRaw, &items); err != nil {
				// may already be objects
				return f.printJSON(json.RawMessage(itemsRaw))
			}
			if f.jsonOut {
				return f.printJSON(map[string]any{"siteId": ep.SiteID, "items": items})
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tSLUG\tMODE\tZH_TITLE")
			for _, it := range items {
				zh := ""
				if title, ok := it["zhTitle"].(string); ok {
					zh = title
				}
				fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%s\n",
					it["id"], it["status"], it["slug"], it["mode"], truncate(zh, 40))
			}
			return w.Flush()
		},
	}
	f.addTo(cmd)
	return cmd
}

func pagesGetCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "GET /admin/pages/:id",
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
			ctx, cancel := f.context()
			defer cancel()
			c := f.client(ep)
			if err := f.verifyIfNeeded(ctx, ep, c); err != nil {
				return err
			}
			page, err := c.GetPage(ctx, id)
			if err != nil {
				return err
			}
			return f.printJSON(page)
		},
	}
	f.addTo(cmd)
	return cmd
}

func pagesGetDraftCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "get-draft <id>",
		Short: "GET /admin/pages/:id/draft",
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
			ctx, cancel := f.context()
			defer cancel()
			c := f.client(ep)
			if err := f.verifyIfNeeded(ctx, ep, c); err != nil {
				return err
			}
			draft, err := c.GetPageDraft(ctx, id)
			if err != nil {
				return err
			}
			return f.printJSON(draft)
		},
	}
	f.addTo(cmd)
	return cmd
}

func pagesPutDraftCmd() *cobra.Command {
	var f remoteFlags
	var fromFile string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "put-draft <id>",
		Short: "PUT /admin/pages/:id/draft from JSON file",
		Args:  cobra.ExactArgs(1),
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
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err != nil {
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
			if dryRun {
				return f.printJSON(map[string]any{"dryRun": true, "id": id, "body": body})
			}
			res, err := c.PutPageDraft(ctx, id, body)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			fmt.Printf("updated draft page %d on site %s\n", id, ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON body for draft update")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print body without writing")
	return cmd
}

func pagesPublishCmd() *cobra.Command {
	var f remoteFlags
	var force bool
	cmd := &cobra.Command{
		Use:   "publish <id>",
		Short: "POST /admin/pages/:id/publish (honors publish_policy)",
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
			if err := guardPublish(ep, force); err != nil {
				return err
			}
			ctx, cancel := f.context()
			defer cancel()
			c := f.client(ep)
			if err := f.verifyIfNeeded(ctx, ep, c); err != nil {
				return err
			}
			res, err := c.PublishPage(ctx, id)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			fmt.Printf("published page %d on site %s\n", id, ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().BoolVar(&force, "force", false, "Required when publish_policy=manual")
	return cmd
}

func pagesPresetsCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "presets",
		Short: "List host page presets (doc-simple | doc-guide | landing-use-cases)",
		RunE: func(cmd *cobra.Command, args []string) error {
			items := pagepresets.All()
			if f.jsonOut {
				return f.printJSON(map[string]any{"presets": items})
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tLAYOUT\tLABEL_ZH\tDESCRIPTION_ZH")
			for _, m := range items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.ID, m.Layout, m.LabelZh, m.DescriptionZh)
			}
			return w.Flush()
		},
	}
	// fleet flags unused but keep --json consistent
	cmd.Flags().BoolVar(&f.jsonOut, "json", false, "JSON output")
	return cmd
}

func pagesCreateCmd() *cobra.Command {
	var f remoteFlags
	var slug, zhTitle, enTitle, preset, bodyFile string
	var showInNav, dryRun bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "POST /admin/pages (optionally from --preset)",
		RunE: func(cmd *cobra.Command, args []string) error {
			slug = strings.TrimSpace(slug)
			if slug == "" {
				return fmt.Errorf("--slug is required")
			}
			body := map[string]any{
				"slug":      slug,
				"zhTitle":   zhTitle,
				"enTitle":   enTitle,
				"mode":      "composable",
				"showInNav": showInNav,
			}
			if strings.TrimSpace(preset) != "" {
				id, err := pagepresets.ParseID(preset)
				if err != nil {
					return err
				}
				opts := pagepresets.Options{ZhTitle: zhTitle, EnTitle: enTitle}
				if bodyFile != "" {
					raw, err := os.ReadFile(bodyFile)
					if err != nil {
						return err
					}
					// optional: {"zhBody","enBody","zhSubtitle","enSubtitle"}
					var extra map[string]string
					if err := json.Unmarshal(raw, &extra); err != nil {
						return fmt.Errorf("body file: %w", err)
					}
					opts.ZhBody = extra["zhBody"]
					opts.EnBody = extra["enBody"]
					opts.ZhSubtitle = extra["zhSubtitle"]
					opts.EnSubtitle = extra["enSubtitle"]
				}
				cfg, err := pagepresets.Build(id, opts)
				if err != nil {
					return err
				}
				body["draftConfig"] = cfg
			} else if bodyFile != "" {
				raw, err := os.ReadFile(bodyFile)
				if err != nil {
					return err
				}
				var draft map[string]any
				if err := json.Unmarshal(raw, &draft); err != nil {
					return err
				}
				body["draftConfig"] = draft
			} else {
				body["draftConfig"] = map[string]any{"sections": []any{}}
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
			if dryRun {
				return f.printJSON(map[string]any{"dryRun": true, "body": body})
			}
			res, err := c.CreatePage(ctx, body)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			fmt.Printf("created page id=%v slug=%s on site %s\n", res["id"], slug, ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&slug, "slug", "", "Public slug (without /p/ prefix)")
	cmd.Flags().StringVar(&zhTitle, "zh-title", "", "Chinese title")
	cmd.Flags().StringVar(&enTitle, "en-title", "", "English title")
	cmd.Flags().StringVar(&preset, "preset", "", "Host preset: doc-simple|doc-guide|landing-use-cases")
	cmd.Flags().StringVar(&bodyFile, "from-file", "", "Optional JSON: full draftConfig, or body fields when using --preset")
	cmd.Flags().BoolVar(&showInNav, "show-in-nav", false, "Show in navigation")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print body without writing")
	return cmd
}

func pagesApplyPresetCmd() *cobra.Command {
	var f remoteFlags
	var preset, zhTitle, enTitle, bodyFile string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply-preset <id>",
		Short: "Replace draft config with a host preset (PUT draft)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idNum, err := parseUintArg(args[0])
			if err != nil {
				return err
			}
			pid, err := pagepresets.ParseID(preset)
			if err != nil {
				return err
			}
			opts := pagepresets.Options{ZhTitle: zhTitle, EnTitle: enTitle}
			if bodyFile != "" {
				raw, err := os.ReadFile(bodyFile)
				if err != nil {
					return err
				}
				var extra map[string]string
				if err := json.Unmarshal(raw, &extra); err != nil {
					return err
				}
				opts.ZhBody = extra["zhBody"]
				opts.EnBody = extra["enBody"]
				opts.ZhSubtitle = extra["zhSubtitle"]
				opts.EnSubtitle = extra["enSubtitle"]
			}
			cfg, err := pagepresets.Build(pid, opts)
			if err != nil {
				return err
			}
			// AdminUpdateDraft expects { draftConfig: ... }
			body := map[string]any{"draftConfig": cfg}

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
			if dryRun {
				return f.printJSON(map[string]any{"dryRun": true, "id": idNum, "body": body})
			}
			res, err := c.PutPageDraft(ctx, idNum, body)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			fmt.Printf("applied preset %s to page %d on site %s (draft only; publish separately)\n", preset, idNum, ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&preset, "preset", "", "Host preset id (required)")
	cmd.Flags().StringVar(&zhTitle, "zh-title", "", "Chinese title for hero/body placeholders")
	cmd.Flags().StringVar(&enTitle, "en-title", "", "English title for hero/body placeholders")
	cmd.Flags().StringVar(&bodyFile, "from-file", "", "Optional JSON with zhBody/enBody/…")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print body without writing")
	_ = cmd.MarkFlagRequired("preset")
	return cmd
}
