import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Input, Button, message, Spin } from 'antd';
import { SearchOutlined, AimOutlined, EnvironmentOutlined } from '@ant-design/icons';
import { MapContainer, TileLayer, Marker, useMapEvents, useMap, ZoomControl } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { geocodeAddress } from '../api/geocoding';

interface LocationPickerProps {
    value?: { lat: number; lng: number } | null;
    onChange?: (location: { lat: number; lng: number } | null) => void;
    address?: string;
    onAddressChange?: (address: string) => void;
    placeholder?: string;
    height?: number;
    defaultCenter?: [number, number];
    defaultZoom?: number;
}

// 自定义标记图标
const createMarkerIcon = () => {
    return L.divIcon({
        className: 'location-picker-marker',
        html: `
      <div style="
        width: 32px;
        height: 32px;
        background: linear-gradient(135deg, #2563eb, #1d4ed8);
        border: 3px solid white;
        border-radius: 50% 50% 50% 0;
        transform: rotate(-45deg);
        box-shadow: 0 4px 12px rgba(37, 99, 235, 0.4);
        display: flex;
        align-items: center;
        justify-content: center;
      ">
        <div style="
          width: 8px;
          height: 8px;
          background: white;
          border-radius: 50%;
          transform: rotate(45deg);
        "></div>
      </div>
    `,
        iconSize: [32, 32],
        iconAnchor: [16, 32],
    });
};

// 地图点击处理器
const MapClickHandler: React.FC<{
    onLocationSelect: (lat: number, lng: number) => void;
}> = ({ onLocationSelect }) => {
    useMapEvents({
        click: (e) => {
            onLocationSelect(e.latlng.lat, e.latlng.lng);
        },
    });
    return null;
};

// 地图视图控制器
const MapViewController: React.FC<{
    center: [number, number] | null;
    zoom?: number;
}> = ({ center, zoom = 14 }) => {
    const map = useMap();

    useEffect(() => {
        if (center) {
            map.flyTo(center, zoom, { duration: 0.5 });
        }
    }, [center, zoom, map]);

    return null;
};

const LocationPicker: React.FC<LocationPickerProps> = ({
    value,
    onChange,
    address = '',
    onAddressChange,
    placeholder = '输入地址搜索...',
    height = 200,
    defaultCenter = [31.2304, 121.4737], // 默认上海
    defaultZoom = 12,
}) => {
    const [searchValue, setSearchValue] = useState(address);
    const [loading, setLoading] = useState(false);
    const [mapCenter, setMapCenter] = useState<[number, number] | null>(null);
    const markerRef = useRef<L.Marker>(null);

    // 同步外部地址变化
    useEffect(() => {
        setSearchValue(address);
    }, [address]);

    // 搜索地址
    const handleSearch = useCallback(async () => {
        if (!searchValue.trim()) {
            message.warning('请输入地址');
            return;
        }

        setLoading(true);
        try {
            const result = await geocodeAddress(searchValue);
            if (result) {
                const newLocation = { lat: result.lat, lng: result.lng };
                onChange?.(newLocation);
                setMapCenter([result.lat, result.lng]);
                message.success(`已定位: ${result.displayName.slice(0, 50)}...`);
            } else {
                message.warning('未找到该地址，请尝试更详细的描述或在地图上点击选择');
            }
        } catch (error) {
            message.error('地址查询失败，请稍后重试');
        } finally {
            setLoading(false);
        }
    }, [searchValue, onChange]);

    // 地图点击选择位置
    const handleMapClick = useCallback((lat: number, lng: number) => {
        const newLocation = { lat, lng };
        onChange?.(newLocation);
    }, [onChange]);

    // 使用当前位置（如果浏览器支持）
    const handleUseCurrentLocation = useCallback(() => {
        if (!navigator.geolocation) {
            message.warning('您的浏览器不支持定位功能');
            return;
        }

        setLoading(true);
        navigator.geolocation.getCurrentPosition(
            (position) => {
                const newLocation = {
                    lat: position.coords.latitude,
                    lng: position.coords.longitude,
                };
                onChange?.(newLocation);
                setMapCenter([newLocation.lat, newLocation.lng]);
                setLoading(false);
                message.success('已使用当前位置');
            },
            (error) => {
                setLoading(false);
                message.error('无法获取当前位置');
                console.error('Geolocation error:', error);
            },
            { enableHighAccuracy: true, timeout: 10000 }
        );
    }, [onChange]);

    // 地址输入变化
    const handleAddressInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const newAddress = e.target.value;
        setSearchValue(newAddress);
        onAddressChange?.(newAddress);
    };

    // 输入框回车搜索
    const handleKeyPress = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handleSearch();
        }
    };

    const markerPosition = value ? [value.lat, value.lng] as [number, number] : null;
    const center = markerPosition || defaultCenter;

    return (
        <div className="location-picker">
            {/* 地址搜索栏 */}
            <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                <Input
                    value={searchValue}
                    onChange={handleAddressInputChange}
                    onKeyPress={handleKeyPress}
                    placeholder={placeholder}
                    prefix={<EnvironmentOutlined style={{ color: '#999' }} />}
                    style={{ flex: 1 }}
                    allowClear
                />
                <Button
                    icon={<SearchOutlined />}
                    onClick={handleSearch}
                    loading={loading}
                >
                    查询
                </Button>
                <Button
                    icon={<AimOutlined />}
                    onClick={handleUseCurrentLocation}
                    title="使用当前位置"
                />
            </div>

            {/* 地图区域 */}
            <Spin spinning={loading}>
                <div style={{
                    height,
                    borderRadius: 8,
                    overflow: 'hidden',
                    border: '1px solid #d9d9d9',
                }}>
                    <MapContainer
                        center={center}
                        zoom={defaultZoom}
                        style={{ height: '100%', width: '100%' }}
                        scrollWheelZoom={true}
                        zoomControl={false}
                    >
                        <ZoomControl position="topright" />
                        <TileLayer
                            attribution='&copy; <a href="https://carto.com/attributions">CARTO</a>'
                            url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
                        />

                        <MapClickHandler onLocationSelect={handleMapClick} />
                        {mapCenter && <MapViewController center={mapCenter} />}

                        {markerPosition && (
                            <Marker
                                ref={markerRef}
                                position={markerPosition}
                                icon={createMarkerIcon()}
                                draggable={true}
                                eventHandlers={{
                                    dragend: (e) => {
                                        const marker = e.target;
                                        const pos = marker.getLatLng();
                                        onChange?.({ lat: pos.lat, lng: pos.lng });
                                    },
                                }}
                            />
                        )}
                    </MapContainer>
                </div>
            </Spin>

            {/* 坐标显示 */}
            {value && (
                <div style={{
                    marginTop: 8,
                    fontSize: 12,
                    color: '#666',
                    display: 'flex',
                    gap: 16,
                }}>
                    <span>纬度: <strong>{value.lat.toFixed(6)}</strong></span>
                    <span>经度: <strong>{value.lng.toFixed(6)}</strong></span>
                    <span style={{ color: '#999', fontSize: 11 }}>
                        (点击地图或拖拽标记可调整位置)
                    </span>
                </div>
            )}

            <style>{`
        .location-picker-marker {
          background: transparent !important;
          border: none !important;
        }
      `}</style>
        </div>
    );
};

export default LocationPicker;
