package utils

import (
	"crypto/hmac"
	"crypto/rand"
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
	staticNonceLen     = 8
)

// SignStaticAccessToken 把用户标识和过期时间密封进令牌。
// Cookie 值是随机 nonce + 流加密密文，不能从中直接读出用户 ID。
func SignStaticAccessToken(secret, userID string, expires time.Time) string {
	nonce := make([]byte, staticNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return ""
	}
	plain := []byte(userID + "|" + strconv.FormatInt(expires.Unix(), 10))
	blob := append(append([]byte{}, nonce...), xorStream(secret, nonce, plain)...)
	enc := base64.RawURLEncoding.EncodeToString(blob)
	return enc + "." + staticTokenMAC(secret, enc)
}

func VerifyStaticAccessToken(secret, token string, now time.Time) (string, bool) {
	if secret == "" {
		return "", false
	}
	encoded, mac, ok := strings.Cut(token, ".")
	if !ok || encoded == "" || mac == "" {
		return "", false
	}
	if !hmac.Equal([]byte(mac), []byte(staticTokenMAC(secret, encoded))) {
		return "", false
	}
	blob, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(blob) <= staticNonceLen {
		return "", false
	}
	nonce, ct := blob[:staticNonceLen], blob[staticNonceLen:]
	plain := xorStream(secret, nonce, ct)
	payload := string(plain)
	i := strings.LastIndex(payload, "|")
	if i <= 0 {
		return "", false
	}
	exp, err := strconv.ParseInt(payload[i+1:], 10, 64)
	if err != nil || now.Unix() > exp {
		return "", false
	}
	return payload[:i], true
}

func xorStream(secret string, nonce, data []byte) []byte {
	out := make([]byte, len(data))
	filled := 0
	for counter := 0; filled < len(data); counter++ {
		block := staticTokenMAC(secret, "seal|"+hex.EncodeToString(nonce)+"|"+strconv.Itoa(counter))
		for i := 0; i < len(block) && filled < len(data); i++ {
			out[filled] = data[filled] ^ block[i]
			filled++
		}
	}
	return out
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
