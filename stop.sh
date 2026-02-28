#!/bin/bash
# 货物追踪平台停止脚本

echo "⏹️  停止货物追踪平台服务..."

pkill -9 -f "go run main.go" 2>/dev/null || true
pkill -9 -f "npm run dev" 2>/dev/null || true
pkill -9 -f "trackcard-server" 2>/dev/null || true

echo "✅ 所有服务已停止"
