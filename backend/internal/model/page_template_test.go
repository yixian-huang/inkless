package model_test

import (
	"testing"
	"blotting-consultancy/internal/model"
)

func TestPageTemplate_Validate_RequiresKey(t *testing.T) {
	pt := &model.PageTemplate{Key: "", NameZh: "test", Category: "custom"}
	if err := pt.Validate(); err == nil {
		t.Error("expected error for empty key")
	}
}

func TestPageTemplate_Validate_RequiresNameZh(t *testing.T) {
	pt := &model.PageTemplate{Key: "test", NameZh: "", Category: "custom"}
	if err := pt.Validate(); err == nil {
		t.Error("expected error for empty nameZh")
	}
}

func TestPageTemplate_Validate_ValidCategories(t *testing.T) {
	for _, c := range []string{"builtin", "custom", "theme"} {
		pt := &model.PageTemplate{Key: "test", NameZh: "测试", Category: c}
		if err := pt.Validate(); err != nil {
			t.Errorf("unexpected error for category %q: %v", c, err)
		}
	}
}

func TestPageTemplate_Validate_InvalidCategory(t *testing.T) {
	pt := &model.PageTemplate{Key: "test", NameZh: "测试", Category: "invalid"}
	if err := pt.Validate(); err == nil {
		t.Error("expected error for invalid category")
	}
}
