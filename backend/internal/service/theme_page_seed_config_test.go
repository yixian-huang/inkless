package service

import (
	"context"
	"testing"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"gorm.io/gorm"
)

// stubUnifiedPageRepo is a minimal in-memory UnifiedPageRepository for seed tests.
type stubUnifiedPageRepo struct {
	bySlug  map[string]*model.UnifiedPage
	nextID  uint
	created []*model.UnifiedPage
	updated []*model.UnifiedPage
}

func newStubUnifiedPageRepo() *stubUnifiedPageRepo {
	return &stubUnifiedPageRepo{
		bySlug: make(map[string]*model.UnifiedPage),
		nextID: 1,
	}
}

func (s *stubUnifiedPageRepo) Create(_ context.Context, page *model.UnifiedPage) error {
	if page.ID == 0 {
		page.ID = s.nextID
		s.nextID++
	}
	cp := *page
	// shallow-clone maps so later mutations in caller don't affect stored page unexpectedly
	if page.DraftConfig != nil {
		cp.DraftConfig = cloneJSONMap(page.DraftConfig)
	}
	if page.PublishedConfig != nil {
		cp.PublishedConfig = model.NullableJSONMap(cloneJSONMap(model.JSONMap(page.PublishedConfig)))
	}
	s.bySlug[page.Slug] = &cp
	s.created = append(s.created, &cp)
	return nil
}

func (s *stubUnifiedPageRepo) Update(_ context.Context, page *model.UnifiedPage) error {
	if _, ok := s.bySlug[page.Slug]; !ok {
		return gorm.ErrRecordNotFound
	}
	cp := *page
	if page.DraftConfig != nil {
		cp.DraftConfig = cloneJSONMap(page.DraftConfig)
	}
	if page.PublishedConfig != nil {
		cp.PublishedConfig = model.NullableJSONMap(cloneJSONMap(model.JSONMap(page.PublishedConfig)))
	} else {
		cp.PublishedConfig = nil
	}
	s.bySlug[page.Slug] = &cp
	s.updated = append(s.updated, &cp)
	return nil
}

func (s *stubUnifiedPageRepo) Delete(context.Context, uint) error { return nil }

func (s *stubUnifiedPageRepo) FindByID(_ context.Context, id uint) (*model.UnifiedPage, error) {
	for _, p := range s.bySlug {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubUnifiedPageRepo) FindBySlug(_ context.Context, slug string) (*model.UnifiedPage, error) {
	if p, ok := s.bySlug[slug]; ok {
		return p, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubUnifiedPageRepo) List(context.Context, string, string, *uint) ([]*model.UnifiedPage, error) {
	return nil, nil
}
func (s *stubUnifiedPageRepo) ListPublished(context.Context) ([]*model.UnifiedPage, error) {
	return nil, nil
}
func (s *stubUnifiedPageRepo) Count(context.Context) (int64, error) { return 0, nil }
func (s *stubUnifiedPageRepo) UpdateDraft(context.Context, uint, int, model.JSONMap) (int, error) {
	return 0, nil
}
func (s *stubUnifiedPageRepo) PublishDraft(context.Context, uint, int, uint, time.Time, bool) (*model.UnifiedPage, int, bool, error) {
	return nil, 0, false, nil
}
func (s *stubUnifiedPageRepo) UpdatePublished(context.Context, uint, model.JSONMap, int, time.Time) error {
	return nil
}
func (s *stubUnifiedPageRepo) UpdateRollback(context.Context, uint, model.JSONMap, int, model.JSONMap, int, time.Time) error {
	return nil
}
func (s *stubUnifiedPageRepo) ClearPublished(context.Context, uint) error { return nil }
func (s *stubUnifiedPageRepo) UpdateSortOrder(context.Context, uint, int) error {
	return nil
}

func TestPublishedConfigHasSections(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  model.NullableJSONMap
		want bool
	}{
		{"nil config", nil, false},
		{"empty map", model.NullableJSONMap{}, false},
		{"sections missing", model.NullableJSONMap{"title": "x"}, false},
		{"sections null", model.NullableJSONMap{"sections": nil}, false},
		{"sections empty array", model.NullableJSONMap{"sections": []interface{}{}}, false},
		{"sections with items", model.NullableJSONMap{"sections": []interface{}{
			map[string]interface{}{"id": "a", "type": "hero"},
		}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := publishedConfigHasSections(tc.cfg); got != tc.want {
				t.Fatalf("publishedConfigHasSections() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestApplyEditorialFirmPageSeeds_CreatesMissingPages(t *testing.T) {
	repo := newStubUnifiedPageRepo()
	if err := ApplyEditorialFirmPageSeeds(context.Background(), repo); err != nil {
		t.Fatalf("ApplyEditorialFirmPageSeeds: %v", err)
	}
	if len(repo.created) != 4 {
		t.Fatalf("expected 4 created pages, got %d", len(repo.created))
	}
	for _, slug := range editorialFirmSeedSlugs {
		p, err := repo.FindBySlug(context.Background(), slug)
		if err != nil {
			t.Fatalf("FindBySlug %s: %v", slug, err)
		}
		if p.Status != "published" {
			t.Errorf("%s: status = %q, want published", slug, p.Status)
		}
		if !publishedConfigHasSections(p.PublishedConfig) {
			t.Errorf("%s: expected non-empty published sections", slug)
		}
		if !publishedConfigHasSections(model.NullableJSONMap(p.DraftConfig)) {
			t.Errorf("%s: expected non-empty draft sections", slug)
		}
		// Stable seed ids for home
		if slug == "home" {
			secs, _ := p.PublishedConfig["sections"].([]interface{})
			if len(secs) < 4 {
				t.Errorf("home: expected ≥4 sections, got %d", len(secs))
			}
			first, _ := secs[0].(map[string]interface{})
			if first["type"] != "ef-hero-editorial" {
				t.Errorf("home first section type = %v, want ef-hero-editorial", first["type"])
			}
		}
	}
}

func TestApplyEditorialFirmPageSeeds_EmptyPageGetsSeed(t *testing.T) {
	repo := newStubUnifiedPageRepo()
	// Pre-create home with empty / missing sections.
	_ = repo.Create(context.Background(), &model.UnifiedPage{
		Slug:            "home",
		ZhTitle:         "首页",
		EnTitle:         "Home",
		Mode:            model.PageModeComposable,
		DraftConfig:     model.JSONMap{"sections": []interface{}{}},
		DraftVersion:    1,
		PublishedConfig: model.NullableJSONMap{"sections": []interface{}{}},
		Status:          "draft",
	})
	// Pre-create about with nil published config.
	_ = repo.Create(context.Background(), &model.UnifiedPage{
		Slug:            "about",
		ZhTitle:         "关于",
		EnTitle:         "About",
		Mode:            model.PageModeComposable,
		DraftConfig:     model.JSONMap{},
		DraftVersion:    1,
		PublishedConfig: nil,
		Status:          "draft",
	})

	if err := ApplyEditorialFirmPageSeeds(context.Background(), repo); err != nil {
		t.Fatalf("ApplyEditorialFirmPageSeeds: %v", err)
	}

	home, _ := repo.FindBySlug(context.Background(), "home")
	if !publishedConfigHasSections(home.PublishedConfig) {
		t.Fatal("home: expected seed sections after apply")
	}
	if home.Status != "published" {
		t.Fatalf("home: status = %q, want published", home.Status)
	}
	about, _ := repo.FindBySlug(context.Background(), "about")
	if !publishedConfigHasSections(about.PublishedConfig) {
		t.Fatal("about: expected seed sections after apply")
	}

	// services + contact were missing → created
	for _, slug := range []string{"services", "contact"} {
		p, err := repo.FindBySlug(context.Background(), slug)
		if err != nil {
			t.Fatalf("%s should have been created: %v", slug, err)
		}
		if !publishedConfigHasSections(p.PublishedConfig) {
			t.Errorf("%s: missing seed sections", slug)
		}
	}
}

func TestApplyEditorialFirmPageSeeds_DoesNotOverwriteExistingSections(t *testing.T) {
	repo := newStubUnifiedPageRepo()
	custom := model.NullableJSONMap{
		"sections": []interface{}{
			map[string]interface{}{
				"id":   "custom-hero",
				"type": "hero",
				"data": map[string]interface{}{"title": "Keep me"},
			},
		},
	}
	_ = repo.Create(context.Background(), &model.UnifiedPage{
		Slug:             "home",
		ZhTitle:          "首页",
		EnTitle:          "Home",
		Mode:             model.PageModeComposable,
		DraftConfig:      model.JSONMap(custom),
		DraftVersion:     2,
		PublishedConfig:  custom,
		PublishedVersion: 2,
		Status:           "published",
	})
	// Empty about should still get seed.
	_ = repo.Create(context.Background(), &model.UnifiedPage{
		Slug:            "about",
		Mode:            model.PageModeComposable,
		DraftConfig:     model.JSONMap{"sections": []interface{}{}},
		PublishedConfig: model.NullableJSONMap{"sections": []interface{}{}},
		Status:          "draft",
	})

	if err := ApplyEditorialFirmPageSeeds(context.Background(), repo); err != nil {
		t.Fatalf("ApplyEditorialFirmPageSeeds: %v", err)
	}

	home, _ := repo.FindBySlug(context.Background(), "home")
	secs, _ := home.PublishedConfig["sections"].([]interface{})
	if len(secs) != 1 {
		t.Fatalf("home: expected 1 custom section preserved, got %d", len(secs))
	}
	first, _ := secs[0].(map[string]interface{})
	if first["id"] != "custom-hero" {
		t.Fatalf("home: section overwritten, got id=%v", first["id"])
	}
	// home should not appear in updated list for overwrite
	for _, u := range repo.updated {
		if u.Slug == "home" {
			// Update might still not have been called — if it was, fail.
			t.Fatal("home with existing sections must not be updated")
		}
	}

	about, _ := repo.FindBySlug(context.Background(), "about")
	if !publishedConfigHasSections(about.PublishedConfig) {
		t.Fatal("about empty page should receive seed")
	}
}

func TestApplyEditorialFirmPageSeeds_NilRepo(t *testing.T) {
	if err := ApplyEditorialFirmPageSeeds(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil repo")
	}
}

func TestLoadEditorialFirmPageSeeds_HasFourPages(t *testing.T) {
	if len(editorialFirmPageSeeds) != 4 {
		t.Fatalf("expected 4 seed pages, got %d keys: %v", len(editorialFirmPageSeeds), keysOf(editorialFirmPageSeeds))
	}
	for _, slug := range editorialFirmSeedSlugs {
		cfg, ok := editorialFirmPageSeeds[slug]
		if !ok {
			t.Fatalf("missing seed for %s", slug)
		}
		if !publishedConfigHasSections(model.NullableJSONMap(cfg)) {
			t.Fatalf("%s seed has no sections", slug)
		}
	}
}

func keysOf(m map[string]model.JSONMap) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
