import React, { useState, useEffect, useMemo } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap, ZoomControl } from 'react-leaflet';
import { Radio } from 'antd';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

interface DeviceData {
    id: string;
    name: string;
    provider: string;
    status: string;
    battery: number;
    latitude: number;
    longitude: number;
    speed: number;
    temperature: number;
    external_device_id: string;
}

interface ProviderConfig {
    label: string;
    color: string;
}

type StatusFilter = 'all' | 'online' | 'offline';

interface Props {
    devices: DeviceData[];
    providers: Record<string, ProviderConfig>;
    focusDevice?: DeviceData | null;
    onClearFocus?: () => void;
}

// CartoDB 地图样式配置
const MAP_STYLES = {
    light: {
        name: '浅色',
        url: 'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png',
    },
    dark: {
        name: '深色',
        url: 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png',
    },
    voyager: {
        name: '街道',
        url: 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png',
    },
};

type MapStyleKey = keyof typeof MAP_STYLES;

// 创建自定义图标
const createMarkerIcon = (status: string, color: string, isFocused: boolean = false) => {
    const baseColor = status === 'online' ? color : '#9ca3af';
    const size = isFocused ? 28 : 20;
    const border = isFocused ? 4 : 2;

    return L.divIcon({
        className: 'device-marker',
        html: `<div style="
            width: ${size}px;
            height: ${size}px;
            background: ${baseColor};
            border: ${border}px solid ${isFocused ? '#fff' : 'white'};
            border-radius: 50%;
            box-shadow: 0 2px ${isFocused ? 12 : 6}px rgba(0,0,0,${isFocused ? 0.5 : 0.3});
            ${isFocused ? 'animation: pulse 1.5s infinite;' : ''}
        "></div>`,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
        // popup显示在标记正上方
        popupAnchor: [0, -size / 2 - 8],
    });
};

// 地图样式切换组件
const MapStyleSwitcher: React.FC<{ currentStyle: MapStyleKey; onChange: (style: MapStyleKey) => void }> = ({ currentStyle, onChange }) => {
    return (
        <div style={{
            position: 'absolute',
            bottom: 24,
            right: 10,
            zIndex: 1000,
            background: 'rgba(255,255,255,0.95)',
            borderRadius: 6,
            padding: '4px 8px',
            boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
            backdropFilter: 'blur(8px)',
        }}>
            <Radio.Group
                value={currentStyle}
                onChange={(e) => onChange(e.target.value)}
                size="small"
                optionType="button"
                buttonStyle="solid"
            >
                {Object.entries(MAP_STYLES).map(([key, style]) => (
                    <Radio.Button key={key} value={key}>{style.name}</Radio.Button>
                ))}
            </Radio.Group>
        </div>
    );
};

// TileLayer 动态更新组件
const DynamicTileLayer: React.FC<{ url: string }> = ({ url }) => {
    const map = useMap();

    React.useEffect(() => {
        map.invalidateSize();
    }, [url, map]);

    return (
        <TileLayer
            url={url}
            attribution='&copy; <a href="https://carto.com/">CartoDB</a>'
        />
    );
};

// 地图聚焦组件
const MapFocuser: React.FC<{ device: DeviceData | null }> = ({ device }) => {
    const map = useMap();

    useEffect(() => {
        if (device && device.latitude && device.longitude) {
            // popup显示在标记上方，地图中心向北偏移（纬度+）
            // 这样标记会显示在屏幕下部，给上方弹窗留出空间
            const offsetLat = device.latitude + 0.06;
            map.flyTo([offsetLat, device.longitude], 11, { duration: 1 });
        }
    }, [device, map]);

    return null;
};



// 地址反查组件 - 通过后端API代理
const AddressDisplay: React.FC<{ lat: number; lng: number }> = ({ lat, lng }) => {
    const [address, setAddress] = useState<string>('地址加载中...');

    useEffect(() => {
        const fetchAddress = async () => {
            try {
                const { default: api } = await import('../api/client');
                const res = await api.reverseGeocode(lat, lng);
                if (res.success && res.display_name) {
                    setAddress(res.display_name);
                } else {
                    // 逆编码失败，显示格式化坐标
                    setAddress(`${lat.toFixed(4)}°${lat >= 0 ? 'N' : 'S'}, ${lng.toFixed(4)}°${lng >= 0 ? 'E' : 'W'}`);
                }
            } catch (error) {
                console.error('Reverse geocoding error:', error);
                setAddress(`${lat.toFixed(4)}°${lat >= 0 ? 'N' : 'S'}, ${lng.toFixed(4)}°${lng >= 0 ? 'E' : 'W'}`);
            }
        };

        if (lat && lng) {
            fetchAddress();
        }
    }, [lat, lng]);

    return <span>{address}</span>;
};

// 带焦点控制的Marker组件 - 使用ref精确控制popup
interface FocusedMarkerProps {
    device: DeviceData;
    isFocused: boolean;
    icon: L.DivIcon;
    getProviderLabel: (provider: string) => string;
}

const FocusedMarker: React.FC<FocusedMarkerProps> = ({ device, isFocused, icon, getProviderLabel }) => {
    const markerRef = React.useRef<L.Marker>(null);

    useEffect(() => {
        // 只在isFocused为true时打开popup
        if (isFocused && markerRef.current) {
            const timer = setTimeout(() => {
                markerRef.current?.openPopup();
            }, 500);
            return () => clearTimeout(timer);
        }
    }, [isFocused]);

    return (
        <Marker
            ref={markerRef}
            position={[device.latitude, device.longitude]}
            icon={icon}
        >
            <Popup
                autoPan={true}
                autoClose={true}
                closeOnClick={true}
                autoPanPaddingTopLeft={[20, 180]}
                autoPanPaddingBottomRight={[20, 30]}
            >
                <div style={{ minWidth: 260, padding: '6px 4px' }}>
                    {/* 标题行：供应商+状态 */}
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                        <span style={{ fontSize: 14, fontWeight: 700, color: '#111827' }}>
                            {getProviderLabel(device.provider)}
                        </span>
                        <span style={{
                            display: 'flex', alignItems: 'center', gap: 3,
                            fontSize: 11, fontWeight: 600,
                            color: device.status === 'online' ? '#22c55e' : '#9ca3af',
                        }}>
                            <span style={{
                                width: 6, height: 6, borderRadius: '50%',
                                background: device.status === 'online' ? '#22c55e' : '#9ca3af',
                            }} />
                            {device.status === 'online' ? '在线' : '离线'}
                        </span>
                    </div>
                    {/* 设备ID */}
                    <div style={{ fontSize: 11, color: '#6b7280', marginBottom: 6, fontFamily: 'monospace' }}>
                        ID: {device.external_device_id || '-'}
                    </div>
                    {/* 紧凑两列布局 */}
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '3px 10px', fontSize: 11 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                            <span>🔋</span>
                            <span style={{ color: '#6b7280' }}>电量:</span>
                            <span style={{ fontWeight: 600 }}>{device.battery}%</span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                            <span>🌡</span>
                            <span style={{ color: '#6b7280' }}>温度:</span>
                            <span style={{ fontWeight: 600 }}>{device.temperature?.toFixed(1) || '-'}°C</span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                            <span>💧</span>
                            <span style={{ color: '#6b7280' }}>湿度:</span>
                            <span style={{ fontWeight: 600 }}>{((device as any).humidity || 0).toFixed(0)}%</span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                            <span>📍</span>
                            <span style={{ fontWeight: 600 }}>{device.latitude?.toFixed(4)},{device.longitude?.toFixed(4)}</span>
                        </div>
                    </div>
                    {/* 时间 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 11, marginTop: 4 }}>
                        <span>🕐</span>
                        <span style={{ color: '#6b7280' }}>时间:</span>
                        <span style={{ fontWeight: 600 }}>
                            {(device as any).last_update
                                ? new Date((device as any).last_update).toLocaleString('zh-CN', {
                                    month: '2-digit', day: '2-digit',
                                    hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
                                }).replace(/\//g, '-')
                                : '-'}
                        </span>
                    </div>
                    {/* 地址 */}
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 4, fontSize: 11, marginTop: 6, color: '#6b7280' }}>
                        <span style={{ flexShrink: 0 }}>🏠</span>
                        <span style={{ lineHeight: 1.4, wordBreak: 'break-word' }}>
                            <AddressDisplay lat={device.latitude} lng={device.longitude} />
                        </span>
                    </div>
                </div>
            </Popup>
        </Marker>
    );
};

// 状态筛选控件组件 - 横向悬浮样式
const StatusFilterControl: React.FC<{
    statusFilter: StatusFilter;
    onChange: (status: StatusFilter) => void;
    totalCount: number;
    onlineCount: number;
    offlineCount: number;
}> = ({ statusFilter, onChange, totalCount, onlineCount, offlineCount }) => {
    return (
        <div style={{
            position: 'absolute',
            top: 10,
            left: 10,
            zIndex: 1000,
            background: 'rgba(255,255,255,0.95)',
            borderRadius: 16,
            padding: '4px 6px',
            boxShadow: '0 1px 8px rgba(0,0,0,0.1)',
            display: 'flex',
            alignItems: 'center',
            gap: 2,
            backdropFilter: 'blur(8px)',
        }}>
            <div
                onClick={() => onChange('all')}
                style={{
                    cursor: 'pointer',
                    padding: '3px 10px',
                    borderRadius: 12,
                    background: statusFilter === 'all' ? '#1890ff' : 'transparent',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    transition: 'all 0.15s',
                }}
            >
                <span style={{
                    fontSize: 11,
                    fontWeight: 500,
                    color: statusFilter === 'all' ? '#fff' : '#666',
                }}>全部</span>
                <span style={{
                    fontSize: 10,
                    color: statusFilter === 'all' ? 'rgba(255,255,255,0.85)' : '#999',
                    fontWeight: 500,
                }}>{totalCount}</span>
            </div>
            <div style={{ width: 1, height: 14, background: '#e5e7eb' }} />
            <div
                onClick={() => onChange('online')}
                style={{
                    cursor: 'pointer',
                    padding: '3px 10px',
                    borderRadius: 12,
                    background: statusFilter === 'online' ? '#52c41a' : 'transparent',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    transition: 'all 0.15s',
                }}
            >
                <span style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: statusFilter === 'online' ? '#fff' : '#52c41a',
                }} />
                <span style={{
                    fontSize: 11,
                    fontWeight: 500,
                    color: statusFilter === 'online' ? '#fff' : '#666',
                }}>在线</span>
                <span style={{
                    fontSize: 10,
                    color: statusFilter === 'online' ? 'rgba(255,255,255,0.85)' : '#52c41a',
                    fontWeight: 600,
                }}>{onlineCount}</span>
            </div>
            <div style={{ width: 1, height: 20, background: '#e5e7eb' }} />
            <div
                onClick={() => onChange('offline')}
                style={{
                    cursor: 'pointer',
                    padding: '3px 10px',
                    borderRadius: 12,
                    background: statusFilter === 'offline' ? '#8c8c8c' : 'transparent',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    transition: 'all 0.15s',
                }}
            >
                <span style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: statusFilter === 'offline' ? '#fff' : '#bfbfbf',
                    border: statusFilter === 'offline' ? 'none' : '1px solid #d9d9d9',
                }} />
                <span style={{
                    fontSize: 11,
                    fontWeight: 500,
                    color: statusFilter === 'offline' ? '#fff' : '#666',
                }}>离线</span>
                <span style={{
                    fontSize: 10,
                    color: statusFilter === 'offline' ? 'rgba(255,255,255,0.85)' : '#999',
                    fontWeight: 500,
                }}>{offlineCount}</span>
            </div>
        </div>
    );
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const DeviceMap: React.FC<Props> = ({ devices, providers, focusDevice, onClearFocus }) => {
    const [mapStyle, setMapStyle] = useState<MapStyleKey>('light');
    const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
    // 内部状态：是否应该显示焦点设备（用户点击筛选后清除）
    const [showFocusDevice, setShowFocusDevice] = useState(true);

    // 当focusDevice变化时，重新显示它
    useEffect(() => {
        if (focusDevice) {
            setShowFocusDevice(true);
        }
    }, [focusDevice]);

    // 统计设备数量
    const onlineCount = useMemo(() => devices.filter(d => d.status === 'online').length, [devices]);
    const offlineCount = useMemo(() => devices.filter(d => d.status === 'offline').length, [devices]);

    // 根据状态筛选设备
    const filteredDevices = useMemo(() => {
        if (statusFilter === 'all') return devices;
        return devices.filter(d => d.status === statusFilter);
    }, [devices, statusFilter]);

    // 处理状态筛选变化：清除焦点设备显示
    const handleStatusFilterChange = (filter: StatusFilter) => {
        setStatusFilter(filter);
        setShowFocusDevice(false); // 点击筛选时取消焦点设备显示
        if (onClearFocus) {
            onClearFocus();
        }
    };

    const getProviderColor = (provider: string) => {
        const providerColors: Record<string, string> = {
            kuaihuoyun: '#2563eb',
            sinoiov: '#f59e0b',
        };
        return providerColors[provider] || '#6b7280';
    };

    const getProviderLabel = (provider: string) => {
        return providers[provider]?.label || provider || '-';
    };

    return (
        <div style={{ position: 'relative', height: '100%', width: '100%' }}>
            <StatusFilterControl
                statusFilter={statusFilter}
                onChange={handleStatusFilterChange}
                totalCount={devices.length}
                onlineCount={onlineCount}
                offlineCount={offlineCount}
            />
            <MapStyleSwitcher currentStyle={mapStyle} onChange={setMapStyle} />
            <MapContainer
                center={[30, 110]}
                zoom={3}
                zoomControl={false}
                style={{ height: '100%', width: '100%' }}
            >
                <ZoomControl position="topright" />
                <DynamicTileLayer url={MAP_STYLES[mapStyle].url} />
                <MapFocuser device={showFocusDevice ? focusDevice ?? null : null} />
                {/* 始终显示筛选后的所有设备，焦点设备会被高亮并自动定位 */}
                {filteredDevices.map((device) => {
                    if (!device.latitude || !device.longitude) return null;
                    const isFocused = focusDevice?.id === device.id && showFocusDevice;

                    return (
                        <FocusedMarker
                            key={device.id}
                            device={device}
                            isFocused={isFocused}
                            icon={createMarkerIcon(device.status, getProviderColor(device.provider), isFocused)}
                            getProviderLabel={getProviderLabel}
                        />
                    );
                })}
            </MapContainer>
            <style>{`
                @keyframes pulse {
                    0% { transform: scale(1); opacity: 1; }
                    50% { transform: scale(1.1); opacity: 0.8; }
                    100% { transform: scale(1); opacity: 1; }
                }
            `}</style>
        </div>
    );
};

export default DeviceMap;
