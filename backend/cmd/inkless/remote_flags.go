package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/yixian-huang/inkless/backend/internal/agentcli"
)

// remoteFlags are shared by site / articles / pages remote commands.
type remoteFlags struct {
	fleetPath string
	siteID    string
	baseURL   string
	apiKey    string
	jsonOut   bool
	noVerify  bool
	timeout   time.Duration
}

func (f *remoteFlags) addTo(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.fleetPath, "fleet", "", "Path to fleet.json (default: INKLESS_FLEET or ~/.config/inkless/fleet.json)")
	cmd.Flags().StringVar(&f.siteID, "site", "", "Site id from fleet (or INKLESS_SITE)")
	cmd.Flags().StringVar(&f.baseURL, "base-url", "", "Single-site base URL (or INKLESS_BASE_URL)")
	cmd.Flags().StringVar(&f.apiKey, "api-key", "", "API key ink_… (or INKLESS_API_KEY); prefer env")
	cmd.Flags().BoolVar(&f.jsonOut, "json", false, "Machine-readable JSON output")
	cmd.Flags().BoolVar(&f.noVerify, "no-verify", false, "Skip whoami baseUrl check before writes")
	cmd.Flags().DurationVar(&f.timeout, "timeout", 60*time.Second, "HTTP timeout")
}

func (f *remoteFlags) resolve() (*agentcli.Endpoint, error) {
	return agentcli.ResolveEndpoint(agentcli.ResolveOptions{
		FleetPath: f.fleetPath,
		SiteID:    f.siteID,
		BaseURL:   f.baseURL,
		APIKey:    f.apiKey,
	})
}

func (f *remoteFlags) client(ep *agentcli.Endpoint) *agentcli.Client {
	c := agentcli.NewClient(ep)
	c.HTTPClient.Timeout = f.timeout
	return c
}

func (f *remoteFlags) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), f.timeout)
}

func (f *remoteFlags) printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// verifyIfNeeded runs whoami baseUrl check unless --no-verify or site disables it.
func (f *remoteFlags) verifyIfNeeded(ctx context.Context, ep *agentcli.Endpoint, c *agentcli.Client) error {
	if f.noVerify || !ep.VerifyWhoami {
		return nil
	}
	_, err := c.VerifyWhoami(ctx, ep.BaseURL)
	return err
}

func printEndpointSummary(ep *agentcli.Endpoint) {
	fmt.Fprintf(os.Stderr, "site=%s label=%q base=%s key=%s policy=%s\n",
		ep.SiteID, ep.Label, ep.BaseURL, agentcli.MaskSecret(ep.APIKey), ep.PublishPolicy)
}

func parseUintArg(s string) (uint, error) {
	var id uint64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid id %q", s)
	}
	return uint(id), nil
}

func guardPublish(ep *agentcli.Endpoint, force bool) error {
	switch ep.PublishPolicy {
	case "never":
		return fmt.Errorf("publish_policy=never for site %s", ep.SiteID)
	case "manual":
		if !force {
			return fmt.Errorf("publish_policy=manual: pass --force to publish on site %s", ep.SiteID)
		}
	}
	return nil
}
