// Package inklessmcp implements a local MCP server for multi-site Inkless
// content agents (MCP 2026-07-28, application-stateless).
package inklessmcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yixian-huang/inkless/backend/internal/agentcli"
)

// Options configures the MCP server process.
type Options struct {
	FleetPath string
	SiteID    string // optional default site
	BaseURL   string // single-site override
	APIKey    string
	Timeout   time.Duration
	NoVerify  bool
	Version   string
}

// Server wraps MCP registration and agentcli access.
type Server struct {
	opts     Options
	previews *previewStore
}

// New creates an Inkless MCP application server (not yet listening).
func New(opts Options) *Server {
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	return &Server{
		opts:     opts,
		previews: newPreviewStore(15 * time.Minute),
	}
}

// MCP builds and returns a configured mcp.Server with tools registered.
func (s *Server) MCP() *mcp.Server {
	impl := &mcp.Implementation{
		Name:        "inkless",
		Title:       "Inkless CMS Agent",
		Description: "Multi-site content maintenance via Admin API (fleet + API keys). Stateless MCP 2026-07-28.",
		Version:     s.opts.Version,
	}
	srv := mcp.NewServer(impl, &mcp.ServerOptions{
		Instructions: `Inkless multi-site content agent.

Rules:
- Prefer list_sites then pass site_id on every call (or use default_site / single-site env).
- Never invent base_url; resolve from fleet or INKLESS_BASE_URL.
- apply_article_patch defaults to dry_run=true; commit with dry_run=false and preview_handle or patch.
- Do not publish unless the user explicitly asks; honor publish_policy.
- Prefer Admin API semantics; never suggest writing the database directly.`,
		// Empty capabilities disables the historical default logging capability
		// (deprecated in MCP 2026-07-28). tools/* are inferred when registered.
		Capabilities: &mcp.ServerCapabilities{},
	})
	s.registerTools(srv)
	return srv
}

// RunStdio serves MCP over stdin/stdout until the client disconnects.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.MCP().Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) resolve(siteID string) (*agentcli.Endpoint, error) {
	sid := siteID
	if sid == "" {
		sid = s.opts.SiteID
	}
	return agentcli.ResolveEndpoint(agentcli.ResolveOptions{
		FleetPath: s.opts.FleetPath,
		SiteID:    sid,
		BaseURL:   s.opts.BaseURL,
		APIKey:    s.opts.APIKey,
	})
}

func (s *Server) client(ep *agentcli.Endpoint) *agentcli.Client {
	c := agentcli.NewClient(ep)
	c.HTTPClient.Timeout = s.opts.Timeout
	return c
}

func (s *Server) verify(ctx context.Context, ep *agentcli.Endpoint, c *agentcli.Client) error {
	if s.opts.NoVerify || !ep.VerifyWhoami {
		return nil
	}
	_, err := c.VerifyWhoami(ctx, ep.BaseURL)
	return err
}

func (s *Server) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.opts.Timeout)
}

func boolPtr(v bool) *bool { return &v }

func textResult(v any) (*mcp.CallToolResult, any, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}},
	}, v, nil
}

func toolError(err error) (*mcp.CallToolResult, any, error) {
	// ToolHandlerFor treats returned error as tool error (IsError content).
	return nil, nil, err
}

// endpointPublic is a safe view of Endpoint for tool output.
type endpointPublic struct {
	SiteID        string   `json:"siteId"`
	Label         string   `json:"label"`
	BaseURL       string   `json:"baseUrl"`
	APIKeyMasked  string   `json:"apiKeyMasked"`
	PublishPolicy string   `json:"publishPolicy"`
	VerifyWhoami  bool     `json:"verifyWhoami"`
	ScopesExpect  []string `json:"scopesExpect,omitempty"`
}

func publicEP(ep *agentcli.Endpoint) endpointPublic {
	return endpointPublic{
		SiteID:        ep.SiteID,
		Label:         ep.Label,
		BaseURL:       ep.BaseURL,
		APIKeyMasked:  agentcli.MaskSecret(ep.APIKey),
		PublishPolicy: ep.PublishPolicy,
		VerifyWhoami:  ep.VerifyWhoami,
		ScopesExpect:  ep.ScopesExpect,
	}
}
