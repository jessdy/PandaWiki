package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeaders 补充基础安全响应头（应对扫描中 CSP / X-Content-Type-Options / Referrer-Policy 缺失）。
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			if h.Get("X-Content-Type-Options") == "" {
				h.Set("X-Content-Type-Options", "nosniff")
			}
			if h.Get("Referrer-Policy") == "" {
				h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			}
			if h.Get("X-Frame-Options") == "" {
				h.Set("X-Frame-Options", "SAMEORIGIN")
			}
			// 保守 CSP：不打断现有前端资源加载；禁止默认对象插件
			if h.Get("Content-Security-Policy") == "" {
				h.Set("Content-Security-Policy", "object-src 'none'; base-uri 'self'; frame-ancestors 'self'")
			}
			// HSTS：仅在 HTTPS 请求上下发，nginx 已配置时不重复覆盖
			if c.IsTLS() || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
				if h.Get("Strict-Transport-Security") == "" {
					h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
				}
			}
			return next(c)
		}
	}
}
