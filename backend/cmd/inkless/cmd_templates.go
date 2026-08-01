package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func templatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Discover active theme templates (theme-as-templates)",
		Long: `List and fetch theme display contracts (templates[]) for the active theme.

Preferred discovery path for agents (T4):
  inkless templates list --site SITE
  inkless templates get product-first/home@1 --site SITE

Native templates[] when present; otherwise Host projects contentSlots → page
templates + default post chrome. Operational page data still uses pages *.

Aliases for migration:
  content slots  → also embeds templatesProjection (deprecated discovery)
  content schema → may include templateKey pointing at this API`,
		SilenceUsage: true,
	}
	cmd.AddCommand(templatesListCmd())
	cmd.AddCommand(templatesGetCmd())
	return cmd
}

func templatesListCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "GET /admin/themes/active/templates",
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
			res, err := c.ListActiveTemplates(ctx)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			themeID, _ := res["activeThemeId"].(string)
			src, _ := res["source"].(string)
			fmt.Printf("activeTheme: %s  source: %s\n", themeID, src)
			if def, ok := res["defaultTemplates"].(map[string]any); ok && len(def) > 0 {
				fmt.Printf("defaults:    home=%v page=%v post=%v\n", def["home"], def["page"], def["post"])
			}
			tmpls, _ := res["templates"].([]any)
			if len(tmpls) == 0 {
				fmt.Println("(no templates)")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "KEY\tAPPLIES\tSLUG\tRENDERER\tSCHEMA\tSOURCE")
			for _, raw := range tmpls {
				t, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				hasSchema := "no"
				if b, ok := t["hasSchema"].(bool); ok && b {
					hasSchema = "yes"
				}
				fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%s\t%v\n",
					t["key"], t["appliesTo"], emptyDash(t["slug"]), t["renderer"], hasSchema, t["source"])
			}
			return w.Flush()
		},
	}
	f.addTo(cmd)
	return cmd
}

func templatesGetCmd() *cobra.Command {
	var f remoteFlags
	var keyFlag string
	cmd := &cobra.Command{
		Use:   "get [templateKey]",
		Short: "GET /admin/themes/active/template?key=… (schema + mediaRefPaths)",
		Long: `Fetch one template by key (full body including schemaInline when projected).

  inkless templates get product-first/home@1 --site SITE --json
  inkless templates get --key product-first/home@1 --site SITE`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(keyFlag)
			if len(args) == 1 {
				key = strings.TrimSpace(args[0])
			}
			if key == "" {
				return fmt.Errorf("template key is required (arg or --key)")
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
			res, err := c.GetActiveTemplate(ctx, key)
			if err != nil {
				return err
			}
			return f.printJSON(res)
		},
	}
	f.addTo(cmd)
	cmd.Flags().StringVar(&keyFlag, "key", "", "Template key (alternative to positional arg)")
	return cmd
}

func emptyDash(v any) string {
	if v == nil {
		return "-"
	}
	s := fmt.Sprint(v)
	if s == "" || s == "<nil>" {
		return "-"
	}
	return s
}
