package v1

import (
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type SystemInfoHandler struct {
	*handler.BaseHandler
	usecase *usecase.SystemInfoUsecase
	logger  *log.Logger
}

func NewSystemInfoHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	systemInfoUsecase *usecase.SystemInfoUsecase,
) *SystemInfoHandler {
	h := &SystemInfoHandler{
		BaseHandler: baseHandler,
		usecase:     systemInfoUsecase,
		logger:      logger.WithModule("handler.v1.system_info"),
	}
	echo.GET("/api/v1/system/info", h.PublicInfo)
	group := echo.Group("/api/v1/system", auth.Authorize)
	group.GET("/alerts", h.Alerts)
	group.GET("/classification", h.GetClassification, auth.ValidateUserRole(consts.UserRoleAdmin))
	group.PUT("/classification", h.SaveClassification, auth.ValidateUserRole(consts.UserRoleAdmin))
	return h
}

func (h *SystemInfoHandler) PublicInfo(c echo.Context) error {
	resp, err := h.usecase.PublicInfo(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get system info failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *SystemInfoHandler) Alerts(c echo.Context) error {
	resp, err := h.usecase.Alerts(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get system alerts failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *SystemInfoHandler) GetClassification(c echo.Context) error {
	resp, err := h.usecase.GetClassification(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get classification failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *SystemInfoHandler) SaveClassification(c echo.Context) error {
	var req domain.ClassificationSetting
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	resp, err := h.usecase.SaveClassification(c.Request().Context(), req)
	if err != nil {
		return h.NewResponseWithError(c, "save classification failed", err)
	}
	return h.NewResponseWithData(c, resp)
}
