package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"blotting-consultancy/internal/eventbus"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/audit"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type serviceAuditWriterStub struct {
	events []audit.Event
	err    error
}

func (w *serviceAuditWriterStub) Write(_ context.Context, event audit.Event) error {
	w.events = append(w.events, event)
	return w.err
}

func setupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&model.UnifiedPage{}, &model.PageVersion{}, &model.PageTemplate{}, &model.SiteConfig{})
	return db
}

func TestUnifiedPageService_Publish(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	svc := service.NewUnifiedPageService(pageRepo, versionRepo)
	ctx := context.Background()

	page := &model.UnifiedPage{
		Slug: "test", Mode: "composable", DraftVersion: 1,
		DraftConfig: model.JSONMap{"sections": []any{}},
	}
	pageRepo.Create(ctx, page)

	err := svc.Publish(ctx, page.ID, 1, 1) // expectedDraftVersion=1, userID=1
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	updated, _ := pageRepo.FindByID(ctx, page.ID)
	if updated.Status != "published" {
		t.Errorf("expected published, got %q", updated.Status)
	}
	if updated.PublishedVersion != 1 {
		t.Errorf("expected publishedVersion 1, got %d", updated.PublishedVersion)
	}

	// Verify version record created
	versions, count, _ := versionRepo.ListByPageID(ctx, page.ID, 0, 10)
	if count != 1 {
		t.Errorf("expected 1 version, got %d", count)
	}
	if versions[0].CreatedBy != 1 {
		t.Errorf("expected createdBy 1, got %d", versions[0].CreatedBy)
	}
}

func TestUnifiedPageService_Publish_VersionConflict(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	svc := service.NewUnifiedPageService(pageRepo, versionRepo)
	ctx := context.Background()

	page := &model.UnifiedPage{
		Slug: "test", Mode: "composable", DraftVersion: 2,
		DraftConfig: model.JSONMap{"sections": []any{}},
	}
	pageRepo.Create(ctx, page)

	err := svc.Publish(ctx, page.ID, 1, 1) // wrong expectedDraftVersion
	if err == nil {
		t.Error("expected version conflict error")
	}
}

func TestUnifiedPageService_Rollback(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	svc := service.NewUnifiedPageService(pageRepo, versionRepo)
	ctx := context.Background()

	page := &model.UnifiedPage{
		Slug: "test", Mode: "composable", DraftVersion: 1,
		DraftConfig: model.JSONMap{"sections": []any{map[string]any{"type": "hero"}}},
	}
	pageRepo.Create(ctx, page)

	// Publish v1 (draftVersion=1)
	if err := svc.Publish(ctx, page.ID, 1, 1); err != nil {
		t.Fatalf("publish v1: %v", err)
	}

	// Modify draft (current draftVersion=1, UpdateDraft increments to 2)
	newDraftVer, err := pageRepo.UpdateDraft(ctx, page.ID, 1, model.JSONMap{"sections": []any{map[string]any{"type": "hero"}, map[string]any{"type": "rich-text"}}})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}

	// Publish v2 (draftVersion=newDraftVer)
	if err := svc.Publish(ctx, page.ID, newDraftVer, 1); err != nil {
		t.Fatalf("publish v2: %v", err)
	}

	// Rollback to v1 → creates v3
	err = svc.Rollback(ctx, page.ID, 1, 1)
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}

	updated, _ := pageRepo.FindByID(ctx, page.ID)
	if updated.PublishedVersion != 3 {
		t.Errorf("expected publishedVersion 3 after rollback, got %d", updated.PublishedVersion)
	}
}

func TestUnifiedPageService_Unpublish(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	svc := service.NewUnifiedPageService(pageRepo, versionRepo)
	ctx := context.Background()

	page := &model.UnifiedPage{
		Slug: "test", Mode: "composable", DraftVersion: 1,
		DraftConfig: model.JSONMap{"sections": []any{}},
	}
	pageRepo.Create(ctx, page)

	// Publish first
	if err := svc.Publish(ctx, page.ID, 1, 1); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Unpublish
	if _, err := svc.Unpublish(ctx, page.ID); err != nil {
		t.Fatalf("unpublish: %v", err)
	}

	updated, _ := pageRepo.FindByID(ctx, page.ID)
	if updated.Status != "draft" {
		t.Errorf("expected draft, got %q", updated.Status)
	}
	if updated.PublishedConfig != nil {
		t.Error("expected PublishedConfig to be nil after unpublish")
	}
}

func TestUnifiedPageService_PublishLifecycleEmitsEventsAndAudit(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	bus := eventbus.New()
	writer := &serviceAuditWriterStub{}
	svc := service.NewUnifiedPageService(pageRepo, versionRepo, bus).WithAuditWriter(writer)

	var domainEvents []eventbus.Event
	for _, eventType := range []string{
		eventbus.ContentPublished,
		eventbus.ContentUnpublished,
		eventbus.ContentRolledBack,
	} {
		bus.Subscribe(eventType, eventbus.SyncHandler(func(event eventbus.Event) {
			domainEvents = append(domainEvents, event)
		}))
	}

	ctx := audit.WithMetadata(context.Background(), audit.Metadata{
		Actor:     "publisher",
		ActorID:   9,
		IP:        "127.0.0.1",
		UserAgent: "service-test",
		RequestID: "req-publish",
	})
	page := &model.UnifiedPage{
		Slug:         "audited-page",
		ZhTitle:      "审计页面",
		Mode:         "composable",
		DraftVersion: 1,
		DraftConfig:  model.JSONMap{"sections": []any{}},
	}
	if err := pageRepo.Create(ctx, page); err != nil {
		t.Fatalf("create page: %v", err)
	}

	if err := svc.Publish(ctx, page.ID, 1, 9); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := svc.Unpublish(ctx, page.ID, 9); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	if err := svc.Rollback(ctx, page.ID, 1, 9); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if len(domainEvents) != 3 {
		t.Fatalf("expected 3 domain events, got %d", len(domainEvents))
	}
	wantEventTypes := []string{
		eventbus.ContentPublished,
		eventbus.ContentUnpublished,
		eventbus.ContentRolledBack,
	}
	for i, wantType := range wantEventTypes {
		if domainEvents[i].Type != wantType {
			t.Errorf("domain event %d type=%q, want %q", i, domainEvents[i].Type, wantType)
		}
		payload, ok := domainEvents[i].Payload.(eventbus.ContentEventPayload)
		if !ok {
			t.Fatalf("domain event %d payload type %T", i, domainEvents[i].Payload)
		}
		if payload.ActorID != 9 || payload.ContentID != page.ID || payload.Slug != page.Slug {
			t.Errorf("unexpected domain payload: %+v", payload)
		}
	}

	if len(writer.events) != 3 {
		t.Fatalf("expected 3 audit events, got %d", len(writer.events))
	}
	wantActions := []string{"content.publish", "content.unpublish", "content.rollback"}
	for i, wantAction := range wantActions {
		event := writer.events[i]
		if event.Action != wantAction || event.Result != "success" {
			t.Errorf("audit event %d=%+v, want action=%s success", i, event, wantAction)
		}
		if event.Actor != "publisher" || event.Resource != fmt.Sprintf("pages:%d", page.ID) {
			t.Errorf("unexpected audit identity: %+v", event)
		}
		if event.Details["request_id"] != "req-publish" {
			t.Errorf("missing request metadata: %+v", event.Details)
		}
	}
}

func TestUnifiedPageService_FailureIsAuditedWithoutDomainEvent(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	bus := eventbus.New()
	writer := &serviceAuditWriterStub{}
	svc := service.NewUnifiedPageService(pageRepo, versionRepo, bus).WithAuditWriter(writer)

	publishedEvents := 0
	bus.Subscribe(eventbus.ContentPublished, eventbus.SyncHandler(func(eventbus.Event) {
		publishedEvents++
	}))

	page := &model.UnifiedPage{
		Slug:         "conflict-page",
		Mode:         "composable",
		DraftVersion: 2,
		DraftConfig:  model.JSONMap{"sections": []any{}},
	}
	if err := pageRepo.Create(context.Background(), page); err != nil {
		t.Fatalf("create page: %v", err)
	}

	err := svc.Publish(context.Background(), page.ID, 1, 9)
	if !errors.Is(err, service.ErrPageVersionConflict) {
		t.Fatalf("publish error=%v, want version conflict", err)
	}
	if publishedEvents != 0 {
		t.Fatalf("expected no published event, got %d", publishedEvents)
	}
	if len(writer.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(writer.events))
	}
	if writer.events[0].Result != "failure" || writer.events[0].Details["reason"] != "draft version conflict" {
		t.Errorf("unexpected failure audit: %+v", writer.events[0])
	}
}

func TestUnifiedPageService_UnpublishNotFoundIsServiceAudited(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	bus := eventbus.New()
	writer := &serviceAuditWriterStub{}
	svc := service.NewUnifiedPageService(pageRepo, versionRepo, bus).WithAuditWriter(writer)

	unpublishedEvents := 0
	bus.Subscribe(eventbus.ContentUnpublished, eventbus.SyncHandler(func(eventbus.Event) {
		unpublishedEvents++
	}))

	page, err := svc.Unpublish(context.Background(), 999, 9)
	if !errors.Is(err, service.ErrUnifiedPageNotFound) {
		t.Fatalf("unpublish error=%v, want not found", err)
	}
	if page != nil {
		t.Fatalf("page=%+v, want nil", page)
	}
	if unpublishedEvents != 0 {
		t.Fatalf("expected no unpublished event, got %d", unpublishedEvents)
	}
	if len(writer.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(writer.events))
	}
	if writer.events[0].Action != "content.unpublish" || writer.events[0].Result != "failure" {
		t.Errorf("unexpected failure audit: %+v", writer.events[0])
	}
}

func TestUnifiedPageService_AuditWriteFailureDoesNotFailPublish(t *testing.T) {
	db := setupServiceTestDB(t)
	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	writer := &serviceAuditWriterStub{err: errors.New("audit unavailable")}
	svc := service.NewUnifiedPageService(pageRepo, versionRepo).WithAuditWriter(writer)

	page := &model.UnifiedPage{
		Slug:         "best-effort-audit",
		Mode:         "composable",
		DraftVersion: 1,
		DraftConfig:  model.JSONMap{"sections": []any{}},
	}
	if err := pageRepo.Create(context.Background(), page); err != nil {
		t.Fatalf("create page: %v", err)
	}

	if err := svc.Publish(context.Background(), page.ID, 1, 9); err != nil {
		t.Fatalf("publish should ignore audit failure: %v", err)
	}
	if len(writer.events) != 1 {
		t.Fatalf("expected attempted audit write, got %d", len(writer.events))
	}
}
