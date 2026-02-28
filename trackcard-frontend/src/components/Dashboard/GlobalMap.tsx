import React, { useMemo, useEffect, useRef } from 'react';
import { MapContainer, TileLayer, Marker, Popup, Polyline, ZoomControl, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import type { Device, Shipment } from '../../types';
import { useNavigate } from 'react-router-dom';
import { SendOutlined } from '@ant-design/icons';

// 地图自适应组件 - 当容器大小变化时重新计算地图尺寸
const MapResizeHandler: React.FC = () => {
    const map = useMap();

    useEffect(() => {
        const handleResize = () => {
            setTimeout(() => {
                map.invalidateSize();
            }, 100);
        };

        // 监听窗口大小变化
        window.addEventListener('resize', handleResize);

        // 初始化时也调用一次
        handleResize();

        return () => {
            window.removeEventListener('resize', handleResize);
        };
    }, [map]);

    return null;
};

// 运单状态颜色 - 与 DonutChart 保持一致
const STATUS_COLORS = {
    pending: '#f59e0b',      // 待发运 - 橙色
    in_transit: '#2563eb',   // 运输中 - 蓝色
    delivered: '#10b981',    // 已送达 - 绿色
};

// 自定义标记图标 - 根据运单状态
const createShipmentMarkerIcon = (status: string) => {
    const color = STATUS_COLORS[status as keyof typeof STATUS_COLORS] || '#9ca3af';
    const isActive = status === 'in_transit';

    return L.divIcon({
        className: 'custom-marker',
        html: `
            <div style="
                width: 24px;
                height: 24px;
                background: ${color};
                border: 3px solid white;
                border-radius: 50%;
                box-shadow: 0 2px 8px rgba(0,0,0,0.2);
                position: relative;
            ">
                ${isActive ? `
                    <div style="
                        position: absolute;
                        top: -3px;
                        left: -3px;
                        right: -3px;
                        bottom: -3px;
                        border: 2px solid ${color};
                        border-radius: 50%;
                        opacity: 0.4;
                        animation: ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite;
                    "></div>
                ` : ''}
            </div>
            <style>
                @keyframes ping {
                    75%, 100% {
                        transform: scale(1.5);
                        opacity: 0;
                    }
                }
            </style>
        `,
        iconSize: [24, 24],
        iconAnchor: [12, 12],
        popupAnchor: [0, -12],
    });
};

interface GlobalMapProps {
    devices: Device[];
    shipments?: Shipment[];
    height?: string;
    showRoutes?: boolean;
}

const GlobalMap: React.FC<GlobalMapProps> = ({
    devices: _devices, // eslint-disable-line @typescript-eslint/no-unused-vars
    shipments = [],
    height = '400px',
    showRoutes = true,
}) => {
    const navigate = useNavigate();
    const containerRef = useRef<HTMLDivElement>(null);
    const mapRef = useRef<L.Map | null>(null);

    // 使用 ResizeObserver 监听容器大小变化
    useEffect(() => {
        if (!containerRef.current) return;

        const resizeObserver = new ResizeObserver(() => {
            if (mapRef.current) {
                setTimeout(() => {
                    mapRef.current?.invalidateSize();
                }, 100);
            }
        });

        resizeObserver.observe(containerRef.current);

        return () => {
            resizeObserver.disconnect();
        };
    }, []);

    // 生成模拟运单位置数据
    const shipmentLocations = useMemo(() => {
        const cities = [
            { name: '上海', lat: 31.2304, lng: 121.4737 },
            { name: '北京', lat: 39.9042, lng: 116.4074 },
            { name: '深圳', lat: 22.5431, lng: 114.0579 },
            { name: '广州', lat: 23.1291, lng: 113.2644 },
            { name: '香港', lat: 22.3193, lng: 114.1694 },
            { name: '东京', lat: 35.6762, lng: 139.6503 },
            { name: '新加坡', lat: 1.3521, lng: 103.8198 },
            { name: '迪拜', lat: 25.2048, lng: 55.2708 },
            { name: '伦敦', lat: 51.5074, lng: -0.1278 },
            { name: '纽约', lat: 40.7128, lng: -74.0060 },
            { name: '洛杉矶', lat: 34.0522, lng: -118.2437 },
            { name: '悉尼', lat: -33.8688, lng: 151.2093 },
            { name: '鹿特丹', lat: 51.9244, lng: 4.4777 },
            { name: '汉堡', lat: 53.5511, lng: 9.9937 },
            { name: '首尔', lat: 37.5665, lng: 126.978 },
            { name: '曼谷', lat: 13.7563, lng: 100.5018 },
            { name: '孟买', lat: 19.076, lng: 72.8777 },
            { name: '巴拿马城', lat: 8.9824, lng: -79.5199 },
            { name: '圣保罗', lat: -23.5505, lng: -46.6333 },
            { name: '开普敦', lat: -33.9249, lng: 18.4241 },
        ];

        const statuses: ('pending' | 'in_transit' | 'delivered')[] = ['pending', 'in_transit', 'delivered'];

        // 如果有真实运单数据，使用真实数据
        if (shipments.length > 0) {
            return shipments.map((shipment, i) => {
                const city = cities[i % cities.length];
                return {
                    id: shipment.id,
                    shipment_no: shipment.id, // 使用id作为运单号
                    status: shipment.status as 'pending' | 'in_transit' | 'delivered',
                    origin: shipment.origin,
                    destination: shipment.destination,
                    lat: city.lat + (Math.random() - 0.5) * 2,
                    lng: city.lng + (Math.random() - 0.5) * 2,
                    cityName: city.name,
                };
            });
        }

        // 否则生成模拟数据
        return Array.from({ length: 50 }, (_, i) => {
            const city = cities[i % cities.length];
            const status = statuses[i % 3];
            return {
                id: `SHP-${String(i + 1).padStart(5, '0')}`,
                shipment_no: `WB${new Date().getFullYear()}${String(i + 1).padStart(6, '0')}`,
                status,
                origin: cities[i % cities.length].name,
                destination: cities[(i + 5) % cities.length].name,
                lat: city.lat + (Math.random() - 0.5) * 2,
                lng: city.lng + (Math.random() - 0.5) * 2,
                cityName: city.name,
            };
        });
    }, [shipments]);

    // 模拟航线数据
    const routes = useMemo(() => {
        if (!showRoutes) return [];

        // 创建一些示例航线
        return [
            [[22.5, 114], [34.5, 135], [35.5, 139.7]], // 深圳-大阪-东京
            [[22.5, 114], [1.3, 103.8], [51.5, -0.1]], // 深圳-新加坡-伦敦
            [[31.2, 121.5], [33.9, -118.2]], // 上海-洛杉矶
        ];
    }, [showRoutes]);

    const center: [number, number] = [25, 105];

    // 统计各状态数量
    const pendingCount = shipmentLocations.filter(s => s.status === 'pending').length;
    const inTransitCount = shipmentLocations.filter(s => s.status === 'in_transit').length;
    const deliveredCount = shipmentLocations.filter(s => s.status === 'delivered').length;

    const getStatusLabel = (status: string) => {
        switch (status) {
            case 'pending': return '待发运';
            case 'in_transit': return '运输中';
            case 'delivered': return '已送达';
            default: return status;
        }
    };

    return (
        <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
            {/* 标题栏 */}
            <div style={{
                padding: '16px 20px',
                borderBottom: '1px solid var(--border-light)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between'
            }}>
                <div>
                    <h3 className="card-title" style={{ margin: 0 }}>全球运输概览</h3>
                    <p className="card-subtitle">Global Shipments Overview</p>
                </div>
                <div style={{ display: 'flex', gap: 16, fontSize: 13 }}>
                    {/* 待发运 */}
                    <span style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--text-tertiary)' }}>
                        <span style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: STATUS_COLORS.pending,
                        }} />
                        待发运 {pendingCount}
                    </span>
                    {/* 运输中 */}
                    <span style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--text-tertiary)' }}>
                        <span style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: STATUS_COLORS.in_transit,
                            boxShadow: `0 0 6px ${STATUS_COLORS.in_transit}`
                        }} />
                        运输中 {inTransitCount}
                    </span>
                    {/* 已送达 */}
                    <span style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--text-tertiary)' }}>
                        <span style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: STATUS_COLORS.delivered,
                        }} />
                        已送达 {deliveredCount}
                    </span>
                </div>
            </div>

            {/* 地图 */}
            <div ref={containerRef} style={{ height }} className="map-container">
                <MapContainer
                    center={center}
                    zoom={3}
                    style={{ height: '100%', width: '100%' }}
                    scrollWheelZoom={true}
                    zoomControl={false}
                    ref={(map) => { if (map) mapRef.current = map; }}
                >
                    {/* 地图自适应处理器 */}
                    <MapResizeHandler />

                    {/* 浅色地图瓦片 */}
                    <TileLayer
                        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>'
                        url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
                    />

                    <ZoomControl position="topright" />

                    {/* 航线 */}
                    {routes.map((route, idx) => (
                        <Polyline
                            key={idx}
                            positions={route as [number, number][]}
                            pathOptions={{
                                color: '#2563eb',
                                weight: 2,
                                opacity: 0.6,
                                dashArray: '8, 8',
                            }}
                        />
                    ))}

                    {/* 运单标记 */}
                    {shipmentLocations.map((shipment) => (
                        <Marker
                            key={shipment.id}
                            position={[shipment.lat, shipment.lng]}
                            icon={createShipmentMarkerIcon(shipment.status)}
                        >
                            <Popup>
                                <div style={{ padding: 8, minWidth: 200 }}>
                                    <div style={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: 10,
                                        marginBottom: 12
                                    }}>
                                        <div style={{
                                            width: 36,
                                            height: 36,
                                            borderRadius: 8,
                                            background: STATUS_COLORS[shipment.status] + '20',
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'center',
                                            color: STATUS_COLORS[shipment.status],
                                        }}>
                                            <SendOutlined />
                                        </div>
                                        <div>
                                            <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-primary)' }}>
                                                {shipment.shipment_no}
                                            </div>
                                            <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                                                {shipment.cityName}
                                            </div>
                                        </div>
                                    </div>

                                    <div style={{ display: 'flex', flexDirection: 'column', gap: 8, fontSize: 13 }}>
                                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                            <span style={{ color: 'var(--text-tertiary)' }}>状态</span>
                                            <span style={{
                                                padding: '2px 8px',
                                                borderRadius: 4,
                                                fontSize: 12,
                                                background: STATUS_COLORS[shipment.status] + '20',
                                                color: STATUS_COLORS[shipment.status],
                                                fontWeight: 500
                                            }}>
                                                {getStatusLabel(shipment.status)}
                                            </span>
                                        </div>
                                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                            <span style={{ color: 'var(--text-tertiary)' }}>起点</span>
                                            <span style={{ fontWeight: 500, color: 'var(--text-primary)' }}>
                                                {shipment.origin}
                                            </span>
                                        </div>
                                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                            <span style={{ color: 'var(--text-tertiary)' }}>终点</span>
                                            <span style={{ fontWeight: 500, color: 'var(--text-primary)' }}>
                                                {shipment.destination}
                                            </span>
                                        </div>
                                    </div>

                                    <button
                                        onClick={() => navigate(`/shipments?id=${shipment.id}`)}
                                        className="btn btn-primary"
                                        style={{ width: '100%', marginTop: 12, padding: '8px 12px', fontSize: 13 }}
                                    >
                                        查看详情
                                    </button>
                                </div>
                            </Popup>
                        </Marker>
                    ))}
                </MapContainer>
            </div>
        </div>
    );
};

export default GlobalMap;
