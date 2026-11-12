package domain

import (
	"time"

	"github.com/chaitin/panda-wiki/consts"
)

// table: settings
type SystemSetting struct {
	ID          int                     `json:"id" gorm:"primary_key"`
	Key         consts.SystemSettingName `json:"key"`
	Value       []byte                  `json:"value" gorm:"type:jsonb"` // JSON string
	Description string                  `json:"description"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}

// ModelModeSetting 模型配置结构体
type ModelModeSetting struct {
	Mode                     consts.ModelSettingMode `json:"mode"`                        // 模式: manual 或 auto
	AutoModeAPIKey           string                  `json:"auto_mode_api_key"`           // 百智云 API Key
	ChatModel                string                  `json:"chat_model"`                  // 自定义对话模型名称
	IsManualEmbeddingUpdated bool                    `json:"is_manual_embedding_updated"` // 手动模式下嵌入模型是否更新
}

// CloudAuthSetting 平台会话验证地址。登录时拼成 Scheme://IP:端口（默认 https）。
type CloudAuthSetting struct {
	Host   string `json:"host" validate:"max=253"`
	Port   int    `json:"port" validate:"min=0,max=65535"`
	Scheme string `json:"scheme" validate:"omitempty,oneof=http https"` // 空则 https
}

// TestEnvSetting 测试环境。打开后问答不使用向量模型和重排模型。
type TestEnvSetting struct {
	Enabled bool `json:"enabled"`
}

// UploadDeniedExtensionsSetting 上传扩展名配置，内置白名单之外的类型需在 allowed_extensions 中放行
// INSERT INTO "public"."system_settings" ("key", "value") VALUES ('upload', '{"denied_extensions": ["rar"], "allowed_extensions": ["dwg"]}')
type UploadDeniedExtensionsSetting struct {
	DeniedExtensions  []string `json:"denied_extensions"`  // 禁止上传的扩展名，不带点
	AllowedExtensions []string `json:"allowed_extensions"` // 额外放行的扩展名，不带点；内置禁止的类型无法放行
}
