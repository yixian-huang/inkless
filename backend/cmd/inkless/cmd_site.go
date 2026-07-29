package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yixian-huang/inkless/backend/internal/agentcli"
)

func siteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "Multi-site fleet helpers for remote Admin API agents",
		Long: `Manage and probe Inkless instances listed in a local fleet registry.

Fleet schema: docs/agent-fleet.schema.json
Access guide: docs/agent-access.md

Single-site mode (no fleet): set INKLESS_BASE_URL + INKLESS_API_KEY.`,
		SilenceUsage: true,
	}
	cmd.AddCommand(siteListCmd())
	cmd.AddCommand(siteResolveCmd())
	cmd.AddCommand(siteWhoamiCmd())
	return cmd
}

func siteListCmd() *cobra.Command {
	var fleetPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sites in the fleet registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, fleet, err := agentcli.ListSites(fleetPath)
			if err != nil {
				return err
			}
			type row struct {
				ID            string   `json:"id"`
				Label         string   `json:"label"`
				BaseURL       string   `json:"baseUrl"`
				APIKeyEnv     string   `json:"apiKeyEnv,omitempty"`
				APIKeyFile    string   `json:"apiKeyFile,omitempty"`
				PublishPolicy string   `json:"publishPolicy"`
				Brand         string   `json:"brand,omitempty"`
				Default       bool     `json:"default"`
				Scopes        []string `json:"scopesExpected,omitempty"`
			}
			ids := make([]string, 0, len(fleet.Sites))
			for id := range fleet.Sites {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			rows := make([]row, 0, len(ids))
			for _, id := range ids {
				s := fleet.Sites[id]
				policy := s.PublishPolicy
				if policy == "" {
					policy = "manual"
				}
				rows = append(rows, row{
					ID:            id,
					Label:         s.Label,
					BaseURL:       agentcli.NormalizeBaseURL(s.BaseURL),
					APIKeyEnv:     s.APIKeyEnv,
					APIKeyFile:    s.APIKeyFile,
					PublishPolicy: policy,
					Brand:         s.Brand,
					Default:       id == fleet.DefaultSite,
					Scopes:        s.ScopesExpected,
				})
			}
			if jsonOut {
				enc := map[string]any{"fleet": path, "defaultSite": fleet.DefaultSite, "sites": rows}
				return printJSONValue(enc)
			}
			fmt.Fprintf(os.Stderr, "fleet: %s\n", path)
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tLABEL\tBASE_URL\tPOLICY\tKEY\tDEFAULT")
			for _, r := range rows {
				keySrc := r.APIKeyEnv
				if keySrc == "" {
					keySrc = r.APIKeyFile
				}
				def := ""
				if r.Default {
					def = "*"
				}
				label := r.Label
				if label == "" {
					label = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", r.ID, label, r.BaseURL, r.PublishPolicy, keySrc, def)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&fleetPath, "fleet", "", "Path to fleet.json")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "JSON output")
	return cmd
}

func siteResolveCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve site endpoint (masks API key)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ep, err := f.resolve()
			if err != nil {
				return err
			}
			out := map[string]any{
				"siteId":        ep.SiteID,
				"label":         ep.Label,
				"baseUrl":       ep.BaseURL,
				"apiKeyMasked":  agentcli.MaskSecret(ep.APIKey),
				"publishPolicy": ep.PublishPolicy,
				"verifyWhoami":  ep.VerifyWhoami,
				"scopesExpect":  ep.ScopesExpect,
			}
			if f.jsonOut {
				return f.printJSON(out)
			}
			fmt.Printf("siteId:        %s\n", ep.SiteID)
			fmt.Printf("label:         %s\n", ep.Label)
			fmt.Printf("baseUrl:       %s\n", ep.BaseURL)
			fmt.Printf("apiKey:        %s\n", agentcli.MaskSecret(ep.APIKey))
			fmt.Printf("publishPolicy: %s\n", ep.PublishPolicy)
			fmt.Printf("verifyWhoami:  %v\n", ep.VerifyWhoami)
			if len(ep.ScopesExpect) > 0 {
				fmt.Printf("scopesExpect:  %s\n", strings.Join(ep.ScopesExpect, ", "))
			}
			return nil
		},
	}
	f.addTo(cmd)
	return cmd
}

func siteWhoamiCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Probe GET /admin/agent/whoami on the resolved site",
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
			w, err := c.VerifyWhoami(ctx, ep.BaseURL)
			if err != nil {
				// still print raw whoami if available in error chain — VerifyWhoami returns partial
				return err
			}
			if f.jsonOut {
				return f.printJSON(w)
			}
			fmt.Printf("baseUrl:     %s\n", w.BaseURL)
			fmt.Printf("version:     %s\n", w.Version)
			fmt.Printf("authMethod:  %s\n", w.AuthMethod)
			if w.APIKeyID != nil {
				fmt.Printf("apiKeyId:    %d\n", *w.APIKeyID)
			}
			fmt.Printf("user:        %s (id=%d role=%s)\n", w.User.Username, w.User.ID, w.User.Role)
			if len(w.Scopes) > 0 {
				fmt.Printf("scopes:      %s\n", strings.Join(w.Scopes, ", "))
			}
			if w.Capabilities != nil {
				raw, _ := jsonMarshalIndent(w.Capabilities)
				fmt.Printf("capabilities:\n%s\n", string(raw))
			}
			return nil
		},
	}
	f.addTo(cmd)
	return cmd
}

func printJSONValue(v any) error {
	return jsonNewEncoder(os.Stdout)(v)
}
