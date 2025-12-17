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

