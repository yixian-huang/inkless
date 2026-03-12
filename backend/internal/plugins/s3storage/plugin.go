// Package s3storage provides an S3-compatible storage plugin for Impress CMS.
// It implements the provider.StorageProvider interface and supports AWS S3,
// MinIO, and Alibaba Cloud OSS via their S3-compatible APIs.
package s3storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"blotting-consultancy/internal/plugin"
	"blotting-consultancy/internal/provider"
)

// Manifest describes this plugin's metadata.
var Manifest = plugin.PluginMeta{
	ID:            "s3s-storage",
	Name:          "S3 Storage Plugin",
	NameZh:        "S3 存储插件",
	Version:       "1.0.0",
	Description:   "Stores uploaded files in any S3-compatible service: AWS S3, MinIO, Aliyun OSS.",
	Author:        "Impress CMS",
	License:       "MIT",
	MinAppVersion: "1.0.0",
	Permissions:   []plugin.Permission{plugin.PermNetworkOutbound, plugin.PermFileSystemRead},
	Providers: []plugin.ProviderDecl{
		{Type: "storage", Name: "s3"},
	},
}

// Config holds the S3 storage plugin configuration.
type Config struct {
	// Endpoint is the S3-compatible endpoint URL (e.g. "https://s3.amazonaws.com").
	// For MinIO: "http://localhost:9000". For Aliyun OSS: "https://oss-cn-hangzhou.aliyuncs.com".
	Endpoint string

	// Region is the S3 region (e.g. "us-east-1"). Required for AWS S3.
	Region string

	// Bucket is the S3 bucket name.
	Bucket string

	// AccessKeyID is the S3 access key ID.
	AccessKeyID string

	// SecretAccessKey is the S3 secret access key.
	SecretAccessKey string

	// BaseURL is the public base URL for stored files.
	// If empty, files are served from the endpoint URL.
	BaseURL string

	// UsePathStyle controls whether to use path-style addressing (required for MinIO).
	// Default is false (virtual-hosted style for AWS S3).
	UsePathStyle bool
}

// Plugin implements provider.StorageProvider using S3-compatible storage.
type Plugin struct {
	config     Config
	httpClient *http.Client
}

// New creates a new S3 storage plugin with the provided configuration.
func New(cfg Config) (*Plugin, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("s3storage: endpoint is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3storage: bucket is required")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("s3storage: access credentials are required")
	}

	return &Plugin{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// NewFromSettings creates a Plugin from a string settings map (used by plugin manager).
func NewFromSettings(settings map[string]string) (*Plugin, error) {
	cfg := Config{
		Endpoint:        settings["endpoint"],
		Region:          settings["region"],
		Bucket:          settings["bucket"],
		AccessKeyID:     settings["access_key_id"],
		SecretAccessKey: settings["secret_access_key"],
		BaseURL:         settings["base_url"],
		UsePathStyle:    settings["use_path_style"] == "true",
	}
	return New(cfg)
}

// bucketURL returns the base URL for the bucket.
func (p *Plugin) bucketURL() string {
	endpoint := strings.TrimRight(p.config.Endpoint, "/")
	if p.config.UsePathStyle {
		return fmt.Sprintf("%s/%s", endpoint, p.config.Bucket)
	}
	// Virtual-hosted style: bucket is subdomain
	// For simplicity in this demo plugin, parse the endpoint host
	return fmt.Sprintf("%s/%s", endpoint, p.config.Bucket)
}

// objectURL returns the full URL for an object.
func (p *Plugin) objectURL(path string) string {
	base := strings.TrimRight(p.bucketURL(), "/")
	return fmt.Sprintf("%s/%s", base, strings.TrimLeft(path, "/"))
}

// Save stores a file in S3 and returns its relative path.
// The path format is: filename (as provided, no directory structure added here).
func (p *Plugin) Save(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("s3storage: failed to read file data: %w", err)
	}

	url := p.objectURL(filename)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("s3storage: failed to build PUT request: %w", err)
	}
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "application/octet-stream")

	// Sign and send the request
	if err := p.signRequest(req, data); err != nil {
		return "", fmt.Errorf("s3storage: failed to sign request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("s3storage: PUT request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("s3storage: PUT failed with status %d: %s", resp.StatusCode, string(body))
	}

	return filename, nil
}

// Get retrieves a file from S3 by its path.
func (p *Plugin) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	url := p.objectURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("s3storage: failed to build GET request: %w", err)
	}

	if err := p.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3storage: failed to sign request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3storage: GET request failed: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("s3storage: object not found: %s", path)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("s3storage: GET failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// Delete removes a file from S3 by its path.
func (p *Plugin) Delete(ctx context.Context, path string) error {
	url := p.objectURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("s3storage: failed to build DELETE request: %w", err)
	}

	if err := p.signRequest(req, nil); err != nil {
		return fmt.Errorf("s3storage: failed to sign request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("s3storage: DELETE request failed: %w", err)
	}
	defer resp.Body.Close()

	// S3 returns 204 No Content on success; also accept 404 as "already gone"
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3storage: DELETE failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// URL returns the public URL for a stored file.
func (p *Plugin) URL(path string) string {
	if p.config.BaseURL != "" {
		base := strings.TrimRight(p.config.BaseURL, "/")
		return fmt.Sprintf("%s/%s", base, strings.TrimLeft(path, "/"))
	}
	return p.objectURL(path)
}

// Exists checks whether a file exists in S3.
func (p *Plugin) Exists(ctx context.Context, path string) (bool, error) {
	url := p.objectURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, fmt.Errorf("s3storage: failed to build HEAD request: %w", err)
	}

	if err := p.signRequest(req, nil); err != nil {
		return false, fmt.Errorf("s3storage: failed to sign request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("s3storage: HEAD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, fmt.Errorf("s3storage: HEAD returned unexpected status %d", resp.StatusCode)
}

// signRequest adds AWS Signature Version 4 authorization headers to the request.
// This is a simplified implementation sufficient for demo/reference purposes.
// Production implementations should use the official AWS SDK or a well-tested signing library.
func (p *Plugin) signRequest(req *http.Request, body []byte) error {
	now := time.Now().UTC()
	dateShort := now.Format("20060102")
	dateLong := now.Format("20060102T150405Z")

	req.Header.Set("x-amz-date", dateLong)
	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")
	req.Header.Set("Host", req.URL.Host)

	// Construct canonical headers
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:UNSIGNED-PAYLOAD\nx-amz-date:%s\n",
		req.URL.Host, dateLong)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateShort, p.config.Region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		dateLong,
		credentialScope,
		fmt.Sprintf("%x", sha256sum([]byte(canonicalRequest))),
	}, "\n")

	// Derive signing key
	signingKey := deriveSigningKey(p.config.SecretAccessKey, dateShort, p.config.Region)
	signature := fmt.Sprintf("%x", hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		p.config.AccessKeyID, credentialScope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", authHeader)
	return nil
}

// Ensure Plugin implements StorageProvider at compile time.
var _ provider.StorageProvider = (*Plugin)(nil)
