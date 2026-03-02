#!/bin/bash
# 货物追踪平台启动脚本
# 使用方法: ./start.sh

set -e

PROJECT_DIR="/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk"
BACKEND_DIR="$PROJECT_DIR/trackcard-server"
FRONTEND_DIR="$PROJECT_DIR/trackcard-frontend"

echo "🚀 启动货物追踪平台..."

# 1. 停止已有进程
echo "⏹️  停止已有服务..."
pkill -9 -f "go run main.go" 2>/dev/null || true
pkill -9 -f "npm run dev" 2>/dev/null || true
sleep 2

# 2. 启动后端
echo "🔧 启动后端服务 (端口 5052)..."
cd "$BACKEND_DIR"
export PORT=5052
export DB_PATH="$BACKEND_DIR/trackcard.db"
go run main.go &
BACKEND_PID=$!
sleep 5

# 3. 验证后端
if curl -s http://localhost:5052/api/health > /dev/null 2>&1 || curl -s http://localhost:5052/api/auth/login -X POST > /dev/null 2>&1; then
    echo "✅ 后端服务已启动"
else
    echo "⚠️  后端服务可能未正常启动，请检查日志"
fi

# 4. 启动前端
echo "🎨 启动前端服务 (端口 5173)..."
cd "$FRONTEND_DIR"
npm run dev &
FRONTEND_PID=$!
sleep 3

echo ""
echo "=========================================="
echo "✅ 服务已启动！"
echo "=========================================="
echo "前端地址: http://localhost:5173/"
echo "后端地址: http://localhost:5052/"
echo ""
echo "登录信息:"
echo "  邮箱: admin@trackcard.com"
echo "  密码: admin123"
echo ""
echo "停止服务: Ctrl+C 或运行 ./stop.sh"
echo "=========================================="

# 保持脚本运行
wait
