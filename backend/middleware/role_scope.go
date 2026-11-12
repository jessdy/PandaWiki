package middleware

import (
	"net/http"
	"strings"

	"github.com/chaitin/panda-wiki/consts"
)

type roleRoute struct {
	method string // 空表示任意方法
	path   string
	prefix bool
}

var separatedRoleCommonRoutes = []roleRoute{
	{method: http.MethodGet, path: "/api/v1/user"},
	{method: http.MethodPost, path: "/api/v1/user/logout"},
	{method: http.MethodGet, path: "/api/v1/system/alerts"},
	{method: http.MethodGet, path: "/api/v1/system/info"},
}

// 安全管理员、审计管理员只能访问白名单内的接口，其余一律拒绝
var separatedRoleRoutes = map[consts.UserRole][]roleRoute{
	consts.UserRoleSecurityAdmin: {
		{path: "/api/v1/approval", prefix: true},
		{method: http.MethodGet, path: "/api/v1/user/list"},
	},
	consts.UserRoleAuditAdmin: {
		{path: "/api/v1/audit", prefix: true},
	},
}

func (r roleRoute) match(method, path string) bool {
	if r.method != "" && r.method != method {
		return false
	}
	if path == r.path {
		return true
	}
	return r.prefix && strings.HasPrefix(path, r.path+"/")
}

func roleAllowsRequest(role consts.UserRole, method, path string) bool {
	if !consts.IsSeparatedAdminRole(role) {
		return true
	}
	if method == http.MethodHead {
		method = http.MethodGet
	}
	for _, r := range separatedRoleCommonRoutes {
		if r.match(method, path) {
			return true
		}
	}
	for _, r := range separatedRoleRoutes[role] {
		if r.match(method, path) {
			return true
		}
	}
	return false
}
