package v1

import (
	"time"

	"github.com/chaitin/panda-wiki/consts"
)

type CreateUserReq struct {
	Account  string          `json:"account" validate:"required"`
	Password string          `json:"password" validate:"required,min=8"`
	Role     consts.UserRole `json:"role" validate:"required,oneof=admin user guest"`
}

type CreateUserResp struct {
	ID string `json:"id"`
}

type UserInfoResp struct {
	ID         string          `json:"id"`
	Account    string          `json:"account"`
	Role       consts.UserRole `json:"role"`
	IsToken    bool            `json:"is_token"`
	LastAccess *time.Time      `json:"last_access,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type UserListReq struct {
}

type UserListItemResp struct {
	ID         string          `json:"id"`
	Account    string          `json:"account"`
	Role       consts.UserRole `json:"role"`
	LastAccess *time.Time      `json:"last_access"`
	CreatedAt  *time.Time      `json:"created_at"`
}

type LoginReq struct {
	Account  string `json:"account" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResp struct {
	Token string `json:"token"`
}

type UserListResp struct {
	Users []UserListItemResp `json:"users"`
}

type ResetPasswordReq struct {
	ID          string `json:"id" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type DeleteUserReq struct {
	UserID string `json:"user_id" query:"user_id" validate:"required"`
}

// GetUserGroupsReq 获取用户所属的用户组请求
type GetUserGroupsReq struct {
	UserID string `json:"user_id" query:"user_id" validate:"required"`
}

// GetUserGroupsResp 获取用户所属的用户组响应
type GetUserGroupsResp struct {
	Groups []AuthGroupListItemResp `json:"groups"`
}

// UpdateUserGroupsReq 更新用户所属的用户组请求
type UpdateUserGroupsReq struct {
	UserID  string  `json:"user_id" validate:"required"`
	GroupIDs []uint  `json:"group_ids"` // 用户组 ID 列表
}
