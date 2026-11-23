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

// CookieSecure 决定会话 Cookie 是否带 Secure。
// 优先级：COOKIE_SECURE 显式开关 > 当前请求是否 HTTPS（TLS / X-Forwarded-Proto）>
// 无请求上下文时默认 false（避免中间件初始化阶段把 Secure 写死，导致 HTTP 站点浏览器丢弃 Cookie）。
func CookieSecure(c echo.Context) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE"))) {
	case "false", "0", "off":
		return false
	case "true", "1", "on":
		return true
	}
	if c != nil && c.Request() != nil {
		proto := strings.ToLower(strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Proto")))
		if i := strings.IndexByte(proto, ','); i >= 0 {
			proto = strings.TrimSpace(proto[:i])
		}
		if c.Request().TLS != nil || proto == "https" {
			return true
		}
		if proto == "http" {
			return false
		}
		// 无转发头且非 TLS：按明文 HTTP 处理，否则浏览器会静默丢弃 Secure Cookie
		return false
	}
	return false
}

func SessionCookieOptions(c echo.Context) *sessions.Options {
	return &sessions.Options{
		Path: "/",
		// 会话级 Cookie，避免永久 Cookie 存储敏感会话信息
		MaxAge:   0,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   CookieSecure(c),
	}
}

// authTokenCookiePath 管理端 JWT 仅需随 /api 请求发送，收窄 Path 避免整站可读。
const authTokenCookiePath = "/api"

// SetAuthTokenCookie 写入管理端 JWT（HttpOnly + SameSite=Strict + Path=/api）
func SetAuthTokenCookie(c echo.Context, token string, maxAge time.Duration) {
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}
	c.SetCookie(&http.Cookie{
		Name:     domain.AuthTokenCookieName,
		Value:    token,
		Path:     authTokenCookiePath,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   CookieSecure(c),
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearAuthTokenCookie 退出登录时清除 JWT Cookie（Path/SameSite 须与写入一致）
func ClearAuthTokenCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     domain.AuthTokenCookieName,
		Value:    "",
		Path:     authTokenCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   CookieSecure(c),
		SameSite: http.SameSiteStrictMode,
	})
}
