package domain

import (
	"strconv"
	"time"
)

const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)

// AuditLog 安全审计记录。时间、地点(IP)、事件类型、主体唯一标识、客体、结果六要素齐全。
type AuditLog struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	CreatedAt  time.Time `json:"created_at"`
	UserID     string    `json:"user_id"`
	Account    string    `json:"account"`
	Role       string    `json:"role"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	Category   string    `json:"category"`
	Action     string    `json:"action"`
	Event      string    `json:"event"`
	ObjectType string    `json:"object_type"`
	ObjectID   string    `json:"object_id"`
	Detail     string    `json:"detail"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Result     string    `json:"result"`
	StatusCode int       `json:"status_code"`
	Message    string    `json:"message"`
	PrevHash   string    `json:"prev_hash"`
	Hash       string    `json:"hash"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// CanonicalFields 参与哈希链计算的字段，顺序和格式一旦发布不能再改
func (l *AuditLog) CanonicalFields() []string {
	return []string{
		strconv.FormatInt(l.ID, 10),
		l.CreatedAt.UTC().Format(time.RFC3339Nano),
		l.UserID,
		l.Account,
		l.Role,
		l.IP,
		l.UserAgent,
		l.Category,
		l.Action,
		l.Event,
		l.ObjectType,
		l.ObjectID,
		l.Detail,
		l.Method,
		l.Path,
		l.Result,
		strconv.Itoa(l.StatusCode),
		l.Message,
	}
}

type AuditLogFilter struct {
	Account  string
	UserID   string
	Category string
	Result   string
	Keyword  string
	Start    *time.Time
	End      *time.Time
}

type AuditLogListReq struct {
	Page     int    `query:"page" json:"page" validate:"omitempty,min=1"`
	PerPage  int    `query:"per_page" json:"per_page" validate:"omitempty,min=1,max=200"`
	Account  string `query:"account" json:"account" validate:"max=100"`
	UserID   string `query:"user_id" json:"user_id" validate:"max=100"`
	Category string `query:"category" json:"category" validate:"max=50"`
	Result   string `query:"result" json:"result" validate:"omitempty,oneof=success failure"`
	Keyword  string `query:"keyword" json:"keyword" validate:"max=100"`
	Start    string `query:"start" json:"start" validate:"max=40"`
	End      string `query:"end" json:"end" validate:"max=40"`
}

type AuditLogListResp struct {
	Total int64      `json:"total"`
	List  []AuditLog `json:"list"`
}

type AuditExportReq struct {
	AuditLogListReq
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type AuditVerifyResult struct {
	Total     int64     `json:"total"`
	Valid     bool      `json:"valid"`
	BrokenID  int64     `json:"broken_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// AuditStorageSetting 审计日志存储容量与告警阈值，由审计管理员配置
type AuditStorageSetting struct {
	LimitMB      int64 `json:"limit_mb" validate:"min=100,max=10485760"`
	AlertPercent int   `json:"alert_percent" validate:"min=50,max=99"`
}

func DefaultAuditStorageSetting() AuditStorageSetting {
	return AuditStorageSetting{LimitMB: 10240, AlertPercent: 80}
}

type AuditStorageStatus struct {
	Rows          int64   `json:"rows"`
	TableBytes    int64   `json:"table_bytes"`
	DatabaseBytes int64   `json:"database_bytes"`
	LimitBytes    int64   `json:"limit_bytes"`
	UsedPercent   float64 `json:"used_percent"`
	AlertPercent  int     `json:"alert_percent"`
	Alert         bool    `json:"alert"`
}

// AuditExportFile 审计日志导出文件：明文 CSV 经 SM4-GCM 加密，整体由 HMAC-SM3 签名
type AuditExportFile struct {
	Format     string           `json:"format"`
	Version    int              `json:"version"`
	CreatedAt  time.Time        `json:"created_at"`
	ExportedBy string           `json:"exported_by"`
	Count      int              `json:"count"`
	KDF        AuditExportKDF   `json:"kdf"`
	Cipher     AuditExportCipher `json:"cipher"`
	Digest     string           `json:"digest"`
	Data       string           `json:"data"`
	Signature  string           `json:"signature"`
}

type AuditExportKDF struct {
	Alg        string `json:"alg"`
	Salt       string `json:"salt"`
	Iterations int    `json:"iterations"`
}

type AuditExportCipher struct {
	Alg   string `json:"alg"`
	Nonce string `json:"nonce"`
}

type AuditExportVerifyResp struct {
	SignatureValid bool      `json:"signature_valid"`
	Decrypted      bool      `json:"decrypted"`
	DigestValid    bool      `json:"digest_valid"`
	Count          int       `json:"count"`
	ExportedBy     string    `json:"exported_by"`
	CreatedAt      time.Time `json:"created_at"`
	CSV            string    `json:"csv,omitempty"`
}
