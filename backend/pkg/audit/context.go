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
func (m Metadata) ActorLabel() string {
	if m.Actor != "" {
		return m.Actor
	}
	if m.ActorID != 0 {
		return "user:" + strconv.FormatUint(uint64(m.ActorID), 10)
	}
	return "system"
}

// AddMetadata adds non-sensitive request metadata to audit event details.
func AddMetadata(details map[string]interface{}, metadata Metadata) map[string]interface{} {
	if details == nil {
		details = make(map[string]interface{})
	}
	if metadata.ActorID != 0 {
		details["actor_id"] = metadata.ActorID
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
