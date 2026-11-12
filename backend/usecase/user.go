package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	v1 "github.com/chaitin/panda-wiki/api/user/v1"
	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/utils"
)

type UserUsecase struct {
	repo   *pg.UserRepository
	logger *log.Logger
	config *config.Config
}

func NewUserUsecase(repo *pg.UserRepository, logger *log.Logger, config *config.Config) (*UserUsecase, error) {
	u := &UserUsecase{
		repo:   repo,
		logger: logger.WithModule("usecase.user"),
		config: config,
	}
	if err := u.ensureBuiltinUsers(); err != nil {
		return nil, err
	}
	return u, nil
}

func (u *UserUsecase) ensureBuiltinUsers() error {
	type builtin struct {
		account  string
		password string
		role     consts.UserRole
	}
	users := []builtin{
		{account: "admin", password: u.config.AdminPassword, role: consts.UserRoleAdmin},
		{account: consts.SecurityAdminAccount, password: u.config.SecurityAdminPassword, role: consts.UserRoleSecurityAdmin},
		{account: consts.AuditAdminAccount, password: u.config.AuditAdminPassword, role: consts.UserRoleAuditAdmin},
	}
	for _, item := range users {
		if item.password == "" {
			continue
		}
		if err := utils.ValidatePasswordComplexity(item.password); err != nil {
			// 不阻断启动：现场弱口令会导致整站起不来；跳过写入并要求尽快更换
			u.logger.Error("内置账号口令不符合复杂度要求，已跳过写入",
				log.String("account", item.account), log.Error(err))
			continue
		}
		if err := u.repo.UpsertDefaultUser(context.Background(), &domain.User{
			ID:       uuid.New().String(),
			Account:  item.account,
			Password: item.password,
			Role:     item.role,
		}); err != nil {
			return fmt.Errorf("failed to upsert builtin user %s: %w", item.account, err)
		}
	}
	return nil
}

func (u *UserUsecase) CreateUser(ctx context.Context, user *domain.User, edition consts.LicenseEdition) error {
	return u.repo.CreateUser(ctx, user, edition)
}

func (u *UserUsecase) VerifyUser(ctx context.Context, account string, password string) (*domain.User, error) {
	return u.repo.VerifyUser(ctx, account, password)
}

func (u *UserUsecase) VerifyUserAndGenerateToken(ctx context.Context, req v1.LoginReq) (string, error) {
	user, err := u.VerifyUser(ctx, req.Account, req.Password)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   user.ID,
		"role": string(user.Role),
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(u.config.Auth.JWT.Secret))
}

func (u *UserUsecase) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	return u.repo.GetUser(ctx, userID)
}

func (u *UserUsecase) GetUserByAccount(ctx context.Context, account string) (*domain.User, error) {
	return u.repo.GetUserByAccount(ctx, account)
}

func (u *UserUsecase) ListUsers(ctx context.Context) (*v1.UserListResp, error) {
	users, err := u.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.UserListResp{Users: users}, nil
}

func (u *UserUsecase) ListGuestUsers(ctx context.Context) (*v1.UserListResp, error) {
	users, err := u.repo.ListGuestUsers(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.UserListResp{Users: users}, nil
}

func (u *UserUsecase) ResetPassword(ctx context.Context, req *v1.ResetPasswordReq) error {
	return u.repo.UpdateUserPassword(ctx, req.ID, req.NewPassword)
}

func (u *UserUsecase) UpdateUserRegion(ctx context.Context, userID, region string) error {
	return u.repo.UpdateUserRegion(ctx, userID, region)
}

func (u *UserUsecase) DeleteUser(ctx context.Context, userID string) error {
	return u.repo.DeleteUser(ctx, userID)
}
