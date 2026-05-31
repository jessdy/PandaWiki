package pg

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

type ConsultRepo struct {
	db     *pg.DB
	logger *log.Logger
}

func NewConsultRepo(db *pg.DB, logger *log.Logger) *ConsultRepo {
	return &ConsultRepo{db: db, logger: logger.WithModule("repo.pg.consult")}
}

// CreateInquiryWithFirstMessage 在一个事务里：创建咨询单 + 写入首条用户消息。
// 入参 inquiry 由 usecase 装配；message 的 InquiryID 会在事务里自动回填。
func (r *ConsultRepo) CreateInquiryWithFirstMessage(
	ctx context.Context,
	inquiry *domain.ConsultInquiry,
	message *domain.ConsultMessage,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(inquiry).Error; err != nil {
			return err
		}
		message.InquiryID = inquiry.ID
		return tx.Create(message).Error
	})
}

// AppendMessage 追加一条消息 + 同步 inquiry.last_message_at 与 status：
//   - user 追加 → 若当前 status=replied，回退到 processing（"还在追问中"）
//   - admin 追加 → 不在这里改 status（admin 端通过显式 SetStatus 或回复时切到 replied）
func (r *ConsultRepo) AppendMessage(
	ctx context.Context,
	inquiryID int64,
	message *domain.ConsultMessage,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inquiry domain.ConsultInquiry
		if err := tx.Where("id = ?", inquiryID).First(&inquiry).Error; err != nil {
			return err
		}
		message.InquiryID = inquiryID
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"last_message_at": time.Now(),
			"updated_at":      time.Now(),
		}
		if message.SenderKind == domain.ConsultSenderUser && inquiry.Status == domain.ConsultStatusReplied {
			updates["status"] = domain.ConsultStatusProcessing
		}
		if message.SenderKind == domain.ConsultSenderAdmin && inquiry.Status == domain.ConsultStatusPending {
			// admin 一旦下场就视为处理中（不强行设为 replied，让 admin 自己决定）
			updates["status"] = domain.ConsultStatusProcessing
		}
		return tx.Model(&domain.ConsultInquiry{}).Where("id = ?", inquiryID).Updates(updates).Error
	})
}

// GetInquiryByID 取单条咨询。caller 可用 userID 做权限校验。
func (r *ConsultRepo) GetInquiryByID(ctx context.Context, id int64) (*domain.ConsultInquiry, error) {
	var row domain.ConsultInquiry
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetMessages 取某咨询下全部消息（升序，便于直接渲染时间线）。
func (r *ConsultRepo) GetMessages(ctx context.Context, inquiryID int64) ([]domain.ConsultMessage, error) {
	var rows []domain.ConsultMessage
	err := r.db.WithContext(ctx).
		Where("inquiry_id = ?", inquiryID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

type ConsultListParams struct {
	// UserID 非空时仅返回该用户的咨询（前台 share 必填）；空时返回全部（admin 全局视图）。
	UserID string
	// Status 非空时按状态过滤。
	Status string
	// 关键字（content / contact LIKE 模糊匹配），admin 用。
	Keyword string
	Page    int
	PerPage int
}

// List 返回咨询单列表与总数。每条带 reply_count（消息总数 - 首条 = 后续追加条数）。
func (r *ConsultRepo) List(ctx context.Context, p ConsultListParams) ([]domain.ConsultInquiryListItem, int64, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 20
	}

	base := r.db.WithContext(ctx).Model(&domain.ConsultInquiry{})
	if uid := strings.TrimSpace(p.UserID); uid != "" {
		base = base.Where("user_id = ?", uid)
	}
	if s := strings.TrimSpace(p.Status); s != "" {
		base = base.Where("status = ?", s)
	}
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		base = base.Where("(content ILIKE ? OR contact ILIKE ? OR title ILIKE ?)", like, like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []domain.ConsultInquiryListItem
	err := base.
		Select(`consult_inquiries.*, COALESCE(
            NULLIF(TRIM((
                SELECT m.sender_name FROM consult_messages m
                 WHERE m.inquiry_id = consult_inquiries.id
                 ORDER BY m.created_at ASC, m.id ASC
                 LIMIT 1
            )), ''),
            NULLIF(TRIM(consult_inquiries.info->>'submitter_name'), ''),
            NULLIF(TRIM((
                SELECT a.user_info->>'username' FROM auths a
                 WHERE a.id::text = consult_inquiries.user_id
                 LIMIT 1
            )), '')
        ) AS submitter_name, (
            SELECT COUNT(*) FROM consult_messages m
             WHERE m.inquiry_id = consult_inquiries.id
        ) - 1 AS reply_count`).
		Order("last_message_at DESC, id DESC").
		Offset((p.Page - 1) * p.PerPage).
		Limit(p.PerPage).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// CountOpen 统计待处理 + 处理中的咨询数量（admin 侧边栏角标用）。
func (r *ConsultRepo) CountOpen(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.ConsultInquiry{}).
		Where("status IN ?", []string{
			domain.ConsultStatusPending,
			domain.ConsultStatusProcessing,
		}).
		Count(&count).Error
	return count, err
}

// SetStatus admin 显式改状态。
func (r *ConsultRepo) SetStatus(ctx context.Context, id int64, status string) error {
	if !domain.IsValidConsultStatus(status) {
		return errors.New("invalid status")
	}
	return r.db.WithContext(ctx).
		Model(&domain.ConsultInquiry{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error
}

// DeleteByIDs admin 批量删除（消息因 FK ON DELETE CASCADE 同时删）。
func (r *ConsultRepo) DeleteByIDs(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&domain.ConsultInquiry{}).Error
}
