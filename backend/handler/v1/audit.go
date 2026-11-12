package v1

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type AuditHandler struct {
	*handler.BaseHandler
	usecase *usecase.AuditUsecase
	logger  *log.Logger
}

func NewAuditHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	auditUsecase *usecase.AuditUsecase,
) *AuditHandler {
	h := &AuditHandler{
		BaseHandler: baseHandler,
		usecase:     auditUsecase,
		logger:      logger.WithModule("handler.v1.audit"),
	}
	group := echo.Group(
		"/api/v1/audit",
		auth.Authorize,
		auth.ValidateExactRole(consts.UserRoleAuditAdmin),
	)
	group.GET("/logs", h.List)
	group.GET("/verify", h.Verify)
	group.GET("/storage", h.Storage)
	group.PUT("/storage", h.SaveStorage)
	group.POST("/export", h.Export)
	group.POST("/export/verify", h.VerifyExport)
	return h
}

func (h *AuditHandler) List(c echo.Context) error {
	var req domain.AuditLogListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	resp, err := h.usecase.List(c.Request().Context(), req)
	if err != nil {
		return h.NewResponseWithError(c, "list audit logs failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *AuditHandler) Verify(c echo.Context) error {
	resp, err := h.usecase.Verify(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "verify audit chain failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *AuditHandler) Storage(c echo.Context) error {
	resp, err := h.usecase.StorageStatus(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get audit storage failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *AuditHandler) SaveStorage(c echo.Context) error {
	var req domain.AuditStorageSetting
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	resp, err := h.usecase.SaveStorageSetting(c.Request().Context(), req)
	if err != nil {
		return h.NewResponseWithError(c, "save audit storage setting failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *AuditHandler) Export(c echo.Context) error {
	var req domain.AuditExportReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	exporterAccount := ""
	if user, ok := c.Get(domain.AuthUserKey).(*domain.User); ok && user != nil {
		exporterAccount = user.Account
	}
	file, err := h.usecase.Export(c.Request().Context(), req, exporterAccount)
	if err != nil {
		return h.NewResponseWithError(c, "export audit logs failed", err)
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="audit-export-%s.json"`, file.CreatedAt.Format("20060102150405")))
	return c.JSON(http.StatusOK, domain.PWResponse{Success: true, Data: file})
}

type auditExportVerifyReq struct {
	Password string                   `json:"password" validate:"max=72"`
	File     *domain.AuditExportFile  `json:"file" validate:"required"`
}

func (h *AuditHandler) VerifyExport(c echo.Context) error {
	var req auditExportVerifyReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	resp, err := h.usecase.VerifyExport(req.File, req.Password)
	if err != nil {
		return h.NewResponseWithError(c, "verify export failed", err)
	}
	return h.NewResponseWithData(c, resp)
}
