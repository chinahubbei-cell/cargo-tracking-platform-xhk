-- ============================================================
-- TMS 机场主数据表 - PostgreSQL + PostGIS 迁移脚本
-- ============================================================
-- 适用场景: 从 SQLite 升级到 PostgreSQL 时执行
-- 优势: 毫秒级大圆距离计算、空间索引、围栏判定
-- ============================================================

-- 1. 启用 PostGIS 扩展 (处理经纬度最强的工具)
CREATE EXTENSION IF NOT EXISTS postgis;

-- 2. 创建机场主数据表
CREATE TABLE IF NOT EXISTS tms_airport_master (
    id SERIAL PRIMARY KEY,
    
    -- 核心代码
    iata_code CHAR(3) NOT NULL UNIQUE,  -- 例如: HKG
    icao_code CHAR(4),                  -- 例如: VHHH
    
    -- 基础信息
    airport_name VARCHAR(255) NOT NULL,
    airport_name_en VARCHAR(255),
    city_name VARCHAR(100),
    country_code CHAR(2),               -- ISO 3166-1 alpha-2 (例如: CN, US)
    region VARCHAR(50),                 -- 区域 (例如: Asia, North America)
    
    -- 地理位置 (关键)
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    location GEOGRAPHY(POINT, 4326),    -- PostGIS 空间列，自动处理地球曲率
    
    -- 时间与时区 (对跨境ETA计算至关重要)
    timezone VARCHAR(50),               -- 例如: 'Asia/Shanghai', 'America/New_York'
    gmt_offset FLOAT,                   -- 例如: 8.0 或 -5.0
    
    -- 机场属性
    airport_type VARCHAR(20) DEFAULT 'mixed', -- cargo/mixed/passenger
    tier INTEGER DEFAULT 2,             -- 等级: 1=一级枢纽, 2=二级, 3=三级
    geofence_km INTEGER DEFAULT 10,     -- 围栏半径(公里)
    
    -- 运营指标
    is_cargo_hub BOOLEAN DEFAULT FALSE, -- 是否为货运枢纽
    annual_cargo_tons INTEGER,          -- 年货运量(万吨)
    customs_efficiency INTEGER DEFAULT 3, -- 清关效率 1-5
    congestion_level INTEGER DEFAULT 2,   -- 拥堵程度 1-5
    runway_count INTEGER DEFAULT 2,
    cargo_terminals INTEGER DEFAULT 1,
    
    -- 业务标记
    is_active BOOLEAN DEFAULT TRUE,     -- 软删除标记
    
    -- 维护信息
    data_source VARCHAR(50) DEFAULT 'seed', 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. 创建索引 (提升查询速度)
CREATE INDEX IF NOT EXISTS idx_airport_iata ON tms_airport_master(iata_code);
CREATE INDEX IF NOT EXISTS idx_airport_country ON tms_airport_master(country_code);
CREATE INDEX IF NOT EXISTS idx_airport_region ON tms_airport_master(region);
CREATE INDEX IF NOT EXISTS idx_airport_cargo_hub ON tms_airport_master(is_cargo_hub);
-- 创建空间索引 (极速查找附近的机场)
CREATE INDEX IF NOT EXISTS idx_airport_location ON tms_airport_master USING GIST(location);

-- 4. 创建触发器: 自动将经纬度转为 PostGIS 坐标对象
CREATE OR REPLACE FUNCTION update_airport_geometry() RETURNS TRIGGER AS $$
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_airport_update_geo ON tms_airport_master;
CREATE TRIGGER trg_airport_update_geo
BEFORE INSERT OR UPDATE ON tms_airport_master
FOR EACH ROW EXECUTE FUNCTION update_airport_geometry();

-- ============================================================
-- 5. 实用函数: 直接在 SQL 中计算两个机场的距离 (单位: 公里)
-- 调用示例: SELECT calculate_airport_distance_km('HKG', 'LAX');
-- ============================================================
CREATE OR REPLACE FUNCTION calculate_airport_distance_km(code_a TEXT, code_b TEXT)
RETURNS NUMERIC AS $$
DECLARE
    loc_a GEOGRAPHY;
    loc_b GEOGRAPHY;
BEGIN
    SELECT location INTO loc_a FROM tms_airport_master WHERE iata_code = UPPER(code_a);
    SELECT location INTO loc_b FROM tms_airport_master WHERE iata_code = UPPER(code_b);
    
    IF loc_a IS NULL THEN
        RAISE EXCEPTION 'Airport not found: %', code_a;
    END IF;
    IF loc_b IS NULL THEN
        RAISE EXCEPTION 'Airport not found: %', code_b;
    END IF;
    
    -- ST_Distance 返回米，除以1000转为公里
    RETURN ROUND((ST_Distance(loc_a, loc_b) / 1000)::NUMERIC, 2);
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 6. 实用函数: 计算飞行时间估算 (含航路修正系数1.08)
-- 调用示例: SELECT estimate_flight_time('PVG', 'LAX');
-- ============================================================
CREATE OR REPLACE FUNCTION estimate_flight_time(code_a TEXT, code_b TEXT, cruise_speed_kmh NUMERIC DEFAULT 850)
RETURNS TABLE(
    distance_km NUMERIC,
    actual_distance_km NUMERIC,
    flight_hours NUMERIC,
    formatted_duration TEXT
) AS $$
DECLARE
    dist_km NUMERIC;
    actual_km NUMERIC;
    hours NUMERIC;
    h INTEGER;
    m INTEGER;
BEGIN
    -- 计算大圆距离
    dist_km := calculate_airport_distance_km(code_a, code_b);
    
    -- 应用航路修正系数 1.08 (避开禁飞区、顺逆风等)
    actual_km := ROUND(dist_km * 1.08, 2);
    
    -- 计算飞行时间
    hours := ROUND(actual_km / cruise_speed_kmh, 2);
    h := FLOOR(hours);
    m := ROUND((hours - h) * 60);
    
    RETURN QUERY SELECT 
        dist_km,
        actual_km,
        hours,
        CASE 
            WHEN h > 0 THEN h || '小时' || m || '分钟'
            ELSE m || '分钟'
        END;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 7. 实用函数: 查找指定坐标附近的机场
-- 调用示例: SELECT * FROM find_nearby_airports(31.2, 121.5, 200);
-- ============================================================
CREATE OR REPLACE FUNCTION find_nearby_airports(
    lat NUMERIC, 
    lon NUMERIC, 
    radius_km NUMERIC DEFAULT 100
)
RETURNS TABLE(
    iata_code CHAR(3),
    airport_name VARCHAR(255),
    distance_km NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        a.iata_code,
        a.airport_name,
        ROUND((ST_Distance(
            a.location, 
            ST_SetSRID(ST_MakePoint(lon, lat), 4326)::GEOGRAPHY
        ) / 1000)::NUMERIC, 2) AS distance_km
    FROM tms_airport_master a
    WHERE ST_DWithin(
        a.location,
        ST_SetSRID(ST_MakePoint(lon, lat), 4326)::GEOGRAPHY,
        radius_km * 1000  -- 转换为米
    )
    AND a.is_active = TRUE
    ORDER BY distance_km;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 8. 实用函数: 检查坐标是否在机场围栏内
-- 调用示例: SELECT is_within_airport_geofence(22.31, 113.92, 'HKG');
-- ============================================================
CREATE OR REPLACE FUNCTION is_within_airport_geofence(
    lat NUMERIC, 
    lon NUMERIC, 
    airport_code TEXT
)
RETURNS BOOLEAN AS $$
DECLARE
    airport_loc GEOGRAPHY;
    geofence_radius INTEGER;
    point_loc GEOGRAPHY;
    distance_m NUMERIC;
BEGIN
    -- 获取机场位置和围栏半径
    SELECT location, geofence_km INTO airport_loc, geofence_radius
    FROM tms_airport_master 
    WHERE iata_code = UPPER(airport_code);
    
    IF airport_loc IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- 创建检测点
    point_loc := ST_SetSRID(ST_MakePoint(lon, lat), 4326)::GEOGRAPHY;
    
    -- 计算距离
    distance_m := ST_Distance(airport_loc, point_loc);
    
    -- 判断是否在围栏内
    RETURN distance_m <= (geofence_radius * 1000);
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 9. 机场围栏表 (可选 - 用于更复杂的围栏形状)
-- ============================================================
CREATE TABLE IF NOT EXISTS tms_airport_geofence (
    id SERIAL PRIMARY KEY,
    airport_code CHAR(3) NOT NULL REFERENCES tms_airport_master(iata_code),
    airport_name VARCHAR(255),
    city VARCHAR(100),
    country VARCHAR(50),
    
    -- 圆形围栏
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    radius_km INTEGER DEFAULT 10,
    
    -- PostGIS 几何
    center_point GEOGRAPHY(POINT, 4326),
    geofence_circle GEOGRAPHY(POLYGON, 4326),  -- 可存储复杂多边形
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_geofence_airport ON tms_airport_geofence(airport_code);
CREATE INDEX IF NOT EXISTS idx_geofence_center ON tms_airport_geofence USING GIST(center_point);

-- 触发器: 自动创建围栏几何
CREATE OR REPLACE FUNCTION update_geofence_geometry() RETURNS TRIGGER AS $$
BEGIN
    NEW.center_point = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    -- 创建圆形围栏 (使用 ST_Buffer)
    NEW.geofence_circle = ST_Buffer(NEW.center_point, NEW.radius_km * 1000);
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_geofence_update_geo ON tms_airport_geofence;
CREATE TRIGGER trg_geofence_update_geo
BEFORE INSERT OR UPDATE ON tms_airport_geofence
FOR EACH ROW EXECUTE FUNCTION update_geofence_geometry();

-- ============================================================
-- 使用示例
-- ============================================================
/*
-- 计算两机场距离
SELECT calculate_airport_distance_km('HKG', 'LAX');
-- 结果: 11654.23 (公里)

-- 估算飞行时间
SELECT * FROM estimate_flight_time('PVG', 'LAX');
-- 结果: distance_km=10456.78, actual_distance_km=11293.32, flight_hours=13.29, formatted_duration='13小时17分钟'

-- 查找上海100公里内的机场
SELECT * FROM find_nearby_airports(31.2, 121.5, 100);
-- 结果: PVG (15.2km), SHA (30.5km)...

-- 检查货物是否到达香港机场
SELECT is_within_airport_geofence(22.3085, 113.9190, 'HKG');
-- 结果: TRUE
*/

-- ============================================================
-- 10. 城市-机场映射表 (City Mapping)
-- ============================================================
-- 业务场景: 客户说"发货到伦敦"，系统需要知道伦敦有多个机场
-- 用于: 智能推荐最佳机场、报价时自动匹配
-- ============================================================
CREATE TABLE IF NOT EXISTS tms_city_airport_mapping (
    id SERIAL PRIMARY KEY,
    
    -- 城市信息
    city_name VARCHAR(100) NOT NULL,           -- 标准城市名 (英文)
    city_name_cn VARCHAR(100),                 -- 中文城市名
    city_name_local VARCHAR(100),              -- 当地语言名
    country_code CHAR(2) NOT NULL,             -- ISO 3166-1 alpha-2
    
    -- 关联机场
    iata_code CHAR(3) NOT NULL REFERENCES tms_airport_master(iata_code),
    
    -- 优先级与属性
    is_primary BOOLEAN DEFAULT FALSE,          -- 是否为该城市主要机场
    priority INTEGER DEFAULT 1,                -- 推荐优先级 (1=最高)
    distance_to_center_km NUMERIC(6,2),        -- 距市中心距离
    typical_transfer_time_min INTEGER,         -- 典型转运时间(分钟)
    
    -- 货运属性
    cargo_capacity VARCHAR(20) DEFAULT 'medium', -- high/medium/low
    customs_speed VARCHAR(20) DEFAULT 'normal',  -- fast/normal/slow
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(city_name, iata_code)
);

CREATE INDEX IF NOT EXISTS idx_city_mapping_city ON tms_city_airport_mapping(city_name);
CREATE INDEX IF NOT EXISTS idx_city_mapping_city_cn ON tms_city_airport_mapping(city_name_cn);
CREATE INDEX IF NOT EXISTS idx_city_mapping_country ON tms_city_airport_mapping(country_code);

-- ============================================================
-- 11. 城市机场查询函数
-- 调用示例: SELECT * FROM get_airports_for_city('London');
-- ============================================================
CREATE OR REPLACE FUNCTION get_airports_for_city(search_city TEXT)
RETURNS TABLE(
    iata_code CHAR(3),
    airport_name VARCHAR(255),
    is_primary BOOLEAN,
    priority INTEGER,
    distance_to_center_km NUMERIC,
    is_cargo_hub BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.iata_code,
        a.airport_name,
        m.is_primary,
        m.priority,
        m.distance_to_center_km,
        a.is_cargo_hub
    FROM tms_city_airport_mapping m
    JOIN tms_airport_master a ON m.iata_code = a.iata_code
    WHERE (
        LOWER(m.city_name) = LOWER(search_city) OR
        m.city_name_cn = search_city OR
        LOWER(m.city_name_local) = LOWER(search_city)
    )
    AND m.is_active = TRUE
    AND a.is_active = TRUE
    ORDER BY m.priority, m.distance_to_center_km;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 12. 示例数据: 多机场城市映射
-- ============================================================
INSERT INTO tms_city_airport_mapping (city_name, city_name_cn, country_code, iata_code, is_primary, priority, distance_to_center_km, cargo_capacity, customs_speed) VALUES
-- 伦敦 (5个机场)
('London', '伦敦', 'GB', 'LHR', TRUE, 1, 24.0, 'high', 'normal'),
('London', '伦敦', 'GB', 'LGW', FALSE, 2, 45.0, 'medium', 'normal'),
('London', '伦敦', 'GB', 'STN', FALSE, 3, 55.0, 'high', 'fast'),  -- Stansted 是货运枢纽
('London', '伦敦', 'GB', 'LTN', FALSE, 4, 56.0, 'low', 'normal'),
('London', '伦敦', 'GB', 'LCY', FALSE, 5, 11.0, 'low', 'fast'),

-- 纽约 (3个机场)
('New York', '纽约', 'US', 'JFK', TRUE, 1, 24.0, 'high', 'normal'),
('New York', '纽约', 'US', 'EWR', FALSE, 2, 18.0, 'high', 'normal'),
('New York', '纽约', 'US', 'LGA', FALSE, 3, 13.0, 'low', 'normal'),

-- 东京 (2个机场)
('Tokyo', '东京', 'JP', 'NRT', TRUE, 1, 60.0, 'high', 'fast'),  -- 成田主要货运
('Tokyo', '东京', 'JP', 'HND', FALSE, 2, 18.0, 'medium', 'fast'),

-- 上海 (2个机场)
('Shanghai', '上海', 'CN', 'PVG', TRUE, 1, 30.0, 'high', 'fast'),
('Shanghai', '上海', 'CN', 'SHA', FALSE, 2, 13.0, 'low', 'normal'),

-- 北京 (2个机场)
('Beijing', '北京', 'CN', 'PEK', TRUE, 1, 32.0, 'high', 'normal'),
('Beijing', '北京', 'CN', 'PKX', FALSE, 2, 46.0, 'high', 'fast'),  -- 大兴机场

-- 首尔 (2个机场)
('Seoul', '首尔', 'KR', 'ICN', TRUE, 1, 52.0, 'high', 'fast'),
('Seoul', '首尔', 'KR', 'GMP', FALSE, 2, 15.0, 'low', 'fast'),

-- 迪拜 (2个机场)
('Dubai', '迪拜', 'AE', 'DXB', TRUE, 1, 4.0, 'high', 'fast'),
('Dubai', '迪拜', 'AE', 'DWC', FALSE, 2, 37.0, 'high', 'fast'),  -- Al Maktoum 货运枢纽

-- 洛杉矶 (多机场)
('Los Angeles', '洛杉矶', 'US', 'LAX', TRUE, 1, 27.0, 'high', 'normal'),
('Los Angeles', '洛杉矶', 'US', 'ONT', FALSE, 2, 56.0, 'medium', 'fast')

ON CONFLICT (city_name, iata_code) DO NOTHING;

-- ============================================================
-- 使用示例: 城市映射查询
-- ============================================================
/*
-- 查询伦敦所有机场
SELECT * FROM get_airports_for_city('London');
-- 结果:
-- LHR | London Heathrow | true  | 1 | 24.0 | true
-- LGW | London Gatwick  | false | 2 | 45.0 | false
-- STN | London Stansted | false | 3 | 55.0 | true
-- ...

-- 查询东京机场 (中文)
SELECT * FROM get_airports_for_city('东京');

-- 智能推荐: 获取城市的主要货运机场
SELECT m.iata_code, a.airport_name, a.annual_cargo_tons
FROM tms_city_airport_mapping m
JOIN tms_airport_master a ON m.iata_code = a.iata_code
WHERE m.city_name = 'London' 
AND m.cargo_capacity = 'high'
ORDER BY a.annual_cargo_tons DESC
LIMIT 1;
*/

