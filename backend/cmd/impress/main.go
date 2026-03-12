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
		Use:     "impress",
		Short:   "Impress CMS CLI - manage your bilingual CMS instance",
		Long:    "impress is a command-line tool for managing Impress CMS.\nIt provides commands for initialization, migration, seeding, and data management.",
		Version: Version,
	}

	// Register subcommands
	cmd.AddCommand(initCmd())
	cmd.AddCommand(serveCmd())
	cmd.AddCommand(migrateCmd())
	cmd.AddCommand(seedCmd())
	cmd.AddCommand(exportCmd())
	cmd.AddCommand(importCmd())
	cmd.AddCommand(pluginCmd())

	return cmd
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
