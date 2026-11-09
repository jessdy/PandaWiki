# PandaWiki 现场部署说明（ARM 插桩版）

面向**无编译环境**的现场机器。插桩镜像与交付包已在 **252 服务器**打好，现场只做加载镜像、填配置、启动。

| 项 | 说明 |
| --- | --- |
| 目标架构 | **仅 linux/arm64** |
| 本包镜像 | **仅** `panda-wiki-api:prod-iast`、`panda-wiki-consumer:prod-iast` |
| 不替换 | `app` / `nginx` 及 Postgres、Redis、MinIO、NATS、Qdrant、raglite 等沿用现场已有 |
| 雳鉴地址 | 现场填入宿主机 toml，改完 recreate 即可 |

---

## 交付包内容（来自 252）

解压后应包含：

```text
panda-wiki-iast-deploy/
├── panda-wiki-iast-arm64.tar.gz           # api / consumer 插桩镜像
├── docker-compose.iast.yml
├── go-moresec-agent.api.toml.example
├── go-moresec-agent.consumer.toml.example
└── 更新操作说明.md                         # 本文
```

252 上路径示例（以实际拷贝为准）：

`/root/projects/PandaWiki-Back/panda-wiki-iast-deploy.tar.gz`

---

## 现场检查

1. `uname -m` 为 `aarch64` / `arm64`。
2. Docker ≥ 20，可用 `docker compose`。
3. 部署目录已有主 `docker-compose.yml`、`.env`、`data/`，整套服务曾正常运行。
4. 可访问即将填写的雳鉴 Name Server；业务访问勿用 `localhost`。

可选备份：

```bash
docker tag panda-wiki-api:prod panda-wiki-api:prod-bak-$(date +%Y%m%d) 2>/dev/null || true
docker tag panda-wiki-consumer:prod panda-wiki-consumer:prod-bak-$(date +%Y%m%d) 2>/dev/null || true
```

---

## 现场部署步骤

假设：

- 部署目录：`/opt/panda-wiki`
- 交付包解压在：`/opt/packages/panda-wiki-iast-deploy`

### 1. 解压并加载镜像

```bash
mkdir -p /opt/packages
cd /opt/packages
tar -xzf panda-wiki-iast-deploy.tar.gz
gunzip -c panda-wiki-iast-deploy/panda-wiki-iast-arm64.tar.gz | docker load
docker images | grep prod-iast
```

应看到：

- `panda-wiki-api:prod-iast`
- `panda-wiki-consumer:prod-iast`

### 2. 填写雳鉴 IP / 端口（映射配置）

```bash
cd /opt/panda-wiki
mkdir -p data/conf/iast

cp /opt/packages/panda-wiki-iast-deploy/go-moresec-agent.api.toml.example \
  data/conf/iast/api.toml
cp /opt/packages/panda-wiki-iast-deploy/go-moresec-agent.consumer.toml.example \
  data/conf/iast/consumer.toml

vi data/conf/iast/api.toml
vi data/conf/iast/consumer.toml
```

两文件都改 `IP`、`Port`（禁止 `127.0.0.1` / `localhost`）：

```toml
[nameserver]
IP="192.168.1.100"
Port=8888
```

`api.toml` 的 `agent_tag` 为 `panda-wiki-api`，`consumer.toml` 为 `panda-wiki-consumer`。

> 必须是**文件**。若出现名为 `api.toml` 的目录，删掉后按上面重建。

### 3. .env 切到插桩镜像

```bash
API_IMAGE_TAG=prod-iast
CONSUMER_IMAGE_TAG=prod-iast
```

有映射 toml 时不必再填 `IAST_SERVER_IP`。

```bash
set -a && source .env && set +a
```

### 4. 启动（recreate）

```bash
cp -n /opt/packages/panda-wiki-iast-deploy/docker-compose.iast.yml ./

docker compose -f docker-compose.yml -f docker-compose.iast.yml \
  up -d --no-build --pull never --force-recreate api consumer
```

### 5. 验证

```bash
docker compose -f docker-compose.yml -f docker-compose.iast.yml ps api consumer
docker logs --tail 80 panda-wiki-api
docker logs --tail 50 panda-wiki-consumer
```

日志应含：

```text
IAST agent config from mounted file: /iast-config/go-moresec-agent.toml -> ... (IP=...)
```

再确认：雳鉴【插桩 Agent】在线；前台用真实 IP/域名可访问；`tgt` / 授权链接可登录。

---

## 只改雳鉴地址

```bash
vi data/conf/iast/api.toml
vi data/conf/iast/consumer.toml
docker compose -f docker-compose.yml -f docker-compose.iast.yml \
  up -d --no-build --pull never --force-recreate api consumer
```

---

## 关闭插桩

本机需仍有 `:prod` 镜像：

```bash
# .env
API_IMAGE_TAG=prod
CONSUMER_IMAGE_TAG=prod

set -a && source .env && set +a
docker compose -f docker-compose.yml \
  up -d --no-build --pull never --force-recreate api consumer
```

不要带 `-f docker-compose.iast.yml`。

---

## 常见问题

| 现象 | 处理 |
| --- | --- |
| `exec format error` | 机器不是 arm64，或镜像不对 |
| 配置仍为占位符 / 启动失败 | 把 toml 里 IP 改成真实地址 |
| 出现 `api.toml` 目录 | 删除后重建为文件 |
| Agent 不上线 | 查 IP/端口与防火墙；业务勿走 localhost |
| 去构建或拉取 | 命令带 `--no-build --pull never`，并先 `docker load` |
