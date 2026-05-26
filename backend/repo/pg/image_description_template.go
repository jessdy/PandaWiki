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

// ImageDescriptionTemplateRepo 复用 settings 表：每个 KB 一行 JSON，
// key = SettingKeyImageDescriptionTemplates。读操作返回全量，写操作整表替换。
type ImageDescriptionTemplateRepo struct {
	db     *pg.DB
	logger *log.Logger
}

type imageDescriptionTemplatesJSON struct {
	Items []domain.ImageDescriptionTemplate `json:"items"`
}

func NewImageDescriptionTemplateRepo(db *pg.DB, logger *log.Logger) *ImageDescriptionTemplateRepo {
	return &ImageDescriptionTemplateRepo{db: db, logger: logger}
}

// GetByKBID 返回该 KB 下的全部模版。settings 行未建立时返回空切片。
func (r *ImageDescriptionTemplateRepo) GetByKBID(ctx context.Context, kbID string) ([]domain.ImageDescriptionTemplate, error) {
	var setting domain.Setting
	err := r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeyImageDescriptionTemplates).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var payload imageDescriptionTemplatesJSON
	if err := json.Unmarshal(setting.Value, &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

// Replace 整表替换。调用方在 usecase 层完成「拉出 → 追加 → 写回」的原子语义。
func (r *ImageDescriptionTemplateRepo) Replace(ctx context.Context, kbID string, items []domain.ImageDescriptionTemplate) error {
	payload := imageDescriptionTemplatesJSON{Items: items}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var existing domain.Setting
	err = r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeyImageDescriptionTemplates).
		First(&existing).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row := domain.Setting{
			KBID:        kbID,
			Key:         domain.SettingKeyImageDescriptionTemplates,
			Value:       b,
			Description: "图片描述模版",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		return r.db.WithContext(ctx).Table("settings").Create(&row).Error
	}
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeyImageDescriptionTemplates).
		Updates(map[string]interface{}{
			"value":      b,
			"updated_at": now,
		}).Error
}
