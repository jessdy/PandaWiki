package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// 「疑难咨询」状态机：
//   - pending    待处理（首条提问后默认）
//   - processing 处理中（admin 已查看或部分回复，未结案）
//   - replied    已回复（admin 给出明确答复后）
//   - closed     已关闭（不再继续追问）
const (
	ConsultStatusPending    = "pending"
	ConsultStatusProcessing = "processing"
	ConsultStatusReplied    = "replied"
	ConsultStatusClosed     = "closed"
)

// IsValidConsultStatus 仅在 admin 改状态接口里做校验；前台 share API 不直接接受 status 入参。
func IsValidConsultStatus(s string) bool {
	switch s {
	case ConsultStatusPending, ConsultStatusProcessing, ConsultStatusReplied, ConsultStatusClosed:
		return true
	}
	return false
}

const (
	ConsultSenderUser  = "user"
	ConsultSenderAdmin = "admin"
)

// ConsultAttachment 附件统一结构。type 取 image / video；其它字段由前台上传成功后透传。
type ConsultAttachment struct {
	Type string `json:"type"`           // image / video
	URL  string `json:"url"`            // 形如 /static-file/xxx
	Name string `json:"name,omitempty"` // 原始文件名（便于 admin 排查）
	Size int64  `json:"size,omitempty"` // 字节，前端校验后传入
	Mime string `json:"mime,omitempty"`
}

// ConsultAttachments 是 ConsultAttachment 的切片，实现 GORM 的 jsonb 序列化。
type ConsultAttachments []ConsultAttachment

func (a ConsultAttachments) Value() (driver.Value, error) {
	if a == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(a)
}

func (a *ConsultAttachments) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return errors.New("invalid consult attachments type")
	}
	if len(raw) == 0 {
		*a = nil
		return nil
	}
	return json.Unmarshal(raw, a)
}

// ConsultInquiry 一条咨询单。Title 由 usecase 从 content 自动派生（前 30 字），便于列表展示。
type ConsultInquiry struct {
	ID            int64              `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        string             `json:"user_id" gorm:"column:user_id;index"`
	Title         string             `json:"title" gorm:"column:title"`
	Content       string             `json:"content" gorm:"column:content"`
	Contact       string             `json:"contact" gorm:"column:contact"`
	Attachments   ConsultAttachments `json:"attachments" gorm:"column:attachments;type:jsonb;default:'[]'"`
	Status        string             `json:"status" gorm:"column:status"`
	Info          []byte             `json:"-" gorm:"column:info;type:jsonb;default:'{}'"`
	CreatedAt     time.Time          `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time          `json:"updated_at" gorm:"column:updated_at"`
	LastMessageAt time.Time          `json:"last_message_at" gorm:"column:last_message_at"`
}

func (ConsultInquiry) TableName() string { return "consult_inquiries" }

// ConsultMessage 一条线程消息。
type ConsultMessage struct {
	ID          int64              `json:"id" gorm:"primaryKey;autoIncrement"`
	InquiryID   int64              `json:"inquiry_id" gorm:"column:inquiry_id;index"`
	SenderKind  string             `json:"sender_kind" gorm:"column:sender_kind"`
	SenderID    string             `json:"sender_id" gorm:"column:sender_id"`
	SenderName  string             `json:"sender_name" gorm:"column:sender_name"`
	Content     string             `json:"content" gorm:"column:content"`
	Attachments ConsultAttachments `json:"attachments" gorm:"column:attachments;type:jsonb;default:'[]'"`
	CreatedAt   time.Time          `json:"created_at" gorm:"column:created_at"`
}

func (ConsultMessage) TableName() string { return "consult_messages" }

// ConsultInquiryListItem 列表展示用，含最近一条消息时间与回复条数。
type ConsultInquiryListItem struct {
	ConsultInquiry
	SubmitterName string `json:"submitter_name" gorm:"column:submitter_name"`
	ReplyCount    int64  `json:"reply_count" gorm:"column:reply_count"`
}

// ConsultInquiryDetail 详情（含消息流）。
type ConsultInquiryDetail struct {
	ConsultInquiry
	Messages []ConsultMessage `json:"messages"`
}

/* ---------------------------------------------------------------- */
/* DTO（API 请求体）                                                  */
/* ---------------------------------------------------------------- */

// CreateConsultInquiryReq 前台创建咨询入参。
type CreateConsultInquiryReq struct {
	Content     string             `json:"content" validate:"required"`
	Contact     string             `json:"contact,omitempty"`
	Attachments ConsultAttachments `json:"attachments,omitempty"`
}

// CreateConsultMessageReq 前台或后台追加消息入参。
type CreateConsultMessageReq struct {
	InquiryID   int64              `json:"inquiry_id" validate:"required"`
	Content     string             `json:"content" validate:"required"`
	Attachments ConsultAttachments `json:"attachments,omitempty"`
}

// UpdateConsultStatusReq admin 改状态入参。
type UpdateConsultStatusReq struct {
	InquiryID int64  `json:"inquiry_id" validate:"required"`
	Status    string `json:"status" validate:"required"`
}
