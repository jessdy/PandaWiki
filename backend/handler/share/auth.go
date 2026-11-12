package share

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/share/v1"
	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/pkg/ratelimit"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/usecase"
	"github.com/chaitin/panda-wiki/utils"
)

type ShareAuthHandler struct {
	*handler.BaseHandler
	logger           *log.Logger
	kbUsecase        *usecase.KnowledgeBaseUsecase
	authUsecase      *usecase.AuthUsecase
	userUsecase      *usecase.UserUsecase
	cloudAuthUsecase *usecase.CloudAuthUsecase
	config           *config.Config
	rateLimiter      *ratelimit.RateLimiter
}

func NewShareAuthHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	kbUsecase *usecase.KnowledgeBaseUsecase,
	authUsecase *usecase.AuthUsecase,
	userUsecase *usecase.UserUsecase,
	cloudAuthUsecase *usecase.CloudAuthUsecase,
	config *config.Config,
	cache *cache.Cache,
) *ShareAuthHandler {
	h := &ShareAuthHandler{
		BaseHandler:      baseHandler,
		logger:           logger.WithModule("handler.share.auth"),
		kbUsecase:        kbUsecase,
		authUsecase:      authUsecase,
		userUsecase:      userUsecase,
		cloudAuthUsecase: cloudAuthUsecase,
		config:           config,
		rateLimiter:      ratelimit.NewRateLimiter(logger, cache),
	}

	shareAuthMiddleware := middleware.NewShareAuthMiddleware(logger, kbUsecase)

	share := e.Group("share/v1/auth", shareAuthMiddleware.CheckForbidden)
	share.GET("/get", h.AuthGet)
	share.GET("/info", h.AuthInfo)
	share.POST("/login/simple", h.AuthLoginSimple)
	// 已移除 /login/user_password：禁止对外直接校验用户名密码（安全扫描成批分配/凭据暴露）。
	// 企业用户请走 /login/tgt 或 /login/user_link。
	share.POST("/login/user_link", h.AuthLoginUserLink)
	share.POST("/login/tgt", h.AuthLoginTgt)
	share.POST("/github", h.AuthGitHub)

	// PRO 前端使用的路由前缀：/share/pro/v1/auth/...
	// 为了兼容，注册一份 pro 路由，并同时为 v1/pro 路径提供 logout 接口。
	proShare := e.Group("share/pro/v1/auth", shareAuthMiddleware.CheckForbidden)
	proShare.POST("/logout", h.AuthLogout)
	share.POST("/logout", h.AuthLogout)

	return h
}

// AuthGet auth获取
//
//	@Tags			share_auth
//	@Summary		AuthGet
//	@Description	AuthGet
//	@ID				v1-AuthGet
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string			true	"kb_id"
//	@Param			param	query		v1.AuthGetReq	true	"para"
//	@Success		200		{object}	domain.PWResponse{data=v1.AuthGetResp}
//	@Router			/share/v1/auth/get [get]
func (h *ShareAuthHandler) AuthGet(c echo.Context) error {
	ctx := c.Request().Context()

	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	kb, err := h.kbUsecase.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get knowledge base detail", err)
	}

	resp := &v1.AuthGetResp{
		AuthType:       kb.AccessSettings.GetAuthType(),
		SourceType:     kb.AccessSettings.SourceType,
		LicenseEdition: consts.GetLicenseEdition(c),
	}
	return h.NewResponseWithData(c, resp)
}

// AuthInfo 当前会话身份；未登录时 data 为空
//
//	@Tags			share_auth
//	@Summary		AuthInfo
//	@Description	前端据此展示登录状态，不在浏览器存储中保存用户名
//	@ID				v1-AuthInfo
//	@Produce		json
//	@Param			X-KB-ID	header		string	true	"kb_id"
//	@Success		200		{object}	domain.PWResponse{data=v1.AuthInfoResp}
//	@Router			/share/v1/auth/info [get]
func (h *ShareAuthHandler) AuthInfo(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	sess, err := session.Get(domain.SessionName, c)
	if err != nil {
		return h.NewResponseWithData(c, nil)
	}
	if sessKBID, ok := sessionKbIDString(sess); !ok || sessKBID != kbID {
		return h.NewResponseWithData(c, nil)
	}
	userID, ok := shareAuthUserID(c, kbID)
	if !ok {
		// 简单口令会话没有账号身份，与前端 SITE_FEEDBACK_SIMPLE_GATE_USERNAME 保持一致
		return h.NewResponseWithData(c, v1.AuthInfoResp{Username: "访客"})
	}
	auth, err := h.authUsecase.GetAuthInfo(c.Request().Context(), kbID, userID)
	if err != nil {
		h.logger.Warn("get auth info failed", log.Error(err))
		return h.NewResponseWithData(c, nil)
	}
	return h.NewResponseWithData(c, v1.AuthInfoResp{
		ID:        auth.ID,
		Username:  auth.UserInfo.Username,
		Email:     auth.UserInfo.Email,
		AvatarUrl: auth.UserInfo.AvatarUrl,
	})
}

// AuthLoginSimple 简单口令登录
//
//	@Tags			share_auth
//	@Summary		AuthLoginSimple
//	@Description	AuthLoginSimple
//	@ID				v1-AuthLoginSimple
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string					true	"kb_id"
//	@Param			param	body		v1.AuthLoginSimpleReq	true	"para"
//	@Success		200		{object}	domain.Response
//	@Router			/share/v1/auth/login/simple [post]
func (h *ShareAuthHandler) AuthLoginSimple(c echo.Context) error {
	ctx := c.Request().Context()

	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	if err := h.rejectIfLoginLocked(c); err != nil {
		return err
	}

	var req v1.AuthLoginSimpleReq
	if err := c.Bind(&req); err != nil {
		h.logger.Error("parse request failed", log.Error(err))
		return h.NewResponseWithError(c, "AuthGet bind failed", nil)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "密码不能为空", nil)
	}

	kb, err := h.kbUsecase.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get knowledge base detail", err)
	}

	if !kb.AccessSettings.SimpleAuth.Enabled {
		return h.NewResponseWithError(c, "simple auth is not enabled", nil)
	}

	if !utils.VerifySecret(kb.AccessSettings.SimpleAuth.Password, req.Password) {
		h.rateLimiter.LockAttempt(ctx, c.RealIP())
		return h.NewResponseWithError(c, "simple auth password is incorrect", nil)
	}
	_ = h.rateLimiter.ResetLoginAttempts(ctx, c.RealIP())

	s := c.Get(domain.SessionStoreName)
	if s == nil {
		return h.NewResponseWithError(c, "get session cache key failed", nil)
	}
	store := s.(sessions.Store)

	newSess := sessions.NewSession(store, domain.SessionName)
	newSess.IsNew = true

	newSess.Options = utils.SessionCookieOptions(c)

	newSess.Values["kb_id"] = kb.ID

	if err := newSess.Save(c.Request(), c.Response()); err != nil {
		return h.NewResponseWithError(c, "save session failed", nil)
	}

	return h.NewResponseWithData(c, nil)
}

// AuthGitHub GitHub登录
//
//	@Tags			ShareAuth
//	@Summary		GitHub登录
//	@Description	GitHub登录
//	@ID				v1-AuthGitHub
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string				true	"kb id"
//	@Param			param	body		v1.AuthGitHubReq	true	"para"
//	@Success		200		{object}	domain.PWResponse{data=v1.AuthGitHubResp}
//	@Router			/share/v1/auth/github [post]
func (h *ShareAuthHandler) AuthGitHub(c echo.Context) error {
	if err := h.rejectIfLoginLocked(c); err != nil {
		return err
	}
	ctx := context.WithValue(c.Request().Context(), consts.CtxEdition, consts.GetLicenseEdition(c))

	var req v1.AuthGitHubReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	req.KbID = kbID

	valid, err := h.authUsecase.ValidateRedirectUrl(ctx, req.KbID, req.RedirectUrl)
	if err != nil || !valid {
		return h.NewResponseWithError(c, "invalid redirect url", err)
	}

	url, err := h.authUsecase.GenerateGitHubAuthUrl(ctx, req)
	if err != nil {
		return h.NewResponseWithError(c, "GenerateGitHubAuthUrl failed", err)
	}

	return h.NewResponseWithData(c, v1.AuthGitHubResp{
		Url: url,
	})
}

// AuthLogout 退出登录（清除 share 会话）
//
//	@Tags			share_auth
//	@Summary		用户登出
//	@Description	清除 share 会话 cookie，前端通常调用 /share/pro/v1/auth/logout
//	@ID				v1-AuthLogout
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	domain.Response
//	@Router			/share/pro/v1/auth/logout [post]
func (h *ShareAuthHandler) AuthLogout(c echo.Context) error {
	if middleware.RejectCrossSiteRequest(c) {
		return c.JSON(http.StatusForbidden, domain.PWResponse{Success: false, Message: "forbidden cross-site request"})
	}
	// 使用 echo-contrib/session 获取当前会话
	sess, err := session.Get(domain.SessionName, c)
	if err != nil {
		h.logger.Error("session get failed on logout", log.Error(err))
		// 即便获取失败，也返回 200，前端只关心本地 cookie 状态
		return h.NewResponseWithData(c, nil)
	}

	// 通过设置 MaxAge<0 来让浏览器删除 cookie
	opts := utils.SessionCookieOptions(c)
	opts.MaxAge = -1
	sess.Options = opts
	sess.Values = map[interface{}]interface{}{}

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		h.logger.Error("session save failed on logout", log.Error(err))
		// 不抛给前端错误，避免卡在登出界面
	}

	return h.NewResponseWithData(c, nil)
}

func (h *ShareAuthHandler) rejectIfLoginLocked(c echo.Context) error {
	locked, remaining := h.rateLimiter.CheckIPLocked(c.Request().Context(), c.RealIP())
	if !locked {
		return nil
	}
	return h.NewResponseWithError(c, fmt.Sprintf("登录失败次数过多，请 %s 后重试", remaining.Round(time.Second)), nil)
}
