package domain

import "strings"

const SettingKeyCategoryPrompts = "category_prompts"

// CategoryAttributeSpec 单个属性的结构化定义：属性名 + 允许的枚举值列表。
// values 为空表示该属性不限定枚举（兼容旧数据：从字符串 attributes 自动升级时无枚举）。
type CategoryAttributeSpec struct {
	Name   string   `json:"name"`
	Values []string `json:"values,omitempty"`
}

// CategoryPromptItem 后台「提示词」按品类维护的单条记录。
//
// 历史兼容：
//   - 旧版 Attributes 字段是逗号分隔的属性名字符串（无枚举）。
//   - 新版 AttributeSpecs 是结构化属性 + 枚举值列表。
//   - 后端读取时优先用 AttributeSpecs，为空则从 Attributes 升级出无枚举的 specs。
//   - 写入时同时落两个字段：specs 是权威，Attributes 由 specs 派生（便于老前端读取属性名）。
type CategoryPromptItem struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	Content        string                  `json:"content"`
	Attributes     string                  `json:"attributes"`              // 逗号分隔属性名（派生 / 兼容旧前端）
	AttributeSpecs []CategoryAttributeSpec `json:"attribute_specs,omitempty"` // 结构化属性 + 枚举
}

// ResolveAttributeSpecs 始终返回结构化属性列表：specs 为空时用 attributes 字符串升级。
// 升级出来的项 values 为空，调用方可据此判断"老数据未配置枚举"。
func (c *CategoryPromptItem) ResolveAttributeSpecs() []CategoryAttributeSpec {
	if len(c.AttributeSpecs) > 0 {
		out := make([]CategoryAttributeSpec, 0, len(c.AttributeSpecs))
		for _, s := range c.AttributeSpecs {
			name := strings.TrimSpace(s.Name)
			if name == "" {
				continue
			}
			values := make([]string, 0, len(s.Values))
			for _, v := range s.Values {
				v = strings.TrimSpace(v)
				if v != "" {
					values = append(values, v)
				}
			}
			out = append(out, CategoryAttributeSpec{Name: name, Values: values})
		}
		return out
	}
	raw := strings.TrimSpace(c.Attributes)
	if raw == "" {
		return nil
	}
	norm := strings.ReplaceAll(raw, "\uff0c", ",")
	parts := strings.Split(norm, ",")
	out := make([]CategoryAttributeSpec, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		out = append(out, CategoryAttributeSpec{Name: s})
	}
	return out
}

// CategoryPromptsReq 保存品类提示词列表（整表替换）
type CategoryPromptsReq struct {
	KBID  string               `json:"kb_id" validate:"required"`
	Items []CategoryPromptItem `json:"items"`
}
