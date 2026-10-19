package share

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/share/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/log"
)

const (
	kfsUserPermission = "100000:function:kfsuser"
	tgtUserRegion     = "上海"

	loginVerifyPath      = "/CloudUserManagement/rest/UserManageRestService/loginVerify"
	checkPermissionsPath = "/CloudAuthorization/rest/CloudAuthorizationRestService/checkPermissions"
	cloudAuthTimeout     = 10 * time.Second
)

type cloudSessionUser struct {
	LoginName string
	TrueName  string
	UserID    int64
}

type cloudAuthClient struct {
	baseURL string
	http    *http.Client
}

func (h *ShareAuthHandler) cloudAuthBaseURL(c echo.Context) (string, error) {
	if h.cloudAuthUsecase != nil {
		baseURL, err := h.cloudAuthUsecase.BaseURL(c.Request().Context())
		if err != nil {
			return "", fmt.Errorf("读取平台地址失败")
		}
		if strings.TrimSpace(baseURL) != "" {
			return baseURL, nil
		}
	}
	if h.config != nil && strings.TrimSpace(h.config.CloudAuth.BaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(h.config.CloudAuth.BaseURL), "/"), nil
	}
	return "", fmt.Errorf("未配置会话验证地址")
}

func newCloudAuthClient(baseURL string) *cloudAuthClient {
	return &cloudAuthClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: cloudAuthTimeout},
	}
}

func (h *ShareAuthHandler) AuthLoginTgt(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	var req v1.AuthLoginTgtReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "会话令牌无效", err)
	}
	tgt := strings.TrimSpace(req.Tgt)
	if tgt == "" {
		return h.NewResponseWithError(c, "会话令牌不能为空", nil)
	}
	baseURL, err := h.cloudAuthBaseURL(c)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}

	client := newCloudAuthClient(baseURL)
	ip := strings.TrimSpace(c.RealIP())
	if ip == "" {
		ip = "127.0.0.1"
	}
	sessionUser, err := client.loginVerify(c.Request().Context(), tgt, ip)
	if err != nil {
		h.logger.Error("tgt login verify failed", log.Error(err))
		return h.NewResponseWithError(c, err.Error(), err)
	}
	allowed, err := client.hasKfsUserPermission(c.Request().Context(), sessionUser, ip)
	if err != nil {
		h.logger.Error("tgt permission check failed", log.String("login_name", sessionUser.LoginName), log.Error(err))
		return h.NewResponseWithError(c, err.Error(), err)
	}
	if !allowed {
		return h.NewResponseWithError(c, "权限验证失败", nil)
	}

	user, err := h.loginLinkedUser(c, kbID, decodedUserLink{
		Username: sessionUser.LoginName,
		Region:   tgtUserRegion,
		Role:     consts.UserRoleGuest,
	})
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, v1.AuthLoginTgtResp{
		Username: user.Account,
		Region:   user.Region,
		Role:     userLinkRoleName(user.Role),
	})
}

func (c *cloudAuthClient) loginVerify(ctx context.Context, tgt, ip string) (cloudSessionUser, error) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			LoginName string `json:"loginName"`
			TrueName  string `json:"trueName"`
			UserID    int64  `json:"userId"`
		} `json:"data"`
	}
	if err := c.post(ctx, loginVerifyPath, map[string]any{
		"generalArgument": map[string]any{"ip": ip},
		"jsonArg":         map[string]any{"tgt": tgt},
	}, &resp); err != nil {
		return cloudSessionUser{}, fmt.Errorf("会话验证失败")
	}
	if resp.Code != 0 {
		return cloudSessionUser{}, cloudAPIError("会话验证失败", resp.Message)
	}
	loginName := strings.TrimSpace(resp.Data.LoginName)
	if loginName == "" || resp.Data.UserID == 0 {
		return cloudSessionUser{}, fmt.Errorf("会话验证未返回用户信息")
	}
	return cloudSessionUser{
		LoginName: loginName,
		TrueName:  strings.TrimSpace(resp.Data.TrueName),
		UserID:    resp.Data.UserID,
	}, nil
}

func (c *cloudAuthClient) hasKfsUserPermission(ctx context.Context, user cloudSessionUser, ip string) (bool, error) {
	var resp struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    []string `json:"data"`
	}
	if err := c.post(ctx, checkPermissionsPath, map[string]any{
		"generalArgument": map[string]any{
			"ip":        ip,
			"loginName": user.LoginName,
			"userId":    user.UserID,
			"id":        user.UserID,
		},
		"jsonArg": map[string]any{
			"permissions": []string{kfsUserPermission},
		},
	}, &resp); err != nil {
		return false, fmt.Errorf("权限验证失败")
	}
	if resp.Code != 0 {
		return false, cloudAPIError("权限验证失败", resp.Message)
	}
	for _, permission := range resp.Data {
		if permission == kfsUserPermission {
			return true, nil
		}
	}
	return false, nil
}

func (c *cloudAuthClient) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	return nil
}

func cloudAPIError(prefix, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("%s", prefix)
	}
	return fmt.Errorf("%s：%s", prefix, message)
}
