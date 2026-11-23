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
// 默认跟随请求是否 HTTPS（TLS / X-Forwarded-Proto）；无上下文时 false（避免中间件初始化写死 Secure）。
// 仅允许 COOKIE_SECURE=true 强制开启；禁止用环境变量强制关闭（防误配导致明文会话劫持）。
// 本地 HTTP 联调请勿伪造 X-Forwarded-Proto=https，或走真实明文请求让本函数自然返回 false。
func CookieSecure(c echo.Context) bool {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE"))); v == "true" || v == "1" || v == "on" {
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

// SessionCookieMaxAge 前台会话 Cookie / Redis TTL（秒）。
// 注意：boj/redistore 把 MaxAge<=0 当成“删除会话并写空 Cookie”，
// 不能用 gorilla 文档里的 MaxAge=0（浏览器会话级）语义。
const SessionCookieMaxAge = 30 * 60

func SessionCookieOptions(c echo.Context) *sessions.Options {
	return &sessions.Options{
		Path:     "/",
		MaxAge:   SessionCookieMaxAge,
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
		maxAge = 8 * time.Hour
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
