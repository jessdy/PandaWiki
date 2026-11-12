package v1

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type ApprovalHandler struct {
	*handler.BaseHandler
	usecase *usecase.ApprovalUsecase
	logger  *log.Logger
}

func NewApprovalHandler(
	echo *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	approvalUsecase *usecase.ApprovalUsecase,
) *ApprovalHandler {
	h := &ApprovalHandler{
		BaseHandler: baseHandler,
		usecase:     approvalUsecase,
		logger:      logger.WithModule("handler.v1.approval"),
	}
	group := echo.Group("/api/v1/approval", auth.Authorize)
	group.GET("", h.List, auth.ValidateExactRole(consts.UserRoleSecurityAdmin, consts.UserRoleAdmin))
	group.GET("/pending_count", h.PendingCount, auth.ValidateExactRole(consts.UserRoleSecurityAdmin, consts.UserRoleAdmin))
	group.POST("/:id/approve", h.Approve, auth.ValidateExactRole(consts.UserRoleSecurityAdmin))
	group.POST("/:id/reject", h.Reject, auth.ValidateExactRole(consts.UserRoleSecurityAdmin))
	return h
}

func (h *ApprovalHandler) actor(c echo.Context) (usecase.ApprovalActor, error) {
	user, ok := c.Get(domain.AuthUserKey).(*domain.User)
	if !ok || user == nil {
		return usecase.ApprovalActor{}, echo.ErrUnauthorized
	}
	return usecase.ApprovalActor{ID: user.ID, Account: user.Account, Role: user.Role}, nil
}

func (h *ApprovalHandler) List(c echo.Context) error {
	actor, err := h.actor(c)
	if err != nil {
		return h.NewResponseWithError(c, "unauthorized", err)
	}
	var req domain.ApprovalListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	resp, err := h.usecase.List(c.Request().Context(), actor, req)
	if err != nil {
		return h.NewResponseWithError(c, "list approvals failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

func (h *ApprovalHandler) PendingCount(c echo.Context) error {
	actor, err := h.actor(c)
	if err != nil {
		return h.NewResponseWithError(c, "unauthorized", err)
	}
	n, err := h.usecase.CountPending(c.Request().Context(), actor)
	if err != nil {
		return h.NewResponseWithError(c, "count pending approvals failed", err)
	}
	return h.NewResponseWithData(c, map[string]int64{"count": n})
}

func (h *ApprovalHandler) Approve(c echo.Context) error {
	return h.review(c, true)
}

func (h *ApprovalHandler) Reject(c echo.Context) error {
	return h.review(c, false)
}

func (h *ApprovalHandler) review(c echo.Context, approve bool) error {
	actor, err := h.actor(c)
	if err != nil {
		return h.NewResponseWithError(c, "unauthorized", err)
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return h.NewResponseWithError(c, "invalid approval id", err)
	}
	var req domain.ApprovalReviewReq
	_ = c.Bind(&req)
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if approve {
		err = h.usecase.Approve(c.Request().Context(), actor, id, req.Comment)
	} else {
		err = h.usecase.Reject(c.Request().Context(), actor, id, req.Comment)
	}
	if err != nil {
		return h.NewResponseWithError(c, "review approval failed", err)
	}
	return h.NewResponseWithData(c, nil)
}
