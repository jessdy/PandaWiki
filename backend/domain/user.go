package domain

import (
	"time"

	"github.com/lib/pq"

	"github.com/chaitin/panda-wiki/consts"
)

type User struct {
	ID         string          `json:"id" gorm:"primaryKey"`
	Account    string          `json:"account" gorm:"uniqueIndex"`
	Password   string          `json:"password"`
	Role       consts.UserRole `json:"role" gorm:"default:'user'"`
	Region     string          `json:"region" gorm:"default:''"`
	CreatedAt  time.Time       `json:"created_at"`
	LastAccess time.Time       `json:"last_access" gorm:"default:null"`
}

// KBUsers 知识库用户关联表（多对多关系）
type KBUsers struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	KBId      string         `json:"kb_id" gorm:"uniqueIndex:idx_uniq_kb_users_kb_id_user_id"`
	UserId    string         `json:"user_id" gorm:"uniqueIndex:idx_uniq_kb_users_kb_id_user_id"`
	Perms     pq.StringArray `json:"perms" gorm:"type:text[];column:perms"`
	CreatedAt time.Time      `json:"created_at"`
}

func (k KBUsers) GetPerms() consts.UserKBPermissions {
	perms := make(consts.UserKBPermissions, len(k.Perms))
	for i, p := range k.Perms {
		perms[i] = consts.UserKBPermission(p)
	}
	return perms
}

func (KBUsers) TableName() string {
	return "kb_users"
}

type UserAccessTime struct {
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// AuthUserSlot echo.Context 插槽名，保存当前登录用户（id/account/role），供审计与审批读取主体。
// 不是加密密钥；命名避免 *Key 以免静态扫描误报。
const AuthUserSlot = "pw_auth_user"

// AuditSubjectAccountSlot 登录类接口写入的账号或约定主体，供审计中间件填充主体（勿写入口令/tgt 原文）。
const AuditSubjectAccountSlot = "pw_audit_subject_account"

// AuditSubjectUserIDSlot 登录失败等路径补上的用户唯一标识。已存在账号写 users.id，TGT 写平台用户号。
const AuditSubjectUserIDSlot = "pw_audit_subject_user_id"

// AuditDetailSlot 处理函数补充的审计明细（导出条数、文件名等），中间件会拼进 detail。
const AuditDetailSlot = "pw_audit_detail"

// AuthTokenCookieName 管理端 JWT 的 HttpOnly Cookie 名（避免 localStorage 被 XSS 读取）
const AuthTokenCookieName = "panda_wiki_token"
