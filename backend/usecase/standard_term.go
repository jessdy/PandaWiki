package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/repo/pg"
)

const (
	maxStandardTerms      = 200
	maxAliasesPerStandard = 50
	maxStandardTermRunes  = 100
	maxStandardAliasRunes = 100
)

type StandardTermUsecase struct {
	systemSettingRepo *pg.SystemSettingRepo
}

func NewStandardTermUsecase(systemSettingRepo *pg.SystemSettingRepo) *StandardTermUsecase {
	return &StandardTermUsecase{systemSettingRepo: systemSettingRepo}
}

func (u *StandardTermUsecase) Get(ctx context.Context) (*domain.StandardTermsSetting, error) {
	setting, err := u.systemSettingRepo.GetSystemSetting(ctx, consts.SystemSettingStandardTerms)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &domain.StandardTermsSetting{Items: []domain.StandardTermItem{}}, nil
	}
	if err != nil {
		return nil, err
	}

	result := &domain.StandardTermsSetting{Items: []domain.StandardTermItem{}}
	if len(setting.Value) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(setting.Value, result); err != nil {
		return nil, fmt.Errorf("invalid standard terms setting: %w", err)
	}
	if result.Items == nil {
		result.Items = []domain.StandardTermItem{}
	}
	return result, nil
}

func (u *StandardTermUsecase) Replace(ctx context.Context, req *domain.ReplaceStandardTermsReq) (*domain.StandardTermsSetting, error) {
	items, err := sanitizeStandardTerms(req.Items)
	if err != nil {
		return nil, err
	}
	result := &domain.StandardTermsSetting{Items: items}
	value, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if err := u.systemSettingRepo.UpsertSystemSetting(
		ctx,
		consts.SystemSettingStandardTerms,
		string(value),
		"Global standard terms and aliases",
	); err != nil {
		return nil, err
	}
	return result, nil
}

func sanitizeStandardTerms(items []domain.StandardTermItem) ([]domain.StandardTermItem, error) {
	if len(items) > maxStandardTerms {
		return nil, fmt.Errorf("标准术语最多可配置 %d 个", maxStandardTerms)
	}

	result := make([]domain.StandardTermItem, 0, len(items))
	standardOwners := make(map[string]string, len(items))
	usedIDs := make(map[string]struct{}, len(items))
	for index, item := range items {
		term := strings.TrimSpace(item.Term)
		if term == "" {
			return nil, fmt.Errorf("第 %d 个标准术语不能为空", index+1)
		}
		if utf8.RuneCountInString(term) > maxStandardTermRunes {
			return nil, fmt.Errorf("标准术语「%s」不能超过 %d 个字符", term, maxStandardTermRunes)
		}
		normalized := normalizeTermKey(term)
		if previous, exists := standardOwners[normalized]; exists {
			return nil, fmt.Errorf("标准术语「%s」与「%s」重复", term, previous)
		}
		standardOwners[normalized] = term
		id := strings.TrimSpace(item.ID)
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.NewString()
		}
		if _, duplicate := usedIDs[id]; duplicate {
			id = uuid.NewString()
		}
		usedIDs[id] = struct{}{}
		result = append(result, domain.StandardTermItem{
			ID:      id,
			Term:    term,
			Aliases: item.Aliases,
		})
	}

	aliasOwners := make(map[string]string)
	for index := range result {
		item := &result[index]
		if len(item.Aliases) > maxAliasesPerStandard {
			return nil, fmt.Errorf("标准术语「%s」最多可配置 %d 个别名", item.Term, maxAliasesPerStandard)
		}
		cleaned := make([]string, 0, len(item.Aliases))
		seenInItem := make(map[string]struct{}, len(item.Aliases))
		for _, rawAlias := range item.Aliases {
			alias := strings.TrimSpace(rawAlias)
			if alias == "" {
				continue
			}
			if utf8.RuneCountInString(alias) > maxStandardAliasRunes {
				return nil, fmt.Errorf("别名「%s」不能超过 %d 个字符", alias, maxStandardAliasRunes)
			}
			key := normalizeTermKey(alias)
			if _, duplicate := seenInItem[key]; duplicate {
				continue
			}
			if standard, exists := standardOwners[key]; exists {
				return nil, fmt.Errorf("别名「%s」与标准术语「%s」冲突", alias, standard)
			}
			if owner, exists := aliasOwners[key]; exists {
				return nil, fmt.Errorf("别名「%s」已属于标准术语「%s」", alias, owner)
			}
			seenInItem[key] = struct{}{}
			aliasOwners[key] = item.Term
			cleaned = append(cleaned, alias)
		}
		if len(cleaned) == 0 {
			return nil, fmt.Errorf("标准术语「%s」至少需要一个别名", item.Term)
		}
		item.Aliases = cleaned
	}
	return result, nil
}

func normalizeTermKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
