package domain

import "strings"

const SettingKeyMethodRules = "method_rules"

// MethodRule 一条「开封方法规则」。
//
// 一条规则表示「在某品类下，当属性满足某个条件组合时，应当推荐用户去看某个文档」。
//   - Category    关联到 CategoryPromptItem.Name
//   - Name        方法名（卡片标题），需在同一品类下唯一
//   - Description 可选附属说明，前台卡片展示
//   - Conditions  属性名 → 允许值数组（OR 关系）；列出的属性必须全部命中（AND）；未列出的属性视为通配
//   - NodeID      关联到具体的文档节点（前台卡片"查看完整文档"跳转目标）
type MethodRule struct {
	ID          string              `json:"id"`
	Category    string              `json:"category"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Conditions  map[string][]string `json:"conditions"`
	NodeID      string              `json:"node_id"`
	CreatedAt   int64               `json:"created_at"`
	UpdatedAt   int64               `json:"updated_at"`
}

// ReplaceMethodRulesReq 整表替换接口入参。
type ReplaceMethodRulesReq struct {
	KBID  string       `json:"kb_id" validate:"required"`
	Items []MethodRule `json:"items"`
}

// MatchesCollected 判定一条规则是否被用户「已收集属性」命中。
//
//   - 遍历规则的每个 Conditions 项：用户必须为该属性提供值
//     且该值与规则枚举数组中至少一项做 contains-match。
//   - 全部命中返回 true；任一不满足返回 false。
//   - 规则未列出的属性视为通配，不参与判定。
//   - 规则 Conditions 为空（无任何属性条件）时视为「永远命中」（兜底规则）。
func (r *MethodRule) MatchesCollected(collected map[string]string) bool {
	if len(r.Conditions) == 0 {
		return true
	}
	for key, allowed := range r.Conditions {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		actual, ok := collected[key]
		actual = strings.TrimSpace(actual)
		if !ok || actual == "" {
			return false
		}
		if !matchAnyValue(actual, allowed) {
			return false
		}
	}
	return true
}

// matchAnyValue 与现有 attrValueMatches 行为一致：忽略大小写 + 首尾空白；包含关系视为命中。
// allowed 为空数组表示「该属性必须有值即可」。
func matchAnyValue(actual string, allowed []string) bool {
	a := strings.ToLower(strings.TrimSpace(actual))
	if a == "" {
		return false
	}
	if len(allowed) == 0 {
		return true
	}
	for _, v := range allowed {
		b := strings.ToLower(strings.TrimSpace(v))
		if b == "" {
			continue
		}
		if a == b || strings.Contains(a, b) || strings.Contains(b, a) {
			return true
		}
	}
	return false
}
