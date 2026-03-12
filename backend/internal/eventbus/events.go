package eventbus

// Content lifecycle event types.
const (
	ContentCreated   = "content.created"
	ContentUpdated   = "content.updated"
	ContentPublished = "content.published"
	ContentDeleted   = "content.deleted"
)

// Comment lifecycle event types.
const (
	CommentCreated  = "comment.created"
	CommentApproved = "comment.approved"
	CommentDeleted  = "comment.deleted"
)

// ContentEventPayload carries data for content lifecycle events.
type ContentEventPayload struct {
	ContentType string // "article" or "page"
	ContentID   uint
	Slug        string
	Locale      string
	Title       string
	Action      string // the event type constant, for convenience
}

// CommentEventPayload carries data for comment lifecycle events.
type CommentEventPayload struct {
	CommentID   uint
	ContentType string // "article" or "page"
	ContentID   uint
	AuthorName  string
	Action      string
}
