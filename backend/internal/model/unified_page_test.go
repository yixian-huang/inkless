package model_test

import (
	"testing"
	"blotting-consultancy/internal/model"
)

func TestUnifiedPage_Validate_RequiresSlug(t *testing.T) {
	p := &model.UnifiedPage{Slug: ""}
	if err := p.Validate(); err == nil {
		t.Error("expected error for empty slug")
	}
}

func TestUnifiedPage_Validate_RequiresMode(t *testing.T) {
	p := &model.UnifiedPage{Slug: "test", Mode: "invalid"}
	if err := p.Validate(); err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestUnifiedPage_Validate_ValidComposable(t *testing.T) {
	p := &model.UnifiedPage{Slug: "test", Mode: "composable"}
	if err := p.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnifiedPage_Validate_TemplateRequiresTemplateID(t *testing.T) {
	p := &model.UnifiedPage{Slug: "test", Mode: "template", TemplateID: nil}
	if err := p.Validate(); err == nil {
		t.Error("expected error for template mode without templateId")
	}
}

func TestUnifiedPage_Validate_ValidTemplate(t *testing.T) {
	tid := uint(1)
	p := &model.UnifiedPage{Slug: "test", Mode: "template", TemplateID: &tid}
	if err := p.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnifiedPage_Validate_ValidStatuses(t *testing.T) {
	for _, s := range []string{"draft", "published", "scheduled"} {
		p := &model.UnifiedPage{Slug: "test", Mode: "composable", Status: s}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error for status %q: %v", s, err)
		}
	}
}
