package share

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	kbv1 "github.com/chaitin/panda-wiki/api/kb/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
)

// linkedLoginUser TGT 校验通过后用于本系统建号/建会话的身份（不再支持 Base64 user_link 直登）。
type linkedLoginUser struct {
	Username string
	Region   string
	Role     consts.UserRole
}

func linkedRoleName(role consts.UserRole) string {
	if role == consts.UserRoleUser {
		return "admin"
	}
	return "user"
}

func (h *ShareAuthHandler) loginLinkedUser(c echo.Context, kbID string, link linkedLoginUser) (*domain.User, error) {
	ctx := c.Request().Context()
	user, created, err := h.findOrCreateLinkUser(ctx, c, link)
	if err != nil {
		return nil, err
	}
	if err := matchLinkedUserRole(user.Role, link.Role); err != nil {
		return nil, err
	}
	if created {
		if err := h.grantLinkUserAccess(ctx, kbID, user); err != nil {
			h.logger.Error("failed to grant linked user access", log.Error(err))
			return nil, err
		}
	}

	auth := &domain.Auth{
		KBID:       kbID,
		UnionID:    user.Account,
		SourceType: consts.SourceTypeUserPassword,
		IP:         c.RealIP(),
		UserInfo: domain.AuthUserInfo{
			Username: user.Account,
		},
		LastLoginTime: time.Now(),
	}
	ctxWithEdition := context.WithValue(ctx, consts.CtxEdition, consts.GetLicenseEdition(c))
	auth, err = h.authUsecase.AuthRepo.GetOrCreateAuth(ctxWithEdition, auth, consts.SourceTypeUserPassword)
	if err != nil {
		h.logger.Error("failed to get or create auth", log.Error(err))
		return nil, fmt.Errorf("创建登录会话失败")
	}
	if err := h.authUsecase.SaveNewSession(c, auth); err != nil {
		h.logger.Error("failed to save session", log.Error(err))
		return nil, fmt.Errorf("保存登录会话失败")
	}
	return user, nil
}

func matchLinkedUserRole(actual, requested consts.UserRole) error {
	if requested == consts.UserRoleGuest && actual != consts.UserRoleGuest {
		return fmt.Errorf("该用户名对应的不是普通用户")
	}
	if requested == consts.UserRoleUser && actual == consts.UserRoleAdmin {
		return fmt.Errorf("超级管理员不能通过此方式登录")
	}
	if requested == consts.UserRoleUser && actual != consts.UserRoleUser {
		return fmt.Errorf("该用户名对应的不是管理员")
	}
	return nil
}

func (h *ShareAuthHandler) findOrCreateLinkUser(
	ctx context.Context,
	c echo.Context,
	link linkedLoginUser,
) (*domain.User, bool, error) {
	user, err := h.userUsecase.GetUserByAccount(ctx, link.Username)
	if err == nil {
		return user, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("查询用户失败")
	}

	password, err := randomLinkPassword()
	if err != nil {
		return nil, false, fmt.Errorf("创建用户失败")
	}
	user = &domain.User{
		ID:       uuid.New().String(),
		Account:  link.Username,
		Password: password,
		Role:     link.Role,
		Region:   link.Region,
	}
	if err := h.userUsecase.CreateUser(ctx, user, consts.GetLicenseEdition(c)); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") {
			existing, getErr := h.userUsecase.GetUserByAccount(ctx, link.Username)
			if getErr == nil {
				return existing, false, nil
			}
		}
		return nil, false, fmt.Errorf("创建用户失败")
	}
	return user, true, nil
}

func (h *ShareAuthHandler) grantLinkUserAccess(ctx context.Context, kbID string, user *domain.User) error {
	if user.Role == consts.UserRoleGuest {
		if err := h.addUserToAllGroups(ctx, kbID, user.ID); err != nil {
			return fmt.Errorf("加入用户组失败")
		}
		return nil
	}
	if err := h.kbUsecase.KBUserInvite(ctx, kbv1.KBUserInviteReq{
		KBId:   kbID,
		UserId: user.ID,
		Perms:  []string{string(consts.UserKBPermissionFullControl)},
	}); err != nil {
		return fmt.Errorf("授予知识库权限失败")
	}
	return nil
}

func (h *ShareAuthHandler) addUserToAllGroups(ctx context.Context, kbID, userID string) error {
	groups, err := h.authUsecase.ListAuthGroups(ctx, kbID)
	if err != nil {
		return err
	}
	groupIDs := make([]uint, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	if len(groupIDs) == 0 {
		return nil
	}
	return h.authUsecase.UpdateUserGroups(ctx, userID, groupIDs)
}

func randomLinkPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
