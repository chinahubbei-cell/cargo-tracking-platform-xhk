import React, { useMemo, useEffect, useState } from 'react';
import { MapContainer, TileLayer, Polyline, Marker, Popup, ZoomControl, useMap, Circle } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import type { Shipment, ShipmentStage, PortGeofence } from '../types';
import api from '../api/client';
import { gcj02ToWgs84 } from '../utils/coordTransform';

interface ShipmentMapProps {
    shipments: Shipment[];
    selectedShipment: Shipment | null;
    onSelectShipment: (shipment: Shipment) => void;
}

// 本地轨迹点类型（匹配API返回的lat/lng格式）
interface ShipmentTrackPoint {
    id: number;
    device_id: string;
    lat: number;
    lng: number;
    speed: number;
    direction: number;
    temperature?: number;
    humidity?: number;
    locate_time: string;
}

// 轨迹数据类型
interface ShipmentTrackData {
    tracks: ShipmentTrackPoint[];
    currentPosition: {
        lat: number;
        lng: number;
        speed: number;
        timestamp: string;
    } | null;
}

// 计算贝塞尔曲线点（航线效果）
const calculateCurvePoints = (
    origin: [number, number],
    dest: [number, number],
    numPoints: number = 50
): [number, number][] => {
    const points: [number, number][] = [];
    const midLat = (origin[0] + dest[0]) / 2;
    const midLng = (origin[1] + dest[1]) / 2;

    // 计算曲线高度（距离越远，曲线越高）
    const distance = Math.sqrt(
        Math.pow(dest[0] - origin[0], 2) + Math.pow(dest[1] - origin[1], 2)
    );
    const curveHeight = Math.min(distance * 0.3, 30);

    for (let i = 0; i <= numPoints; i++) {
        const t = i / numPoints;
        const lat = Math.pow(1 - t, 2) * origin[0] +
            2 * (1 - t) * t * (midLat + curveHeight) +
            Math.pow(t, 2) * dest[0];
        const lng = Math.pow(1 - t, 2) * origin[1] +
            2 * (1 - t) * t * midLng +
            Math.pow(t, 2) * dest[1];
        points.push([lat, lng]);
    }
    return points;
};

// 创建端点图标
const createEndpointIcon = (type: 'origin' | 'dest', isSelected: boolean) => {
    const color = type === 'origin' ? '#2563eb' : '#10b981';
    const size = isSelected ? 16 : 12;

    return L.divIcon({
        className: 'endpoint-marker',
        html: `
            <div style="
                width: ${size}px;
                height: ${size}px;
                background: ${color};
                border: 2px solid white;
                border-radius: 50%;
                box-shadow: 0 2px 6px ${color}60;
                ${isSelected ? 'animation: pulse-marker 1.5s infinite;' : ''}
            "></div>
        `,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
    });
};

// 创建货车图标（运输中）
const createTruckIcon = (isSelected: boolean, speed?: number) => {
    const size = isSelected ? 32 : 24;
    const speedText = speed !== undefined ? `${speed.toFixed(0)} km/h` : '';

    return L.divIcon({
        className: 'truck-marker',
        html: `
            <div style="
                width: ${size}px;
                height: ${size}px;
                background: linear-gradient(135deg, #2563eb, #1d4ed8);
                border: 2px solid white;
                border-radius: 8px;
                display: flex;
                align-items: center;
                justify-content: center;
                box-shadow: 0 4px 12px rgba(37, 99, 235, 0.4);
                ${isSelected ? 'animation: bounce-marker 1s infinite;' : ''}
                position: relative;
            ">
                <span style="font-size: ${size * 0.5}px;">🚚</span>
                ${speedText ? `<span style="
                    position: absolute;
                    bottom: -16px;
                    left: 50%;
                    transform: translateX(-50%);
                    background: rgba(0,0,0,0.7);
                    color: white;
                    font-size: 10px;
                    padding: 1px 4px;
                    border-radius: 3px;
                    white-space: nowrap;
                ">${speedText}</span>` : ''}
            </div>
        `,
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2],
        popupAnchor: [0, -size / 2],
    });
};

// 地图视图控制器
const MapViewController: React.FC<{ shipment: Shipment | null; shipments: Shipment[]; trackData: ShipmentTrackData | null }> = ({
    shipment,
    shipments,
    trackData
}) => {
    const map = useMap();

    useEffect(() => {
        // 延迟执行确保地图容器渲染完成
        const timer = setTimeout(() => {
            // 确保地图容器尺寸正确
            map.invalidateSize();

            if (shipment && shipment.origin_lat && shipment.origin_lng) {
                // 选中运单：显示发货地 + 当前位置/目的地 + 轨迹
                const allPoints: [number, number][] = [];

                // 1. 始终添加发货地
                allPoints.push([shipment.origin_lat, shipment.origin_lng]);

                // 2. 添加当前位置
                if (trackData?.currentPosition?.lat && trackData?.currentPosition?.lng) {
                    allPoints.push([trackData.currentPosition.lat, trackData.currentPosition.lng]);
                }

                // 3. 如果没有当前位置，或者已送达，添加目的地
                if ((!trackData?.currentPosition || shipment.status === 'delivered') && shipment.dest_lat && shipment.dest_lng) {
                    allPoints.push([shipment.dest_lat, shipment.dest_lng]);
                }

                // 4. 添加所有轨迹点（确保每个点都有效）
                if (trackData?.tracks && trackData.tracks.length > 0) {
                    trackData.tracks.forEach(t => {
                        if (t.lat && t.lng && !isNaN(t.lat) && !isNaN(t.lng)) {
                            allPoints.push([t.lat, t.lng]);
                        }
                    });
                }

                console.log('[MapViewController] 选中运单:', shipment.id, '总点数:', allPoints.length);

                if (allPoints.length >= 2) {
                    const bounds = L.latLngBounds(allPoints);
                    // 使用较大的 padding 确保边缘内容可见
                    map.fitBounds(bounds, {
                        padding: [40, 40],
                        animate: false  // 禁用动画避免干扰
                    });
                } else if (allPoints.length === 1) {
                    map.setView(allPoints[0], 10);
                }
            } else if (shipments.length > 0) {
                // 未选中运单：显示所有运单的发货地和目的地
                const allPoints: [number, number][] = [];
                shipments.forEach(s => {
                    if (s.origin_lat && s.origin_lng) allPoints.push([s.origin_lat, s.origin_lng]);
                    if (s.dest_lat && s.dest_lng) allPoints.push([s.dest_lat, s.dest_lng]);
                });

                console.log('[MapViewController] 全部运单，总点数:', allPoints.length);

                if (allPoints.length > 0) {
                    const bounds = L.latLngBounds(allPoints);
                    map.fitBounds(bounds, {
                        padding: [30, 30],
                        animate: false
                    });
                }
            }
        }, 200);  // 延迟200ms确保数据加载

        return () => clearTimeout(timer);
    }, [shipment?.id, shipments.length, map, trackData?.tracks?.length, trackData?.currentPosition?.lat]);

    return null;
};

const ShipmentMap: React.FC<ShipmentMapProps> = ({
    shipments,
    selectedShipment,
    onSelectShipment,
}) => {
    // 选中运单的轨迹数据
    const [trackData, setTrackData] = useState<ShipmentTrackData | null>(null);
    const [trackLoading, setTrackLoading] = useState(false);

    // 选中运单的环节数据（用于多段路线渲染）
    const [stagesData, setStagesData] = useState<ShipmentStage[]>([]);

    // 路由几何数据缓存 (key: lat,lng_lat,lng)
    const [routeGeometries, setRouteGeometries] = useState<Record<string, [number, number][]>>({});

    // 港口围栏数据 - 只显示与当前运单关联的港口围栏
    const [portGeofences, setPortGeofences] = useState<PortGeofence[]>([]);

    // 地址缓存
    const [addressCache, setAddressCache] = useState<Record<string, { zh: string; en: string }>>({});
    // 当前选中运单的当前位置地址
    const [currentAddress, setCurrentAddress] = useState<string>('');
    // const [currentAddressEn, setCurrentAddressEn] = useState<string>(''); // Unused

    // 当轨迹数据更新时，获取当前地理地址（逆编码）- 通过后端API代理
    useEffect(() => {
        const fetchAddress = async () => {
            if (!trackData?.currentPosition) {
                setCurrentAddress('');
                return;
            }

            const { lat, lng } = trackData.currentPosition;
            // 转换为 WGS-84
            const [wgsLng, wgsLat] = gcj02ToWgs84(lng, lat);
            // 使用精度截断来增加缓存命中率
            const cacheKey = `${wgsLat.toFixed(5)},${wgsLng.toFixed(5)}`;

            if (addressCache[cacheKey]) {
                setCurrentAddress(addressCache[cacheKey].zh);
                return;
            }

            try {
                const res = await api.reverseGeocode(wgsLat, wgsLng);
                const addressZh = (res.success && res.display_name) ? res.display_name : '无法识别当前位置地址';
                setCurrentAddress(addressZh);
                setAddressCache(prev => ({ ...prev, [cacheKey]: { zh: addressZh, en: addressZh } }));
            } catch (err) {
                console.error('Failed to fetch address:', err);
                if (!currentAddress) setCurrentAddress('获取地址失败');
            }
        };

        fetchAddress();
    }, [trackData?.currentPosition?.lat, trackData?.currentPosition?.lng]);

    // 获取与当前运单关联的港口围栏数据（基于运单的stages港口代码）
    useEffect(() => {
        const fetchRelevantGeofences = async () => {
            // 只有选中运单且有stages数据时才获取相关围栏
            if (!selectedShipment || stagesData.length === 0) {
                setPortGeofences([]);
                return;
            }

            // 提取运单stages中的港口代码
            const portCodes = stagesData
                .filter(stage => stage.port_code)
                .map(stage => stage.port_code as string);

            if (portCodes.length === 0) {
                setPortGeofences([]);
                return;
            }

            try {
                const res = await api.getPortGeofences();
                if (res.success && res.data) {
                    // 只保留与当前运单stages相关的港口围栏
                    const relevantGeofences = res.data.filter(gf => portCodes.includes(gf.code));
                    setPortGeofences(relevantGeofences);
                }
            } catch (err) {
                console.error('Failed to fetch port geofences:', err);
                setPortGeofences([]);
            }
        };
        fetchRelevantGeofences();
    }, [selectedShipment?.id, stagesData]);

    // 获取选中运单的轨迹数据
    useEffect(() => {
        const fetchTrackData = async () => {
            // 对于已完成/取消的运单，device_id可能为空（已解绑），但后端能查到历史设备
            // 所以只要有运单ID就应该尝试请求
            if (!selectedShipment?.id) {
                setTrackData(null);
                return;
            }

            // 只有当不仅没有device_id，且状态也不是终态时，才认为是真的没有设备
            if (!selectedShipment.device_id && selectedShipment.status !== 'delivered' && selectedShipment.status !== 'cancelled') {
                setTrackData(null);
                return;
            }

            setTrackLoading(true);
            try {
                const res = await api.getShipmentTracks(selectedShipment.id);
                if (res.success && res.data) {
                    setTrackData({
                        tracks: res.data.tracks || [],
                        currentPosition: res.data.current_position,
                    });
                } else {
                    setTrackData(null);
                }
            } catch (err) {
                console.error('Failed to fetch shipment tracks:', err);
                setTrackData(null);
            } finally {
                setTrackLoading(false);
            }
        };

        fetchTrackData();
        // 定时刷新轨迹，避免页面长时间停留后轨迹点不更新
        const interval = window.setInterval(fetchTrackData, 30000);
        return () => window.clearInterval(interval);
    }, [selectedShipment?.id, selectedShipment?.device_id]);

    // 获取选中运单的环节数据（用于多段路线渲染）
    useEffect(() => {
        const fetchStagesData = async () => {
            if (!selectedShipment?.id) {
                setStagesData([]);
                return;
            }
            try {
                const res = await api.getShipmentStages(selectedShipment.id);
                if (res.success && res.data?.stages) {
                    setStagesData(res.data.stages);
                } else {
                    setStagesData([]);
                }
            } catch (err) {
                console.error('Failed to fetch shipment stages:', err);
                setStagesData([]);
            }
        };
        fetchStagesData();
    }, [selectedShipment?.id]);

    // 获取驾车导航路径 (OSRM)
    const fetchDrivingRoute = async (from: [number, number], to: [number, number]) => {
        const key = `${from[0]},${from[1]}_${to[0]},${to[1]}`;
        if (routeGeometries[key]) return;

        // 超时控制
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 10000); // 10秒超时

        try {
            // OSRM Public Demo Server (Note: Use responsibly, or use internal OSRM)
            // Coord format: lng,lat
            const url = `https://router.project-osrm.org/route/v1/driving/${from[1]},${from[0]};${to[1]},${to[0]}?overview=full&geometries=geojson`;
            const response = await fetch(url, { signal: controller.signal });

            if (!response.ok) {
                console.warn(`OSRM request failed: ${response.status}`);
                return;
            }

            const data = await response.json();

            if (data.code === 'Ok' && data.routes && data.routes.length > 0) {
                const geometry = data.routes[0].geometry;
                if (geometry && geometry.coordinates) {
                    // Convert [lng, lat] to [lat, lng]
                    const path: [number, number][] = geometry.coordinates.map((coord: number[]) => [coord[1], coord[0]]);
                    setRouteGeometries(prev => ({ ...prev, [key]: path }));
                }
            }
        } catch (error) {
            if (error instanceof Error && error.name === 'AbortError') {
                console.warn('OSRM request timeout');
            } else {
                console.error('Failed to fetch driving route:', error);
            }
        } finally {
            clearTimeout(timeoutId);
        }
    };

    // 为陆运分段获取详细路径
    useEffect(() => {
        if (!stagesData || stagesData.length === 0 || !selectedShipment) return;

        // 守卫检查：确保运单坐标有效
        if (!selectedShipment.origin_lat || !selectedShipment.origin_lng ||
            !selectedShipment.dest_lat || !selectedShipment.dest_lng) {
            return;
        }

        const origin: [number, number] = [selectedShipment.origin_lat, selectedShipment.origin_lng];
        const dest: [number, number] = [selectedShipment.dest_lat, selectedShipment.dest_lng];

        // 陆运特殊处理：直接请求发货地到目的地的导航
        if (selectedShipment.transport_type === 'land') {
            fetchDrivingRoute(origin, dest);
            return;
        }

        const portsWithCoords = stagesData
            .filter(stage => stage.port_lat && stage.port_lng && stage.port_code)
            .sort((a, b) => a.stage_order - b.stage_order);

        if (portsWithCoords.length === 0) return;


        const firstPort = portsWithCoords[0];
        const lastPort = portsWithCoords[portsWithCoords.length - 1];

        // 1. First Mile (Origin -> First Port)
        if (firstPort.port_lat && firstPort.port_lng) {
            fetchDrivingRoute(origin, [firstPort.port_lat, firstPort.port_lng]);
        }

        // 2. Intermediate legs - check stage code?
        // Usually ports to ports are Sea/Air.

        // 3. Last Mile (Last Port -> Dest)
        if (lastPort.port_lat && lastPort.port_lng) {
            fetchDrivingRoute([lastPort.port_lat, lastPort.port_lng], dest);
        }

    }, [stagesData, selectedShipment]);

    // 获取设备当前位置（优先使用真实数据，否则使用进度计算）
    const getDevicePosition = (shipment: Shipment): { position: [number, number]; speed?: number; isReal: boolean } | null => {
        // [修复] 已送达状态下，强制显示在目的地，避免停留在围栏边缘
        if (shipment.status === 'delivered' && shipment.dest_lat && shipment.dest_lng) {
            return {
                position: [shipment.dest_lat, shipment.dest_lng],
                speed: 0,
                isReal: true,
            };
        }

        // 如果是选中的运单且有真实位置数据
        if (selectedShipment?.id === shipment.id && trackData?.currentPosition) {
            // GCJ-02 -> WGS-84 坐标转换（修复中国地图偏移）
            const [wgsLng, wgsLat] = gcj02ToWgs84(trackData.currentPosition.lng, trackData.currentPosition.lat);
            return {
                position: [wgsLat, wgsLng],
                speed: trackData.currentPosition.speed,
                isReal: true,
            };
        }

        // 否则使用进度计算位置（仅用于非选中运单的展示）
        if (!shipment.origin_lat || !shipment.origin_lng || !shipment.dest_lat || !shipment.dest_lng) {
            return null;
        }
        const progress = (shipment.progress || 0) / 100;
        const lat = shipment.origin_lat + (shipment.dest_lat - shipment.origin_lat) * progress;
        const lng = shipment.origin_lng + (shipment.dest_lng - shipment.origin_lng) * progress;
        return { position: [lat, lng], isReal: false };
    };

    // 将轨迹点转换为polyline路径（应用 GCJ-02 -> WGS-84 转换）
    const getTrackPath = (): [number, number][] => {
        if (!trackData?.tracks || trackData.tracks.length === 0) return [];
        return trackData.tracks
            .filter(t => t.lat && t.lng)
            .map(t => {
                const [wgsLng, wgsLat] = gcj02ToWgs84(t.lng, t.lat);
                return [wgsLat, wgsLng] as [number, number];
            });
    };

    // 过滤有效运单
    const validShipments = useMemo(() =>
        shipments.filter(s => s.origin_lat && s.origin_lng && s.dest_lat && s.dest_lng),
        [shipments]
    );

    // 缓存贝塞尔曲线计算结果 (避免每次渲染时重新计算)
    const curvePointsCache = useMemo(() => {
        const cache = new Map<string, [number, number][]>();
        validShipments.forEach(shipment => {
            const key = `${shipment.origin_lat},${shipment.origin_lng}_${shipment.dest_lat},${shipment.dest_lng}`;
            if (!cache.has(key)) {
                const origin: [number, number] = [shipment.origin_lat!, shipment.origin_lng!];
                const dest: [number, number] = [shipment.dest_lat!, shipment.dest_lng!];
                cache.set(key, calculateCurvePoints(origin, dest));
            }
        });
        return cache;
    }, [validShipments]);

    // 获取缓存的曲线点或即时计算
    const getCachedCurvePoints = (origin: [number, number], dest: [number, number]) => {
        const key = `${origin[0]},${origin[1]}_${dest[0]},${dest[1]}`;
        return curvePointsCache.get(key) || calculateCurvePoints(origin, dest);
    };

    const center: [number, number] = [30, 110]; // 中国中心
    const trackPath = getTrackPath();

    return (
        <div style={{ flex: 1, minHeight: 0, position: 'relative' }}>
            {/* ... MapContainer ... */}
            <MapContainer
                center={center}
                zoom={4}
                style={{ height: '100%', width: '100%', position: 'absolute', top: 0, left: 0 }}
                scrollWheelZoom={true}
                zoomControl={false}
            >
                {/* 浅色地图瓦片 */}
                <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>'
                    url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
                />

                <ZoomControl position="topright" />
                <MapViewController shipment={selectedShipment} shipments={validShipments} trackData={trackData} />

                {/* 渲染所有运单路线 */}
                {validShipments.map((shipment) => {
                    const isSelected = selectedShipment?.id === shipment.id;
                    const origin: [number, number] = [shipment.origin_lat!, shipment.origin_lng!];
                    const dest: [number, number] = [shipment.dest_lat!, shipment.dest_lng!];
                    const devicePos = getDevicePosition(shipment);

                    // 根据运输类型确定路线样式
                    const transportType = shipment.transport_type || 'air';

                    // 简易路线样式配置 (用于未选中或无详细数据时)
                    const routeConfig = {
                        // 陆运：绿色虚线直连
                        land: {
                            points: [origin, dest],
                            color: '#10b981',
                            dashArray: '6, 6',
                            weight: isSelected ? 4 : 2,
                        },
                        // 海运：蓝色虚线曲线
                        sea: {
                            points: getCachedCurvePoints(origin, dest),
                            color: '#2563eb',
                            dashArray: '10, 10',
                            weight: isSelected ? 4 : 2,
                        },
                        // 空运：蓝色虚线曲线（弧度更大）
                        air: {
                            points: getCachedCurvePoints(origin, dest),
                            color: '#1890ff',
                            dashArray: '8, 8',
                            weight: isSelected ? 4 : 2,
                        },
                        // 多式联运：紫色虚线曲线
                        multimodal: {
                            points: getCachedCurvePoints(origin, dest),
                            color: '#722ed1',
                            dashArray: '12, 6',
                            weight: isSelected ? 4 : 2,
                        },
                    };

                    const config = routeConfig[transportType as keyof typeof routeConfig] || routeConfig.air;

                    // 判断是否应该显示复杂路线（多段/即时导航）
                    // 1. 必须选中当前运单
                    // 2. 必须有环节数据
                    // 3. 必须是陆运（陆运总是有特殊逻辑）或者有带坐标的有效港口（空运/海运）
                    const hasValidPorts = isSelected && stagesData.some(s => s.port_lat && s.port_lng && s.port_code);
                    const showComplexRoute = isSelected && stagesData.length > 0 && (transportType === 'land' || hasValidPorts);

                    return (
                        <React.Fragment key={shipment.id}>
                            {/* 计划路线（根据运输类型差异化显示） */}
                            {/* 如果满足显示复杂路线的条件，则隐藏此简易连线 */}
                            {!showComplexRoute && (
                                <Polyline
                                    positions={config.points}
                                    pathOptions={{
                                        color: config.color,
                                        weight: config.weight,
                                        opacity: isSelected ? 0.9 : 0.5,
                                        dashArray: config.dashArray,
                                    }}
                                    eventHandlers={{
                                        click: () => onSelectShipment(shipment),
                                    }}
                                />
                            )}

                            {/* 真实轨迹路径（仅选中运单且有轨迹数据时显示） */}
                            {isSelected && trackPath.length > 0 && (
                                <Polyline
                                    positions={trackPath}
                                    pathOptions={{
                                        color: '#2563eb',
                                        weight: 3,
                                        opacity: 1,
                                    }}
                                />
                            )}

                            {/* 起点/终点/围栏标记 (省略，保持不变) */}
                            <Marker position={origin} icon={createEndpointIcon('origin', isSelected)}>
                                <Popup><div><strong>起点：{shipment.origin}</strong></div></Popup>
                            </Marker>
                            {isSelected && shipment.origin_radius && (
                                <Circle center={origin} radius={shipment.origin_radius} pathOptions={{ color: '#2563eb', fillColor: '#2563eb', fillOpacity: 0.1, weight: 2, dashArray: '5, 5' }} />
                            )}
                            <Marker position={dest} icon={createEndpointIcon('dest', isSelected)}>
                                <Popup><div><strong>终点：{shipment.destination}</strong></div></Popup>
                            </Marker>
                            {isSelected && shipment.dest_radius && (
                                <Circle center={dest} radius={shipment.dest_radius} pathOptions={{ color: '#10b981', fillColor: '#10b981', fillOpacity: 0.1, weight: 2, dashArray: '5, 5' }} />
                            )}

                            {/* 多段路线渲染：选中运单且有港口坐标的环节数据时显示 */}
                            {isSelected && stagesData.length > 0 && (() => {
                                // 筛选有港口坐标的环节
                                const portsWithCoords = stagesData
                                    .filter(stage => stage.port_lat && stage.port_lng && stage.port_code)
                                    .sort((a, b) => a.stage_order - b.stage_order);

                                if (portsWithCoords.length === 0) return null;

                                // 生成多段路线：起点 -> 港口1 -> 港口2 -> ... -> 终点
                                const segments: { from: [number, number]; to: [number, number]; stageCode: string }[] = [];

                                // 针对陆运 (land) 的特殊处理：不显示中间港口，直接显示发货地到目的地的导航
                                if (shipment.transport_type === 'land') {
                                    // 陆运直接渲染 Origin -> Dest
                                    const key = `${origin[0]},${origin[1]}_${dest[0]},${dest[1]}`;
                                    const detailedPath = routeGeometries[key];
                                    return (
                                        <Polyline
                                            positions={detailedPath || [origin, dest]}
                                            pathOptions={{
                                                color: '#10b981',
                                                weight: detailedPath ? 4 : 3,
                                                opacity: 0.8,
                                                dashArray: '6, 6',
                                            }}
                                        />
                                    );
                                }

                                // 起点 -> 第一个港口
                                const firstPort = portsWithCoords[0];
                                segments.push({
                                    from: origin,
                                    to: [firstPort.port_lat as number, firstPort.port_lng as number],
                                    stageCode: firstPort.stage_code,
                                });

                                // 港口之间的连接
                                for (let i = 0; i < portsWithCoords.length - 1; i++) {
                                    const currPort = portsWithCoords[i];
                                    const nextPort = portsWithCoords[i + 1];
                                    segments.push({
                                        from: [currPort.port_lat as number, currPort.port_lng as number],
                                        to: [nextPort.port_lat as number, nextPort.port_lng as number],
                                        stageCode: nextPort.stage_code,
                                    });
                                }

                                // 最后一个港口 -> 终点
                                const lastPort = portsWithCoords[portsWithCoords.length - 1];
                                segments.push({
                                    from: [lastPort.port_lat as number, lastPort.port_lng as number],
                                    to: dest,
                                    stageCode: 'last_mile',
                                });

                                // 根据环节类型确定线条样式
                                const getSegmentStyle = (stageCode: string) => {
                                    // 陆运段：使用绿色虚线
                                    if (['first_mile', 'last_mile', 'pre_transit', 'pickup', 'delivery', 'origin_port'].includes(stageCode)) {
                                        return { color: '#10b981', dashArray: '6, 6', isLand: true };
                                    }
                                    // 海运/空运段：使用蓝色虚线
                                    if (['main_line', 'transit_port', 'dest_port'].includes(stageCode)) {
                                        return { color: '#2563eb', dashArray: '10, 10', isLand: false };
                                    }
                                    return { color: '#722ed1', dashArray: '8, 8', isLand: false };
                                };

                                return (
                                    <>
                                        {/* 多段路线 */}
                                        {segments.map((seg, idx) => {
                                            const style = getSegmentStyle(seg.stageCode);
                                            // Check if we have detailed geometry for Land segments
                                            const key = `${seg.from[0]},${seg.from[1]}_${seg.to[0]},${seg.to[1]}`;
                                            const detailedPath = style.isLand ? routeGeometries[key] : null;

                                            const points = detailedPath || [seg.from, seg.to];

                                            return (
                                                <Polyline
                                                    key={`segment-${idx}`}
                                                    positions={points}
                                                    pathOptions={{
                                                        color: style.color,
                                                        weight: detailedPath ? 4 : 3, // Thicker if detailed
                                                        opacity: 0.8,
                                                        dashArray: style.dashArray,
                                                    }}
                                                />
                                            );
                                        })}

                                        {/* 港口标记 */}
                                        {portsWithCoords.map((stage, idx) => (
                                            <Marker
                                                key={`port-${stage.id}`}
                                                position={[stage.port_lat!, stage.port_lng!]}
                                                icon={L.divIcon({
                                                    className: 'port-marker',
                                                    html: `<div style="width:16px;height:16px;background:#faad14;border-radius:50%;border:2px solid white;box-shadow:0 2px 4px rgba(0,0,0,0.3);display:flex;align-items:center;justify-content:center;font-size:10px;color:white;font-weight:bold;">${idx + 1}</div>`,
                                                    iconSize: [16, 16],
                                                    iconAnchor: [8, 8],
                                                })}
                                            >
                                                <Popup>
                                                    <div style={{ padding: '4px 0' }}>
                                                        <strong>中转港 {idx + 1}：{stage.port_name || stage.port_code}</strong>
                                                        <div style={{ fontSize: '12px', color: '#666', marginTop: '4px' }}>
                                                            {stage.stage_name} - {stage.status === 'completed' ? '✓ 已完成' : stage.status === 'in_progress' ? '进行中' : '待处理'}
                                                        </div>
                                                    </div>
                                                </Popup>
                                            </Marker>
                                        ))}
                                    </>
                                );
                            })()}
                            {/* 设备当前位置（在途/待发/已送达状态都显示）- 移除device_id强校验，依靠devicePos是否存在 */}
                            {devicePos && (shipment.status === 'in_transit' || shipment.status === 'pending' || shipment.status === 'delivered') && (
                                <Marker
                                    position={devicePos.position}
                                    icon={createTruckIcon(isSelected, devicePos.speed)}
                                    eventHandlers={{
                                        click: () => onSelectShipment(shipment),
                                    }}
                                >
                                    <Popup>
                                        <div style={{ padding: '8px', minWidth: '180px' }}>
                                            <div style={{ fontWeight: 600, marginBottom: '8px' }}>
                                                {shipment.id}
                                            </div>
                                            <div style={{ fontSize: '12px', color: '#666' }}>

                                                {shipment.status === 'delivered' ? (
                                                    <div style={{ marginTop: '4px', color: '#10b981', fontWeight: 600 }}>
                                                        ✓ 已送达
                                                    </div>
                                                ) : devicePos.isReal ? (
                                                    <>
                                                        {currentAddress && (
                                                            <div style={{
                                                                fontSize: '11px',
                                                                color: '#666',
                                                                lineHeight: '1.4',
                                                                maxHeight: '60px',
                                                                overflow: 'hidden',
                                                                textOverflow: 'ellipsis',
                                                                display: '-webkit-box',
                                                                WebkitLineClamp: 3,
                                                                WebkitBoxOrient: 'vertical'
                                                            }}>
                                                                📍 {currentAddress}
                                                            </div>
                                                        )}
                                                        <div style={{ marginTop: '4px', color: '#10b981' }}>
                                                            ✓ 实时位置
                                                        </div>
                                                        {devicePos.speed !== undefined && (
                                                            <div>速度：{devicePos.speed.toFixed(1)} km/h</div>
                                                        )}
                                                        {trackData?.currentPosition?.timestamp && (
                                                            <div style={{ fontSize: '11px', color: '#999' }}>
                                                                更新于：{new Date(trackData.currentPosition.timestamp).toLocaleString()}
                                                            </div>
                                                        )}
                                                    </>
                                                ) : (
                                                    <div style={{ marginTop: '4px', color: '#faad14' }}>
                                                        ⚠ 估算位置 (进度 {shipment.progress}%)
                                                    </div>
                                                )}
                                            </div>
                                            {/* 发货地 -> 目的地 (移到底部) */}
                                            <div style={{
                                                marginTop: '8px',
                                                paddingTop: '8px',
                                                borderTop: '1px solid #eee',
                                                display: 'flex',
                                                alignItems: 'center',
                                                gap: '6px'
                                            }}>
                                                <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#2563eb' }}></div>
                                                <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#10b981' }}></div>
                                                <span style={{ marginLeft: '4px' }}>{shipment.origin} → {shipment.destination}</span>
                                            </div>
                                        </div>
                                    </Popup>
                                </Marker>
                            )}
                        </React.Fragment>
                    );
                })}

                {/* 港口围栏渲染 - 只在选中运单时显示相关的港口围栏 */}
                {
                    selectedShipment && portGeofences.map((geofence) => (
                        <Circle
                            key={geofence.id}
                            center={[geofence.center_lat, geofence.center_lng]}
                            radius={geofence.radius}
                            pathOptions={{
                                color: geofence.color || '#1890ff',
                                fillColor: geofence.color || '#1890ff',
                                fillOpacity: 0.08,
                                weight: 1.5,
                                dashArray: '4, 4',
                            }}
                        >
                            <Popup>
                                <div style={{ minWidth: 120 }}>
                                    <strong>{geofence.name_cn || geofence.name}</strong>
                                    <div style={{ color: '#666', fontSize: 12 }}>{geofence.code}</div>
                                    <div style={{ color: '#888', fontSize: 11, marginTop: 4 }}>
                                        围栏半径: {(geofence.radius / 1000).toFixed(1)} km
                                    </div>
                                </div>
                            </Popup>
                        </Circle>
                    ))
                }
            </MapContainer >

            {/* 轨迹加载指示器 */}
            {
                trackLoading && (
                    <div style={{
                        position: 'absolute',
                        top: '10px',
                        left: '10px',
                        background: 'rgba(255,255,255,0.9)',
                        padding: '8px 12px',
                        borderRadius: '4px',
                        fontSize: '12px',
                        boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                        zIndex: 1000
                    }}>
                        🔄 加载轨迹...
                    </div>
                )
            }

            {/* 轨迹数据状态指示 - 只要有轨迹数据或者正在加载就准备显示，或者运单是终态 */}
            {
                selectedShipment && (selectedShipment.device_id || trackData || selectedShipment.status === 'delivered') && !trackLoading && (
                    <div style={{
                        position: 'absolute',
                        top: '10px',
                        left: '10px',
                        background: 'rgba(255,255,255,0.9)',
                        padding: '8px 12px',
                        borderRadius: '4px',
                        fontSize: '12px',
                        boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                        zIndex: 1000
                    }}>
                        {trackData?.currentPosition ? (
                            <span style={{ color: '#10b981' }}>
                                ✓ 设备在线 · {trackPath.length} 轨迹点
                            </span>
                        ) : trackData?.tracks && trackData.tracks.length > 0 ? (
                            <span style={{ color: '#faad14' }}>
                                ⚠ 离线 · {trackPath.length} 历史轨迹点
                            </span>
                        ) : (
                            <span style={{ color: '#999' }}>
                                暂无轨迹数据
                            </span>
                        )}
                    </div>
                )
            }

            {/* 地图图例 */}
            <div style={{
                position: 'absolute',
                bottom: '20px',
                left: '10px',
                background: 'rgba(255,255,255,0.95)',
                padding: '10px 14px',
                borderRadius: '6px',
                fontSize: '12px',
                boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
                zIndex: 1000
            }}>
                <div style={{ fontWeight: 600, marginBottom: '8px', color: '#333' }}>图例</div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    {/* 运输类型图例 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <svg width="24" height="10">
                            <line x1="0" y1="5" x2="24" y2="5" stroke="#2563eb" strokeWidth="2" strokeDasharray="5,5" />
                        </svg>
                        <span style={{ color: '#666' }}>海运</span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <svg width="24" height="10">
                            <line x1="0" y1="5" x2="24" y2="5" stroke="#1890ff" strokeWidth="2" strokeDasharray="4,4" />
                        </svg>
                        <span style={{ color: '#666' }}>空运</span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <svg width="24" height="10">
                            <line x1="0" y1="5" x2="24" y2="5" stroke="#10b981" strokeWidth="2" strokeDasharray="3,3" />
                        </svg>
                        <span style={{ color: '#666' }}>陆运</span>
                    </div>
                    {/* 轨迹与端点图例 */}
                    <div style={{ borderTop: '1px solid #eee', marginTop: '4px', paddingTop: '6px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <svg width="24" height="10">
                            <line x1="0" y1="5" x2="24" y2="5" stroke="#2563eb" strokeWidth="3" />
                        </svg>
                        <span style={{ color: '#666' }}>实际轨迹</span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <div style={{ width: '12px', height: '12px', background: '#2563eb', borderRadius: '50%', border: '2px solid white', boxShadow: '0 1px 3px rgba(0,0,0,0.3)' }} />
                        <span style={{ color: '#666' }}>发货地</span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <div style={{ width: '12px', height: '12px', background: '#10b981', borderRadius: '50%', border: '2px solid white', boxShadow: '0 1px 3px rgba(0,0,0,0.3)' }} />
                        <span style={{ color: '#666' }}>目的地</span>
                    </div>
                </div>
            </div>

            {/* 地图样式注入 */}
            <style>{`
                @keyframes pulse-marker {
                    0%, 100% { transform: scale(1); }
                    50% { transform: scale(1.2); }
                }
                @keyframes bounce-marker {
                    0%, 100% { transform: translateY(0); }
                    50% { transform: translateY(-4px); }
                }
                .leaflet-popup-content-wrapper {
                    border-radius: 8px;
                    box-shadow: 0 4px 16px rgba(0,0,0,0.15);
                }
            `}</style>
        </div >
    );
};

export default ShipmentMap;
