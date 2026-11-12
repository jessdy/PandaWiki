package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echoMiddleware "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/cache"
)

type JWTMiddleware struct {
	config         *config.Config
	jwtMiddleware  echo.MiddlewareFunc
	logger         *log.Logger
	userAccessRepo *pg.UserAccessRepository
	apiTokenRepo   *pg.APITokenRepo
	cache          *cache.Cache
}

func revokedTokenKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "pw:jwt:revoked:" + hex.EncodeToString(sum[:])
}

// RevokeToken 退出登录后令牌立即失效，直到其原本的过期时间
func (m *JWTMiddleware) RevokeToken(ctx context.Context, c echo.Context) error {
	token := extractRequestToken(c)
	jwtToken, ok := c.Get("user").(*jwt.Token)
	if token == "" || !ok || jwtToken == nil {
		return nil
	}
	ttl := 24 * time.Hour
	if exp, err := jwtToken.Claims.GetExpirationTime(); err == nil && exp != nil {
		ttl = time.Until(exp.Time)
	}
	if ttl <= 0 {
		return nil
	}
	return m.cache.Set(ctx, revokedTokenKey(token), 1, ttl).Err()
}

func (m *JWTMiddleware) tokenRevoked(ctx context.Context, token string) bool {
	n, err := m.cache.Exists(ctx, revokedTokenKey(token)).Result()
	if err != nil {
		m.logger.Error("check revoked token failed", log.Error(err))
		return false
	}
	return n > 0
}

func NewJWTMiddleware(config *config.Config, logger *log.Logger, userAccessRepo *pg.UserAccessRepository, apiTokenRepo *pg.APITokenRepo, cache *cache.Cache) *JWTMiddleware {
	jwtMiddleware := echoMiddleware.WithConfig(echoMiddleware.Config{
		SigningKey:  []byte(config.Auth.JWT.Secret),
		TokenLookup: "header:Authorization:Bearer ,cookie:" + domain.AuthTokenCookieName,
		ErrorHandler: func(c echo.Context, err error) error {
			logger.Error("jwt auth failed", log.Error(err))
			return c.JSON(http.StatusUnauthorized, domain.PWResponse{
				Success: false,
				Message: "Unauthorized",
			})
		},
	})
	return &JWTMiddleware{
		config:         config,
		jwtMiddleware:  jwtMiddleware,
		logger:         logger.WithModule("middleware.jwt"),
		userAccessRepo: userAccessRepo,
		apiTokenRepo:   apiTokenRepo,
		cache:          cache,
	}
}

func extractRequestToken(c echo.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	if cookie, err := c.Cookie(domain.AuthTokenCookieName); err == nil && cookie != nil {
		return cookie.Value
	}
	return ""
}

func unauthorized(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, domain.PWResponse{
		Success: false,
		Message: "Unauthorized",
	})
}

func (m *JWTMiddleware) Authorize(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		token := extractRequestToken(c)
		if strings.HasPrefix(authHeader, "Bearer ") && token != "" && !strings.Contains(token, ".") {
			return m.validateAPIToken(c, token, next)
		}

		return m.jwtMiddleware(func(c echo.Context) error {
			userID, ok := m.MustGetUserID(c)
			if !ok {
				return unauthorized(c)
			}
			token = extractRequestToken(c)
			if m.tokenRevoked(c.Request().Context(), token) {
				return unauthorized(c)
			}
			user, err := m.userAccessRepo.GetAuthUser(userID)
			if err != nil {
				m.logger.Warn("jwt user not found", log.String("user_id", userID), log.Error(err))
				return unauthorized(c)
			}
			c.Set(domain.AuthUserKey, user)
			if !roleAllowsRequest(user.Role, c.Request().Method, c.Request().URL.Path) {
				return c.JSON(http.StatusForbidden, domain.PWResponse{
					Success: false,
					Message: "当前账号无权访问该功能",
				})
			}

			ctx := context.WithValue(c.Request().Context(), domain.CtxAuthInfoSlot, &domain.CtxAuthInfo{
				IsToken:    false,
				Permission: consts.UserKBPermissionNull,
				UserId:     userID,
			})
			c.SetRequest(c.Request().WithContext(ctx))

			m.userAccessRepo.UpdateAccessTime(userID)
			return next(c)
		})(c)
	}
}

// ValidateExactRole 只允许指定角色访问，系统管理员不享有豁免（用于三员各自的专属功能）
func (m *JWTMiddleware) ValidateExactRole(roles ...consts.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get(domain.AuthUserKey).(*domain.User)
			if !ok || user == nil || !slices.Contains(roles, user.Role) {
				return c.JSON(http.StatusForbidden, domain.PWResponse{
					Success: false,
					Message: "当前账号无权访问该功能",
				})
			}
			return next(c)
		}
	}
}

// validateAPIToken validates API token and sets user context
func (m *JWTMiddleware) validateAPIToken(c echo.Context, token string, next echo.HandlerFunc) error {
	if m.apiTokenRepo == nil {
		m.logger.Debug("API token repository not available")
		return c.JSON(http.StatusUnauthorized, domain.PWResponse{
			Success: false,
			Message: "Unauthorized",
		})
	}

	apiToken, err := m.apiTokenRepo.GetByTokenWithCache(c.Request().Context(), token)
	if err != nil || apiToken == nil {
		m.logger.Error("failed to get API token", log.Error(err))
		return c.JSON(http.StatusUnauthorized, domain.PWResponse{
			Success: false,
			Message: "Unauthorized",
		})
	}

	ctx := context.WithValue(c.Request().Context(), domain.CtxAuthInfoSlot, &domain.CtxAuthInfo{
		IsToken:    true,
		Permission: apiToken.Permission,
		UserId:     apiToken.UserID,
		KBId:       apiToken.KbId,
	})

	req := c.Request().WithContext(ctx)
	c.SetRequest(req)

	return next(c)
}

func (m *JWTMiddleware) ValidateUserRole(role consts.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authInfo := domain.GetAuthInfoFromCtx(c.Request().Context())
			if authInfo == nil {
				return c.JSON(http.StatusUnauthorized, domain.PWResponse{
					Success: false,
					Message: "Unauthorized",
				})
			}

			if authInfo.IsToken {
				// token 视为普通用户 没有管理员相关权限
				if role == consts.UserRoleAdmin {
					return c.JSON(http.StatusUnauthorized, domain.PWResponse{
						Success: false,
						Message: "token not support admin role",
					})
				}
			} else {
				valid, err := m.userAccessRepo.ValidateRole(authInfo.UserId, role)

				if err != nil || !valid {
					m.logger.Error("ValidateRole check", log.Any("user_id", authInfo.UserId), log.Any("valid", valid))
					return c.JSON(http.StatusForbidden, domain.PWResponse{
						Success: false,
						Message: "StatusForbidden ValidateRole",
					})
				}
			}

			return next(c)
		}
	}
}

func (m *JWTMiddleware) ValidateUserRoleOrAnyKBPerm(role consts.UserRole, perm consts.UserKBPermission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authInfo := domain.GetAuthInfoFromCtx(c.Request().Context())
			if authInfo == nil {
				return c.JSON(http.StatusUnauthorized, domain.PWResponse{
					Success: false,
					Message: "Unauthorized",
				})
			}

			if authInfo.IsToken {
				return c.JSON(http.StatusForbidden, domain.PWResponse{
					Success: false,
					Message: "token not supported",
				})
			}

			valid, err := m.userAccessRepo.ValidateRole(authInfo.UserId, role)
			if err == nil && valid {
				return next(c)
			}

			validPerm, err := m.userAccessRepo.ValidateAnyKBPerm(authInfo.UserId, perm)
			if err != nil || !validPerm {
				m.logger.Error("ValidateUserRoleOrAnyKBPerm check failed",
					log.Any("user_id", authInfo.UserId))
				return c.JSON(http.StatusForbidden, domain.PWResponse{
					Success: false,
					Message: "StatusForbidden",
				})
			}

			return next(c)
		}
	}
}

func (m *JWTMiddleware) ValidateKBUserPerm(perm consts.UserKBPermission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authInfo := domain.GetAuthInfoFromCtx(c.Request().Context())
			if authInfo == nil {
				return c.JSON(http.StatusUnauthorized, domain.PWResponse{
					Success: false,
					Message: "Unauthorized",
				})
			}

			kbId, _ := GetKbID(c)

			if authInfo.IsToken {

				if authInfo.KBId != kbId {
					m.logger.Error("ValidateKBUserPerm ValidateTokenKBPerm kbId", "authInfo.KBId", authInfo.KBId, "kbId", kbId)
					return c.JSON(http.StatusForbidden, domain.PWResponse{
						Success: false,
						Message: "Unauthorized ValidateTokenKBPerm kbId",
					})
				}

				if authInfo.Permission != consts.UserKBPermissionFullControl && authInfo.Permission != perm {
					return c.JSON(http.StatusForbidden, domain.PWResponse{
						Success: false,
						Message: "Unauthorized ValidateTokenKBPerm",
					})
				}
			} else {
				// 正常用户请求
				valid, err := m.userAccessRepo.ValidateKBPerm(kbId, authInfo.UserId, perm)
				if err != nil || !valid {
					if err != nil {
						m.logger.Error("ValidateKBUserPerm ValidateKBPerm failed", log.Error(err))
					} else {
						m.logger.Info("ValidateKBUserPerm ValidateKBPerm failed", log.String("kb_id", kbId), log.String("user_id", authInfo.UserId))
					}
					return c.JSON(http.StatusForbidden, domain.PWResponse{
						Success: false,
						Message: "Unauthorized ValidateKBPerm",
					})
				}
			}

			return next(c)
		}
	}
}

func (m *JWTMiddleware) ValidateKBUserPermAny(perms ...consts.UserKBPermission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authInfo := domain.GetAuthInfoFromCtx(c.Request().Context())
			if authInfo == nil {
				return c.JSON(http.StatusUnauthorized, domain.PWResponse{
					Success: false,
					Message: "Unauthorized",
				})
			}

			kbId, _ := GetKbID(c)

			if authInfo.IsToken {
				if authInfo.KBId != kbId {
					return c.JSON(http.StatusForbidden, domain.PWResponse{
						Success: false,
						Message: "Unauthorized ValidateTokenKBPerm kbId",
					})
				}
				matched := false
				for _, perm := range perms {
					if authInfo.Permission == consts.UserKBPermissionFullControl || authInfo.Permission == perm {
						matched = true
						break
					}
				}
				if !matched {
					return c.JSON(http.StatusForbidden, domain.PWResponse{
						Success: false,
						Message: "Unauthorized ValidateTokenKBPerm",
					})
				}
			} else {
				matched := false
				for _, perm := range perms {
					valid, err := m.userAccessRepo.ValidateKBPerm(kbId, authInfo.UserId, perm)
					if err == nil && valid {
						matched = true
						break
					}
				}
				if !matched {
					m.logger.Info("ValidateKBUserPermAny failed", log.String("kb_id", kbId), log.String("user_id", authInfo.UserId))
					return c.JSON(http.StatusForbidden, domain.PWResponse{
						Success: false,
						Message: "Unauthorized ValidateKBPerm",
					})
				}
			}

			return next(c)
		}
	}
}

func (m *JWTMiddleware) ValidateLicenseEdition(needEditions ...consts.LicenseEdition) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			edition, ok := c.Get("edition").(consts.LicenseEdition)
			if !ok {
				return c.JSON(http.StatusForbidden, domain.PWResponse{
					Success: false,
					Message: "Unauthorized ValidateLicenseEdition",
				})
			}

			if !slices.Contains(needEditions, edition) {
				return c.JSON(http.StatusForbidden, domain.PWResponse{
					Success: false,
					Message: "Unauthorized ValidateLicenseEdition",
				})
			}

			return next(c)
		}
	}
}

func (m *JWTMiddleware) MustGetUserID(c echo.Context) (string, bool) {
	user, ok := c.Get("user").(*jwt.Token)
	if !ok || user == nil {
		return "", false
	}
	claims, ok := user.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}
	id, ok := claims["id"].(string)
	return id, ok
}

func GetKbID(c echo.Context) (string, error) {
	switch c.Request().Method {
	case http.MethodGet, http.MethodDelete:
		var kbId string
		if strings.Contains(c.Request().URL.Path, "knowledge_base") {
			kbId = c.QueryParam("id")
			if kbId != "" {
				return kbId, nil
			}
		}

		kbId = c.QueryParam("kb_id")
		if kbId != "" {
			return kbId, nil
		}

		return "", nil

	case http.MethodPost, http.MethodPatch, http.MethodPut:

		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return "", err
		}

		c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var m map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &m); err == nil {
			if strings.Contains(c.Request().URL.Path, "knowledge_base") {
				if id, exists := m["id"].(string); exists && id != "" {
					return id, nil
				}
			}

			if id, exists := m["kb_id"].(string); exists && id != "" {
				return id, nil
			}
		}
		return "", nil
	default:
		return "", nil
	}
}
