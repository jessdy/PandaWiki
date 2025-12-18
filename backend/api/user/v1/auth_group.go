package v1

import "time"

type AuthGroupListItemResp struct {
	ID        uint     `json:"id"`
	Name      string   `json:"name"`
	KbID      string   `json:"kb_id"`
	ParentID  *uint    `json:"parent_id"`
	Position  float64  `json:"position"`
	AuthIDs   []int64  `json:"auth_ids"`
	UserIDs   []string `json:"user_ids"` // 关联 users 表的用户 ID
	Path      string   `json:"path"`     // 用户组路径，用于显示层级结构
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthGroupListResp struct {
	Groups []AuthGroupListItemResp `json:"groups"`
	List   []AuthGroupListItemResp `json:"list"` // 兼容前端类型定义
}

type CreateAuthGroupReq struct {
	Name     string   `json:"name" validate:"required"`
	KbID     string   `json:"kb_id" validate:"required"`
	ParentID *uint    `json:"parent_id"`
	Position float64  `json:"position"`
	AuthIDs  []int64  `json:"auth_ids"`
	UserIDs  []string `json:"user_ids"` // 关联 users 表的用户 ID
}

type CreateAuthGroupResp struct {
	ID uint `json:"id"`
}

type UpdateAuthGroupReq struct {
	Name     string   `json:"name" validate:"required"`
	ParentID *uint    `json:"parent_id"`
	Position float64  `json:"position"`
	AuthIDs  []int64  `json:"auth_ids"`
	UserIDs  []string `json:"user_ids"` // 关联 users 表的用户 ID
}

