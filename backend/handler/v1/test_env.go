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

type TestEnvHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.TestEnvUsecase
}

func NewTestEnvHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	testEnvUsecase *usecase.TestEnvUsecase,
) *TestEnvHandler {
	h := &TestEnvHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.test_env"),
		usecase:     testEnvUsecase,
	}
	group := echo.Group(
		"/api/v1/system/test-env",
		auth.Authorize,
		auth.ValidateUserRole(consts.UserRoleAdmin),
	)
	group.GET("", h.Get)
	group.PUT("", h.Save)
	return h
}

func (h *TestEnvHandler) Get(c echo.Context) error {
	result, err := h.usecase.Get(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get test env setting failed", err)
	}
	return h.NewResponseWithData(c, result)
}

func (h *TestEnvHandler) Save(c echo.Context) error {
	var req domain.TestEnvSetting
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	result, err := h.usecase.Save(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, "save test env setting failed", err)
	}
	return h.NewResponseWithData(c, result)
}
