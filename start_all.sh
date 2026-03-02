#!/bin/bash
# 货物追踪平台全系统启动脚本
# 使用方法: ./start_all.sh

set -e

PROJECT_DIR="/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk"
TRACK_BACKEND_DIR="$PROJECT_DIR/trackcard-server"
TRACK_FRONTEND_DIR="$PROJECT_DIR/trackcard-frontend"
ADMIN_BACKEND_DIR="$PROJECT_DIR/trackcard-admin/admin-server"
ADMIN_FRONTEND_DIR="$PROJECT_DIR/trackcard-admin/admin-frontend"

echo "🚀 启动货物追踪全平台系统..."

# 1. 停止已有进程
echo "⏹️  停止已有服务..."
pkill -9 -f "go run main.go" 2>/dev/null || true
pkill -9 -f "npm run dev" 2>/dev/null || true
# Kill processes on specific ports if pkill by name misses them
if command -v lsof >/dev/null 2>&1; then
  lsof -ti:5051 | xargs kill -9 2>/dev/null || true
  lsof -ti:5180 | xargs kill -9 2>/dev/null || true
  lsof -ti:8001 | xargs kill -9 2>/dev/null || true
  lsof -ti:5181 | xargs kill -9 2>/dev/null || true
else
  echo "⚠️  lsof 不可用，跳过按端口清理（仅按进程名清理）"
fi
sleep 2

# 2. 启动追踪后端 (5051)
echo "🔧 启动追踪后端服务 (端口 5051)..."
cd "$TRACK_BACKEND_DIR"
export PORT=5051
export DB_PATH="$TRACK_BACKEND_DIR/trackcard.db"
go run main.go > "$PROJECT_DIR/track-backend.log" 2>&1 &
TRACK_BACKEND_PID=$!
echo "   PID: $TRACK_BACKEND_PID"

# 3. 启动管理后台后端 (8001)
echo "🔧 启动管理后台后端 (端口 8001)..."
cd "$ADMIN_BACKEND_DIR"
export PORT=8001
export DB_PATH="$TRACK_BACKEND_DIR/trackcard.db"
go run main.go > "$PROJECT_DIR/admin-backend.log" 2>&1 &
ADMIN_BACKEND_PID=$!
echo "   PID: $ADMIN_BACKEND_PID"

sleep 5

# 4. 验证后端
if curl -s http://localhost:5051/api/health > /dev/null 2>&1 || curl -s http://localhost:5051/api/auth/login -X POST > /dev/null 2>&1; then
    echo "✅ 追踪后端已启动"
else
    echo "⚠️  追踪后端可能未正常启动，请检查 track-backend.log"
fi

if curl -s http://localhost:8001/api/health > /dev/null 2>&1 || curl -s http://localhost:8001/api/admin/auth/login -X POST > /dev/null 2>&1; then
    echo "✅ 管理后台后端已启动"
else
    echo "⚠️  管理后台后端可能未正常启动，请检查 admin-backend.log"
fi

# 5. 启动追踪前端 (5180)
echo "🎨 启动追踪前端 (端口 5180)..."
cd "$TRACK_FRONTEND_DIR"
npm run dev > "$PROJECT_DIR/track-frontend.log" 2>&1 &
TRACK_FRONTEND_PID=$!
echo "   PID: $TRACK_FRONTEND_PID"

# 6. 启动管理后台前端 (5181)
echo "🎨 启动管理后台前端 (端口 5181)..."
cd "$ADMIN_FRONTEND_DIR"
npm run dev > "$PROJECT_DIR/admin-frontend.log" 2>&1 &
ADMIN_FRONTEND_PID=$!
echo "   PID: $ADMIN_FRONTEND_PID"

sleep 5

echo ""
echo "=========================================="
echo "✅ 全系统服务已启动！"
echo "=========================================="
echo "追踪系统前端: http://localhost:5180/"
echo "追踪系统后端: http://localhost:5051/"
echo "管理后台前端: http://localhost:5181/"
echo "管理后台后端: http://localhost:8001/"
echo ""
echo "日志文件位置:"
echo "  - $PROJECT_DIR/track-backend.log"
echo "  - $PROJECT_DIR/admin-backend.log"
echo "  - $PROJECT_DIR/track-frontend.log"
echo "  - $PROJECT_DIR/admin-frontend.log"
echo ""
echo "停止服务: 运行 pkill -f 'go run main.go'; pkill -f 'npm run dev'"
echo "=========================================="

# 保持脚本
wait
