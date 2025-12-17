#!/bin/bash

# PandaWiki 开发环境启动脚本

set -e

echo "🚀 启动 PandaWiki 开发环境..."

# 检查 .env 文件
if [ ! -f .env ]; then
    echo "⚠️  未找到 .env 文件，请先复制 .env.example 并配置"
    echo "   cp .env.example .env"
    exit 1
fi

# 加载环境变量
echo "📝 加载环境变量..."
source .env

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "❌ 未找到 Go，请先安装 Go 1.24.3 或更高版本"
    exit 1
fi

# 检查依赖
echo "📦 检查 Go 依赖..."
if [ ! -f go.sum ]; then
    echo "📥 下载依赖..."
    go mod download
fi

# 生成代码（如果需要）
if [ "$1" == "--generate" ]; then
    echo "🔧 生成代码..."
    make generate
fi

# 运行服务
echo "🎯 启动 API 服务..."
echo "   访问地址: http://localhost:${HTTP_PORT:-8000}"
echo ""

go run cmd/api/main.go

