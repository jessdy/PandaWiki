package pg

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
	"github.com/chaitin/panda-wiki/utils"
)

// 串行化哈希链写入的 advisory lock，多个 API 实例共用同一条链
const auditChainLockID int64 = 7419300001

type AuditRepo struct {
	db        *pg.DB
	logger    *log.Logger
	chainKeys [][]byte
	exportKey []byte
}

func NewAuditRepo(db *pg.DB, config *config.Config, logger *log.Logger) (*AuditRepo, error) {
	secrets := config.FieldEncryptionSecrets()
	if len(secrets) == 0 {
		return nil, errors.New("audit chain key is empty: set DATA_ENCRYPTION_KEY or JWT_SECRET")
	}
	keys := make([][]byte, 0, len(secrets))
	for _, s := range secrets {
		keys = append(keys, utils.DeriveKey(s, "audit-chain"))
	}
	return &AuditRepo{
		db:        db,
		logger:    logger.WithModule("repo.pg.audit"),
		chainKeys: keys,
		exportKey: utils.DeriveKey(secrets[0], "audit-export"),
	}, nil
}

func truncateUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	s = s[:max]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

func normalizeAuditLog(entry *domain.AuditLog) {
	entry.UserID = truncateUTF8(entry.UserID, 100)
	entry.Account = truncateUTF8(entry.Account, 100)
	entry.Role = truncateUTF8(entry.Role, 50)
	entry.IP = truncateUTF8(entry.IP, 64)
	entry.UserAgent = truncateUTF8(entry.UserAgent, 300)
	entry.Category = truncateUTF8(entry.Category, 50)
	entry.Action = truncateUTF8(entry.Action, 100)
	entry.Event = truncateUTF8(entry.Event, 100)
	entry.ObjectType = truncateUTF8(entry.ObjectType, 50)
	entry.ObjectID = truncateUTF8(entry.ObjectID, 200)
	entry.Detail = truncateUTF8(entry.Detail, 1000)
	entry.Method = truncateUTF8(entry.Method, 10)
	entry.Path = truncateUTF8(entry.Path, 500)
	entry.Message = truncateUTF8(entry.Message, 500)
	if entry.Result != domain.AuditResultFailure {
		entry.Result = domain.AuditResultSuccess
	}
}

func (r *AuditRepo) chainHash(key []byte, entry *domain.AuditLog) string {
	return utils.HMACSM3Hex(key, append([]string{entry.PrevHash}, entry.CanonicalFields()...)...)
}

// Append 追加一条审计记录并接入哈希链
func (r *AuditRepo) Append(ctx context.Context, entry *domain.AuditLog) error {
	normalizeAuditLog(entry)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", auditChainLockID).Error; err != nil {
			return err
		}
		var prev string
		if err := tx.Raw("SELECT hash FROM audit_logs ORDER BY id DESC LIMIT 1").Scan(&prev).Error; err != nil {
			return err
		}
		if prev == "" {
			prev = utils.AuditGenesisHash
		}
		var id int64
		if err := tx.Raw("SELECT nextval(pg_get_serial_sequence('audit_logs', 'id'))").Scan(&id).Error; err != nil {
			return err
		}
		entry.ID = id
		entry.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
		entry.PrevHash = prev
		entry.Hash = r.chainHash(r.chainKeys[0], entry)
		return tx.Create(entry).Error
	})
}

func escapeLikePattern(s string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s) + "%"
}

func applyAuditFilter(q *gorm.DB, f domain.AuditLogFilter) *gorm.DB {
	if f.Account != "" {
		q = q.Where("account ILIKE ?", escapeLikePattern(f.Account))
	}
	if f.UserID != "" {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Result != "" {
		q = q.Where("result = ?", f.Result)
	}
	if f.Keyword != "" {
		kw := escapeLikePattern(f.Keyword)
		q = q.Where("(event ILIKE ? OR path ILIKE ? OR object_id ILIKE ? OR detail ILIKE ? OR ip ILIKE ? OR message ILIKE ?)",
			kw, kw, kw, kw, kw, kw)
	}
	if f.Start != nil {
		q = q.Where("created_at >= ?", *f.Start)
	}
	if f.End != nil {
		q = q.Where("created_at < ?", *f.End)
	}
	return q
}

func (r *AuditRepo) List(ctx context.Context, f domain.AuditLogFilter, offset, limit int) ([]domain.AuditLog, int64, error) {
	var total int64
	if err := applyAuditFilter(r.db.WithContext(ctx).Model(&domain.AuditLog{}), f).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	list := make([]domain.AuditLog, 0, limit)
	err := applyAuditFilter(r.db.WithContext(ctx).Model(&domain.AuditLog{}), f).
		Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

// Iterate 按 id 升序分批遍历，max<=0 表示不限制条数
func (r *AuditRepo) Iterate(ctx context.Context, f domain.AuditLogFilter, max int, fn func([]domain.AuditLog) error) error {
	var lastID int64
	seen := 0
	for {
		batchSize := 1000
		if max > 0 && max-seen < batchSize {
			batchSize = max - seen
		}
		if batchSize <= 0 {
			return nil
		}
		var batch []domain.AuditLog
		err := applyAuditFilter(r.db.WithContext(ctx).Model(&domain.AuditLog{}), f).
			Where("id > ?", lastID).Order("id ASC").Limit(batchSize).Find(&batch).Error
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		if err := fn(batch); err != nil {
			return err
		}
		seen += len(batch)
		lastID = batch[len(batch)-1].ID
	}
}

// Verify 逐条校验 HMAC 与前后链接，发现被改写、删除或插入的记录
func (r *AuditRepo) Verify(ctx context.Context) (*domain.AuditVerifyResult, error) {
	result := &domain.AuditVerifyResult{Valid: true}
	expectedPrev := utils.AuditGenesisHash
	errBroken := errors.New("chain broken")
	err := r.Iterate(ctx, domain.AuditLogFilter{}, 0, func(batch []domain.AuditLog) error {
		for i := range batch {
			rec := &batch[i]
			result.Total++
			if rec.PrevHash != expectedPrev {
				result.Valid, result.BrokenID = false, rec.ID
				result.Reason = "与上一条记录的哈希不衔接，存在记录被删除或插入"
				return errBroken
			}
			matched := false
			for _, key := range r.chainKeys {
				if utils.HexEqual(r.chainHash(key, rec), rec.Hash) {
					matched = true
					break
				}
			}
			if !matched {
				result.Valid, result.BrokenID = false, rec.ID
				result.Reason = "记录内容与哈希不符，记录被篡改"
				return errBroken
			}
			expectedPrev = rec.Hash
		}
		return nil
	})
	if err != nil && !errors.Is(err, errBroken) {
		return nil, err
	}
	result.CheckedAt = time.Now()
	return result, nil
}

func (r *AuditRepo) Storage(ctx context.Context) (rows, tableBytes, dbBytes int64, err error) {
	db := r.db.WithContext(ctx)
	if err = db.Raw("SELECT GREATEST(reltuples, 0)::bigint FROM pg_class WHERE oid = 'audit_logs'::regclass").Scan(&rows).Error; err != nil {
		return
	}
	if rows < 100000 {
		if err = db.Raw("SELECT COUNT(*) FROM audit_logs").Scan(&rows).Error; err != nil {
			return
		}
	}
	if err = db.Raw("SELECT pg_total_relation_size('audit_logs')").Scan(&tableBytes).Error; err != nil {
		return
	}
	err = db.Raw("SELECT pg_database_size(current_database())").Scan(&dbBytes).Error
	return
}

func (r *AuditRepo) SignExport(parts ...string) string {
	return utils.HMACSM3Hex(r.exportKey, parts...)
}
