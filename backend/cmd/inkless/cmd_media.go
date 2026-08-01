package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func mediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "media",
		Short:        "Remote media Admin API (upload)",
		SilenceUsage: true,
	}
	cmd.AddCommand(mediaUploadCmd())
	return cmd
}

func mediaUploadCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "POST /admin/media/upload (multipart field file)",
		Long: `Upload a local image/video/audio/font file. Requires media:create.

Returns the media JSON (use .url in product-first MediaRef).

Example:
  inkless media upload ./shot.png --site inkless-ops --json | jq -r .url`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := strings.TrimSpace(args[0])
			if path == "" {
				return fmt.Errorf("file path is required")
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
			res, err := c.UploadMedia(ctx, path)
			if err != nil {
				return err
			}
			if f.jsonOut {
				return f.printJSON(res)
			}
			url, _ := res["url"].(string)
			id := res["id"]
			fmt.Printf("uploaded %s → id=%v url=%s (site %s)\n", filepath.Base(path), id, url, ep.SiteID)
			return nil
		},
	}
	f.addTo(cmd)
	return cmd
}
