#!/bin/bash

# PandaWiki 开发环境启动脚本

set -e

echo "🚀 启动 PandaWiki 开发环境..."

# 检查 .env 文件
if [ ! -f .env ]; then
    echo "⚠️  未找到 .env 文件"
    echo ""
    echo "请创建 .env 文件并配置环境变量。示例："
    echo ""
    echo "cat > .env << 'EOF'"
    echo "# Linux 服务器 IP"
    echo "LINUX_SERVER_IP=192.168.1.100"
    echo ""
    echo "# PostgreSQL"
    echo "export PG_DSN=\"host=\${LINUX_SERVER_IP} user=panda-wiki password=your_password dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai\""
    echo ""
    echo "# Redis"
    echo "export REDIS_ADDR=\"\${LINUX_SERVER_IP}:6379\""
    echo "export REDIS_PASSWORD=\"your_password\""
    echo ""
    echo "# NATS"
    echo "export MQ_NATS_SERVER=\"nats://\${LINUX_SERVER_IP}:4222\""
    echo "export NATS_PASSWORD=\"your_password\""
    echo ""
    echo "# JWT Secret"
    echo "export JWT_SECRET=\"your_jwt_secret\""
    echo ""
    echo "# 管理员密码"
    echo "export ADMIN_PASSWORD=\"your_admin_password\""
    echo "EOF"
    echo ""
    echo "或者查看 QUICKSTART.md 获取详细说明"
    exit 1
fi

# 加载环境变量
echo "📝 加载环境变量..."
source .env

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "❌ 未找到 Go 命令"
    echo ""
    
    # 检查常见安装位置
    if [ -f "/usr/local/go/bin/go" ]; then
        echo "⚠️  检测到 Go 已安装在 /usr/local/go/bin/go，但未添加到 PATH"
        echo ""
        echo "请将以下内容添加到你的 shell 配置文件 (~/.zshrc 或 ~/.bash_profile):"
        echo ""
        echo "export GOROOT=/usr/local/go"
        echo "export GOPATH=\$HOME/go"
        echo "export PATH=\$GOROOT/bin:\$GOPATH/bin:\$PATH"
        echo ""
        echo "然后运行:"
        echo "  source ~/.zshrc  # 或 source ~/.bash_profile"
        echo ""
        echo "或者临时添加:"
        echo "  export PATH=/usr/local/go/bin:\$PATH"
    elif [ -f "$HOME/go/bin/go" ]; then
        echo "⚠️  检测到 Go 已安装在 $HOME/go/bin/go，但未添加到 PATH"
        echo ""
        echo "请将以下内容添加到你的 shell 配置文件:"
        echo "  export PATH=\$HOME/go/bin:\$PATH"
    else
        echo "请先安装 Go 1.24.3 或更高版本"
        echo ""
        echo "安装方法："
        echo "1. 使用 Homebrew: brew install go"
        echo "2. 从官网下载: https://go.dev/dl/"
        echo "3. 使用包管理器安装"
    fi
    exit 1
fi

# 检查依赖
echo "📦 检查 Go 依赖..."
if [ ! -f go.sum ]; then
    echo "📥 下载依赖..."
    go mod download
fi

# 检查是否需要生成代码
NEED_GENERATE=false
if [ ! -f cmd/api/wire_gen.go ]; then
    NEED_GENERATE=true
    echo "⚠️  wire_gen.go 文件不存在，需要生成"
elif [ "$1" == "--generate" ]; then
    NEED_GENERATE=true
    echo "⚠️  使用 --generate 参数，强制重新生成代码"
elif ! grep -q "func createApp" cmd/api/wire_gen.go 2>/dev/null; then
    NEED_GENERATE=true
    echo "⚠️  检测到 wire_gen.go 文件不完整，需要重新生成"
fi

# 如果文件存在，尝试验证是否可以编译（快速检查）
if [ "$NEED_GENERATE" = false ] && [ -f cmd/api/wire_gen.go ]; then
    # 快速语法检查
    if ! go list -f '{{.GoFiles}}' ./cmd/api 2>/dev/null | grep -q wire_gen; then
        echo "⚠️  wire_gen.go 可能未被包含在构建中，需要重新生成"
        NEED_GENERATE=true
    fi
fi

if [ "$NEED_GENERATE" = true ]; then
    echo "🔧 生成代码（Wire 依赖注入）..."
    
    # 确保 GOPATH/bin 在 PATH 中（用于安装的工具）
    if [ -n "$GOPATH" ]; then
        export PATH="$GOPATH/bin:$PATH"
    elif [ -d "$HOME/go/bin" ]; then
        export PATH="$HOME/go/bin:$PATH"
    fi
    
    if ! command -v wire &> /dev/null; then
        echo "⚠️  wire 命令未找到，正在安装..."
        go install github.com/google/wire/cmd/wire@latest
        # 重新加载 PATH
        if [ -n "$GOPATH" ]; then
            export PATH="$GOPATH/bin:$PATH"
        elif [ -d "$HOME/go/bin" ]; then
            export PATH="$HOME/go/bin:$PATH"
        fi
    fi
    if ! command -v swag &> /dev/null; then
        echo "⚠️  swag 命令未找到，正在安装..."
        go install github.com/swaggo/swag/cmd/swag@latest
        # 重新加载 PATH
        if [ -n "$GOPATH" ]; then
            export PATH="$GOPATH/bin:$PATH"
        elif [ -d "$HOME/go/bin" ]; then
            export PATH="$HOME/go/bin:$PATH"
        fi
    fi
    
    # 验证工具已安装
    if ! command -v wire &> /dev/null || ! command -v swag &> /dev/null; then
        echo "❌ 工具安装失败，请手动安装："
        echo "   go install github.com/google/wire/cmd/wire@latest"
        echo "   go install github.com/swaggo/swag/cmd/swag@latest"
        echo "   然后确保 \$GOPATH/bin 或 \$HOME/go/bin 在 PATH 中"
        exit 1
    fi
    
    make generate || {
        echo "❌ 代码生成失败，请检查错误信息"
        exit 1
    }
fi

# 运行服务
echo "🎯 启动 API 服务..."
echo "   访问地址: http://localhost:${HTTP_PORT:-8000}"
echo ""

# 尝试运行，如果失败可能是代码生成问题
# 注意：使用 ./cmd/api 而不是 cmd/api/main.go，以确保包含同包的所有文件（如 wire_gen.go）
TEMP_LOG=$(mktemp)
set +e  # 暂时关闭错误退出，以便捕获错误
go run ./cmd/api 2>&1 | tee "$TEMP_LOG"
EXIT_CODE=${PIPESTATUS[0]}  # 获取 go run 的退出码
set -e  # 重新开启错误退出

if [ $EXIT_CODE -ne 0 ]; then
    # 检查是否是 createApp 未定义的错误
    if grep -q "undefined: createApp" "$TEMP_LOG" 2>/dev/null; then
        echo ""
        echo "❌ 检测到 createApp 未定义错误"
        echo "🔧 尝试重新生成代码..."
        
        # 确保工具在 PATH 中
        if [ -n "$GOPATH" ]; then
            export PATH="$GOPATH/bin:$PATH"
        elif [ -d "$HOME/go/bin" ]; then
            export PATH="$HOME/go/bin:$PATH"
        fi
        
        # 安装工具（如果需要）
        if ! command -v wire &> /dev/null; then
            go install github.com/google/wire/cmd/wire@latest
            export PATH="$HOME/go/bin:$PATH"
        fi
        if ! command -v swag &> /dev/null; then
            go install github.com/swaggo/swag/cmd/swag@latest
            export PATH="$HOME/go/bin:$PATH"
        fi
        
        # 重新生成代码
        if make generate; then
            echo "✅ 代码重新生成成功，再次尝试启动..."
            echo ""
            go run ./cmd/api
        else
            echo "❌ 代码生成失败，请手动运行: make generate"
            rm -f "$TEMP_LOG"
            exit 1
        fi
    else
        # 其他错误，直接退出
        rm -f "$TEMP_LOG"
        exit 1
    fi
    rm -f "$TEMP_LOG"
fi

