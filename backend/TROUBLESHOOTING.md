# 故障排查指南

## Docker Compose 常见问题

### 1. version 警告

**问题**: 
```
WARN[0000] the attribute `version` is obsolete
```

**解决方案**: 
已修复！`docker-compose.services.yml` 已移除 `version` 字段。如果仍有警告，请使用最新版本的配置文件。

### 2. RAG 服务镜像拉取失败

**问题**:
```
Error response from daemon: Head "https://registry-1.docker.io/v2/chaitin/ct-rag/manifests/latest": unauthorized: incorrect username or password
```

**原因**: 
- RAG 服务镜像可能不存在或需要特殊权限
- RAG 服务是可选的，不是必需的

**解决方案**:

1. **临时跳过 RAG 服务**（推荐）:
   ```bash
   # RAG 服务在 docker-compose.yml 中默认已注释
   # 直接启动其他服务即可
   docker compose up -d
   ```

2. **验证其他服务是否正常**:
   ```bash
   docker compose ps
   # 应该看到 postgres, redis, nats, minio 正常运行
   ```

3. **如果需要 RAG 服务**:
   - 联系项目维护者获取正确的镜像地址
   - 或使用其他可用的 RAG 服务
   - 或配置外部 RAG API

### 3. 端口被占用

**问题**: 
```
Error: bind: address already in use
```

**解决方案**:
```bash
# 检查端口占用
sudo lsof -i :5432
sudo netstat -tulpn | grep :5432

# 停止占用端口的服务
sudo systemctl stop postgresql  # 如果系统安装了 PostgreSQL

# 或修改 docker-compose.yml 中的端口映射
# 例如: "5433:5432"  # 使用 5433 端口
```

### 4. 容器无法启动

**问题**: 容器启动后立即退出

**解决方案**:
```bash
# 查看容器日志
docker logs panda-wiki-postgres
docker logs panda-wiki-redis

# 查看容器状态
docker ps -a

# 检查配置文件
docker compose config
```

### 5. 数据卷权限问题

**问题**: 容器无法写入数据卷

**解决方案**:
```bash
# 检查数据卷权限
docker volume inspect postgres_data

# 如果需要，删除并重新创建
docker compose down -v
docker compose up -d
```

### 6. 网络连接问题

**问题**: 从 Mac 无法连接到 Linux 服务器上的服务

**解决方案**:

1. **检查防火墙**:
   ```bash
   # 在 Linux 服务器上
   sudo ufw status
   sudo ufw allow 5432/tcp
   sudo ufw allow 6379/tcp
   sudo ufw allow 4222/tcp
   ```

2. **检查服务是否监听正确地址**:
   ```bash
   # 在 Linux 服务器上
   docker port panda-wiki-postgres
   # 应该显示: 0.0.0.0:5432->5432/tcp
   ```

3. **测试连接**:
   ```bash
   # 在 Mac 上测试
   telnet YOUR_LINUX_IP 5432
   ```

### 7. 服务健康检查失败

**问题**: 服务启动但健康检查失败

**解决方案**:
```bash
# 检查服务是否真的在运行
docker exec panda-wiki-postgres pg_isready -U panda-wiki

# 如果服务正常，可以暂时禁用健康检查
# 在 docker-compose.yml 中注释掉 healthcheck 部分
```

## 服务特定问题

### PostgreSQL

**连接失败**:
```bash
# 检查 PostgreSQL 是否运行
docker exec panda-wiki-postgres pg_isready -U panda-wiki

# 查看日志
docker logs panda-wiki-postgres

# 常见问题: 数据库未初始化
# 删除数据卷重新创建
docker compose down -v
docker compose up -d postgres
```

### Redis

**连接失败**:
```bash
# 测试连接
docker exec panda-wiki-redis redis-cli -a panda-wiki-redis-password ping
# 应该返回: PONG

# 如果密码错误，检查启动命令
docker logs panda-wiki-redis
```

### NATS

**连接失败**:
```bash
# 检查健康状态
curl http://localhost:8222/healthz

# 查看日志
docker logs panda-wiki-nats
```

### MinIO

**无法访问控制台**:
```bash
# 检查服务状态
curl http://localhost:9000/minio/health/live

# 访问控制台: http://YOUR_LINUX_IP:9001
# 用户名: s3panda-wiki
# 密码: panda-wiki-minio-secret
```

## 快速诊断命令

```bash
# 查看所有服务状态
docker compose ps

# 查看所有服务日志
docker compose logs

# 查看特定服务日志
docker compose logs postgres
docker compose logs redis

# 重启所有服务
docker compose restart

# 停止所有服务
docker compose down

# 停止并删除数据
docker compose down -v
```

## 获取帮助

如果以上方法都无法解决问题：

1. 查看详细日志: `docker compose logs -f`
2. 检查 Docker 版本: `docker --version` 和 `docker compose version`
3. 查看系统资源: `docker stats`
4. 提交 Issue 到项目仓库，附上相关日志

