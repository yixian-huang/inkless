package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin management commands",
		Long:  "Create and manage Impress CMS plugins.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create [name]",
		Short: "Generate a new plugin project from template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("impress plugin create %s: not yet implemented\n", args[0])
			return nil
		},
	})

	return cmd
}
