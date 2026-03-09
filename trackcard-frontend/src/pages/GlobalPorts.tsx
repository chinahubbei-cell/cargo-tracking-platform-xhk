import React, { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import { Card, Table, Select, Tag, Space, Button, message, Modal, Descriptions, Statistic, Row, Col, AutoComplete, Input, Tabs, Spin } from 'antd';
import { SearchOutlined, EnvironmentOutlined, GlobalOutlined, CompassOutlined, ReloadOutlined, RadarChartOutlined } from '@ant-design/icons';
import PortGeofenceMap from '../components/PortGeofenceMap';
import { MapContainer, TileLayer, Marker, Popup, useMap, useMapEvents, Polyline, ZoomControl } from 'react-leaflet';
import L from 'leaflet';
import axios from 'axios';
import 'leaflet/dist/leaflet.css';

// 港口类型定义
interface Port {
    id: string;
    code: string;
    name: string;
    name_en: string;
    country: string;
    region: string;
    type: string;
    tier: number;
    latitude: number;
    longitude: number;
    timezone: string;
    geofence_km: number;
    is_transit_hub: boolean;
    customs_efficiency: number;
    congestion_level: number;
}

// 缓存港口图标 - 避免重复创建
const portIconCache = new Map<string, L.DivIcon>();

const createPortIcon = (tier: number, isTransitHub: boolean) => {
    const key = `port-${tier}-${isTransitHub}`;
    if (portIconCache.has(key)) {
        return portIconCache.get(key)!;
    }

    const color = tier === 1 ? '#1890ff' : tier === 2 ? '#52c41a' : '#faad14';
    const size = tier === 1 ? 12 : tier === 2 ? 10 : 8;

    const icon = L.divIcon({
        className: 'port-marker',
        html: `<div style="
            width: ${size}px;
            height: ${size}px;
            background: ${color};
            border: 2px solid white;
            border-radius: 50%;
            box-shadow: 0 2px 6px rgba(0,0,0,0.4);
            ${isTransitHub ? 'animation: pulse 2s infinite;' : ''}
        "></div>`,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
    });

    portIconCache.set(key, icon);
    return icon;
};

// 视口内港口过滤组件
interface ViewportFilterProps {
    ports: Port[];
    geofenceCodes: Set<string>;
    onVisiblePortsChange: (ports: Port[]) => void;
}

const ViewportFilter: React.FC<ViewportFilterProps> = ({ ports, geofenceCodes, onVisiblePortsChange }) => {
    const map = useMap();
    const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const filterByViewport = useCallback(() => {
        if (debounceRef.current) {
            clearTimeout(debounceRef.current);
        }

        debounceRef.current = setTimeout(() => {
            const bounds = map.getBounds();
            const zoom = map.getZoom();

            // 过滤可见港口
            const visible = ports.filter(p =>
                bounds.contains([p.latitude, p.longitude])
            );

            // 限制渲染数量，低缩放级别只显示主要港口，但始终显示包含围栏的港口
            let filtered = visible;
            if (zoom < 4) {
                // 只显示枢纽港，或有围栏的港口
                filtered = visible.filter(p => p.tier === 1 || geofenceCodes.has(p.code));
            } else if (zoom < 6) {
                // 显示枢纽+干线，或有围栏的港口
                filtered = visible.filter(p => p.tier <= 2 || geofenceCodes.has(p.code));
            }

            // 最大渲染200个标记
            if (filtered.length > 200) {
                filtered = filtered.slice(0, 200);
            }

            onVisiblePortsChange(filtered);
        }, 150); // 150ms 防抖
    }, [map, ports, onVisiblePortsChange]);

    useMapEvents({
        moveend: filterByViewport,
        zoomend: filterByViewport,
    });

    useEffect(() => {
        filterByViewport();
    }, [ports]);

    return null;
};

// 地图自适应组件
const MapAutoFit: React.FC<{ ports: Port[], shouldFit: boolean }> = ({ ports, shouldFit }) => {
    const map = useMap();
    const hasFitted = useRef(false);

    useEffect(() => {
        if (shouldFit && ports.length > 0 && !hasFitted.current) {
            const sample = ports.filter(p => p.tier === 1).slice(0, 30);
            if (sample.length > 0) {
                const bounds = L.latLngBounds(sample.map(p => [p.latitude, p.longitude]));
                map.fitBounds(bounds, { padding: [30, 30] });
            }
            hasFitted.current = true;
        }
    }, [ports, shouldFit, map]);

    return null;
};

// 地图聚焦控制器 - 点击表格行时聚焦到港口
interface MapFocusControllerProps {
    port: Port | null;
    onFocused: () => void;
    resetToGlobal?: boolean;
}

const MapFocusController: React.FC<MapFocusControllerProps> = ({ port, onFocused, resetToGlobal }) => {
    const map = useMap();

    useEffect(() => {
        if (resetToGlobal) {
            // 重置到全球视图
            map.setView([20, 0], 2, { animate: true });
            return;
        }
        if (port) {
            // 使用 flyTo 飞到目标位置，zoom 设为 12 以获得更好的视图
            map.flyTo([port.latitude, port.longitude], 12, {
                duration: 1.5,
                animate: true,
            });

            // 通知父组件（由父组件处理弹窗打开）
            const timer = setTimeout(() => {
                onFocused();
            }, 1600);

            return () => clearTimeout(timer);
        }
    }, [port, map, onFocused, resetToGlobal]);

    return null;
};

const GlobalPorts: React.FC = () => {
    // 表格数据 (分页)
    const [tablePorts, setTablePorts] = useState<Port[]>([]);
    const [tableTotal, setTableTotal] = useState(0);
    const [currentPage, setCurrentPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);

    // 地图数据
    const [mapPorts, setMapPorts] = useState<Port[]>([]);
    const [visiblePorts, setVisiblePorts] = useState<Port[]>([]);

    const [loading, setLoading] = useState(false);
    const [mapLoading, setMapLoading] = useState(false);
    const [searchText, setSearchText] = useState('');
    const [regionFilter, setRegionFilter] = useState<string>('');
    const [tierFilter, setTierFilter] = useState<number | null>(null);
    const [geofenceFilter, setGeofenceFilter] = useState<string>('');
    const [geofenceCodes, setGeofenceCodes] = useState<Set<string>>(new Set());
    const [regions, setRegions] = useState<string[]>([]);
    const [selectedPort, setSelectedPort] = useState<Port | null>(null);
    const [distanceModalVisible, setDistanceModalVisible] = useState(false);
    const [fromPort, setFromPort] = useState<string>('');
    const [toPort, setToPort] = useState<string>('');
    const [distanceResult, setDistanceResult] = useState<{ km: number, nm: number } | null>(null);
    const [distanceLine, setDistanceLine] = useState<[number, number][] | null>(null);
    const [geofenceRefreshKey, setGeofenceRefreshKey] = useState(0);
    const [stats, setStats] = useState({ total: 0, hubs: 0, transitHubs: 0 });
    const [focusedPort, setFocusedPort] = useState<Port | null>(null);
    const [shouldResetMap, setShouldResetMap] = useState(false);
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
    const loadTablePorts = useCallback(async () => {
        setLoading(true);
        try {
            const token = localStorage.getItem('token');
            const params = new URLSearchParams();
            params.append('page', currentPage.toString());
            params.append('page_size', pageSize.toString());
            if (searchText) params.append('search', searchText);
            if (regionFilter) params.append('region', regionFilter);
            if (tierFilter) params.append('tier', tierFilter.toString());

            const res = await axios.get(`/api/ports?${params.toString()}`, {
                headers: { Authorization: `Bearer ${token}` }
            });
            console.log('📦 Ports API Response:', res.data);
            if (res.data.code === 200 || res.data.data) {
                let data = res.data.data || [];
                console.log('📦 Ports data before filter:', data.length);

                // 围栏筛选 (前端过滤)
                if (geofenceFilter === 'created') {
                    data = data.filter((p: Port) => geofenceCodes.has(p.code));
                } else if (geofenceFilter === 'not_created') {
                    data = data.filter((p: Port) => !geofenceCodes.has(p.code));
                }

                console.log('📦 Ports data after filter:', data.length);
                setTablePorts(data);
                setTableTotal(res.data.total || data.length);
            } else {
                console.error('❌ Ports API failed:', res.data);
            }
        } catch (error) {
            message.error('加载港口数据失败');
        } finally {
            setLoading(false);
        }
    }, [currentPage, pageSize, searchText, regionFilter, tierFilter, geofenceFilter, geofenceCodes]);

    // 加载地图数据
    const loadMapPorts = useCallback(async () => {
        setMapLoading(true);
        try {
            const token = localStorage.getItem('token');
            // 获取所有港口数据以确保地图上能显示(约5000个)
            const res = await axios.get('/api/ports?page_size=10000', {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data.code === 200 || res.data.data) {
                const data = res.data.data || [];
                setMapPorts(data);

                setStats({
                    total: res.data.total || data.length,
                    hubs: data.filter((p: Port) => p.tier === 1).length,
                    transitHubs: data.filter((p: Port) => p.is_transit_hub).length,
                });
            }
        } catch (error) {
            console.error('Failed to load map ports');
        } finally {
            setMapLoading(false);
        }
    }, []);

    // 加载区域列表
    const loadRegions = async () => {
        try {
            const token = localStorage.getItem('token');
            const res = await axios.get('/api/ports/regions', {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data.code === 200 || res.data.data) {
                setRegions(res.data.data || []);
            }
        } catch (error) {
            console.error('Failed to load regions');
        }
    };

    // 加载已创建围栏的港口代码
    const loadGeofenceCodes = async () => {
        try {
            const token = localStorage.getItem('token');
            const res = await axios.get('/api/port-geofences', {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data?.data) {
                const codes = new Set<string>(res.data.data.map((g: { code: string }) => g.code));
                setGeofenceCodes(codes);
            }
        } catch (error) {
            console.error('Failed to load geofence codes');
        }
    };

    // 计算距离
    const calculateDistance = async () => {
        if (!fromPort || !toPort) {
            message.warning('请选择起始和目的港口');
            return;
        }
        try {
            const token = localStorage.getItem('token');
            const res = await axios.get(`/api/ports/distance?from=${fromPort}&to=${toPort}`, {
                headers: { Authorization: `Bearer ${token}` }
            });
            if (res.data.code === 200 || res.data.data) {
                setDistanceResult({
                    km: res.data.data.distance_km,
                    nm: res.data.data.distance_nm
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
                loadMapPorts(),
                loadRegions(),
                loadGeofenceCodes()
            ]);
        };
        loadInitialData();
    }, []);

    // 表格数据加载
    useEffect(() => {
        loadTablePorts();
    }, [loadTablePorts]);

    // 港口选项 (用于距离计算下拉框)
    const portOptions = useMemo(() => {
        return mapPorts.map(p => ({
            value: p.code,
            label: `${p.name} (${p.code})`
        }));
    }, [mapPorts]);

    // 搜索建议
    const searchOptions = useMemo(() => {
        if (!searchText) return [];
        return mapPorts
            .filter(p =>
                p.name.toLowerCase().includes(searchText.toLowerCase()) ||
                p.name_en.toLowerCase().includes(searchText.toLowerCase()) ||
                p.code.toLowerCase().includes(searchText.toLowerCase())
            )
            .slice(0, 8)
            .map(p => ({
                value: p.name,
                label: (
                    <div>
                        <Tag color="blue" style={{ marginRight: 4 }}>{p.code}</Tag>
                        {p.name} / {p.name_en}
                    </div>
                )
            }));
    }, [searchText, mapPorts]);

    // 表格列
    const columns = [
        {
            title: '序号',
            key: 'index',
            width: 60,
            render: (_: unknown, __: Port, index: number) => (currentPage - 1) * pageSize + index + 1,
        },
        {
            title: 'Code',
            dataIndex: 'code',
            key: 'code',
            width: 90,
            render: (code: string, record: Port) => (
                <Tag
                    color="blue"
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                        setFocusedPort(record);
                        message.info(`正在定位到 ${record.name}...`);
                    }}
                >
                    {code}
                </Tag>
            ),
        },
        {
            title: '港口名称',
            key: 'name',
            width: 200,
            render: (_: unknown, record: Port) => (
                <Space
                    direction="vertical"
                    size={0}
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                        setFocusedPort(record);
                        message.info(`正在定位到 ${record.name}...`);
                    }}
                >
                    <span style={{ fontWeight: 'bold', color: '#1890ff' }}>{record.name}</span>
                    <span style={{ color: '#1890ff', fontSize: 12 }}>{record.name_en}</span>
                </Space>
            ),
        },
        {
            title: '国家',
            dataIndex: 'country',
            key: 'country',
            width: 70,
        },
        {
            title: '区域',
            dataIndex: 'region',
            key: 'region',
            width: 120,
            render: (region: string) => <Tag>{regionNameMap[region] || region}</Tag>,
        },
        {
            title: '等级',
            dataIndex: 'tier',
            key: 'tier',
            width: 80,
            render: (tier: number) => {
                const colors = { 1: 'gold', 2: 'green', 3: 'default' };
                const labels = { 1: '⭐ 枢纽港', 2: '干线港', 3: '支线港' };
                return <Tag color={colors[tier as keyof typeof colors]}>{labels[tier as keyof typeof labels]}</Tag>;
            },
        },
        {
            title: '中转枢纽',
            dataIndex: 'is_transit_hub',
            key: 'is_transit_hub',
            width: 80,
            render: (v: boolean) => v ? <Tag color="purple">是</Tag> : <Tag>否</Tag>,
        },
        {
            title: '电子围栏',
            key: 'geofence',
            width: 90,
            render: (_: unknown, record: Port) => {
                const hasGeofence = geofenceCodes.has(record.code);
                return hasGeofence
                    ? <Tag color="green">已创建</Tag>
                    : <Tag color="default">未创建</Tag>;
            },
        },
        {
            title: '坐标',
            key: 'coords',
            width: 140,
            render: (_: unknown, record: Port) => (
                <span style={{ fontSize: 11, color: '#666' }}>
                    {record.latitude.toFixed(4)}, {record.longitude.toFixed(4)}
                </span>
            ),
        },
        {
            title: '操作',
            key: 'action',
            width: 80,
            render: (_: unknown, record: Port) => (
                <Button type="link" size="small" onClick={() => setSelectedPort(record)}>
                    详情
                </Button>
            ),
        },
    ];

    return (
        <div className="global-ports-page">
            <Tabs
                defaultActiveKey="ports"
                tabBarExtraContent={
                    <Space size="small">
                        <Tag color="blue"><EnvironmentOutlined /> 港口 {stats.total}</Tag>
                        <Tag color="cyan">枢纽港 {stats.hubs}</Tag>
                        <Tag color="purple">中转 {stats.transitHubs}</Tag>
                        <Tag>区域 {regions.length}</Tag>
                    </Space>
                }
                items={[
                    {
                        key: 'ports',
                        label: <><GlobalOutlined /> 港口列表</>,
                        children: (
                            <>
                                {/* 世界地图 */}
                                <Card
                                    title={<><GlobalOutlined /> 全球核心港口分布</>}
                                    size="small"
                                    extra={
                                        <Space>
                                            <Button
                                                icon={<CompassOutlined />}
                                                onClick={() => setDistanceModalVisible(true)}
                                            >
                                                计算距离
                                            </Button>
                                            <Button icon={<ReloadOutlined />} onClick={() => {
                                                setFocusedPort(null);
                                                setShouldResetMap(true);
                                                // 重置所有筛选状态
                                                setSearchText('');
                                                setRegionFilter('');
                                                setTierFilter(null);
                                                setGeofenceFilter('');
                                                setCurrentPage(1);

                                                setTimeout(() => setShouldResetMap(false), 100);
                                                loadMapPorts();
                                                loadGeofenceCodes();
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
                                                <MapAutoFit ports={mapPorts} shouldFit={mapPorts.length > 0} />

                                                {/* 视口过滤器 */}
                                                <ViewportFilter
                                                    ports={mapPorts}
                                                    geofenceCodes={geofenceCodes}
                                                    onVisiblePortsChange={setVisiblePorts}
                                                />

                                                {/* 地图聚焦控制器 - 点击表格行时飞行到目标港口 */}
                                                <MapFocusController
                                                    port={focusedPort}
                                                    resetToGlobal={shouldResetMap}
                                                    onFocused={() => {
                                                        // 动画完成后清除 focusedPort 状态，允许用户自由操作地图
                                                        setFocusedPort(null);
                                                    }}
                                                />

                                                {/* 显示港口标记 (仅可见的 + 聚焦的) */}
                                                {[...visiblePorts, ...(focusedPort ? [focusedPort] : [])].map(port => {
                                                    // 避免重复渲染
                                                    const key = port.code;
                                                    const isFocused = focusedPort?.code === key;

                                                    return (
                                                        <Marker
                                                            key={key}
                                                            position={[port.latitude, port.longitude]}
                                                            icon={createPortIcon(port.tier, port.is_transit_hub)}
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
                                                                    <strong style={{ fontSize: 14 }}>{port.name}</strong>
                                                                    <span style={{ color: '#666', marginLeft: 4 }}>({port.code})</span>
                                                                    <br />
                                                                    <span style={{ color: '#888', fontSize: 12 }}>{port.name_en}</span>
                                                                    <br />
                                                                    <span>{regionNameMap[port.region] || port.region} | {port.country}</span>
                                                                    <br />
                                                                    <span style={{ color: port.tier === 1 ? '#faad14' : '#52c41a' }}>
                                                                        {port.tier === 1 ? '⭐ 枢纽港' : port.tier === 2 ? '干线港' : '支线港'}
                                                                    </span>
                                                                    {port.is_transit_hub && <Tag color="purple" style={{ marginLeft: 4 }}>中转枢纽</Tag>}
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

                                {/* 港口列表 */}
                                <Card
                                    title="港口列表"
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
                                                    placeholder="搜索港口代码/名称..."
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
                                                style={{ width: 120 }}
                                                value={tierFilter || 'all'}
                                                onChange={v => { setTierFilter(v === 'all' ? null : v as number); setCurrentPage(1); }}
                                            >
                                                <Select.Option value="all">全部等级</Select.Option>
                                                <Select.Option value={1}>⭐ 枢纽港</Select.Option>
                                                <Select.Option value={2}>干线港</Select.Option>
                                                <Select.Option value={3}>支线港</Select.Option>
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
                                        dataSource={tablePorts}
                                        columns={columns}
                                        rowKey="code"
                                        size="small"
                                        loading={loading}
                                        scroll={{ x: 1100, y: 'calc(100vh - 620px)' }}
                                        pagination={{
                                            current: currentPage,
                                            pageSize: pageSize,
                                            total: tableTotal,
                                            showTotal: total => `共 ${total} 个港口`,
                                            showSizeChanger: true,
                                            pageSizeOptions: ['10', '20', '50'],
                                            onChange: (page, size) => {
                                                setCurrentPage(page);
                                                setPageSize(size);
                                            },
                                        }}
                                    />
                                </Card>

                                {/* 港口详情弹窗 */}
                                <Modal
                                    title={`港口详情 - ${selectedPort?.name}`}
                                    open={!!selectedPort}
                                    onCancel={() => setSelectedPort(null)}
                                    footer={null}
                                    width={600}
                                >
                                    {selectedPort && (
                                        <Descriptions bordered column={2} size="small">
                                            <Descriptions.Item label="港口代码">{selectedPort.code}</Descriptions.Item>
                                            <Descriptions.Item label="英文名">{selectedPort.name_en}</Descriptions.Item>
                                            <Descriptions.Item label="国家">{selectedPort.country}</Descriptions.Item>
                                            <Descriptions.Item label="区域">{selectedPort.region}</Descriptions.Item>
                                            <Descriptions.Item label="类型">{selectedPort.type}</Descriptions.Item>
                                            <Descriptions.Item label="等级">
                                                {selectedPort.tier === 1 ? '⭐ 枢纽港' : selectedPort.tier === 2 ? '干线港' : '支线港'}
                                            </Descriptions.Item>
                                            <Descriptions.Item label="经度">{selectedPort.longitude}</Descriptions.Item>
                                            <Descriptions.Item label="纬度">{selectedPort.latitude}</Descriptions.Item>
                                            <Descriptions.Item label="时区">{selectedPort.timezone || '-'}</Descriptions.Item>
                                            <Descriptions.Item label="电子围栏">{selectedPort.geofence_km} km</Descriptions.Item>
                                            <Descriptions.Item label="中转枢纽">{selectedPort.is_transit_hub ? '是' : '否'}</Descriptions.Item>
                                            <Descriptions.Item label="清关效率">{selectedPort.customs_efficiency}/5</Descriptions.Item>
                                        </Descriptions>
                                    )}
                                </Modal>

                                {/* 距离计算弹窗 */}
                                <Modal
                                    title={<><CompassOutlined /> 港口距离计算</>}
                                    open={distanceModalVisible}
                                    onCancel={() => { setDistanceModalVisible(false); setDistanceResult(null); setDistanceLine(null); }}
                                    footer={null}
                                    width={500}
                                >
                                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                                        <Space>
                                            <Select
                                                placeholder="起始港口"
                                                style={{ width: 180 }}
                                                showSearch
                                                filterOption={(input, option) =>
                                                    (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                                                }
                                                options={portOptions}
                                                value={fromPort || undefined}
                                                onChange={setFromPort}
                                            />
                                            <span>→</span>
                                            <Select
                                                placeholder="目的港口"
                                                style={{ width: 180 }}
                                                showSearch
                                                filterOption={(input, option) =>
                                                    (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                                                }
                                                options={portOptions}
                                                value={toPort || undefined}
                                                onChange={setToPort}
                                            />
                                        </Space>
                                        <Button type="primary" block onClick={calculateDistance}>
                                            计算距离
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
                                                        title="海里距离"
                                                        value={distanceResult.nm.toLocaleString()}
                                                        suffix="NM"
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
                        label: <><RadarChartOutlined /> 港口围栏</>,
                        children: (
                            <Card
                                title={<><RadarChartOutlined /> 全球港口围栏分布</>}
                                size="small"
                                extra={
                                    <Button icon={<ReloadOutlined />} onClick={() => setGeofenceRefreshKey(prev => prev + 1)}>
                                        刷新
                                    </Button>
                                }
                            >
                                <PortGeofenceMap key={geofenceRefreshKey} height="calc(100vh - 220px)" />
                            </Card>
                        )
                    }
                ]}
            />
        </div>
    );
};

export default GlobalPorts;
