package usecase

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "github.com/chaitin/panda-wiki/api/kb/v1"
	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/cache"
	"github.com/chaitin/panda-wiki/repo/mq"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/rag"
)

type KnowledgeBaseUsecase struct {
	repo                  *pg.KnowledgeBaseRepository
	nodeRepo              *pg.NodeRepository
	ragRepo               *mq.RAGRepository
	userRepo              *pg.UserRepository
	rag                   rag.RAGService
	kbCache               *cache.KBRepo
	logger                *log.Logger
	config                *config.Config
	categoryPromptRepo    *pg.CategoryPromptRepo
	imageDescTemplateRepo *pg.ImageDescriptionTemplateRepo
	methodRuleRepo        *pg.MethodRuleRepo
}

func NewKnowledgeBaseUsecase(repo *pg.KnowledgeBaseRepository, nodeRepo *pg.NodeRepository, ragRepo *mq.RAGRepository, userRepo *pg.UserRepository, rag rag.RAGService, kbCache *cache.KBRepo, logger *log.Logger, config *config.Config, categoryPromptRepo *pg.CategoryPromptRepo, imageDescTemplateRepo *pg.ImageDescriptionTemplateRepo, methodRuleRepo *pg.MethodRuleRepo) (*KnowledgeBaseUsecase, error) {
	u := &KnowledgeBaseUsecase{
		repo:                  repo,
		nodeRepo:              nodeRepo,
		ragRepo:               ragRepo,
		userRepo:              userRepo,
		rag:                   rag,
		logger:                logger.WithModule("usecase.knowledge_base"),
		config:                config,
		kbCache:               kbCache,
		categoryPromptRepo:    categoryPromptRepo,
		imageDescTemplateRepo: imageDescTemplateRepo,
		methodRuleRepo:        methodRuleRepo,
	}
	return u, nil
}

// isRAGConnectionError reports whether err is a connection failure to the RAG service
// (e.g. connection refused, timeout, unreachable). It unwraps the error chain so
// wrapped errors from the RAG SDK are still detected.
func isRAGConnectionError(err error) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if _, ok := e.(net.Error); ok {
			return true
		}
		if _, ok := e.(*net.OpError); ok {
			return true
		}
		if urlErr, ok := e.(*url.Error); ok && urlErr.Err != nil {
			if _, ok := urlErr.Err.(net.Error); ok {
				return true
			}
		}
		msg := e.Error()
		if strings.Contains(msg, "connection refused") ||
			strings.Contains(msg, "no such host") ||
			strings.Contains(msg, "timeout") ||
			strings.Contains(msg, "dial tcp") ||
			strings.Contains(msg, "connection reset") {
			return true
		}
	}
	return false
}

func (u *KnowledgeBaseUsecase) CreateKnowledgeBase(ctx context.Context, req *domain.CreateKnowledgeBaseReq) (string, error) {
	// create kb in vector store
	datasetID, err := u.rag.CreateKnowledgeBase(ctx)
	if err != nil {
		// 检查是否是连接错误（RAG 服务不可用）
		// 在开发环境中，如果 RAG 服务不可用，使用占位符 datasetID
		// 这样可以在没有 RAG 服务的情况下创建知识库（RAG 功能将不可用）
		isConnectionError := isRAGConnectionError(err)
		if isConnectionError {
			u.logger.Warn("RAG service unavailable, using placeholder dataset ID", "error", err)
			datasetID = "placeholder-dataset-id-" + uuid.New().String()
		} else {
			// 其他错误（如认证失败、API 错误等）仍然返回错误
			return "", fmt.Errorf("failed to create knowledge base in RAG service: %w", err)
		}
	}
	kbID := uuid.New().String()
	kb := &domain.KnowledgeBase{
		ID:        kbID,
		Name:      req.Name,
		DatasetID: datasetID,
		AccessSettings: domain.AccessSettings{
			Ports:      req.Ports,
			SSLPorts:   req.SSLPorts,
			PublicKey:  req.PublicKey,
			PrivateKey: req.PrivateKey,
			Hosts:      req.Hosts,
		},
	}

	if err := u.repo.CreateKnowledgeBase(ctx, req.MaxKB, kb); err != nil {
		return "", err
	}
	return kbID, nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBaseList(ctx context.Context) ([]*domain.KnowledgeBaseListItem, error) {
	knowledgeBases, err := u.repo.GetKnowledgeBaseList(ctx)
	if err != nil {
		return nil, err
	}
	return knowledgeBases, nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBaseListByUserId(ctx context.Context) ([]*domain.KnowledgeBaseListItem, error) {
	knowledgeBases, err := u.repo.GetKnowledgeBaseListByUserId(ctx)
	if err != nil {
		return nil, err
	}
	return knowledgeBases, nil
}

func (u *KnowledgeBaseUsecase) UpdateKnowledgeBase(ctx context.Context, req *domain.UpdateKnowledgeBaseReq) error {
	isChange, err := u.repo.UpdateKnowledgeBase(ctx, req)
	if err != nil {
		return err
	}

	if isChange {
		if err := u.kbCache.ClearSession(ctx); err != nil {
			return err
		}
	}

	if err := u.kbCache.DeleteKB(ctx, req.ID); err != nil {
		return err
	}

	return nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBase(ctx context.Context, kbID string) (*domain.KnowledgeBase, error) {
	kb, err := u.kbCache.GetKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb != nil {
		return kb, nil
	}
	kb, err = u.repo.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := u.kbCache.SetKB(ctx, kbID, kb); err != nil {
		return nil, err
	}
	return kb, nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBasePerms(ctx context.Context, kbID string) (consts.UserKBPermissions, error) {
	perms, err := u.repo.GetKBPermsByUserId(ctx, kbID)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (u *KnowledgeBaseUsecase) DeleteKnowledgeBase(ctx context.Context, kbID string) error {
	if err := u.repo.DeleteKnowledgeBase(ctx, kbID); err != nil {
		return err
	}
	// delete vector store
	if err := u.rag.DeleteKnowledgeBase(ctx, kbID); err != nil {
		return err
	}
	if err := u.kbCache.DeleteKB(ctx, kbID); err != nil {
		return err
	}
	return nil
}

func (u *KnowledgeBaseUsecase) CreateKBRelease(ctx context.Context, req *domain.CreateKBReleaseReq, userId string) (string, error) {
	if len(req.NodeIDs) > 0 {
		// create published nodes
		releaseIDs, err := u.nodeRepo.CreateNodeReleases(ctx, req.KBID, userId, req.NodeIDs)
		if err != nil {
			return "", fmt.Errorf("failed to create published nodes: %w", err)
		}
		if len(releaseIDs) > 0 {
			// async upsert vector content via mq
			nodeContentVectorRequests := make([]*domain.NodeReleaseVectorRequest, 0)
			for _, releaseID := range releaseIDs {
				nodeContentVectorRequests = append(nodeContentVectorRequests, &domain.NodeReleaseVectorRequest{
					KBID:          req.KBID,
					NodeReleaseID: releaseID,
					Action:        "upsert",
				})
			}
			if err := u.ragRepo.AsyncUpdateNodeReleaseVector(ctx, nodeContentVectorRequests); err != nil {
				return "", err
			}
		}
	}

	release := &domain.KBRelease{
		ID:        uuid.New().String(),
		KBID:      req.KBID,
		Message:   req.Message,
		Tag:       req.Tag,
		CreatedAt: time.Now(),
	}
	if err := u.repo.CreateKBRelease(ctx, release); err != nil {
		return "", fmt.Errorf("failed to create kb release: %w", err)
	}

	return release.ID, nil
}

func (u *KnowledgeBaseUsecase) GetKBReleaseList(ctx context.Context, req *domain.GetKBReleaseListReq) (*domain.GetKBReleaseListResp, error) {
	total, releases, err := u.repo.GetKBReleaseList(ctx, req.KBID)
	if err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(releases, uint64(total)), nil
}

func (u *KnowledgeBaseUsecase) GetKBUserList(ctx context.Context, req v1.KBUserListReq) ([]v1.KBUserListItemResp, error) {
	users, err := u.repo.GetKBUserlist(ctx, req.KBId)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (u *KnowledgeBaseUsecase) KBUserInvite(ctx context.Context, req v1.KBUserInviteReq) error {
	user, err := u.userRepo.GetUser(ctx, req.UserId)
	if err != nil {
		return err
	}
	if user.Role == consts.UserRoleAdmin {
		return fmt.Errorf("knowledge base can not invite to admin user")
	}

	if err := u.repo.CreateKBUser(ctx, &domain.KBUsers{
		KBId:      req.KBId,
		UserId:    req.UserId,
		Perms:     req.Perms,
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	return nil
}

func (u *KnowledgeBaseUsecase) UpdateUserKB(ctx context.Context, req v1.KBUserUpdateReq) error {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return fmt.Errorf("authInfo not found in context")
	}

	kbUser, err := u.repo.GetKBUser(ctx, req.KBId, req.UserId)
	if err != nil {
		return err
	}
	if authInfo.IsToken {
		if authInfo.KBId != req.KBId {
			return fmt.Errorf("invalid knowledge base token")
		}
		if authInfo.Permission != consts.UserKBPermissionFullControl {
			return fmt.Errorf("only admin can update user from knowledge base")
		}
	} else {
		user, err := u.userRepo.GetUser(ctx, authInfo.UserId)
		if err != nil {
			return err
		}
		if user.Role != consts.UserRoleAdmin && !kbUser.GetPerms().Contains(consts.UserKBPermissionFullControl) {
			return fmt.Errorf("only admin can update user from knowledge base")
		}
	}
	return u.repo.UpdateKBUserPerms(ctx, req.KBId, req.UserId, req.Perms)
}

func (u *KnowledgeBaseUsecase) KBUserDelete(ctx context.Context, req v1.KBUserDeleteReq) error {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return fmt.Errorf("authInfo not found in context")
	}

	kbUser, err := u.repo.GetKBUser(ctx, req.KBId, req.UserId)
	if err != nil {
		return err
	}
	if authInfo.IsToken {
		if authInfo.KBId != req.KBId {
			return fmt.Errorf("knowledge base can not delete user from knowledge base")
		}
		if authInfo.Permission != consts.UserKBPermissionFullControl {
			return fmt.Errorf("only admin can delete user from knowledge base")
		}
	} else {
		user, err := u.userRepo.GetUser(ctx, authInfo.UserId)
		if err != nil {
			return err
		}
		if user.Role != consts.UserRoleAdmin && !kbUser.GetPerms().Contains(consts.UserKBPermissionFullControl) {
			return fmt.Errorf("only admin can delete user from knowledge base")
		}
	}
	if err := u.repo.DeleteKBUser(ctx, req.KBId, req.UserId); err != nil {
		return err
	}

	return nil
}

func (u *KnowledgeBaseUsecase) GetCategoryPrompts(ctx context.Context, kbID string) ([]domain.CategoryPromptItem, error) {
	items, err := u.categoryPromptRepo.GetByKBID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []domain.CategoryPromptItem{}, nil
	}
	return items, nil
}

func (u *KnowledgeBaseUsecase) ReplaceCategoryPrompts(ctx context.Context, req *domain.CategoryPromptsReq) error {
	out := make([]domain.CategoryPromptItem, 0, len(req.Items))
	for _, it := range req.Items {
		name := strings.TrimSpace(it.Name)
		content := strings.TrimSpace(it.Content)
		if name == "" {
			continue
		}
		id := strings.TrimSpace(it.ID)
		if id == "" {
			id = uuid.New().String()
		}
		// 优先使用结构化 specs；兼容旧前端只传 Attributes 字符串。
		specs := normalizeAttributeSpecs(it.AttributeSpecs)
		if len(specs) == 0 {
			parsed := (&domain.CategoryPromptItem{Attributes: it.Attributes}).ResolveAttributeSpecs()
			specs = parsed
		}
		// 反向派生 Attributes 字符串，保持兼容旧前端/老代码路径。
		attrNames := make([]string, 0, len(specs))
		for _, s := range specs {
			attrNames = append(attrNames, s.Name)
		}
		out = append(out, domain.CategoryPromptItem{
			ID:             id,
			Name:           name,
			Content:        content,
			Attributes:     strings.Join(attrNames, ","),
			AttributeSpecs: specs,
		})
	}
	return u.categoryPromptRepo.ReplaceForKBID(ctx, req.KBID, out)
}

// normalizeAttributeSpecs 清洗 specs：去空、去重 values，保留入参顺序。
func normalizeAttributeSpecs(in []domain.CategoryAttributeSpec) []domain.CategoryAttributeSpec {
	if len(in) == 0 {
		return nil
	}
	out := make([]domain.CategoryAttributeSpec, 0, len(in))
	for _, s := range in {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			continue
		}
		seen := map[string]struct{}{}
		values := make([]string, 0, len(s.Values))
		for _, v := range s.Values {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			values = append(values, v)
		}
		out = append(out, domain.CategoryAttributeSpec{Name: name, Values: values})
	}
	return out
}

/* ---------------------------------------------------------------- */
/* 开封方法规则                                                      */
/* ---------------------------------------------------------------- */

// ListMethodRules 列规则；category 为空返回该 KB 全部。
func (u *KnowledgeBaseUsecase) ListMethodRules(ctx context.Context, kbID, category string) ([]domain.MethodRule, error) {
	if u.methodRuleRepo == nil {
		return []domain.MethodRule{}, nil
	}
	items, err := u.methodRuleRepo.GetByKBID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	cat := strings.TrimSpace(category)
	out := make([]domain.MethodRule, 0, len(items))
	for _, it := range items {
		if cat != "" && strings.TrimSpace(it.Category) != cat {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

// ReplaceMethodRules 整表替换：保留客户端给出的 ID（缺失则生成），清洗 Conditions 空值。
func (u *KnowledgeBaseUsecase) ReplaceMethodRules(ctx context.Context, req *domain.ReplaceMethodRulesReq) error {
	if u.methodRuleRepo == nil {
		return errors.New("method rule repo unavailable")
	}
	now := time.Now().Unix()
	out := make([]domain.MethodRule, 0, len(req.Items))
	for _, it := range req.Items {
		name := strings.TrimSpace(it.Name)
		category := strings.TrimSpace(it.Category)
		if name == "" || category == "" {
			continue
		}
		conds := make(map[string][]string, len(it.Conditions))
		for k, vs := range it.Conditions {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			seen := map[string]struct{}{}
			values := make([]string, 0, len(vs))
			for _, v := range vs {
				v = strings.TrimSpace(v)
				if v == "" {
					continue
				}
				if _, ok := seen[v]; ok {
					continue
				}
				seen[v] = struct{}{}
				values = append(values, v)
			}
			conds[key] = values
		}
		id := strings.TrimSpace(it.ID)
		createdAt := it.CreatedAt
		if id == "" {
			id = uuid.New().String()
			createdAt = now
		}
		out = append(out, domain.MethodRule{
			ID:          id,
			Category:    category,
			Name:        name,
			Description: strings.TrimSpace(it.Description),
			Conditions:  conds,
			NodeID:      strings.TrimSpace(it.NodeID),
			CreatedAt:   createdAt,
			UpdatedAt:   now,
		})
	}
	return u.methodRuleRepo.Replace(ctx, req.KBID, out)
}

// MatchMethodRules 给定 category + collected，返回命中的规则列表（按规则原顺序）。
// 用于前台「实时联动」预览：纯查表，不走 LLM。
func (u *KnowledgeBaseUsecase) MatchMethodRules(
	ctx context.Context,
	kbID, category string,
	collected map[string]string,
) ([]domain.MethodRule, error) {
	all, err := u.ListMethodRules(ctx, kbID, category)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MethodRule, 0, len(all))
	for i := range all {
		if all[i].MatchesCollected(collected) {
			out = append(out, all[i])
		}
	}
	return out, nil
}

// ListImageDescriptionTemplates 拉取该 KB 下指定品类的全部图片描述模版；
// category 为空时返回该 KB 的全部模版。
func (u *KnowledgeBaseUsecase) ListImageDescriptionTemplates(ctx context.Context, kbID, category string) ([]domain.ImageDescriptionTemplate, error) {
	if u.imageDescTemplateRepo == nil {
		return []domain.ImageDescriptionTemplate{}, nil
	}
	items, err := u.imageDescTemplateRepo.GetByKBID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	cat := strings.TrimSpace(category)
	out := make([]domain.ImageDescriptionTemplate, 0, len(items))
	for _, it := range items {
		if cat != "" && strings.TrimSpace(it.Category) != cat {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

// CreateImageDescriptionTemplate 追加一条模版。先拉全量、追加后再整表写回。
// 同 KB + 同品类下，模版名重复会直接返回错误，避免后续选择时无法区分。
func (u *KnowledgeBaseUsecase) CreateImageDescriptionTemplate(ctx context.Context, req *domain.CreateImageDescriptionTemplateReq) (*domain.ImageDescriptionTemplate, error) {
	if u.imageDescTemplateRepo == nil {
		return nil, errors.New("image description template repo unavailable")
	}
	category := strings.TrimSpace(req.Category)
	name := strings.TrimSpace(req.Name)
	if category == "" {
		return nil, errors.New("category is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}

	existing, err := u.imageDescTemplateRepo.GetByKBID(ctx, req.KBID)
	if err != nil {
		return nil, err
	}
	for _, it := range existing {
		if strings.TrimSpace(it.Category) == category && strings.TrimSpace(it.Name) == name {
			return nil, fmt.Errorf("品类「%s」下已存在同名模版「%s」", category, name)
		}
	}

	cleanAttrs := make(map[string]string, len(req.Attributes))
	for k, v := range req.Attributes {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" {
			continue
		}
		cleanAttrs[key] = val
	}

	now := time.Now().Unix()
	item := domain.ImageDescriptionTemplate{
		ID:         uuid.New().String(),
		Category:   category,
		Name:       name,
		Attributes: cleanAttrs,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	merged := append(existing, item)
	if err := u.imageDescTemplateRepo.Replace(ctx, req.KBID, merged); err != nil {
		return nil, err
	}
	return &item, nil
}
