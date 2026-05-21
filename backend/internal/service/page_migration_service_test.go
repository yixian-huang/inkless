package service

import (
	"testing"

	"blotting-consultancy/internal/model"
)

func TestConvertContentDocConfig_Home(t *testing.T) {
	// Simulate a home page content doc config (object-keyed structure)
	config := model.JSONMap{
		"hero": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "欢迎来到测试站点",
				"en": "Welcome to Test Site",
			},
			"subtitle": map[string]interface{}{
				"zh": "专业的咨询服务",
				"en": "Professional Consulting Services",
			},
			"backgroundImage": map[string]interface{}{
				"url": "/images/hero-bg.jpg",
			},
		},
		"advantages": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "我们的优势",
				"en": "Our Advantages",
			},
			"cards": []interface{}{},
		},
		"about": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "关于我们",
				"en": "About Us",
			},
		},
		"coreServices": map[string]interface{}{
			"title": map[string]interface{}{
				"zh": "核心服务",
				"en": "Core Services",
			},
		},
	}

	result := ConvertContentDocToSections("home", config)

	// Verify sections array exists
	sectionsRaw, ok := result["sections"]
	if !ok {
		t.Fatal("expected sections key in result")
	}

	sections, ok := sectionsRaw.([]interface{})
	if !ok {
		t.Fatal("sections should be a slice")
	}

	// Home page should have 6 sections (hero, card-grid, company-profile, service-cards, team-grid, contact-form)
	if len(sections) != 6 {
		t.Fatalf("expected 6 sections, got %d", len(sections))
	}

	// First section should be hero
	heroSection, ok := sections[0].(model.JSONMap)
	if !ok {
		t.Fatal("first section should be a JSONMap")
	}

	if heroSection["type"] != "hero" {
		t.Errorf("expected first section type 'hero', got %v", heroSection["type"])
	}

	if heroSection["variant"] != "fullscreen" {
		t.Errorf("expected hero variant 'fullscreen', got %v", heroSection["variant"])
	}

	// Check that hero props preserve bilingual data
	heroProps, ok := heroSection["props"].(model.JSONMap)
	if !ok {
		t.Fatal("hero props should be a JSONMap")
	}

	heroTitle, ok := heroProps["title"].(map[string]interface{})
	if !ok {
		t.Fatal("hero title should be a map")
	}

	if heroTitle["zh"] != "欢迎来到测试站点" {
		t.Errorf("expected zh title preserved, got %v", heroTitle["zh"])
	}

	if heroTitle["en"] != "Welcome to Test Site" {
		t.Errorf("expected en title preserved, got %v", heroTitle["en"])
	}

	// Verify second section is card-grid
	cardGrid, ok := sections[1].(model.JSONMap)
	if !ok {
		t.Fatal("second section should be a JSONMap")
	}
	if cardGrid["type"] != "card-grid" {
		t.Errorf("expected second section type 'card-grid', got %v", cardGrid["type"])
	}

	// All sections should have id, type, variant, locked, props
	for i, s := range sections {
		sec := s.(model.JSONMap)
		for _, field := range []string{"id", "type", "variant", "locked", "props"} {
			if _, exists := sec[field]; !exists {
				t.Errorf("section %d missing field %s", i, field)
			}
		}
	}
}

func TestConvertBlockPageConfig_WrapsBilingualFields(t *testing.T) {
	config := model.JSONMap{
		"sections": []interface{}{
			map[string]interface{}{
				"id":   "sec1",
				"type": "hero",
				"props": map[string]interface{}{
					"title":           "Welcome",
					"subtitle":        "Sub text",
					"backgroundImage": "/images/bg.jpg",
				},
			},
			map[string]interface{}{
				"id":   "sec2",
				"type": "card-grid",
				"props": map[string]interface{}{
					"title": "Cards Title",
					"cards": []interface{}{
						map[string]interface{}{
							"title":       "Card 1",
							"description": "Desc 1",
							"icon":        "star",
						},
					},
				},
			},
		},
	}

	result := ConvertBlockPageToUnified(config)

	sectionsRaw := result["sections"].([]interface{})
	if len(sectionsRaw) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sectionsRaw))
	}

	// Check hero section
	hero := sectionsRaw[0].(map[string]interface{})

	// Should have variant and locked added
	if hero["variant"] != "default" {
		t.Errorf("expected variant 'default', got %v", hero["variant"])
	}
	if hero["locked"] != false {
		t.Errorf("expected locked false, got %v", hero["locked"])
	}

	heroProps := hero["props"].(map[string]interface{})

	// title should be wrapped as bilingual
	titleWrapped, ok := heroProps["title"].(map[string]interface{})
	if !ok {
		t.Fatal("hero title should be wrapped as bilingual map")
	}
	if titleWrapped["zh"] != "Welcome" {
		t.Errorf("expected zh='Welcome', got %v", titleWrapped["zh"])
	}
	if titleWrapped["en"] != "" {
		t.Errorf("expected en='', got %v", titleWrapped["en"])
	}

	// backgroundImage should stay as plain string (not localizable)
	bgImg, ok := heroProps["backgroundImage"].(string)
	if !ok {
		t.Fatalf("backgroundImage should remain a string, got %T", heroProps["backgroundImage"])
	}
	if bgImg != "/images/bg.jpg" {
		t.Errorf("expected backgroundImage '/images/bg.jpg', got %v", bgImg)
	}

	// Check card-grid section — cards[].title and cards[].description should be wrapped
	cardGrid := sectionsRaw[1].(map[string]interface{})
	cgProps := cardGrid["props"].(map[string]interface{})

	// title should be wrapped
	cgTitle, ok := cgProps["title"].(map[string]interface{})
	if !ok {
		t.Fatal("card-grid title should be wrapped as bilingual map")
	}
	if cgTitle["zh"] != "Cards Title" {
		t.Errorf("expected card-grid title zh='Cards Title', got %v", cgTitle["zh"])
	}

	// cards[0].title should be wrapped
	cards := cgProps["cards"].([]interface{})
	card0 := cards[0].(map[string]interface{})

	cardTitle, ok := card0["title"].(map[string]interface{})
	if !ok {
		t.Fatal("card title should be wrapped as bilingual map")
	}
	if cardTitle["zh"] != "Card 1" {
		t.Errorf("expected card title zh='Card 1', got %v", cardTitle["zh"])
	}

	// icon should remain a plain string
	icon, ok := card0["icon"].(string)
	if !ok {
		t.Fatalf("icon should remain a string, got %T", card0["icon"])
	}
	if icon != "star" {
		t.Errorf("expected icon 'star', got %v", icon)
	}
}
