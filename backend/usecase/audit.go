package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/utils"
)

type AuditUsecase struct {
	repo   *pg.AuditRepo
	setting *pg.SystemSettingRepo
	logger *log.Logger
}

func NewAuditUsecase(repo *pg.AuditRepo, setting *pg.SystemSettingRepo, logger *log.Logger) *AuditUsecase {
	return &AuditUsecase{
		repo:    repo,
		setting: setting,
		logger:  logger.WithModule("usecase.audit"),
	}
}

func (u *AuditUsecase) Append(ctx context.Context, entry *domain.AuditLog) {
	if entry == nil {
		return
	}
	if err := u.repo.Append(ctx, entry); err != nil {
		u.logger.Error("append audit log failed", log.Error(err),
			log.String("event", entry.Event), log.String("path", entry.Path))
	}
}

func (u *AuditUsecase) AppendFromEcho(c echo.Context, category, action, event, objectType, objectID, detail, result string, statusCode int, message string) {
	entry := &domain.AuditLog{
		IP:         c.RealIP(),
		UserAgent:  c.Request().UserAgent(),
		Category:   category,
		Action:     action,
		Event:      event,
		ObjectType: objectType,
		ObjectID:   objectID,
		Detail:     detail,
		Method:     c.Request().Method,
		Path:       c.Request().URL.Path,
		Result:     result,
		StatusCode: statusCode,
		Message:    message,
	}
	if user, ok := c.Get(domain.AuthUserKey).(*domain.User); ok && user != nil {
		entry.UserID = user.ID
		entry.Account = user.Account
		entry.Role = string(user.Role)
	} else if auth := domain.GetAuthInfoFromCtx(c.Request().Context()); auth != nil {
		entry.UserID = auth.UserId
		if auth.IsToken {
			entry.Account = "api_token"
			entry.Role = "api_token"
		}
	}
	u.Append(c.Request().Context(), entry)
}

func parseAuditTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid time: %s", s)
}

func (u *AuditUsecase) filterFromReq(req domain.AuditLogListReq) (domain.AuditLogFilter, error) {
	start, err := parseAuditTime(req.Start)
	if err != nil {
		return domain.AuditLogFilter{}, err
	}
	end, err := parseAuditTime(req.End)
	if err != nil {
		return domain.AuditLogFilter{}, err
	}
	if end != nil && len(req.End) == 10 {
		t := end.Add(24 * time.Hour)
		end = &t
	}
	return domain.AuditLogFilter{
		Account:  req.Account,
		UserID:   req.UserID,
		Category: req.Category,
		Result:   req.Result,
		Keyword:  req.Keyword,
		Start:    start,
		End:      end,
	}, nil
}

func (u *AuditUsecase) List(ctx context.Context, req domain.AuditLogListReq) (*domain.AuditLogListResp, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PerPage <= 0 {
		req.PerPage = 20
	}
	f, err := u.filterFromReq(req)
	if err != nil {
		return nil, err
	}
	list, total, err := u.repo.List(ctx, f, (req.Page-1)*req.PerPage, req.PerPage)
	if err != nil {
		return nil, err
	}
	return &domain.AuditLogListResp{Total: total, List: list}, nil
}

func (u *AuditUsecase) Verify(ctx context.Context) (*domain.AuditVerifyResult, error) {
	return u.repo.Verify(ctx)
}

func (u *AuditUsecase) StorageSetting(ctx context.Context) (domain.AuditStorageSetting, error) {
	setting := domain.DefaultAuditStorageSetting()
	raw, err := u.setting.GetSystemSetting(ctx, consts.SystemSettingAuditStorage)
	if err != nil {
		return setting, nil
	}
	_ = json.Unmarshal(raw.Value, &setting)
	if setting.LimitMB <= 0 {
		setting.LimitMB = domain.DefaultAuditStorageSetting().LimitMB
	}
	if setting.AlertPercent <= 0 {
		setting.AlertPercent = domain.DefaultAuditStorageSetting().AlertPercent
	}
	return setting, nil
}

func (u *AuditUsecase) SaveStorageSetting(ctx context.Context, setting domain.AuditStorageSetting) (*domain.AuditStorageSetting, error) {
	if setting.LimitMB <= 0 {
		setting.LimitMB = domain.DefaultAuditStorageSetting().LimitMB
	}
	if setting.AlertPercent <= 0 {
		setting.AlertPercent = domain.DefaultAuditStorageSetting().AlertPercent
	}
	raw, err := json.Marshal(setting)
	if err != nil {
		return nil, err
	}
	if err := u.setting.UpsertSystemSetting(ctx, consts.SystemSettingAuditStorage, string(raw), "审计日志存储容量与告警阈值"); err != nil {
		return nil, err
	}
	return &setting, nil
}

func (u *AuditUsecase) StorageStatus(ctx context.Context) (*domain.AuditStorageStatus, error) {
	setting, _ := u.StorageSetting(ctx)
	rows, tableBytes, dbBytes, err := u.repo.Storage(ctx)
	if err != nil {
		return nil, err
	}
	limit := setting.LimitMB * 1024 * 1024
	used := 0.0
	if limit > 0 {
		used = float64(tableBytes) * 100 / float64(limit)
	}
	return &domain.AuditStorageStatus{
		Rows:          rows,
		TableBytes:    tableBytes,
		DatabaseBytes: dbBytes,
		LimitBytes:    limit,
		UsedPercent:   used,
		AlertPercent:  setting.AlertPercent,
		Alert:         used >= float64(setting.AlertPercent),
	}, nil
}

func auditCSVHeader() []string {
	return []string{
		"id", "created_at", "user_id", "account", "role", "ip", "user_agent",
		"category", "action", "event", "object_type", "object_id", "detail",
		"method", "path", "result", "status_code", "message", "prev_hash", "hash",
	}
}

func auditCSVRow(l domain.AuditLog) []string {
	return []string{
		strconv.FormatInt(l.ID, 10),
		l.CreatedAt.UTC().Format(time.RFC3339Nano),
		l.UserID, l.Account, l.Role, l.IP, l.UserAgent,
		l.Category, l.Action, l.Event, l.ObjectType, l.ObjectID, l.Detail,
		l.Method, l.Path, l.Result, strconv.Itoa(l.StatusCode), l.Message,
		l.PrevHash, l.Hash,
	}
}

// Export 导出审计日志：CSV 经 SM4-GCM 加密，导出文件整体由 HMAC-SM3 签名。审计功能不可关闭。
func (u *AuditUsecase) Export(ctx context.Context, req domain.AuditExportReq, exporter string) (*domain.AuditExportFile, error) {
	f, err := u.filterFromReq(req.AuditLogListReq)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(auditCSVHeader()); err != nil {
		return nil, err
	}
	count := 0
	if err := u.repo.Iterate(ctx, f, 100000, func(batch []domain.AuditLog) error {
		for _, item := range batch {
			if err := w.Write(auditCSVRow(item)); err != nil {
				return err
			}
			count++
		}
		return nil
	}); err != nil {
		return nil, err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	csvBytes := buf.Bytes()
	digest := utils.SM3Hex(csvBytes)

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, err := utils.PasswordKey(req.Password, salt, utils.PasswordKDFIterations)
	if err != nil {
		return nil, err
	}
	nonce, sealed, err := utils.SealSM4GCM(key, csvBytes)
	if err != nil {
		return nil, err
	}

	file := &domain.AuditExportFile{
		Format:     "panda-wiki-audit-export",
		Version:    1,
		CreatedAt:  time.Now().UTC(),
		ExportedBy: exporter,
		Count:      count,
		KDF: domain.AuditExportKDF{
			Alg:        "pbkdf2-sm3",
			Salt:       hex.EncodeToString(salt),
			Iterations: utils.PasswordKDFIterations,
		},
		Cipher: domain.AuditExportCipher{
			Alg:   "sm4-gcm",
			Nonce: hex.EncodeToString(nonce),
		},
		Digest: digest,
		Data:   base64.StdEncoding.EncodeToString(sealed),
	}
	file.Signature = u.repo.SignExport(
		file.Format,
		strconv.Itoa(file.Version),
		file.CreatedAt.Format(time.RFC3339Nano),
		file.ExportedBy,
		strconv.Itoa(file.Count),
		file.KDF.Alg, file.KDF.Salt, strconv.Itoa(file.KDF.Iterations),
		file.Cipher.Alg, file.Cipher.Nonce,
		file.Digest, file.Data,
	)
	return file, nil
}

func (u *AuditUsecase) VerifyExport(file *domain.AuditExportFile, password string) (*domain.AuditExportVerifyResp, error) {
	if file == nil {
		return nil, fmt.Errorf("empty export file")
	}
	expected := u.repo.SignExport(
		file.Format,
		strconv.Itoa(file.Version),
		file.CreatedAt.UTC().Format(time.RFC3339Nano),
		file.ExportedBy,
		strconv.Itoa(file.Count),
		file.KDF.Alg, file.KDF.Salt, strconv.Itoa(file.KDF.Iterations),
		file.Cipher.Alg, file.Cipher.Nonce,
		file.Digest, file.Data,
	)
	resp := &domain.AuditExportVerifyResp{
		SignatureValid: utils.HexEqual(expected, file.Signature),
		Count:          file.Count,
		ExportedBy:     file.ExportedBy,
		CreatedAt:      file.CreatedAt,
	}
	if password == "" {
		return resp, nil
	}
	salt, err := hex.DecodeString(file.KDF.Salt)
	if err != nil {
		return resp, err
	}
	key, err := utils.PasswordKey(password, salt, file.KDF.Iterations)
	if err != nil {
		return resp, err
	}
	nonce, err := hex.DecodeString(file.Cipher.Nonce)
	if err != nil {
		return resp, err
	}
	sealed, err := base64.StdEncoding.DecodeString(file.Data)
	if err != nil {
		return resp, err
	}
	plain, err := utils.OpenSM4GCM(key, nonce, sealed)
	if err != nil {
		return resp, nil
	}
	resp.Decrypted = true
	resp.DigestValid = utils.HexEqual(utils.SM3Hex(plain), file.Digest)
	resp.CSV = string(plain)
	return resp, nil
}
