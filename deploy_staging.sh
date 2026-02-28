#!/usr/bin/expect -f

# 设置超时时间 (无限等待)
set timeout -1

# 服务器配置
set HOST "192.168.6.251"
set PORT "31744"
set USER "root"
set PASS "lqPassword9"
set REMOTE_DIR "/data/trackcard-platform"
set LOCAL_DIR "/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk"

# 打印开始信息
send_user "\n>>> Step 1/6: 正在连接服务器创建目录...\n"

# 1. 创建远程目录
spawn ssh -o StrictHostKeyChecking=no -p $PORT $USER@$HOST "mkdir -p $REMOTE_DIR"
expect {
    "password:" { send "$PASS\r" }
    timeout { send_user "\n连接超时，请检查 VPN 连接。\n"; exit 1 }
}
expect eof

# 2. 同步后端代码 (trackcard-server)
send_user "\n>>> Step 2/6: 正在同步后端代码...\n"
spawn rsync -avz -e "ssh -o StrictHostKeyChecking=no -p $PORT" --progress \
    --exclude '.git' \
    --exclude 'tmp' \
    --exclude 'server' \
    --exclude '.DS_Store' \
    $LOCAL_DIR/trackcard-server \
    $USER@$HOST:$REMOTE_DIR/
expect {
    "password:" { send "$PASS\r" }
}
expect eof

# 3. 同步前端代码 (trackcard-frontend)
send_user "\n>>> Step 3/6: 正在同步前端代码 (Main)...\n"
spawn rsync -avz -e "ssh -o StrictHostKeyChecking=no -p $PORT" --progress \
    --exclude '.git' \
    --exclude 'node_modules' \
    --exclude 'dist' \
    --exclude '.DS_Store' \
    $LOCAL_DIR/trackcard-frontend \
    $USER@$HOST:$REMOTE_DIR/
expect {
    "password:" { send "$PASS\r" }
}
expect eof

# 4. 同步管理端代码 (trackcard-admin)
send_user "\n>>> Step 4/6: 正在同步管理端代码 (Admin)...\n"
spawn rsync -avz -e "ssh -o StrictHostKeyChecking=no -p $PORT" --progress \
    --exclude '.git' \
    --exclude 'node_modules' \
    --exclude 'dist' \
    --exclude '.DS_Store' \
    $LOCAL_DIR/trackcard-admin \
    $USER@$HOST:$REMOTE_DIR/
expect {
    "password:" { send "$PASS\r" }
}
expect eof

# 5. 上传 Nginx 配置
send_user "\n>>> Step 5/6: 正在配置 Nginx (域名: xhk-zj.kuaihuoyun.com)...\n"
spawn rsync -avz -e "ssh -o StrictHostKeyChecking=no -p $PORT" \
    $LOCAL_DIR/nginx_staging.conf \
    $USER@$HOST:/etc/nginx/conf.d/xhk-zj.kuaihuoyun.com.conf
expect {
    "password:" { send "$PASS\r" }
}
expect eof

# 6. 远程构建与服务重启
send_user "\n>>> Step 6/6: 正在执行远程已构建与服务重启...\n"

spawn ssh -o StrictHostKeyChecking=no -p $PORT $USER@$HOST
expect {
    "password:" { send "$PASS\r" }
}
expect "#"

# --- 检查环境 ---
send "go version\r"
expect "#"
send "node -v\r"
expect "#"

# --- 构建后端 ---
send_user ">>> 构建 Go 后端...\n"
send "cd $REMOTE_DIR/trackcard-server\r"
send "export GOPROXY=https://goproxy.cn,direct\r"
send "go mod tidy\r"
send "go build -o server main.go\r"
expect "#"

# 重启后端
send "pkill -f 'trackcard-server/server|\\./server' || true\r"
send "nohup env DB_PATH=$REMOTE_DIR/trackcard-server/trackcard.db ./server > server.log 2>&1 &\r"
expect "#"

# --- 构建前端 (Main) ---
send_user ">>> 构建前端资源 (Main)...\n"
send "cd $REMOTE_DIR/trackcard-frontend\r"
send "npm config set registry https://registry.npmmirror.com\r"
send "npm install\r"
send "npm run build\r"
expect "#"

# --- 构建前端 (Admin) ---
send_user ">>> 构建管理端资源 (Admin)...\n"
send "cd $REMOTE_DIR/trackcard-admin/admin-frontend\r"
send "npm install\r"
send "npm run build\r"
expect "#"

# --- 重启 Nginx ---
send_user ">>> 重启 Nginx 服务...\n"
send "nginx -t && nginx -s reload\r"
expect "#"

send "exit\r"
expect eof

send_user "\n✅ 部署完成!\n"
send_user "访问地址: http://xhk-zj.kuaihuoyun.com\n"
