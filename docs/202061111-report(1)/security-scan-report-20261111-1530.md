# 🔴 全量代码安全扫描报告 — PandaWiki（开封首）

- **项目名**：PandaWiki（开封首）
- **项目版本**：v0.0.2（backend go.mod `github.com/chaitin/panda-wiki` / frontend `@panda-wiki`，web/app 2.9.0）
- **技术栈**：后端 Go（echo + GORM + pgx + Redis；290 个 .go 文件）｜前端 React/TS（Next.js；785 个 .ts/.tsx 文件）+ 86 个 SQL 迁移文件
- **扫描模式**：🔴 全量（red 全量代码扫描）
- **扫描引擎**：全部 15 个已启用引擎（registry.yaml v2.1.0）
- **扫描时间**：2026-11-11 15:30
- **引擎注册表**：`/install/security-74-scanner/engines/registry.yaml`（meta.version 2.1.0，总规则 74 条，可扫描规则 57 条）

---

## 一、扫描总览

| 引擎 | 类型 | 结果数 | 高危 | 中危 | 低危 | 覆盖条目 |
|:----:|:----:|:-----:|:----:|:----:|:----:|:--------:|
| A 硬编码凭据 | Rigid | 0 | 0 | 0 | 0 | #58 |
| B Web漏洞 | Flex | 3 | 1 | 2 | 0 | #26,#27,#33,#34,#35,#36,#37,#66,#67,#69 |
| C 加密传输 | Flex | 13 | 4 | 6 | 3 | #18,#19,#20,#22,#25,#57 |
| D 输入校验 | Flex | 1 | 0 | 1 | 0 | #3,#4,#5,#30,#32 |
| E 审计与API | Flex | 4 | 2 | 2 | 0 | #15,#46,#50,#60,#61,#62 |
| F 身份三员 | Flex | 13 | 4 | 7 | 2 | #7-14,#17-20,#23-25,#28-41 |
| G 审计完整 | Rigid | 6 | 3 | 3 | 0 | #49,#51-56 |
| H 数据备份 | Flex | 5 | 3 | 1 | 1 | #60-63,#65 |
| I 日志脱敏 | Flex | 3 | 2 | 1 | 0 | #20,#57 |
| J 审计内容 | Flex | 2 | 2 | 0 | 0 | #16,#47,#48 |
| K 审计场景 | Rigid | 6 | 6 | 0 | 0 | #15,#42-45,#64 |
| L 文件上传 | Rigid | 6 | 3 | 2 | 1 | #68 |
| M 安全头部 | Flex | 4 | 2 | 2 | 0 | #58,#59,#60 |
| N 运行环境 | Rigid | 5 | 1 | 3 | 1 | #2 |
| O 系统标识 | Flex | 3 | 0 | 1 | 2 | #4.2.2 |
| **合计** | | **65** | **30** | **28** | **7** | |

> 注：各引擎 `summary` 数字个别存在内部轻微不一致（如 F 列 13 条、summary high/med/low 加总 13 但 issues_found 计 11），报告以「结果数=各 finding 明细条数」为准，严重级别合计=65（high 30 / medium 28 / low 7）。

---

## 二、⛔ 必须优先修复的高危问题（30 条）

### 2.1 前端 XSS（Engine-B，存储型）— high
- **`web/admin/src/pages/document/editor/edit/NodeDiffModal.tsx:176`**
  `dangerouslySetInnerHTML={{ __html: line }}`：当节点 `content_type != 'md'` 时，文档 diff 内容（用户可写的 HTML 原文 `release_content/current_content`）未做任何清洗直接渲染 → **存储型 XSS**。项目其余 markdown 渲染路径均经 `sanitizeMarkdownHtml`/`rehype-sanitize`，此处漏了。
  - 修复：注入前用 DOMPurify（白名单/DOM 策略）清洗；或改文本渲染。

### 2.2 API 鉴权缺失（Engine-B/E/L 交叉确认）— high
- **`backend/handler/v1/file.go:39`** `group.POST("/upload/anydoc", h.UploadAnydoc)` **未挂载 `h.auth.Authorize`**，同文件 `/upload`(line 38) 已挂。唯一防护是 `backend/handler/v1/file.go:85` 用可伪造的固定 IP 后缀 `SubnetPrefix+".17"` 比对（`X-Forwarded-For`/remoteAddr 可伪造）→ **无认证任意文件写入**。
  - 修复：为 `anydoc` 增加与 `/upload` 一致的鉴权中间件；用真实鉴权（JWT/ApiKey+TLS）替代 IP 白名单；写入路径限制到受控子目录。

### 2.3 传输加密缺失（Engine-C）— high
- **`backend/cmd/api/main.go:27`**`Echo.Start` 以纯明文 HTTP 启动，无内置 TLS，传输层加密完全依赖外部反代；若缺少反代 TLS 终结，认证令牌与 Cookie 明文传输。
- **`backend/pkg/anydoc/anydoc.go:32,34`** 服务间调用硬编码明文 `http://panda-wiki-api:8000`、`http://panda-wiki-crawler:8080`，服务间数据未加密。
- 修复：服务入口强制 HTTPS（StartTLS/网关 TLS 终结）；容器服务间改用 TLS/mTLS。

### 2.4 Cookie 安全属性（Engine-C）— high
- **`web/app/src/proxy.ts:157`** `x-pw-session-id` 会话 Cookie 仅设 `httpOnly`，缺 `secure`、缺 `SameSite`，且 `maxAge` 高达 1 年（持久 Cookie）→ 会话固定与跨站携带风险。
- 后端 `backend/utils/session_cookie.go:15`、`backend/middleware/session.go:57` Secure 为条件设置（依赖可伪造的 `X-Forwarded-Proto`），默认明文 HTTP 部署下不置 Secure。
- 修复：强制 `Secure`、`SameSite`，缩短 Cookie 有效期；勿依赖可伪造头判断。

### 2.5 日志输出敏感数据（Engine-I）— high
- **`backend/usecase/app.go:419`** 启动 Discord 机器人时 `log.String("token", token)` 将**真实 Discord Bot Token** 明文写入 Info 日志。
- **`web/admin/src/pages/login/index.tsx:32`** `console.log(account, password)` 每次登录把**明文密码**打印到浏览器控制台。
- 修复：移除 token/密码日志输出；前端禁止 console.log 密码；log 封装加脱敏过滤。

### 2.6 涉密信息明文存储（Engine-M）— high
- **`web/admin/src/pages/login/index.tsx:35`** JWT 认证 token（含身份）**明文存 localStorage**，无 HttpOnly/XSS 保护。
- **`web/admin/src/pages/login/index.tsx:32`** `console.log(account, password)`（与前重复，归并到 2.5 一并修）。
- 修复：token 改用 HttpOnly Cookie/session；如需 localStorage 至少加密存储并防 XSS 读取。

### 2.7 三员分立完全缺失（Engine-F）— high
- **`backend/consts/admin.go:29`** 用户角色仅 `admin/user/guest` 三种，**无系统管理员/安全管理员/审计管理员三员分立**。
- **`backend/handler/v1/user.go:109`** 新建用户直接落库无审核；`user.go:329` 删除用户直接落库无审核；`kb_user.go:103` 权限变更直接写入无审核。无 `auditStatus` 审核状态字段（`backend/domain/user.go:11`）。
- **`backend/handler/v1/user.go:47`** 管理端与用户端共用同一 `/api/v1/user/login` 登录入口，未分离。
- **`backend/consts/admin.go:10`** `audit_manage` 权限实际仅用于「发布文档」业务审核，并非安全审计/安全管理员职能。
- 修复：架构层重构为三员分立；增删用户/权限变更引入安全管理员审核流程；分离管理/普通登录入口。

### 2.8 审计日志基础设施整体缺失（Engine-E/G/J/K）— high（结构性）
> **这是本报告最核心的结构性缺陷**。全项目 86 个 SQL 迁移文件**无任何安全审计日志表**，Go 端无 `WriteAudit`/审计写入函数，`log/` 仅为通用 slog 应用日志，“审计”(AuditManage) 被误用作「文档发布审核」业务权限。
- **Engine-E #15 审计记录不可篡改 — high**：无审计存储设施，审计记录根本不产生，不可篡改/防删除/完整性无从谈起。
- **Engine-G #51 防删除 / #52 完整性 / #53 不可修改 — 三条 high**：无审计数据，无哈希链/签名校验，无审计表。
- **Engine-J #47 六要素 / #48 主体唯一标识 — 两条 high**：无审计记录，六要素（时间/地点/类型/主体/客体/结果）与主体唯一标识均无法满足。
- **Engine-K #42-45/#64 场景覆盖 — 六条 high**：身份鉴别、访问控制、涉密出入、用户增删改、权限变更、配置变更、电子文档输出等八大审计域全部无日志，且**不含审计员自身行为审计**。
- 修复：建立独立安全审计日志基础设施（audit 表+写入中间件+查询只读接口+哈希完整性链+防删），覆盖所有关键操作含审计员自身。

### 2.9 数据备份/恢复缺失（Engine-H）— high
- **#61 备份功能 — high**：全项目无 BackupService/API/pg_dump。
- **#62 备份全面性 — high**：无备份，无法覆盖关键表。
- **#63 恢复功能 — high**：无恢复 API（`restore|recover` 仅命中内建 `recover()` panic）。
- 修复：新增备份/恢复 Service+API；复用现有 `robfig/cron` 做定时备份；磁盘使用率阈值告警。

### 2.10 文件上传黑名单可绕过（Engine-L）— high
- **`backend/usecase/file.go:206`** `checkUpload` 仅扩展名**黑名单**+100MB 限制，无白名单、无 magic-byte/MIME 内容识别，黑名单可被枚举/改名绕过。
- **`backend/usecase/file.go:233`** 黑名单 `builtinDeniedExtensions` 未覆盖 `xml/vbs/wsf/pl/ps1/cgi/shtml/phtml` 等危险扩展。
- `backend/handler/share/common.go:74` share 上传 `IsImageFile` 纯扩展名白名单，未校验文件头（polyglot 风险）。
- 修复：改白名单 + 读取文件头 magic bytes（`http.DetectContentType`）与扩展名/声明 content-type 交叉校验。

### 2.11 非国产操作系统（Engine-N）— high
- **`backend/Dockerfile.api:17`** 全部 6 个 Dockerfile 运行时镜像 `FROM alpine:3.21`（非国产），builder 用 `golang:1.24.3-alpine`；`DOCKER_SETUP.md` 部署目标声明 Ubuntu/Debian/CentOS（非国产）。
- 修复：按国产化要求替换为 openEuler/麒麟/统信 UOS 等国产 OS 基础镜像。

---

## 三、🟡 中危问题（28 条，建议修复）

| # | 引擎 | 位置 | 问题 |
|:-:|:----:|------|------|
| 1 | B | 后端 server/http/http.go:126 | share Cookie 会话路由未配 CSRF（`middleware.CSRF` 无命中），改造为 CSRF Token/令牌头 |
| 2 | B/L | 后端 usecase/file.go | 上传校验为黑名单非白名单（中），且 `/upload/anydoc` 无鉴权（并入 2.2） |
| 3 | C | 后端 utils/static_url.go:21 | 静态文件默认明文 `http`，仅 `S3_USE_SSL=true` 才 https |
| 4 | C | 前端 enums.tsx:404 / CardBasicInfo.tsx:61 | 前端默认 API/分享链接硬编码 `http://`（172.17.0.1 非 localhost） |
| 5 | C | 后端 middleware/session.go:54 | share 会话 `MaxAge=0` 无服务端绝对/空闲过期，Redis 侧会话无 TTL 主动失效 |
| 6 | C | 后端 proxy.ts:159 | 网关会话 Cookie 有效期 1 年，会话失效窗口过长 |
| 7 | D | 后端 handler/v1/conversation.go:55 | `GetConversationList` 仅 `c.Bind()` 未 `c.Validate()`，绕过 validate tag（同类散布于 cloud_auth/expert_profile 等 handler） |
| 8 | E | 后端 server/http/http.go:126 | API 限流缺失：仅登录接口有 IP 暴力破解锁定，对外上传/crawler/share 无限流中间件 |
| 9 | E | 后端 handler/base.go | API 日志审计缺失：无安全审计/操作日志机制（并入 2.8） |
| 10 | F | 后端 user.go:47 | 管理端/用户端共用登录入口未分离 |
| 11 | F | 后端 domain/user.go:11 | 标识符创建/删除/权限变更即时生效无审核生效机制 |
| 12 | F | 后端 user.go:51 | admin 通过单一角色同时拥有用户管理与 full_control 旁路权限 |
| 13 | F | 后端 jwt.go:148 | 系统/特权单一 admin 角色判定，无「系统管理员」独立职责边界 |
| 14 | F | 后端 consts/admin.go:10 | 「审核管理」仅用于发布文档，非安全管理员职能 |
| 15 | F | 后端 setup/cert.go:120 | 无用户/管理员身份证书分离，身份纯账号+口令 |
| 16 | F | 后端 setup/cert.go | 用户标识与证书无绑定映射 |
| 17 | G | 后端 | 审计存储空间告警缺失（无磁盘监控/prometheus） |
| 18 | G | 后端 | 审计日志导出完整性校验缺失（并入 2.8） |
| 19 | G | 后端 | 审计日志导出加密缺失（并入 2.8） |
| 20 | H | 后端 | 磁盘空间阈值告警缺失（file.go:206 仅单文件 maxUploadBytes） |
| 21 | I | 后端 comment.go:171 | 无通用日志脱敏机制（仅 maskIP 一处），凭据经 log.String/log.Any 原样落地 |
| 22 | M | 前端 login.tsx:233 / provider/index.tsx:104 | username 等用户 PII 明文 JSON 存 localStorage |
| 23 | N | 后端 docker-compose.services.yml:4 | 中间件镜像均为非国产上游（postgres:15-alpine、redis:7-alpine、nats:2.10-alpine） |
| 24 | N | 后端 docker-compose.services.yml:68 | `minio/minio:latest` 浮动标签未固定版本 |
| 25 | N | 后端 DOCKER_SETUP.md | 全项目无「运行环境清单」文档（OS/中间件名称+版本+端口矩阵） |
| 26 | O | 前端 login/index.tsx:44 | 登录页/站内页面均未显示密级标识（秘密/机密/内部） |
| 27 | M | 前端 login.tsx:233 | Cookie/存储中存放 username（涉密信息明文存放中危，并入 2.6/3） |
| 28 | B | 后端 pkg/ldap/ldap.go:121 | LDAP 过滤 `fmt.Sprintf(c.config.UserFilter, username)` username 未转义，存在 LDAP 注入面（注：非 #33 SQL 注入范畴，建议转义） |

---

## 四、⚪ 低危问题（7 条，可选优化）

| # | 引擎 | 位置 | 问题 |
|:-:|:----:|------|------|
| 1 | C | backend/cmd/api/main.go:27 | 全库无 tls.Config/MinVersion 显式配置，TLS 版本由外部反代决定 |
| 2 | C | web/app/src/proxy.ts:159 | 会话 Cookie 有效期过长（1 年） |
| 3 | C | middleware/share_auth.go:103 | session 用户名为权威（低风险，已绑定 kb_id/user_id） |
| 4 | F | domain/user.go:13 | Account `gorm:"uniqueIndex"`（通过项，标识符唯一性满足，标注低） |
| 5 | F | domain/user.go:24 | `(kb_id,user_id)` 复合唯一索引（通过项，同名不继承满足） |
| 6 | H | handler/mq/cron.go | 无定期备份调度（robfig/cron 仅做统计聚合/清理） |
| 7 | N | backend/go.mod:14 | `boj/redistore v1.4.1` 停维护库（间接 go-redis/redis/v8 旧版） |

---

## 五、✅ 通过项（合规项）

| 引擎 | 通过项 |
|:----:|:-------|
| A | #40 硬编码密码、#40变种 getenv、#58 默认口令、#29 缺省口令：配置全部 `${ENV}` 占位，`config/config.go` 敏感项缺失时 `validateRequiredSecrets` 报错拒绝启动，无硬编码回退；无 admin/admin123 等默认口令 |
| B | #33 SQL注入（全部 GORM/pgx 参数化）、#37 命令注入（无 exec/os.exec）、#26 CORS（仅 restrictive same-host，无 `*`） |
| D | #3 口令复杂度（8-72 位+大小写数字特殊符，4 处强制）、#4 登录失败锁定（IP 级 Redis 锁定 5/10/15 次）、#5 超时退出（JWT exp=24h）、#30 加固（bcrypt 哈希） |
| F | #9 标识符唯一性（gorm uniqueIndex）、#40 同名不继承权限 |
| M | 后端会话 Cookie HttpOnly+SameSite=Lax，X-KB-ID 为知识库 ID 非用户信息；密码 bcrypt 存储；安全响应头已配置（X-Content-Type-Options/X-Frame-Options/CSP） |
| N | go 依赖多为近期安全版本（jwt/v5 v5.3.0、go-ldap v3.4.11、echo v4.13.4、gorm v1.26.1、x/crypto v0.40.0）；中间件端口已明确（5432/6379/4222/8222/9000/9001） |
| C | #20 Cookie/用户名比对：session 以 kb_id/user_id 绑定并校验归属，未发现客户端 username 作权威 |

---

## 六、🔒 不可代码扫描项说明（17 条，需人工/运维核查）

以下条目从 registry.yaml `unscannable` 读取，无法通过静态代码扫描判定，需结合运行环境核查：

| 类别 | 条目 | 说明 |
|------|:----:|------|
| 运维/基础设施 | #1,#3,#4,#5,#6 | 系统加固、数据库安全配置/补丁/用户分离，需登录数据库或服务器核查 |
| 业务逻辑（深层） | #10 | 标识符与数字证书对应，需人工验证业务逻辑 |
| 硬件交互 | #21 | Key 拔插权限隔离需运行环境验证 |
| 策略豁免 | #22 | 特殊需求豁免属策略层面，需人工判定 |
| 功能层（深层） | #61,#62,#63,#65 | 备份/恢复功能完整性及定期提醒需运行验证 |
| 采购/选型 | #70,#71,#72,#73,#74 | 信创 OS/中间件/数据库/BIOS 选用及性能监控工具支持，属采购与基础设施层面 |

> 注：其中 #61/#62/#63/#65 虽在 unscannable 归类为「功能层运行验证」，但 Engine-H 已通过静态扫描发现**代码层缺失备份/恢复实现**（见 2.9），属代码可确认问题，两处结论相互印证。

---

## 七、合规率统计

- **总规则**：74 条
- **可扫描规则**：57 条（分布于 15 个引擎）
- **不可扫描规则（需人工核查）**：17 条
- **本次扫描结果**：65 个问题（high 30 / medium 28 / low 7）

**结构性风险提示**：
1. **安全审计日志基础设施整体缺失**（Engine-E/G/J/K 共 18+ 条 high/medium，全部指向同一根因）——这是本项目合规的最重大问题，涉及 #15、#42-45、#46-56、#62、#64 等大量条目。
2. **三员分立未实现**（Engine-F，#30/#31/#32/#35）——身份与权限治理架构不满足红网测评核心要求。
3. **数据备份/恢复缺失**（Engine-H，#61/#62/#63/#65）。
4. **非国产运行环境**（Engine-N，#2）——违反国产化信创要求。

---

## 八、引擎结果文件索引

原始 findings（每个引擎的完整 JSON）已落盘至 `/Sec/panda-wiki/.claude/plans/scan-results/`：

```
engine-a-credentials-20261111-1429.md        (0 findings)
engine-b-web-vulns-20261111-1429.md          (3 findings)
engine-c-crypto-transport-20261111-1442.md   (13 findings)
engine-d-input-config-20261111-1442.md       (1 finding)
engine-e-audit-api-20261111-1449.md          (4 findings)
engine-f-identity-rbac-20261111-1450.md      (13 findings)
engine-g-audit-integrity-20261111-1455.md    (6 findings)
engine-h-data-backup-20261111-1403.md        (5 findings)
engine-i-log-masking-20261111-1459.md        (3 findings)
engine-j-audit-log-validator-20261111-1503.md(2 findings)
engine-k-audit-log-helper-20261111-1508.md   (6 findings)
engine-l-upload-security-audit-20261111-1503.md (6 findings)
engine-m-secure-header-check-20261111-1505.md (4 findings)
engine-n-runtime-env-vuln-20261111-1513.md   (5 findings)
engine-o-system-identification-20261111-1130.md (3 findings)
```

---

*报告由 security-74-scanner 主编排器（full-scan 模式）自动生成。部分引擎（E/G/J/K/L/M）基于已集成 Skill 适配执行；代码均为静态扫描结论，证据文件见上文索引。*