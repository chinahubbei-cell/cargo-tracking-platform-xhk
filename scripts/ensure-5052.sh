#!/bin/bash
set -e
PROJECT_DIR="/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk"

pkill -f "go run main.go" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true
if command -v lsof >/dev/null 2>&1; then
  lsof -ti:5052 | xargs kill -9 2>/dev/null || true
  lsof -ti:5180 | xargs kill -9 2>/dev/null || true
  lsof -ti:5181 | xargs kill -9 2>/dev/null || true
fi
sleep 1

cd "$PROJECT_DIR/trackcard-server"
nohup env PORT=5052 DB_PATH="$PROJECT_DIR/trackcard-server/trackcard.db" go run main.go > "$PROJECT_DIR/track-backend-5052.log" 2>&1 &
cd "$PROJECT_DIR/trackcard-frontend"
nohup npm run dev > "$PROJECT_DIR/track-frontend.log" 2>&1 &
cd "$PROJECT_DIR/trackcard-admin/admin-frontend"
nohup npm run dev > "$PROJECT_DIR/admin-frontend.log" 2>&1 &
sleep 6

curl -s -o /dev/null -w 'backend(5052):%{http_code}\n' http://127.0.0.1:5052/api/health
curl -s -o /dev/null -w 'frontend(5180):%{http_code}\n' http://127.0.0.1:5180/login
curl -s -o /dev/null -w 'admin(5181):%{http_code}\n' http://127.0.0.1:5181/login
