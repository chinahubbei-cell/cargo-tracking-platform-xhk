#!/bin/bash
# 全球机场数据自动更新脚本
# 用法: 
#   ./update_airports.sh           # 手动执行
#   crontab -e 添加:
#   0 3 * * 0 /path/to/update_airports.sh  # 每周日凌晨3点自动执行

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="${SCRIPT_DIR}/data_sources"
LOG_FILE="${SCRIPT_DIR}/update_airports.log"
BACKUP_DIR="${DATA_DIR}/backups"

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# 创建备份目录
mkdir -p "$BACKUP_DIR"

log "=== 开始机场数据更新 ==="

# 1. 备份现有数据
if [ -f "${DATA_DIR}/airports.csv" ]; then
    BACKUP_NAME="airports_$(date '+%Y%m%d_%H%M%S').csv"
    cp "${DATA_DIR}/airports.csv" "${BACKUP_DIR}/${BACKUP_NAME}"
    log "已备份现有数据到 ${BACKUP_NAME}"
fi

# 2. 下载最新数据
log "正在下载 OurAirports 数据..."
curl -L -o "${DATA_DIR}/airports.csv.new" \
    "https://davidmegginson.github.io/ourairports-data/airports.csv" \
    2>> "$LOG_FILE"

# 3. 验证下载
NEW_LINES=$(wc -l < "${DATA_DIR}/airports.csv.new")
if [ "$NEW_LINES" -lt 10000 ]; then
    log "错误: 下载的文件行数过少 ($NEW_LINES)，可能下载失败"
    rm -f "${DATA_DIR}/airports.csv.new"
    exit 1
fi

log "下载成功，共 $NEW_LINES 条记录"

# 4. 替换旧文件
mv "${DATA_DIR}/airports.csv.new" "${DATA_DIR}/airports.csv"

# 5. 执行导入脚本
log "正在导入数据库..."
cd "$SCRIPT_DIR"
go run import_airports.go 2>&1 | tee -a "$LOG_FILE"

# 6. 清理旧备份 (保留最近7个)
cd "$BACKUP_DIR"
ls -t airports_*.csv 2>/dev/null | tail -n +8 | xargs rm -f 2>/dev/null || true

log "=== 机场数据更新完成 ==="
log ""
