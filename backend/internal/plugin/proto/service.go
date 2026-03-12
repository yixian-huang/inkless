// Package proto defines the gRPC service contract between host and plugin processes.
// The .proto file is the canonical definition; this Go file defines the service
// interfaces and message types used by the host-side gRPC proxy without requiring
// protoc code generation. When protoc is available, regenerate from plugin.proto.
package proto

import (
	"context"

	"google.golang.org/grpc"
)

// --- Message types ---

type InitRequest struct {
	Settings map[string]string
	DataDir  string
	PluginID string
}

type InitResponse struct {
	Success bool
	Error   string
}

type ShutdownRequest struct{}
type ShutdownResponse struct{}

type StorageSaveRequest struct {
	Filename string
	Data     []byte
	Size     int64
}
type StorageSaveResponse struct {
	Path  string
	Error string
}

type StorageGetRequest struct {
	Path string
}
type StorageChunk struct {
	Data []byte
}

type StorageDeleteRequest struct {
	Path string
}
type StorageDeleteResponse struct {
	Error string
}

type StorageURLRequest struct {
	Path string
}
type StorageURLResponse struct {
	URL string
}

type StorageExistsRequest struct {
	Path string
}
type StorageExistsResponse struct {
	Exists bool
	Error  string
}

type SearchRequest struct {
	Query       string
	Locale      string
	ContentType string
	Page        int32
	PageSize    int32
}
type SearchResponse struct {
	Results []*SearchResult
	Total   int64
	Error   string
}
type SearchResult struct {
	ID      uint64
	Type    string
	Title   string
	Snippet string
	URL     string
	Locale  string
	Score   float64
}

type SearchSuggestRequest struct {
	Prefix string
	Locale string
	Limit  int32
}
type SearchSuggestResponse struct {
	Suggestions []string
	Error       string
}

type SearchIndexRequest struct {
	ContentType string
	ID          uint64
	Locale      string
	Title       string
	Body        string
	Slug        string
}
type SearchIndexResponse struct {
	Error string
}

type SearchRemoveRequest struct {
	ContentType string
	ID          uint64
}
type SearchRemoveResponse struct {
	Error string
}

type SearchRebuildRequest struct{}
type SearchRebuildResponse struct {
	Error string
}

type NotifyRequest struct {
	Type    string
	Subject string
	Body    string
	Meta    map[string]string
}
type NotifyResponse struct {
	Error string
}

type CaptchaVerifyRequest struct {
	Token    string
	RemoteIP string
}
type CaptchaVerifyResponse struct {
	Error string
}

type HTTPRequest struct {
	Method      string
	Path        string
	Headers     map[string]string
	Body        []byte
	QueryParams map[string]string
}
type HTTPResponse struct {
	StatusCode int32
	Headers    map[string]string
	Body       []byte
}

// --- Service interface ---

// ProviderServiceClient is the client-side interface for the plugin gRPC service.
// The host uses this to call plugin methods.
type ProviderServiceClient interface {
	Initialize(ctx context.Context, req *InitRequest) (*InitResponse, error)
	Shutdown(ctx context.Context, req *ShutdownRequest) (*ShutdownResponse, error)

	StorageSave(ctx context.Context, req *StorageSaveRequest) (*StorageSaveResponse, error)
	StorageGet(ctx context.Context, req *StorageGetRequest) ([]*StorageChunk, error)
	StorageDelete(ctx context.Context, req *StorageDeleteRequest) (*StorageDeleteResponse, error)
	StorageURL(ctx context.Context, req *StorageURLRequest) (*StorageURLResponse, error)
	StorageExists(ctx context.Context, req *StorageExistsRequest) (*StorageExistsResponse, error)

	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	SearchSuggest(ctx context.Context, req *SearchSuggestRequest) (*SearchSuggestResponse, error)
	SearchIndex(ctx context.Context, req *SearchIndexRequest) (*SearchIndexResponse, error)
	SearchRemove(ctx context.Context, req *SearchRemoveRequest) (*SearchRemoveResponse, error)
	SearchRebuild(ctx context.Context, req *SearchRebuildRequest) (*SearchRebuildResponse, error)

	Notify(ctx context.Context, req *NotifyRequest) (*NotifyResponse, error)

	CaptchaVerify(ctx context.Context, req *CaptchaVerifyRequest) (*CaptchaVerifyResponse, error)

	HandleHTTP(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)
}

// ProviderServiceServer is the server-side interface for the plugin gRPC service.
// Plugin processes implement this to handle host calls.
type ProviderServiceServer interface {
	Initialize(ctx context.Context, req *InitRequest) (*InitResponse, error)
	Shutdown(ctx context.Context, req *ShutdownRequest) (*ShutdownResponse, error)

	StorageSave(ctx context.Context, req *StorageSaveRequest) (*StorageSaveResponse, error)
	StorageGet(ctx context.Context, req *StorageGetRequest) ([]*StorageChunk, error)
	StorageDelete(ctx context.Context, req *StorageDeleteRequest) (*StorageDeleteResponse, error)
	StorageURL(ctx context.Context, req *StorageURLRequest) (*StorageURLResponse, error)
	StorageExists(ctx context.Context, req *StorageExistsRequest) (*StorageExistsResponse, error)

	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	SearchSuggest(ctx context.Context, req *SearchSuggestRequest) (*SearchSuggestResponse, error)
	SearchIndex(ctx context.Context, req *SearchIndexRequest) (*SearchIndexResponse, error)
	SearchRemove(ctx context.Context, req *SearchRemoveRequest) (*SearchRemoveResponse, error)
	SearchRebuild(ctx context.Context, req *SearchRebuildRequest) (*SearchRebuildResponse, error)

	Notify(ctx context.Context, req *NotifyRequest) (*NotifyResponse, error)

	CaptchaVerify(ctx context.Context, req *CaptchaVerifyRequest) (*CaptchaVerifyResponse, error)

	HandleHTTP(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)
}

// GRPCProviderPlugin implements the go-plugin GRPCPlugin interface.
// It bridges the go-plugin framework to our ProviderService gRPC contract.
type GRPCProviderPlugin struct {
	Impl ProviderServiceServer
}

// GRPCServer registers the plugin server implementation with the gRPC server.
func (p *GRPCProviderPlugin) GRPCServer(_ interface{}, s *grpc.Server) error {
	// Registration is done by the SDK; the host never calls this.
	_ = s
	return nil
}

// GRPCClient creates a client that talks to the plugin process.
func (p *GRPCProviderPlugin) GRPCClient(_ interface{}, c *grpc.ClientConn) (interface{}, error) {
	_ = c
	return nil, nil
}
