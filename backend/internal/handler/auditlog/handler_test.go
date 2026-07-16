package auditlog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blotting-consultancy/internal/model"

	"github.com/gin-gonic/gin"
)

type auditEventRepositoryStub struct {
	listFunc func(
		ctx context.Context,
		offset, limit int,
		action, actor string,
		from, to *time.Time,
	) ([]model.AuditEvent, int64, error)
}

func (r *auditEventRepositoryStub) Create(context.Context, *model.AuditEvent) error {
	return nil
}

func (r *auditEventRepositoryStub) List(
	ctx context.Context,
	offset, limit int,
	action, actor string,
	from, to *time.Time,
) ([]model.AuditEvent, int64, error) {
	return r.listFunc(ctx, offset, limit, action, actor, from, to)
}

func TestList_ForwardsFiltersAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &auditEventRepositoryStub{
		listFunc: func(
			_ context.Context,
			offset, limit int,
			action, actor string,
			from, to *time.Time,
		) ([]model.AuditEvent, int64, error) {
			if offset != 10 || limit != 10 {
				t.Fatalf("offset/limit=%d/%d, want 10/10", offset, limit)
			}
			if action != "content.publish" || actor != "publisher" {
				t.Fatalf("action/actor=%q/%q", action, actor)
			}
			if from == nil || from.Format(time.RFC3339Nano) != "2026-07-01T00:00:00Z" {
				t.Fatalf("unexpected from: %v", from)
			}
			if to == nil || to.Format(time.RFC3339Nano) != "2026-07-01T23:59:59.999999999Z" {
				t.Fatalf("unexpected to: %v", to)
			}
			return []model.AuditEvent{{ID: 11, Action: action, Actor: actor}}, 21, nil
		},
	}

	router := gin.New()
	router.GET("/admin/audit-logs", NewHandler(repo).List)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/audit-logs?page=2&pageSize=10&action=content.publish&actor=publisher&from=2026-07-01&to=2026-07-01",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items    []model.AuditEvent `json:"items"`
		Total    int64              `json:"total"`
		Page     int                `json:"page"`
		PageSize int                `json:"pageSize"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Total != 21 || response.Page != 2 || response.PageSize != 10 || len(response.Items) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestList_RejectsInvalidDateFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	repo := &auditEventRepositoryStub{
		listFunc: func(
			context.Context,
			int,
			int,
			string,
			string,
			*time.Time,
			*time.Time,
		) ([]model.AuditEvent, int64, error) {
			called = true
			return nil, 0, nil
		},
	}
	router := gin.New()
	router.GET("/admin/audit-logs", NewHandler(repo).List)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit-logs?from=not-a-date", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("repository should not be called for invalid filters")
	}
}
