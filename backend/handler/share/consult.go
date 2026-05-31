package share

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/usecase"
)

// ShareConsultHandler 「疑难咨询」前台 share 入口。
//
// 路由统一前缀 share/pro/v1/consult；与现有 SiteFeedback 不同的是：
//   - 不再读 X-KB-ID（业务上不区分 KB，按 auths.id 隔离用户视图）
//   - 必须从 session 拿到 auths.id 作为 user_id，否则视为未登录
type ShareConsultHandler struct {
	*handler.BaseHandler
	logger      *log.Logger
	usecase     *usecase.ConsultUsecase
	authUsecase *usecase.AuthUsecase
}

func NewShareConsultHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	uc *usecase.ConsultUsecase,
	authUsecase *usecase.AuthUsecase,
) *ShareConsultHandler {
	h := &ShareConsultHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.share.consult"),
		usecase:     uc,
		authUsecase: authUsecase,
	}
	g := e.Group("share/pro/v1/consult",
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Access-Control-Allow-Origin", "*")
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept")
				if c.Request().Method == http.MethodOptions {
					return c.NoContent(http.StatusOK)
				}
				return next(c)
			}
		},
	)
	g.POST("", h.Create)
	g.GET("/list", h.List)
	g.GET("/detail", h.Detail)
	g.POST("/reply", h.Reply)
	g.POST("/close", h.Close)
	return h
}

// resolveUserID 从 session 拿 auths.id（数字字符串）。无登录则返回空串，由调用方决定如何返回。
func (h *ShareConsultHandler) resolveUserID(c echo.Context) string {
	if uid := c.Get("user_id"); uid != nil {
		if v, ok := uid.(uint); ok && v != 0 {
			return strconv.FormatUint(uint64(v), 10)
		}
	}
	sess, err := session.Get(domain.SessionName, c)
	if err != nil {
		return ""
	}
	if uid, ok := sessionAuthUserIDString(sess); ok {
		return uid
	}
	return ""
}

// resolveSubmitterName session 里的 auth username（如有）—— 用于落库 sender_name 快照，
// admin 后台直接看名字不再回查 auths。session 无快照时按 user_id 回查 auths.user_info。
func (h *ShareConsultHandler) resolveSubmitterName(c echo.Context) string {
	sess, err := session.Get(domain.SessionName, c)
	if err == nil {
		v, ok := sess.Values["auth_user_name"]
		if ok {
			if s, ok := v.(string); ok {
				if name := strings.TrimSpace(s); name != "" {
					return name
				}
			}
		}
	}
	uid := h.resolveUserID(c)
	if uid == "" {
		return ""
	}
	id, err := strconv.ParseUint(uid, 10, 64)
	if err != nil || id == 0 {
		return ""
	}
	return h.authUsecase.GetDisplayNameByAuthID(c.Request().Context(), uint(id))
}

// Create POST /share/pro/v1/consult
func (h *ShareConsultHandler) Create(c echo.Context) error {
	uid := h.resolveUserID(c)
	if uid == "" {
		return h.NewResponseWithError(c, "请先登录后再提交咨询", nil)
	}
	req := &domain.CreateConsultInquiryReq{}
	if err := c.Bind(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	inquiry, err := h.usecase.Create(c.Request().Context(), usecase.CreateInquiryShareInput{
		UserID:        uid,
		SubmitterName: h.resolveSubmitterName(c),
		RemoteIP:      c.RealIP(),
		Content:       req.Content,
		Contact:       req.Contact,
		Attachments:   req.Attachments,
	})
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, map[string]any{"item": inquiry})
}

// List GET /share/pro/v1/consult/list?status=&page=&per_page=
func (h *ShareConsultHandler) List(c echo.Context) error {
	uid := h.resolveUserID(c)
	if uid == "" {
		return h.NewResponseWithError(c, "请先登录", nil)
	}
	status := c.QueryParam("status")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	items, total, err := h.usecase.ListByUser(c.Request().Context(), uid, status, page, perPage)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, map[string]any{
		"items": items,
		"total": total,
	})
}

// Detail GET /share/pro/v1/consult/detail?id=
func (h *ShareConsultHandler) Detail(c echo.Context) error {
	uid := h.resolveUserID(c)
	if uid == "" {
		return h.NewResponseWithError(c, "请先登录", nil)
	}
	id, err := strconv.ParseInt(c.QueryParam("id"), 10, 64)
	if err != nil || id <= 0 {
		return h.NewResponseWithError(c, "invalid id", err)
	}
	detail, err := h.usecase.DetailForUser(c.Request().Context(), uid, id)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, detail)
}

// Reply POST /share/pro/v1/consult/reply
func (h *ShareConsultHandler) Reply(c echo.Context) error {
	uid := h.resolveUserID(c)
	if uid == "" {
		return h.NewResponseWithError(c, "请先登录", nil)
	}
	req := &domain.CreateConsultMessageReq{}
	if err := c.Bind(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	msg, err := h.usecase.AppendUserMessage(c.Request().Context(), usecase.AppendShareInput{
		UserID:        uid,
		SubmitterName: h.resolveSubmitterName(c),
		InquiryID:     req.InquiryID,
		Content:       req.Content,
		Attachments:   req.Attachments,
	})
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, map[string]any{"item": msg})
}

// Close POST /share/pro/v1/consult/close — 用户主动关闭自己的咨询单。
func (h *ShareConsultHandler) Close(c echo.Context) error {
	uid := h.resolveUserID(c)
	if uid == "" {
		return h.NewResponseWithError(c, "请先登录", nil)
	}
	req := &domain.UpdateConsultStatusReq{}
	if err := c.Bind(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if req.InquiryID <= 0 {
		return h.NewResponseWithError(c, "inquiry_id is required", nil)
	}
	if err := h.usecase.CloseForUser(c.Request().Context(), uid, req.InquiryID); err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, nil)
}
