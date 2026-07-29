package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
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
