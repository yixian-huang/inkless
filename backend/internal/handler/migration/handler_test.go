package migration

import (
	"bytes"
	"context"
	"errors"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	migrationPkg "github.com/yixian-huang/inkless/backend/internal/migration"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/provider"
)

type handlerArticleRepoStub struct {
	mu       sync.Mutex
	failures map[string]int
	block    chan struct{}
	entered  chan struct{}
}

func (r *handlerArticleRepoStub) Create(_ context.Context, article *model.Article) error {
	if r.entered != nil {
		select {
		case r.entered <- struct{}{}:
		default:
		}
	}
	if r.block != nil {
		<-r.block
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failures[article.Slug] > 0 {
		r.failures[article.Slug]--
		return errors.New("create failed")
	}
	article.ID = 1
	return nil
}

func (r *handlerArticleRepoStub) FindByID(context.Context, uint) (*model.Article, error) {
	return nil, nil
}

func (r *handlerArticleRepoStub) FindBySlug(context.Context, string) (*model.Article, error) {
	return nil, nil
}

func (r *handlerArticleRepoStub) Update(context.Context, *model.Article) error {
	return nil
}

func (r *handlerArticleRepoStub) UpdateScheduledPublication(context.Context, *model.Article, time.Time) error {
	return nil
}

func (r *handlerArticleRepoStub) Delete(context.Context, uint) error {
	return nil
}

func (r *handlerArticleRepoStub) List(context.Context, int, int, string, *uint, *uint) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

func (r *handlerArticleRepoStub) ListPublished(context.Context, int, int, string, string) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

func (r *handlerArticleRepoStub) Count(context.Context, string) (int64, error) {
	return 0, nil
}

type handlerCategoryRepoStub struct{}

func (r *handlerCategoryRepoStub) Create(_ context.Context, category *model.Category) error {
	category.ID = 1
	return nil
}

func (r *handlerCategoryRepoStub) FindByID(context.Context, uint) (*model.Category, error) {
	return nil, nil
}

func (r *handlerCategoryRepoStub) FindBySlug(context.Context, string) (*model.Category, error) {
	return nil, nil
}

func (r *handlerCategoryRepoStub) Update(context.Context, *model.Category) error {
	return nil
}

func (r *handlerCategoryRepoStub) Delete(context.Context, uint) error {
	return nil
}

func (r *handlerCategoryRepoStub) List(context.Context) ([]*model.Category, error) {
	return nil, nil
}

func (r *handlerCategoryRepoStub) ListTree(context.Context) ([]*model.Category, error) {
	return nil, nil
}

func (r *handlerCategoryRepoStub) ListByParentID(context.Context, *uint) ([]*model.Category, error) {
	return nil, nil
}

func (r *handlerCategoryRepoStub) FindByIDs(context.Context, []uint) ([]model.Category, error) {
	return nil, nil
}

type handlerTagRepoStub struct{}

func (r *handlerTagRepoStub) Create(_ context.Context, tag *model.Tag) error {
	tag.ID = 1
	return nil
}

func (r *handlerTagRepoStub) FindByID(context.Context, uint) (*model.Tag, error) {
	return nil, nil
}

func (r *handlerTagRepoStub) FindBySlug(context.Context, string) (*model.Tag, error) {
	return nil, nil
}

func (r *handlerTagRepoStub) Update(context.Context, *model.Tag) error {
	return nil
}

func (r *handlerTagRepoStub) Delete(context.Context, uint) error {
	return nil
}

func (r *handlerTagRepoStub) List(context.Context) ([]*model.Tag, error) {
	return nil, nil
}

func (r *handlerTagRepoStub) FindByIDs(context.Context, []uint) ([]model.Tag, error) {
	return nil, nil
}

type emptyMigrationProvider struct{}

func (emptyMigrationProvider) Source() provider.MigrationSource {
	return provider.SourceMarkdown
}

func (emptyMigrationProvider) Parse(context.Context, io.Reader) (*provider.MigrationResult, error) {
	return &provider.MigrationResult{
		Errors: []string{"no supported markdown files found"},
	}, nil
}

func TestImportRejectsExportWithoutArticles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := migrationPkg.NewService(
		&handlerArticleRepoStub{failures: map[string]int{}},
		&handlerCategoryRepoStub{},
		&handlerTagRepoStub{},
	)
	handler := NewHandler(service)
	handler.providers[provider.SourceMarkdown] = emptyMigrationProvider{}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("source", string(provider.SourceMarkdown)))
	file, err := writer.CreateFormFile("file", "empty.zip")
	require.NoError(t, err)
	_, err = file.Write([]byte("empty"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	router := gin.New()
	router.POST("/admin/migration/import", handler.Import)
	request := httptest.NewRequest(http.MethodPost, "/admin/migration/import", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Contains(t, rec.Body.String(), "no articles found in export file")
	require.Contains(t, rec.Body.String(), "no supported markdown files found")
	require.Empty(t, service.ListJobs())
}

func TestRetryJobRejectsMissingRunningAndNonFailedJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerArticleRepoStub{failures: map[string]int{}, block: make(chan struct{}), entered: make(chan struct{}, 1)}
	service := migrationPkg.NewService(repo, &handlerCategoryRepoStub{}, &handlerTagRepoStub{})
	handler := NewHandler(service)
	router := gin.New()
	router.POST("/admin/migration/jobs/:jobId/retry", handler.RetryJob)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/migration/jobs/missing/retry", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "migration job not found")

	runningID := service.StartImport(context.Background(), provider.SourceMarkdown, []*provider.MigrationArticle{
		{Slug: "running", Title: "Running", Status: model.ArticleStatusPublished},
	}, nil)
	select {
	case <-repo.entered:
	case <-time.After(time.Second):
		t.Fatal("import did not start")
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/migration/jobs/"+runningID+"/retry", nil))
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "still running")
	close(repo.block)
	waitHandlerPhase(t, service, runningID, "done")

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/migration/jobs/"+runningID+"/retry", nil))
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "only failed")
}

func TestRetryJobAcceptedForFailedJobAndStreamSendsInitialTerminalState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := migrationPkg.NewService(
		&handlerArticleRepoStub{failures: map[string]int{"bad": 1}},
		&handlerCategoryRepoStub{},
		&handlerTagRepoStub{},
	)
	handler := NewHandler(service)
	router := gin.New()
	router.POST("/admin/migration/jobs/:jobId/retry", handler.RetryJob)
	router.GET("/admin/migration/jobs/:jobId/stream", handler.StreamProgress)

	jobID := service.StartImport(context.Background(), provider.SourceMarkdown, []*provider.MigrationArticle{
		{Slug: "bad", Title: "Bad", Status: model.ArticleStatusPublished},
	}, nil)
	waitHandlerPhase(t, service, jobID, "failed")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/migration/jobs/"+jobID+"/stream", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "event: progress")
	require.Contains(t, rec.Body.String(), `"phase":"failed"`)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/migration/jobs/"+jobID+"/retry", nil))
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Body.String(), `"attempt":2`)
}

func waitHandlerPhase(t *testing.T, service *migrationPkg.Service, jobID, phase string) *provider.MigrationProgress {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		progress, ok := service.GetProgress(jobID)
		require.True(t, ok)
		if progress.Phase == phase {
			return progress
		}
		time.Sleep(5 * time.Millisecond)
	}
	progress, _ := service.GetProgress(jobID)
	t.Fatalf("job %s did not reach phase %s; last progress: %+v", jobID, phase, progress)
	return nil
}

func (r *handlerArticleRepoStub) UpdateIfMatch(context.Context, *model.Article, time.Time) error {
	return nil
}

func (r *handlerArticleRepoStub) ListFilter(context.Context, repository.ArticleListFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
