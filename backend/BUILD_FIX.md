# 编译错误修复指南

## 错误：undefined: createApp

如果遇到 `undefined: createApp` 错误，说明需要先生成 Wire 依赖注入代码。

## 快速修复

### 方法一：使用更新后的脚本（推荐）

```bash
cd backend
./start-dev.sh --generate
```

脚本会自动：
1. 检查并安装 `wire` 和 `swag` 工具
2. 生成所需的代码
3. 启动服务

### 方法二：手动生成代码

```bash
cd backend

# 1. 安装必要的工具
go install github.com/google/wire/cmd/wire@latest
go install github.com/swaggo/swag/cmd/swag@latest

# 2. 确保工具在 PATH 中
export PATH=$HOME/go/bin:$PATH
# 或者如果设置了 GOPATH
export PATH=$GOPATH/bin:$PATH

# 3. 生成代码
make generate

# 4. 运行服务
go run cmd/api/main.go
```

### 方法三：使用 Makefile

```bash
cd backend
make generate
go run cmd/api/main.go
```

## 工具安装位置

- 默认位置: `$HOME/go/bin/`
- 如果设置了 GOPATH: `$GOPATH/bin/`

## 验证工具安装

```bash
# 检查 wire
which wire
wire version

# 检查 swag
which swag
swag version
```

## 常见问题

### 问题：工具安装后仍然找不到

**解决方案**：
```bash
# 检查工具是否已安装
ls -la $HOME/go/bin/wire
ls -la $HOME/go/bin/swag

# 如果存在，添加到 PATH
export PATH=$HOME/go/bin:$PATH

# 永久添加到 ~/.zshrc 或 ~/.bash_profile
echo 'export PATH=$HOME/go/bin:$PATH' >> ~/.zshrc
source ~/.zshrc
```

### 问题：make generate 失败

**可能原因**：
1. 缺少依赖
2. 工具未正确安装
3. 代码有语法错误

**解决方案**：
```bash
# 下载所有依赖
go mod download

# 重新安装工具
go install github.com/google/wire/cmd/wire@latest
go install github.com/swaggo/swag/cmd/swag@latest

# 清理并重新生成
make generate
```

## 生成的文件

运行 `make generate` 后会生成以下文件：
- `cmd/api/wire_gen.go` - API 服务的依赖注入代码
- `cmd/consumer/wire_gen.go` - Consumer 服务的依赖注入代码
- `cmd/migrate/wire_gen.go` - 迁移工具的依赖注入代码
- `docs/swagger.json` 和 `docs/swagger.yaml` - API 文档

这些文件是自动生成的，不要手动编辑。

