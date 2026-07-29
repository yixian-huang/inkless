package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/yixian-huang/inkless/backend/internal/inklessmcp"
)

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server for local multi-site content agents",
		Long: `Run an MCP server that exposes Inkless Admin API operations to host agents.

Targets MCP specification 2026-07-28 (stateless protocol core). Application
state is limited to short-TTL preview handles for dry-run apply.

See docs/design-inkless-mcp.md and docs/agent-mcp.md.`,
		SilenceUsage: true,
	}
	cmd.AddCommand(mcpServeCmd())
	return cmd
}

func mcpServeCmd() *cobra.Command {
	var (
		fleetPath string
		siteID    string
		baseURL   string
		apiKey    string
		timeout   time.Duration
		noVerify  bool
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve MCP over stdio (for Cursor / Claude Code / etc.)",
		Long: `Start the Inkless MCP server on stdin/stdout.

Configure the host with something like:

  {
    "mcpServers": {
      "inkless": {
        "command": "inkless",
        "args": ["mcp", "serve", "--fleet", "/path/to/fleet.json"],
        "env": {
          "INKLESS_KEY_OPS": "ink_…"
        }
      }
    }
  }

Single-site without fleet: set INKLESS_BASE_URL + INKLESS_API_KEY.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := inklessmcp.New(inklessmcp.Options{
				FleetPath: fleetPath,
				SiteID:    siteID,
				BaseURL:   baseURL,
				APIKey:    apiKey,
				Timeout:   timeout,
				NoVerify:  noVerify,
				Version:   Version,
			})
			// Errors go to stderr via host; avoid writing logs to stdout (stdio MCP).
			if err := srv.RunStdio(cmd.Context()); err != nil {
				return fmt.Errorf("mcp serve: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&fleetPath, "fleet", "", "Path to fleet.json (or INKLESS_FLEET)")
	cmd.Flags().StringVar(&siteID, "site", "", "Default site id (or INKLESS_SITE)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Single-site base URL (or INKLESS_BASE_URL)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (prefer env INKLESS_API_KEY)")
	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "HTTP timeout for Admin API calls")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip whoami baseUrl verification")
	return cmd
}
