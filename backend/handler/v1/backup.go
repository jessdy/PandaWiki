package v1

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type BackupHandler struct {
	*handler.BaseHandler
	usecase *usecase.BackupUsecase
	logger  *log.Logger
}

func NewBackupHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	backupUsecase *usecase.BackupUsecase,
) *BackupHandler {
	h := &BackupHandler{
		BaseHandler: baseHandler,
		usecase:     backupUsecase,
		logger:      logger.WithModule("handler.v1.backup"),
	}
	group := echo.Group(
		"/api/v1/backup",
		auth.Authorize,
		auth.ValidateUserRole(consts.UserRoleAdmin),
	)
	group.GET("", h.List)
	group.GET("/status", h.Status)
	group.GET("/storage", h.Storage)
	group.GET("/setting", h.GetSetting)
	group.PUT("/setting", h.SaveSetting)
	group.POST("", h.Create)
	group.POST("/restore", h.Restore)
	group.GET("/:name/download", h.Download)
	group.DELETE("/:name", h.Delete)
	return h
}

func (h *BackupHandler) List(c echo.Context) error {
	list, err := h.usecase.List()
	if err != nil {
		return h.NewResponseWithError(c, "list backups failed", err)
	}
	return h.NewResponseWithData(c, map[string]any{"list": list})
}

func (h *BackupHandler) Status(c echo.Context) error {
	return h.NewResponseWithData(c, h.usecase.Status())
}

func (h *BackupHandler) Storage(c echo.Context) error {
	status, err := h.usecase.StorageStatus(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get backup storage failed", err)
	}
	return h.NewResponseWithData(c, status)
}

func (h *BackupHandler) GetSetting(c echo.Context) error {
	setting, err := h.usecase.Setting(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get backup setting failed", err)
	}
	return h.NewResponseWithData(c, setting)
}

func (h *BackupHandler) SaveSetting(c echo.Context) error {
	var req domain.BackupSetting
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	resp, err := h.usecase.SaveSetting(c.Request().Context(), req)
	if err != nil {
		return h.NewResponseWithError(c, "save backup setting failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *BackupHandler) Create(c echo.Context) error {
	actor := ""
	if user, ok := c.Get(domain.AuthUserKey).(*domain.User); ok && user != nil {
		actor = user.Account
	}
	info, err := h.usecase.Create(c.Request().Context(), domain.BackupTriggerManual, actor)
	if err != nil {
		return h.NewResponseWithError(c, "create backup failed", err)
	}
	return h.NewResponseWithData(c, info)
}

func (h *BackupHandler) Restore(c echo.Context) error {
	var req domain.BackupRestoreReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if req.Confirm != "RESTORE" {
		return h.NewResponseWithError(c, "请输入 RESTORE 确认恢复操作", nil)
	}
	actor := ""
	if user, ok := c.Get(domain.AuthUserKey).(*domain.User); ok && user != nil {
		actor = user.Account
	}
	if err := h.usecase.Restore(c.Request().Context(), req.Name, actor); err != nil {
		return h.NewResponseWithError(c, "restore backup failed", err)
	}
	return h.NewResponseWithData(c, nil)
}

func (h *BackupHandler) Download(c echo.Context) error {
	name := filepath.Base(c.Param("name"))
	f, err := h.usecase.Open(name)
	if err != nil {
		return h.NewResponseWithError(c, "open backup failed", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return h.NewResponseWithError(c, "stat backup failed", err)
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, name))
	c.Response().Header().Set(echo.HeaderContentLength, fmt.Sprintf("%d", st.Size()))
	_, err = io.Copy(c.Response().Writer, f)
	if err != nil {
		return err
	}
	return nil
}

func (h *BackupHandler) Delete(c echo.Context) error {
	name := filepath.Base(c.Param("name"))
	if err := h.usecase.Delete(name); err != nil {
		return h.NewResponseWithError(c, "delete backup failed", err)
	}
	return h.NewResponseWithData(c, nil)
}
