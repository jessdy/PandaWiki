package share

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/share/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
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

type cloudGeneralArgument struct {
	IP        string `json:"ip"`
	LoginName string `json:"loginName,omitempty"`
	UserID    int64  `json:"userId,omitempty"`
}

type loginVerifyArg struct {
	Tgt string `json:"tgt"`
}

type loginVerifyReq struct {
	GeneralArgument cloudGeneralArgument `json:"generalArgument"`
	JSONArg         loginVerifyArg       `json:"jsonArg"`
}

type checkPermissionsArg struct {
	Permissions []string `json:"permissions"`
}

type checkPermissionsReq struct {
	GeneralArgument cloudGeneralArgument `json:"generalArgument"`
	JSONArg         checkPermissionsArg  `json:"jsonArg"`
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
	if err := h.rejectIfLoginLocked(c); err != nil {
		h.logger.Warn("tgt login rejected: locked",
			log.String("ip", c.RealIP()),
			log.String("kb_id", c.Request().Header.Get("X-KB-ID")),
		)
		return err
	}
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	var req v1.AuthLoginTgtReq
	if err := c.Bind(&req); err != nil {
		h.logger.Warn("tgt login bind failed", log.String("kb_id", kbID), log.String("ip", c.RealIP()), log.Error(err))
		return h.NewResponseWithError(c, "会话令牌无效", err)
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Warn("tgt login validate failed", log.String("kb_id", kbID), log.String("ip", c.RealIP()))
		return h.NewResponseWithError(c, "会话令牌不能为空", nil)
	}
	tgt := strings.TrimSpace(req.Tgt)
	if tgt == "" {
		return h.NewResponseWithError(c, "会话令牌不能为空", nil)
	}
	if len(tgt) > 1024 || strings.ContainsFunc(tgt, unicode.IsControl) {
		h.logger.Warn("tgt login rejected: invalid token format",
			log.String("kb_id", kbID),
			log.String("ip", c.RealIP()),
			log.Int("tgt_len", len(tgt)),
		)
		return h.NewResponseWithError(c, "会话令牌无效", nil)
	}
	baseURL, err := h.cloudAuthBaseURL(c)
	if err != nil {
		h.logger.Error("tgt login missing cloud auth base url", log.String("kb_id", kbID), log.Error(err))
		return h.NewResponseWithError(c, err.Error(), err)
	}

	client := newCloudAuthClient(baseURL)
	ip := strings.TrimSpace(c.RealIP())
	if ip == "" {
		ip = net.IPv4(127, 0, 0, 1).String()
	}
	h.logger.Info("tgt login start",
		log.String("kb_id", kbID),
		log.String("ip", ip),
		log.String("cloud_auth_host", cloudAuthHost(baseURL)),
		log.Int("tgt_len", len(tgt)),
	)
	sessionUser, err := client.loginVerify(c.Request().Context(), tgt, ip)
	if err != nil {
		h.rateLimiter.LockAttempt(c.Request().Context(), c.RealIP())
		h.logger.Error("tgt login verify failed",
			log.String("kb_id", kbID),
			log.String("ip", ip),
			log.String("cloud_auth_host", cloudAuthHost(baseURL)),
			log.Error(err),
		)
		return h.NewResponseWithError(c, publicCloudAuthMessage(err, "会话验证失败"), err)
	}
	c.Set(domain.AuditSubjectAccountSlot, sessionUser.LoginName)
	h.logger.Info("tgt login verify ok",
		log.String("kb_id", kbID),
		log.String("ip", ip),
		log.String("login_name", sessionUser.LoginName),
		log.Int64("platform_user_id", sessionUser.UserID),
	)
	allowed, err := client.hasKfsUserPermission(c.Request().Context(), sessionUser, ip)
	if err != nil {
		h.logger.Error("tgt permission check failed",
			log.String("kb_id", kbID),
			log.String("ip", ip),
			log.String("login_name", sessionUser.LoginName),
			log.Int64("platform_user_id", sessionUser.UserID),
			log.Error(err),
		)
		return h.NewResponseWithError(c, publicCloudAuthMessage(err, "权限验证失败"), err)
	}
	if !allowed {
		h.rateLimiter.LockAttempt(c.Request().Context(), c.RealIP())
		h.logger.Warn("tgt login permission denied",
			log.String("kb_id", kbID),
			log.String("ip", ip),
			log.String("login_name", sessionUser.LoginName),
			log.Int64("platform_user_id", sessionUser.UserID),
			log.String("required_permission", kfsUserPermission),
		)
		return h.NewResponseWithError(c, "权限验证失败", nil)
	}
	h.logger.Info("tgt login permission ok",
		log.String("kb_id", kbID),
		log.String("login_name", sessionUser.LoginName),
	)
	_ = h.rateLimiter.ResetLoginAttempts(c.Request().Context(), c.RealIP())

	user, err := h.loginLinkedUser(c, kbID, linkedLoginUser{
		Username: sessionUser.LoginName,
		Region:   tgtUserRegion,
		Role:     consts.UserRoleGuest,
	})
	if err != nil {
		h.logger.Error("tgt login linked user failed",
			log.String("kb_id", kbID),
			log.String("ip", ip),
			log.String("login_name", sessionUser.LoginName),
			log.Error(err),
		)
		return h.NewResponseWithError(c, err.Error(), err)
	}
	c.Set(domain.AuditSubjectAccountSlot, user.Account)
	h.logger.Info("tgt login success",
		log.String("kb_id", kbID),
		log.String("ip", ip),
		log.String("account", user.Account),
		log.String("role", string(user.Role)),
		log.String("region", user.Region),
	)
	return h.NewResponseWithData(c, v1.AuthLoginTgtResp{
		Username: user.Account,
		Region:   user.Region,
		Role:     linkedRoleName(user.Role),
	})
}

// cloudAuthHost 仅记录主机部分，避免把完整平台 URL 路径刷屏；从不记录 tgt。
func cloudAuthHost(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return ""
	}
	if i := strings.Index(baseURL, "://"); i >= 0 {
		baseURL = baseURL[i+3:]
	}
	if i := strings.IndexAny(baseURL, "/?"); i >= 0 {
		baseURL = baseURL[:i]
	}
	return baseURL
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
	if err := c.post(ctx, loginVerifyPath, loginVerifyReq{
		GeneralArgument: cloudGeneralArgument{IP: ip},
		JSONArg:         loginVerifyArg{Tgt: tgt},
	}, &resp); err != nil {
		return cloudSessionUser{}, fmt.Errorf("会话验证失败: %w", err)
	}
	if resp.Code != 0 {
		return cloudSessionUser{}, cloudAPIError("会话验证失败", resp.Message)
	}
	loginName := strings.TrimSpace(resp.Data.LoginName)
	if loginName == "" || resp.Data.UserID == 0 {
		return cloudSessionUser{}, fmt.Errorf("会话验证未返回用户信息")
	}
	trueName := strings.TrimSpace(resp.Data.TrueName)
	if trueName != "" {
		if err := validateCloudDisplayName(trueName); err != nil {
			return cloudSessionUser{}, err
		}
	}
	return cloudSessionUser{
		LoginName: loginName,
		TrueName:  trueName,
		UserID:    resp.Data.UserID,
	}, nil
}

func validateCloudDisplayName(name string) error {
	if utf8.RuneCountInString(name) > 64 {
		return fmt.Errorf("会话验证返回的姓名过长")
	}
	if strings.ContainsAny(name, "\x00\r\n") {
		return fmt.Errorf("会话验证返回的姓名非法")
	}
	return nil
}

func (c *cloudAuthClient) hasKfsUserPermission(ctx context.Context, user cloudSessionUser, ip string) (bool, error) {
	var resp struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    []string `json:"data"`
	}
	if err := c.post(ctx, checkPermissionsPath, checkPermissionsReq{
		GeneralArgument: cloudGeneralArgument{
			IP:        ip,
			LoginName: user.LoginName,
			UserID:    user.UserID,
		},
		JSONArg: checkPermissionsArg{Permissions: []string{kfsUserPermission}},
	}, &resp); err != nil {
		return false, fmt.Errorf("权限验证失败: %w", err)
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

// publicCloudAuthMessage 对外只回平台业务文案；网络/HTTP 细节留在日志。
func publicCloudAuthMessage(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	msg := err.Error()
	if strings.Contains(msg, ": status ") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "context deadline") {
		return fallback
	}
	return msg
}
