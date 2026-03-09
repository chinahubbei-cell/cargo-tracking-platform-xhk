import React, { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import { Card, Table, Select, Tag, Space, Button, message, Modal, Descriptions, Statistic, Row, Col, AutoComplete, Input, Tabs, Spin } from 'antd';
import { SearchOutlined, RocketOutlined, GlobalOutlined, CompassOutlined, ReloadOutlined, RadarChartOutlined } from '@ant-design/icons';
import { MapContainer, TileLayer, Marker, Popup, useMap, useMapEvents, Polyline, ZoomControl, Circle } from 'react-leaflet';
import L from 'leaflet';
import axios from 'axios';
import 'leaflet/dist/leaflet.css';

// 机场类型定义
interface Airport {
    id: string;
    iata_code: string;
    icao_code: string;
    name: string;
    name_en: string;
    city: string;
    country: string;
    region: string;
    type: string;
    tier: number;
    latitude: number;
    longitude: number;
    timezone: string;
    geofence_km: number;
    is_cargo_hub: boolean;
    customs_efficiency: number;
    congestion_level: number;
    runway_count: number;
    cargo_terminals: number;
    annual_cargo_tons: number;
}

// 围栏类型
interface AirportGeofence {
    id: string;
    airport_code: string;
    airport_name: string;
    city: string;
    country: string;
    longitude: number;
    latitude: number;
    radius_km: number;
    is_active: boolean;
}

// 缓存机场图标 - 避免重复创建
const iconCache = new Map<string, L.DivIcon>();

const createAirportIcon = (tier: number, isCargoHub: boolean) => {
    const key = `${tier}-${isCargoHub}`;
    if (iconCache.has(key)) {
        return iconCache.get(key)!;
    }

    const color = tier === 1 ? '#1890ff' : tier === 2 ? '#52c41a' : '#faad14';
    const size = tier === 1 ? 14 : tier === 2 ? 11 : 9;

    const icon = L.divIcon({
        className: 'airport-marker',
        html: `<div style="
            width: ${size}px;
            height: ${size}px;
            background: ${color};
            border: 2px solid white;
            border-radius: 50%;
            box-shadow: 0 2px 6px rgba(0,0,0,0.4);
            ${isCargoHub ? 'animation: pulse 2s infinite;' : ''}
        "><div style="
            position: absolute;
            top: -8px;
            left: 50%;
            transform: translateX(-50%);
            font-size: 10px;
        ">✈️</div></div>`,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
    });

    iconCache.set(key, icon);
    return icon;
};

// 视口内机场过滤组件
interface ViewportFilterProps {
    airports: Airport[];
    geofences: AirportGeofence[];
    showGeofences: boolean;
    onVisibleAirportsChange: (airports: Airport[]) => void;
    onVisibleGeofencesChange: (geofences: AirportGeofence[]) => void;
}

const ViewportFilter: React.FC<ViewportFilterProps> = ({
    airports,
    geofences,
    showGeofences,
    onVisibleAirportsChange,
    onVisibleGeofencesChange
}) => {
    const map = useMap();
    const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const filterByViewport = useCallback(() => {
        if (debounceRef.current) {
            clearTimeout(debounceRef.current);
        }

        debounceRef.current = setTimeout(() => {
            const bounds = map.getBounds();
            const zoom = map.getZoom();

            // 过滤可见机场
            const visible = airports.filter(a =>
                bounds.contains([a.latitude, a.longitude])
            );

            // 限制渲染数量，低缩放级别只显示主要机场，但始终显示包含围栏的机场
            let filtered = visible;
            if (zoom < 4) {
                filtered = visible.filter(a => a.tier === 1 || geofences.some(g => g.airport_code === a.iata_code));
            } else if (zoom < 6) {
                filtered = visible.filter(a => a.tier <= 2 || geofences.some(g => g.airport_code === a.iata_code));
            }
            // zoom >= 6 显示所有

            // 最大渲染500个标记
            if (filtered.length > 500) {
                filtered = filtered.slice(0, 500);
            }

            onVisibleAirportsChange(filtered);

            // 围栏：在围栏专用tab时始终显示，否则只在高缩放级别显示
            if (showGeofences) {
                // 过滤视口内的围栏
                const visibleGeofences = geofences.filter(g =>
                    bounds.contains([g.latitude, g.longitude])
                );
                // 低缩放级别严格限制数量以防止卡顿，高缩放级别显示更多
                const maxGeofences = zoom < 3 ? 20 : zoom < 5 ? 50 : zoom < 7 ? 100 : 150;
                onVisibleGeofencesChange(visibleGeofences.slice(0, maxGeofences));
            } else {
                onVisibleGeofencesChange([]);
            }
        }, 150); // 150ms 防抖
    }, [map, airports, geofences, showGeofences, onVisibleAirportsChange, onVisibleGeofencesChange]);

    useMapEvents({
        moveend: filterByViewport,
        zoomend: filterByViewport,
    });

    useEffect(() => {
        filterByViewport();
    }, [airports, geofences, showGeofences]);

    return null;
};

// 地图自适应组件 - 仅首次加载时使用
const MapAutoFit: React.FC<{ airports: Airport[], shouldFit: boolean }> = ({ airports, shouldFit }) => {
    const map = useMap();
    const hasFitted = useRef(false);

    useEffect(() => {
        if (shouldFit && airports.length > 0 && !hasFitted.current) {
            // 只取部分点来计算边界，避免处理太多数据
            const sample = airports.filter(a => a.tier === 1).slice(0, 50);
            if (sample.length > 0) {
                const bounds = L.latLngBounds(sample.map(a => [a.latitude, a.longitude]));
                map.fitBounds(bounds, { padding: [30, 30] });
            }
            hasFitted.current = true;
        }
    }, [airports, shouldFit, map]);

    return null;
};

// 地图聚焦控制器 - 点击表格行时聚焦到机场
interface MapFocusControllerProps {
    airport: Airport | null;
    onFocused: () => void;
    resetToGlobal?: boolean;
}

const MapFocusController: React.FC<MapFocusControllerProps> = ({ airport, onFocused, resetToGlobal }) => {
    const map = useMap();

    useEffect(() => {
        if (resetToGlobal) {
            // 重置到全球视图
            map.setView([20, 0], 2, { animate: true });
            return;
        }
        if (airport) {
            // 使用 flyTo 飞到目标位置，zoom 设为 12 以获得更好的视图
            map.flyTo([airport.latitude, airport.longitude], 12, {
                duration: 1.5,
                animate: true,
            });

            // 通知父组件（由父组件处理弹窗打开）
            const timer = setTimeout(() => {
                onFocused();
            }, 1600);

            return () => clearTimeout(timer);
        }
    }, [airport, map, onFocused, resetToGlobal]);

    return null;
};

const GlobalAirports: React.FC = () => {
    // 表格数据 (分页)
    const [tableAirports, setTableAirports] = useState<Airport[]>([]);
    const [tableTotal, setTableTotal] = useState(0);
    const [currentPage, setCurrentPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);

    // 地图数据 (全量用于过滤)
    const [mapAirports, setMapAirports] = useState<Airport[]>([]);
    const [visibleAirports, setVisibleAirports] = useState<Airport[]>([]);
    const [visibleGeofences, setVisibleGeofences] = useState<AirportGeofence[]>([]);

    const [loading, setLoading] = useState(false);
    const [mapLoading, setMapLoading] = useState(false);
    const [searchText, setSearchText] = useState('');
    const [regionFilter, setRegionFilter] = useState<string>('');
    const [tierFilter, setTierFilter] = useState<number | null>(null);
    const [geofenceFilter, setGeofenceFilter] = useState<string>('');
    const [geofences, setGeofences] = useState<AirportGeofence[]>([]);
    const [geofenceCodes, setGeofenceCodes] = useState<Set<string>>(new Set());
    const [regions, setRegions] = useState<string[]>([]);
    const [selectedAirport, setSelectedAirport] = useState<Airport | null>(null);
    const [distanceModalVisible, setDistanceModalVisible] = useState(false);
    const [fromAirport, setFromAirport] = useState<string>('');
    const [toAirport, setToAirport] = useState<string>('');
    const [distanceResult, setDistanceResult] = useState<{ km: number; hours: number; duration: string } | null>(null);
    const [distanceLine, setDistanceLine] = useState<[number, number][] | null>(null);
    const [showGeofences, setShowGeofences] = useState(true);
    const [stats, setStats] = useState({ total: 0, hubs: 0, cargoHubs: 0 });
    const [geofenceLoading, setGeofenceLoading] = useState(false);
    const [focusedAirport, setFocusedAirport] = useState<Airport | null>(null);
    const [shouldResetMap, setShouldResetMap] = useState(false);
    const [geofenceMapKey, setGeofenceMapKey] = useState(0);
    const markerRefs = useRef<{ [key: string]: L.Marker | null }>({});

    // 区域名称中英文映射
    const regionNameMap: Record<string, string> = {
        'Asia': '亚洲',
        'East Asia': '东亚',
        'Southeast Asia': '东南亚',
        'South Asia': '南亚',
        'Europe': '欧洲',
        'North America': '北美',
        'Middle East': '中东',
        'South America': '南美',
        'Africa': '非洲',
        'Oceania': '大洋洲',
        'Other': '其他',
    };

    // 加载表格数据 (分页)
    const loadTableAirports = useCallback(async () => {
        setLoading(true);
        try {
            const token = localStorage.getItem('token');
            const params = new URLSearchParams();
            params.append('page', currentPage.toString());
            params.append('page_size', pageSize.toString());
            if (searchText) params.append('search', searchText);
            if (regionFilter) params.append('region', regionFilter);
            if (tierFilter) params.append('tier', tierFilter.toString());

            const res = await axios.get(`/api/airports?${params.toString()}`, {
                headers: { Authorization: `Bearer ${token}` }
            });
            console.log('✈️ Airports API Response:', res.data);
            if (res.data.success || res.data.data) {
                let data = res.data.data || [];
                console.log('✈️ Airports data before filter:', data.length);

                // 围栏筛选 (前端过滤)
                if (geofenceFilter === 'created') {
                    data = data.filter((a: Airport) => geofenceCodes.has(a.iata_code));
                } else if (geofenceFilter === 'not_created') {
                    data = data.filter((a: Airport) => !geofenceCodes.has(a.iata_code));
                }

                console.log('✈️ Airports data after filter:', data.length);
                setTableAirports(data);
                setTableTotal(res.data.total || data.length);
            } else {
                console.error('❌ Airports API failed:', res.data);
            }
        } catch (error) {
            message.error('加载机场数据失败');
        } finally {
            setLoading(false);
        }
    }, [currentPage, pageSize, searchText, regionFilter, tierFilter, geofenceFilter, geofenceCodes]);

    // 加载地图数据 (只加载 tier 1 和 tier 2 用于地图显示)
    const loadMapAirports = useCallback(async (showMessage = false) => {
        setMapLoading(true);
        try {
            const token = localStorage.getItem('token');
            // 获取所有机场数据以确保地图上能显示(约4500个)
            const res = await axios.get('/api/airports?page_size=10000', {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data.success || res.data.data) {
                const data = res.data.data || [];
                setMapAirports(data);

                // 统计信息
                setStats({
                    total: res.data.total || data.length,
                    hubs: data.filter((a: Airport) => a.tier === 1).length,
                    cargoHubs: data.filter((a: Airport) => a.is_cargo_hub).length,
                });

                if (showMessage) {
                    message.success(`已加载 ${data.length} 个机场`);
                }
            }
        } catch (error) {
            console.error('Failed to load map airports');
            if (showMessage) {
                message.error('加载机场数据失败');
            }
        } finally {
            setMapLoading(false);
        }
    }, []);

    // 加载区域列表
    const loadRegions = async () => {
        try {
            const token = localStorage.getItem('token');
            const res = await axios.get('/api/airports/regions', {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data.success || res.data.data) {
                setRegions(res.data.data || []);
            }
        } catch (error) {
            console.error('Failed to load regions');
        }
    };

    // 加载机场围栏
    const loadGeofences = async () => {
        setGeofenceLoading(true);
        try {
            const token = localStorage.getItem('token');
            const res = await axios.get('/api/airport-geofences', {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data?.data) {
                setGeofences(res.data.data);
                const codes = new Set<string>(res.data.data.map((g: AirportGeofence) => g.airport_code));
                setGeofenceCodes(codes);
            }
        } catch (error) {
            console.error('Failed to load airport geofences');
        } finally {
            setGeofenceLoading(false);
        }
    };

    // 计算距离
    const calculateDistance = async () => {
        if (!fromAirport || !toAirport) {
            message.warning('请选择起始和目的机场');
            return;
        }
        try {
            const token = localStorage.getItem('token');
            const res = await axios.get(`/api/airports/distance?from=${fromAirport}&to=${toAirport}`, {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data.success || res.data.data) {
                setDistanceResult({
                    km: res.data.data.distance_km,
                    hours: res.data.data.estimated_hours,
                    duration: res.data.data.estimated_duration
                });
                const from = res.data.data.from;
                const to = res.data.data.to;
                setDistanceLine([[from.latitude, from.longitude], [to.latitude, to.longitude]]);
            }
        } catch (error) {
            message.error('计算距离失败');
        }
    };

    // 初始加载 - 并行请求优化
    useEffect(() => {
        const loadInitialData = async () => {
            await Promise.all([
                loadMapAirports(),
                loadRegions(),
                loadGeofences()
            ]);
        };
        loadInitialData();
    }, []);

    // 表格数据加载
    useEffect(() => {
        loadTableAirports();
    }, [loadTableAirports]);

    // 搜索机场选项 (用于距离计算下拉框)
    const airportOptions = useMemo(() => {
        return mapAirports.map(a => ({
            value: a.iata_code,
            label: `${a.name} (${a.iata_code})`
        }));
    }, [mapAirports]);

    // 搜索建议
    const searchOptions = useMemo(() => {
        if (!searchText) return [];
        return mapAirports
            .filter(a =>
                a.name.toLowerCase().includes(searchText.toLowerCase()) ||
                a.name_en.toLowerCase().includes(searchText.toLowerCase()) ||
                a.iata_code.toLowerCase().includes(searchText.toLowerCase()) ||
                a.city.toLowerCase().includes(searchText.toLowerCase())
            )
            .slice(0, 8)
            .map(a => ({
                value: a.name,
                label: (
                    <div>
                        <Tag color="blue" style={{ marginRight: 4 }}>{a.iata_code}</Tag>
                        {a.name} / {a.city}
                    </div>
                )
            }));
    }, [searchText, mapAirports]);

    // 表格列
    const columns = [
        {
            title: '序号',
            key: 'index',
            width: 60,
            render: (_: unknown, __: Airport, index: number) => (currentPage - 1) * pageSize + index + 1,
        },
        {
            title: 'IATA',
            dataIndex: 'iata_code',
            key: 'iata_code',
            width: 80,
            render: (code: string, record: Airport) => (
                <Tag
                    color="blue"
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                        setFocusedAirport(record);
                        message.info(`正在定位到 ${record.name}...`);
                    }}
                >
                    {code}
                </Tag>
            ),
        },
        {
            title: '机场名称',
            key: 'name',
            width: 220,
            render: (_: unknown, record: Airport) => (
                <Space
                    direction="vertical"
                    size={0}
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                        setFocusedAirport(record);
                        message.info(`正在定位到 ${record.name}...`);
                    }}
                >
                    <span style={{ fontWeight: 'bold', color: '#1890ff' }}>{record.name}</span>
                    <span style={{ color: '#1890ff', fontSize: 12 }}>{record.name_en}</span>
                </Space>
            ),
        },
        {
            title: '城市',
            dataIndex: 'city',
            key: 'city',
            width: 80,
        },
        {
            title: '国家',
            dataIndex: 'country',
            key: 'country',
            width: 60,
        },
        {
            title: '区域',
            dataIndex: 'region',
            key: 'region',
            width: 100,
            render: (region: string) => <Tag>{regionNameMap[region] || region}</Tag>,
        },
        {
            title: '等级',
            dataIndex: 'tier',
            key: 'tier',
            width: 100,
            render: (tier: number) => {
                const colors = { 1: 'gold', 2: 'green', 3: 'default' };
                const labels = { 1: '✈️ 枢纽机场', 2: '区域机场', 3: '支线机场' };
                return <Tag color={colors[tier as keyof typeof colors]}>{labels[tier as keyof typeof labels]}</Tag>;
            },
        },
        {
            title: '货运枢纽',
            dataIndex: 'is_cargo_hub',
            key: 'is_cargo_hub',
            width: 80,
            render: (v: boolean) => v ? <Tag color="purple">是</Tag> : <Tag>否</Tag>,
        },
        {
            title: '年货运量',
            dataIndex: 'annual_cargo_tons',
            key: 'annual_cargo_tons',
            width: 100,
            render: (v: number) => v ? `${v}万吨` : '-',
        },
        {
            title: '电子围栏',
            key: 'geofence',
            width: 90,
            render: (_: unknown, record: Airport) => {
                const hasGeofence = geofenceCodes.has(record.iata_code);
                return hasGeofence
                    ? <Tag color="green">已创建</Tag>
                    : <Tag color="default">未创建</Tag>;
            },
        },
        {
            title: '操作',
            key: 'action',
            width: 80,
            render: (_: unknown, record: Airport) => (
                <Button type="link" size="small" onClick={() => setSelectedAirport(record)}>
                    详情
                </Button>
            ),
        },
    ];

    return (
        <div className="global-airports-page">
            <Tabs
                defaultActiveKey="airports"
                tabBarExtraContent={
                    <Space size="small">
                        <Tag color="blue"><RocketOutlined /> 机场 {stats.total}</Tag>
                        <Tag color="cyan">枢纽 {stats.hubs}</Tag>
                        <Tag color="purple">货运 {stats.cargoHubs}</Tag>
                        <Tag color="green">围栏 {geofenceCodes.size}</Tag>
                        <Tag>区域 {regions.length}</Tag>
                    </Space>
                }
                items={[
                    {
                        key: 'airports',
                        label: <><RocketOutlined /> 机场列表</>,
                        children: (
                            <>
                                {/* 世界地图 */}
                                <Card
                                    title={<><GlobalOutlined /> 全球货运机场分布</>}
                                    size="small"
                                    extra={
                                        <Space>
                                            <Button
                                                type={showGeofences ? 'primary' : 'default'}
                                                icon={<RadarChartOutlined />}
                                                onClick={() => setShowGeofences(!showGeofences)}
                                            >
                                                {showGeofences ? '隐藏围栏' : '显示围栏'}
                                            </Button>
                                            <Button
                                                icon={<CompassOutlined />}
                                                onClick={() => setDistanceModalVisible(true)}
                                            >
                                                计算飞行距离
                                            </Button>
                                            <Button icon={<ReloadOutlined />} onClick={() => {
                                                setFocusedAirport(null);
                                                setShouldResetMap(true);
                                                // 重置所有筛选状态
                                                setSearchText('');
                                                setRegionFilter('');
                                                setTierFilter(null);
                                                setGeofenceFilter('');
                                                setCurrentPage(1);

                                                setTimeout(() => setShouldResetMap(false), 100);
                                                loadMapAirports(true);
                                                loadGeofences();
                                            }} loading={mapLoading}>
                                                刷新
                                            </Button>
                                        </Space>
                                    }
                                >
                                    <Spin spinning={mapLoading}>
                                        <div style={{ height: 350, borderRadius: 8, overflow: 'hidden' }}>
                                            <MapContainer
                                                center={[20, 0]}
                                                zoom={2}
                                                style={{ height: '100%', width: '100%' }}
                                                zoomControl={false}
                                            >
                                                <ZoomControl position="topright" />
                                                <TileLayer
                                                    url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
                                                    attribution='&copy; <a href="https://carto.com/">CARTO</a>'
                                                />
                                                <MapAutoFit airports={mapAirports} shouldFit={mapAirports.length > 0} />

                                                {/* 视口过滤器 */}
                                                <ViewportFilter
                                                    airports={mapAirports}
                                                    geofences={geofences}
                                                    showGeofences={showGeofences}
                                                    onVisibleAirportsChange={setVisibleAirports}
                                                    onVisibleGeofencesChange={setVisibleGeofences}
                                                />

                                                {/* 地图聚焦控制器 - 点击表格行时飞行到目标机场 */}
                                                <MapFocusController
                                                    airport={focusedAirport}
                                                    resetToGlobal={shouldResetMap}
                                                    onFocused={() => {
                                                        // 动画完成后清除 focusedAirport 状态，允许用户自由操作地图
                                                        setFocusedAirport(null);
                                                    }}
                                                />

                                                {/* 显示围栏圆圈 (仅可见的) */}
                                                {visibleGeofences.map(gf => (
                                                    <Circle
                                                        key={gf.airport_code}
                                                        center={[gf.latitude, gf.longitude]}
                                                        radius={gf.radius_km * 1000}
                                                        pathOptions={{
                                                            color: '#1890ff',
                                                            fillColor: '#1890ff',
                                                            fillOpacity: 0.15,
                                                            weight: 1,
                                                        }}
                                                    />
                                                ))}

                                                {/* 显示机场标记 (仅可见的 + 聚焦的) */}
                                                {[...visibleAirports, ...(focusedAirport ? [focusedAirport] : [])].map(airport => {
                                                    // 避免重复渲染
                                                    const key = airport.iata_code;
                                                    const isFocused = focusedAirport?.iata_code === key;

                                                    return (
                                                        <Marker
                                                            key={key}
                                                            position={[airport.latitude, airport.longitude]}
                                                            icon={createAirportIcon(airport.tier, airport.is_cargo_hub)}
                                                            ref={(ref) => {
                                                                if (ref) {
                                                                    markerRefs.current[key] = ref;
                                                                }
                                                            }}
                                                        >
                                                            {isFocused && (
                                                                <span style={{
                                                                    position: 'absolute',
                                                                    top: -30,
                                                                    left: '50%',
                                                                    transform: 'translateX(-50%)',
                                                                    background: '#ff4d4f',
                                                                    color: 'white',
                                                                    padding: '2px 8px',
                                                                    borderRadius: '4px',
                                                                    fontSize: '12px',
                                                                    fontWeight: 'bold',
                                                                    whiteSpace: 'nowrap',
                                                                    zIndex: 1000
                                                                }}>
                                                                    🔍 当前选中
                                                                </span>
                                                            )}
                                                            <Popup>
                                                                <div style={{ minWidth: 180 }}>
                                                                    <strong style={{ fontSize: 14 }}>{airport.name}</strong>
                                                                    <span style={{ color: '#666', marginLeft: 4 }}>({airport.iata_code})</span>
                                                                    <br />
                                                                    <span style={{ color: '#888', fontSize: 12 }}>{airport.name_en}</span>
                                                                    <br />
                                                                    <span>{airport.city}, {airport.country} | {regionNameMap[airport.region] || airport.region}</span>
                                                                    <br />
                                                                    <span style={{ color: airport.tier === 1 ? '#faad14' : '#52c41a' }}>
                                                                        {airport.tier === 1 ? '✈️ 枢纽机场' : airport.tier === 2 ? '区域机场' : '支线机场'}
                                                                    </span>
                                                                    {airport.is_cargo_hub && <Tag color="purple" style={{ marginLeft: 4 }}>货运枢纽</Tag>}
                                                                    {airport.annual_cargo_tons > 0 && <><br />年货运量: {airport.annual_cargo_tons}万吨</>}
                                                                </div>
                                                            </Popup>
                                                        </Marker>
                                                    );
                                                })}

                                                {distanceLine && (
                                                    <Polyline
                                                        positions={distanceLine}
                                                        color="#ff4d4f"
                                                        weight={2}
                                                        dashArray="5, 10"
                                                    />
                                                )}
                                            </MapContainer>
                                        </div>
                                    </Spin>
                                </Card>

                                {/* 机场列表 */}
                                <Card
                                    title="机场列表"
                                    size="small"
                                    style={{ marginTop: 16 }}
                                    extra={
                                        <Space>
                                            <AutoComplete
                                                style={{ width: 200 }}
                                                options={searchOptions}
                                                value={searchText}
                                                onChange={v => setSearchText(v)}
                                                onSelect={v => { setSearchText(v); setCurrentPage(1); }}
                                            >
                                                <Input
                                                    placeholder="搜索机场代码/名称/城市..."
                                                    prefix={<SearchOutlined />}
                                                    onPressEnter={() => setCurrentPage(1)}
                                                />
                                            </AutoComplete>
                                            <Select
                                                placeholder="区域筛选"
                                                style={{ width: 140 }}
                                                value={regionFilter || 'all'}
                                                onChange={v => { setRegionFilter(v === 'all' ? '' : v); setCurrentPage(1); }}
                                            >
                                                <Select.Option value="all">全部区域</Select.Option>
                                                {regions.map(r => <Select.Option key={r} value={r}>{regionNameMap[r] || r}</Select.Option>)}
                                            </Select>
                                            <Select
                                                placeholder="等级筛选"
                                                style={{ width: 130 }}
                                                value={tierFilter || 'all'}
                                                onChange={v => { setTierFilter(v === 'all' ? null : v as number); setCurrentPage(1); }}
                                            >
                                                <Select.Option value="all">全部等级</Select.Option>
                                                <Select.Option value={1}>✈️ 枢纽机场</Select.Option>
                                                <Select.Option value={2}>区域机场</Select.Option>
                                                <Select.Option value={3}>支线机场</Select.Option>
                                            </Select>
                                            <Select
                                                placeholder="围栏筛选"
                                                style={{ width: 120 }}
                                                value={geofenceFilter || 'all'}
                                                onChange={v => { setGeofenceFilter(v === 'all' ? '' : v); setCurrentPage(1); }}
                                            >
                                                <Select.Option value="all">全部围栏</Select.Option>
                                                <Select.Option value="created">已创建</Select.Option>
                                                <Select.Option value="not_created">未创建</Select.Option>
                                            </Select>
                                        </Space>
                                    }
                                >
                                    <Table
                                        dataSource={tableAirports}
                                        columns={columns}
                                        rowKey="iata_code"
                                        size="small"
                                        loading={loading}
                                        scroll={{ x: 1200, y: 'calc(100vh - 620px)' }}
                                        pagination={{
                                            current: currentPage,
                                            pageSize: pageSize,
                                            total: tableTotal,
                                            showTotal: total => `共 ${total} 个机场`,
                                            showSizeChanger: true,
                                            pageSizeOptions: ['10', '20', '50'],
                                            onChange: (page, size) => {
                                                setCurrentPage(page);
                                                setPageSize(size);
                                            },
                                        }}
                                    />
                                </Card>

                                {/* 机场详情弹窗 */}
                                <Modal
                                    title={`机场详情 - ${selectedAirport?.name}`}
                                    open={!!selectedAirport}
                                    onCancel={() => setSelectedAirport(null)}
                                    footer={null}
                                    width={700}
                                >
                                    {selectedAirport && (
                                        <Descriptions bordered column={2} size="small">
                                            <Descriptions.Item label="IATA代码">{selectedAirport.iata_code}</Descriptions.Item>
                                            <Descriptions.Item label="ICAO代码">{selectedAirport.icao_code}</Descriptions.Item>
                                            <Descriptions.Item label="英文名">{selectedAirport.name_en}</Descriptions.Item>
                                            <Descriptions.Item label="所在城市">{selectedAirport.city}</Descriptions.Item>
                                            <Descriptions.Item label="国家">{selectedAirport.country}</Descriptions.Item>
                                            <Descriptions.Item label="区域">{regionNameMap[selectedAirport.region] || selectedAirport.region}</Descriptions.Item>
                                            <Descriptions.Item label="类型">{selectedAirport.type === 'cargo' ? '货运机场' : selectedAirport.type === 'mixed' ? '客货两用' : '客运机场'}</Descriptions.Item>
                                            <Descriptions.Item label="等级">
                                                {selectedAirport.tier === 1 ? '✈️ 枢纽机场' : selectedAirport.tier === 2 ? '区域机场' : '支线机场'}
                                            </Descriptions.Item>
                                            <Descriptions.Item label="经度">{selectedAirport.longitude.toFixed(4)}</Descriptions.Item>
                                            <Descriptions.Item label="纬度">{selectedAirport.latitude.toFixed(4)}</Descriptions.Item>
                                            <Descriptions.Item label="时区">{selectedAirport.timezone || '-'}</Descriptions.Item>
                                            <Descriptions.Item label="电子围栏">{selectedAirport.geofence_km} km</Descriptions.Item>
                                            <Descriptions.Item label="货运枢纽">{selectedAirport.is_cargo_hub ? '是' : '否'}</Descriptions.Item>
                                            <Descriptions.Item label="清关效率">{selectedAirport.customs_efficiency}/5</Descriptions.Item>
                                            <Descriptions.Item label="跑道数量">{selectedAirport.runway_count}</Descriptions.Item>
                                            <Descriptions.Item label="货运航站楼">{selectedAirport.cargo_terminals}</Descriptions.Item>
                                            <Descriptions.Item label="年货运量" span={2}>{selectedAirport.annual_cargo_tons ? `${selectedAirport.annual_cargo_tons}万吨` : '-'}</Descriptions.Item>
                                        </Descriptions>
                                    )}
                                </Modal>

                                {/* 飞行距离计算弹窗 */}
                                <Modal
                                    title={<><CompassOutlined /> 机场飞行距离计算</>}
                                    open={distanceModalVisible}
                                    onCancel={() => { setDistanceModalVisible(false); setDistanceResult(null); setDistanceLine(null); }}
                                    footer={null}
                                    width={500}
                                >
                                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                                        <Space>
                                            <Select
                                                placeholder="起飞机场"
                                                style={{ width: 180 }}
                                                showSearch
                                                filterOption={(input, option) =>
                                                    (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                                                }
                                                options={airportOptions}
                                                value={fromAirport || undefined}
                                                onChange={setFromAirport}
                                            />
                                            <span>✈️→</span>
                                            <Select
                                                placeholder="到达机场"
                                                style={{ width: 180 }}
                                                showSearch
                                                filterOption={(input, option) =>
                                                    (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                                                }
                                                options={airportOptions}
                                                value={toAirport || undefined}
                                                onChange={setToAirport}
                                            />
                                        </Space>
                                        <Button type="primary" block onClick={calculateDistance}>
                                            计算飞行距离
                                        </Button>
                                        {distanceResult && (
                                            <Row gutter={16}>
                                                <Col span={12}>
                                                    <Statistic
                                                        title="直线距离"
                                                        value={distanceResult.km.toLocaleString()}
                                                        suffix="公里"
                                                        valueStyle={{ color: '#1890ff' }}
                                                    />
                                                </Col>
                                                <Col span={12}>
                                                    <Statistic
                                                        title="预计飞行时间"
                                                        value={distanceResult.duration}
                                                        valueStyle={{ color: '#52c41a' }}
                                                    />
                                                </Col>
                                            </Row>
                                        )}
                                    </Space>
                                </Modal>

                                <style>{`
                                    @keyframes pulse {
                                        0% { box-shadow: 0 0 0 0 rgba(24, 144, 255, 0.7); }
                                        70% { box-shadow: 0 0 0 10px rgba(24, 144, 255, 0); }
                                        100% { box-shadow: 0 0 0 0 rgba(24, 144, 255, 0); }
                                    }
                                `}</style>
                            </>
                        )
                    },
                    {
                        key: 'geofences',
                        label: <><RadarChartOutlined /> 机场围栏</>,
                        children: (
                            <Card
                                title={<><RadarChartOutlined /> 全球机场围栏分布</>}
                                size="small"
                                extra={
                                    <Button icon={<ReloadOutlined />} onClick={() => {
                                        setGeofenceMapKey(prev => prev + 1);
                                        loadGeofences();
                                    }} loading={geofenceLoading}>
                                        刷新
                                    </Button>
                                }
                            >
                                <div style={{ height: 'calc(100vh - 220px)', borderRadius: 8, overflow: 'hidden' }}>
                                    <MapContainer
                                        key={geofenceMapKey}
                                        center={[20, 0]}
                                        zoom={2}
                                        style={{ height: '100%', width: '100%' }}
                                        zoomControl={false}
                                    >
                                        <ZoomControl position="topright" />
                                        <TileLayer
                                            url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
                                            attribution='&copy; <a href="https://carto.com/">CARTO</a>'
                                        />

                                        {/* 围栏Tab也使用视口过滤 */}
                                        <ViewportFilter
                                            airports={[]}
                                            geofences={geofences}
                                            showGeofences={true}
                                            onVisibleAirportsChange={() => { }}
                                            onVisibleGeofencesChange={setVisibleGeofences}
                                        />

                                        {visibleGeofences.map(gf => (
                                            <React.Fragment key={gf.airport_code}>
                                                <Circle
                                                    center={[gf.latitude, gf.longitude]}
                                                    radius={gf.radius_km * 1000}
                                                    pathOptions={{
                                                        color: '#722ed1',
                                                        fillColor: '#722ed1',
                                                        fillOpacity: 0.2,
                                                        weight: 2,
                                                        interactive: false,
                                                    }}
                                                />
                                                <Marker
                                                    position={[gf.latitude, gf.longitude]}
                                                    zIndexOffset={1000}
                                                    icon={L.divIcon({
                                                        className: 'geofence-marker',
                                                        html: '<div style="font-size: 20px;">✈️</div>',
                                                        iconSize: [24, 24],
                                                        iconAnchor: [12, 12],
                                                    })}
                                                >
                                                    <Popup>
                                                        <div>
                                                            <strong>{gf.airport_name}</strong> ({gf.airport_code})<br />
                                                            {gf.city}, {gf.country}<br />
                                                            围栏半径: {gf.radius_km} km
                                                        </div>
                                                    </Popup>
                                                </Marker>
                                            </React.Fragment>
                                        ))}
                                    </MapContainer>
                                </div>
                            </Card>
                        )
                    }
                ]}
            />
        </div>
    );
};

export default GlobalAirports;
