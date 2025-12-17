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

### 4. 确保 Linux 服务器服务可访问

在 Linux 服务器上：

#### PostgreSQL
```bash
# 编辑 postgresql.conf
sudo nano /etc/postgresql/*/main/postgresql.conf
# 设置: listen_addresses = '*'

# 编辑 pg_hba.conf
sudo nano /etc/postgresql/*/main/pg_hba.conf
# 添加: host    all    all    0.0.0.0/0    md5

# 重启 PostgreSQL
sudo systemctl restart postgresql

# 开放防火墙端口
sudo ufw allow 5432/tcp
```

#### Redis
```bash
# 编辑 redis.conf
sudo nano /etc/redis/redis.conf
# 设置: bind 0.0.0.0
# 设置: requirepass your_password

# 重启 Redis
sudo systemctl restart redis

# 开放防火墙端口
sudo ufw allow 6379/tcp
```

#### NATS
```bash
# 开放防火墙端口
sudo ufw allow 4222/tcp
```

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

