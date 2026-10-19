# 前台用户链接登录

第三方系统把用户浏览器跳转到 系统 登录地址，即可使用指定用户名登录。用户名不存在时，系统按 `role` 创建账号：

- 普通用户出现在后台「用户管理」，并加入当前知识库的全部用户组。
- 管理员出现在后台「管理员管理」，角色为普通管理员，并拥有当前知识库的完全控制。

第三方不需要调用登录接口，也不需要保存密码。只需要生成链接并让用户访问。

## 登录地址

```text
{站点地址}/auth/login?user={Base64编码的用户信息}
```

`user` 必须做 URL 编码后再放入查询参数。Base64 中的 `+`、`/`、`=` 分别编码为 `%2B`、`%2F`、`%3D`。

可选参数 `redirect` 表示登录成功后的站内路径，同样需要 URL 编码。未提供时进入站点首页。

```text
{站点地址}/auth/login?user={用户信息}&redirect={登录后的路径}
```

## 用户信息

先构造 UTF-8 JSON，再进行标准 Base64 编码。

```json
{
  "username": "lisi",
  "region": "上海",
  "role": "user"
}
```


| 字段         | 必填  | 说明                                 |
| ---------- | --- | ---------------------------------- |
| `username` | 是   | 用户名，对应后台账号。也接受 `user_name` 或 `用户名` |
| `region`   | 否   | 地区。新建用户时写入；已有用户不覆盖。也接受 `地区`        |
| `role`     | 否   | 角色。也接受字段名 `角色`。不传时按 `user` 处理      |


`role` 取值：


| 传入值       | 账号                     |
| --------- | ---------------------- |
| 不传、`user` | 普通用户，写入「用户管理」          |
| `admin`   | 管理员，写入「管理员管理」，显示为普通管理员 |


限制：

- 用户名去掉首尾空格后不能为空，最长 100 个字符。
- 地区最长 100 个字符。
- 推荐使用后台地区选项中的省级简称，例如 `北京`、`上海`、`广东`、`四川`。
- 其他角色值会登录失败。超级管理员不能通过此链接创建或登录。



## 处理规则

1. 按用户名查找后台账号。
2. 已存在且角色匹配的账号直接登录。已有账号的密码、地区、角色和权限保持不变。
3. 用户不存在时创建账号：
  - 用户名使用传入值。
  - 密码随机生成，不返回给第三方。
  - 地区使用传入值。
  - 普通用户加入本次访问知识库在后台「用户组」中显示的全部用户组。
  - 管理员获得当前知识库的完全控制。
  - 创建完成后登录。
4. 角色不匹配时拒绝登录，也不修改原账号。例如用管理员链接登录一个普通用户，或用普通用户链接登录一个管理员。

登录成功后，浏览器获得有效期 30 天的登录会话。接口返回的 `role` 为 `user` 或 `admin`。

## 会话令牌登录

第三方也可以把已经拿到的平台会话令牌放到登录地址上：

```text
{站点地址}/auth/login?tgt={tgt}
```

`tgt` 需要做 URL 编码。可选参数 `redirect` 与用户链接登录相同。

系统按下面的顺序处理：

1. 调用平台 `loginVerify`，用 `tgt` 换取 `loginName` 和 `userId`。
2. 调用平台 `checkPermissions`，确认该用户拥有 `100000:function:kfsuser`。没有该权限时登录失败。
3. 用 `loginName` 查找本系统账号。
4. 已存在的普通用户直接登录。密码、地区、角色和用户组保持不变。
5. 用户不存在时创建普通用户：密码随机生成，地区固定为 `上海`，并加入当前知识库的全部用户组，然后登录。
6. 同名账号如果不是普通用户，拒绝登录，也不修改原账号。

平台 IP 和端口在后台「系统配置 → 平台地址」中填写。系统按 `http://IP:端口` 调用：

```text
POST http://IP:端口/CloudUserManagement/rest/UserManageRestService/loginVerify
POST http://IP:端口/CloudAuthorization/rest/CloudAuthorizationRestService/checkPermissions
```

后台未填写时，才使用环境变量 `CLOUD_AUTH_BASE_URL`。

## 示例

李四，上海，普通用户：

```json
{"username":"lisi","region":"上海","role":"user"}
```

```text
/auth/login?user=eyJ1c2VybmFtZSI6Imxpc2kiLCJyZWdpb24iOiLkuIrmtbciLCJyb2xlIjoidXNlciJ9
```

王五，广东，管理员：

```json
{"username":"wangwu","region":"广东","role":"admin"}
```

```text
/auth/login?user=eyJ1c2VybmFtZSI6Indhbmd3dSIsInJlZ2lvbiI6IuW5v%2BS4nCIsInJvbGUiOiJhZG1pbiJ9
```

赵六，四川，登录后打开 `/consult`。未传 `role`，按普通用户处理：

```json
{"username":"zhaoliu","region":"四川"}
```

```text
/auth/login?user=eyJ1c2VybmFtZSI6InpoYW9saXUiLCJyZWdpb24iOiLlm5vlt50ifQ%3D%3D&redirect=%2Fconsult
```

中文字段名同样可用：

```json
{"用户名":"chenqi","地区":"北京","角色":"管理员"}
```

```text
/auth/login?user=eyLnlKjmiLflkI0iOiJjaGVucWkiLCLlnLDljLoiOiLljJfkuqwiLCLop5LoibIiOiLnrqHnkIblkZgifQ%3D%3D
```



## 生成代码

```javascript
const payload = {
  username: "lisi",
  region: "上海",
  role: "user",
};
const user = btoa(unescape(encodeURIComponent(JSON.stringify(payload))));
const url = `${siteOrigin}/auth/login?user=${encodeURIComponent(user)}`;
window.location.href = url;
```

```python
import base64
import json
from urllib.parse import quote

payload = {"username": "lisi", "region": "上海", "role": "管理员"}
user = base64.b64encode(
    json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode()
).decode()
url = f"{site_origin}/auth/login?user={quote(user, safe='')}"
```

```java
String json = "{\"username\":\"lisi\",\"region\":\"上海\",\"role\":\"user\"}";
String user = Base64.getEncoder().encodeToString(json.getBytes(StandardCharsets.UTF_8));
String url = siteOrigin + "/auth/login?user=" + URLEncoder.encode(user, StandardCharsets.UTF_8);
```

