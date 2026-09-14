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

type StandardTermHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.StandardTermUsecase
}

func NewStandardTermHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	standardTermUsecase *usecase.StandardTermUsecase,
) *StandardTermHandler {
	h := &StandardTermHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.standard_term"),
		usecase:     standardTermUsecase,
	}
	group := echo.Group(
		"/api/v1/system/standard-terms",
		auth.Authorize,
		auth.ValidateUserRole(consts.UserRoleAdmin),
	)
	group.GET("", h.Get)
	group.PUT("", h.Replace)
	return h
}

// Get
//
//	@Summary		get global standard terms
//	@Description	get globally shared standard terms and aliases
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	domain.Response{data=domain.StandardTermsSetting}
//	@Router			/api/v1/system/standard-terms [get]
func (h *StandardTermHandler) Get(c echo.Context) error {
	result, err := h.usecase.Get(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get standard terms failed", err)
	}
	return h.NewResponseWithData(c, result)
}

// Replace
//
//	@Summary		replace global standard terms
//	@Description	replace globally shared standard terms and aliases
//	@Tags			system
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.ReplaceStandardTermsReq	true	"standard terms"
//	@Success		200		{object}	domain.Response{data=domain.StandardTermsSetting}
//	@Router			/api/v1/system/standard-terms [put]
func (h *StandardTermHandler) Replace(c echo.Context) error {
	var req domain.ReplaceStandardTermsReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	result, err := h.usecase.Replace(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, result)
}
