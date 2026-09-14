package v1

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type ExpertProfileHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.ExpertProfileUsecase
}

func NewExpertProfileHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	expertProfileUsecase *usecase.ExpertProfileUsecase,
) *ExpertProfileHandler {
	h := &ExpertProfileHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.expert_profile"),
		usecase:     expertProfileUsecase,
	}
	group := echo.Group(
		"/api/v1/expert",
		auth.Authorize,
		auth.ValidateUserRole(consts.UserRoleAdmin),
	)
	group.GET("/list", h.List)
	group.PUT("/:user_id", h.Update)
	return h
}

func (h *ExpertProfileHandler) List(c echo.Context) error {
	items, err := h.usecase.List(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "获取专家信息失败", err)
	}
	return h.NewResponseWithData(c, map[string]any{"items": items})
}

func (h *ExpertProfileHandler) Update(c echo.Context) error {
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		return h.NewResponseWithError(c, "管理员 ID 不能为空", nil)
	}
	var req domain.UpdateExpertProfileReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "请求参数无效", err)
	}
	profile, err := h.usecase.Update(c.Request().Context(), userID, &req)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, profile)
}
