package consts

type UserKBPermission string

const (
	UserKBPermissionNull        UserKBPermission = ""             // 无权限
	UserKBPermissionFullControl UserKBPermission = "full_control" // 完全控制
	UserKBPermissionDocManage   UserKBPermission = "doc_manage"   // 文档管理
	UserKBPermissionDataOperate UserKBPermission = "data_operate" // 数据运营
	UserKBPermissionAuditManage UserKBPermission = "audit_manage" // 审核管理（发布文档）
	UserKBPermissionUserManage  UserKBPermission = "user_manage"  // 用户管理
	UserKBPermissionConsultManage UserKBPermission = "consult_manage" // 咨询管理（疑难咨询与用户反馈）
)

type UserKBPermissions []UserKBPermission

func (p UserKBPermissions) Contains(perm UserKBPermission) bool {
	for _, v := range p {
		if v == perm || v == UserKBPermissionFullControl {
			return true
		}
	}
	return false
}

type UserRole string

const (
	UserRoleAdmin         UserRole = "admin"          // 系统管理员
	UserRoleUser          UserRole = "user"           // 普通用户
	UserRoleGuest         UserRole = "guest"          // 访客用户
	UserRoleSecurityAdmin UserRole = "security_admin" // 安全管理员：只负责审批用户增删与权限变更
	UserRoleAuditAdmin    UserRole = "audit_admin"    // 审计管理员：只能查看、导出审计日志
)

const (
	SecurityAdminAccount = "secadmin"
	AuditAdminAccount    = "auditadmin"
)

// IsSeparatedAdminRole 安全管理员与审计管理员只能访问各自的功能，与系统管理员的权限互不重叠
func IsSeparatedAdminRole(role UserRole) bool {
	return role == UserRoleSecurityAdmin || role == UserRoleAuditAdmin
}
