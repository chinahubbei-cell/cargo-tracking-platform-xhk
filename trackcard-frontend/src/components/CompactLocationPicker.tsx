import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Input, Button, Modal, message, Spin, Space } from 'antd';
import { SearchOutlined, AimOutlined, EnvironmentOutlined } from '@ant-design/icons';
import { MapContainer, TileLayer, Marker, useMapEvents, useMap, ZoomControl } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { geocodeAddress } from '../api/geocoding';

interface CompactLocationPickerProps {
    value?: { lat: number; lng: number } | null;
    onChange?: (location: { lat: number; lng: number } | null) => void;
    placeholder?: string;
    label?: string;
}

// 自定义标记图标
const createMarkerIcon = (color: string = '#2563eb') => {
    return L.divIcon({
        className: 'location-picker-marker',
        html: `
      <div style="
        width: 32px;
        height: 32px;
        background: linear-gradient(135deg, ${color}, ${color}dd);
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

const CompactLocationPicker: React.FC<CompactLocationPickerProps> = ({
    value,
    onChange,
    placeholder = '输入地址搜索或点击图标选择',
    label,
}) => {
    const [modalVisible, setModalVisible] = useState(false);
    const [searchValue, setSearchValue] = useState('');
    const [loading, setLoading] = useState(false);
    const [tempLocation, setTempLocation] = useState<{ lat: number; lng: number } | null>(null);
    const [mapCenter, setMapCenter] = useState<[number, number] | null>(null);
    const markerRef = useRef<L.Marker>(null);

    const defaultCenter: [number, number] = [31.2304, 121.4737]; // 上海

    // 打开Modal时同步当前值
    useEffect(() => {
        if (modalVisible) {
            setTempLocation(value || null);
            if (value) {
                setMapCenter([value.lat, value.lng]);
            }
        }
    }, [modalVisible, value]);

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
                setTempLocation(newLocation);
                setMapCenter([result.lat, result.lng]);
                message.success(`已定位: ${result.displayName.slice(0, 40)}...`);
            } else {
                message.warning('未找到该地址，请在地图上点击选择');
            }
        } catch (error) {
            message.error('地址查询失败');
        } finally {
            setLoading(false);
        }
    }, [searchValue]);

    // 地图点击选择位置
    const handleMapClick = useCallback((lat: number, lng: number) => {
        setTempLocation({ lat, lng });
    }, []);

    // 使用当前位置
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
                setTempLocation(newLocation);
                setMapCenter([newLocation.lat, newLocation.lng]);
                setLoading(false);
                message.success('已使用当前位置');
            },
            () => {
                setLoading(false);
                message.error('无法获取当前位置');
            },
            { enableHighAccuracy: true, timeout: 10000 }
        );
    }, []);

    // 确认选择
    const handleConfirm = () => {
        onChange?.(tempLocation);
        setModalVisible(false);
    };

    // 清除位置
    const handleClear = () => {
        setTempLocation(null);
        onChange?.(null);
    };

    const markerPosition = tempLocation ? [tempLocation.lat, tempLocation.lng] as [number, number] : null;
    const center = markerPosition || defaultCenter;

    // 格式化坐标显示
    const formatCoords = (loc: { lat: number; lng: number } | null) => {
        if (!loc) return '';
        return `${loc.lat.toFixed(4)}, ${loc.lng.toFixed(4)}`;
    };

    return (
        <>
            {/* 紧凑显示 */}
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                {label && <span style={{ minWidth: 60, color: '#666' }}>{label}:</span>}
                <Input
                    value={value ? formatCoords(value) : ''}
                    placeholder={placeholder}
                    readOnly
                    prefix={<EnvironmentOutlined style={{ color: value ? '#2563eb' : '#999' }} />}
                    style={{ flex: 1, cursor: 'pointer' }}
                    onClick={() => setModalVisible(true)}
                />
                <Button
                    icon={<AimOutlined />}
                    onClick={() => setModalVisible(true)}
                    type={value ? 'primary' : 'default'}
                    ghost={!!value}
                    title="选择位置"
                />
                {value && (
                    <Button size="small" type="text" danger onClick={handleClear}>
                        清除
                    </Button>
                )}
            </div>

            {/* 地图选择Modal */}
            <Modal
                title="选择位置"
                open={modalVisible}
                onCancel={() => setModalVisible(false)}
                onOk={handleConfirm}
                okText="确认位置"
                cancelText="取消"
                width={700}
                destroyOnClose
            >
                {/* 搜索栏 */}
                <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                    <Input
                        value={searchValue}
                        onChange={(e) => setSearchValue(e.target.value)}
                        onPressEnter={handleSearch}
                        placeholder="输入地址搜索..."
                        prefix={<EnvironmentOutlined style={{ color: '#999' }} />}
                        style={{ flex: 1 }}
                        allowClear
                    />
                    <Button icon={<SearchOutlined />} onClick={handleSearch} loading={loading}>
                        搜索
                    </Button>
                    <Button icon={<AimOutlined />} onClick={handleUseCurrentLocation} title="当前位置" />
                </div>

                {/* 地图 */}
                <Spin spinning={loading}>
                    <div style={{
                        height: 350,
                        borderRadius: 8,
                        overflow: 'hidden',
                        border: '1px solid #d9d9d9',
                    }}>
                        <MapContainer
                            center={center}
                            zoom={12}
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
                                            setTempLocation({ lat: pos.lat, lng: pos.lng });
                                        },
                                    }}
                                />
                            )}
                        </MapContainer>
                    </div>
                </Spin>

                {/* 坐标显示 */}
                {tempLocation && (
                    <div style={{
                        marginTop: 12,
                        padding: '8px 12px',
                        background: '#f5f5f5',
                        borderRadius: 6,
                        fontSize: 13,
                    }}>
                        <Space>
                            <span>已选择坐标:</span>
                            <strong style={{ color: '#2563eb' }}>
                                {tempLocation.lat.toFixed(6)}, {tempLocation.lng.toFixed(6)}
                            </strong>
                            <span style={{ color: '#999', fontSize: 11 }}>
                                (点击地图或拖拽标记可调整)
                            </span>
                        </Space>
                    </div>
                )}
            </Modal>

            <style>{`
                .location-picker-marker {
                    background: transparent !important;
                    border: none !important;
                }
            `}</style>
        </>
    );
};

export default CompactLocationPicker;
