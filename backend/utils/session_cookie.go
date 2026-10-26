package utils

import (
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

func SessionCookieOptions(c echo.Context) *sessions.Options {
	secure := false
	if c != nil && c.Request() != nil {
		secure = c.Request().TLS != nil || strings.EqualFold(c.Request().Header.Get("X-Forwarded-Proto"), "https")
	}
	if os.Getenv("COOKIE_SECURE") == "true" {
		secure = true
	}
	return &sessions.Options{
		Path:     "/",
		MaxAge:   7 * 86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}
