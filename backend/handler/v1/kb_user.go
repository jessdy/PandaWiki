package v1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/kb/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/usecase"
)

func (h *KnowledgeBaseHandler) approvalActor(c echo.Context) (usecase.ApprovalActor, error) {
	if user, ok := c.Get(domain.AuthUserKey).(*domain.User); ok && user != nil {
		return usecase.ApprovalActor{ID: user.ID, Account: user.Account, Role: user.Role}, nil
	}
	return usecase.ApprovalActor{}, fmt.Errorf("unauthorized")
}

// KBUserList
//
//	@Summary		KBUserList
//	@Description	KBUserList
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			kb_id	query		string	true	"Knowledge Base ID"
//	@Success		200		{object}	domain.PWResponse{data=[]v1.KBUserListItemResp}
//	@Router			/api/v1/knowledge_base/user/list [get]
func (h *KnowledgeBaseHandler) KBUserList(c echo.Context) error {
	var req v1.KBUserListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "request params is invalid", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "validate request params failed", err)
	}

	resp, err := h.usecase.GetKBUserList(c.Request().Context(), req)
	if err != nil {
		return h.NewResponseWithError(c, "get kb user list failed", err)
	}

	return h.NewResponseWithData(c, resp)
}

// KBUserInvite
//
//	@Summary		KBUserInvite
//	@Description	Invite user to knowledge base
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			param	body		v1.KBUserInviteReq	true	"Invite User Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/knowledge_base/user/invite [post]
func (h *KnowledgeBaseHandler) KBUserInvite(c echo.Context) error {
	var req v1.KBUserInviteReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}

	if !domain.GetBaseEditionLimitation(c.Request().Context()).AllowAdminPerm {
		for _, p := range req.Perms {
			if p != string(consts.UserKBPermissionFullControl) {
				return h.NewResponseWithError(c, "当前版本不支持管理员分权控制", nil)
			}
		}
	}

	actor, err := h.approvalActor(c)
	if err != nil {
		return h.NewResponseWithError(c, "unauthorized", err)
	}
	resp, err := h.approvalUsecase.Submit(c.Request().Context(), actor, domain.ApprovalKBUserInvite,
		fmt.Sprintf("邀请用户 %s 加入知识库 %s", req.UserId, req.KBId),
		req.UserId, "", req.KBId, req)
	if err != nil {
		return h.NewResponseWithError(c, "submit approval failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

// KBUserUpdate
//
//	@Summary		KBUserUpdate
//	@Description	Update user permission in knowledge base
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			param	body		v1.KBUserUpdateReq	true	"Update User Permission Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/knowledge_base/user/update [patch]
func (h *KnowledgeBaseHandler) KBUserUpdate(c echo.Context) error {
	var req v1.KBUserUpdateReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}

	if !domain.GetBaseEditionLimitation(c.Request().Context()).AllowAdminPerm {
		for _, p := range req.Perms {
			if p != string(consts.UserKBPermissionFullControl) {
				return h.NewResponseWithError(c, "当前版本不支持管理员分权控制", nil)
			}
		}
	}

	actor, err := h.approvalActor(c)
	if err != nil {
		return h.NewResponseWithError(c, "unauthorized", err)
	}
	resp, err := h.approvalUsecase.Submit(c.Request().Context(), actor, domain.ApprovalKBUserUpdate,
		fmt.Sprintf("更新用户 %s 在知识库 %s 的权限", req.UserId, req.KBId),
		req.UserId, "", req.KBId, req)
	if err != nil {
		return h.NewResponseWithError(c, "submit approval failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

// KBUserDelete
//
//	@Summary		KBUserDelete
//	@Description	Remove user from knowledge base
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			param	query		v1.KBUserDeleteReq	true	"Remove User Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/knowledge_base/user/delete [delete]
func (h *KnowledgeBaseHandler) KBUserDelete(c echo.Context) error {
	var req v1.KBUserDeleteReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}

	actor, err := h.approvalActor(c)
	if err != nil {
		return h.NewResponseWithError(c, "unauthorized", err)
	}
	resp, err := h.approvalUsecase.Submit(c.Request().Context(), actor, domain.ApprovalKBUserDelete,
		fmt.Sprintf("移除用户 %s 出知识库 %s", req.UserId, req.KBId),
		req.UserId, "", req.KBId, req)
	if err != nil {
		return h.NewResponseWithError(c, "submit approval failed", err)
	}
	return h.NewResponseWithData(c, resp)
}
