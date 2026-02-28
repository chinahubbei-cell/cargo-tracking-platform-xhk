import React, { useState, useEffect } from 'react';
import { Spin, Empty } from 'antd';
import api from '../api/client';
import './DeviceStopRecords.css';

interface DeviceStopRecord {
    id: string;
    device_id: string;
    device_external_id: string;
    shipment_id: string;
    start_time: string;
    end_time: string | null;
    duration_seconds: number;
    duration_text: string;
    latitude: number | null;
    longitude: number | null;
    address: string;
    status: 'active' | 'completed';
    alert_sent: boolean;
    created_at: string;
}

interface TransitCity {
    id: string;
    country: string;
    province: string;
    city: string;
    latitude: number;
    longitude: number;
    entered_at: string;
    is_oversea: boolean;
}

// 统一时间线节点
interface TimelineNode {
    key: string;
    type: 'stop' | 'city';
    timestamp: string;
    data: DeviceStopRecord | TransitCity;
    locationKey: string;
    description: string;
}

interface ParsedLocation {
    country: string;
    province: string;
    city: string;
    label: string;
    key: string;
}

const COUNTRY_ALIASES: Array<{ pattern: RegExp; country: string }> = [
    { pattern: /中国|china|prc|people'?s republic of china/i, country: '中国' },
    { pattern: /美国|united states|usa|america/i, country: '美国' },
    { pattern: /俄罗斯|russia/i, country: '俄罗斯' },
    { pattern: /日本|japan/i, country: '日本' },
    { pattern: /德国|germany/i, country: '德国' },
    { pattern: /法国|france/i, country: '法国' },
];

const hasChinese = (text: string): boolean => /[\u4e00-\u9fa5]/.test(text);

const normalizeCountryName = (country: string): string => {
    const value = country.trim();
    if (!value) return '';
    for (const item of COUNTRY_ALIASES) {
        if (item.pattern.test(value)) {
            return item.country;
        }
    }
    return value;
};

const normalizeProvinceName = (province: string): string => {
    let value = (province || '').trim().replace(/^[,;.\s-]+|[,;.\s-]+$/g, '');
    if (!value) return '';
    if (hasChinese(value)) {
        return value.replace(/\s+/g, '');
    }
    value = value.replace(/\b(State|Province)\b$/i, '').trim();
    value = value.replace(/\s{2,}/g, ' ');
    return toTitleCase(value);
};

const toTitleCase = (text: string): string => {
    return text
        .toLowerCase()
        .replace(/\b[a-z]/g, (ch) => ch.toUpperCase())
        .trim();
};

const normalizeCityName = (city: string): string => {
    let value = city.trim().replace(/^[,;.\s-]+|[,;.\s-]+$/g, '');
    if (!value) return '';

    if (hasChinese(value)) {
        value = value.replace(/\s+/g, '');
        if (!value || value === '市辖区') return '';
        if (value.startsWith('坐标区域(')) return value;

        const cityMatches = value.match(/[\u4e00-\u9fa5]{2,16}(?:自治州|地区|盟|市)/g);
        if (cityMatches && cityMatches.length > 0) {
            return cityMatches[cityMatches.length - 1];
        }
        if (/(?:区|县|旗|省|特别行政区|自治区)$/.test(value)) {
            return value;
        }
        return `${value}市`;
    }

    value = value.replace(/\b(City|District|County|Prefecture|Province|State)\b$/i, '').trim();
    value = value.replace(/\s{2,}/g, ' ');
    return toTitleCase(value);
};

const extractCountryFromText = (text: string): string => {
    const value = text.trim();
    if (!value) return '';
    for (const item of COUNTRY_ALIASES) {
        if (item.pattern.test(value)) {
            return item.country;
        }
    }
    return '';
};

const extractChineseCity = (text: string): string => {
    const value = text.trim();
    if (!value) return '';

    const cityMatch = value.match(/[\u4e00-\u9fa5]{2,12}(?:自治州|地区|盟|市)/);
    if (cityMatch?.[0]) {
        return cityMatch[0];
    }

    const districtMatch = value.match(/[\u4e00-\u9fa5]{2,12}(?:区|县|旗)/);
    return districtMatch?.[0] || '';
};

const extractEnglishCity = (text: string): string => {
    const value = text.trim();
    if (!value) return '';

    const segments = value
        .replace(/[，；;]/g, ',')
        .replace(/[()（）]/g, ' ')
        .split(',')
        .map((segment) => segment.trim())
        .filter(Boolean);

    for (const segment of segments) {
        const cityMatch = segment.match(/([A-Za-z][A-Za-z\s'-]{1,40})\s+City\b/i);
        if (cityMatch?.[1]) {
            return cityMatch[1].trim();
        }
    }

    for (const segment of segments) {
        const districtMatch = segment.match(/([A-Za-z][A-Za-z\s'-]{1,40})\s+(District|County|Prefecture|Province|State)\b/i);
        if (districtMatch?.[1]) {
            return districtMatch[1].trim();
        }
    }

    return '';
};

const resolveNodeLocation = (address: string, fallbackCountry = '', fallbackProvince = '', fallbackCity = ''): ParsedLocation => {
    const text = (address || '').trim();
    const parts = text.split('/');
    const zhPart = (parts[0] || '').trim();
    const enPart = (parts[1] || '').trim();

    let country = normalizeCountryName(fallbackCountry) || extractCountryFromText(`${zhPart} ${enPart}`);
    let province = normalizeProvinceName(fallbackProvince);
    let city = extractChineseCity(zhPart);
    if (!city) city = extractEnglishCity(enPart);
    if (!city) city = extractEnglishCity(zhPart);
    if (!city) city = fallbackCity;

    city = normalizeCityName(city);
    if (!country && city && hasChinese(city)) {
        country = '中国';
    }
    if (!country) {
        country = '未知国家';
    }
    if (!city) {
        city = '未知城市';
    }

    const segments = [country, province, city].filter((segment) => !!segment);
    const label = segments.join('-');
    return {
        country,
        province,
        city,
        label,
        key: `${country.toLowerCase()}|${province.toLowerCase()}|${city.toLowerCase()}`,
    };
};

interface DeviceStopRecordsProps {
    shipmentId: string;
    refreshInterval?: number;
    maxRecords?: number;
    deviceStatus?: {
        isRunning: boolean;
        speed: number;
    } | null;
}

const DeviceStopRecords: React.FC<DeviceStopRecordsProps> = ({
    shipmentId,
    refreshInterval = 30000,
    maxRecords = 100,
}) => {
    const [loading, setLoading] = useState(true);
    const [records, setRecords] = useState<DeviceStopRecord[]>([]);
    const [transitCities, setTransitCities] = useState<TransitCity[]>([]);
    const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
    const [nowTs, setNowTs] = useState(() => Date.now());

    // 加载停留记录
    const loadRecords = async () => {
        if (!shipmentId) return;
        try {
            const response = await api.getShipmentStops(shipmentId, 1, maxRecords);
            const data = response as any;
            let recs: DeviceStopRecord[] = [];
            if (data?.data?.data?.records) {
                recs = data.data.data.records;
            } else if (data?.data?.records) {
                recs = data.data.records;
            } else if (data?.records) {
                recs = data.records;
            }
            setRecords(recs);
        } catch (error) {
            console.error('获取停留记录失败:', error);
            setRecords([]);
        }
    };

    // 加载途经城市
    const loadTransitCities = async () => {
        if (!shipmentId) return;
        try {
            const response = await api.getTransitCities(shipmentId);
            const data = response as any;
            const cities = data?.data || [];
            setTransitCities(cities);
        } catch (error) {
            console.error('获取途经城市失败:', error);
            setTransitCities([]);
        }
    };

    // 初始加载
    useEffect(() => {
        setLoading(true);
        Promise.all([loadRecords(), loadTransitCities()]).finally(() => {
            setLoading(false);
        });
    }, [shipmentId]);

    // 轮询更新运输节点（停留记录 + 途经城市）
    useEffect(() => {
        if (!shipmentId || refreshInterval <= 0) return;
        const interval = setInterval(() => {
            Promise.all([loadRecords(), loadTransitCities()]);
        }, refreshInterval);
        return () => clearInterval(interval);
    }, [shipmentId, refreshInterval]);

    // 活跃停留时长本地实时刷新，避免停留文案“卡住”
    useEffect(() => {
        const timer = setInterval(() => setNowTs(Date.now()), 1000);
        return () => clearInterval(timer);
    }, []);

    // 格式化完整时间
    const formatFullTime = (dateStr: string) => {
        if (!dateStr) return '-';
        const date = new Date(dateStr);
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
        });
    };

    const formatDurationText = (seconds: number): string => {
        if (seconds < 60) return `${seconds}秒`;
        if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`;
        if (seconds < 86400) {
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            return minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`;
        }
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        return hours > 0 ? `${days}天${hours}小时` : `${days}天`;
    };

    const getLiveDurationText = (record: DeviceStopRecord): string => {
        if (record.status !== 'active') {
            return record.duration_text || '未知时长';
        }
        const startMs = new Date(record.start_time).getTime();
        if (!Number.isFinite(startMs)) {
            return record.duration_text || '未知时长';
        }
        const sec = Math.max(0, Math.floor((nowTs - startMs) / 1000));
        return formatDurationText(sec);
    };

    // 合并数据为统一时间线并按时间倒序排列
    // 仅“途径城市”按国家-省份-城市去重；停留节点保留完整双语地址与原始记录粒度
    const timelineNodes: TimelineNode[] = (() => {
        const stopNodes: TimelineNode[] = records
            .filter((record) => !!record.start_time)
            .map((record, index) => {
                const stopAddress = (record.address || '').trim() || '未知位置';
                const duration = getLiveDurationText(record);
                return {
                    key: `stop-${record.id || index}`,
                    type: 'stop',
                    timestamp: record.start_time,
                    data: record,
                    locationKey: `stop:${record.id || index}`,
                    description: `货物在：${stopAddress}停留，时长${duration}`,
                };
            });

        const rawCityNodes: TimelineNode[] = transitCities
            .filter((city) => !!city.entered_at)
            .map((city, index) => {
                const location = resolveNodeLocation('', city.country, city.province, city.city);
                return {
                    key: `city-${city.id || index}`,
                    type: 'city',
                    timestamp: city.entered_at,
                    data: city,
                    locationKey: location.key,
                    description: `货物进入：${location.label}`,
                };
            });

        const cityNodes = rawCityNodes.sort(
            (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
        );

        const dedupedCityNodes: TimelineNode[] = [];
        const seenCityLocation = new Set<string>();
        for (const node of cityNodes) {
            if (seenCityLocation.has(node.locationKey)) {
                continue;
            }
            seenCityLocation.add(node.locationKey);
            dedupedCityNodes.push(node);
        }

        return [...stopNodes, ...dedupedCityNodes].sort(
            (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
        );
    })();

    if (loading && records.length === 0 && transitCities.length === 0) {
        return (
            <div className="transport-nodes-loading">
                <Spin size="small" />
                <span className="transport-nodes-loading-text">加载运输节点信息...</span>
            </div>
        );
    }

    if (timelineNodes.length === 0) {
        return (
            <div className="transport-nodes-empty">
                <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description="暂无运输节点信息"
                />
            </div>
        );
    }

    return (
        <div className="transport-nodes">
            <div className="transport-nodes-header">
                <span className="transport-nodes-icon">📍</span>
                <span className="transport-nodes-title">运输节点信息</span>
                <span className="transport-nodes-count">({timelineNodes.length})</span>
            </div>

            <div className="transport-nodes-timeline">
                {timelineNodes.map((node, index) => {
                    const isStop = node.type === 'stop';
                    const stopRecord = isStop ? (node.data as DeviceStopRecord) : null;
                    const isActive = isStop && stopRecord?.status === 'active';
                    const isLast = index === timelineNodes.length - 1;

                    return (
                        <div
                            key={node.key}
                            className={`timeline-item ${isActive ? 'active' : ''} ${selectedIndex === index ? 'selected' : ''}`}
                            onClick={() => setSelectedIndex(index)}
                        >
                            {/* 左侧图标列 */}
                            <div className="timeline-left">
                                <div className={`timeline-dot ${isStop ? 'stop' : 'city'} ${isActive ? 'active' : ''}`}>
                                    {isStop ? (
                                        <span className="dot-icon">⏸</span>
                                    ) : (
                                        <span className="dot-icon">🌐</span>
                                    )}
                                    {isActive && <div className="dot-pulse" />}
                                </div>
                                {!isLast && <div className="timeline-line" />}
                            </div>

                            {/* 右侧内容 */}
                            <div className="timeline-content">
                                <div className="timeline-desc">{node.description}</div>
                                <div className="timeline-time">
                                    {formatFullTime(node.timestamp)}
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
};

export default DeviceStopRecords;
