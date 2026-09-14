package share

import (
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/usecase"
)

type ShareConversationHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.ConversationUsecase
}

func NewShareConversationHandler(
	baseHandler *handler.BaseHandler,
	echo *echo.Echo,
	usecase *usecase.ConversationUsecase,
	logger *log.Logger,
) *ShareConversationHandler {
	h := &ShareConversationHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.share.conversation"),
		usecase:     usecase,
	}

	group := echo.Group("share/v1/conversation",
		h.ShareAuthMiddleware.Authorize,
	)
	group.GET("/detail", h.GetConversationDetail)
	group.GET("/history", h.GetConversationHistory)
	group.GET("/history/detail", h.GetConversationHistoryDetail)

	return h
}

// GetConversationDetail
//
//	@Summary		GetConversationDetail
//	@Description	GetConversationDetail
//	@Tags			share_conversation
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string	true	"kb id"
//	@Param			id		query		string	true	"conversation id"
//	@Success		200		{object}	domain.PWResponse{data=domain.ShareConversationDetailResp}
//	@Router			/share/v1/conversation/detail [get]
func (h *ShareConversationHandler) GetConversationDetail(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	id := c.QueryParam("id")
	if id == "" {
		return h.NewResponseWithError(c, "id is required", nil)
	}

	node, err := h.usecase.GetShareConversationDetail(c.Request().Context(), kbID, id)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get node detail", err)
	}
	return h.NewResponseWithData(c, node)
}

// GetConversationHistory
//
//	@Summary	Get current user's training conversation history
//	@Tags		share_conversation
//	@Produce	json
//	@Param		X-KB-ID		header		string	true	"kb id"
//	@Param		page		query		int		true	"page"
//	@Param		per_page	query		int		true	"page size"
//	@Success	200			{object}	domain.PWResponse{data=domain.ShareConversationHistoryResp}
//	@Router		/share/v1/conversation/history [get]
func (h *ShareConversationHandler) GetConversationHistory(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	userID, ok := shareAuthUserID(c, kbID)
	if !ok {
		return h.NewResponseWithData(c, &domain.ShareConversationHistoryResp{
			Available: false,
			Data:      []*domain.ShareConversationHistoryItem{},
		})
	}

	req := domain.ShareConversationHistoryReq{}
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	result, err := h.usecase.GetShareConversationHistory(c.Request().Context(), kbID, userID, &req)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get conversation history", err)
	}
	return h.NewResponseWithData(c, result)
}

// GetConversationHistoryDetail
//
//	@Summary	Get an owned training conversation
//	@Tags		share_conversation
//	@Produce	json
//	@Param		X-KB-ID	header		string	true	"kb id"
//	@Param		id		query		string	true	"conversation id"
//	@Success	200		{object}	domain.PWResponse{data=domain.ShareConversationDetailResp}
//	@Router		/share/v1/conversation/history/detail [get]
func (h *ShareConversationHandler) GetConversationHistoryDetail(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	userID, ok := shareAuthUserID(c, kbID)
	if !ok {
		return h.NewResponseWithError(c, "login required", nil)
	}
	id := c.QueryParam("id")
	if id == "" {
		return h.NewResponseWithError(c, "id is required", nil)
	}
	result, err := h.usecase.GetShareConversationHistoryDetail(c.Request().Context(), kbID, id, userID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get conversation history detail", err)
	}
	return h.NewResponseWithData(c, result)
}
