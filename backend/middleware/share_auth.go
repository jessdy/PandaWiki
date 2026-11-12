package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/usecase"
)

// rejectCrossSiteRequest 对 Cookie 会话做 Origin/Referer 同源校验，缓解 CSRF。
// 写操作校验 Origin，缺失时退回 Referer；读操作只校验 Origin（浏览器跨站 fetch/XHR 必带 Origin，顶层导航不带）。
func rejectCrossSiteRequest(c echo.Context) bool {
	req := c.Request()
	allowedHosts := []string{req.Host}
	for _, h := range strings.Split(req.Header.Get("X-Forwarded-Host"), ",") {
		if h = strings.TrimSpace(h); h != "" {
			allowedHosts = append(allowedHosts, h)
		}
	}
	sameHost := func(raw string) bool {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || u.Host == "" {
			return false
		}
		for _, h := range allowedHosts {
			if strings.EqualFold(u.Host, h) {
				return true
			}
		}
		return false
	}
	if origin := req.Header.Get("Origin"); origin != "" {
		return !sameHost(origin)
	}
	switch req.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		if referer := req.Header.Get("Referer"); referer != "" {
			return !sameHost(referer)
		}
	}
	// 无 Origin/Referer 的非浏览器客户端（如服务间调用）放行
	return false
}

type ShareAuthMiddleware struct {
	logger    *log.Logger
	kbUsecase *usecase.KnowledgeBaseUsecase
}

func NewShareAuthMiddleware(logger *log.Logger, kbUsecase *usecase.KnowledgeBaseUsecase) *ShareAuthMiddleware {
	return &ShareAuthMiddleware{
		logger:    logger.WithModule("middleware.share_auth"),
		kbUsecase: kbUsecase,
	}
}

func (h *ShareAuthMiddleware) CheckForbidden(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		kbID := c.Request().Header.Get("X-KB-ID")
		if kbID == "" {
			h.logger.Error("kb_id is empty")
			return c.JSON(http.StatusBadRequest, domain.PWResponse{
				Success: false,
				Message: "kb_id is required",
			})
		}

		kb, err := h.kbUsecase.GetKnowledgeBase(c.Request().Context(), kbID)
		if err != nil {
			h.logger.Error("get knowledge base failed", log.String("kb_id", kbID), log.Error(err))
			sentry.CaptureException(err)
			return c.JSON(http.StatusInternalServerError, domain.PWResponse{
				Success: false,
				Message: "failed to get knowledge base detail",
			})
		}

		if kb.AccessSettings.IsForbidden {
			h.logger.Warn("access forbidden", log.String("kb_id", kbID))
			return c.JSON(http.StatusForbidden, domain.PWResponse{
				Success: false,
				Message: "access is forbidden",
			})
		}

		return next(c)
	}
}

func (h *ShareAuthMiddleware) Authorize(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if rejectCrossSiteRequest(c) {
			h.logger.Warn("rejected cross-site share request",
				log.String("method", c.Request().Method),
				log.String("path", c.Request().URL.Path),
				log.String("origin", c.Request().Header.Get("Origin")),
			)
			return c.JSON(http.StatusForbidden, domain.PWResponse{
				Success: false,
				Message: "forbidden cross-site request",
			})
		}

		kbID := c.Request().Header.Get("X-KB-ID")
		if kbID == "" {
			h.logger.Error("kb_id is empty")
			return c.JSON(http.StatusUnauthorized, domain.PWResponse{
				Success: false,
				Message: "Unauthorized",
			})
		}

		kb, err := h.kbUsecase.GetKnowledgeBase(c.Request().Context(), kbID)
		if err != nil {
			h.logger.Error("get knowledge base failed", log.String("kb_id", kbID), log.Error(err))
			return c.JSON(http.StatusUnauthorized, domain.PWResponse{
				Success: false,
				Message: "Unauthorized",
			})
		}

		if kb.AccessSettings.IsForbidden {
			h.logger.Warn("access forbidden", log.String("kb_id", kbID))
			return c.JSON(http.StatusForbidden, domain.PWResponse{
				Success: false,
				Message: "access is forbidden",
			})
		}

		// 未开启认证
		if !kb.AccessSettings.EnterpriseAuth.Enabled && !kb.AccessSettings.SimpleAuth.Enabled {
			return next(c)
		}

		sess, err := session.Get(domain.SessionName, c)
		if err != nil {
			h.logger.Error("session get failed", log.Error(err))
			return c.JSON(http.StatusUnauthorized, domain.PWResponse{
				Success: false,
				Message: "Unauthorized",
			})
		}

		KbIDSess, ok := sess.Values["kb_id"].(string)
		if !ok || kbID == "" || KbIDSess != kb.ID {
			h.logger.Error("kb_id valid failed", log.Error(err))
			return c.JSON(http.StatusUnauthorized, domain.PWResponse{
				Success: false,
				Message: "Unauthorized",
			})
		}

		// 企业认证
		if kb.AccessSettings.EnterpriseAuth.Enabled {
			userId, ok := sess.Values["user_id"].(uint)
			if !ok || userId == 0 {
				h.logger.Error("session user_id get failed", log.Error(err))
				return c.JSON(http.StatusUnauthorized, domain.PWResponse{
					Success: false,
					Message: "Unauthorized",
				})
			}
			c.Set("user_id", userId)
			return next(c)
		}

		return next(c)
	}
}
