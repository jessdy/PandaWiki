package domain

const SettingKeyImageDescriptionTemplates = "image_description_templates"

// ImageDescriptionTemplate 一条「图片描述模版」。模版属于某个 KB 下的某个品类
// （Category 字段冗余存品类名，便于品类被改名后仍可显示历史模版）。
// 模版的 Attributes 是「品类 → 属性维护」里维护的属性键的具体取值；选用模版时，
// 前端把这些键值对按 K-V 拼接后写入图片的 title 文本。
type ImageDescriptionTemplate struct {
	ID         string            `json:"id"`
	Category   string            `json:"category"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
	CreatedAt  int64             `json:"created_at"`
	UpdatedAt  int64             `json:"updated_at"`
}

// CreateImageDescriptionTemplateReq 新增一条模版的入参
type CreateImageDescriptionTemplateReq struct {
	KBID       string            `json:"kb_id" validate:"required"`
	Category   string            `json:"category" validate:"required"`
	Name       string            `json:"name" validate:"required"`
	Attributes map[string]string `json:"attributes"`
}
