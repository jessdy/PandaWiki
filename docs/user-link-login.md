# 前台企业登录（仅 TGT）

Base64 `?user=` 授权链接登录已下线（仅编码、无签名，存在冒充风险）。企业用户请使用平台会话令牌（TGT）。

## 登录地址

```text
{站点地址}/auth/login?tgt={tgt}
```

`tgt` 需要做 URL 编码。可选参数 `redirect` 表示登录成功后的站内路径（同样 URL 编码）；未提供时进入站点首页。

```text
{站点地址}/auth/login?tgt={tgt}&redirect={登录后的路径}
```

## 处理流程

1. 调用平台 `loginVerify`，用 `tgt` 换取 `loginName` 和 `userId`。
2. 调用平台 `checkPermissions`，确认该用户拥有 `100000:function:kfsuser`。没有该权限时登录失败。
3. 用 `loginName` 查找本系统账号。
4. 已存在的普通用户直接登录。密码、地区、角色和用户组保持不变。
5. 用户不存在时创建普通用户：密码随机生成，地区固定为 `上海`，并加入当前知识库的全部用户组，然后登录。
6. 同名账号如果不是普通用户，拒绝登录，也不修改原账号。

平台 IP 和端口在后台「系统配置 → 平台地址」中填写。系统按配置的协议调用：

```text
POST {平台地址}/CloudUserManagement/rest/UserManageRestService/loginVerify
POST {平台地址}/CloudAuthorization/rest/CloudAuthorizationRestService/checkPermissions
```

后台未填写时，才使用环境变量 `CLOUD_AUTH_BASE_URL`。

## 接口

前端页面发起：

```text
POST /share/v1/auth/login/tgt
Header: X-KB-ID: {知识库ID}
Body: {"tgt":"..."}
```

已移除：`POST /share/v1/auth/login/user_link`。
