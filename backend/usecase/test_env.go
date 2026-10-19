package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/repo/pg"
)

type TestEnvUsecase struct {
	systemSettingRepo *pg.SystemSettingRepo
}

func NewTestEnvUsecase(systemSettingRepo *pg.SystemSettingRepo) *TestEnvUsecase {
	return &TestEnvUsecase{systemSettingRepo: systemSettingRepo}
}

func (u *TestEnvUsecase) Get(ctx context.Context) (*domain.TestEnvSetting, error) {
	setting, err := u.systemSettingRepo.GetSystemSetting(ctx, consts.SystemSettingTestEnv)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &domain.TestEnvSetting{}, nil
	}
	if err != nil {
		return nil, err
	}
	var value domain.TestEnvSetting
	if len(setting.Value) > 0 {
		if err := json.Unmarshal(setting.Value, &value); err != nil {
			return nil, err
		}
	}
	return &value, nil
}

func (u *TestEnvUsecase) Enabled(ctx context.Context) bool {
	setting, err := u.Get(ctx)
	if err != nil || setting == nil {
		return false
	}
	return setting.Enabled
}

func (u *TestEnvUsecase) Save(ctx context.Context, setting *domain.TestEnvSetting) (*domain.TestEnvSetting, error) {
	if setting == nil {
		setting = &domain.TestEnvSetting{}
	}
	raw, err := json.Marshal(setting)
	if err != nil {
		return nil, err
	}
	if err := u.systemSettingRepo.UpsertSystemSetting(
		ctx,
		consts.SystemSettingTestEnv,
		string(raw),
		"测试环境开关",
	); err != nil {
		return nil, err
	}
	return setting, nil
}
