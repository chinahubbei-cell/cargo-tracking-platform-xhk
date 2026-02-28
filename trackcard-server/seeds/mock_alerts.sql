
DELETE FROM alerts WHERE id LIKE 'alert-sim-%';

INSERT INTO alerts (id, shipment_id, type, severity, title, message, category, status, created_at, device_id)
VALUES
-- 1. 物理环境类 (Physical)
('alert-sim-01', (SELECT id FROM shipments LIMIT 1), 'temp_high', 'warning', '高温预警', '当前温度 42.5°C 超过阈值 40.0°C (药品柜)', 'physical', 'pending', datetime('now'), (SELECT device_id FROM shipments LIMIT 1)),
('alert-sim-02', (SELECT id FROM shipments LIMIT 1), 'shock_detected', 'critical', '剧烈震动', '检测到 5.2g 震动，疑似跌落 (精密仪器)', 'physical', 'pending', datetime('now', '-1 hour'), (SELECT device_id FROM shipments LIMIT 1)),

-- 2. 节点异动类 (Node)
('alert-sim-03', (SELECT id FROM shipments LIMIT 1), 'eta_delay', 'warning', 'ETA严重延误', '原预计到达 2026-01-15，已超期 3 天', 'node', 'pending', datetime('now', '-2 hour'), NULL),
('alert-sim-04', (SELECT id FROM shipments LIMIT 1), 'route_deviation', 'critical', '航线异常', '船舶偏离预定航线，挂靠港口取消 (Red Sea Detour)', 'node', 'pending', datetime('now', '-3 hour'), NULL),

-- 3. 末端作业与关务类 (Operation)
('alert-sim-05', (SELECT id FROM shipments LIMIT 1), 'customs_hold', 'warning', '海关查验滞留', '货物处于查验状态已超过 52 小时，请提交补充文件', 'operation', 'pending', datetime('now', '-30 minute'), NULL),
('alert-sim-06', (SELECT id FROM shipments LIMIT 1), 'free_time_expiring', 'critical', '免租期即将到期', '免堆期仅剩 4 小时，请立即安排提货避免滞期费', 'operation', 'pending', datetime('now', '-10 minute'), NULL);
