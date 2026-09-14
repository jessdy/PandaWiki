package usecase

import (
	"strings"
	"testing"

	"github.com/chaitin/panda-wiki/domain"
)

func TestSanitizeStandardTerms(t *testing.T) {
	items, err := sanitizeStandardTerms([]domain.StandardTermItem{
		{Term: " 聚对苯二甲酸乙二醇酯 ", Aliases: []string{" PET ", "pet", ""}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Term != "聚对苯二甲酸乙二醇酯" {
		t.Fatalf("unexpected items: %#v", items)
	}
	if items[0].ID == "" {
		t.Fatal("expected generated id")
	}
	if len(items[0].Aliases) != 1 || items[0].Aliases[0] != "PET" {
		t.Fatalf("unexpected aliases: %#v", items[0].Aliases)
	}
}

func TestSanitizeStandardTermsRejectsAliasConflict(t *testing.T) {
	_, err := sanitizeStandardTerms([]domain.StandardTermItem{
		{Term: "术语一", Aliases: []string{"共同别名"}},
		{Term: "术语二", Aliases: []string{"共同别名"}},
	})
	if err == nil || !strings.Contains(err.Error(), "已属于") {
		t.Fatalf("expected alias conflict, got %v", err)
	}
}

func TestSanitizeStandardTermsRepairsDuplicateIDs(t *testing.T) {
	duplicateID := "486fe592-5524-4ed0-a099-528a5fa6cad2"
	items, err := sanitizeStandardTerms([]domain.StandardTermItem{
		{ID: duplicateID, Term: "术语一", Aliases: []string{"别名一"}},
		{ID: duplicateID, Term: "术语二", Aliases: []string{"别名二"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].ID == items[1].ID {
		t.Fatalf("expected unique IDs, got %q", items[0].ID)
	}
}

func TestApplyStandardTermReplacementsUsesWhitelistAndLongestSpan(t *testing.T) {
	terms := []domain.StandardTermItem{
		{Term: "聚对苯二甲酸乙二醇酯", Aliases: []string{"PET"}},
		{Term: "聚酯瓶", Aliases: []string{"PET瓶"}},
	}
	got, notices := applyStandardTermReplacements(
		"这个PET瓶怎么处理？",
		terms,
		[]standardTermNormalizeReplacement{
			{From: "PET", To: "聚对苯二甲酸乙二醇酯"},
			{From: "PET瓶", To: "聚酯瓶"},
			{From: "PET瓶", To: "聚对苯二甲酸乙二醇酯"},
			{From: "怎么处理", To: "配置外术语"},
		},
	)
	if len(notices) != 1 {
		t.Fatal("expected replacement to be applied")
	}
	if got != "这个聚酯瓶怎么处理？" {
		t.Fatalf("unexpected normalized question: %q", got)
	}
	if notices[0].Original != "PET瓶" || notices[0].Standard != "聚酯瓶" {
		t.Fatalf("unexpected notices: %#v", notices)
	}
}

func TestConfiguredAliasReplacementsAreDeterministic(t *testing.T) {
	terms := []domain.StandardTermItem{
		{Term: "商业信封", Aliases: []string{"公司信套", "企业壳"}},
	}
	replacements := configuredAliasReplacements("企业壳是什么", terms)
	got, notices := applyStandardTermReplacements(
		"企业壳是什么",
		terms,
		replacements,
	)
	if got != "商业信封是什么" {
		t.Fatalf("unexpected normalized question: %q", got)
	}
	if len(notices) != 1 ||
		notices[0].Original != "企业壳" ||
		notices[0].Standard != "商业信封" {
		t.Fatalf("unexpected notices: %#v", notices)
	}
}

func TestApplyStandardTermReplacementsRejectsInventedSource(t *testing.T) {
	question := "如何处理这个瓶子？"
	got, notices := applyStandardTermReplacements(
		question,
		[]domain.StandardTermItem{{Term: "聚酯瓶", Aliases: []string{"PET瓶"}}},
		[]standardTermNormalizeReplacement{{From: "PET瓶", To: "聚酯瓶"}},
	)
	if len(notices) > 0 || got != question {
		t.Fatalf("unexpected normalization: %q, notices=%#v", got, notices)
	}
}

func TestFormatStandardTermNotices(t *testing.T) {
	got := formatStandardTermNotices([]standardTermNormalizationNotice{
		{Original: "PET瓶", Standard: "聚酯瓶"},
		{Original: "铁罐", Standard: "金属包装"},
	})
	want := "> 术语提示：“PET瓶”，标准术语为“聚酯瓶”。\n" +
		"> 术语提示：“铁罐”，标准术语为“金属包装”。\n\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestConversationMessageContentForRAGUsesNormalizedTrainingHistory(t *testing.T) {
	message := &domain.ConversationMessage{
		Content:          "PET瓶怎么处理？",
		RetrievalContent: "聚酯瓶怎么处理？",
	}
	if got := conversationMessageContentForRAG(message, domain.QaModeTraining); got != message.RetrievalContent {
		t.Fatalf("training mode got %q", got)
	}
	if got := conversationMessageContentForRAG(message, domain.QaModeWork); got != message.Content {
		t.Fatalf("work mode got %q", got)
	}
	message.RetrievalContent = ""
	if got := conversationMessageContentForRAG(message, domain.QaModeTraining); got != message.Content {
		t.Fatalf("empty retrieval content got %q", got)
	}
}
