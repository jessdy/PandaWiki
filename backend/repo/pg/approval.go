package pg

import (
	"context"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

type ApprovalRepo struct {
	db     *pg.DB
	logger *log.Logger
}

func NewApprovalRepo(db *pg.DB, logger *log.Logger) *ApprovalRepo {
	return &ApprovalRepo{db: db, logger: logger.WithModule("repo.pg.approval")}
}

func (r *ApprovalRepo) Create(ctx context.Context, a *domain.UserApproval) error {
	a.Status = domain.ApprovalStatusPending
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *ApprovalRepo) Get(ctx context.Context, id int64) (*domain.UserApproval, error) {
	var a domain.UserApproval
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// List requesterID 非空时只返回该用户发起的申请
func (r *ApprovalRepo) List(ctx context.Context, status, requesterID string, offset, limit int) ([]domain.UserApproval, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.UserApproval{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if requesterID != "" {
		q = q.Where("requester_id = ?", requesterID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	list := make([]domain.UserApproval, 0, limit)
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *ApprovalRepo) CountPending(ctx context.Context, requesterID string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.UserApproval{}).Where("status = ?", domain.ApprovalStatusPending)
	if requesterID != "" {
		q = q.Where("requester_id = ?", requesterID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *ApprovalRepo) ExistsPending(ctx context.Context, types []domain.ApprovalType, targetAccount, targetUserID string) (bool, error) {
	q := r.db.WithContext(ctx).Model(&domain.UserApproval{}).
		Where("status = ? AND type IN ?", domain.ApprovalStatusPending, types)
	if targetAccount != "" {
		q = q.Where("target_account = ?", targetAccount)
	}
	if targetUserID != "" {
		q = q.Where("target_user_id = ?", targetUserID)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

// Claim 把待审批记录原子地标记为已处理，返回 false 表示已被他人处理
func (r *ApprovalRepo) Claim(ctx context.Context, id int64, status domain.ApprovalStatus, reviewerID, reviewerAccount, comment string) (bool, error) {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&domain.UserApproval{}).
		Where("id = ? AND status = ?", id, domain.ApprovalStatusPending).
		Updates(map[string]any{
			"status":           status,
			"reviewer_id":      reviewerID,
			"reviewer_account": reviewerAccount,
			"review_comment":   comment,
			"reviewed_at":      now,
		})
	return res.RowsAffected == 1, res.Error
}

func (r *ApprovalRepo) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	return r.db.WithContext(ctx).Model(&domain.UserApproval{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": domain.ApprovalStatusFailed, "error": errMsg}).Error
}
