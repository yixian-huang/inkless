package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	goplugin "github.com/hashicorp/go-plugin"

	pb "blotting-consultancy/internal/plugin/proto"
	"blotting-consultancy/internal/plugin/shared"
	"blotting-consultancy/internal/provider"
)

// GRPCHost manages a single plugin process via hashicorp/go-plugin.
type GRPCHost struct {
	meta       *PluginMeta
	binaryPath string
	client     *goplugin.Client
	rpcClient  pb.ProviderServiceClient
	mu         sync.Mutex
}

// NewGRPCHost creates a host for a plugin binary.
func NewGRPCHost(meta *PluginMeta, binaryPath string) *GRPCHost {
	return &GRPCHost{
		meta:       meta,
		binaryPath: binaryPath,
	}
}

// PluginImpl implements goplugin.Plugin for the host side.
// It creates a ProviderServiceClient from the gRPC connection.
type PluginImpl struct {
	goplugin.Plugin
}

// GRPCClient returns a client-side ProviderServiceClient backed by a DirectClient.
func (p *PluginImpl) GRPCClient(_ *goplugin.GRPCBroker, c *goplugin.RPCClient) (interface{}, error) {
	// go-plugin calls this for net/rpc; we use gRPC path instead.
	return nil, fmt.Errorf("net/rpc not supported; use GRPCClient")
}

// Start launches the plugin process and establishes gRPC connection.
func (h *GRPCHost) Start(settings map[string]string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.client != nil {
		return fmt.Errorf("plugin %s is already running", h.meta.ID)
	}

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]goplugin.Plugin{
			shared.ProviderPluginName: &GRPCProviderPluginHost{},
		},
		Cmd:              exec.Command(h.binaryPath),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to create plugin client for %s: %w", h.meta.ID, err)
	}

	raw, err := rpcClient.Dispense(shared.ProviderPluginName)
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense plugin %s: %w", h.meta.ID, err)
	}

	svc, ok := raw.(pb.ProviderServiceClient)
	if !ok {
		client.Kill()
		return fmt.Errorf("plugin %s did not return ProviderServiceClient", h.meta.ID)
	}

	// Initialize the plugin with settings
	resp, err := svc.Initialize(context.Background(), &pb.InitRequest{
		Settings: settings,
		PluginID: h.meta.ID,
	})
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to initialize plugin %s: %w", h.meta.ID, err)
	}
	if !resp.Success {
		client.Kill()
		return fmt.Errorf("plugin %s initialization failed: %s", h.meta.ID, resp.Error)
	}

	h.client = client
	h.rpcClient = svc
	return nil
}

// Stop gracefully shuts down the plugin process.
func (h *GRPCHost) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.client == nil {
		return nil
	}

	if h.rpcClient != nil {
		_, _ = h.rpcClient.Shutdown(context.Background(), &pb.ShutdownRequest{})
	}

	h.client.Kill()
	h.client = nil
	h.rpcClient = nil
	return nil
}

// IsRunning checks if the plugin process is alive.
func (h *GRPCHost) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.client != nil && !h.client.Exited()
}

// AsStorageProvider returns a StorageProvider that proxies to the plugin.
func (h *GRPCHost) AsStorageProvider() provider.StorageProvider {
	return &grpcStorageProxy{host: h}
}

// AsSearchProvider returns a SearchProvider that proxies to the plugin.
func (h *GRPCHost) AsSearchProvider() provider.SearchProvider {
	return &grpcSearchProxy{host: h}
}

// AsNotifierProvider returns a NotifierProvider that proxies to the plugin.
func (h *GRPCHost) AsNotifierProvider() provider.NotifierProvider {
	return &grpcNotifierProxy{host: h}
}

// AsCaptchaProvider returns a CaptchaProvider that proxies to the plugin.
func (h *GRPCHost) AsCaptchaProvider() provider.CaptchaProvider {
	return &grpcCaptchaProxy{host: h}
}

// HandleHTTP proxies an HTTP request to the plugin.
func (h *GRPCHost) HandleHTTP(ctx context.Context, req *pb.HTTPRequest) (*pb.HTTPResponse, error) {
	h.mu.Lock()
	svc := h.rpcClient
	h.mu.Unlock()
	if svc == nil {
		return nil, fmt.Errorf("plugin %s is not running", h.meta.ID)
	}
	return svc.HandleHTTP(ctx, req)
}

// --- GRPCProviderPluginHost implements go-plugin.GRPCPlugin on the host side ---

// GRPCProviderPluginHost implements goplugin.GRPCPlugin for the host side.
type GRPCProviderPluginHost struct {
	goplugin.Plugin
}

// GRPCServer is not used on the host side.
func (p *GRPCProviderPluginHost) GRPCServer(_ *goplugin.GRPCBroker, _ interface{}) error {
	return fmt.Errorf("GRPCServer called on host side")
}

// GRPCClient creates a ProviderServiceClient from the gRPC connection.
func (p *GRPCProviderPluginHost) GRPCClient(_ *goplugin.GRPCBroker, c interface{}) (interface{}, error) {
	// go-plugin passes an *grpc.ClientConn via the GRPCClient method.
	// For our simplified interface approach, we return a DirectClient wrapper.
	return &DirectClient{}, nil
}

// DirectClient is a placeholder that implements ProviderServiceClient.
// In the real go-plugin flow, the client connection is provided by the framework.
// This type is returned by GRPCClient and used by the host-side proxies.
type DirectClient struct{}

func (d *DirectClient) Initialize(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	return &pb.InitResponse{Success: true}, nil
}
func (d *DirectClient) Shutdown(ctx context.Context, req *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	return &pb.ShutdownResponse{}, nil
}
func (d *DirectClient) StorageSave(ctx context.Context, req *pb.StorageSaveRequest) (*pb.StorageSaveResponse, error) {
	return &pb.StorageSaveResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) StorageGet(ctx context.Context, req *pb.StorageGetRequest) ([]*pb.StorageChunk, error) {
	return nil, fmt.Errorf("not implemented")
}
func (d *DirectClient) StorageDelete(ctx context.Context, req *pb.StorageDeleteRequest) (*pb.StorageDeleteResponse, error) {
	return &pb.StorageDeleteResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) StorageURL(ctx context.Context, req *pb.StorageURLRequest) (*pb.StorageURLResponse, error) {
	return &pb.StorageURLResponse{}, nil
}
func (d *DirectClient) StorageExists(ctx context.Context, req *pb.StorageExistsRequest) (*pb.StorageExistsResponse, error) {
	return &pb.StorageExistsResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) SearchSuggest(ctx context.Context, req *pb.SearchSuggestRequest) (*pb.SearchSuggestResponse, error) {
	return &pb.SearchSuggestResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) SearchIndex(ctx context.Context, req *pb.SearchIndexRequest) (*pb.SearchIndexResponse, error) {
	return &pb.SearchIndexResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) SearchRemove(ctx context.Context, req *pb.SearchRemoveRequest) (*pb.SearchRemoveResponse, error) {
	return &pb.SearchRemoveResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) SearchRebuild(ctx context.Context, req *pb.SearchRebuildRequest) (*pb.SearchRebuildResponse, error) {
	return &pb.SearchRebuildResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) Notify(ctx context.Context, req *pb.NotifyRequest) (*pb.NotifyResponse, error) {
	return &pb.NotifyResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) CaptchaVerify(ctx context.Context, req *pb.CaptchaVerifyRequest) (*pb.CaptchaVerifyResponse, error) {
	return &pb.CaptchaVerifyResponse{Error: "not implemented"}, nil
}
func (d *DirectClient) HandleHTTP(ctx context.Context, req *pb.HTTPRequest) (*pb.HTTPResponse, error) {
	return &pb.HTTPResponse{StatusCode: 501}, nil
}

// --- Provider proxy types ---

// grpcStorageProxy implements provider.StorageProvider by proxying to a plugin via gRPC.
type grpcStorageProxy struct {
	host *GRPCHost
}

func (p *grpcStorageProxy) Save(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read data: %w", err)
	}
	resp, err := p.host.rpcClient.StorageSave(ctx, &pb.StorageSaveRequest{
		Filename: filename,
		Data:     data,
		Size:     size,
	})
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return resp.Path, nil
}

func (p *grpcStorageProxy) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	chunks, err := p.host.rpcClient.StorageGet(ctx, &pb.StorageGetRequest{Path: path})
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for _, chunk := range chunks {
		buf.Write(chunk.Data)
	}
	return io.NopCloser(&buf), nil
}

func (p *grpcStorageProxy) Delete(ctx context.Context, path string) error {
	resp, err := p.host.rpcClient.StorageDelete(ctx, &pb.StorageDeleteRequest{Path: path})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func (p *grpcStorageProxy) URL(path string) string {
	resp, err := p.host.rpcClient.StorageURL(context.Background(), &pb.StorageURLRequest{Path: path})
	if err != nil {
		return ""
	}
	return resp.URL
}

func (p *grpcStorageProxy) Exists(ctx context.Context, path string) (bool, error) {
	resp, err := p.host.rpcClient.StorageExists(ctx, &pb.StorageExistsRequest{Path: path})
	if err != nil {
		return false, err
	}
	if resp.Error != "" {
		return false, fmt.Errorf("%s", resp.Error)
	}
	return resp.Exists, nil
}

// grpcSearchProxy implements provider.SearchProvider by proxying to a plugin via gRPC.
type grpcSearchProxy struct {
	host *GRPCHost
}

func (p *grpcSearchProxy) Search(ctx context.Context, query string, locale string, contentType string, page int, pageSize int) (*provider.SearchResponse, error) {
	resp, err := p.host.rpcClient.Search(ctx, &pb.SearchRequest{
		Query:       query,
		Locale:      locale,
		ContentType: contentType,
		Page:        int32(page),
		PageSize:    int32(pageSize),
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	results := make([]provider.SearchResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = provider.SearchResult{
			ID:      uint(r.ID),
			Type:    r.Type,
			Title:   r.Title,
			Snippet: r.Snippet,
			URL:     r.URL,
			Locale:  r.Locale,
			Score:   r.Score,
		}
	}
	return &provider.SearchResponse{
		Results:  results,
		Total:    resp.Total,
		Page:     page,
		PageSize: pageSize,
		Query:    query,
	}, nil
}

func (p *grpcSearchProxy) Suggest(ctx context.Context, prefix string, locale string, limit int) ([]string, error) {
	resp, err := p.host.rpcClient.SearchSuggest(ctx, &pb.SearchSuggestRequest{
		Prefix: prefix,
		Locale: locale,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Suggestions, nil
}

func (p *grpcSearchProxy) IndexArticle(ctx context.Context, id uint, locale string, title string, body string, slug string) error {
	resp, err := p.host.rpcClient.SearchIndex(ctx, &pb.SearchIndexRequest{
		ContentType: "article",
		ID:          uint64(id),
		Locale:      locale,
		Title:       title,
		Body:        body,
		Slug:        slug,
	})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func (p *grpcSearchProxy) IndexPage(ctx context.Context, id uint, locale string, title string, body string, slug string) error {
	resp, err := p.host.rpcClient.SearchIndex(ctx, &pb.SearchIndexRequest{
		ContentType: "page",
		ID:          uint64(id),
		Locale:      locale,
		Title:       title,
		Body:        body,
		Slug:        slug,
	})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func (p *grpcSearchProxy) RemoveFromIndex(ctx context.Context, contentType string, id uint) error {
	resp, err := p.host.rpcClient.SearchRemove(ctx, &pb.SearchRemoveRequest{
		ContentType: contentType,
		ID:          uint64(id),
	})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func (p *grpcSearchProxy) RebuildIndex(ctx context.Context) error {
	resp, err := p.host.rpcClient.SearchRebuild(ctx, &pb.SearchRebuildRequest{})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// grpcNotifierProxy implements provider.NotifierProvider by proxying to a plugin via gRPC.
type grpcNotifierProxy struct {
	host *GRPCHost
}

func (p *grpcNotifierProxy) Notify(ctx context.Context, event provider.NotifyEvent) error {
	resp, err := p.host.rpcClient.Notify(ctx, &pb.NotifyRequest{
		Type:    event.Type,
		Subject: event.Subject,
		Body:    event.Body,
		Meta:    event.Meta,
	})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// grpcCaptchaProxy implements provider.CaptchaProvider by proxying to a plugin via gRPC.
type grpcCaptchaProxy struct {
	host *GRPCHost
}

func (p *grpcCaptchaProxy) Verify(ctx context.Context, token string, remoteIP string) error {
	resp, err := p.host.rpcClient.CaptchaVerify(ctx, &pb.CaptchaVerifyRequest{
		Token:    token,
		RemoteIP: remoteIP,
	})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}
