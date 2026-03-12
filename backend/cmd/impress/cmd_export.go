package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"blotting-consultancy/internal/model"
)

type siteExport struct {
	ExportedAt      string                  `json:"exportedAt"`
	Version         string                  `json:"version"`
	Articles        []model.Article         `json:"articles"`
	Pages           []model.Page            `json:"pages"`
	Categories      []model.Category        `json:"categories"`
	Tags            []model.Tag             `json:"tags"`
	InstalledThemes []model.InstalledTheme   `json:"installedThemes"`
	ContentDocs     []model.ContentDocument `json:"contentDocuments"`
}

func exportCmd() *cobra.Command {
	var dsn, outputFile string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export site data to JSON",
		Long:  "Export articles, pages, categories, tags, themes, and content documents to a JSON file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := openDatabase(dsn)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			export := siteExport{
				ExportedAt: time.Now().UTC().Format(time.RFC3339),
				Version:    Version,
			}

			database.DB.Find(&export.Articles)
			database.DB.Find(&export.Pages)
			database.DB.Find(&export.Categories)
			database.DB.Find(&export.Tags)
			database.DB.Find(&export.InstalledThemes)
			database.DB.Find(&export.ContentDocs)

			data, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal export: %w", err)
			}

			if outputFile == "" {
				outputFile = fmt.Sprintf("impress-export-%s.json", time.Now().Format("20060102-150405"))
			}

			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			fmt.Printf("Exported to %s (%d articles, %d pages, %d categories, %d tags, %d themes)\n",
				outputFile,
				len(export.Articles), len(export.Pages),
				len(export.Categories), len(export.Tags),
				len(export.InstalledThemes))
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "Database DSN")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: impress-export-TIMESTAMP.json)")
	return cmd
}
