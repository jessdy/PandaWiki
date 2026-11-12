package pg

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
	"github.com/chaitin/panda-wiki/utils"
)

type ModelRepository struct {
	db     *pg.DB
	logger *log.Logger
	cipher *utils.FieldCipher
}

func NewModelRepository(db *pg.DB, cipher *utils.FieldCipher, logger *log.Logger) *ModelRepository {
	return &ModelRepository{db: db, logger: logger.WithModule("repo.pg.model"), cipher: cipher}
}

func (r *ModelRepository) Create(ctx context.Context, model *domain.Model) error {
	plain := model.APIKey
	encrypted, err := r.cipher.Encrypt(plain)
	if err != nil {
		return fmt.Errorf("encrypt model api key failed: %w", err)
	}
	model.APIKey = encrypted
	defer func() { model.APIKey = plain }()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *ModelRepository) GetList(ctx context.Context) ([]*domain.ModelListItem, error) {
	var models []*domain.ModelListItem
	if err := r.db.WithContext(ctx).
		Model(&domain.Model{}).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	for _, m := range models {
		apiKey, err := r.cipher.Decrypt(m.APIKey)
		if err != nil {
			// 密钥被更换后旧密文无法解开，置空让管理员重新填写，不影响其他模型展示
			r.logger.Error("decrypt model api key failed", log.String("model_id", m.ID), log.Error(err))
		}
		m.APIKey = apiKey
	}
	return models, nil
}

func (r *ModelRepository) Update(ctx context.Context, req *domain.UpdateModelReq) error {
	param := domain.ModelParam{}
	if req.Parameters != nil {
		param = *req.Parameters
	}
	apiKey, err := r.cipher.Encrypt(req.APIKey)
	if err != nil {
		return fmt.Errorf("encrypt model api key failed: %w", err)
	}
	updateMap := map[string]any{
		"model":       req.Model,
		"api_key":     apiKey,
		"api_header":  req.APIHeader,
		"base_url":    req.BaseURL,
		"api_version": req.APIVersion,
		"provider":    req.Provider,
		"type":        req.Type,
		"parameters":  param,
	}
	if req.IsActive != nil {
		updateMap["is_active"] = *req.IsActive
	}
	return r.db.WithContext(ctx).
		Model(&domain.Model{}).
		Where("id = ?", req.ID).
		Updates(updateMap).Error
}

func (r *ModelRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// delete model
		if err := tx.Where("id = ?", id).
			Delete(&domain.Model{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *ModelRepository) GetChatModel(ctx context.Context) (*domain.Model, error) {
	return r.GetModelByType(ctx, domain.ModelTypeChat)
}

func (r *ModelRepository) GetModelByType(ctx context.Context, modelType domain.ModelType) (*domain.Model, error) {
	var model domain.Model
	if err := r.db.WithContext(ctx).
		Model(&domain.Model{}).
		Where("type = ?", modelType).
		First(&model).Error; err != nil {
		return nil, err
	}
	apiKey, err := r.cipher.Decrypt(model.APIKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key of model %s failed, please re-enter the api key: %w", model.ID, err)
	}
	model.APIKey = apiKey
	return &model, nil
}

func (r *ModelRepository) UpdateUsage(ctx context.Context, modelID string, usage *schema.TokenUsage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// update model usage
		if err := tx.Model(&domain.Model{}).
			Where("id = ?", modelID).
			Updates(map[string]any{
				"prompt_tokens":     gorm.Expr("prompt_tokens + ?", usage.PromptTokens),
				"completion_tokens": gorm.Expr("completion_tokens + ?", usage.CompletionTokens),
				"total_tokens":      gorm.Expr("total_tokens + ?", usage.TotalTokens),
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

// EncryptStoredAPIKeys 把历史明文或旧密钥加密的 api_key 统一改为当前主密钥加密。
func (r *ModelRepository) EncryptStoredAPIKeys(tx *gorm.DB) (int, error) {
	var rows []struct {
		ID     string
		APIKey string
	}
	if err := tx.Model(&domain.Model{}).Select("id", "api_key").Find(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		if !r.cipher.NeedsReencrypt(row.APIKey) {
			continue
		}
		plain, err := r.cipher.Decrypt(row.APIKey)
		if err != nil {
			r.logger.Error("skip model api key that cannot be decrypted", log.String("model_id", row.ID), log.Error(err))
			continue
		}
		encrypted, err := r.cipher.Encrypt(plain)
		if err != nil {
			return updated, err
		}
		if err := tx.Model(&domain.Model{}).Where("id = ?", row.ID).Update("api_key", encrypted).Error; err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}
