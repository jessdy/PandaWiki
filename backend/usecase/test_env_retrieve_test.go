package usecase

import (
	"strings"
	"testing"

	"github.com/chaitin/panda-wiki/domain"
)

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("你好世界", 4); got != "你好世界" {
		t.Fatalf("short content changed: %s", got)
	}
	got := truncateRunes("你好世界", 2)
	if !strings.HasPrefix(got, "你好") || !strings.Contains(got, "截断") {
		t.Fatalf("unexpected truncate result: %s", got)
	}
}

func TestAnswerContentFromReleaseIncludesAttributes(t *testing.T) {
	content := answerContentFromRelease(&domain.NodeRelease{
		Name:    "信封",
		Content: "白色信封正文",
		Meta: domain.NodeMeta{
			Attributes: map[string]string{"颜色": "白"},
		},
	})
	if !strings.Contains(content, "颜色") || !strings.Contains(content, "白色信封正文") {
		t.Fatalf("unexpected content: %s", content)
	}
}
