package pg

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

// MethodRuleRepo 复用 settings 表：每个 KB 一行 JSON，
// key = SettingKeyMethodRules。读全量、整表替换。
type MethodRuleRepo struct {
	db     *pg.DB
	logger *log.Logger
}

type methodRulesJSON struct {
	Items []domain.MethodRule `json:"items"`
}

func NewMethodRuleRepo(db *pg.DB, logger *log.Logger) *MethodRuleRepo {
	return &MethodRuleRepo{db: db, logger: logger}
}

// GetByKBID 返回该 KB 下的全部规则。settings 行未建立时返回空切片。
func (r *MethodRuleRepo) GetByKBID(ctx context.Context, kbID string) ([]domain.MethodRule, error) {
	var setting domain.Setting
	err := r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeyMethodRules).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var payload methodRulesJSON
	if err := json.Unmarshal(setting.Value, &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

// Replace 整表替换。调用方负责 ID 稳定性与字段清洗。
func (r *MethodRuleRepo) Replace(ctx context.Context, kbID string, items []domain.MethodRule) error {
	payload := methodRulesJSON{Items: items}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var existing domain.Setting
	err = r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeyMethodRules).
		First(&existing).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row := domain.Setting{
			KBID:        kbID,
			Key:         domain.SettingKeyMethodRules,
			Value:       b,
			Description: "开封方法规则",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		return r.db.WithContext(ctx).Table("settings").Create(&row).Error
	}
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeyMethodRules).
		Updates(map[string]interface{}{
			"value":      b,
			"updated_at": now,
		}).Error
}
