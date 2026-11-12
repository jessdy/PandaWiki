package middleware

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/boj/redistore"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/utils"
)

const (
	SessionKey = "SessionKey"
)

type SessionMiddleware struct {
	logger *log.Logger
	store  *redistore.RediStore
}

func NewSessionMiddleware(logger *log.Logger, config *config.Config, cache *cache.Cache) (*SessionMiddleware, error) {

	secretKey, err := cache.GetOrSet(context.Background(), SessionKey, uuid.New().String(), time.Duration(0))
	if err != nil {
		logger.Error("session store create secret key failed: %v", log.Error(err))
		return nil, err
	}

	store, err := redistore.NewRediStore(
		10,
		"tcp",
		config.Redis.Addr,
		"",
		config.Redis.Password,
		[]byte(secretKey.(string)),
	)

	if err != nil {
		logger.Error("init session store failed: %v", log.Error(err))
		return nil, err
	}

	// 服务端会话最长 30 分钟无操作失效（Cookie 仍为会话级 MaxAge=0）
	store.SetMaxAge(30 * 60)

	store.Options = &sessions.Options{
		Path: "/",
		// 会话级 Cookie（关闭浏览器即失效），避免永久 Cookie 存会话信息
		MaxAge:   0,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		// 默认 Secure；COOKIE_SECURE=false 时关闭（见 utils.CookieSecure）
		Secure: utils.CookieSecure(nil) || os.Getenv("COOKIE_SECURE") == "true",
	}

	return &SessionMiddleware{
		logger: logger.WithModule("middleware.session"),
		store:  store,
	}, nil
}

func (s *SessionMiddleware) Session() echo.MiddlewareFunc {
	return session.MiddlewareWithConfig(session.Config{
		Store: s.store,
	})
}
