-- 设备停留记录表
-- 用于记录设备基于 runStatus 的停留信息

CREATE TABLE IF NOT EXISTS device_stop_records (
    id VARCHAR(50) PRIMARY KEY,
    
    -- 设备和运单关联
    device_external_id VARCHAR(50) NOT NULL,
    device_id VARCHAR(50),
    shipment_id VARCHAR(50),
    
    -- 停留时间信息
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration_seconds INT DEFAULT 0,
    duration_text VARCHAR(50),
    
    -- 停留位置信息
    latitude DECIMAL(10,7),
    longitude DECIMAL(10,7),
    address VARCHAR(500),
    
    -- 停留状态: active=停留中, completed=已结束
    status VARCHAR(20) DEFAULT 'active',
    
    -- 预警状态
    alert_sent BOOLEAN DEFAULT FALSE,
    alert_threshold_hours INT DEFAULT 24,
    
    -- 元数据
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_device_stop_external ON device_stop_records(device_external_id);
CREATE INDEX IF NOT EXISTS idx_device_stop_id ON device_stop_records(device_id);
CREATE INDEX IF NOT EXISTS idx_device_stop_shipment ON device_stop_records(shipment_id);
CREATE INDEX IF NOT EXISTS idx_device_stop_status_time ON device_stop_records(status, start_time);
CREATE INDEX IF NOT EXISTS idx_device_stop_end_time ON device_stop_records(end_time);
