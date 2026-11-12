package usecase

import (
	"context"
	"encoding/json"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/repo/pg"
)

type SystemInfoUsecase struct {
	setting *pg.SystemSettingRepo
	audit   *AuditUsecase
	backup  *BackupUsecase
}

func NewSystemInfoUsecase(setting *pg.SystemSettingRepo, audit *AuditUsecase, backup *BackupUsecase) *SystemInfoUsecase {
	return &SystemInfoUsecase{setting: setting, audit: audit, backup: backup}
}

func (u *SystemInfoUsecase) GetClassification(ctx context.Context) (domain.ClassificationSetting, error) {
	setting := domain.ClassificationSetting{}
	raw, err := u.setting.GetSystemSetting(ctx, consts.SystemSettingClassification)
	if err != nil {
		return setting, nil
	}
	_ = json.Unmarshal(raw.Value, &setting)
	return setting, nil
}

func (u *SystemInfoUsecase) SaveClassification(ctx context.Context, setting domain.ClassificationSetting) (*domain.ClassificationSetting, error) {
	raw, err := json.Marshal(setting)
	if err != nil {
		return nil, err
	}
	if err := u.setting.UpsertSystemSetting(ctx, consts.SystemSettingClassification, string(raw), "密级标识"); err != nil {
		return nil, err
	}
	return &setting, nil
}

func (u *SystemInfoUsecase) PublicInfo(ctx context.Context) (*domain.SystemPublicInfo, error) {
	cls, _ := u.GetClassification(ctx)
	return &domain.SystemPublicInfo{Classification: cls}, nil
}

func (u *SystemInfoUsecase) Alerts(ctx context.Context) ([]domain.SystemAlert, error) {
	alerts := make([]domain.SystemAlert, 0)
	if status, err := u.audit.StorageStatus(ctx); err == nil && status.Alert {
		alerts = append(alerts, domain.SystemAlert{
			Type:    "audit_storage",
			Level:   "warning",
			Message: "审计日志存储空间已超过告警阈值，请尽快导出并扩容",
		})
	}
	if status, err := u.backup.StorageStatus(ctx); err == nil {
		if status.Alert {
			alerts = append(alerts, domain.SystemAlert{
				Type:    "backup_storage",
				Level:   "warning",
				Message: "备份存储空间不足，请清理旧备份或扩容",
			})
		}
		if status.Remind {
			alerts = append(alerts, domain.SystemAlert{
				Type:    "backup_remind",
				Level:   "info",
				Message: "距离上次成功备份已超过提醒周期，请尽快执行备份",
			})
		}
	}
	return alerts, nil
}
