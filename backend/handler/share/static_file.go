package share

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/s3"
	"github.com/chaitin/panda-wiki/usecase"
	"github.com/chaitin/panda-wiki/utils"
)

// 这些类型浏览器不会当作页面执行脚本，可以直接内联展示；其余一律按附件下载并加 CSP sandbox。
var inlineStaticTypes = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/gif":                true,
	"image/webp":               true,
	"image/bmp":                true,
	"image/avif":               true,
	"image/x-icon":             true,
	"image/vnd.microsoft.icon": true,
	"video/mp4":                true,
	"video/webm":               true,
	"video/ogg":                true,
	"video/quicktime":          true,
	"audio/mpeg":               true,
	"audio/mp4":                true,
	"audio/ogg":                true,
	"audio/wav":                true,
	"audio/x-wav":              true,
	"audio/webm":               true,
	"application/pdf":          true,
	"text/plain":               true,
}

type StaticFileHandler struct {
	logger    *log.Logger
	s3        *s3.MinioClient
	kbUsecase *usecase.KnowledgeBaseUsecase
	config    *config.Config
}

func NewStaticFileHandler(e *echo.Echo, logger *log.Logger, s3 *s3.MinioClient, kbUsecase *usecase.KnowledgeBaseUsecase, config *config.Config) *StaticFileHandler {
	h := &StaticFileHandler{
		logger:    logger.WithModule("handler.share.static_file"),
		s3:        s3,
		kbUsecase: kbUsecase,
		config:    config,
	}
	e.GET("/static-file/*", h.Serve)
	e.HEAD("/static-file/*", h.Serve)
	// 只在容器网络内使用（Caddy 与后台 nginx 都不转发这个前缀），签名见 utils.SignedStaticFileURL
	e.GET("/static-file-signed/:exp/:sig/*", h.ServeSigned)
	e.HEAD("/static-file-signed/:exp/:sig/*", h.ServeSigned)
	return h
}

func (h *StaticFileHandler) Serve(c echo.Context) error {
	key := staticObjectKey(c.Param("*"))
	if key == "" {
		return echo.ErrNotFound
	}
	if !h.allowed(c, key) {
		return echo.ErrForbidden
	}
	return h.serveObject(c, key)
}

func (h *StaticFileHandler) ServeSigned(c echo.Context) error {
	key := staticObjectKey(c.Param("*"))
	if key == "" {
		return echo.ErrNotFound
	}
	if !utils.VerifyStaticFileSignature(h.config.Auth.JWT.Secret, key, c.Param("exp"), c.Param("sig"), time.Now()) {
		return echo.ErrForbidden
	}
	return h.serveObject(c, key)
}

func (h *StaticFileHandler) serveObject(c echo.Context, key string) error {
	ctx := c.Request().Context()
	obj, err := h.s3.GetObject(ctx, domain.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer obj.Close()
	stat, err := obj.Stat()
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return echo.ErrNotFound
		}
		return fmt.Errorf("stat object: %w", err)
	}

	header := c.Response().Header()
	contentType := staticContentType(stat.ContentType, key)
	header.Set("Content-Type", contentType)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "private, max-age=3600")
	if stat.ETag != "" {
		header.Set("ETag", `"`+stat.ETag+`"`)
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if !inlineStaticTypes[mediaType] {
		name := originalName(stat.UserMetadata)
		if name == "" {
			name = path.Base(key)
		}
		header.Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(name))
		header.Set("Content-Security-Policy", "sandbox; default-src 'none'")
	}

	http.ServeContent(c.Response(), c.Request(), "", stat.LastModified, obj)
	return nil
}

// allowed 管理后台登录用户可读全部文件；对象 key 首段是知识库 ID 时，公开知识库直接放行，
// 开启认证的知识库要求该知识库的 share 会话。
func (h *StaticFileHandler) allowed(c echo.Context, key string) bool {
	if cookie, err := c.Cookie(utils.StaticAccessCookie); err == nil {
		if _, ok := utils.VerifyStaticAccessToken(h.config.Auth.JWT.Secret, cookie.Value, time.Now()); ok {
			return true
		}
	}

	kbID, _, _ := strings.Cut(key, "/")
	kb, err := h.kbUsecase.GetKnowledgeBase(c.Request().Context(), kbID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 站点图标等品牌素材和早期编辑器上传的文件以随机 UUID 为前缀、不归属知识库，
		// 登录页也要展示，只依赖不可猜测的对象 key
		return true
	}
	if err != nil {
		h.logger.Error("get knowledge base for static file failed", log.String("kb_id", kbID), log.Error(err))
		return false
	}
	settings := kb.AccessSettings
	if settings.IsForbidden {
		return false
	}
	if !settings.EnterpriseAuth.Enabled && !settings.SimpleAuth.Enabled {
		return true
	}
	sess, err := session.Get(domain.SessionName, c)
	if err != nil {
		return false
	}
	sessKBID, ok := sessionKbIDString(sess)
	return ok && sessKBID == kb.ID
}

func staticObjectKey(raw string) string {
	if unescaped, err := url.PathUnescape(raw); err == nil {
		raw = unescaped
	}
	key := strings.TrimPrefix(path.Clean("/"+raw), "/")
	if key == "." {
		return ""
	}
	return key
}

func staticContentType(stored, key string) string {
	if mediaType, _, err := mime.ParseMediaType(stored); err == nil && mediaType != "application/octet-stream" {
		if mediaType == "text/plain" {
			return "text/plain; charset=utf-8"
		}
		return stored
	}
	if byExt := mime.TypeByExtension(strings.ToLower(path.Ext(key))); byExt != "" {
		return byExt
	}
	return "application/octet-stream"
}

func originalName(meta map[string]string) string {
	for k, v := range meta {
		if strings.EqualFold(k, "originalname") {
			return v
		}
	}
	return ""
}
