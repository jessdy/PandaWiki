package share

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/usecase"
)

// ShareMethodRuleHandler 为前台「工作模式属性面板」提供实时联动接口。
// 不走 LLM、不写 conversation；调用方拖动 Select 后用本接口刷新方法卡片即可。
type ShareMethodRuleHandler struct {
	*handler.BaseHandler
	logger   *log.Logger
	kbUC     *usecase.KnowledgeBaseUsecase
	llmUC    *usecase.LLMUsecase
	nodeUC   *usecase.NodeUsecase
	categoryHelper *categoryHelper
}

// categoryHelper 是 share 侧用来从 category name 反查 specs 的小工具。
// 复用 LLMUsecase 已有的 GetWorkModeCategoryPrompts，避免直接依赖 categoryPromptRepo。
type categoryHelper struct {
	llmUC *usecase.LLMUsecase
}

func (h *categoryHelper) findCategoryByName(items []domain.CategoryPromptItem, name string) *domain.CategoryPromptItem {
	target := strings.TrimSpace(name)
	if target == "" {
		return nil
	}
	for i := range items {
		if strings.TrimSpace(items[i].Name) == target {
			return &items[i]
		}
	}
	return nil
}

func NewShareMethodRuleHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	kbUC *usecase.KnowledgeBaseUsecase,
	llmUC *usecase.LLMUsecase,
	nodeUC *usecase.NodeUsecase,
) *ShareMethodRuleHandler {
	h := &ShareMethodRuleHandler{
		BaseHandler:    baseHandler,
		logger:         logger.WithModule("handler.share.method_rule"),
		kbUC:           kbUC,
		llmUC:          llmUC,
		nodeUC:         nodeUC,
		categoryHelper: &categoryHelper{llmUC: llmUC},
	}

	group := e.Group("share/v1/method_rules",
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Access-Control-Allow-Origin", "*")
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, X-KB-ID")
				if c.Request().Method == "OPTIONS" {
					return c.NoContent(http.StatusOK)
				}
				return next(c)
			}
		})
	group.POST("/match", h.Match)
	return h
}

// MatchReq 前台属性面板调本接口时的入参。
type MatchReq struct {
	Category  string            `json:"category"`
	Collected map[string]string `json:"collected"`
}

// MatchMethodView 命中规则的前端视图（含文档名）。
type MatchMethodView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	NodeID      string `json:"node_id"`
	NodeName    string `json:"node_name,omitempty"`
}

// MatchAttrSpec 把品类的 specs 也回传给前台，让面板有「枚举」可渲染。
type MatchAttrSpec struct {
	Name   string   `json:"name"`
	Values []string `json:"values,omitempty"`
}

// MatchResp 接口响应：当前命中的方法 + 该品类完整的 specs（前台首次拉取需要 specs 来渲染 Select）。
type MatchResp struct {
	Category string            `json:"category"`
	Specs    []MatchAttrSpec   `json:"specs"`
	Methods  []MatchMethodView `json:"methods"`
}

// Match 实时联动查表：给定 category + collected，返回命中方法列表 + 该品类的 specs。
//
//	@Summary		MatchMethodRules
//	@Description	根据品类 + 已收集属性查表，返回命中的开封方法（含文档名）。无 LLM 调用。
//	@Tags			share_method_rule
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string		true	"kb id"
//	@Param			body	body		MatchReq	true	"request"
//	@Success		200		{object}	domain.Response{data=MatchResp}
//	@Router			/share/v1/method_rules/match [post]
func (h *ShareMethodRuleHandler) Match(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	req := MatchReq{}
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		return h.NewResponseWithError(c, "category is required", nil)
	}

	ctx := c.Request().Context()

	// 取 specs（让前台首次拉取就能拿到，便于切品类时同步刷新枚举）
	cats, cErr := h.llmUC.GetWorkModeCategoryPrompts(ctx, kbID)
	if cErr != nil {
		return h.NewResponseWithError(c, "failed to load categories", cErr)
	}
	cat := h.categoryHelper.findCategoryByName(cats, category)
	specsOut := make([]MatchAttrSpec, 0)
	if cat != nil {
		for _, s := range cat.ResolveAttributeSpecs() {
			specsOut = append(specsOut, MatchAttrSpec{Name: s.Name, Values: s.Values})
		}
	}

	matched, mErr := h.kbUC.MatchMethodRules(ctx, kbID, category, req.Collected)
	if mErr != nil {
		return h.NewResponseWithError(c, "match failed", mErr)
	}

	// 补文档名
	ids := make([]string, 0, len(matched))
	for _, r := range matched {
		if r.NodeID != "" {
			ids = append(ids, r.NodeID)
		}
	}
	nameMap, _ := h.llmUC.GetNodeNamesByIDs(ctx, kbID, ids)

	views := make([]MatchMethodView, 0, len(matched))
	for _, r := range matched {
		views = append(views, MatchMethodView{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			NodeID:      r.NodeID,
			NodeName:    nameMap[r.NodeID],
		})
	}

	return h.NewResponseWithData(c, MatchResp{
		Category: category,
		Specs:    specsOut,
		Methods:  views,
	})
}
