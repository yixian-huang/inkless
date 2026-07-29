package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inkless",
		Short: "Inkless CMS CLI - manage instances and remote content agents",
		Long: `inkless is a command-line tool for managing Inkless CMS.

Local instance: init, serve, migrate, seed, export/import, plugin.
Remote multi-site agents: site, articles, pages (API Key + fleet registry).
See docs/agent-access.md.`,
		Version: Version,
	}

	// Local instance management
	cmd.AddCommand(initCmd())
	cmd.AddCommand(serveCmd())
	cmd.AddCommand(migrateCmd())
	cmd.AddCommand(seedCmd())
	cmd.AddCommand(exportCmd())
	cmd.AddCommand(importCmd())
	cmd.AddCommand(pluginCmd())

	// Remote content agent (Admin API)
	cmd.AddCommand(siteCmd())
	cmd.AddCommand(articlesCmd())
	cmd.AddCommand(pagesCmd())

	return cmd
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
