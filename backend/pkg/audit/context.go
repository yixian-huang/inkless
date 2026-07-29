package audit

import (
	"context"
	"strconv"
)

type metadataContextKey struct{}

// Metadata describes the actor and request that triggered an audited operation.
type Metadata struct {
	Actor     string
	ActorID   uint
	// APIKeyID is set when the request authenticated with a personal API key (ink_…).
	APIKeyID  uint
	IP        string
	UserAgent string
	RequestID string
}

// WithMetadata attaches audit metadata to an operation context.
func WithMetadata(ctx context.Context, metadata Metadata) context.Context {
	return context.WithValue(ctx, metadataContextKey{}, metadata)
}

// MetadataFromContext retrieves audit metadata from an operation context.
func MetadataFromContext(ctx context.Context) Metadata {
	if ctx == nil {
		return Metadata{}
	}
	metadata, _ := ctx.Value(metadataContextKey{}).(Metadata)
	return metadata
}

// ActorLabel returns a stable, human-readable actor identifier.
// API key calls are labeled as "username (api_key:N)" so audit logs distinguish
// browser sessions from local agents / PicGo clients without losing the owner.
func (m Metadata) ActorLabel() string {
	base := m.Actor
	if base == "" {
		if m.ActorID != 0 {
			base = "user:" + strconv.FormatUint(uint64(m.ActorID), 10)
		} else {
			base = "system"
		}
	}
	if m.APIKeyID != 0 {
		return base + " (api_key:" + strconv.FormatUint(uint64(m.APIKeyID), 10) + ")"
	}
	return base
}

// AddMetadata adds non-sensitive request metadata to audit event details.
func AddMetadata(details map[string]interface{}, metadata Metadata) map[string]interface{} {
	if details == nil {
		details = make(map[string]interface{})
	}
	if metadata.ActorID != 0 {
		details["actor_id"] = metadata.ActorID
	}
	if metadata.APIKeyID != 0 {
		details["api_key_id"] = metadata.APIKeyID
		details["auth_method"] = "api_key"
	}
	if metadata.IP != "" {
		details["ip"] = metadata.IP
	}
	if metadata.UserAgent != "" {
		details["user_agent"] = metadata.UserAgent
	}
	if metadata.RequestID != "" {
		details["request_id"] = metadata.RequestID
	}
	return details
}
