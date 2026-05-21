package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/provider"
)

// mockAIProvider is a test-only AI provider that returns a configurable response.
type mockAIProvider struct {
	chatCompleteResponse string
	chatCompleteErr      error
}

func (m *mockAIProvider) ChatComplete(_ context.Context, _, _ string) (string, error) {
	return m.chatCompleteResponse, m.chatCompleteErr
}

func (m *mockAIProvider) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{Content: "mock"}, nil
}

func (m *mockAIProvider) Complete(_ context.Context, _ provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return &provider.CompletionResponse{Text: "mock"}, nil
}

func (m *mockAIProvider) Summarize(_ context.Context, text string, _ int) (string, error) {
	return text, nil
}

func (m *mockAIProvider) SuggestTitles(_ context.Context, _ string, count int) ([]string, error) {
	titles := make([]string, count)
	for i := range titles {
		titles[i] = "Mock Title"
	}
	return titles, nil
}

func (m *mockAIProvider) SuggestTags(_ context.Context, _ string, _ []string) ([]string, error) {
	return []string{"mock-tag"}, nil
}

func (m *mockAIProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.ChatChunk, error) {
	ch := make(chan provider.ChatChunk, 1)
	ch <- provider.ChatChunk{Content: "mock stream"}
	close(ch)
	return ch, nil
}

func (m *mockAIProvider) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{0.1, 0.2}, nil
}

func (m *mockAIProvider) Name() string { return "mock" }

// mockPageRepo is a simple in-memory PageRepository for testing.
type mockPageRepo struct {
	pages  map[string]*model.Page
	nextID uint
}

func newMockPageRepo() *mockPageRepo {
	return &mockPageRepo{
		pages:  make(map[string]*model.Page),
		nextID: 1,
	}
}

func (r *mockPageRepo) Create(_ context.Context, page *model.Page) error {
	page.ID = r.nextID
	r.nextID++
	r.pages[page.Slug] = page
	return nil
}

func (r *mockPageRepo) Update(_ context.Context, page *model.Page) error {
	r.pages[page.Slug] = page
	return nil
}

func (r *mockPageRepo) Delete(_ context.Context, id uint) error {
	for slug, p := range r.pages {
		if p.ID == id {
			delete(r.pages, slug)
			return nil
		}
	}
	return errors.New("not found")
}

func (r *mockPageRepo) FindByID(_ context.Context, id uint) (*model.Page, error) {
	for _, p := range r.pages {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *mockPageRepo) FindBySlug(_ context.Context, slug string) (*model.Page, error) {
	p, ok := r.pages[slug]
	if !ok {
		return nil, errors.New("not found")
	}
	return p, nil
}

func (r *mockPageRepo) FindByThemeIDAndContentKey(_ context.Context, _, _ string) (*model.Page, error) {
	return nil, errors.New("not found")
}

func (r *mockPageRepo) List(_ context.Context, _ string, _ *uint) ([]*model.Page, error) {
	pages := make([]*model.Page, 0, len(r.pages))
	for _, p := range r.pages {
		pages = append(pages, p)
	}
	return pages, nil
}

func (r *mockPageRepo) ListByThemeID(_ context.Context, _ string, _ string) ([]*model.Page, error) {
	return nil, nil
}

func (r *mockPageRepo) ListPublished(_ context.Context) ([]*model.Page, error) {
	return nil, nil
}

func (r *mockPageRepo) ListPublishedByThemeID(_ context.Context, _ string) ([]*model.Page, error) {
	return nil, nil
}

func (r *mockPageRepo) UpdateSortOrder(_ context.Context, _ uint, _ int) error {
	return nil
}

// --- Tests ---

const validSitePlanJSON = `{
  "recommendedTheme": "modern-dark",
  "rationale": "Tech companies benefit from modern dark themes.",
  "colorScheme": {
    "primary": "#2563EB",
    "secondary": "#7C3AED",
    "background": "#0F172A",
    "text": "#F8FAFC",
    "rationale": "Blue conveys trust, purple adds creativity."
  },
  "pages": [
    {
      "slug": "home",
      "title": {"zh": "首页", "en": "Home"},
      "layout": "hero-cta",
      "sections": ["hero", "features", "cta"],
      "sortOrder": 0
    },
    {
      "slug": "contact",
      "title": {"zh": "联系我们", "en": "Contact"},
      "layout": "simple",
      "sections": ["contact-form"],
      "sortOrder": 10
    }
  ],
  "suggestedContent": [
    {
      "pageSlug": "home",
      "heading": "Build Better Software Faster",
      "subheading": "Expert technology consulting for modern teams",
      "body": "We help companies modernize their tech stack.",
      "ctaText": "Get Started"
    }
  ]
}`

func TestWizardService_GenerateSitePlan_Success(t *testing.T) {
	ai := &mockAIProvider{chatCompleteResponse: validSitePlanJSON}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	q := model.Questionnaire{
		Industry:        "technology",
		StylePreference: "modern",
		BrandName:       "TechCo",
		Locale:          "en",
	}

	plan, err := svc.GenerateSitePlan(context.Background(), q)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "modern-dark", plan.RecommendedTheme)
	assert.Len(t, plan.Pages, 2)
	assert.Equal(t, "home", plan.Pages[0].Slug)
	assert.Equal(t, "contact", plan.Pages[1].Slug)
	assert.Equal(t, "#2563EB", plan.ColorScheme.Primary)
	assert.Len(t, plan.SuggestedContent, 1)
	assert.Equal(t, "Build Better Software Faster", plan.SuggestedContent[0].Heading)
}

func TestWizardService_GenerateSitePlan_MissingIndustry(t *testing.T) {
	ai := &mockAIProvider{chatCompleteResponse: validSitePlanJSON}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	_, err := svc.GenerateSitePlan(context.Background(), model.Questionnaire{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "industry is required")
}

func TestWizardService_GenerateSitePlan_AIError(t *testing.T) {
	ai := &mockAIProvider{chatCompleteErr: errors.New("AI unavailable")}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	_, err := svc.GenerateSitePlan(context.Background(), model.Questionnaire{Industry: "tech"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI plan generation failed")
}

func TestWizardService_GenerateSitePlan_InvalidJSON(t *testing.T) {
	ai := &mockAIProvider{chatCompleteResponse: "not valid json at all"}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	_, err := svc.GenerateSitePlan(context.Background(), model.Questionnaire{Industry: "tech"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse AI site plan")
}

func TestWizardService_GenerateSitePlan_ThemeSanitization(t *testing.T) {
	planJSON := `{
		"recommendedTheme": "WARM_EARTH",
		"colorScheme": {"primary":"#fff","secondary":"#000","background":"#eee","text":"#111","rationale":"ok"},
		"pages": [],
		"suggestedContent": []
	}`
	ai := &mockAIProvider{chatCompleteResponse: planJSON}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	plan, err := svc.GenerateSitePlan(context.Background(), model.Questionnaire{Industry: "wellness"})
	require.NoError(t, err)
	assert.Equal(t, "warm-earth", plan.RecommendedTheme)
}

func TestWizardService_ScaffoldSite_CreatesPages(t *testing.T) {
	ai := &mockAIProvider{}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	plan := model.SitePlan{
		RecommendedTheme: "default",
		ColorScheme: model.ColorScheme{
			Primary: "#333", Secondary: "#666", Background: "#fff", Text: "#000",
		},
		Pages: []model.PagePlan{
			{
				Slug:      "home",
				Title:     map[string]string{"zh": "首页", "en": "Home"},
				Layout:    "hero-cta",
				Sections:  []string{"hero", "features"},
				SortOrder: 0,
			},
			{
				Slug:      "about",
				Title:     map[string]string{"zh": "关于我们", "en": "About"},
				Layout:    "simple",
				Sections:  []string{"hero"},
				SortOrder: 1,
			},
		},
		SuggestedContent: []model.SuggestedContent{
			{
				PageSlug:   "home",
				Heading:    "Welcome",
				Subheading: "subtitle",
				Body:       "body text",
				CTAText:    "Learn More",
			},
		},
	}

	result, err := svc.ScaffoldSite(context.Background(), plan)
	require.NoError(t, err)
	assert.Equal(t, "default", result.AppliedTheme)
	assert.Contains(t, result.CreatedPages, "home")
	assert.Contains(t, result.CreatedPages, "about")
	assert.Empty(t, result.SkippedPages)

	// Verify pages were actually stored
	homePage, err := repo.FindBySlug(context.Background(), "home")
	require.NoError(t, err)
	assert.Equal(t, "首页", homePage.Title["zh"])
}

func TestWizardService_ScaffoldSite_SkipsExistingPages(t *testing.T) {
	ai := &mockAIProvider{}
	repo := newMockPageRepo()

	// Pre-create the "home" page
	_ = repo.Create(context.Background(), &model.Page{
		Slug:   "home",
		Status: model.PageStatusPublished,
	})

	svc := NewWizardService(ai, repo)

	plan := model.SitePlan{
		RecommendedTheme: "default",
		Pages: []model.PagePlan{
			{Slug: "home", Title: map[string]string{"zh": "首页"}, SortOrder: 0},
			{Slug: "contact", Title: map[string]string{"zh": "联系"}, SortOrder: 1},
		},
	}

	result, err := svc.ScaffoldSite(context.Background(), plan)
	require.NoError(t, err)
	assert.Contains(t, result.SkippedPages, "home")
	assert.Contains(t, result.CreatedPages, "contact")
}

func TestWizardService_SuggestColors_Success(t *testing.T) {
	colorJSON := `{"primary":"#2563EB","secondary":"#7C3AED","background":"#FFFFFF","text":"#1F2937","rationale":"Blue for trust"}`
	ai := &mockAIProvider{chatCompleteResponse: colorJSON}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	req := model.ColorSuggestionRequest{
		Industry:  "technology",
		BrandName: "TechCo",
		Locale:    "en",
	}

	scheme, err := svc.SuggestColors(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "#2563EB", scheme.Primary)
	assert.Equal(t, "#7C3AED", scheme.Secondary)
	assert.Equal(t, "Blue for trust", scheme.Rationale)
}

func TestWizardService_SuggestColors_MissingIndustry(t *testing.T) {
	ai := &mockAIProvider{}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	_, err := svc.SuggestColors(context.Background(), model.ColorSuggestionRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "industry is required")
}

func TestWizardService_SuggestColors_MarkdownWrappedJSON(t *testing.T) {
	markdownWrapped := "```json\n{\"primary\":\"#FF0000\",\"secondary\":\"#00FF00\",\"background\":\"#FFFFFF\",\"text\":\"#000000\",\"rationale\":\"test\"}\n```"
	ai := &mockAIProvider{chatCompleteResponse: markdownWrapped}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	scheme, err := svc.SuggestColors(context.Background(), model.ColorSuggestionRequest{Industry: "retail"})
	require.NoError(t, err)
	assert.Equal(t, "#FF0000", scheme.Primary)
}

func TestWizardService_GenerateContent_Success(t *testing.T) {
	contentJSON := `{"pageSlug":"home","heading":"Welcome","subheading":"Sub","body":"Body text.","ctaText":"Click Here"}`
	ai := &mockAIProvider{chatCompleteResponse: contentJSON}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	req := model.GenerateContentRequest{
		PageType:  "home",
		Industry:  "consulting",
		BrandName: "TestBrand",
		Locale:    "en",
	}

	content, err := svc.GenerateContent(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "home", content.PageSlug)
	assert.Equal(t, "Welcome", content.Heading)
	assert.Equal(t, "Click Here", content.CTAText)
}

func TestWizardService_GenerateContent_MissingRequiredFields(t *testing.T) {
	ai := &mockAIProvider{}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	_, err := svc.GenerateContent(context.Background(), model.GenerateContentRequest{PageType: "home"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "industry is required")

	_, err = svc.GenerateContent(context.Background(), model.GenerateContentRequest{Industry: "tech"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pageType is required")
}

func TestWizardService_GenerateContent_FallsBackSlug(t *testing.T) {
	// pageSlug omitted from JSON — should fall back to req.PageType
	contentJSON := `{"heading":"H","subheading":"S","body":"B","ctaText":"C"}`
	ai := &mockAIProvider{chatCompleteResponse: contentJSON}
	repo := newMockPageRepo()
	svc := NewWizardService(ai, repo)

	content, err := svc.GenerateContent(context.Background(), model.GenerateContentRequest{
		PageType: "about",
		Industry: "consulting",
	})
	require.NoError(t, err)
	assert.Equal(t, "about", content.PageSlug)
}

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"key":"value"}`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "markdown fenced",
			input:    "```json\n{\"key\":\"value\"}\n```",
			expected: `{"key":"value"}`,
		},
		{
			name:     "JSON with surrounding text",
			input:    "Here is the result: {\"key\":\"value\"} — end",
			expected: `{"key":"value"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractJSON(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestSanitizeThemeName(t *testing.T) {
	assert.Equal(t, "default", sanitizeThemeName("unknown"))
	assert.Equal(t, "default", sanitizeThemeName(""))
	assert.Equal(t, "default", sanitizeThemeName("default"))
	assert.Equal(t, "modern-dark", sanitizeThemeName("modern-dark"))
	assert.Equal(t, "modern-dark", sanitizeThemeName("MODERN-DARK"))
	assert.Equal(t, "modern-dark", sanitizeThemeName("modern_dark"))
	assert.Equal(t, "warm-earth", sanitizeThemeName("warm-earth"))
	assert.Equal(t, "warm-earth", sanitizeThemeName("WARM_EARTH"))
}
