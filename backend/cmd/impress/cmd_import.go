package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func importCmd() *cobra.Command {
	var dsn, inputFile string

	cmd := &cobra.Command{
		Use:   "import [file]",
		Short: "Import site data from JSON",
		Long:  "Import site content from a previously exported JSON file. Creates new records without overwriting existing ones.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				inputFile = args[0]
			}
			if inputFile == "" {
				return fmt.Errorf("input file is required: impress import <file.json>")
			}

			data, err := os.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			var export siteExport
			if err := json.Unmarshal(data, &export); err != nil {
				return fmt.Errorf("failed to parse import file: %w", err)
			}

			database, err := openDatabase(dsn)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			var counts [4]int
			for i := range export.Articles {
				export.Articles[i].ID = 0 // reset ID for new insert
				if err := database.DB.Create(&export.Articles[i]).Error; err == nil {
					counts[0]++
				}
			}
			for i := range export.Pages {
				export.Pages[i].ID = 0
				if err := database.DB.Create(&export.Pages[i]).Error; err == nil {
					counts[1]++
				}
			}
			for i := range export.Categories {
				export.Categories[i].ID = 0
				if err := database.DB.Create(&export.Categories[i]).Error; err == nil {
					counts[2]++
				}
			}
			for i := range export.Tags {
				export.Tags[i].ID = 0
				if err := database.DB.Create(&export.Tags[i]).Error; err == nil {
					counts[3]++
				}
			}

			fmt.Printf("Imported: %d articles, %d pages, %d categories, %d tags\n",
				counts[0], counts[1], counts[2], counts[3])
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "Database DSN")
	cmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input JSON file path")
	return cmd
}
