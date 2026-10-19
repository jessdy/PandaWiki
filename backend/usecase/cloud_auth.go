package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/repo/pg"
)

type CloudAuthUsecase struct {
	systemSettingRepo *pg.SystemSettingRepo
}

func NewCloudAuthUsecase(systemSettingRepo *pg.SystemSettingRepo) *CloudAuthUsecase {
	return &CloudAuthUsecase{systemSettingRepo: systemSettingRepo}
}

func (u *CloudAuthUsecase) Get(ctx context.Context) (*domain.CloudAuthSetting, error) {
	setting, err := u.systemSettingRepo.GetSystemSetting(ctx, consts.SystemSettingCloudAuth)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &domain.CloudAuthSetting{}, nil
	}
	if err != nil {
		return nil, err
	}
	var value domain.CloudAuthSetting
	if len(setting.Value) > 0 {
		if err := json.Unmarshal(setting.Value, &value); err != nil {
			return nil, err
		}
	}
	return &value, nil
}

func (u *CloudAuthUsecase) BaseURL(ctx context.Context) (string, error) {
	setting, err := u.Get(ctx)
	if err != nil {
		return "", err
	}
	return CloudAuthBaseURL(setting), nil
}

func (u *CloudAuthUsecase) Save(ctx context.Context, setting *domain.CloudAuthSetting) (*domain.CloudAuthSetting, error) {
	normalized, err := NormalizeCloudAuthSetting(setting)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	if err := u.systemSettingRepo.UpsertSystemSetting(
		ctx,
		consts.SystemSettingCloudAuth,
		string(raw),
		"平台会话验证地址",
	); err != nil {
		return nil, err
	}
	return normalized, nil
}

func NormalizeCloudAuthSetting(setting *domain.CloudAuthSetting) (*domain.CloudAuthSetting, error) {
	if setting == nil {
		setting = &domain.CloudAuthSetting{}
	}
	host := strings.TrimSpace(setting.Host)
	host = strings.Trim(host, "[]")
	if host == "" && setting.Port == 0 {
		return &domain.CloudAuthSetting{}, nil
	}
	if host == "" || setting.Port == 0 {
		return nil, fmt.Errorf("请同时填写 IP 和端口")
	}
	if strings.Contains(host, "://") || strings.ContainsAny(host, "/\\ ") {
		return nil, fmt.Errorf("IP 只填写地址，不要包含协议和路径")
	}
	if setting.Port < 1 || setting.Port > 65535 {
		return nil, fmt.Errorf("端口必须在 1 到 65535 之间")
	}
	if net.ParseIP(host) == nil && !validCloudAuthHost(host) {
		return nil, fmt.Errorf("IP 格式无效")
	}
	return &domain.CloudAuthSetting{Host: host, Port: setting.Port}, nil
}

func CloudAuthBaseURL(setting *domain.CloudAuthSetting) string {
	if setting == nil || strings.TrimSpace(setting.Host) == "" || setting.Port <= 0 {
		return ""
	}
	host := strings.Trim(strings.TrimSpace(setting.Host), "[]")
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%d", host, setting.Port)
}

func validCloudAuthHost(host string) bool {
	if host == "" || len(host) > 253 || strings.Contains(host, "..") {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}
