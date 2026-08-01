package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yixian-huang/inkless/backend/internal/agentcli"
)

func contentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Theme-bound content_documents Admin API (home, etc.)",
		Long: `Read/write theme page content slots (content_documents), not unified /pages.

Typical product-first flow:
  inkless content get home --site SITE
  inkless content apply home --from-file home.json --dry-run
  inkless content apply home --from-file home.json
  inkless content publish home   # honors publish_policy

MediaRef leaves (url/alt/caption) must be plain strings.`,
		SilenceUsage: true,
	}
	cmd.AddCommand(contentGetCmd())
	cmd.AddCommand(contentApplyCmd())
	cmd.AddCommand(contentValidateCmd())
	cmd.AddCommand(contentPublishCmd())
	return cmd
}

func contentGetCmd() *cobra.Command {
	var f remoteFlags
	var public bool
	var locale string
	cmd := &cobra.Command{
		Use:   "get <pageKey>",
		Short: "GET draft (default) or --public /public/content/:pageKey",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageKey := strings.TrimSpace(args[0])
			if pageKey == "" {
				return fmt.Errorf("pageKey is required")
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
			var out map[string]any
			if public {
				out, err = c.GetPublicContent(ctx, pageKey, locale)
			} else {
				out, err = c.GetContentDraft(ctx, pageKey)
			}
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(map[string]any{
					"siteId":  ep.SiteID,
					"pageKey": pageKey,
					"public":  public,
					"data":    out,
				})
			}
			return f.printJSON(out)
		},
	}
	f.addTo(cmd)
	cmd.Flags().BoolVar(&public, "public", false, "Read published config via GET /public/content/:pageKey")
	cmd.Flags().StringVar(&locale, "locale", "zh", "Locale for --public")
	return cmd
}

func contentApplyCmd() *cobra.Command {
	var f remoteFlags
	var fromFile, changeNote string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply <pageKey>",
		Short: "PUT draft from file (If-Match from current version); supports --dry-run diff",
		Long: `Replace the theme content draft with the JSON file body.

File may be either:
  { "hero": {...}, "features": {...} }           # bare config
  { "config": { ... }, "changeNote": "..." }     # wrapped

Always uses optimistic locking (If-Match = current draft version).
Use --dry-run to print shallow key diff without writing.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageKey := strings.TrimSpace(args[0])
			if pageKey == "" {
				return fmt.Errorf("pageKey is required")
			}
			if strings.TrimSpace(fromFile) == "" {
				return fmt.Errorf("--from-file is required")
			}
			rawBytes, err := os.ReadFile(fromFile)
			if err != nil {
				return err
			}
			var raw map[string]any
			if err := json.Unmarshal(rawBytes, &raw); err != nil {
				return fmt.Errorf("parse config file: %w", err)
			}
			config, err := agentcli.ContentConfigFromFileBody(raw)
			if err != nil {
				return err
			}
			if note, ok := raw["changeNote"].(string); ok && changeNote == "" {
				changeNote = note
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

			draft, err := c.GetContentDraft(ctx, pageKey)
			if err != nil {
				return err
			}
			ver := agentcli.ContentDraftVersion(draft)
			curCfg, _ := draft["config"].(map[string]any)
			if curCfg == nil {
				curCfg = map[string]any{}
			}
			diff := agentcli.ShallowConfigDiff(curCfg, config)

			if dryRun {
				// Also run server validate when possible for MediaRef errors.
				val, valErr := c.ValidateContent(ctx, pageKey, config)
				out := map[string]any{
					"dryRun":          true,
					"siteId":          ep.SiteID,
					"pageKey":         pageKey,
					"currentVersion":  ver,
					"diff":            diff,
					"config":          config,
					"changeNote":      changeNote,
					"validate":        val,
					"validateError":   nil,
				}
				if valErr != nil {
					out["validateError"] = valErr.Error()
				}
				return f.printJSON(out)
			}

			res, err := c.PutContentDraft(ctx, pageKey, ver, config, changeNote)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(map[string]any{
					"siteId":  ep.SiteID,
					"pageKey": pageKey,
					"diff":    diff,
					"result":  res,
				})
			}
			fmt.Printf("updated content draft %s version→%v on site %s\n", pageKey, res["version"], ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON config file (bare or {config:...})")
	cmd.Flags().StringVar(&changeNote, "change-note", "", "Optional change note")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show diff + validate without writing")
	return cmd
}

func contentValidateCmd() *cobra.Command {
	var f remoteFlags
	var fromFile string
	cmd := &cobra.Command{
		Use:   "validate <pageKey>",
		Short: "POST /admin/content/:pageKey/validate from file or current draft",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageKey := strings.TrimSpace(args[0])
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
			var config map[string]any
			if strings.TrimSpace(fromFile) != "" {
				rawBytes, err := os.ReadFile(fromFile)
				if err != nil {
					return err
				}
				var raw map[string]any
				if err := json.Unmarshal(rawBytes, &raw); err != nil {
					return err
				}
				config, err = agentcli.ContentConfigFromFileBody(raw)
				if err != nil {
					return err
				}
			} else {
				draft, err := c.GetContentDraft(ctx, pageKey)
				if err != nil {
					return err
				}
				config, _ = draft["config"].(map[string]any)
				if config == nil {
					config = map[string]any{}
				}
			}
			res, err := c.ValidateContent(ctx, pageKey, config)
			if err != nil {
				return err
			}
			return f.printJSON(res)
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Optional JSON config; default = current draft")
	return cmd
}

func contentPublishCmd() *cobra.Command {
	var f remoteFlags
	var force bool
	var changeNote string
	cmd := &cobra.Command{
		Use:   "publish <pageKey>",
		Short: "POST publish current draft (honors publish_policy)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageKey := strings.TrimSpace(args[0])
			ep, err := f.resolve()
			if err != nil {
				return err
			}
			if err := guardPublish(ep, force); err != nil {
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
			draft, err := c.GetContentDraft(ctx, pageKey)
			if err != nil {
				return err
			}
			ver := agentcli.ContentDraftVersion(draft)
			res, err := c.PublishContent(ctx, pageKey, ver, changeNote)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			fmt.Printf("published content %s publishedVersion=%v on site %s\n",
				pageKey, res["publishedVersion"], ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().BoolVar(&force, "force", false, "Required when publish_policy=manual")
	cmd.Flags().StringVar(&changeNote, "change-note", "", "Optional change note")
	return cmd
}
