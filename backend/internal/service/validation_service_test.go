package service

import (
	"blotting-consultancy/internal/model"
	"testing"
)

func TestValidateConfig_HomePage_Valid(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "欢迎",
				"en": "Welcome",
			},
			"subtitle": map[string]interface{}{
				"zh": "副标题",
				"en": "Subtitle",
			},
			"backgroundImage": map[string]interface{}{
				"url": "/images/hero.jpg",
				"alt": map[string]interface{}{
					"zh": "背景图",
					"en": "Background",
				},
			},
		},
		"about": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "关于我们",
				"en": "About Us",
			},
			"descriptions": []interface{}{
				map[string]interface{}{
					"zh": "描述1",
					"en": "Description 1",
				},
			},
			"image": map[string]interface{}{
				"url": "/images/about.jpg",
				"alt": map[string]interface{}{
					"zh": "关于图",
					"en": "About Image",
				},
			},
			"cta": map[string]interface{}{
				"label": map[string]interface{}{
					"zh": "了解更多",
					"en": "Learn More",
				},
				"href": "/about",
			},
		},
		"advantages": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "优势",
				"en": "Advantages",
			},
			"cards": []interface{}{
				map[string]interface{}{
					"title": map[string]interface{}{
						"zh": "优势1",
						"en": "Advantage 1",
					},
					"titleEn": map[string]interface{}{
						"zh": "优势1英文",
						"en": "Advantage 1 EN",
					},
					"description": map[string]interface{}{
						"zh": "描述",
						"en": "Description",
					},
					"image": map[string]interface{}{
						"url": "/images/adv1.jpg",
						"alt": map[string]interface{}{
							"zh": "优势图",
							"en": "Advantage Image",
						},
					},
				},
			},
		},
		"coreServices": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "核心服务",
				"en": "Core Services",
			},
			"items": []interface{}{
				map[string]interface{}{
					"title": map[string]interface{}{
						"zh": "服务1",
						"en": "Service 1",
					},
					"description": map[string]interface{}{
						"zh": "服务描述",
						"en": "Service Description",
					},
					"image": map[string]interface{}{
						"url": "/images/service1.jpg",
						"alt": map[string]interface{}{
							"zh": "服务图",
							"en": "Service Image",
						},
					},
					"cta": map[string]interface{}{
						"label": map[string]interface{}{
							"zh": "了解更多",
							"en": "Learn More",
						},
						"href": "/service",
					},
				},
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyHome, config)

	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d errors", len(result.Errors))
	}

	// Check translation states
	if result.TranslationStatus["hero.title"] != TranslationStateDone {
		t.Errorf("Expected hero.title to be done, got %s", result.TranslationStatus["hero.title"])
	}
}

func TestValidateConfig_HomePage_MissingSection(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "欢迎",
				"en": "Welcome",
			},
			"subtitle": map[string]interface{}{
				"zh": "副标题",
				"en": "Subtitle",
			},
			"backgroundImage": map[string]interface{}{
				"url": "/images/hero.jpg",
				"alt": map[string]interface{}{
					"zh": "背景图",
					"en": "Background",
				},
			},
		},
		// Missing about, advantages, coreServices
	}

	result := vs.ValidateConfig(model.PageKeyHome, config)

	if result.Valid {
		t.Error("Expected invalid config for missing sections")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors")
	}

	// Should have errors for missing sections
	hasAboutError := false
	for _, err := range result.Errors {
		if err.Path == "about" {
			hasAboutError = true
		}
	}
	if !hasAboutError {
		t.Error("Expected error for missing about section")
	}
}

func TestValidateConfig_MissingTranslation(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"label": map[string]interface{}{
				"zh": "标签",
				"en": "", // Missing English
			},
			"title": map[string]interface{}{
				"zh": "标题",
				"en": "Title",
			},
			"image": map[string]interface{}{
				"url": "/images/hero.jpg",
				"alt": map[string]interface{}{
					"zh": "背景图",
					"en": "Background",
				},
			},
		},
		"companyProfile": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "公司简介",
				"en": "Company Profile",
			},
			"description": map[string]interface{}{
				"zh": "描述",
				"en": "Description",
			},
		},
		"blocks": []interface{}{},
	}

	result := vs.ValidateConfig(model.PageKeyAbout, config)

	if result.Valid {
		t.Error("Expected invalid config for missing translation")
	}

	// Check translation status
	if result.TranslationStatus["hero.label"] != TranslationStateMissing {
		t.Errorf("Expected hero.label to be missing, got %s", result.TranslationStatus["hero.label"])
	}

	// Should have error for missing English
	hasEnError := false
	for _, err := range result.Errors {
		if err.Path == "hero.label.en" && err.Code == "REQUIRED" {
			hasEnError = true
		}
	}
	if !hasEnError {
		t.Error("Expected REQUIRED error for hero.label.en")
	}
}

func TestValidateConfig_InvalidPageKey(t *testing.T) {
	vs := NewValidationService()

	result := vs.ValidateConfig(model.PageKey("invalid"), model.JSONMap{})

	if result.Valid {
		t.Error("Expected invalid result for invalid page key")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation error")
	}

	if result.Errors[0].Code != "INVALID_PAGE_KEY" {
		t.Errorf("Expected INVALID_PAGE_KEY error, got %s", result.Errors[0].Code)
	}
}

func TestValidateConfig_AdvantagesPage_Valid(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"label": map[string]interface{}{
				"zh": "我们的优势",
				"en": "Our Advantages",
			},
			"title": map[string]interface{}{
				"zh": "专业优势",
				"en": "Professional Advantages",
			},
			"image": map[string]interface{}{
				"url": "/images/advantages-hero.jpg",
				"alt": map[string]interface{}{
					"zh": "优势背景",
					"en": "Advantages Background",
				},
			},
		},
		"blocks": []interface{}{
			map[string]interface{}{
				"title": map[string]interface{}{
					"zh": "优势1",
					"en": "Advantage 1",
				},
				"description": map[string]interface{}{
					"zh": "描述1",
					"en": "Description 1",
				},
				"image": map[string]interface{}{
					"url": "/images/adv1.jpg",
					"alt": map[string]interface{}{
						"zh": "优势图1",
						"en": "Advantage Image 1",
					},
				},
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyAdvantages, config)

	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestValidateConfig_CoreServicesPage_Valid(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"label": map[string]interface{}{
				"zh": "核心服务",
				"en": "Core Services",
			},
			"title": map[string]interface{}{
				"zh": "我们的服务",
				"en": "Our Services",
			},
			"image": map[string]interface{}{
				"url": "/images/services-hero.jpg",
				"alt": map[string]interface{}{
					"zh": "服务背景",
					"en": "Services Background",
				},
			},
		},
		"services": []interface{}{
			map[string]interface{}{
				"title": map[string]interface{}{
					"zh": "服务1",
					"en": "Service 1",
				},
				"description": map[string]interface{}{
					"zh": "服务描述",
					"en": "Service Description",
				},
				"image": map[string]interface{}{
					"url": "/images/service1.jpg",
					"alt": map[string]interface{}{
						"zh": "服务图",
						"en": "Service Image",
					},
				},
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyCoreServices, config)

	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestValidateConfig_CasesPage_Valid(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"label": map[string]interface{}{
				"zh": "成功案例",
				"en": "Success Cases",
			},
			"title": map[string]interface{}{
				"zh": "我们的案例",
				"en": "Our Cases",
			},
			"image": map[string]interface{}{
				"url": "/images/cases-hero.jpg",
				"alt": map[string]interface{}{
					"zh": "案例背景",
					"en": "Cases Background",
				},
			},
		},
		"cases": []interface{}{
			map[string]interface{}{
				"title": map[string]interface{}{
					"zh": "案例1",
					"en": "Case 1",
				},
				"items": []interface{}{
					map[string]interface{}{
						"zh": "案例项1",
						"en": "Case Item 1",
					},
				},
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyCases, config)

	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestValidateConfig_ExpertsPage_Valid(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"label": map[string]interface{}{
				"zh": "专家团队",
				"en": "Expert Team",
			},
			"title": map[string]interface{}{
				"zh": "我们的专家",
				"en": "Our Experts",
			},
			"image": map[string]interface{}{
				"url": "/images/experts-hero.jpg",
				"alt": map[string]interface{}{
					"zh": "专家背景",
					"en": "Experts Background",
				},
			},
		},
		"sectionTitle": map[string]interface{}{
			"zh": "专家介绍",
			"en": "Expert Introduction",
		},
		"experts": []interface{}{
			map[string]interface{}{
				"id": "expert1",
				"name": map[string]interface{}{
					"zh": "张三",
					"en": "Zhang San",
				},
				"title": map[string]interface{}{
					"zh": "高级顾问",
					"en": "Senior Consultant",
				},
				"avatar": map[string]interface{}{
					"url": "/images/expert1.jpg",
					"alt": map[string]interface{}{
						"zh": "张三头像",
						"en": "Zhang San Avatar",
					},
				},
				"bioParagraphs": []interface{}{
					map[string]interface{}{
						"zh": "简介段落1",
						"en": "Bio paragraph 1",
					},
				},
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyExperts, config)

	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestValidateConfig_ContactPage_Valid(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "联系我们",
				"en": "Contact Us",
			},
			"subtitle": map[string]interface{}{
				"zh": "欢迎咨询",
				"en": "Welcome to inquire",
			},
			"backgroundColor": "#f0f0f0",
		},
		"form": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "表单标题",
				"en": "Form Title",
			},
			"subtitle": map[string]interface{}{
				"zh": "表单副标题",
				"en": "Form Subtitle",
			},
			"submitLabel": map[string]interface{}{
				"zh": "提交",
				"en": "Submit",
			},
		},
		"contactInfo": map[string]interface{}{
			"phone": map[string]interface{}{
				"zh": "电话: 123456",
				"en": "Phone: 123456",
			},
			"address": map[string]interface{}{
				"zh": "地址: xxx",
				"en": "Address: xxx",
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyContact, config)

	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestValidateConfig_GlobalPage_Valid(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"branding": map[string]interface{}{
			"logo": map[string]interface{}{
				"url": "/images/logo.png",
				"alt": map[string]interface{}{
					"zh": "公司Logo",
					"en": "Company Logo",
				},
			},
			"companyName": map[string]interface{}{
				"zh": "测试站点",
				"en": "Test Site",
			},
		},
		"nav": map[string]interface{}{
			"items": []interface{}{},
		},
		"footer": map[string]interface{}{
			"address": map[string]interface{}{
				"zh": "地址",
				"en": "Address",
			},
			"phone": map[string]interface{}{
				"zh": "电话",
				"en": "Phone",
			},
			"copyright": map[string]interface{}{
				"zh": "版权所有",
				"en": "Copyright",
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyGlobal, config)

	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestCanPublish_Valid(t *testing.T) {
	vs := NewValidationService()

	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
		TranslationStatus: map[string]TranslationState{
			"hero.title": TranslationStateDone,
			"hero.label": TranslationStateDone,
		},
	}

	if !vs.CanPublish(result) {
		t.Error("Expected CanPublish to be true for valid config with all done translations")
	}
}

func TestCanPublish_Invalid(t *testing.T) {
	vs := NewValidationService()

	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Path: "hero.title", Code: "REQUIRED", Message: "Required"},
		},
		TranslationStatus: map[string]TranslationState{},
	}

	if vs.CanPublish(result) {
		t.Error("Expected CanPublish to be false for invalid config")
	}
}

func TestCanPublish_MissingTranslation(t *testing.T) {
	vs := NewValidationService()

	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
		TranslationStatus: map[string]TranslationState{
			"hero.title": TranslationStateDone,
			"hero.label": TranslationStateMissing,
		},
	}

	if vs.CanPublish(result) {
		t.Error("Expected CanPublish to be false when translation is missing")
	}
}

func TestCanPublish_StaleTranslation(t *testing.T) {
	vs := NewValidationService()

	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
		TranslationStatus: map[string]TranslationState{
			"hero.title": TranslationStateDone,
			"hero.label": TranslationStateStale,
		},
	}

	if vs.CanPublish(result) {
		t.Error("Expected CanPublish to be false when translation is stale")
	}
}

func TestValidateConfig_EmptyArrays(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"label": map[string]interface{}{
				"zh": "标签",
				"en": "Label",
			},
			"title": map[string]interface{}{
				"zh": "标题",
				"en": "Title",
			},
			"image": map[string]interface{}{
				"url": "/images/hero.jpg",
				"alt": map[string]interface{}{
					"zh": "背景图",
					"en": "Background",
				},
			},
		},
		"blocks": []interface{}{}, // Empty array should fail
	}

	result := vs.ValidateConfig(model.PageKeyAdvantages, config)

	if result.Valid {
		t.Error("Expected invalid config for empty required array")
	}

	hasBlocksError := false
	for _, err := range result.Errors {
		if err.Path == "blocks" {
			hasBlocksError = true
		}
	}
	if !hasBlocksError {
		t.Error("Expected error for empty blocks array")
	}
}

func TestValidateConfig_NestedArrayValidation(t *testing.T) {
	vs := NewValidationService()

	config := model.JSONMap{
		"hero": map[string]interface{}{
			"label": map[string]interface{}{
				"zh": "成功案例",
				"en": "Success Cases",
			},
			"title": map[string]interface{}{
				"zh": "案例",
				"en": "Cases",
			},
			"image": map[string]interface{}{
				"url": "/images/hero.jpg",
				"alt": map[string]interface{}{
					"zh": "背景",
					"en": "Background",
				},
			},
		},
		"cases": []interface{}{
			map[string]interface{}{
				"title": map[string]interface{}{
					"zh": "案例1",
					"en": "", // Missing English
				},
				"items": []interface{}{
					map[string]interface{}{
						"zh": "项目1",
						"en": "Item 1",
					},
				},
			},
		},
	}

	result := vs.ValidateConfig(model.PageKeyCases, config)

	if result.Valid {
		t.Error("Expected invalid config for missing nested translation")
	}

	// Should have missing translation status
	if result.TranslationStatus["cases[0].title"] != TranslationStateMissing {
		t.Errorf("Expected cases[0].title to be missing, got %s", result.TranslationStatus["cases[0].title"])
	}
}
