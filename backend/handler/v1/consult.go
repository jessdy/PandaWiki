package v1

import (
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

// ConsultHandler 「疑难咨询」后台管理 API。
//
// 路由前缀 /api/v1/consult；全部要求 admin 角色。
// 与前台不同：admin 视图不区分 user_id，全平台咨询都能看到 / 回复 / 改状态 / 删除。
type ConsultHandler struct {
	*handler.BaseHandler
	logger      *log.Logger
	auth        middleware.AuthMiddleware
	usecase     *usecase.ConsultUsecase
	userUsecase *usecase.UserUsecase
}

func NewConsultHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	auth middleware.AuthMiddleware,
	uc *usecase.ConsultUsecase,
	userUsecase *usecase.UserUsecase,
) *ConsultHandler {
	h := &ConsultHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.consult"),
		auth:        auth,
		usecase:     uc,
		userUsecase: userUsecase,
	}

	// 仅 admin（后台用户）可访问；不需要 KB 维度权限校验，因为本功能不区分 KB。
	g := e.Group("/api/v1/consult",
		h.auth.Authorize,
		h.auth.ValidateUserRole(consts.UserRoleAdmin),
	)
	g.GET("/list", h.List)
	g.GET("/open_count", h.OpenCount)
	g.GET("/detail", h.Detail)
	g.POST("/reply", h.Reply)
	g.POST("/status", h.SetStatus)
	g.DELETE("", h.Delete)
	return h
}

type consultListData struct {
	Data  []domain.ConsultInquiryListItem `json:"data"`
	Total int64                           `json:"total"`
}

// List GET /api/v1/consult/list?status=&keyword=&page=&per_page=
func (h *ConsultHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	status := c.QueryParam("status")
	keyword := c.QueryParam("keyword")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	rows, total, err := h.usecase.AdminList(ctx, status, keyword, page, perPage)
	if err != nil {
		return h.NewResponseWithError(c, "list failed", err)
	}
	return h.NewResponseWithData(c, consultListData{Data: rows, Total: total})
}

// OpenCount GET /api/v1/consult/open_count — 待处理 + 处理中数量，供侧边栏角标。
func (h *ConsultHandler) OpenCount(c echo.Context) error {
	count, err := h.usecase.AdminOpenCount(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "count failed", err)
	}
	return h.NewResponseWithData(c, map[string]any{"count": count})
}

// Detail GET /api/v1/consult/detail?id=
func (h *ConsultHandler) Detail(c echo.Context) error {
	id, err := strconv.ParseInt(c.QueryParam("id"), 10, 64)
	if err != nil || id <= 0 {
		return h.NewResponseWithError(c, "invalid id", err)
	}
	detail, err := h.usecase.AdminDetail(c.Request().Context(), id)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, detail)
}

// adminReplyReq 在 domain.CreateConsultMessageReq 基础上多一个「是否同时把单据置为已回复」选项。
type adminReplyReq struct {
	domain.CreateConsultMessageReq
	MarkReplied bool `json:"mark_replied"`
}

// Reply POST /api/v1/consult/reply
func (h *ConsultHandler) Reply(c echo.Context) error {
	req := &adminReplyReq{}
	if err := c.Bind(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if req.InquiryID == 0 {
		return h.NewResponseWithError(c, "inquiry_id is required", nil)
	}

	// 拿到当前 admin 的展示名，落库到 sender_name；frontend 可直接渲染不再回查 users 表。
	authInfo := domain.GetAuthInfoFromCtx(c.Request().Context())
	adminID := ""
	adminName := ""
	if authInfo != nil {
		adminID = authInfo.UserId
		if u, err := h.userUsecase.GetUser(c.Request().Context(), authInfo.UserId); err == nil {
			adminName = strings.TrimSpace(u.Account)
		}
	}

	msg, err := h.usecase.AdminReply(c.Request().Context(), usecase.AdminReplyInput{
		AdminID:     adminID,
		AdminName:   adminName,
		InquiryID:   req.InquiryID,
		Content:     req.Content,
		Attachments: req.Attachments,
		MarkReplied: req.MarkReplied,
	})
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, map[string]any{"item": msg})
}

// SetStatus POST /api/v1/consult/status
func (h *ConsultHandler) SetStatus(c echo.Context) error {
	req := &domain.UpdateConsultStatusReq{}
	if err := c.Bind(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if req.InquiryID == 0 || req.Status == "" {
		return h.NewResponseWithError(c, "inquiry_id & status are required", nil)
	}
	if err := h.usecase.AdminSetStatus(c.Request().Context(), req.InquiryID, req.Status); err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, nil)
}

// Delete DELETE /api/v1/consult?ids=1,2,3
func (h *ConsultHandler) Delete(c echo.Context) error {
	raw := c.QueryParams()["ids"]
	if len(raw) == 0 {
		return h.NewResponseWithError(c, "ids is required", nil)
	}
	var ids []int64
	for _, s := range raw {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return h.NewResponseWithError(c, "invalid id", err)
			}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return h.NewResponseWithError(c, "ids is required", nil)
	}
	if err := h.usecase.AdminDelete(c.Request().Context(), ids); err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, nil)
}
