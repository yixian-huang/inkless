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
  inkless content versions home
  inkless content rollback home 3

MediaRef leaves (url/alt/caption) must be plain strings.`,
		SilenceUsage: true,
	}
	cmd.AddCommand(contentGetCmd())
	cmd.AddCommand(contentApplyCmd())
	cmd.AddCommand(contentValidateCmd())
	cmd.AddCommand(contentPublishCmd())
	cmd.AddCommand(contentVersionsCmd())
	cmd.AddCommand(contentVersionCmd())
	cmd.AddCommand(contentRollbackCmd())
	cmd.AddCommand(contentKeysCmd())
	cmd.AddCommand(contentSlotsCmd())
	cmd.AddCommand(contentSchemaCmd())
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
	var dryRun, validateSchema, noSchema bool
	cmd := &cobra.Command{
		Use:   "apply <pageKey>",
		Short: "PUT draft from file (If-Match from current version); supports --dry-run diff",
		Long: `Replace the theme content draft with the JSON file body.

File may be either:
  { "hero": {...}, "features": {...} }           # bare config
  { "config": { ... }, "changeNote": "..." }     # wrapped

Always uses optimistic locking (If-Match = current draft version).
Use --dry-run for deep path diff + local MediaRef preflight + server validate.
--validate-schema requires theme contentSlots (schemaSource=theme) and valid=true.
--no-schema skips local MediaRef preflight block (still sends server validate unless dry-run only).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if validateSchema && noSchema {
				return fmt.Errorf("--validate-schema and --no-schema are mutually exclusive")
			}
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
			diff := agentcli.DeepConfigDiff(curCfg, config)
			localMedia := agentcli.LocalMediaRefIssues(config)

			val, valErr := c.ValidateContent(ctx, pageKey, config)
			if dryRun {
				out := map[string]any{
					"dryRun":           true,
					"siteId":           ep.SiteID,
					"pageKey":          pageKey,
					"currentVersion":   ver,
					"diff":             diff,
					"localMediaIssues": localMedia,
					"config":           config,
					"changeNote":       changeNote,
					"validate":         val,
					"validateError":    nil,
				}
				if valErr != nil {
					out["validateError"] = valErr.Error()
				}
				if err := f.printJSON(out); err != nil {
					return err
				}
				if validateSchema {
					return requireThemeSchemaValid(val, valErr)
				}
				return nil
			}

			if !noSchema && len(localMedia) > 0 {
				return fmt.Errorf("local MediaRef preflight failed (%d issue(s)); fix url/alt/caption to strings or use --dry-run", len(localMedia))
			}
			if validateSchema {
				if err := requireThemeSchemaValid(val, valErr); err != nil {
					return err
				}
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
			fmt.Printf("updated content draft %s version→%v on site %s (%s)\n",
				pageKey, res["version"], ep.SiteID, agentcli.FormatDiffHuman(diff))
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON config file (bare or {config:...})")
	cmd.Flags().StringVar(&changeNote, "change-note", "", "Optional change note")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Deep path diff + local MediaRef + server validate; no write")
	cmd.Flags().BoolVar(&validateSchema, "validate-schema", false, "Require theme contentSlots validation (schemaSource=theme, valid=true)")
	cmd.Flags().BoolVar(&noSchema, "no-schema", false, "Skip local MediaRef preflight block on write")
	return cmd
}

func requireThemeSchemaValid(val map[string]any, valErr error) error {
	if valErr != nil {
		return fmt.Errorf("--validate-schema: validate request failed: %w", valErr)
	}
	if val == nil {
		return fmt.Errorf("--validate-schema: empty validate response")
	}
	src, _ := val["schemaSource"].(string)
	if src != "theme" {
		return fmt.Errorf("--validate-schema: active theme has no contentSlots for this pageKey (schemaSource=%q)", src)
	}
	valid, _ := val["valid"].(bool)
	if !valid {
		return fmt.Errorf("--validate-schema: validation failed (schemaId=%v errors=%v)", val["schemaId"], val["errors"])
	}
	return nil
}

func contentSlotsCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "slots",
		Short: "GET /admin/content/slots (active theme contentSlots)",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			res, err := c.ListContentSlots(ctx)
			if err != nil {
				return err
			}
			return f.printJSON(res)
		},
	}
	f.addTo(cmd)
	return cmd
}

func contentSchemaCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "schema <pageKey>",
		Short: "GET /admin/content/:pageKey/schema",
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
			res, err := c.GetContentSchema(ctx, pageKey)
			if err != nil {
				return err
			}
			return f.printJSON(res)
		},
	}
	f.addTo(cmd)
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

func contentVersionsCmd() *cobra.Command {
	var f remoteFlags
	var page, pageSize int
	cmd := &cobra.Command{
		Use:   "versions <pageKey>",
		Short: "GET /admin/content/:pageKey/versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageKey := strings.TrimSpace(args[0])
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
			res, err := c.ListContentVersions(ctx, pageKey, page, pageSize)
			if err != nil {
				return err
			}
			return f.printJSON(res)
		},
	}
	f.addTo(cmd)
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func contentVersionCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "version <pageKey> <version>",
		Short: "GET /admin/content/:pageKey/versions/:version",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageKey := strings.TrimSpace(args[0])
			ver, err := parseUintArg(args[1])
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
			res, err := c.GetContentVersion(ctx, pageKey, int(ver))
			if err != nil {
				return err
			}
			return f.printJSON(res)
		},
	}
	f.addTo(cmd)
	return cmd
}

func contentRollbackCmd() *cobra.Command {
	var f remoteFlags
	var force bool
	var changeNote string
	cmd := &cobra.Command{
		Use:   "rollback <pageKey> <sourceVersion>",
		Short: "POST rollback published content to a historical version (honors publish_policy)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageKey := strings.TrimSpace(args[0])
			src, err := parseUintArg(args[1])
			if err != nil {
				return err
			}
			ep, err := f.resolve()
			if err != nil {
				return err
			}
			// Rollback mutates published state — same gate as publish.
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
			res, err := c.RollbackContent(ctx, pageKey, int(src), changeNote)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			fmt.Printf("rolled back content %s → publishedVersion=%v (source=%v) on site %s\n",
				pageKey, res["publishedVersion"], res["sourceVersion"], ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	cmd.Flags().BoolVar(&force, "force", false, "Required when publish_policy=manual")
	cmd.Flags().StringVar(&changeNote, "change-note", "", "Optional change note")
	return cmd
}

func contentKeysCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "List theme content pageKeys from whoami (activeTheme + contentSlots)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ep, err := f.resolve()
			if err != nil {
				return err
			}
			ctx, cancel := f.context()
			defer cancel()
			c := f.client(ep)
			w, err := c.GetWhoami(ctx)
			if err != nil {
				return err
			}
			caps := w.Capabilities
			themeContent, _ := caps["themeContent"].(bool)
			var keys []any
			if raw, ok := caps["themeContentKeys"].([]any); ok {
				keys = raw
			}
			var slots []any
			if raw, ok := caps["contentSlots"].([]any); ok {
				slots = raw
			}
			out := map[string]any{
				"siteId":               ep.SiteID,
				"baseUrl":              w.BaseURL,
				"themeContent":         themeContent,
				"themeContentKeys":     keys,
				"activeThemeId":        caps["activeThemeId"],
				"activeThemeVersion":   caps["activeThemeVersion"],
				"contentSlots":         slots,
			}
			return f.printJSON(out)
		},
	}
	f.addTo(cmd)
	return cmd
}
