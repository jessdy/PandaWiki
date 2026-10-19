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

type CloudAuthHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.CloudAuthUsecase
}

func NewCloudAuthHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	cloudAuthUsecase *usecase.CloudAuthUsecase,
) *CloudAuthHandler {
	h := &CloudAuthHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.cloud_auth"),
		usecase:     cloudAuthUsecase,
	}
	group := echo.Group(
		"/api/v1/system/cloud-auth",
		auth.Authorize,
		auth.ValidateUserRole(consts.UserRoleAdmin),
	)
	group.GET("", h.Get)
	group.PUT("", h.Save)
	return h
}

func (h *CloudAuthHandler) Get(c echo.Context) error {
	result, err := h.usecase.Get(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "读取平台地址失败", err)
	}
	return h.NewResponseWithData(c, result)
}

func (h *CloudAuthHandler) Save(c echo.Context) error {
	var req domain.CloudAuthSetting
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "平台地址无效", err)
	}
	result, err := h.usecase.Save(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, result)
}
