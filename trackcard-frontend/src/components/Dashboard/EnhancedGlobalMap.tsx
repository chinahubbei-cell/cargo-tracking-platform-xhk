import React, { useState, useMemo, useCallback } from 'react';
import { MapContainer, TileLayer, Marker, Popup, ZoomControl, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import type { Device } from '../../types';
import { useNavigate } from 'react-router-dom';
import {
    EnvironmentOutlined,
    ThunderboltOutlined,
    AimOutlined,
    FullscreenOutlined,
    FullscreenExitOutlined,
} from '@ant-design/icons';

// 自定义标记图标
const createCustomIcon = (status: string, isSelected: boolean = false) => {
    const color = status === 'online' ? '#10b981' : '#64748b';
    const size = isSelected ? 36 : 28;
    const pulseSize = isSelected ? 50 : 40;

    return L.divIcon({
        className: 'custom-marker',
        html: `
            <div style="position: relative; width: ${size}px; height: ${size}px;">
                ${status === 'online' ? `
                    <div style="
                        position: absolute;
                        top: 50%;
                        left: 50%;
                        transform: translate(-50%, -50%);
                        width: ${pulseSize}px;
                        height: ${pulseSize}px;
                        background: ${color}30;
                        border-radius: 50%;
                        animation: pulse-map 2s infinite;
                    "></div>
                ` : ''}
                <div style="
                    position: absolute;
                    top: 50%;
                    left: 50%;
                    transform: translate(-50%, -50%);
                    width: ${size}px;
                    height: ${size}px;
                    background: linear-gradient(135deg, ${color}, ${color}cc);
                    border-radius: 50%;
                    border: 3px solid white;
                    box-shadow: 0 4px 12px ${color}60;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                ">
                    <svg width="${size * 0.5}" height="${size * 0.5}" viewBox="0 0 24 24" fill="white">
                        <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>
                    </svg>
                </div>
            </div>
        `,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
        popupAnchor: [0, -size / 2],
    });
};

// 地图控制组件
const MapControls: React.FC<{
    onLocateAll: () => void;
    isFullscreen: boolean;
    onToggleFullscreen: () => void;
}> = ({ onLocateAll, isFullscreen, onToggleFullscreen }) => {
    useMap(); // needed for map context

    return (
        <div className="absolute top-4 right-4 z-[1000] flex flex-col gap-2">
            <button
                onClick={onLocateAll}
                className="w-10 h-10 rounded-lg bg-slate-800/90 hover:bg-slate-700 
                           border border-slate-600/50 text-slate-300 hover:text-white
                           flex items-center justify-center transition-all duration-200
                           backdrop-blur-sm"
                title="定位所有设备"
            >
                <AimOutlined style={{ fontSize: '18px' }} />
            </button>
            <button
                onClick={onToggleFullscreen}
                className="w-10 h-10 rounded-lg bg-slate-800/90 hover:bg-slate-700 
                           border border-slate-600/50 text-slate-300 hover:text-white
                           flex items-center justify-center transition-all duration-200
                           backdrop-blur-sm"
                title={isFullscreen ? "退出全屏" : "全屏显示"}
            >
                {isFullscreen
                    ? <FullscreenExitOutlined style={{ fontSize: '18px' }} />
                    : <FullscreenOutlined style={{ fontSize: '18px' }} />
                }
            </button>
        </div>
    );
};

interface EnhancedGlobalMapProps {
    devices: Device[];
    selectedDeviceId?: string;
    onDeviceSelect?: (device: Device) => void;
    height?: string;
}

const EnhancedGlobalMap: React.FC<EnhancedGlobalMapProps> = ({
    devices,
    selectedDeviceId,
    onDeviceSelect,
    height = '450px',
}) => {
    const navigate = useNavigate();
    const [isFullscreen, setIsFullscreen] = useState(false);
    const [mapRef, setMapRef] = useState<L.Map | null>(null);

    // 计算设备边界
    const bounds = useMemo(() => {
        const validDevices = devices.filter(d => d.latitude && d.longitude);
        if (validDevices.length === 0) return undefined;

        const lats = validDevices.map(d => d.latitude!);
        const lngs = validDevices.map(d => d.longitude!);

        return L.latLngBounds(
            [Math.min(...lats), Math.min(...lngs)],
            [Math.max(...lats), Math.max(...lngs)]
        );
    }, [devices]);

    const handleLocateAll = useCallback(() => {
        if (mapRef && bounds) {
            mapRef.fitBounds(bounds, { padding: [50, 50], maxZoom: 10 });
        }
    }, [mapRef, bounds]);

    const handleToggleFullscreen = () => {
        setIsFullscreen(!isFullscreen);
    };

    const center: [number, number] = [25, 110]; // 默认中国中心

    // 统计在线/离线设备
    const onlineCount = devices.filter(d => d.status === 'online').length;
    const offlineCount = devices.filter(d => d.status === 'offline').length;

    return (
        <div className={`
            glass-card overflow-hidden relative
            ${isFullscreen ? 'fixed inset-4 z-50' : ''}
        `}>
            {/* 标题栏 */}
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                    <h3 className="text-lg font-semibold text-slate-100 m-0">全球实时追踪</h3>
                    <div className="flex items-center gap-4 text-xs">
                        <span className="flex items-center gap-1.5">
                            <span className="w-2 h-2 rounded-full bg-emerald-500 shadow-lg shadow-emerald-500/50" />
                            <span className="text-slate-400">在线 {onlineCount}</span>
                        </span>
                        <span className="flex items-center gap-1.5">
                            <span className="w-2 h-2 rounded-full bg-slate-500" />
                            <span className="text-slate-400">离线 {offlineCount}</span>
                        </span>
                    </div>
                </div>
            </div>

            {/* 地图容器 */}
            <div
                className="rounded-xl overflow-hidden border border-slate-700/50"
                style={{ height: isFullscreen ? 'calc(100% - 60px)' : height }}
            >
                <MapContainer
                    center={center}
                    zoom={3}
                    style={{ height: '100%', width: '100%' }}
                    scrollWheelZoom={true}
                    zoomControl={false}
                    ref={setMapRef}
                >
                    {/* 暗色瓦片图层 */}
                    <TileLayer
                        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>'
                        url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
                    />

                    <ZoomControl position="topright" />

                    {/* 设备标记 */}
                    {devices.map((device) => {
                        if (!device.latitude || !device.longitude) return null;
                        const position: [number, number] = [device.latitude, device.longitude];
                        const isSelected = device.id === selectedDeviceId;

                        return (
                            <Marker
                                key={device.id}
                                position={position}
                                icon={createCustomIcon(device.status, isSelected)}
                                eventHandlers={{
                                    click: () => onDeviceSelect?.(device),
                                }}
                            >
                                <Popup className="custom-popup">
                                    <div className="p-2 min-w-[200px]">
                                        <div className="flex items-center gap-2 mb-3">
                                            <div className={`
                                                w-8 h-8 rounded-lg flex items-center justify-center
                                                ${device.status === 'online'
                                                    ? 'bg-emerald-500/20 text-emerald-400'
                                                    : 'bg-slate-500/20 text-slate-400'
                                                }
                                            `}>
                                                <EnvironmentOutlined />
                                            </div>
                                            <div>
                                                <h4 className="font-bold text-gray-800 m-0 text-sm">
                                                    {device.name}
                                                </h4>
                                                <p className="text-xs text-gray-500 m-0">{device.id}</p>
                                            </div>
                                        </div>

                                        <div className="space-y-2 text-xs">
                                            <div className="flex items-center justify-between">
                                                <span className="text-gray-500">状态</span>
                                                <span className={`
                                                    px-2 py-0.5 rounded-full font-medium
                                                    ${device.status === 'online'
                                                        ? 'bg-emerald-100 text-emerald-700'
                                                        : 'bg-gray-100 text-gray-600'
                                                    }
                                                `}>
                                                    {device.status === 'online' ? '在线' : '离线'}
                                                </span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span className="text-gray-500">电量</span>
                                                <span className={`font-medium ${device.battery > 50 ? 'text-emerald-600' :
                                                    device.battery > 20 ? 'text-amber-600' : 'text-rose-600'
                                                    }`}>
                                                    <ThunderboltOutlined className="mr-1" />
                                                    {device.battery}%
                                                </span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span className="text-gray-500">速度</span>
                                                <span className="text-gray-700 font-medium">
                                                    {device.speed} km/h
                                                </span>
                                            </div>
                                        </div>

                                        <button
                                            onClick={() => navigate(`/devices?id=${device.id}`)}
                                            className="mt-3 w-full text-xs bg-blue-600 text-white 
                                                       px-3 py-1.5 rounded-lg hover:bg-blue-700 
                                                       transition-colors font-medium"
                                        >
                                            查看详情
                                        </button>
                                    </div>
                                </Popup>
                            </Marker>
                        );
                    })}

                    <MapControls
                        onLocateAll={handleLocateAll}
                        isFullscreen={isFullscreen}
                        onToggleFullscreen={handleToggleFullscreen}
                    />
                </MapContainer>
            </div>

            {/* 全屏遮罩 */}
            {isFullscreen && (
                <div
                    className="fixed inset-0 bg-black/80 -z-10"
                    onClick={handleToggleFullscreen}
                />
            )}
        </div>
    );
};

export default EnhancedGlobalMap;
