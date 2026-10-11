# 正式服务器恢复

把测试服务器上的程序镜像和业务数据恢复到一台新的 Linux 服务器。正式机不重新初始化业务数据，只导入测试机导出的库和文件。

在仓库根目录执行下面的命令。正式机使用已经改好的 `docker-compose.yml`：四个自研服务使用 `:prod` 镜像，并且没有测试机上的前端目录挂载。

## 要带到正式机的文件

放到同一个目录，例如 `/opt/panda-wiki`：

| 文件 | 来源 |
| --- | --- |
| `docker-compose.yml` | 本仓库里改好的正式环境配置 |
| `.env` | 测试机部署目录原文件，不要改密码 |
| `panda-wiki-images.tar.gz` | 测试机 `docker save` 的镜像包 |
| `panda-wiki.dump` | 测试机 Postgres 逻辑备份 |
| `panda-wiki-files.tar.gz` | 测试机 `data/` 里的 MinIO、Qdrant、raglite、配置和证书 |

`.env` 里的 `POSTGRES_PASSWORD`、`S3_SECRET_KEY`、`QDRANT_API_KEY`、`JWT_SECRET`、`NATS_PASSWORD`、`REDIS_PASSWORD` 必须与测试机相同。这些值已经写进数据库和对象存储，换一套会导致服务连不上原数据。

Redis 和 NATS 是缓存和队列，不用迁移。

## 测试机上生成迁移包

在测试机仓库目录执行。先更新代码并构建四个自研镜像：

```bash
git pull
cd web && npm run build && cd ..

docker build -f backend/Dockerfile.api -t panda-wiki-api:prod backend
docker build -f backend/Dockerfile.consumer -t panda-wiki-consumer:prod backend
docker build -f web/app/Dockerfile -t panda-wiki-app:prod web/app
docker build -f web/admin/Dockerfile -t panda-wiki-nginx:prod web/admin
```

导出镜像。Postgres、Redis、MinIO、NATS、Qdrant、Caddy、crawler、raglite 使用测试机上已经在运行的镜像：

```bash
docker save \
  panda-wiki-api:prod \
  panda-wiki-consumer:prod \
  panda-wiki-app:prod \
  panda-wiki-nginx:prod \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/postgres-zhparser:17.6-bookworm \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/panda-wiki-redis:7.4.2-alpine \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/panda-wiki-minio:RELEASE.2025-04-22T22-12-26Z-cpuv1 \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/panda-wiki-nats:2.11.3-alpine \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/panda-wiki-qdrant:v1.14.1 \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/panda-wiki-caddy:2.10-alpine \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/anydoc:v0.7.1 \
  chaitin-registry.cn-hangzhou.cr.aliyuncs.com/chaitin/raglite:v2.8.5 \
  | gzip > panda-wiki-images.tar.gz
```

停写后导出数据：

```bash
docker compose stop api consumer raglite crawler

docker exec panda-wiki-postgres pg_dump -U panda-wiki -d panda-wiki -Fc -f /tmp/panda-wiki.dump
docker cp panda-wiki-postgres:/tmp/panda-wiki.dump ./panda-wiki.dump

tar -C data -czf panda-wiki-files.tar.gz minio qdrant raglite conf nginx/ssl caddy
```

导出完成后，在测试机执行下面的命令恢复测试环境：

```bash
docker compose start api consumer raglite crawler
```

## 正式机恢复

系统需要 Docker 20 以上，并且能执行 `docker compose`。把上面的五个文件放进部署目录后执行。

### 1. 载入镜像并解压文件

```bash
gunzip -c panda-wiki-images.tar.gz | docker load
mkdir -p data
tar -C data -xzf panda-wiki-files.tar.gz
```

此时还不要启动全部服务。`data/postgres` 必须不存在，Postgres 要在空数据目录里完成第一次初始化，然后再导入备份。

### 2. 初始化空库并导入

```bash
docker compose up -d postgres
docker compose exec postgres pg_isready -U panda-wiki -d panda-wiki

docker cp panda-wiki.dump panda-wiki-postgres:/tmp/panda-wiki.dump
docker exec panda-wiki-postgres pg_restore -U panda-wiki -d panda-wiki --clean --if-exists --no-owner /tmp/panda-wiki.dump
```

`pg_isready` 输出 `accepting connections` 后再执行 `pg_restore`。`pg_restore` 可能提示部分对象已存在，业务表里能查到用户和文档即可。

确认导入结果：

```bash
docker exec panda-wiki-postgres psql -U panda-wiki -d panda-wiki -c '\dt'
```

### 3. 启动全部服务

```bash
docker compose up -d --no-build --pull never
docker compose ps
```

不要执行 `docker compose down -v`，该命令会删除 `data/` 里的数据。

## 核对

- `docker compose ps` 中各容器为运行状态。
- 用测试环境的前台账号登录，文档、图片和附件能打开。
- 后台能看到原来的用户和用户组。
- 搜索或问答能返回迁移前的文档。问答为空时，先确认 `data/qdrant` 已解压，而不是重新导入 Postgres。

后台端口是 `.env` 中的 `ADMIN_PORT`。前台由 Caddy 以主机网络对外提供，证书和站点配置来自解压后的 `data/caddy`。

## 导入失败时重来

只删除正式机上新建的数据库目录，保留已经解压的 MinIO、Qdrant 和其他文件：

```bash
docker compose stop postgres
rm -rf data/postgres
docker compose up -d postgres
```

等 `pg_isready` 通过后，重新执行上面的 `docker cp` 和 `pg_restore`。
