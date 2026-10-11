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
}

func decodeUserLink(raw string) (string, string, error) {
	payload, err := decodeUserLinkBase64(raw)
	if err != nil {
		return "", "", fmt.Errorf("用户信息无法解析")
	}

	var data userLinkPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", "", fmt.Errorf("用户信息格式无效")
	}

	username := firstNonEmpty(data.Username, data.UserName, data.UsernameCN)
	region := firstNonEmpty(data.Region, data.RegionCN)
	username = strings.TrimSpace(username)
	region = strings.TrimSpace(region)
	if username == "" {
		return "", "", fmt.Errorf("用户名不能为空")
	}
	if utf8.RuneCountInString(username) > maxUserLinkUsernameRunes {
		return "", "", fmt.Errorf("用户名不能超过 %d 个字符", maxUserLinkUsernameRunes)
	}
	if utf8.RuneCountInString(region) > maxUserLinkRegionRunes {
		return "", "", fmt.Errorf("地区不能超过 %d 个字符", maxUserLinkRegionRunes)
	}
	return username, region, nil
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
	ctx := c.Request().Context()
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	var req v1.AuthLoginUserLinkReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "用户信息无效", err)
	}
	username, region, err := decodeUserLink(req.User)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}

	user, created, err := h.findOrCreateLinkUser(ctx, c, username, region)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	if user.Role != consts.UserRoleGuest {
		return h.NewResponseWithError(c, "该用户名对应的不是前台用户", nil)
	}
	if created {
		if err := h.addUserToAllGroups(ctx, kbID, user.ID); err != nil {
			h.logger.Error("failed to add linked user to groups", log.Error(err))
			return h.NewResponseWithError(c, "加入用户组失败", err)
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
		return h.NewResponseWithError(c, "创建登录会话失败", err)
	}
	if err := h.authUsecase.SaveNewSession(c, auth); err != nil {
		h.logger.Error("failed to save session", log.Error(err))
		return h.NewResponseWithError(c, "保存登录会话失败", err)
	}
	return h.NewResponseWithData(c, v1.AuthLoginUserLinkResp{
		Username: user.Account,
		Region:   user.Region,
	})
}

func (h *ShareAuthHandler) findOrCreateLinkUser(
	ctx context.Context,
	c echo.Context,
	username string,
	region string,
) (*domain.User, bool, error) {
	user, err := h.userUsecase.GetUserByAccount(ctx, username)
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
		Account:  username,
		Password: password,
		Role:     consts.UserRoleGuest,
		Region:   region,
	}
	if err := h.userUsecase.CreateUser(ctx, user, consts.GetLicenseEdition(c)); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") {
			existing, getErr := h.userUsecase.GetUserByAccount(ctx, username)
			if getErr == nil {
				return existing, false, nil
			}
		}
		return nil, false, fmt.Errorf("创建用户失败")
	}
	return user, true, nil
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
