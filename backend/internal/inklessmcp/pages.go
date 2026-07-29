package inklessmcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerPageTools(srv *mcp.Server) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(true), Title: "Read-only"}
	rw := &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(true),
		OpenWorldHint:   boolPtr(true),
		Title:           "Mutating",
	}

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_pages",
		Title:       "List pages",
		Description: "List unified pages via GET /admin/pages.",
		Annotations: ro,
	}, s.toolListPages)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_page",
		Title:       "Get page",
		Description: "GET /admin/pages/:id (page metadata / SEO fields).",
		Annotations: ro,
	}, s.toolGetPage)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_page_draft",
		Title:       "Get page draft",
		Description: "GET /admin/pages/:id/draft.",
		Annotations: ro,
	}, s.toolGetPageDraft)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "put_page_draft",
		Title: "Put page draft",
		Description: `PUT /admin/pages/:id/draft.
Default dry_run=true stores a preview_handle without writing.
Commit with dry_run=false + preview_handle, or dry_run=false + body.`,
		Annotations: rw,
	}, s.toolPutPageDraft)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "publish_page",
		Title: "Publish page",
		Description: `POST /admin/pages/:id/publish.

Respects fleet publish_policy:
- never: always rejected
- manual / allow: requires confirmation via MCP MRTR (input_required) unless force=true

MRTR: first call returns resultType input_required with elicitation "confirm";
host retries with inputResponses (accept) to complete publish.`,
		Annotations: rw,
	}, s.toolPublishPage)
}

type pageIDArgs struct {
	SiteID string `json:"site_id" jsonschema:"Fleet site id"`
	ID     uint   `json:"id" jsonschema:"Page id"`
}

type putPageDraftArgs struct {
	SiteID        string         `json:"site_id" jsonschema:"Fleet site id"`
	ID            uint           `json:"id" jsonschema:"Page id"`
	Body          map[string]any `json:"body,omitempty" jsonschema:"Draft body (config, changeNote, …)"`
	DryRun        *bool          `json:"dry_run,omitempty" jsonschema:"Default true"`
	PreviewHandle string         `json:"preview_handle,omitempty" jsonschema:"Handle from prior dry_run"`
}

type publishPageArgs struct {
	SiteID string `json:"site_id" jsonschema:"Fleet site id"`
	ID     uint   `json:"id" jsonschema:"Page id"`
	Force  bool   `json:"force,omitempty" jsonschema:"Skip MRTR when true (use carefully; required for policy=manual without host MRTR)"`
}

func (s *Server) toolListPages(ctx context.Context, _ *mcp.CallToolRequest, args siteArgs) (*mcp.CallToolResult, any, error) {
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}
	raw, err := c.ListPages(cctx)
	if err != nil {
		return toolError(err)
	}
	var items any
	if err := json.Unmarshal(raw, &items); err != nil {
		items = json.RawMessage(raw)
	}
	return textResult(map[string]any{
		"siteId":  ep.SiteID,
		"baseUrl": ep.BaseURL,
		"items":   items,
	})
}

func (s *Server) toolGetPage(ctx context.Context, _ *mcp.CallToolRequest, args pageIDArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == 0 {
		return toolError(fmt.Errorf("id is required"))
	}
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}
	page, err := c.GetPage(cctx, args.ID)
	if err != nil {
		return toolError(err)
	}
	return textResult(map[string]any{"siteId": ep.SiteID, "baseUrl": ep.BaseURL, "page": page})
}

func (s *Server) toolGetPageDraft(ctx context.Context, _ *mcp.CallToolRequest, args pageIDArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == 0 {
		return toolError(fmt.Errorf("id is required"))
	}
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}
	draft, err := c.GetPageDraft(cctx, args.ID)
	if err != nil {
		return toolError(err)
	}
	return textResult(map[string]any{"siteId": ep.SiteID, "baseUrl": ep.BaseURL, "draft": draft})
}

func (s *Server) toolPutPageDraft(ctx context.Context, _ *mcp.CallToolRequest, args putPageDraftArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == 0 {
		return toolError(fmt.Errorf("id is required"))
	}
	dryRun := true
	if args.DryRun != nil {
		dryRun = *args.DryRun
	}
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}
	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}

	var body map[string]any
	if !dryRun && args.PreviewHandle != "" {
		body, err = s.previews.Take(args.PreviewHandle, ep.SiteID, args.ID)
		if err != nil {
			return toolError(err)
		}
	} else {
		if len(args.Body) == 0 {
			return toolError(fmt.Errorf("body is required for dry_run (or provide preview_handle to commit)"))
		}
		body = args.Body
	}

	if dryRun {
		handle, err := s.previews.Put(ep.SiteID, args.ID, body)
		if err != nil {
			return toolError(err)
		}
		return textResult(map[string]any{
			"dryRun":        true,
			"previewHandle": handle,
			"siteId":        ep.SiteID,
			"pageId":        args.ID,
			"body":          body,
			"commitHint":    "Call put_page_draft with dry_run=false and preview_handle to write.",
		})
	}

	res, err := c.PutPageDraft(cctx, args.ID, body)
	if err != nil {
		return toolError(err)
	}
	return textResult(map[string]any{
		"dryRun": false,
		"siteId": ep.SiteID,
		"pageId": args.ID,
		"result": res,
	})
}

func (s *Server) toolPublishPage(ctx context.Context, req *mcp.CallToolRequest, args publishPageArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == 0 {
		return toolError(fmt.Errorf("id is required"))
	}
	ep, err := s.resolve(args.SiteID)
	if err != nil {
		return toolError(err)
	}

	policy := strings.ToLower(strings.TrimSpace(ep.PublishPolicy))
	if policy == "" {
		policy = "manual"
	}
	if policy == "never" {
		return toolError(fmt.Errorf("publish_policy=never for site %s", ep.SiteID))
	}

	// Verify requestState matches if present (MRTR retry).
	if req != nil && req.Params != nil && req.Params.RequestState != "" {
		st, err := decodePublishState(req.Params.RequestState)
		if err != nil {
			return toolError(fmt.Errorf("invalid requestState: %w", err))
		}
		if st.SiteID != ep.SiteID || st.PageID != args.ID {
			return toolError(fmt.Errorf("requestState does not match site_id/page id"))
		}
	}

	confirmed, declined := publishConfirmation(req, args.Force)
	if declined {
		return toolError(fmt.Errorf("publish declined by user"))
	}
	if !confirmed {
		// allow policy also requires confirm unless force — safer default
		state, err := encodePublishState(publishState{SiteID: ep.SiteID, PageID: args.ID})
		if err != nil {
			return toolError(err)
		}
		msg := fmt.Sprintf(
			"Confirm publish of page %d on site %q (%s)? publish_policy=%s",
			args.ID, ep.SiteID, ep.BaseURL, policy,
		)
		// Content must be empty when InputRequests is set (SDK MRTR rule).
		return &mcp.CallToolResult{
			InputRequests: mcp.InputRequestMap{
				"confirm": &mcp.ElicitParams{
					Message: msg,
					RequestedSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"confirm": map[string]any{
								"type":        "boolean",
								"description": "Set true to publish this page",
								"title":       "Confirm publish",
							},
						},
						"required": []string{"confirm"},
					},
				},
			},
			RequestState: state,
		}, nil, nil
	}

	cctx, cancel := s.ctx()
	defer cancel()
	c := s.client(ep)
	if err := s.verify(cctx, ep, c); err != nil {
		return toolError(err)
	}
	res, err := c.PublishPage(cctx, args.ID)
	if err != nil {
		return toolError(err)
	}
	return textResult(map[string]any{
		"published": true,
		"siteId":    ep.SiteID,
		"pageId":    args.ID,
		"baseUrl":   ep.BaseURL,
		"result":    res,
	})
}

type publishState struct {
	SiteID string `json:"siteId"`
	PageID uint   `json:"pageId"`
	Action string `json:"action"`
}

func encodePublishState(st publishState) (string, error) {
	if st.Action == "" {
		st.Action = "publish_page"
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodePublishState(s string) (publishState, error) {
	var st publishState
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(raw, &st); err != nil {
		return st, err
	}
	if st.Action != "" && st.Action != "publish_page" {
		return st, fmt.Errorf("unexpected action %q", st.Action)
	}
	return st, nil
}

// publishConfirmation returns whether the user confirmed publish, and whether they declined.
// force=true skips MRTR. MRTR retries carry InputResponses["confirm"] as *ElicitResult.
func publishConfirmation(req *mcp.CallToolRequest, force bool) (confirmed, declined bool) {
	if force {
		return true, false
	}
	if req == nil || req.Params == nil || len(req.Params.InputResponses) == 0 {
		return false, false
	}
	raw, ok := req.Params.InputResponses["confirm"]
	if !ok || raw == nil {
		return false, false
	}
	er, ok := raw.(*mcp.ElicitResult)
	if !ok {
		return false, false
	}
	switch strings.ToLower(er.Action) {
	case "decline", "cancel":
		return false, true
	case "accept":
		if er.Content != nil {
			if v, ok := er.Content["confirm"]; ok {
				switch t := v.(type) {
				case bool:
					if !t {
						return false, true
					}
					return true, false
				case string:
					if t == "false" || t == "0" || t == "" {
						return false, true
					}
				}
			}
		}
		return true, false
	default:
		return false, false
	}
}
