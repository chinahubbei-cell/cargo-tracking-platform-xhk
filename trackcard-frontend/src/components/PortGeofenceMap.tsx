import React, { useEffect, useState, useCallback, useRef, useMemo } from 'react';
import { MapContainer, TileLayer, Circle, Polygon, Marker, Popup, useMap, useMapEvents, ZoomControl } from 'react-leaflet';
import { Spin, Tag, Space, Typography, Switch } from 'antd';
import { EnvironmentOutlined } from '@ant-design/icons';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

const { Text } = Typography;

// 港口围栏数据类型
interface PortGeofence {
    id: string;
    code: string;
    name: string;
    name_cn: string;
    country: string;
    country_cn: string;
    geofence_type: 'circle' | 'polygon';
    center_lat: number;
    center_lng: number;
    radius: number;
    polygon_points?: string;
    color: string;
    is_active: boolean;
}

// 缓存港口图标 - 避免重复创建
const portIconCache = new Map<string, L.DivIcon>();

const createPortIcon = (color: string): L.DivIcon => {
    if (portIconCache.has(color)) {
        return portIconCache.get(color)!;
    }

    const icon = L.divIcon({
        className: 'port-marker-icon',
        html: `<div style="
            background-color: ${color};
            width: 20px;
            height: 20px;
            border-radius: 50%;
            border: 2px solid white;
            box-shadow: 0 2px 6px rgba(0,0,0,0.3);
            display: flex;
            align-items: center;
            justify-content: center;
        ">
            <span style="color: white; font-size: 10px;">⚓</span>
        </div>`,
        iconSize: [20, 20],
        iconAnchor: [10, 10],
    });

    portIconCache.set(color, icon);
    return icon;
};

// 视口过滤组件
interface ViewportFilterProps {
    ports: PortGeofence[];
    showGeofences: boolean;
    onVisiblePortsChange: (ports: PortGeofence[]) => void;
}

const ViewportFilter: React.FC<ViewportFilterProps> = ({ ports, showGeofences, onVisiblePortsChange }) => {
    const map = useMap();
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    const filterByViewport = useCallback(() => {
        if (debounceRef.current) {
            clearTimeout(debounceRef.current);
        }

        debounceRef.current = setTimeout(() => {
            const bounds = map.getBounds();


            // 过滤可见港口
            const visible = ports.filter(p =>
                bounds.contains([p.center_lat, p.center_lng])
            );

            // 限制渲染数量
            let filtered = visible;
            if (filtered.length > 100) {
                filtered = filtered.slice(0, 100);
            }

            onVisiblePortsChange(filtered);
        }, 150);
    }, [map, ports, onVisiblePortsChange]);

    useMapEvents({
        moveend: filterByViewport,
        zoomend: filterByViewport,
    });

    useEffect(() => {
        filterByViewport();
    }, [ports, showGeofences]);

    return null;
};

// 地图自动适应组件 - 仅首次加载
const MapAutoFit: React.FC<{ ports: PortGeofence[] }> = ({ ports }) => {
    const map = useMap();
    const hasFitted = useRef(false);

    useEffect(() => {
        if (ports.length > 0 && !hasFitted.current) {
            const sample = ports.slice(0, 30);
            const bounds = L.latLngBounds(
                sample.map(p => [p.center_lat, p.center_lng] as L.LatLngTuple)
            );
            map.fitBounds(bounds, { padding: [50, 50] });
            hasFitted.current = true;
        }
    }, [ports, map]);

    return null;
};

interface PortGeofenceMapProps {
    height?: string | number;
    showLabels?: boolean;
    filterCountry?: string;
    onPortClick?: (port: PortGeofence) => void;
}

const PortGeofenceMap: React.FC<PortGeofenceMapProps> = ({
    height = '600px',
    filterCountry,
    onPortClick
}) => {
    const [ports, setPorts] = useState<PortGeofence[]>([]);
    const [visiblePorts, setVisiblePorts] = useState<PortGeofence[]>([]);
    const [loading, setLoading] = useState(true);
    const [showGeofences, setShowGeofences] = useState(true);

    useEffect(() => {
        loadPorts();
    }, []);

    const loadPorts = async () => {
        try {
            const token = localStorage.getItem('token');
            const response = await fetch('/api/port-geofences', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            const data = await response.json();
            if (data?.data) {
                setPorts(data.data);
            }
        } catch (error) {
            console.error('Failed to load port geofences:', error);
        } finally {
            setLoading(false);
        }
    };

    const filteredPorts = useMemo(() => {
        if (!filterCountry) return ports;
        return ports.filter(p => p.country === filterCountry || p.country_cn === filterCountry);
    }, [ports, filterCountry]);

    const handlePortClick = useCallback((port: PortGeofence) => {
        onPortClick?.(port);
    }, [onPortClick]);

    // 缓存解析多边形顶点的结果
    const parsePolygonPoints = useCallback((pointsJson: string): L.LatLngTuple[] => {
        try {
            const points = JSON.parse(pointsJson);
            return points.map((p: number[]) => [p[0], p[1]] as L.LatLngTuple);
        } catch {
            return [];
        }
    }, []);

    if (loading) {
        return (
            <div style={{ height, display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#1a1a2e' }}>
                <Spin size="large" tip="加载港口围栏数据..." />
            </div>
        );
    }

    return (
        <div style={{ position: 'relative', height }}>
            {/* 控制面板 - 左下角 */}
            <div style={{
                position: 'absolute',
                bottom: 10,
                left: 10,
                zIndex: 1000,
                background: 'white',
                padding: '8px 12px',
                borderRadius: 6,
                boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
            }}>
                <Space>
                    <Text>显示围栏</Text>
                    <Switch
                        checked={showGeofences}
                        onChange={setShowGeofences}
                        size="small"
                    />
                </Space>
            </div>

            {/* 港口统计 - 左上角 */}
            <div style={{
                position: 'absolute',
                top: 10,
                left: 10,
                zIndex: 1000,
                background: 'rgba(255,255,255,0.95)',
                padding: '8px 12px',
                borderRadius: 6,
                boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
            }}>
                <Space>
                    <EnvironmentOutlined />
                    <Text>
                        总计 <strong>{filteredPorts.length}</strong> 个 |
                        可见 <strong>{visiblePorts.length}</strong> 个
                    </Text>
                </Space>
            </div>

            <MapContainer
                center={[30, 120]}
                zoom={3}
                zoomControl={false}
                style={{ height: '100%', width: '100%', borderRadius: 8 }}
            >
                <TileLayer
                    url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
                    attribution='&copy; CartoDB'
                />
                <ZoomControl position="topright" />

                <MapAutoFit ports={filteredPorts} />

                {/* 视口过滤器 */}
                <ViewportFilter
                    ports={filteredPorts}
                    showGeofences={showGeofences}
                    onVisiblePortsChange={setVisiblePorts}
                />

                {/* 仅渲染可见的港口 */}
                {visiblePorts.map(port => (
                    <React.Fragment key={port.id}>
                        {/* 围栏 */}
                        {showGeofences && port.geofence_type === 'circle' && (
                            <Circle
                                center={[port.center_lat, port.center_lng]}
                                radius={port.radius}
                                pathOptions={{
                                    color: port.color,
                                    fillColor: port.color,
                                    fillOpacity: 0.2,
                                    weight: 2,
                                }}
                            />
                        )}

                        {showGeofences && port.geofence_type === 'polygon' && port.polygon_points && (
                            <Polygon
                                positions={parsePolygonPoints(port.polygon_points)}
                                pathOptions={{
                                    color: port.color,
                                    fillColor: port.color,
                                    fillOpacity: 0.2,
                                    weight: 2,
                                }}
                            />
                        )}

                        {/* 港口标记 */}
                        <Marker
                            position={[port.center_lat, port.center_lng]}
                            icon={createPortIcon(port.color)}
                            eventHandlers={{
                                click: () => handlePortClick(port)
                            }}
                        >
                            <Popup>
                                <div style={{ minWidth: 180 }}>
                                    <h4 style={{ margin: '0 0 8px 0' }}>
                                        {port.name_cn} ({port.code})
                                    </h4>
                                    <p style={{ margin: '4px 0', color: '#666', fontSize: 12 }}>
                                        {port.name}
                                    </p>
                                    <Space direction="vertical" size={4}>
                                        <div>
                                            <Tag color="blue">{port.country_cn}</Tag>
                                        </div>
                                        <div>
                                            <Text type="secondary" style={{ fontSize: 12 }}>围栏类型: </Text>
                                            <Tag color={port.geofence_type === 'circle' ? 'green' : 'orange'} style={{ fontSize: 11 }}>
                                                {port.geofence_type === 'circle' ? '圆形' : '多边形'}
                                            </Tag>
                                        </div>
                                        {port.geofence_type === 'circle' && (
                                            <div>
                                                <Text type="secondary" style={{ fontSize: 12 }}>半径: </Text>
                                                <Text strong>{(port.radius / 1000).toFixed(1)} km</Text>
                                            </div>
                                        )}
                                    </Space>
                                </div>
                            </Popup>
                        </Marker>
                    </React.Fragment>
                ))}
            </MapContainer>
        </div>
    );
};

export default PortGeofenceMap;
