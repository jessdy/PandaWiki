package share

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	kbv1 "github.com/chaitin/panda-wiki/api/kb/v1"
	v1 "github.com/chaitin/panda-wiki/api/share/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
)

const (
	maxUserLinkUsernameRunes = 100
	maxUserLinkRegionRunes   = 100
)

type userLinkPayload struct {
	Username   string `json:"username"`
	UserName   string `json:"user_name"`
	UsernameCN string `json:"用户名"`
	Region     string `json:"region"`
	RegionCN   string `json:"地区"`
	Role       string `json:"role"`
	RoleCN     string `json:"角色"`
}

type decodedUserLink struct {
	Username string
	Region   string
	Role     consts.UserRole
}

func decodeUserLink(raw string) (decodedUserLink, error) {
	payload, err := decodeUserLinkBase64(raw)
	if err != nil {
		return decodedUserLink{}, fmt.Errorf("用户信息无法解析")
	}

	var data userLinkPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return decodedUserLink{}, fmt.Errorf("用户信息格式无效")
	}

	username := strings.TrimSpace(firstNonEmpty(data.Username, data.UserName, data.UsernameCN))
	region := strings.TrimSpace(firstNonEmpty(data.Region, data.RegionCN))
	role, err := parseUserLinkRole(firstNonEmpty(data.Role, data.RoleCN))
	if err != nil {
		return decodedUserLink{}, err
	}
	if username == "" {
		return decodedUserLink{}, fmt.Errorf("用户名不能为空")
	}
	if utf8.RuneCountInString(username) > maxUserLinkUsernameRunes {
		return decodedUserLink{}, fmt.Errorf("用户名不能超过 %d 个字符", maxUserLinkUsernameRunes)
	}
	if utf8.RuneCountInString(region) > maxUserLinkRegionRunes {
		return decodedUserLink{}, fmt.Errorf("地区不能超过 %d 个字符", maxUserLinkRegionRunes)
	}
	return decodedUserLink{Username: username, Region: region, Role: role}, nil
}

func parseUserLinkRole(raw string) (consts.UserRole, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "user", "guest":
		return consts.UserRoleGuest, nil
	case "admin", "管理员":
		return consts.UserRoleUser, nil
	default:
		return "", fmt.Errorf("角色只能是 user 或 admin")
	}
}

func userLinkRoleName(role consts.UserRole) string {
	if role == consts.UserRoleUser {
		return "admin"
	}
	return "user"
}

func decodeUserLinkBase64(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, " ", "+")
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (h *ShareAuthHandler) AuthLoginUserLink(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	var req v1.AuthLoginUserLinkReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "用户信息无效", err)
	}
	link, err := decodeUserLink(req.User)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}

	user, err := h.loginLinkedUser(c, kbID, link)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, v1.AuthLoginUserLinkResp{
		Username: user.Account,
		Region:   user.Region,
		Role:     userLinkRoleName(user.Role),
	})
}

func (h *ShareAuthHandler) loginLinkedUser(c echo.Context, kbID string, link decodedUserLink) (*domain.User, error) {
	ctx := c.Request().Context()
	user, created, err := h.findOrCreateLinkUser(ctx, c, link)
	if err != nil {
		return nil, err
	}
	if err := matchUserLinkRole(user.Role, link.Role); err != nil {
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
	ctxWithEdition := context.WithValue(ctx, consts.ContextKeyEdition, consts.GetLicenseEdition(c))
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

func matchUserLinkRole(actual, requested consts.UserRole) error {
	if requested == consts.UserRoleGuest && actual != consts.UserRoleGuest {
		return fmt.Errorf("该用户名对应的不是普通用户")
	}
	if requested == consts.UserRoleUser && actual == consts.UserRoleAdmin {
		return fmt.Errorf("超级管理员不能通过此链接登录")
	}
	if requested == consts.UserRoleUser && actual != consts.UserRoleUser {
		return fmt.Errorf("该用户名对应的不是管理员")
	}
	return nil
}

func (h *ShareAuthHandler) findOrCreateLinkUser(
	ctx context.Context,
	c echo.Context,
	link decodedUserLink,
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
