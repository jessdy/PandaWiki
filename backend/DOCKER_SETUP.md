# Docker 服务安装指南

本指南详细说明如何在 Linux 服务器上使用 Docker 安装 PandaWiki 所需的所有服务。

## 前置要求

- Linux 服务器（Ubuntu/Debian/CentOS 等）
- Docker 20.x 或更高版本
- Docker Compose 2.x 或更高版本（如果使用 Compose 方式）

### 安装 Docker

如果服务器上还没有安装 Docker：

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo systemctl enable docker
sudo systemctl start docker

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

## 方式一：使用 Docker Compose（推荐）

### 1. 准备配置文件

在 Linux 服务器上创建服务目录：

```bash
mkdir -p ~/panda-wiki-services
cd ~/panda-wiki-services
```

从项目复制 `docker-compose.services.yml` 到服务器，或直接创建：

```bash
# 从 Mac 复制（在 Mac 上执行）
scp backend/docker-compose.services.yml user@YOUR_LINUX_IP:~/panda-wiki-services/docker-compose.yml

# 或者在 Linux 服务器上直接创建
nano docker-compose.yml
# 复制 docker-compose.services.yml 的内容
```

### 2. 启动服务

```bash
cd ~/panda-wiki-services
docker compose up -d
```

### 3. 验证服务

```bash
# 查看所有服务状态
docker compose ps

# 查看服务日志
docker compose logs -f

# 测试 PostgreSQL
docker exec panda-wiki-postgres pg_isready -U panda-wiki

# 测试 Redis
docker exec panda-wiki-redis redis-cli -a panda-wiki-redis-password ping

# 测试 NATS
curl http://localhost:8222/healthz
```

### 4. 管理服务

```bash
# 停止所有服务
docker compose down

# 停止并删除数据卷（谨慎使用！）
docker compose down -v

# 重启服务
docker compose restart

# 查看日志
docker compose logs -f [service_name]
```

## 方式二：单独使用 Docker 命令

### PostgreSQL

```bash
docker run -d \
  --name panda-wiki-postgres \
  --restart unless-stopped \
  -e POSTGRES_USER=panda-wiki \
  -e POSTGRES_PASSWORD=panda-wiki-secret \
  -e POSTGRES_DB=panda-wiki \
  -e TZ=Asia/Shanghai \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:15-alpine
```

**验证:**
```bash
docker exec panda-wiki-postgres pg_isready -U panda-wiki
```

### Redis

```bash
docker run -d \
  --name panda-wiki-redis \
  --restart unless-stopped \
  -p 6379:6379 \
  -v redis_data:/data \
  redis:7-alpine \
  redis-server --requirepass panda-wiki-redis-password
```

**验证:**
```bash
docker exec panda-wiki-redis redis-cli -a panda-wiki-redis-password ping
```

### NATS

```bash
docker run -d \
  --name panda-wiki-nats \
  --restart unless-stopped \
  -p 4222:4222 \
  -p 8222:8222 \
  nats:2.10-alpine \
  -js -m 8222 --user panda-wiki --pass panda-wiki-nats-password
```

**验证:**
```bash
curl http://localhost:8222/healthz
```

### MinIO

```bash
docker run -d \
  --name panda-wiki-minio \
  --restart unless-stopped \
  -e MINIO_ROOT_USER=s3panda-wiki \
  -e MINIO_ROOT_PASSWORD=panda-wiki-minio-secret \
  -p 9000:9000 \
  -p 9001:9001 \
  -v minio_data:/data \
  minio/minio:latest \
  server /data --console-address ":9001"
```

**访问控制台:**
- 浏览器访问: `http://YOUR_LINUX_IP:9001`
- 用户名: `s3panda-wiki`
- 密码: `panda-wiki-minio-secret`

**验证:**
```bash
curl http://localhost:9000/minio/health/live
```

### RAG 服务 (CT-RAG) - 可选

```bash
# 注意：
# 1. RAG 服务是可选的，不是必需的，可以先不安装
# 2. 如果镜像不存在或需要认证，请使用其他可用的 RAG 服务镜像
# 3. 或者联系项目维护者获取正确的镜像地址
# 4. 在 docker-compose.yml 中，RAG 服务默认是注释掉的

docker run -d \
  --name panda-wiki-ct-rag \
  --restart unless-stopped \
  -e API_KEY=sk-1234567890 \
  -p 5050:5050 \
  YOUR_RAG_IMAGE:latest
```

> **重要提示**: 
> - RAG 服务是可选的，如果镜像拉取失败，可以暂时跳过
> - 在 `docker-compose.yml` 中，`ct-rag` 服务默认已被注释
> - 如果需要使用 RAG 功能，请替换为实际可用的镜像

## 配置防火墙

确保开放必要的端口：

```bash
# Ubuntu/Debian (UFW)
sudo ufw allow 5432/tcp comment 'PostgreSQL'
sudo ufw allow 6379/tcp comment 'Redis'
sudo ufw allow 4222/tcp comment 'NATS'
sudo ufw allow 9000/tcp comment 'MinIO API'
sudo ufw allow 9001/tcp comment 'MinIO Console'
sudo ufw allow 5050/tcp comment 'RAG Service'

# 或者只允许特定 IP 访问（更安全）
sudo ufw allow from YOUR_MAC_IP to any port 5432
sudo ufw allow from YOUR_MAC_IP to any port 6379
sudo ufw allow from YOUR_MAC_IP to any port 4222
sudo ufw allow from YOUR_MAC_IP to any port 9000
sudo ufw allow from YOUR_MAC_IP to any port 5050

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-port=5432/tcp
sudo firewall-cmd --permanent --add-port=6379/tcp
sudo firewall-cmd --permanent --add-port=4222/tcp
sudo firewall-cmd --permanent --add-port=9000/tcp
sudo firewall-cmd --permanent --add-port=5050/tcp
sudo firewall-cmd --reload
```

## 服务连接信息

使用默认配置，服务的连接信息如下：

| 服务 | 地址 | 用户名 | 密码 | 说明 |
|------|------|--------|------|------|
| PostgreSQL | `YOUR_LINUX_IP:5432` | `panda-wiki` | `panda-wiki-secret` | 数据库服务 |
| Redis | `YOUR_LINUX_IP:6379` | - | `panda-wiki-redis-password` | 缓存服务 |
| NATS | `nats://YOUR_LINUX_IP:4222` | `panda-wiki` | `panda-wiki-nats-password` | 消息队列 |
| MinIO | `YOUR_LINUX_IP:9000` | `s3panda-wiki` | `panda-wiki-minio-secret` | 对象存储 |
| RAG | `http://YOUR_LINUX_IP:5050` | - | `sk-1234567890` (API Key) | RAG 服务 |

## 数据持久化

所有服务的数据都存储在 Docker volumes 中，即使容器删除，数据也不会丢失：

```bash
# 查看 volumes
docker volume ls | grep panda-wiki

# 备份数据
docker run --rm -v postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_backup.tar.gz /data

# 恢复数据
docker run --rm -v postgres_data:/data -v $(pwd):/backup alpine tar xzf /backup/postgres_backup.tar.gz -C /
```

## 修改配置

### 修改密码

1. **PostgreSQL**: 修改环境变量 `POSTGRES_PASSWORD`
2. **Redis**: 修改启动命令中的 `--requirepass` 参数
3. **NATS**: 修改启动命令中的 `--pass` 参数
4. **MinIO**: 修改环境变量 `MINIO_ROOT_PASSWORD`

修改后需要重新创建容器：

```bash
# 使用 Docker Compose
docker compose down
# 修改 docker-compose.yml
docker compose up -d

# 或单独修改
docker stop panda-wiki-postgres
docker rm panda-wiki-postgres
# 重新运行 docker run 命令（使用新密码）
```

### 修改端口

如果需要修改端口映射，编辑 `docker-compose.yml` 或修改 `docker run` 命令中的 `-p` 参数。

## 故障排查

### 服务无法启动

```bash
# 查看容器日志
docker logs panda-wiki-postgres
docker logs panda-wiki-redis
docker logs panda-wiki-nats

# 查看容器状态
docker ps -a | grep panda-wiki
```

### 端口被占用

```bash
# 检查端口占用
sudo netstat -tulpn | grep :5432
sudo lsof -i :5432

# 停止占用端口的服务或修改端口映射
```

### 连接被拒绝

1. 检查防火墙设置
2. 检查服务是否正在运行: `docker ps`
3. 检查端口是否正确映射: `docker port panda-wiki-postgres`
4. 检查服务日志: `docker logs panda-wiki-postgres`

### RAG 服务镜像拉取失败

如果遇到 `ct-rag` 服务镜像拉取失败（如 `unauthorized: incorrect username or password`），这是正常的：

1. **RAG 服务是可选的**：可以先跳过，只启动必需的服务
2. **解决方案**：
   ```bash
   # 在 docker-compose.yml 中注释掉 ct-rag 服务
   # 然后重新启动
   docker compose down
   docker compose up -d
   ```
3. **验证其他服务**：
   ```bash
   # 检查已启动的服务
   docker compose ps
   
   # 应该能看到 postgres, redis, nats, minio 正常运行
   # ct-rag 服务可以稍后单独配置
   ```
4. **如果需要 RAG 服务**：
   - 联系项目维护者获取正确的镜像地址
   - 或使用其他可用的 RAG 服务镜像
   - 或使用外部 RAG 服务 API

## 生产环境建议

1. **修改默认密码**: 所有服务的默认密码都应该修改为强密码
2. **使用 TLS/SSL**: 生产环境建议启用 TLS 加密连接
3. **限制网络访问**: 使用防火墙规则限制只有必要的 IP 可以访问
4. **定期备份**: 设置定期备份数据库和重要数据
5. **监控和日志**: 配置监控和日志收集系统
6. **资源限制**: 为容器设置 CPU 和内存限制

## 清理

如果需要完全清理所有服务和数据：

```bash
# 停止并删除所有容器和数据卷
docker compose down -v

# 或单独删除
docker stop panda-wiki-postgres panda-wiki-redis panda-wiki-nats panda-wiki-minio panda-wiki-ct-rag
docker rm panda-wiki-postgres panda-wiki-redis panda-wiki-nats panda-wiki-minio panda-wiki-ct-rag
docker volume rm postgres_data redis_data minio_data
```

