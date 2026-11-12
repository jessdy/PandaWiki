package domain

import "time"

type ApprovalType string

const (
	ApprovalUserCreate       ApprovalType = "user_create"
	ApprovalGuestCreate      ApprovalType = "guest_create"
	ApprovalUserDelete       ApprovalType = "user_delete"
	ApprovalGuestDelete      ApprovalType = "guest_delete"
	ApprovalKBUserInvite     ApprovalType = "kb_user_invite"
	ApprovalKBUserUpdate     ApprovalType = "kb_user_update"
	ApprovalKBUserDelete     ApprovalType = "kb_user_delete"
	ApprovalAuthGroupCreate  ApprovalType = "auth_group_create"
	ApprovalAuthGroupUpdate  ApprovalType = "auth_group_update"
	ApprovalAuthGroupDelete  ApprovalType = "auth_group_delete"
	ApprovalUserGroupsUpdate ApprovalType = "user_groups_update"
)

type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusFailed   ApprovalStatus = "failed"
)

// UserApproval 待安全管理员审批的用户增删、权限变更
type UserApproval struct {
	ID               int64          `json:"id" gorm:"primaryKey"`
	Type             ApprovalType   `json:"type"`
	Status           ApprovalStatus `json:"status"`
	Summary          string         `json:"summary"`
	TargetUserID     string         `json:"target_user_id"`
	TargetAccount    string         `json:"target_account"`
	KbID             string         `json:"kb_id"`
	Payload          []byte         `json:"-" gorm:"type:jsonb"` // 可能含口令哈希，不对外返回
	RequesterID      string         `json:"requester_id"`
	RequesterAccount string         `json:"requester_account"`
	ReviewerID       string         `json:"reviewer_id"`
	ReviewerAccount  string         `json:"reviewer_account"`
	ReviewComment    string         `json:"review_comment"`
	Error            string         `json:"error"`
	CreatedAt        time.Time      `json:"created_at"`
	ReviewedAt       *time.Time     `json:"reviewed_at"`
}

func (UserApproval) TableName() string {
	return "user_approvals"
}

type ApprovalListReq struct {
	Status  string `query:"status" validate:"omitempty,oneof=pending approved rejected failed"`
	Page    int    `query:"page" validate:"omitempty,min=1"`
	PerPage int    `query:"per_page" validate:"omitempty,min=1,max=100"`
}

type ApprovalListResp struct {
	Total int64          `json:"total"`
	List  []UserApproval `json:"list"`
}

type ApprovalReviewReq struct {
	Comment string `json:"comment" validate:"max=500"`
}

// ApprovalSubmitResp 需要审批的接口返回该结构，前端据此提示"已提交审批"
type ApprovalSubmitResp struct {
	Pending    bool   `json:"pending"`
	ApprovalID int64  `json:"approval_id"`
	Message    string `json:"message"`
}
