import React, { useEffect, useMemo } from 'react';
import { MapContainer, TileLayer, Marker, Popup, Polyline, useMap, ZoomControl } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { gcj02ToWgs84 } from '../utils/coordTransform';

interface TrackPoint {
    latitude: number;
    longitude: number;
    speed: number;
    timestamp: string;
    temperature: number;
}

interface Props {
    trackData: TrackPoint[];
    currentIndex: number;
}

// 转换单个坐标点
const convertPoint = (point: TrackPoint): TrackPoint => {
    const [wgsLng, wgsLat] = gcj02ToWgs84(point.longitude, point.latitude);
    return {
        ...point,
        longitude: wgsLng,
        latitude: wgsLat,
    };
};

// 自动适应轨迹边界的组件
const FitBoundsToTrack: React.FC<{ trackData: TrackPoint[] }> = ({ trackData }) => {
    const map = useMap();

    useEffect(() => {
        if (trackData.length > 0) {
            const bounds = L.latLngBounds(
                trackData.map(p => [p.latitude, p.longitude] as [number, number])
            );
            // 减少边距，提高最大缩放，使轨迹更好填充视野
            map.fitBounds(bounds, { padding: [20, 20], maxZoom: 18 });
        }
    }, [map, trackData]);

    return null;
};

const TrackMap: React.FC<Props> = ({ trackData, currentIndex }) => {
    if (!trackData.length) return null;

    // 使用useMemo缓存转换后的数据，避免每次渲染都重新计算
    const convertedTrackData = useMemo(() => {
        return trackData.map(convertPoint);
    }, [trackData]);

    const currentPoint = convertedTrackData[currentIndex];
    const passedPoints = convertedTrackData.slice(0, currentIndex + 1);
    const remainingPoints = convertedTrackData.slice(currentIndex);

    // 当前位置图标
    const currentIcon = L.divIcon({
        className: 'track-current-marker',
        html: `<div style="
            width: 24px;
            height: 24px;
            background: #2563eb;
            border: 3px solid white;
            border-radius: 50%;
            box-shadow: 0 2px 8px rgba(0,0,0,0.3);
        "></div>`,
        iconSize: [24, 24],
        iconAnchor: [12, 12],
    });

    // 起点图标
    const startIcon = L.divIcon({
        className: 'track-start-marker',
        html: `<div style="
            width: 16px;
            height: 16px;
            background: #10b981;
            border: 2px solid white;
            border-radius: 50%;
        "></div>`,
        iconSize: [16, 16],
        iconAnchor: [8, 8],
    });

    // 终点图标
    const endIcon = L.divIcon({
        className: 'track-end-marker',
        html: `<div style="
            width: 16px;
            height: 16px;
            background: #ef4444;
            border: 2px solid white;
            border-radius: 50%;
        "></div>`,
        iconSize: [16, 16],
        iconAnchor: [8, 8],
    });

    return (
        <MapContainer
            center={[convertedTrackData[0].latitude, convertedTrackData[0].longitude]}
            zoom={6}
            style={{ height: '100%', width: '100%' }}
            zoomControl={false}
        >
            <ZoomControl position="topright" />
            <TileLayer
                url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
                attribution='&copy; <a href="https://carto.com/">CartoDB</a>'
            />

            {/* 自动适应轨迹边界 */}
            <FitBoundsToTrack trackData={convertedTrackData} />

            {/* 待播放轨迹（浅灰色虚线，在底层） */}
            {remainingPoints.length > 1 && (
                <Polyline
                    positions={remainingPoints.map(p => [p.latitude, p.longitude])}
                    color="#faad14"
                    dashArray="10, 10"
                    weight={5}
                    opacity={0.8}
                />
            )}

            {/* 已播放轨迹（深蓝色实线，在顶层） */}
            <Polyline
                positions={passedPoints.map(p => [p.latitude, p.longitude])}
                color="#2563eb"
                weight={6}
                opacity={0.95}
            />

            {/* 起点标记 */}
            <Marker
                position={[trackData[0].latitude, trackData[0].longitude]}
                icon={startIcon}
            >
                <Popup>
                    <div style={{ fontSize: 12 }}>
                        <strong>起点</strong>
                        <div>时间: {new Date(trackData[0].timestamp).toLocaleString('zh-CN')}</div>
                    </div>
                </Popup>
            </Marker>

            {/* 当前位置标记 */}
            {currentPoint && (
                <Marker
                    position={[currentPoint.latitude, currentPoint.longitude]}
                    icon={currentIcon}
                >
                    <Popup>
                        <div style={{ fontSize: 12 }}>
                            <strong>当前位置</strong>
                            <div>时间: {new Date(currentPoint.timestamp).toLocaleString('zh-CN')}</div>
                            <div>速度: {currentPoint.speed.toFixed(1)} km/h</div>
                            <div>温度: {currentPoint.temperature.toFixed(1)}°C</div>
                        </div>
                    </Popup>
                </Marker>
            )}

            {/* 终点标记 */}
            <Marker
                position={[trackData[trackData.length - 1].latitude, trackData[trackData.length - 1].longitude]}
                icon={endIcon}
            >
                <Popup>
                    <div style={{ fontSize: 12 }}>
                        <strong>终点</strong>
                        <div>时间: {new Date(trackData[trackData.length - 1].timestamp).toLocaleString('zh-CN')}</div>
                    </div>
                </Popup>
            </Marker>
        </MapContainer>
    );
};

export default TrackMap;

