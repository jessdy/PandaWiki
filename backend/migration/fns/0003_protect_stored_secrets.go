package fns

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/cache"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/utils"
)

// MigrationProtectStoredSecrets 每次启动都执行且幂等：模型 api_key 用当前主密钥 SM4 加密，
// 知识库访问口令改存 bcrypt 哈希。覆盖历史明文、密钥轮换以及从旧备份恢复的数据。
type MigrationProtectStoredSecrets struct {
	Name      string
	logger    *log.Logger
	modelRepo *pg.ModelRepository
	cipher    *utils.FieldCipher
	kbCache   *cache.KBRepo
}

func NewMigrationProtectStoredSecrets(logger *log.Logger, modelRepo *pg.ModelRepository, cipher *utils.FieldCipher, kbCache *cache.KBRepo) *MigrationProtectStoredSecrets {
	return &MigrationProtectStoredSecrets{
		Name:      "0003_protect_stored_secrets",
		logger:    logger,
		modelRepo: modelRepo,
		cipher:    cipher,
		kbCache:   kbCache,
	}
}

func (m *MigrationProtectStoredSecrets) Execute(tx *gorm.DB) error {
	count, err := m.modelRepo.EncryptStoredAPIKeys(tx)
	if err != nil {
		return fmt.Errorf("encrypt model api keys: %w", err)
	}
	if count > 0 {
		m.logger.Info("encrypted model api keys", log.Int("count", count))
	}
	if err := m.encryptAutoModeAPIKey(tx); err != nil {
		return fmt.Errorf("encrypt auto mode api key: %w", err)
	}
	if err := m.hashSimpleAuthPasswords(tx); err != nil {
		return fmt.Errorf("hash simple auth passwords: %w", err)
	}
	return nil
}

func (m *MigrationProtectStoredSecrets) encryptAutoModeAPIKey(tx *gorm.DB) error {
	var setting domain.SystemSetting
	result := tx.Where("key = ?", consts.SystemSettingModelMode).Limit(1).Find(&setting)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	var value map[string]any
	if err := json.Unmarshal(setting.Value, &value); err != nil {
		return err
	}
	apiKey, _ := value["auto_mode_api_key"].(string)
	if !m.cipher.NeedsReencrypt(apiKey) {
		return nil
	}
	plain, err := m.cipher.Decrypt(apiKey)
	if err != nil {
		m.logger.Error("skip auto mode api key that cannot be decrypted", log.Error(err))
		return nil
	}
	if value["auto_mode_api_key"], err = m.cipher.Encrypt(plain); err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return tx.Model(&domain.SystemSetting{}).Where("id = ?", setting.ID).Update("value", raw).Error
}

func (m *MigrationProtectStoredSecrets) hashSimpleAuthPasswords(tx *gorm.DB) error {
	var kbs []struct {
		ID       string
		Password string
	}
	if err := tx.Model(&domain.KnowledgeBase{}).
		Select("id", "access_settings->'simple_auth'->>'password' AS password").
		Find(&kbs).Error; err != nil {
		return err
	}
	for _, kb := range kbs {
		if kb.Password == "" || utils.IsBcryptHash(kb.Password) {
			continue
		}
		hash, err := utils.HashSecret(kb.Password)
		if err != nil {
			return err
		}
		if err := tx.Exec(
			`UPDATE knowledge_bases SET access_settings = jsonb_set(access_settings, '{simple_auth,password}', to_jsonb(?::text)) WHERE id = ?`,
			hash, kb.ID,
		).Error; err != nil {
			return err
		}
		// 缓存里的知识库详情不过期，不删会一直保留明文口令
		if err := m.kbCache.DeleteKB(context.Background(), kb.ID); err != nil {
			m.logger.Warn("delete kb cache failed", log.String("kb_id", kb.ID), log.Error(err))
		}
		m.logger.Info("hashed simple auth password", log.String("kb_id", kb.ID))
	}
	return nil
}
