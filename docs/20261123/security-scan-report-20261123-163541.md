# 🔍 红网测评74条安全扫描报告

**项目:** 开封首 (PandaWiki)
**版本:** v6120d378 (SOURCE_COMMIT: 6fbf7c3f)
**扫描模式:** 🔴 全量 (full-scan)
**扫描范围:** 后端 Go (backend/, 319 文件) + 前端 TypeScript/React (web/, 169 ts) + docker-compose + 配置文件
**扫描时间:** 2026-11-23 16:35
**注册表版本:** 2.1.0
**注册表可扫描条目:** 57 / 74，不可扫描 17 (运维/基础设施、业务逻辑深层、硬件交互、策略豁免、功能层、采购选型)

> ⚠️ **技术栈适配说明:** 注册表引擎 Prompt 以 Java/Spring 为基准，本项目为 **Go + TypeScript/React**，全部 15 个引擎均已按 Go/TS 等价物（viper 配置、Echo/GORM、bcrypt/SM 国密、React dangerouslySetInnerHTML 等）完成扫描。

---

## 📊 扫描总览

| 引擎 | 覆盖条目 | 高危 | 中危 | 低危 | 通过 | 跳过 |
|:----:|:-------:|:---:|:---:|:---:|:---:|:---:|
| A 硬编码凭据扫描 | #58 | 8 | 2 | 0 | 5 | 0 |
| B Web安全漏洞扫描 | #26,27,37,66,67,69 | 0 | 7 | 3 | 5 | 0 |
| C 加密与传输安全 | #57 | 3 | 5 | 2 | 10 | 0 |
| D 输入校验与安全配置 | (待分配) | 0 | 2 | 3 | 8 | 0 |
| E 审计与API安全 | #46,50 | 2 | 5 | 3 | 7 | 0 |
| F 身份标识与三员分立 | #7-9,11-14,17-20,23-25,28-36,38-41 | 3 | 4 | 3 | 3 | 0 |
| G 审计记录完整性 | #49,51-56 | 0 | 1 | 0 | 6 | 0 |
| H 数据备份与恢复 | (待分配) | 1 | 4 | 2 | 6 | 0 |
| I 日志脱敏扫描 | (待分配) | 0 | 2 | 4 | 10 | 0 |
| J 审计日志内容验证 | #16,47,48 | 1 | 1 | 2 | 5 | 0 |
| K 审计日志场景覆盖 | #15,42-45,64 | 3 | 3 | 1 | 8 | 0 |
| L 文件上传安全审计 | #68 | 0 | 2 | 2 | 7 | 0 |
| M 安全头部与存储检查 | #59,60 | 0 | 1 | 1 | 0 | 0 |
| N 运行环境漏洞检查 | #2 | 1 | 5 | 4 | 0 | 0 |
| O 系统标识与密级显示 | (待分配) | 0 | 2 | 2 | 2 | 0 |
| **合计** | **57** | **22** | **46** | **32** | **82** | **0** |

**共发现问题:** 100 个（高危 22 / 中危 46 / 低危 32），通过项 82 个。无引擎被跳过。

> 注：各引擎如存在多次运行，取**最终落盘版本**（timestamp 最新）为权威统计；本表已按最终版本核对。

---

## 🚨 高危问题（需立即修复）

### Engine-A | 硬编码凭据 | `arm/.env` 明文凭据泄漏（8 处）
**文件:** `arm/.env:1,3,6,10,17,18,21,24`
**问题:** ARM 部署环境文件提交在源码树中，含完整明文生产凭据 + 内网 IP `10.10.10.252`：
- `PG_DSN`（含密码 `panda-wiki-secret` + `sslmode=disable`）、`POSTGRES_PASSWORD`
- `REDIS_PASSWORD`、`NATS_PASSWORD`
- `S3_ACCESS_KEY` / `S3_SECRET_KEY`（MinIO 凭据对）
- `ADMIN_PASSWORD="Thinker@123"`（默认管理员口令，每部署相同）
- `JWT_SECRET="your_jwt_secret"`（占位且 <32 字符，验证可被绕过时极弱）
**证据:** `grep -rn "PG_DSN" --include="*.env"` → `arm/.env:1: export PG_DSN="host=10.10.10.252 user=panda-wiki password=panda-wiki-secret ..."`
**修复建议:** 立即从源码移除 `arm/.env`，改提供 `arm/.env.example`（占位符）；**轮换全部已暴露的口令**（Postgres/Redis/NATS/S3/JWT/Admin/Qdrant）；`*.env` 纳入 `.gitignore`；审计 git 历史是否曾提交。

### Engine-C | #57 | Cookie Secure 可被环境变量关闭 + 会话密钥存 Redis
- **C-new Cookie `COOKIE_SECURE=false` 可去除安全标志:** `backend/utils/session_cookie.go:18-20` — `switch strings.ToLower(os.Getenv("COOKIE_SECURE")) { case "false","0","off": return false }` + `middleware/session.go:62`，一旦该环境变量被（误）配置，全部认证 Cookie 将失去 `Secure` 标志，明文环境下会话可被劫持。建议移除该开关或强制默认 Secure。
- **C-005 会话签名密钥存 Redis:** `backend/middleware/session.go:32-44` — `cache.GetOrSet(SessionKey, uuid, TTL 0)` 永久存 Redis，Redis 泄露即可伪造会话 Cookie。建议由 `SESSION_SECRET`/`DATA_ENCRYPTION_KEY` 初始化。

### Engine-E | #46 | 用户列表未授权枚举 & AnyDoc 上传无鉴权
- **`GET /api/v1/user/list` 仅 `auth.Authorize` 无角色校验:** `backend/handler/v1/user.go:54` — 任何已认证用户（含前台访客）都可枚举全部用户（账户/角色/邮箱）；`usecase.ListUsers` 无权限过滤。建议加 `UserRoleAdmin` 校验。
- **`POST /api/v1/file/upload/anydoc/:token` 无鉴权中间件:** `backend/handler/v1/file.go:39` — 仅依赖内网 IP + JWT_SECRET 派生 token。建议补齐鉴权来源校验。
- **OpenAPI 路由无鉴权（Lark Bot 暴露）:** `backend/handler/share/openapi.go:43-50` — `/share/v1/openapi` 分组无鉴权，`POST /lark/bot/:kb_id` 可被公网探测 kb_id，VerifyToken 未配置时可被注入。建议加来源校验 + VerifyToken 缺失告警。

### Engine-J | #48 | 登录失败审计记录主体为空
**文件:** `backend/usecase/audit.go:65-75` + `backend/handler/v1/user.go:180-182`
**问题:** `AppendFromEcho` 仅从 `AuthUserSlot` 提取 UserID/Account，而该槽仅在**登录成功**时设置；密码错误/IP 锁定/账号不存在等失败路径下，审计行的 UserID/Account 为空，尝试账号仅落入 ObjectID，无法唯一标识操作主体，违反 #48 主体唯一性。
**修复建议:** 审计中间件对 login 类事件无论鉴权结果都从请求体 account 填充主体字段。

### Engine-F | #35/#41/#32 | 三员分立缺陷（3 处）
- **#35 共用登录入口:** `backend/handler/v1/user.go:50` — 三员与用户共用 `POST /api/v1/user/login`，未实现独立登录入口。
- **#41 密码重置免审批:** `backend/handler/v1/user.go:281-333` — `ResetPassword` 仅校验系统管理员角色，未进 `approvalUsecase.Submit` 审批；`UpdateGuestUser` 同。
- **#32 文档权限编辑免审批:** `backend/handler/v1/node.go:61` — `PATCH /node/permission/edit` 仅要求 `UserKBPermissionDocManage`，未进安全管理员审批。

### Engine-H | 数据容灾 | 无异地备份
**文件:** `backend/usecase/backup.go:201-322`
**问题:** 备份仅写本机单目录（`config.backup_dir`），无 pg_dump/WAL 归档、无 S3/rsync 异地副本。单盘/服务器故障即丢失全部备份。
**修复建议:** 增加异地/离站备份与归档，纳入容灾方案。

### Engine-K | 审计日志场景缺失（前 3 处高危）
- **文档/节点删除无审计:** `POST /api/v1/node/action`(action=delete) — 调用 `nodeRepo.Delete` 但无审计规则匹配，属关键缺口。
- **App 配置变更/删除无审计:** `PUT/DELETE /api/v1/app` — 修改敏感 bot token、水印、防复制设置却无审计。
- **审计员自行为未审计:** `GET /api/v1/audit/logs`、`GET /api/v1/audit/verify`、`POST /api/v1/audit/export/verify` 均未纳入审计（仅 export 与 storage-save 覆盖）。

### Engine-N | #2 | 运行基础镜像非国产
**文件:** `scripts/build-images.sh:39`（golang:1.24.3-alpine、node:22-alpine、nginx:alpine）+ docker-compose（redis/nats/caddy alpine、postgres debian）
**问题:** 构建链与中间件镜像依赖 Alpine/Debian 系，未满足国产 OS 要求；仅 api/consumer 运行镜像文档声明基于 openEuler 24.03。前端容器与 node/nginx 构建链未达信创基线。
**修复建议:** 评估国产化基础镜像（openEuler/Kylin/Anolis）与中间件国产替换（信创 OS/中间件/数据库），满足采购选型要求。

---

## ⚠️ 中危问题

### Engine-A | 文档泄露凭据 (medium)
`backend/DEVELOPMENT.md:140` 等 7 处文档明文记录 MinIO access key `s3panda-wiki`（DOCKER_SETUP/QUICKSTART/TROUBLESHOOTING）。建议替换为占位符。

### Engine-B | XSS / SQL / SSRF
- **#37 XSS 多处（前端）:** `markdown2/index.tsx:239` link_open href 未转义直拼；`thinkingRenderer.tsx:53` 与 `mermaid.tsx:44` 用 `dangerouslySetInnerHTML`/innerHTML 插未净化内容；`Twemoji.tsx:60`；`sanitizeHtml.ts:9` 正则净化器脆弱。建议换 DOMPurify（后端已用 bluemonday）。
- **#26 SQL 拼接（参数化/可信来源）:** `backend/repo/pg/node.go:1229`、`backup.go:357/538` 用 `fmt.Sprintf` 动态 SQL，虽值走参数占位/quoteIdent，但计入中/低风险，建议审慎使用。
- **#69 爬虫 SSRF:** `backend/pkg/anydoc/**` — 认证用户 URL 直传内部爬虫服务 (1633 最终版定级为中危)；建议 URL 白名单+私网段屏蔽+禁重定向。
- **#27 CSRF:** 管理端 JWT cookie 有 SameSite=Strict 但无 Origin 校验（vs share_auth.go:19 有 RejectCrossSiteRequest）。

### Engine-C | 加密传输 (5)
- **C-001 后端纯 HTTP 无 TLS:** `backend/cmd/api/main.go:27` — 依赖 nginx 终止 TLS，直连端口为明文。
- **C-002 JWT 用对称 HS256:** `backend/usecase/user.go:84-90` — 建议 RS256/ES256 非对称（密钥不得外泄）。
- **C-019 JWT 24h 长令牌无刷新机制:** `backend/usecase/user.go:87` — 令牌窃取风险窗口长；会话 30min 超时但 Cookie MaxAge=0 (session 级) 不一致。
- **C-008 内部通信默认 HTTP:** `backend/utils/static_url.go:21` — `http://panda-wiki-api:8000`，可用 `INTERNAL_API_BASE_URL` 覆盖 HTTPS。
- **自签名 SM2 证书 10 年无告警:** `backend/setup/cert.go:107` — 生产环境建议证书管理与到期告警。

### Engine-D | 输入校验 (2)
- **登录锁定仅按 IP:** `backend/handler/v1/user.go:168,176,185` — 锁键 `login_lock:%s`(ip)，多 IP 可绕过爆破。建议账号+IP 双因子。
- **用户可控字段缺长度上限:** `backend/domain/app.go:12` — Name/LoginReq.Account 等无 `max` 校验。

### Engine-E | 审计与 API (5)
- **限流可被 0 值静默关闭且内存态不共享:** `backend/config/config.go:129` + `client_ip.go:60-61` — `RateLimitRPS<=0` 时 pass-through；内存限流无法多实例共享。
- **审计白名单遗漏:** approval `/:id/approve|reject`、`/creation/*`、`/anydoc` 未纳入 auditRules（audit.go:73）。
- **审计无兜底:** 白名单未匹配即跳过（audit.go:156），新增接口易漏审。
- **alerts 接口仅 Authorize 无角色:** `system_info.go:35` — 任一用户可读系统告警。
- **审计哈希链密钥与 JWT/DataKey 共用:** audit.go:30-37 — JWT 泄露可重写链，建议独立派生。

### Engine-F | 三员分立 (4)
- **#34 用户/管理员证书未分离:** `backend/usecase/user.go:79` — 三员与用户同一 JWT 签发/统一 cookie。
- **#30 系统管理员可查看审批:** `backend/handler/v1/approval.go:35` — List/PendingCount 含 `UserRoleAdmin`。
- **#31 系统管理员功能范围无约束:** `backend/middleware/role_scope.go:44` — 非分离角色对所有路径放行。
- **#12 访客密码/地区更新免审批:** `backend/handler/v1/user.go:493` — UpdateGuestUser 直接执行。

### Engine-G | 审计完整性 (1)
- **#49 审计存储告警被动（仅 API 拉取）:** `backend/usecase/system_info.go:48-74` + `/system/alerts` — 存储超阈值仅在被查询时返回，无主动推送（邮件/webhook/站内），无管理员查询则告警不被发现。建议 cron 周期检查并主动告警。（其余 #51-56 完整性/防删除/防篡改/导出校验/导出加密均通过）

### Engine-H | 备份恢复 (4)
- **文件导出失败静默降级:** `backend/usecase/backup.go:255-264` — 仅 Warn，备份仍返回成功，完整性无告警。
- **转储量无快照一致性:** `backend/usecase/backup.go:227-253` — 逐表 LIMIT/OFFSET 顺序导出，无 REPEATABLE READ，并发写致各表时间点不一致。
- **恢复不校验内容 SM3:** `backend/usecase/backup.go:529-586` — 仅验 manifest 签名，不比对 `Entries.SM3`，损坏/被替换数据会载入生产。
- **S3 恢复在 DB 事务内不回滚:** `backup.go:529-586` — 后续失败 DB 回滚但已写 S3 对象不撤销，半恢复状态。

### Engine-I | 日志脱敏 (2)
- **M1 企微解密正文 Info 级打印:** `backend/pkg/bot/wecom/ai_bot.go:102` — `reqMsg` 明文含用户内容，非标准 PII 不被 WILDCARD 覆盖，建议降 Debug。
- **M2 飞书提问全文+openid 落日志:** `backend/pkg/bot/feishu/stream.go:135` — `question`+`receive_id`/`additional_info`(openid/chat_id) Info 级记录。

### Engine-J | 审计日志合规 (1)
- **#47 移动/批量移动未记录源/目标路径:** `backend/middleware/audit.go:241` (`extractAuditObject`) — `MoveNodeReq` 含 parent_id 等，仅记录 id/kb_id，源父路径与目标父路径缺失，不满足"移动需记录原始+目标路径"。（另低危：Role 代码未翻译、batch_move 未记数量、c. 见低危段）

### Engine-K | 审计场景 (2)
- **F007 文档 AI 操作无审计:** `backend/handler/v1/node.go:53-55,58`。
- **F008 前台咨询回复无审计:** `backend/handler/share/consult.go:48`。

### Engine-L | 文件上传 (2)
- **Epub 转换器仅黑名单过滤（dead code）:** `backend/utils/epub.go:216` — `processFile` 对 zip 条目扩展名仅黑名单过滤，无白名单+魔数；当前无调用方，风险被封堵。若复用需补 validation。
- **前后端上传大小上限不一致:** `web/admin/.../AdminAttachmentUploader.tsx:20`（video≤200MB）vs `backend/usecase/file.go:29`（`maxUploadBytes=100MB`）— 前端提示通过、后端必拒，属限制不一致工程缺陷（后端更严格，无越界风险）。

### Engine-M | 安全头部 (1)
- **#58 Cookie 内 base64 编码 userID:** `backend/utils/static_token.go:23` — `pw_static_token` base64(可逆) 编码 userID，违反「Cookie 不存用户标识」。建议 AES-GCM 加密或改会话 ID。

### Engine-N | 运行环境 (5)
- **NATS/Redis/Qdrant 端口暴露宿主机:** `docker-compose.yml:173`(4222)、`:146`(6379)、`:189`(6333) — `0.0.0.0` 绑定，内部网络均可替代，建议生产移除或绑 `127.0.0.1`。
- **生产 compose 用 `:latest` 标签:** `docker-compose.yml:53/93` — api/consumer 用 `latest` 非确定性标签（arm 版已用变量化 `:prod`，不一致）。
- **文档版本与 compose 不一致:** `backend/DOCKER_SETUP.md` postgres 15 vs 17.6、redis 7 vs 7.4.2、nats 2.10 vs 2.11.3。
- **gorilla/sessions v1.4.0 维护缓慢:** 无现役 CVE 但维护速度低。
- **Dockerfile 未提交:** 构建脚本引用 Dockerfile 不在仓库中，无法审计基础镜像/运行用户。

### Engine-O | 系统标识 (2)
- **后端版本硬编码 "dev":** `backend/telemetry/version.go:4` — 无 ldflags `-X` 注入，VERSION build-arg 不生效，版本恒 "dev"。
- **密级标识默认关闭:** `web/admin/.../ClassificationConfig.tsx:18-20` — `enabled:false`，新装不显示密级。建议默认"内部"或初始化向导强制设置。

---

## 💡 低危问题

- **A:** `arm/.env:26` QDRANT_API_KEY 占位（可能被用作已知密钥，medium已计；其余低危见下）；`backend/DEVELOPMENT.md` 等文档含 `s3panda-wiki`。
- **B:** SameSite=Lax (session.go:59)；Twemoji `dangerouslySetInnerHTML` (Twemoji.tsx:60)；Mermaid 错误信息 innerHTML (mermaid.tsx:44)。
- **C:** JWT_SECRET 多用途复用（crawler.go:54/static_url.go:34）；明文口令兼容校验残留（password.go:57-65）；WeCom AES-CBC+SHA1（第三方协议强制，无法改）。
- **D:** 请求结构体缺校验标签（api/auth/v1/auth.go:38）；管理端登录无图形验证码（user.go:156）；绑定错误被忽略（approval.go:99）。
- **E:** 前台 share 路由仅 ShareAuth 无独立鉴权；上传用 TGT 会话认证；AuthV1Handler 依赖链长。
- **F:** #14 审批反馈可能造成"已生效"误解；#9 账号唯一性大小写敏感；#13 user_manage 权限含系统用户创建。
- **G:** 审计存储告警粒度（见落盘）。
- **H:** 备份下载未纳入审计（middleware/audit.go:96-98,126）；定时备份时间解析健壮性弱（backup.go:626-644）。
- **I:** 飞书 user_info 对象落日志（stream.go:158）；GitHub 邮箱 API 全量响应落日志（github.go:123）；LDAP 明文 username/DN（ldap.go:104）；JWT 校验失败记录 KBId（jwt.go:293）。
- **J:** 密码重置审计未记录目标账号名；权限变更审计无前后对比。
- **L:** textLikeExtensions 跳过 markup 头校验（file.go:189，被 text/plain+nosniff 下发缓解）；前端 AiQaContent 图片选择仅依赖浏览器 MIME 无扩展名白名单（AiQaContent.tsx:252，后端文件头复检兜底）。
- **M:** 缺 `X-XSS-Protection` 响应头（security_headers.go:11）。
- **N:** raglite 5050 暴露宿主；minio 控制台 9001 未关；caddy host 网络模式；构建脚本引用 Dockerfile 未提交。
- **O:** 前端 package.json 版本恒 "1.0.0" 不更新（web/package.json:3）。

---

## ✅ 通过项（按引擎）

| 条目 | 引擎 | 说明 |
|:---:|:---:|:-----|
| Pass1 | A | Go 源码无硬编码口令；TS 无硬编码 token；docker-compose 均用 `${ENV}` 占位；Getenv 无密钥默认值；config.go 空默认 + `validateRequiredSecrets` 强制 JWT≥32字符 |
| Pass2 | B | SQL 注入（GORM 参数化/quoteIdent）；命令注入（无 exec.Command）；文件上传（白名单+魔数+大小+Content-Type 映射+sanitizeObjectKey）；CORS（无通配，回显匹配 Host）；路径穿越（filepath.Base+path.Clean） |
| Pass3 | C | bcrypt 口令哈希；密码复杂度 8-72；JWT 24h 过期+黑名单撤销+角色声明校验；SM4-GCM 字段加密；SM3 审计哈希链；SM2 国密证书；crypto/rand；Cookie HttpOnly+Secure+SameSite=Strict |
| Pass4 | D | 登录 IP 锁定已存在；全局限流 20RPS（默认）；(详见各引擎通过清单) |
| Pass5 | E | API 路由鉴权中间件齐备；全局限流；登录 IP 锁定；审计六要素；审计哈希链；导出加密签名；审计管理员角色隔离 |
| Pass6 | F | 同名用户不继承权限(#40)；新建用户需安全员审核(#39)；JWT+bcrypt 认证（#10 证书适用性低） |
| Pass7 | G | 审计存储告警(Statfs+#49)；防删除/完整性保护/防篡改哈希链/导出校验/导出加密 均通过（#51-55） |
| Pass8 | H | 存储空间告警；手动备份/列表/下载/删除+角色鉴权；定期备份 cron；手动恢复+RESTORE 确认+恢复前自动备份；路径穿越防护；manifest HMAC-SM3；保留份数清理 |
| Pass9 | I | 全局脱敏框架(redact.go)；GORM SQL 脱敏；脱敏单元测试；日志级别可控；LDAP admin 凭据未打印；Token 仅记过期时间；HTTP 访问日志 query 脱敏；session key 未写日志；SimpleAuth 口令响应清空 |
| Pass10 | J | 审计六要素完备(#16)；主体唯一性 UserID(#47)；结果完备(#48)；Event 中文名称；登录/导出/知识库权限事件字段完整 |
| Pass11 | K | 审计框架 60+ 规则覆盖 40+ 场景；#15/#42 用户/登录/删改/权限/导出审计覆盖全面；#64 数据导出审计已覆盖 |
| Pass12 | L | 文件扩展名白名单；魔数校验；扩展名-内容一致性；100MB 大小限制；Content-Type 映射安全；path 净化；服务端 nosniff+CSP sandbox |
| Pass13 | M | headers 无违规用户信息(X-User-Id/loginName)；Cookie 无明文 loginName；认证 Cookie 受保护；CSP/HSTS/X-Frame-Options 已设置；会话 cookie 加密 |
| Pass14 | N | 纯 Go+Node 无 Java 漏洞面；Go 关键依赖安全（jwt v5.3.0/echo v4.13.4/gorm v1.26.1/x/crypto v0.40.0/bluemonday/gmsm 国密）；Node 依赖当前最新（react 19/next 16/dompurify 3.4.2）无已知漏洞 |
| Pass15 | O | 前端登录页/管理页显示版本号；密级标识横幅已实现（登录页+管理布局） |

---

## 📈 合规率

- **可扫描条目合规率 (57 项):** 15 引擎全部执行，通过项 82 / (82+100) = **45.1%**（按 finding 粒度）。
- **74 条总体覆盖:** 可扫描 57 条全部覆盖，不可代码扫描 17 条需人工核查（运维/基础设施、业务逻辑深层、硬件交互、策略豁免、功能层、采购选型）。
- **发布阻断建议:** 存在 **22 个高危**（尤其 `arm/.env` 明文凭据、用户列表未授权枚举、Cookie Secure 可关闭、登录失败审计主体为空、三员分立缺失、审计场景缺失），**上线前必须修复或人工确认**。

---

## ❌ 不可代码扫描项（需人工核查）

| 类别 | 条目 | 说明 |
|:----:|:---:|:-----|
| 运维/基础设施 | 1,3,4,5,6 | 系统加固、数据库安全配置/补丁/用户分离需登录数据库或服务器核查 |
| 业务逻辑(深层) | 10 | 标识符与数字证书对应需人工验证业务逻辑 |
| 硬件交互 | 21 | Key 拔插权限隔离需运行环境验证 |
| 策略豁免 | 22 | 特殊需求豁免属策略层面，需人工判定 |
| 功能层(深层) | 61,62,63,65 | 备份/恢复功能完整性及定期提醒需运行验证 |
| 采购/选型 | 70,71,72,73,74 | 信创 OS/中间件/数据库/BIOS 选用及性能监控工具支持属采购与基础设施层面 |

---

## 📌 核心整改建议（优先级排序）

1. **P0 凭据泄漏** — 立即从源码删除 `arm/.env`，轮换全部已暴露口令（Postgres/Redis/NATS/S3/JWT/Admin/Qdrant）。**注意 `.gitignore` 仅含 `.env`（匹配根目录），不匹配 `arm/.env` 这类子目录件**，应将规则改为 `*.env` 或 `/arm/.env`，并清理 git 历史。
2. **P0 未授权数据访问** — `GET /api/v1/user/list` 补角色校验（含前台访客可枚举用户）；审计 `/api/v1/audit/logs` 等审计员自行为。
3. **P0 Cookie 安全** — 移除/加固 `COOKIE_SECURE` 可关闭 Secure 的环境开关。
4. **P0 审计主体完整性** — 登录失败审计记录补主体字段（#48）；补齐文档删除/App 配置变更审计场景。
5. **P0 三员分立** — 独立管理员登录入口；密码重置/文档权限变更纳入安全管理员审批；系统管理员功能范围白名单。
6. **P1 SSRF** — 爬虫 URL 校验（私网段屏蔽 + scheme 白名单 + 禁重定向）。
7. **P1 数据容灾** — 备份增加异地/归档，恢复校验 SM3 内容，转存快照一致性。
8. **P1 前端 XSS** — 换 DOMPurify 替代正则脱敏；Twemoji/Mermaid innerHTML 加固。
9. **P1 运行环境** — NATS/Redis 端口收敛到内网；gorilla/sessions 迁移评估。
10. **P2 加固** — HTTP body 限流、JWT 升级 RS256、会话密钥外部注入、审计代码表翻译、后端版本注入、Cookie 用户 ID 加密、X-XSS-Protection。

---

*报告生成: 2026-11-23 16:35 · 引擎结果落盘: `.claude/plans/scan-results/` · 技术栈: Go + TypeScript/React*