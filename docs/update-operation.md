# PandaWiki 现场部署说明（含雳鉴 IAST 插桩开关）

> 从旧版本升级需要改哪些镜像、compose、.env，见 **[升级说明-20261112](./升级说明-20261112.md)**。

从本版本起只有**一个版本**的 api / consumer 镜像（`panda-wiki-api:prod`、`panda-wiki-consumer:prod`），不再单独提供 `prod-iast` 镜像和 `docker-compose.iast.yml`。

| 项 | 说明 |
| --- | --- |
| 镜像内容 | 同时包含普通版与雳鉴主动插桩版二进制（`*.iast`） |
| 是否插桩 | 由开关文件 `data/conf/iast/iast.conf` 中的 `IAST_ENABLED=true/false` 决定，无需换镜像 |
| 默认 | 没有开关文件或 `IAST_ENABLED=false` = 普通版，行为与以前的 `prod` 完全一致 |
| 架构限制 | 插桩版只在 **linux/arm64** 镜像中提供（雳鉴 `iast_go` 只有 arm64 版）；x86 镜像即使开启也以普通版启动，并在日志中提示 |

---

## 插桩开关文件

主 `docker-compose.yml` 中 api / consumer 已把宿主机 `./data/conf/iast` 只读挂到容器 `/iast-config`，并分别设置 `IAST_AGENT_TAG=panda-wiki-api` / `panda-wiki-consumer`。

开关文件 `data/conf/iast/iast.conf`（模板见源码 `docs/iast/iast.conf.example`）：

```bash
# true 开启插桩 / false 关闭（普通版）
IAST_ENABLED=false

# 雳鉴 Name Server 地址（不可用 127.0.0.1 / localhost）
IAST_SERVER_IP=改成雳鉴IP
IAST_SERVER_PORT=8888

# https 端口、关联版本号，一般不用改
IAST_XSCHEMA_PORTS=443,8443
IAST_PROJECT_VERSION=v1.0
```

api 与 consumer 共用这一个文件，agent_tag 自动区分。

### 开启 / 关闭 / 改地址

编辑 `data/conf/iast/iast.conf` 后重建 api、consumer（只改文件不重建不会生效）：

```bash
cd /opt/panda-wiki
vi data/conf/iast/iast.conf
docker compose up -d --no-build --pull never --force-recreate api consumer
docker logs panda-wiki-api 2>&1 | head -1
docker logs panda-wiki-consumer 2>&1 | head -1
```

日志第一行：

| 日志 | 含义 |
| --- | --- |
| `IAST: 开关已开启 (server=IP:端口, tag=...)，以插桩版启动` | 插桩已生效 |
| `IAST: 开关为关闭（IAST_ENABLED=false），以普通版启动` | 已关闭 |
| `ERROR: 已开启 IAST 但 IAST_SERVER_IP 无效或仍为占位符…以普通版启动` | 开关为 true 但 IP 没填对；为保证业务可用会以普通版启动，改好 IP 再重建 |
| `WARN: 已开启 IAST，但镜像不含插桩版二进制…` | 镜像不是 arm64 插桩构建 |

开启后确认：雳鉴【插桩 Agent】在线（tag 为 `panda-wiki-api`、`panda-wiki-consumer`）；前台用真实 IP/域名可访问；`tgt` / 授权链接可登录。

### 高级：完整 agent toml

如需自定义雳鉴 agent 的其他参数，可在 `data/conf/iast/` 放 `api.toml`、`consumer.toml`（模板 `docs/iast/go-moresec-agent.*.toml.example`）。开关为 true 时优先使用这两个文件，`iast.conf` 中只有 `IAST_ENABLED` 生效。

> 必须是**文件**。若出现名为 `iast.conf`/`api.toml` 的目录，删掉后重建为文件。

## 从旧的 prod-iast 部署迁移

1. `docker load` 新的 `prod` 镜像，`.env` 中删除 `API_IMAGE_TAG=prod-iast`、`CONSUMER_IMAGE_TAG=prod-iast`（或改为 `prod`）。
2. 换上新版主 `docker-compose.yml`，不要再带 `-f docker-compose.iast.yml`。
3. 原 `data/conf/iast/api.toml`、`consumer.toml` 可原样保留继续插桩；建议新增 `iast.conf` 用 `IAST_ENABLED` 统一开关。
4. `docker compose up -d --no-build --pull never --force-recreate caddy api consumer`

---

## 构建镜像（开发侧）

插桩版需要雳鉴编译器 `go-active-linux-arm64`（不入 git），构建前放到 `backend/iast/`：

```bash
cp arm/iast/go-active-linux-arm64 backend/iast/
docker build --platform linux/arm64 -f backend/Dockerfile.api      -t panda-wiki-api:prod      backend
docker build --platform linux/arm64 -f backend/Dockerfile.consumer -t panda-wiki-consumer:prod backend
```

- 缺少编译器或构建 amd64 时，`iast-builder` 阶段跳过，镜像只含普通版，其余不受影响。
- 在 x86 机器上构建 arm64 插桩版需要 qemu/binfmt（编译器本身是 arm64 程序），推荐直接在 arm64 机器上构建。
- 校验镜像是否含插桩版：`docker run --rm --entrypoint ls panda-wiki-api:prod /app`，应看到 `panda-wiki-api.iast`。

---

## 若出现 failed to sync kb access settings to caddy

api 只在**启动时**（以及后台修改知识库访问设置时）把站点配置推给 Caddy。先看具体原因：

```bash
docker logs panda-wiki-api 2>&1 | grep -i caddy | tail -3
ls -la data/caddy/run/caddy-admin.sock
```

`docker logs` 会带出容器的历史日志，先看时间戳是不是最近一次重启之后的（日志是 UTC 时间，比北京时间慢 8 小时）。

**情况一：`connect: permission denied`，socket 显示为 `s-w-------`**

api 以非 root（uid 10001）运行，主 compose 中 caddy 需带 `CADDY_ADMIN=unix//var/run/caddy/caddy-admin.sock|0666`：

```bash
grep -n CADDY_ADMIN docker-compose.yml      # 必须能看到 |0666
docker compose up -d --no-build --pull never --force-recreate caddy api consumer
ls -la data/caddy/run/caddy-admin.sock      # 应为 srw-rw-rw-
```

**情况二：`failed to update caddy config`，后面带 Caddy 返回的错误**

常见原因是知识库配置的访问端口被宿主机其他程序占用（caddy 使用 host 网络），或证书无效。查看 `docker logs --tail 50 panda-wiki-caddy`，修正后 `docker restart panda-wiki-api`。

**情况三：`Caddy socket not found`**

确认 `docker ps | grep caddy` 在运行后 `docker restart panda-wiki-api`。

---

## 常见问题

| 现象 | 处理 |
| --- | --- |
| 开启后日志仍是普通版 | 看日志第一行的 ERROR/WARN 提示，多为 IP 未填或仍为占位符 |
| 已配置但日志提示“镜像不含插桩版二进制” | 镜像是 x86 版或构建时缺少 `iast_go`，按“构建镜像”重新构建 arm64 镜像 |
| 出现 `api.toml` 目录 | 删除后重建为文件 |
| Agent 不上线 | 查 IP/端口与防火墙；业务勿走 localhost |
| 去构建或拉取 | 命令带 `--no-build --pull never`，并先 `docker load` |

## 更换 JWT_SECRET

`JWT_SECRET` 除签发登录令牌外，在设置 `DATA_ENCRYPTION_KEY` 之前还用于加密敏感字段和审计哈希链。更换时把旧值写入 `PREVIOUS_JWT_SECRET`（只用于解密/校验旧数据，多个用逗号分隔），否则旧数据无法解密：

```bash
cd /opt/panda-wiki
OLD=$(grep '^JWT_SECRET=' .env | cut -d= -f2-)
cp .env .env.bak-$(date +%Y%m%d%H%M)
sed -i "s|^JWT_SECRET=.*|JWT_SECRET=$(openssl rand -hex 32)|" .env
grep -q '^PREVIOUS_JWT_SECRET=' .env || echo "PREVIOUS_JWT_SECRET=$OLD" >> .env
docker compose up -d --no-build --pull never --force-recreate api consumer
```

更换后所有用户需重新登录。

## 安全加固补充（审计 / 三员 / 备份 / 国产 OS）

`.env` 需具备（api 与 consumer 保持一致）：

```bash
S3_ACCESS_KEY=...             # 必填，须与 MinIO 实际用户名一致
SECURITY_ADMIN_PASSWORD=...   # 安全管理员 secadmin
AUDIT_ADMIN_PASSWORD=...      # 审计管理员 auditadmin
DATA_ENCRYPTION_KEY=...       # 字段加密与审计哈希链；建议独立于 JWT_SECRET
BACKUP_DIR=/app/data/backups  # 备份目录（compose 已挂载 ./data/backups）
```

行为变化：

1. 用户创建/删除、知识库权限变更、用户组变更改为「提交审批」，由 `secadmin` 通过后生效。
2. `auditadmin` 只能访问审计日志页：查询、哈希链校验、加密导出、存储告警。
3. MinIO 桶为私有，`/static-file` 经 API 鉴权读取。
4. api/consumer 运行镜像基于 openEuler 24.03 LTS。
5. 后台可配置密级标识（默认关闭）；侧边栏展示构建时注入的 `VITE_APP_VERSION`。
