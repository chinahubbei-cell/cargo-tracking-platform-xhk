-- ============================================================
-- 性能优化数据库索引
-- 执行时间: 2026-01-30
-- ============================================================

-- 1. Shipments 表索引优化
-- ============================================================

-- 组织+状态 复合索引 (常用于列表过滤)
CREATE INDEX IF NOT EXISTS idx_shipments_org_status 
ON shipments(org_id, status);

-- 状态+创建时间 复合索引 (列表排序)
CREATE INDEX IF NOT EXISTS idx_shipments_status_created 
ON shipments(status, created_at DESC);

-- 设备+状态 复合索引 (设备关联查询)
CREATE INDEX IF NOT EXISTS idx_shipments_device_status 
ON shipments(device_id, status);

-- 提单号索引 (搜索优化)
CREATE INDEX IF NOT EXISTS idx_shipments_bill_of_lading 
ON shipments(bill_of_lading);

-- 箱号索引 (搜索优化)
CREATE INDEX IF NOT EXISTS idx_shipments_container_no 
ON shipments(container_no);

-- 运输类型索引
CREATE INDEX IF NOT EXISTS idx_shipments_transport_type 
ON shipments(transport_type);

-- 2. Device Tracks 表索引优化 (轨迹查询核心)
-- ============================================================

-- 设备+时间 复合索引 (轨迹查询核心)
CREATE INDEX IF NOT EXISTS idx_device_tracks_device_time 
ON device_tracks(device_id, locate_time DESC);

-- 时间索引 (时间范围查询)
CREATE INDEX IF NOT EXISTS idx_device_tracks_locate_time 
ON device_tracks(locate_time DESC);

-- 3. Alerts 表索引优化
-- ============================================================

-- 运单+状态 复合索引
CREATE INDEX IF NOT EXISTS idx_alerts_shipment_status 
ON alerts(shipment_id, status);

-- 状态+创建时间 复合索引
CREATE INDEX IF NOT EXISTS idx_alerts_status_created 
ON alerts(status, created_at DESC);

-- 类型索引
CREATE INDEX IF NOT EXISTS idx_alerts_type 
ON alerts(type);

-- 4. Shipment Stages 表索引优化
-- ============================================================

-- 运单+阶段顺序 复合索引
CREATE INDEX IF NOT EXISTS idx_shipment_stages_shipment_order 
ON shipment_stages(shipment_id, stage_order);

-- 状态索引
CREATE INDEX IF NOT EXISTS idx_shipment_stages_status 
ON shipment_stages(status);

-- 5. Ports 表索引优化
-- ============================================================

-- 国家索引
CREATE INDEX IF NOT EXISTS idx_ports_country 
ON ports(country);

-- 类型索引
CREATE INDEX IF NOT EXISTS idx_ports_type 
ON ports(type);

-- 6. Airports 表索引优化
-- ============================================================

-- 国家索引
CREATE INDEX IF NOT EXISTS idx_airports_country 
ON airports(country);

-- 是否货运枢纽
CREATE INDEX IF NOT EXISTS idx_airports_cargo_hub 
ON airports(is_cargo_hub);

-- 7. Shipment Logs 表索引优化
-- ============================================================

-- 运单+创建时间 复合索引
CREATE INDEX IF NOT EXISTS idx_shipment_logs_shipment_created 
ON shipment_logs(shipment_id, created_at DESC);

-- 8. Users 表索引优化
-- ============================================================

-- 邮箱索引 (登录查询)
CREATE INDEX IF NOT EXISTS idx_users_email 
ON users(email);

-- 9. 验证索引创建
-- ============================================================
-- SQLite: 使用 .indexes 命令查看
-- MySQL: SHOW INDEX FROM table_name;
