package middleware

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/boj/redistore"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/utils"
)

type SessionMiddleware struct {
	logger *log.Logger
	store  *redistore.RediStore
}

func NewSessionMiddleware(logger *log.Logger, config *config.Config, cache *cache.Cache) (*SessionMiddleware, error) {
	_ = cache // 保留注入以兼容 wire；会话密钥不再写入 Redis
	secretKey, err := sessionSigningSecret(config)
	if err != nil {
		logger.Error("session store resolve secret failed: %v", log.Error(err))
		return nil, err
	}

	store, err := redistore.NewRediStore(
		10,
		"tcp",
		config.Redis.Addr,
		"",
		config.Redis.Password,
		secretKey,
	)
	if err != nil {
		logger.Error("init session store failed: %v", log.Error(err))
		return nil, err
	}

	// 与 Cookie MaxAge 对齐；redistore 在 MaxAge<=0 时会删会话并写空 Cookie
	store.SetMaxAge(utils.SessionCookieMaxAge)

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   utils.SessionCookieMaxAge,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		// 实际 Secure 以每次 Save 时 SessionCookieOptions(c) 为准（按请求是否 HTTPS）
		Secure: utils.CookieSecure(nil),
	}

	return &SessionMiddleware{
		logger: logger.WithModule("middleware.session"),
		store:  store,
	}, nil
}

// sessionSigningSecret 优先 SESSION_SECRET，其次 DATA_ENCRYPTION_KEY，再次 JWT_SECRET；
// 派生固定长度密钥，避免把可轮换的主密钥原文直接当 Cookie 签名材料。
func sessionSigningSecret(cfg *config.Config) ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SESSION_SECRET"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("DATA_ENCRYPTION_KEY"))
	}
	if raw == "" && cfg != nil {
		raw = strings.TrimSpace(cfg.Auth.JWT.Secret)
	}
	if len(raw) < 16 {
		return nil, fmt.Errorf("SESSION_SECRET / DATA_ENCRYPTION_KEY / JWT_SECRET 过短，无法初始化会话签名")
	}
	sum := sha256.Sum256([]byte("panda-wiki-session-v1:" + raw))
	return sum[:], nil
}

func (s *SessionMiddleware) Session() echo.MiddlewareFunc {
	return session.MiddlewareWithConfig(session.Config{
		Store: s.store,
	})
}
