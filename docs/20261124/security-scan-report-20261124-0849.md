# 🔍 红网测评74条安全扫描报告

**项目：** 开封首（PandaWiki）
**项目版本：** e8b1c95f
**扫描模式：** 全量
**扫描范围：** 全部源码（backend/ Go 328 文件 + web/ TS/React 前端）
**扫描时间：** 2026-11-24 08:49
**注册表版本：** 2.1.0
**完成引擎数：** 15/15（引擎 A–O 全部执行完成，无跳过、无失败）

> 说明：本项目为 **Go + TypeScript/React** 技术栈，非 Java/Spring 项目。registry.yaml 中面向 Java/SQL/Vue 的 tier3 规则（application.yml、SecurityConfig.java、logback.xml、*.sql、*.vue 等）无对应文件，相关引擎已按 Go/TS 实际形态适配检测。

---

## 📊 扫描总览

| 引擎 | 覆盖条目 | 高危 | 中危 | 低危 | 通过 |
|:----:|:-------:|:---:|:---:|:---:|:---:|
| A 硬编码凭据 | #40 #58 #29 | 0 | 0 | 0 | 7 |
| B Web安全漏洞 | #33 #34 #35 #36 #37 #26 | 0 | 0 | 1 | 5 |
| C 加密与传输安全 | #18 #22 #25 #19 #20 | 1 | 2 | 0 | 2 |
| D 输入校验与安全配置 | #32 #3 #4 #5 #30 | 0 | 0 | 0 | 5 |
| E 审计与API安全 | #60 #61 #62 #15 | 0 | 0 | 0 | 4 |
| F 身份标识与三员分立 | #9-#41(12项) | 0 | 2 | 0 | 11 |
| G 审计记录完整性 | #49 #51 #52 #53 #55 #56 | 0 | 0 | 0 | 6 |
| H 数据备份与恢复 | #60 #61 #62 #63 #65 | 0 | 0 | 0 | 5 |
| I 日志脱敏 | #20 #57 | 0 | 0 | 2 | 6 |
| J 审计日志内容合规 | #47 #48 | 3 | 0 | 0 | 0 |
| K 审计日志场景覆盖 | #15 #42 #43 #44 #45 #64 | 2 | 2 | 1 | 1 |
| L 文件上传安全 | #68 | 0 | 0 | 0 | 1 |
| M 安全头部与数据存储 | #58 #59 | 2 | 0 | 1 | 1 |
| N 运行环境与组件漏洞 | #2 | 0 | 2 | 1 | 6 |
| O 系统标识与密级版本 | #4.2.2 | 0 | 0 | 2 | 1 |
| **合计** | | **8** | **8** | **8** | **61** |

**总计发现问题：24 个**（高频 8 / 中危 8 / 低危 8）

---

## 🚨 高危问题（需立即修复）

### Engine-C | 第18条 | 传输加密缺失
**文件：** `backend/cmd/api/main.go:27`（及 `server/http/http.go:62`、`web/app/src/proxy.ts:131-136`）
**代码：** `echo.Start(fmt.Sprintf(":%d", port))`（明文 HTTP 启动，非 StartTLS）
**问题：** Go 后端以纯明文 HTTP 启动，无 TLS 监听、无 HTTPS 强制跳转；HTTPS 仅依赖可选的外部 nginx 反代（自签证书）。前端 Next 代理在 HTTP 页面主动剥离 Session Cookie 的 Secure 属性（`stripCookieSecure`），日志注释自认"生产环境若仍是 HTTP"。会话 Cookie/JWT (`panda_wiki_token`)、登录凭据在明文部署下可被中间人窃听/篡改。
**修复建议：** 应用层强制 HTTPS——后端启用 `echo.StartTLS`/TLS 配置，或部署层强制重定向 http→https；移除前端对 Secure 属性的 HTTP 剥离逻辑；确保生产环境统一 HTTPS。
**证据：** `backend/cmd/api/main.go:27` `app.HTTPServer.Echo.Logger.Fatal(app.HTTPServer.Echo.Start(...))`；`web/app/src/proxy.ts:152` `stripCookieSecure` 移除 `Secure`；`web/app/src/proxy.ts:193` 注释承认生产可能为 HTTP。

### Engine-J | 第47条 | 简单口令登录审计主体缺失
**文件：** `backend/handler/share/auth.go:161`
**问题：** 前台简单口令登录（`/share/v1/auth/login/simple`）从未设置 `AuditSubjectAccountSlot` 与 `AuthUserSlot`，请求体仅有 password，导致审计日志 UserID 与 Account 均为空，六要素（时间/地点/类型/主体/客体/结果）中缺**主体**要素。
**修复建议：** 为简单登录补充可辨识的唯一主体标识（会话ID/Key号/约定系统主体）写入 AuditSubjectAccountSlot，确保主体不为空。
**证据：** grep `AuditSubjectAccountSlot|AuthUserSlot` `backend/handler/share/auth.go` → 无匹配；唯一审计写入路径为 `middleware/audit.go:280` / `usecase/audit.go:50`。

### Engine-J | 第47条 | TGT 登录失败路径审计主体缺失
**文件：** `backend/handler/share/tgt_login.go:143`
**问题：** TGT 会话登录在云账号 `loginVerify` 失败（line 143-151）或权限校验失败（line 160-181）时提前 return，`AuditSubjectAccountSlot` 仅在验证成功后（line 153/202）才设置。失败路径审计日志主体为空，无法定位操作主体。
**修复建议：** 在失败返回前记录可识别的主体（平台返回的 LoginName/UserID 或会话来源唯一号）。
**证据：** `tgt_login.go:153/202` 设置主体均位于验证成功分支；`:143-151` 失败路径提前 return 未设主体。

### Engine-J | 第48条 | 登录失败审计主体非唯一标识
**文件：** `backend/usecase/audit.go:76`
**问题：** 登录失败路径仅用 Account（登录名 loginName）作为主体，`UserID` 保持空串（`AuthUserSlot` 未填充）。对已存在用户的失败登录，主体无法用 userId 唯一标识，仅凭可重名 loginName 记录，不符合 #48"主体应为唯一标识符，不可同名"。
**修复建议：** 失败路径当 Account 非空时反查该账号 UserID 一并写入；未知账号在 detail 标注"未知账号"。
**证据：** `backend/usecase/audit.go:65-67` 仅成功路径填充 UserID；`audit.go:76-83` 仅 Account 补齐无 UserID 分支；`handler/v1/user.go:176` 登录前 `c.Set(AuditSubjectAccountSlot, req.Account)` 但无 UserID。

### Engine-K | 第15条 | 审计范围未覆盖电子文档下载
**文件：** `backend/handler/share/static_file.go:64`
**问题：** 文档下载接口 `/static-file/*` 与 `/static-file-signed/*` 完全不在审计中间件覆盖范围。审计中间件 `shouldAuditRequest`（audit.go:166）仅处理 `/api/` 和 `/share/` 前缀，`serveObject` 对非内联类型还设置 attachment 下载（static_file.go:123）。涉密/敏感文档下载操作不可追溯。
**修复建议：** 为 `/static-file/*` 与 `/static-file-signed/*` 下载在审计中间件显式配置 auditRule（category=file, action=download），仅记对象 key/文件名不记全文；或 handler 内调 `AppendFromEcho`。
**证据：** grep `static-file` `backend/middleware/audit.go` → 0 命中；`static_file.go:64-68` 注册 GET/HEAD；`audit.go:166` 前缀白名单不含 static-file。

### Engine-K | 第42条 | 用户关键行为（文档下载）未审计
**文件：** `backend/handler/share/static_file.go:72`
**问题：** 文档下载是用户关键行为（涉密数据/电子文档输出），但 `Serve/ServeSigned/serveObject` 全程无任何审计函数调用，所有 `/static-file/`、`/static-file-signed/` 发起的下载（含附件、签名下载）零审计痕迹。类比其它下载（如 `/api/v1/backup/*/download`）已被显式覆盖。
**修复建议：** 在 `serveObject` 成功返回后补审计（account 取 `c.Get(AuthUserSlot)`，event=下载文档，object=document，detail 仅记对象 key/文件名），失败/越权时记 failure。
**证据：** grep `AppendFromEcho|recordLog|Append\(|audit` `backend/handler/share/static_file.go` → 0 命中；`audit.go:96/124` 表明其它下载均走中间件规则，唯 static-file 无规则无内联。

### Engine-M | 第59条 | OAuth ClientSecret 明文落库
**文件：** `backend/domain/auth.go:85-89`
**代码：** `ClientSecret string \`json:"client_secret,omitempty"\``（明文，`auth_setting` JSONB gorm 类型）
**问题：** OAuth（GitHub 等）ClientSecret 作为 JSONB 明文存入 `auth_configs` 表，字段加密（FieldCipher SM4）未覆盖该路径。数据库泄露即泄露第三方应用机密。明文读回用于 OAuth 流程（auth_github.go:22）。
**修复建议：** 在 `SetAuth`/`CreateAuthConfig` 写库前对 ClientSecret 调用 `FieldCipher.Encrypt`，读库 Decrypt；密文加 `sm4:` 前缀，与 `model.api_key` 同方案。
**证据：** `backend/domain/auth.go:63` `AuthSetting gorm:"type:jsonb"`；`backend/usecase/auth.go:60-64` 原样写入 ClientSecret；`repo/pg/auth.go:201-228` 直接落库无加密。

### Engine-M | 第59条 | AppSettings 集成凭据明文落库
**文件：** `backend/domain/app.go:515-560`
**问题：** `apps.settings` JSONB 明文保存多项第三方集成密钥：`dingtalk_bot_client_secret`、`feishu_bot_app_secret`、`wechat_app_secret`、`wechat_app_encodingaeskey`、`discord_bot_token`、`wechat_official_account_*`、`wecom_ai_bot_settings` 等。字段加密未覆盖应用集成凭据。
**修复建议：** 对 AppSettings 中 Secret/Token/AESKey 字段写库前加密（复用 FieldCipher SM4），读库后解密；或整块 settings 可逆加密后存 JSONB。
**证据：** `backend/domain/app.go:80` `Settings AppSettings json:"settings" gorm:"type:jsonb"`；`usecase/app.go:166`、`repo/pg/app.go:44-47` settings 整体 JSONB `Updates` 无字段加密。

---

## ⚠️ 中危问题

### Engine-C | 第25条 | Cookie Secure 非强制
**文件：** `backend/utils/session_cookie.go:19-38`
**问题：** Cookie HttpOnly/SameSite 正确，但 `CookieSecure()` 在纯 HTTP 时返回 false 不置 Secure；前端代理 HTTP 继续剥离 Secure。明文会话传输风险（与 #18 关联）。
**证据：** `session_cookie.go:19-38` `CookieSecure` proto=="http" 返回 false；`proxy.ts:152` stripCookieSecure。

### Engine-C | 第19条 | 会话超时/重鉴别不完整
**问题：** 前台会话 30min TTL（session_cookie.go:43）+ Redis 强制失效已有；但管理端 JWT 默认 8 小时（session_cookie.go:59-62）、静态令牌 12 小时（static_token.go:21），无空闲重鉴别。
**证据：** `session_cookie.go:43` `SessionCookieMaxAge=30*60`；`session_cookie.go:59-62` `maxAge=8*time.Hour`；`static_token.go:21` `StaticAccessTTL=12*time.Hour`。

### Engine-F | 第10条 | 标识符与证书无绑定
**问题：** 仅发现服务端 TLS(SM2自签)证书、DB SSL、知识库公钥；无 username→X509 个人证书映射。
**建议：** 为安全管理员等关键主体引入个人数字证书/USBKey，证书指纹绑定 User.Account。
**证据：** grep `certificate|cert|X509|证书` → 仅 `setup/cert.go`、`config.go:328`、`repo/pg/knowledge_base.go:93`（全为 TLS/服务端证书或知识库公钥）。

### Engine-F | 第34条 | 用户/管理员证书分离缺失
**问题：** 身份认证是口令(bcrypt)+JWT(HS256)，无证书/mTLS 体系，无法实现用户/管理员证书分离。（#35 登录入口分离已通过 ensureLoginPortal，但属门户级非证书级。）
**证据：** `usecase/user.go:80` 口令+JWT；`usecase/approval.go:292` bcrypt；`usecase/user.go:101` ensureLoginPortal 三门户互斥。

### Engine-K | 第44条 | 电子文档输出审计缺失
**问题：** static_file.go 对非内联类型设 attachment 下载（static_file.go:123），但该文档输出路径无审计。补审计时须 detail 仅记文件名/对象 key，不记全文。
**证据：** `static_file.go:123` `Content-Disposition attachment`；`audit.go:332-346` extractAuditObject 仅提取标识字段。

### Engine-K | 第64条 | 涉密/文档输出审计覆盖（签名下载）
**问题：** 签名下载 `/static-file-signed/:exp/:sig/*`（ServeSigned）同样不受审计覆盖，容器网络内文档输出不可追溯。
**证据：** `static_file.go:67` `GET /static-file-signed/...`；`audit.go:166` 前缀白名单不含；grep "static-file" audit.go → 0 命中。

### Engine-N | 第2条 | 仓库无 Dockerfile / docker-compose，国产OS无法源码级验证
**问题：** 仓库不存在 Dockerfile/docker-compose，部署依赖远程 `manager.sh`；仅有 DOCKER_SETUP.md 文档声明 openeuler 基础镜像。容器基础镜像（国产OS）无法从源码审计确证。
**证据：** Glob `**/Dockerfile*` → No files；README.md:36 依赖 `https://release.baizhi.cloud/panda-wiki/manager.sh`；DOCKER_SETUP.md:16 仅文档声明 openeuler。

### Engine-N | 第2条 | docker-compose.services.yml 引用但缺失
**问题：** DOCKER_SETUP.md:66、DEVELOPMENT.md:31、QUICKSTART.md:75 多处引用 `backend/docker-compose.services.yml`，但该文件不存在于仓库。中间件版本/端口/镜像无法从源码复现。
**证据：** 多文档引用 `backend/docker-compose.services.yml`；Glob `**/docker-compose*` → No files。

---

## 💡 低危问题

### Engine-B | 第33条 | backup.go 动态拼接 SELECT 表名
**文件：** `backend/usecase/backup.go:357` `q := fmt.Sprintf(\`SELECT * FROM %s ORDER BY 1 LIMIT 500 OFFSET %d\`, quoteIdent(table), offset)`
**说明：** table 经 quoteIdent 双引号翻倍转义、offset 为整型自增、表名来自 information_schema 真实表非用户输入，实际**不可利用**，降为 LOW 审计项。多数 `.Raw()` 用 `?` 占位符参数化安全。
**建议：** 无强制修复；如追求规范可进一步用参数化/白名单表名列表。

### Engine-I | 第20条 | GitHub 邮箱 API 原始响应整体记录
**文件：** `backend/pkg/oauth/github.go:123` `c.logger.Info("GetGithubPrimaryEmail:", log.String("buf", string(buf)))`
**说明：** 原始 JSON 响应（含邮箱）以 Info 记录；emailPattern 会脱敏，但建议只记录提取后的邮箱。
**建议：** 改为记录提取处理后的主邮箱，而非原始 buffer。

### Engine-I | 第20条 | CAS 验证响应 XML 记录
**文件：** `backend/pkg/cas/cas.go:157` `c.logger.Info("CAS validation response", log.String("response", string(body)))`
**说明：** CAS 验证响应原始 XML 以 Info 记录，`<proxyGrantingTicket>`/TGT 值不在 `redactString` 正则覆盖内。
**建议：** 降为 Debug 或增加 ticket XML 蒙版化/正则覆盖。

### Engine-K | 第43条 | 审计员导出行为 detail 为空
**文件：** `backend/usecase/audit.go:222`
**说明：** 审计员查询/导出已由中间件覆盖，但 audit export 的 extractAuditObject 无法匹配导出请求体字段，detail/objectID 为空，缺乏导出范围/条数/格式要素。
**建议：** Export 后回写 detail（导出条数 count、格式、导出人）。

### Engine-M | 第59条(低) | 遗留 localStorage Bearer Token
**文件：** `web/admin/src/utils/auth.ts:28`
**说明：** 兼容旧版从 localStorage 读取 JWT token；新版已迁移 HttpOnly Cookie，属遗留兼容分支。若 XSS 窃取 localStorage token 属存储隐患。
**建议：** 移除 localStorage 存储 token 的兼容分支。

### Engine-N | 第2条(低) | config.yml.example 中间件凭据为空
**问题：** config.yml.example 中 Redis/NATS/S3 等 password/secret_key 为空字符串，有文档说明用环境变量注入缓解。
**证据：** `config.yml.example:32/48/54/61/62` 空值，注释建议用环境变量。

### Engine-O | #4.2.2(低) | 版本号仅登录页显示
**问题：** 版本号仅在 app 登录页底部展示（login.tsx:610-621）；主文档页、Admin 后台 Layout 均无版本号。
**建议：** 在 app Footer 与 Admin Layout/Header 添加版本号显示，通过 `/api/v1/system/info` 获取。

### Engine-O | #4.2.2(低) | 版本号默认"dev"，升级一致性不可验证
**问题：** `backend/telemetry/version.go:4` 默认 `"dev"`；Makefile 传 `--build-arg VERSION=${VERSION}`，但 Dockerfile 不在源码快照，无法确认 ldflags 注入是否生效。若缺失版本永远显示"dev"不随升级改变。
**建议：** 确保 Dockerfile 用 `-ldflags "-X .../telemetry.Version=${VERSION}"` 注入版本；CI 增加注入验证步骤；启动时打印版本。

---

## ✅ 通过项

| 条目 | 引擎 | 说明 |
|:---:|:---:|:-----|
| #40 #58 #29 | A | 无硬编码凭据；密码/密钥均环境变量注入；`validateRequiredSecrets` 强制校验；无默认口令 |
| #34 XSS | B | 所有 dangerouslySetInnerHTML/innerHTML 均经 DOMPurify 净化或 React JSX 转义兜底 |
| #35 CSRF | B | 管理端 JWT SameSite=Strict + AdminCSRF Origin/Referer 同源校验；会话 Cookie SameSite=Lax |
| #36 文件上传 | B | 100MB上限 + 黑/白名单 + 魔数 sniffing + Content-Type 不采信客户端 + 路径清洗 |
| #37 命令注入 | B | 全库无 exec.Command/os/exec/sh -c 调用 |
| #26 CORS | B | 无 `Allow-Origin:*`；share CORS 仅回显同源 Origin + Vary |
| #22 TLS版本 | C | 唯一 TLS 配置 nginx `TLSv1.2 TLSv1.3`，无 SSLv3/TLS1.0/1.1/3DES |
| #20 会话用户名绑定 | C | 会话绑定 kb_id+user_id 逐请求校验，存 user_id 非明文 loginName |
| #32 输入校验 | D | handler 全量 c.Validate(155处) + validate tag |
| #3 口令复杂度 | D | `ValidatePasswordComplexity` 8-72位+四类字符，已接全部口令入口 |
| #4 登录失败锁定 | D | IP+账号双维度锁定，阈值5/10/15，时长阶梯递增 |
| #5 超时自动退出 | D | 会话30min + JWT8h过期 + Logout撤销 token 立即失效 |
| #30 安全加固 | D | SecurityHeaders(CSP/X-Frame/nosniff/HSTS) + AdminCSRF 全局注册 |
| #60 API鉴权 | E | JWT/API Token Authorize 覆盖全部 /api/v1 写操作；share 域 ShareAuth + KB 会话鉴权 |
| #61 API限流 | E | 全局 IP 限流 20RPS/60Burst + 登录账号锁定防爆破 |
| #62 API日志审计 | E | 全局 AuditMiddleware 对 /api/ /share/ 写操作记审计，显式规则表+兜底 |
| #15 审计不可篡改 | E/G | audit_logs 仅 INSERT，无 UPDATE/DELETE；HMAC-SM3 哈希链 + /audit/verify 检篡改 |
| #9 标识唯一性 | F | User.Account gorm uniqueIndex 数据库唯一索引 |
| #12 #13 #14 审核流程 | F | pending→approved/rejected 状态机；增删用户/权限变更均 Submit 审批，仅 security_admin 可 approve |
| #30 #31 #32 三员分立 | F | admin/security_admin/audit_admin 三角色职责互斥；白名单路由隔离 |
| #35 登录入口分离 | F | ensureLoginPortal 三门户互斥 |
| #39 新建用户审核 / #40 同名不继承 / #41 权限变更审核 | F | 全部通过 |
| #49 审计存储告警 | G | LimitMB/AlertPercent 可配置，AuditStorageStatus.Alert，system_info 告警 |
| #51 防删除 | G | 无审计删除/清空/truncate 端点，AuditRepo 无 Delete 方法，仅追加 |
| #52 完整性保护 | G | prev_hash+hash HMAC-SM3 哈希链(advisory lock 串行)，Verify 逐条校验 |
| #53 防修改 | G | 唯一 PUT 为 /audit/storage 配置；UI 只读表格；AuditRepo 无 Update |
| #55 导出完整性 / #56 导出加密 | G | SM3 digest + HMAC-SM3 signature + 导出校验接口；SM4-GCM + PBKDF2-SM3(20万次) 加密 |
| #60-#63 #65 备份与恢复 | H | 完整备份(全表+MinIO文件,SM3签名manifest)/恢复(预备份+事务truncate+confirm)/审计存储告警/定期备份调度+7天提醒 |
| #20/#57 日志脱敏 | I | slog ReplaceAttr redactAttr 全局脱敏 + secretKeySuffixes/PII 后缀 + 四级正则蒙版 + GORM SQL 脱敏 |
| #68 文件上传 | L | 后端权威校验：黑白名单+魔数+大小+路径穿越+Content-Type 白名单映射，3上传入口均鉴权 |
| #58 Cookie/headers无用户信息 | M | 会话 Cookie 存会话ID(Redis服务端)，JWT HttpOnly Cookie，静态token XOR+HMAC密封，前端仅转发加密 Cookie/Bearer |
| #2 组件漏洞 | N | Go依赖全部修复已知高危CVE(x/net 0.42, x/crypto 0.40, echo 4.13.4, jwt 5.3.0)；npm Next16.0/Vite6/React19 最新版；无 Java/log4j 生态 |
| #4.2.2 密级标识 | O | 系统实现完整：后端默认"内部"+登录页/Admin Layout 红色密级横幅 + 可配置密级 + 变更审计 |

---

## ❌ 不可代码扫描项

以下条目需人工核查（来源：engines/registry.yaml）：

| 类别 | 条目 | 说明 |
|:----:|:---:|:-----|
| 运维/基础设施 | 1,3,4,5,6 | 系统加固、数据库安全配置/补丁/用户分离需登录数据库或服务器核查 |
| 业务逻辑(深层) | 10 | 标识符与数字证书对应需人工验证业务逻辑 |
| 硬件交互 | 21 | Key拔插权限隔离需运行环境验证 |
| 策略豁免 | 22 | 特殊需求豁免属策略层面，需人工判定 |
| 功能层(深层) | 61,62,63,65 | 备份/恢复功能完整性及定期提醒需运行验证 |
| 采购/选型 | 70,71,72,73,74 | 信创OS/中间件/数据库/BIOS选用及性能监控工具支持属采购与基础设施层面 |

---

## 📈 合规率

- **可扫描条目（57 条）地址统计：** 各引擎共覆盖并通过了涉及 52 个可代码扫描条目的检测项（61 项通过判定，部分条目多引擎交叉覆盖），24 项待整改问题集中在 **8 个高危、8 个中危、8 个低危**。
- **重点整改域：** ① 传输加密/HTTPS 强制（#18/C）；② 审计主体完整性与文档输出审计覆盖（#47/#48/J、#15/#42/K）；③ 敏感数据明文落库（#59/M）；④ 运行容器国产OS可验证性（#2/N）。
- **核心检出率：** 15 引擎全部正常完成，唯一高风险集中点已明确——**上线前务必优先修复 8 项高危**。

### 高危整改优先级（按风险影响）

1. **#59 敏感配置明文落库**（OAuth ClientSecret、AppSettings 集成密钥）— DB 泄露即机密泄露，改动局部、易落地（复用 FieldCipher SM4）。
2. **#47/#48 审计主体缺失/非唯一**（登录失败、简单登录、TGT 失败）— 合规硬伤，修改点明确。
3. **#15/#42 文档下载未审计**（static-file 路径）— 涉密文档输出不可追溯，补中间件规则即可。
4. **#18 传输加密缺失** — 部署侧 + 后端 TLS 双向整改，涉及面较广，建议结合上线环境统一 HTTPS。

---

## 📋 报告信息

- 全量扫描报告：`security-scan-report-20261124-0849.md`
- 引擎结果落盘：`.claude/plans/scan-results/engine-{a..o}-*.md`（15 个文件）
- 引擎注册表：`/install/security-74-scanner/engines/registry.yaml`（v2.1.0）
- 项目路径：`/Sec/panda-wiki/PandaWiki_src20261124`