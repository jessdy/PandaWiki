package share

import (
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
	echosession "github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
)

// shareAuthUserID 优先读取中间件注入的用户 ID；普通用户名密码登录在未开启企业认证时，
// 中间件不会注入该值，因此还需从已校验知识库归属的 Share Session 中读取。
func shareAuthUserID(c echo.Context, kbID string) (uint, bool) {
	if value := c.Get("user_id"); value != nil {
		if userID, ok := value.(uint); ok && userID != 0 {
			return userID, true
		}
	}

	sess, err := echosession.Get(domain.SessionName, c)
	if err != nil {
		return 0, false
	}
	sessionKBID, ok := sessionKbIDString(sess)
	if !ok || sessionKBID != kbID {
		return 0, false
	}
	userIDString, ok := sessionAuthUserIDString(sess)
	if !ok {
		return 0, false
	}
	userID, err := strconv.ParseUint(userIDString, 10, 0)
	if err != nil || userID == 0 {
		return 0, false
	}
	return uint(userID), true
}

// sessionAuthUserIDString 从 gorilla session 中解析 auths 主键（与 SaveNewSession 写入的 user_id 一致）。
// Redis 等存储反序列化后类型可能不是 uint，需兼容多种数值与字符串形态。
func sessionAuthUserIDString(sess *sessions.Session) (string, bool) {
	v, ok := sess.Values["user_id"]
	if !ok || v == nil {
		return "", false
	}
	switch x := v.(type) {
	case uint:
		if x == 0 {
			return "", false
		}
		return strconv.FormatUint(uint64(x), 10), true
	case uint32:
		if x == 0 {
			return "", false
		}
		return strconv.FormatUint(uint64(x), 10), true
	case uint64:
		if x == 0 {
			return "", false
		}
		return strconv.FormatUint(x, 10), true
	case int:
		if x <= 0 {
			return "", false
		}
		return strconv.FormatInt(int64(x), 10), true
	case int32:
		if x <= 0 {
			return "", false
		}
		return strconv.FormatInt(int64(x), 10), true
	case int64:
		if x <= 0 {
			return "", false
		}
		return strconv.FormatInt(x, 10), true
	case float64:
		if x <= 0 || x != float64(int64(x)) {
			return "", false
		}
		return strconv.FormatInt(int64(x), 10), true
	case float32:
		if x <= 0 {
			return "", false
		}
		return strconv.FormatInt(int64(x), 10), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return "", false
		}
		return s, true
	default:
		return "", false
	}
}

func sessionKbIDString(sess *sessions.Session) (string, bool) {
	v, ok := sess.Values["kb_id"]
	if !ok || v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return "", false
		}
		return s, true
	case []byte:
		s := strings.TrimSpace(string(x))
		if s == "" {
			return "", false
		}
		return s, true
	default:
		return "", false
	}
}
