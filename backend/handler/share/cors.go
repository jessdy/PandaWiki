package share

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

// shareCORSMiddleware 仅回显与当前请求 Host 一致的 Origin，禁止 Access-Control-Allow-Origin: *。
func shareCORSMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		applyShareCORS(c)
		if c.Request().Method == http.MethodOptions {
			return c.NoContent(http.StatusOK)
		}
		return next(c)
	}
}

func applyShareCORS(c echo.Context) {
	origin := strings.TrimSpace(c.Request().Header.Get("Origin"))
	if origin == "" {
		return
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return
	}
	reqHost := c.Request().Host
	if !strings.EqualFold(u.Host, reqHost) {
		return
	}
	h := c.Response().Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Vary", "Origin")
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, X-KB-ID")
}
