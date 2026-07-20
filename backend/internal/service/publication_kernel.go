package service

import (
	"context"
	"fmt"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
)

// ContentPublisher is the shared publication contract for schedule-and-publish
// resources (articles, pages, future types). SchedulerService and other
// lifecycle coordinators depend on this interface instead of type-switching
// on concrete services.
//
// Concurrency models differ by resource:
//   - Articles: optimistic lock via ExpectedUpdatedAt (row UpdatedAt).
//   - Pages: optimistic lock via ExpectedVersion (draftVersion).
//
// PrepareSchedule resolves and returns the token to persist on the job;
// ExecuteScheduled re-checks that token at publish time.
type ContentPublisher interface {
	ContentType() model.ScheduledContentType

	// PrepareSchedule validates that contentID can be scheduled with the
	// given optional concurrency hint and publish payload. It returns the
	// concurrency token to store on the durable job.
	PrepareSchedule(
		ctx context.Context,
		contentID uint,
		expectedVersion *int,
		expectedUpdatedAt *time.Time,
		payload model.JSONMap,
	) (version *int, updatedAt *time.Time, err error)

	// ExecuteScheduled publishes contentID for a claimed due job.
	// publishedAt is the scheduler tick time (not wall-clock re-read).
	ExecuteScheduled(
		ctx context.Context,
		contentID uint,
		publishedAt time.Time,
		actorID uint,
		expectedVersion *int,
		expectedUpdatedAt *time.Time,
		payload model.JSONMap,
	) error

	// MarkScheduled updates resource-level schedule metadata after a job is
	// enqueued (status=scheduled when not already published, ScheduledAt set).
	MarkScheduled(ctx context.Context, contentID uint, scheduledAt time.Time) error

	// ClearSchedule clears resource-level schedule metadata after cancel.
	ClearSchedule(ctx context.Context, contentID uint) error

	// Describe returns admin-list title/slug (payload snapshot preferred).
	Describe(ctx context.Context, contentID uint, payload model.JSONMap) (title, slug string)
}

// PublicationKernel routes lifecycle operations by ScheduledContentType.
// It is the single registry used by SchedulerService so adding a new
// publishable resource is "implement ContentPublisher + Register".
type PublicationKernel struct {
	byType map[model.ScheduledContentType]ContentPublisher
}

// NewPublicationKernel builds a registry from the given publishers.
// Duplicate content types panic (wiring bug at process start).
func NewPublicationKernel(publishers ...ContentPublisher) *PublicationKernel {
	k := &PublicationKernel{
		byType: make(map[model.ScheduledContentType]ContentPublisher, len(publishers)),
	}
	for _, p := range publishers {
		if p == nil {
			continue
		}
		t := p.ContentType()
		if _, exists := k.byType[t]; exists {
			panic(fmt.Sprintf("publication kernel: duplicate publisher for %q", t))
		}
		k.byType[t] = p
	}
	return k
}

// Publisher returns the ContentPublisher for t, or a typed error.
func (k *PublicationKernel) Publisher(t model.ScheduledContentType) (ContentPublisher, error) {
	if k == nil {
		return nil, fmt.Errorf("publication kernel is not configured")
	}
	p, ok := k.byType[t]
	if !ok || p == nil {
		return nil, fmt.Errorf("unsupported content type %q", t)
	}
	return p, nil
}

// PrepareSchedule delegates to the registered publisher.
func (k *PublicationKernel) PrepareSchedule(
	ctx context.Context,
	contentType model.ScheduledContentType,
	contentID uint,
	expectedVersion *int,
	expectedUpdatedAt *time.Time,
	payload model.JSONMap,
) (*int, *time.Time, error) {
	p, err := k.Publisher(contentType)
	if err != nil {
		return nil, nil, err
	}
	return p.PrepareSchedule(ctx, contentID, expectedVersion, expectedUpdatedAt, payload)
}

// ExecuteScheduled delegates to the registered publisher.
func (k *PublicationKernel) ExecuteScheduled(
	ctx context.Context,
	contentType model.ScheduledContentType,
	contentID uint,
	publishedAt time.Time,
	actorID uint,
	expectedVersion *int,
	expectedUpdatedAt *time.Time,
	payload model.JSONMap,
) error {
	p, err := k.Publisher(contentType)
	if err != nil {
		return err
	}
	return p.ExecuteScheduled(
		ctx,
		contentID,
		publishedAt,
		actorID,
		expectedVersion,
		expectedUpdatedAt,
		payload,
	)
}

// MarkScheduled delegates to the registered publisher.
func (k *PublicationKernel) MarkScheduled(
	ctx context.Context,
	contentType model.ScheduledContentType,
	contentID uint,
	scheduledAt time.Time,
) error {
	p, err := k.Publisher(contentType)
	if err != nil {
		return err
	}
	return p.MarkScheduled(ctx, contentID, scheduledAt)
}

// ClearSchedule delegates to the registered publisher.
func (k *PublicationKernel) ClearSchedule(
	ctx context.Context,
	contentType model.ScheduledContentType,
	contentID uint,
) error {
	p, err := k.Publisher(contentType)
	if err != nil {
		return err
	}
	return p.ClearSchedule(ctx, contentID)
}

// Describe delegates to the registered publisher.
func (k *PublicationKernel) Describe(
	ctx context.Context,
	contentType model.ScheduledContentType,
	contentID uint,
	payload model.JSONMap,
) (title, slug string) {
	p, err := k.Publisher(contentType)
	if err != nil {
		return "", ""
	}
	return p.Describe(ctx, contentID, payload)
}

// ContentTypes returns registered types (stable for diagnostics).
func (k *PublicationKernel) ContentTypes() []model.ScheduledContentType {
	if k == nil {
		return nil
	}
	out := make([]model.ScheduledContentType, 0, len(k.byType))
	for t := range k.byType {
		out = append(out, t)
	}
	return out
}
