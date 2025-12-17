# 快速开始 - Mac 开发环境连接 Linux 服务器

## 快速配置步骤

### 1. 创建配置文件

```bash
cd backend
cp config.yml.example config.yml
```

### 2. 修改配置文件中的服务地址

编辑 `config.yml`，将所有 `localhost` 或 Docker 容器名替换为你的 Linux 服务器 IP：

```yaml
pg:
  dsn: "host=YOUR_LINUX_IP user=panda-wiki password=your_password dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai"

redis:
  addr: "YOUR_LINUX_IP:6379"

mq:
  nats:
    server: "nats://YOUR_LINUX_IP:4222"
```

### 3. 使用环境变量（推荐方式）

创建环境变量文件 `.env`（不会被 git 跟踪）：

```bash
# 在 backend 目录下创建 .env 文件
cat > .env << 'EOF'
# Linux 服务器 IP
LINUX_SERVER_IP=192.168.1.100

# PostgreSQL
export PG_DSN="host=${LINUX_SERVER_IP} user=panda-wiki password=your_password dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai"

# Redis
export REDIS_ADDR="${LINUX_SERVER_IP}:6379"
export REDIS_PASSWORD="your_password"

# NATS
export MQ_NATS_SERVER="nats://${LINUX_SERVER_IP}:4222"
export NATS_PASSWORD="your_password"

# JWT Secret
export JWT_SECRET="your_jwt_secret"

# 管理员密码
export ADMIN_PASSWORD="your_admin_password"
EOF
```

然后加载环境变量：

```bash
source .env
```

### 4. 在 Linux 服务器上安装服务

#### 方式一：使用 Docker Compose（推荐）

在 Linux 服务器上：

```bash
# 1. 创建服务目录
mkdir -p ~/panda-wiki-services
cd ~/panda-wiki-services

# 2. 从 Mac 复制 docker-compose 文件到 Linux 服务器
# 在 Mac 上执行：
scp backend/docker-compose.services.yml user@YOUR_LINUX_IP:~/panda-wiki-services/docker-compose.yml

# 3. 在 Linux 服务器上启动所有服务
cd ~/panda-wiki-services
docker compose up -d

# 4. 查看服务状态
docker compose ps

# 5. 开放防火墙端口
sudo ufw allow 5432/tcp  # PostgreSQL
sudo ufw allow 6379/tcp  # Redis
sudo ufw allow 4222/tcp  # NATS
sudo ufw allow 9000/tcp  # MinIO
sudo ufw allow 5050/tcp  # RAG
```

#### 方式二：单独安装服务

如果不想使用 Docker Compose，可以单独安装：

**PostgreSQL:**
```bash
docker run -d \
  --name panda-wiki-postgres \
  --restart unless-stopped \
  -e POSTGRES_USER=panda-wiki \
  -e POSTGRES_PASSWORD=panda-wiki-secret \
  -e POSTGRES_DB=panda-wiki \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:15-alpine
```

**Redis:**
```bash
docker run -d \
  --name panda-wiki-redis \
  --restart unless-stopped \
  -p 6379:6379 \
  -v redis_data:/data \
  redis:7-alpine \
  redis-server --requirepass panda-wiki-redis-password
```

**NATS:**
```bash
docker run -d \
  --name panda-wiki-nats \
  --restart unless-stopped \
  -p 4222:4222 \
  -p 8222:8222 \
  nats:2.10-alpine \
  -js -m 8222 --user panda-wiki --password panda-wiki-nats-password
```

**MinIO:**
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

**RAG 服务:**
```bash
docker run -d \
  --name panda-wiki-ct-rag \
  --restart unless-stopped \
  -e API_KEY=sk-1234567890 \
  -p 5050:5050 \
  chaitin/ct-rag:latest
```

> 详细说明请查看 [DEVELOPMENT.md](./DEVELOPMENT.md) 中的 Docker 安装章节

### 5. 安装依赖并运行

```bash
# 安装 Go 依赖
go mod download

# 运行数据库迁移（首次运行）
go run cmd/migrate/main.go

# 启动 API 服务
go run cmd/api/main.go
```

或者使用启动脚本：

```bash
./start-dev.sh
```

## 验证连接

启动后，访问 `http://localhost:8000` 应该能看到 API 服务运行。

## 常见问题

**Q: 连接被拒绝怎么办？**
A: 检查 Linux 服务器防火墙和服务的监听地址配置

**Q: 数据库连接失败？**
A: 确认 PostgreSQL 用户权限和 `pg_hba.conf` 配置

**Q: 如何查看日志？**
A: 设置 `LOG_LEVEL=-4` 查看 debug 日志

## 下一步

- 查看 [DEVELOPMENT.md](./DEVELOPMENT.md) 了解详细配置
- 配置前端项目（web 目录）
- 配置 AI 模型以启用 AI 功能

