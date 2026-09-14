package share

import (
	"testing"

	"github.com/chaitin/panda-wiki/domain"
)

func TestBuildCategoryViewsDoesNotExposePromptContent(t *testing.T) {
	items := buildCategoryViews([]domain.CategoryPromptItem{
		{
			Name:    "  饮料罐  ",
			Content: "private recognition prompt",
			AttributeSpecs: []domain.CategoryAttributeSpec{
				{Name: "材质", Values: []string{"铝", "马口铁"}},
			},
		},
		{Name: "  "},
	})
	if len(items) != 1 {
		t.Fatalf("got %d categories, want 1", len(items))
	}
	if items[0].Name != "饮料罐" {
		t.Fatalf("got category %q", items[0].Name)
	}
	if len(items[0].Specs) != 1 || items[0].Specs[0].Name != "材质" {
		t.Fatalf("unexpected specs: %#v", items[0].Specs)
	}
}

func TestCategoryHelperFindCategoryByName(t *testing.T) {
	helper := &categoryHelper{}
	categories := []domain.CategoryPromptItem{{Name: "纸箱"}}
	if helper.findCategoryByName(categories, " 纸箱 ") == nil {
		t.Fatal("expected category to be found")
	}
	if helper.findCategoryByName(categories, "不存在") != nil {
		t.Fatal("unexpected category match")
	}
}
