package utils

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tjfoc/gmsm/sm3"
)

// 管理后台的图片由 <img> 直接加载，带不上 Bearer 头；登录用户拿到一个只对 /static-file/ 路径生效的
// HttpOnly Cookie，凭证只能读文件，不能调用其他接口。
const (
	StaticAccessCookie = "pw_static_token"
	StaticAccessTTL    = 12 * time.Hour
)

func SignStaticAccessToken(secret, userID string, expires time.Time) string {
	payload := userID + "." + strconv.FormatInt(expires.Unix(), 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + staticTokenMAC(secret, payload)
}

func VerifyStaticAccessToken(secret, token string, now time.Time) (string, bool) {
	if secret == "" {
		return "", false
	}
	encoded, mac, ok := strings.Cut(token, ".")
	if !ok {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	payload := string(raw)
	if !hmac.Equal([]byte(mac), []byte(staticTokenMAC(secret, payload))) {
		return "", false
	}
	i := strings.LastIndex(payload, ".")
	if i <= 0 {
		return "", false
	}
	exp, err := strconv.ParseInt(payload[i+1:], 10, 64)
	if err != nil || now.Unix() > exp {
		return "", false
	}
	return payload[:i], true
}

func staticTokenMAC(secret, payload string) string {
	mac := hmac.New(sm3.New, []byte(secret))
	mac.Write([]byte("panda-wiki/static-file|" + payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// SetStaticAccessCookie 会话级 Cookie，关闭浏览器即失效；令牌本身另有 12 小时有效期。
func SetStaticAccessCookie(c echo.Context, secret, userID string) {
	opts := SessionCookieOptions(c)
	c.SetCookie(&http.Cookie{
		Name:     StaticAccessCookie,
		Value:    SignStaticAccessToken(secret, userID, time.Now().Add(StaticAccessTTL)),
		Path:     "/static-file/",
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}
