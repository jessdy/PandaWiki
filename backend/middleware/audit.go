package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
)

// AuditRecorder 由 usecase 实现，中间件只依赖接口以避免循环引用
type AuditRecorder interface {
	AppendFromEcho(c echo.Context, category, action, event, objectType, objectID, detail, result string, statusCode int, message string)
}

type AuditMiddleware struct {
	recorder AuditRecorder
}

func NewAuditMiddleware(recorder AuditRecorder) *AuditMiddleware {
	return &AuditMiddleware{recorder: recorder}
}

type auditWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	wrote      bool
}

func (w *auditWriter) WriteHeader(code int) {
	if !w.wrote {
		w.statusCode = code
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *auditWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.statusCode = http.StatusOK
		w.wrote = true
	}
	if w.body.Len() < 4096 {
		remain := 4096 - w.body.Len()
		if len(b) <= remain {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remain])
		}
	}
	return w.ResponseWriter.Write(b)
}

func (w *auditWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type auditRule struct {
	method   string
	prefix   string
	exact    string
	category string
	action   string
	event    string
	object   string
}

var auditRules = []auditRule{
	{method: http.MethodPost, exact: "/api/v1/user/login", category: "auth", action: "login", event: "用户登录", object: "user"},
	{method: http.MethodPost, exact: "/api/v1/user/logout", category: "auth", action: "logout", event: "用户登出", object: "user"},
	{method: http.MethodPost, exact: "/api/v1/user/create", category: "user", action: "create", event: "创建用户", object: "user"},
	{method: http.MethodDelete, exact: "/api/v1/user/delete", category: "user", action: "delete", event: "删除用户", object: "user"},
	{method: http.MethodPut, exact: "/api/v1/user/reset_password", category: "user", action: "reset_password", event: "重置密码", object: "user"},
	{method: http.MethodPost, exact: "/api/v1/user/guest/create", category: "user", action: "guest_create", event: "创建访客", object: "user"},
	{method: http.MethodDelete, prefix: "/api/v1/user/guest/", category: "user", action: "guest_delete", event: "删除访客", object: "user"},
	{method: http.MethodPut, prefix: "/api/v1/user/guest/", category: "user", action: "guest_update", event: "更新访客", object: "user"},
	{method: http.MethodPost, exact: "/api/v1/user/auth_group/create", category: "permission", action: "auth_group_create", event: "创建用户组", object: "auth_group"},
	{method: http.MethodPut, prefix: "/api/v1/user/auth_group/", category: "permission", action: "auth_group_update", event: "更新用户组", object: "auth_group"},
	{method: http.MethodDelete, prefix: "/api/v1/user/auth_group/", category: "permission", action: "auth_group_delete", event: "删除用户组", object: "auth_group"},
	{method: http.MethodPut, exact: "/api/v1/user/groups", category: "permission", action: "user_groups_update", event: "更新用户所属组", object: "user"},
	{method: http.MethodPost, exact: "/api/v1/knowledge_base/user/invite", category: "permission", action: "kb_user_invite", event: "邀请用户加入知识库", object: "kb_user"},
	{method: http.MethodPatch, exact: "/api/v1/knowledge_base/user/update", category: "permission", action: "kb_user_update", event: "更新知识库用户权限", object: "kb_user"},
	{method: http.MethodDelete, exact: "/api/v1/knowledge_base/user/delete", category: "permission", action: "kb_user_delete", event: "移除知识库用户", object: "kb_user"},
	{method: http.MethodPost, exact: "/api/v1/file/upload", category: "file", action: "upload", event: "上传文件", object: "file"},
	{method: http.MethodPost, exact: "/api/v1/file/upload/anydoc", category: "file", action: "upload_anydoc", event: "anydoc 回传文件", object: "file"},
	{method: http.MethodPost, exact: "/api/v1/crawler/export", category: "export", action: "crawler_export", event: "导出文档", object: "document"},
	{method: http.MethodPost, exact: "/api/v1/audit/export", category: "export", action: "audit_export", event: "导出审计日志", object: "audit"},
	{method: http.MethodPost, exact: "/api/v1/backup", category: "backup", action: "create", event: "创建备份", object: "backup"},
	{method: http.MethodPost, exact: "/api/v1/backup/restore", category: "backup", action: "restore", event: "恢复备份", object: "backup"},
	{method: http.MethodDelete, prefix: "/api/v1/backup/", category: "backup", action: "delete", event: "删除备份", object: "backup"},
	{method: http.MethodPost, prefix: "/api/v1/approval/", category: "approval", action: "review", event: "审批申请", object: "approval"},
	// ---- 2026-11-12 扫描补充：敏感写操作审计 ----
	{method: http.MethodPut, exact: "/api/v1/audit/storage", category: "config", action: "audit_storage_save", event: "修改审计存储配置", object: "audit"},
	{method: http.MethodPatch, exact: "/api/v1/node/permission/edit", category: "permission", action: "node_permission_edit", event: "更新文档授权", object: "node"},
	{method: http.MethodPut, exact: "/api/v1/system/classification", category: "config", action: "classification_save", event: "保存密级分类", object: "system"},
	{method: http.MethodPost, exact: "/api/v1/knowledge_base", category: "kb", action: "create", event: "创建知识库", object: "kb"},
	{method: http.MethodPut, exact: "/api/v1/knowledge_base/detail", category: "kb", action: "update", event: "更新知识库", object: "kb"},
	{method: http.MethodDelete, exact: "/api/v1/knowledge_base/detail", category: "kb", action: "delete", event: "删除知识库", object: "kb"},
	{method: http.MethodPost, exact: "/api/v1/knowledge_base/release", category: "kb", action: "release_create", event: "创建知识库发布", object: "kb"},
	{method: http.MethodPut, exact: "/api/v1/knowledge_base/category_prompts", category: "kb", action: "category_prompts_save", event: "更新类别提示词", object: "kb"},
	{method: http.MethodPost, exact: "/api/v1/knowledge_base/image_description_templates", category: "kb", action: "image_template_create", event: "创建图片描述模板", object: "kb"},
	{method: http.MethodPut, exact: "/api/v1/knowledge_base/method_rules", category: "kb", action: "method_rules_save", event: "更新方法规则", object: "kb"},
	{method: http.MethodPost, exact: "/api/v1/node", category: "node", action: "create", event: "创建文档", object: "node"},
	{method: http.MethodPut, exact: "/api/v1/node/detail", category: "node", action: "update", event: "更新文档", object: "node"},
	{method: http.MethodPost, exact: "/api/v1/node/move", category: "node", action: "move", event: "移动文档", object: "node"},
	{method: http.MethodPost, exact: "/api/v1/node/batch_move", category: "node", action: "batch_move", event: "批量移动文档", object: "node"},
	{method: http.MethodPost, exact: "/api/v1/node/lock", category: "node", action: "lock", event: "锁定文档", object: "node"},
	{method: http.MethodPost, exact: "/api/v1/node/unlock", category: "node", action: "unlock", event: "解锁文档", object: "node"},
	{method: http.MethodPost, exact: "/api/v1/node/force_unlock", category: "node", action: "force_unlock", event: "强制解锁文档", object: "node"},
	{method: http.MethodPost, exact: "/api/v1/model", category: "config", action: "model_create", event: "创建AI模型", object: "model"},
	{method: http.MethodPut, exact: "/api/v1/model", category: "config", action: "model_update", event: "更新AI模型", object: "model"},
	{method: http.MethodPost, exact: "/api/v1/model/switch-mode", category: "config", action: "model_switch_mode", event: "切换AI模型模式", object: "model"},
	{method: http.MethodPut, exact: "/api/v1/cloud_auth", category: "config", action: "cloud_auth_save", event: "保存平台鉴权配置", object: "cloud_auth"},
	{method: http.MethodPost, exact: "/api/v1/auth/set", category: "config", action: "open_auth_set", event: "设置开放鉴权", object: "auth"},
	{method: http.MethodDelete, exact: "/api/v1/auth/delete", category: "config", action: "open_auth_delete", event: "删除开放鉴权", object: "auth"},
	{method: http.MethodPut, exact: "/api/v1/standard_term", category: "kb", action: "standard_term_replace", event: "替换标准术语", object: "standard_term"},
	{method: http.MethodPut, prefix: "/api/v1/expert_profile/", category: "user", action: "expert_profile_update", event: "更新专家档案", object: "expert_profile"},
	{method: http.MethodPut, exact: "/api/v1/backup/setting", category: "config", action: "backup_setting_save", event: "保存备份策略", object: "backup"},
	{method: http.MethodPut, exact: "/api/v1/test_env", category: "config", action: "test_env_save", event: "保存测试环境配置", object: "system"},
	{method: http.MethodDelete, exact: "/api/v1/comment/list", category: "kb", action: "comment_delete", event: "删除评论", object: "comment"},
	{method: http.MethodPost, exact: "/api/v1/consult/reply", category: "kb", action: "consult_reply", event: "回复咨询", object: "consult"},
	{method: http.MethodPost, exact: "/api/v1/consult/status", category: "kb", action: "consult_status", event: "更新咨询状态", object: "consult"},
	{method: http.MethodDelete, exact: "/api/v1/consult", category: "kb", action: "consult_delete", event: "删除咨询", object: "consult"},
}

func matchAuditRule(method, path string) *auditRule {
	for i := range auditRules {
		r := &auditRules[i]
		if r.method != method {
			continue
		}
		if r.exact != "" && r.exact == path {
			return r
		}
		if r.prefix != "" && strings.HasPrefix(path, r.prefix) {
			return r
		}
	}
	return nil
}

var bodyPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func (m *AuditMiddleware) Record() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rule := matchAuditRule(c.Request().Method, c.Request().URL.Path)
			if rule == nil || m.recorder == nil {
				return next(c)
			}

			reqBody := peekJSONBody(c)
			buf := bodyPool.Get().(*bytes.Buffer)
			buf.Reset()
			writer := &auditWriter{ResponseWriter: c.Response().Writer, body: buf}
			c.Response().Writer = writer

			err := next(c)

			status := writer.statusCode
			if status == 0 {
				status = c.Response().Status
			}
			if status == 0 {
				status = http.StatusOK
			}
			result := domain.AuditResultSuccess
			message := ""
			if err != nil || status >= 400 {
				result = domain.AuditResultFailure
			}
			if msg, ok := parseResponseMessage(writer.body.Bytes()); ok {
				message = msg
			}
			if !parseResponseSuccess(writer.body.Bytes()) {
				result = domain.AuditResultFailure
			}
			objectID, detail := extractAuditObject(rule, c, reqBody)
			m.recorder.AppendFromEcho(c, rule.category, rule.action, rule.event, rule.object, objectID, detail, result, status, message)

			bodyPool.Put(buf)
			return err
		}
	}
}

func peekJSONBody(c echo.Context) map[string]any {
	req := c.Request()
	if req.Body == nil || req.ContentLength == 0 {
		return nil
	}
	ct := req.Header.Get(echo.HeaderContentType)
	if !strings.Contains(ct, echo.MIMEApplicationJSON) {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		return nil
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

func parseResponseMessage(body []byte) (string, bool) {
	var resp struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &resp) != nil || resp.Message == "" {
		return "", false
	}
	return resp.Message, true
}

func parseResponseSuccess(body []byte) bool {
	if len(body) == 0 {
		return true
	}
	var resp struct {
		Success *bool `json:"success"`
	}
	if json.Unmarshal(body, &resp) != nil || resp.Success == nil {
		return true
	}
	return *resp.Success
}

func extractAuditObject(rule *auditRule, c echo.Context, body map[string]any) (objectID, detail string) {
	if body != nil {
		for _, key := range []string{"id", "user_id", "account", "kb_id", "name", "path"} {
			if v, ok := body[key]; ok {
				if s, ok := v.(string); ok && s != "" {
					if objectID == "" && (key == "id" || key == "user_id" || key == "account" || key == "name" || key == "path") {
						objectID = s
					}
					if detail != "" {
						detail += ","
					}
					detail += key + "=" + s
				}
			}
		}
	}
	if objectID == "" {
		if id := c.Param("id"); id != "" {
			objectID = id
		} else if id := c.QueryParam("user_id"); id != "" {
			objectID = id
		} else if id := c.QueryParam("id"); id != "" {
			objectID = id
		}
	}
	if rule.action == "login" || rule.action == "logout" {
		if body != nil {
			if account, ok := body["account"].(string); ok {
				objectID = account
			}
		}
		if user, ok := c.Get(domain.AuthUserKey).(*domain.User); ok && user != nil && objectID == "" {
			objectID = user.Account
		}
	}
	return objectID, detail
}
