#!/bin/bash
# ============================================================
#  全量部署脚本 (deploy_full.sh)
#  一键将本地开发环境的 前端 + 数据库 + 后端 完整部署到内部虚拟机
# ============================================================
#
# 使用方法:
#   chmod +x deploy_full.sh
#   ./deploy_full.sh              # 部署全部 (前端+数据库+后端重启)
#   ./deploy_full.sh --frontend   # 仅部署前端
#   ./deploy_full.sh --db         # 仅同步数据库
#   ./deploy_full.sh --restart    # 仅重启后端
#
# 前置要求: VPN 已连接, expect 已安装
# ============================================================

set -e

# ---- 服务器配置 ----
HOST="192.168.6.251"
PORT="31744"
REMOTE_USER="root"
PASS="lqPassword9"
REMOTE_DIR="/data/trackcard-platform"

# ---- 本地路径 ----
LOCAL_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCAL_FRONTEND="$LOCAL_DIR/trackcard-frontend"
LOCAL_ADMIN="$LOCAL_DIR/trackcard-admin/admin-frontend"
LOCAL_SERVER="$LOCAL_DIR/trackcard-server"
LOCAL_DB="${DB_PATH:-$LOCAL_SERVER/trackcard.db}"

# ---- 颜色输出 ----
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info()  { echo -e "${BLUE}[INFO]${NC}  $1"; }
log_ok()    { echo -e "${GREEN}[✅]${NC}    $1"; }
log_warn()  { echo -e "${YELLOW}[⚠️]${NC}    $1"; }
log_err()   { echo -e "${RED}[❌]${NC}    $1"; }
log_step()  { echo -e "\n${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${GREEN}▶  $1${NC}"; echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

# ---- 检查 VPN 连通性 ----
check_connectivity() {
    log_info "检查 VPN 连接到 $HOST:$PORT ..."
    if ! nc -z -w 5 "$HOST" "$PORT" 2>/dev/null; then
        log_err "无法连接到 $HOST:$PORT，请确认 VPN 已连接!"
        exit 1
    fi
    log_ok "VPN 连接正常"
}

# ---- SSH 执行远程命令 (自动输入密码) ----
remote_exec() {
    local CMD="$1"
    expect -c "
        set timeout 60
        spawn ssh -o StrictHostKeyChecking=no -p $PORT $REMOTE_USER@$HOST
        expect \"password:\" { send \"$PASS\r\" }
        expect \"#\"
        send \"$CMD\r\"
        expect \"#\"
        send \"exit\r\"
        expect eof
    " 2>&1
}

# ---- SCP 上传文件 (自动输入密码) ----
scp_upload() {
    local LOCAL_PATH="$1"
    local REMOTE_PATH="$2"
    expect -c "
        set timeout 300
        spawn scp -P $PORT -o StrictHostKeyChecking=no -o ConnectTimeout=30 $LOCAL_PATH $REMOTE_USER@$HOST:$REMOTE_PATH
        expect {
            \"password:\" { send \"$PASS\r\"; exp_continue }
            \"100%\" { }
            timeout { send_user \"\nUpload timed out\n\"; exit 1 }
            eof
        }
    "
}

# ---- RSYNC 同步目录 (自动输入密码) ----
rsync_upload() {
    local LOCAL_PATH="$1"
    local REMOTE_PATH="$2"
    expect -c "
        set timeout 300
        spawn rsync -avz --delete -e \"ssh -p $PORT -o StrictHostKeyChecking=no\" \"$LOCAL_PATH\" $REMOTE_USER@$HOST:\"$REMOTE_PATH\"
        expect {
            \"password:\" { send \"$PASS\r\"; exp_continue }
            eof
        }
    "
}

# ============================================================
# Step 1: 构建前端
# ============================================================
build_frontend() {
    log_step "Step 1/5: 本地构建前端资源"
    
    log_info "构建 Main Frontend ..."
    cd "$LOCAL_FRONTEND"
    npm run build
    log_ok "Main Frontend 构建完成"
    
    # Admin Frontend 如果有变动也构建
    if [ -f "$LOCAL_ADMIN/package.json" ]; then
        log_info "构建 Admin Frontend ..."
        cd "$LOCAL_ADMIN"
        npm run build 2>/dev/null || log_warn "Admin Frontend 构建跳过 (可能无变动)"
        log_ok "Admin Frontend 处理完成"
    fi
    
    cd "$LOCAL_DIR"
}

# ============================================================
# Step 2: 上传前端 dist
# ============================================================
deploy_frontend() {
    log_step "Step 2/5: 上传前端资源到 VM"
    
    log_info "上传 Main Frontend dist ..."
    rsync_upload "$LOCAL_FRONTEND/dist/" "$REMOTE_DIR/trackcard-frontend/dist/"
    log_ok "Main Frontend 上传完成"
    
    if [ -d "$LOCAL_ADMIN/dist" ]; then
        log_info "上传 Admin Frontend dist ..."
        rsync_upload "$LOCAL_ADMIN/dist/" "$REMOTE_DIR/trackcard-admin/admin-frontend/dist/"
        log_ok "Admin Frontend 上传完成"
    fi
}

# ============================================================
# Step 3: 停止远程后端
# ============================================================
stop_remote_backend() {
    log_step "Step 3/5: 停止 VM 后端服务"
    
    expect -c "
        set timeout 30
        spawn ssh -o StrictHostKeyChecking=no -p $PORT $REMOTE_USER@$HOST
        expect \"password:\" { send \"$PASS\r\" }
        expect \"#\"
        send \"pkill -9 -f '$REMOTE_DIR/trackcard-server/server|\\./server' || true\r\"
        expect \"#\"
        send \"sleep 2 && echo BACKEND_STOPPED\r\"
        expect \"BACKEND_STOPPED\"
        expect \"#\"
        send \"exit\r\"
        expect eof
    "
    log_ok "后端服务已停止"
}

# ============================================================
# Step 4: 同步数据库
# ============================================================
deploy_database() {
    log_step "Step 4/5: 同步本地数据库到 VM (SQL dump 方式)"
    
    if [ ! -f "$LOCAL_DB" ]; then
        log_err "本地数据库不存在: $LOCAL_DB"
        exit 1
    fi
    
    local DB_SIZE=$(du -h "$LOCAL_DB" | cut -f1)
    log_info "本地数据库大小: $DB_SIZE"
    
    # 导出为 SQL 文本 (避免 SQLite 版本不兼容)
    log_info "导出本地数据库为 SQL dump ..."
    sqlite3 "$LOCAL_DB" ".dump" > /tmp/trackcard_dump.sql
    gzip -f /tmp/trackcard_dump.sql
    local DUMP_SIZE=$(du -h /tmp/trackcard_dump.sql.gz | cut -f1)
    log_ok "SQL dump 导出完成 (压缩后 $DUMP_SIZE)"
    
    # 上传 SQL dump
    log_info "上传 SQL dump 到 VM ..."
    scp_upload /tmp/trackcard_dump.sql.gz /tmp/trackcard_dump.sql.gz
    log_ok "SQL dump 上传完成"
    
    # 在远程重建数据库
    log_info "在 VM 上重建数据库 ..."
    expect -c "
        set timeout 120
        spawn ssh -o StrictHostKeyChecking=no -p $PORT $REMOTE_USER@$HOST
        expect \"password:\" { send \"$PASS\r\" }
        expect \"#\"
        send \"cd $REMOTE_DIR/trackcard-server\r\"
        expect \"#\"
        send \"cp trackcard.db trackcard.db.bak 2>/dev/null; rm -f trackcard.db trackcard.db-shm trackcard.db-wal\r\"
        expect \"#\"
        send \"gunzip -c /tmp/trackcard_dump.sql.gz | sqlite3 trackcard.db && echo DB_REBUILD_OK\r\"
        expect \"DB_REBUILD_OK\"
        expect \"#\"
        send \"sqlite3 trackcard.db 'PRAGMA integrity_check;' | head -1\r\"
        expect \"#\"
        send \"sqlite3 trackcard.db 'SELECT COUNT(*) FROM shipments;'\r\"
        expect \"#\"
        send \"exit\r\"
        expect eof
    "
    log_ok "数据库重建完成 ✅"
}

# ============================================================
# Step 5: 重启远程后端 + Nginx
# ============================================================
restart_remote_services() {
    log_step "Step 5/5: 重启 VM 后端服务和 Nginx"
    
    expect -c "
        set timeout 30
        spawn ssh -o StrictHostKeyChecking=no -p $PORT $REMOTE_USER@$HOST
        expect \"password:\" { send \"$PASS\r\" }
        expect \"#\"
        send \"cd $REMOTE_DIR/trackcard-server && nohup env DB_PATH=$REMOTE_DIR/trackcard-server/trackcard.db ./server > server.log 2>&1 &\r\"
        expect \"#\"
        send \"sleep 3\r\"
        expect \"#\"
        send \"nginx -s reload 2>/dev/null || true\r\"
        expect \"#\"
        send \"ps aux | grep '$REMOTE_DIR/trackcard-server/server' | grep -v grep | head -1\r\"
        expect \"#\"
        send \"netstat -tlnp 2>/dev/null | grep -E '8080|8000' || ss -tlnp | grep -E '8080|8000'\r\"
        expect \"#\"
        send \"exit\r\"
        expect eof
    "
    log_ok "后端服务和 Nginx 已重启"
}

# ============================================================
# 主流程
# ============================================================
main() {
    echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║       🚀 货物追踪平台 - 全量部署脚本                    ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}\n"
    
    local MODE="${1:-all}"
    
    check_connectivity
    
    case "$MODE" in
        --frontend)
            build_frontend
            deploy_frontend
            restart_remote_services
            ;;
        --db)
            stop_remote_backend
            deploy_database
            restart_remote_services
            ;;
        --restart)
            restart_remote_services
            ;;
        all|*)
            build_frontend
            deploy_frontend
            stop_remote_backend
            deploy_database
            restart_remote_services
            ;;
    esac
    
    echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  ✅ 部署完成!                                           ║${NC}"
    echo -e "${GREEN}║  🌐 访问: http://xhk-zj.kuaihuoyun.com                  ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}\n"
}

main "$@"
