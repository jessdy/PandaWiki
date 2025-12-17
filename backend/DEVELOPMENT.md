# 开发环境配置指南

本指南帮助你在 Mac 开发环境中运行 PandaWiki，同时连接到 Linux 服务器上的数据库等服务。

## 前置要求

- Go 1.24.3 或更高版本
- 已安装并运行在 Linux 服务器上的服务：
  - PostgreSQL
  - Redis
  - NATS
  - RAG 服务（可选）
  - MinIO/S3（可选）

## 在 Linux 服务器上使用 Docker 安装服务

### 方式一：使用 Docker Compose（推荐）

这是最简单的方式，一键启动所有服务：

1. **在 Linux 服务器上创建服务目录**

```bash
# 在 Linux 服务器上执行
mkdir -p ~/panda-wiki-services
cd ~/panda-wiki-services
```

2. **创建 docker-compose.yml 文件**

将项目中的 `backend/docker-compose.services.yml` 复制到服务器，或创建新文件：

```bash
# 从 Mac 复制到 Linux 服务器
scp backend/docker-compose.services.yml user@your-linux-server:~/panda-wiki-services/docker-compose.yml
```

3. **启动所有服务**

```bash
# 在 Linux 服务器上执行
cd ~/panda-wiki-services
docker compose up -d
```

> **注意**: 
> - 如果遇到 `ct-rag` 服务拉取失败，这是正常的，因为 RAG 服务是可选的
> - RAG 服务镜像可能需要特殊权限或使用其他镜像
> - 可以先注释掉 `docker-compose.yml` 中的 `ct-rag` 服务，只启动必需的服务
> - 其他服务（PostgreSQL、Redis、NATS、MinIO）应该能正常启动

4. **查看服务状态**

```bash
docker compose ps
```

5. **查看服务日志**

```bash
# 查看所有服务日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f postgres
docker compose logs -f redis
docker compose logs -f nats
```

6. **停止服务**

```bash
docker compose down
```

7. **修改配置后重启**

```bash
docker compose down
docker compose up -d
```

### 方式二：单独使用 Docker 命令

如果不想使用 Docker Compose，也可以单独启动每个服务：

#### 安装 PostgreSQL

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

#### 安装 Redis

```bash
docker run -d \
  --name panda-wiki-redis \
  --restart unless-stopped \
  -p 6379:6379 \
  -v redis_data:/data \
  redis:7-alpine \
  redis-server --requirepass panda-wiki-redis-password
```

#### 安装 NATS

```bash
docker run -d \
  --name panda-wiki-nats \
  --restart unless-stopped \
  -p 4222:4222 \
  -p 8222:8222 \
  nats:2.10-alpine \
  -js -m 8222 --user panda-wiki --password panda-wiki-nats-password
```

#### 安装 MinIO

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

#### 安装 RAG 服务（CT-RAG，可选）

```bash
# 注意：
# 1. RAG 服务是可选的，不是必需的
# 2. 需要根据实际的 RAG 服务镜像调整
# 3. 如果镜像不存在或需要认证，请使用其他可用的 RAG 服务镜像
# 4. 或者联系项目维护者获取正确的镜像地址

# 示例（需要替换为实际可用的镜像）:
docker run -d \
  --name panda-wiki-ct-rag \
  --restart unless-stopped \
  -e API_KEY=sk-1234567890 \
  -p 5050:5050 \
  YOUR_RAG_IMAGE:latest
```
<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>
read_file

### 配置防火墙

确保 Linux 服务器开放了必要的端口：

```bash
# Ubuntu/Debian
sudo ufw allow 5432/tcp  # PostgreSQL
sudo ufw allow 6379/tcp  # Redis
sudo ufw allow 4222/tcp  # NATS
sudo ufw allow 9000/tcp  # MinIO API
sudo ufw allow 9001/tcp  # MinIO Console
sudo ufw allow 5050/tcp  # RAG Service

# 或者一次性开放所有端口（仅开发环境）
sudo ufw allow from YOUR_MAC_IP
```

### 验证服务运行

在 Linux 服务器上验证服务是否正常运行：

```bash
# 检查 PostgreSQL
docker exec panda-wiki-postgres pg_isready -U panda-wiki

# 检查 Redis
docker exec panda-wiki-redis redis-cli -a panda-wiki-redis-password ping

# 检查 NATS
curl http://localhost:8222/healthz

# 检查 MinIO
curl http://localhost:9000/minio/health/live

# 检查所有容器状态
docker ps | grep panda-wiki
```

### 默认服务配置

使用上述 Docker 配置，服务的默认连接信息如下：

- **PostgreSQL**: 
  - 地址: `YOUR_LINUX_IP:5432`
  - 用户: `panda-wiki`
  - 密码: `panda-wiki-secret`
  - 数据库: `panda-wiki`

- **Redis**: 
  - 地址: `YOUR_LINUX_IP:6379`
  - 密码: `panda-wiki-redis-password`

- **NATS**: 
  - 地址: `nats://YOUR_LINUX_IP:4222`
  - 用户: `panda-wiki`
  - 密码: `panda-wiki-nats-password`

- **MinIO**: 
  - API 地址: `YOUR_LINUX_IP:9000`
  - 控制台: `http://YOUR_LINUX_IP:9001`
  - Access Key: `s3panda-wiki`
  - Secret Key: `panda-wiki-minio-secret`

- **RAG 服务**: 
  - 地址: `http://YOUR_LINUX_IP:5050`
  - API Key: `sk-1234567890`

> **安全提示**: 生产环境请务必修改默认密码！

## 配置步骤

### 1. 创建配置文件

复制配置文件示例：

```bash
cd backend
cp config.yml.example config.yml
```

### 2. 修改配置文件

编辑 `config.yml`，将服务地址指向你的 Linux 服务器：

```yaml
# PostgreSQL 配置
pg:
  dsn: "host=YOUR_LINUX_IP user=panda-wiki password=your_password dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai"

# Redis 配置
redis:
  addr: "YOUR_LINUX_IP:6379"
  password: "your_redis_password"

# NATS 配置
mq:
  nats:
    server: "nats://YOUR_LINUX_IP:4222"
    user: "panda-wiki"
    password: "your_nats_password"

# RAG 服务配置（如果使用）
rag:
  ct_rag:
    base_url: "http://YOUR_LINUX_IP:5050"
    api_key: "your_api_key"

# S3/MinIO 配置（如果使用）
s3:
  endpoint: "YOUR_LINUX_IP:9000"
  access_key: "your_access_key"
  secret_key: "your_secret_key"
```

**将 `YOUR_LINUX_IP` 替换为你的 Linux 服务器 IP 地址**

### 3. 使用环境变量（推荐）

你也可以使用环境变量来覆盖配置，这样更安全且不需要修改配置文件：

```bash
# PostgreSQL
export PG_DSN="host=YOUR_LINUX_IP user=panda-wiki password=your_password dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai"

# Redis
export REDIS_ADDR="YOUR_LINUX_IP:6379"
export REDIS_PASSWORD="your_redis_password"

# NATS
export MQ_NATS_SERVER="nats://YOUR_LINUX_IP:4222"
export NATS_PASSWORD="your_nats_password"

# RAG
export RAG_CT_RAG_BASE_URL="http://YOUR_LINUX_IP:5050"

# S3
export S3_ENDPOINT="YOUR_LINUX_IP:9000"
export S3_SECRET_KEY="your_secret_key"

# JWT Secret
export JWT_SECRET="your_jwt_secret"

# 管理员密码
export ADMIN_PASSWORD="your_admin_password"
```

### 4. 确保 Linux 服务器服务可访问

确保你的 Linux 服务器上的服务允许从 Mac 访问：

#### PostgreSQL
- 修改 `postgresql.conf`: `listen_addresses = '*'`
- 修改 `pg_hba.conf`: 添加允许连接的规则
- 确保防火墙开放 5432 端口

#### Redis
- 修改 `redis.conf`: `bind 0.0.0.0` 或 `bind YOUR_LINUX_IP`
- 确保防火墙开放 6379 端口

#### NATS
- 确保防火墙开放 4222 端口

### 5. 安装依赖并运行

```bash
cd backend

# 安装 Go 依赖
go mod download

# 生成代码（如果需要）
make generate

# 运行数据库迁移（首次运行）
go run cmd/migrate/main.go

# 运行 API 服务
go run cmd/api/main.go
```

### 6. 运行 Consumer（可选）

如果需要运行消息队列消费者：

```bash
go run cmd/consumer/main.go
```

## 常见问题

### 连接被拒绝

- 检查 Linux 服务器防火墙是否开放了相应端口
- 检查服务是否正在运行
- 检查服务配置是否允许远程连接

### 数据库连接失败

- 确认 PostgreSQL 用户和密码正确
- 确认数据库已创建
- 检查 `pg_hba.conf` 配置

### Redis 连接失败

- 确认 Redis 密码正确
- 检查 Redis 是否配置为允许远程连接

## 开发建议

1. **使用环境变量**：敏感信息（密码、密钥）建议使用环境变量而不是配置文件
2. **创建 .env 文件**：可以使用 `.env` 文件管理环境变量，然后使用 `source .env` 加载
3. **使用配置管理工具**：如 `direnv` 自动加载环境变量

## 快速启动脚本示例

创建 `start-dev.sh`:

```bash
#!/bin/bash

# 加载环境变量
source .env

# 运行服务
go run cmd/api/main.go
```

然后：

```bash
chmod +x start-dev.sh
./start-dev.sh
```

