package utils

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
)

// CookieSecure 默认开启 Secure；仅当显式 COOKIE_SECURE=false 时关闭（本地明文 HTTP 调试）。
// 若未设置 COOKIE_SECURE，则在 TLS 或 X-Forwarded-Proto=https 时置 Secure。
func CookieSecure(c echo.Context) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE"))) {
	case "false", "0", "off":
		return false
	case "true", "1", "on":
		return true
	}
	if c != nil && c.Request() != nil {
		if c.Request().TLS != nil || strings.EqualFold(c.Request().Header.Get("X-Forwarded-Proto"), "https") {
			return true
		}
	}
	// 生产部署默认强制 Secure，防止明文 Cookie
	return os.Getenv("APP_ENV") != "dev" && os.Getenv("APP_ENV") != "development"
}

func SessionCookieOptions(c echo.Context) *sessions.Options {
	return &sessions.Options{
		Path:     "/",
		// 会话级 Cookie，避免永久 Cookie 存储敏感会话信息
		MaxAge:   0,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   CookieSecure(c),
	}
}

// SetAuthTokenCookie 写入管理端 JWT（HttpOnly + SameSite=Lax）
func SetAuthTokenCookie(c echo.Context, token string, maxAge time.Duration) {
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}
	c.SetCookie(&http.Cookie{
		Name:     domain.AuthTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   CookieSecure(c),
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearAuthTokenCookie 退出登录时清除 JWT Cookie
func ClearAuthTokenCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     domain.AuthTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   CookieSecure(c),
		SameSite: http.SameSiteLaxMode,
	})
}
