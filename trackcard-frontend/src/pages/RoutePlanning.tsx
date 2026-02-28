import React, { useState, useEffect } from 'react';
import { Card, Form, Button, Radio, Table, Tag, Space, message, Spin, Empty, Select, InputNumber, Divider } from 'antd';
import { SwapRightOutlined, ThunderboltOutlined, DollarOutlined, SafetyOutlined } from '@ant-design/icons';
import { MapContainer, TileLayer, Marker, Polyline, Popup, useMap, ZoomControl } from 'react-leaflet';
import { useNavigate } from 'react-router-dom';
import L from 'leaflet';
import { api } from '../api/client';
import axios from 'axios'; // Keep axios for isAxiosError check if needed, or remove if using client types
import 'leaflet/dist/leaflet.css';
import AddressInput from '../components/AddressInput';

// 类型定义
interface RouteSegment {
    type: string;
    from: string;
    to: string;
    from_lat: number;
    from_lng: number;
    to_lat: number;
    to_lng: number;
    mode: string;
    carrier?: string;
    days: number;
    cost: number;
    distance_km?: number;
    transit_ports?: string[];
}

interface RouteRecommendation {
    type: string;
    label: string;
    total_days: number;
    total_cost: number;
    device_coverage?: string;
    segments: RouteSegment[];
}

interface CalculateResponse {
    origin: string;
    destination: string;
    routes: RouteRecommendation[];
}

// 运输方式图标
const transportIcons: Record<string, string> = {
    truck: '🚚',
    ocean: '🚢',
    air: '✈️',
    rail: '🚂',
};

// 创建自定义图标
const createIcon = (color: string, size: number = 14) => L.divIcon({
    className: 'custom-marker',
    html: `<div style="width:${size}px;height:${size}px;background:${color};border:2px solid white;border-radius:50%;box-shadow:0 2px 6px rgba(0,0,0,0.3);"></div>`,
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2],
});

// 地图自动适配组件
const MapFitBounds: React.FC<{ bounds: L.LatLngBoundsExpression | null }> = ({ bounds }) => {
    const map = useMap();

    useEffect(() => {
        if (bounds) {
            map.fitBounds(bounds, { padding: [30, 30], maxZoom: 4 });
        }
    }, [bounds, map]);

    return null;
};

const RoutePlanning: React.FC = () => {
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<CalculateResponse | null>(null);
    const [selectedRoute, setSelectedRoute] = useState<string>('fastest');
    const [selectedSegmentIndex, setSelectedSegmentIndex] = useState<number | null>(null);
    const [transportMode, setTransportMode] = useState<string>('lcl');
    const [transportType, setTransportType] = useState<string>('sea'); // 运输类型: sea/air/land/multimodal
    const [currency, setCurrency] = useState<string>('CNY');
    const navigate = useNavigate();

    // 货币符号
    const currencySymbol = currency === 'USD' ? '$' : '¥';

    // 提取地址的前两级（省+市或国家+城市）
    const extractShortAddress = (address: string): string => {
        if (!address) return '';

        // 判断是否为中国地址
        const isChineseAddress = /[\u4e00-\u9fa5]/.test(address) &&
            (address.includes('省') || address.includes('市') || address.includes('区') || address.includes('县'));

        if (isChineseAddress) {
            // 中国地址：提取省+市
            // 格式可能是：贵州省铜仁市碧江区xxx 或 广东省广州市番禺区xxx
            const provinceMatch = address.match(/([\u4e00-\u9fa5]+省)/);
            const cityMatch = address.match(/([\u4e00-\u9fa5]+市)/);
            const directCityMatch = address.match(/^(北京|上海|天津|重庆)/);

            if (directCityMatch) {
                // 直辖市
                return directCityMatch[1];
            } else if (provinceMatch && cityMatch) {
                return provinceMatch[1] + cityMatch[1];
            } else if (cityMatch) {
                return cityMatch[1];
            } else if (provinceMatch) {
                return provinceMatch[1];
            }
        } else {
            // 海外地址：取前两个逗号分隔的部分或城市名
            const parts = address.split(',').map(p => p.trim());
            if (parts.length >= 2) {
                // 取城市和国家
                return parts.slice(-2).join(', ');
            }
        }

        // 如果无法解析，返回原地址的前20个字符
        return address.length > 20 ? address.substring(0, 20) + '...' : address;
    };

    // 应用到运单 - 跳转并传递地址信息和线路规划数据
    const handleApplyToShipment = () => {
        if (!result) return;
        const currentRouteData = result.routes?.find(r => r.type === selectedRoute);
        const formValues = form.getFieldsValue();

        // 从线路规划中提取承运商信息
        let carrier = '';
        if (currentRouteData?.segments) {
            const lineHaul = currentRouteData.segments.find((s: { type: string; carrier?: string }) => s.type === 'line_haul');
            if (lineHaul?.carrier) {
                carrier = lineHaul.carrier;
            }
        }

        // 从线路规划中提取起止坐标
        let origin_lat: number | undefined;
        let origin_lng: number | undefined;
        let dest_lat: number | undefined;
        let dest_lng: number | undefined;

        if (currentRouteData?.segments && currentRouteData.segments.length > 0) {
            // 起点坐标：取第一个segment的from坐标
            const firstSegment = currentRouteData.segments[0];
            origin_lat = firstSegment.from_lat;
            origin_lng = firstSegment.from_lng;

            // 终点坐标：取最后一个segment的to坐标
            const lastSegment = currentRouteData.segments[currentRouteData.segments.length - 1];
            dest_lat = lastSegment.to_lat;
            dest_lng = lastSegment.to_lng;
        }

        // 计算预计到达时间 (ETD=今天, ETA=今天+天数)
        const today = new Date();
        const etd = today.toISOString().split('T')[0];
        const eta = currentRouteData?.total_days
            ? new Date(today.getTime() + currentRouteData.total_days * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
            : undefined;

        // 提取简短地址（省+市）
        const originShort = extractShortAddress(result.origin);
        const destShort = extractShortAddress(result.destination);

        navigate('/business/shipments', {
            state: {
                fromRoutePlanning: true,
                // 地址信息 - origin/destination 只取二级
                origin: originShort,
                destination: destShort,
                origin_address: result.origin,  // 详细地址
                dest_address: result.destination, // 详细地址
                // 坐标信息 - 从线路规划提取
                origin_lat,
                origin_lng,
                dest_lat,
                dest_lng,
                // 线路规划数据
                routePlan: currentRouteData,
                // 运输参数 - 全部传递
                transport_type: formValues.transport_type || transportType,
                transport_mode: formValues.transport_mode || transportMode,
                container_type: formValues.container_type,
                // 货物参数
                weight_kg: formValues.weight_kg,
                volume_cbm: formValues.volume_cbm,
                quantity: formValues.quantity,
                cargo_type: formValues.cargo_type,
                // 费用和时间 - 运费四舍五入保留两位小数
                freight_cost: currentRouteData?.total_cost
                    ? Math.round(currentRouteData.total_cost * 100) / 100
                    : undefined,
                total_days: currentRouteData?.total_days,
                etd: etd,
                eta: eta,
                // 承运商
                carrier: carrier,
                // 货币
                currency: formValues.currency || currency,
            }
        });
        message.success('正在跳转到运单创建页面...');
    };

    // 计算路径
    const handleCalculate = async (values: {
        origin: string;
        destination: string;
        transport_type: string;
        transport_mode: string;
        container_type?: string;
        weight_kg: number;
        volume_cbm?: number;
        quantity?: number;
        cargo_type: string;
        currency: string;
    }) => {
        setLoading(true);
        try {
            const res = await api.post('/route-planning/calculate', {
                origin: values.origin,
                destination: values.destination,
                transport_type: values.transport_type || 'sea',
                transport_mode: values.transport_mode || 'lcl',
                container_type: values.container_type || '20GP',
                weight_kg: values.weight_kg || 500,
                volume_cbm: values.volume_cbm || 1,
                quantity: values.quantity || 1,
                cargo_type: values.cargo_type || 'general',
                currency: values.currency || 'CNY',
            });
            if (res.success) {
                setResult(res.data);
                setCurrency(values.currency || 'CNY');
                message.success('AI规划线路完成');
            } else {
                message.error(res.error || res.message || '计算失败');
            }
        } catch (error: unknown) {
            // 提取详细错误信息
            if (axios.isAxiosError(error)) {
                const errorMsg = error.response?.data?.message || error.message || '请求失败';
                message.error(errorMsg);
            } else {
                message.error('请求失败');
            }
        } finally {
            setLoading(false);
        }
    };

    // 获取当前选中的路径
    const currentRoute = result?.routes?.find(r => r.type === selectedRoute);

    // 计算地图边界 - 使用当前路径的所有节点坐标或选中的段落
    const mapBounds: L.LatLngBoundsExpression | null = currentRoute && currentRoute.segments.length > 0 ? (() => {
        // 如果选中了某个段落，只显示该段落
        if (selectedSegmentIndex !== null && currentRoute.segments[selectedSegmentIndex]) {
            const seg = currentRoute.segments[selectedSegmentIndex];
            const latPadding = Math.abs(seg.to_lat - seg.from_lat) * 0.3 + 2;
            const lngPadding = Math.abs(seg.to_lng - seg.from_lng) * 0.3 + 5;
            return [
                [Math.min(seg.from_lat, seg.to_lat) - latPadding, Math.min(seg.from_lng, seg.to_lng) - lngPadding],
                [Math.max(seg.from_lat, seg.to_lat) + latPadding, Math.max(seg.from_lng, seg.to_lng) + lngPadding],
            ] as L.LatLngBoundsExpression;
        }
        // 否则显示全部路线
        const allLats = currentRoute.segments.flatMap(s => [s.from_lat, s.to_lat]);
        const allLngs = currentRoute.segments.flatMap(s => [s.from_lng, s.to_lng]);
        return [
            [Math.min(...allLats) - 5, Math.min(...allLngs) - 10],
            [Math.max(...allLats) + 5, Math.max(...allLngs) + 10],
        ] as L.LatLngBoundsExpression;
    })() : null;

    // 段落表格列
    const segmentColumns = [
        {
            title: '阶段',
            dataIndex: 'type',
            key: 'type',
            width: 80,
            render: (type: string) => {
                const labels: Record<string, string> = {
                    first_mile: '头程',
                    line_haul: '干线',
                    last_mile: '尾程',
                };
                return <Tag color="blue">{labels[type] || type}</Tag>;
            },
        },
        {
            title: '路线',
            key: 'route',
            render: (_: unknown, record: RouteSegment) => (
                <span>
                    {transportIcons[record.mode] || '📦'} {record.from} → {record.to}
                    {record.transit_ports && record.transit_ports.length > 0 && (
                        <Tag color="orange" style={{ marginLeft: 8 }}>经 {record.transit_ports.join(', ')}</Tag>
                    )}
                </span>
            ),
        },
        {
            title: '承运商',
            dataIndex: 'carrier',
            key: 'carrier',
            width: 100,
            render: (carrier: string) => carrier || '-',
        },
        {
            title: '天数',
            dataIndex: 'days',
            key: 'days',
            width: 70,
            render: (days: number) => `${days}天`,
        },
        {
            title: '费用',
            dataIndex: 'cost',
            key: 'cost',
            width: 90,
            render: (cost: number) => <span style={{ color: '#52c41a', fontWeight: 'bold' }}>{currencySymbol}{cost.toLocaleString()}</span>,
        },
    ];

    return (
        <div className="route-planning-page">
            {/* 输入区域 */}
            <Card title="线路自动化规划" size="small">
                <Form form={form} layout="vertical" onFinish={handleCalculate}>
                    {/* 第一行: 地址 + 运输类型 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16, flexWrap: 'wrap' }}>
                        {/* 地址输入 */}
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 600 }}>
                            <Form.Item name="origin" rules={[{ required: true, message: '请输入发货地址' }]} style={{ marginBottom: 0, flex: 1 }}>
                                <AddressInput
                                    placeholder="发货地址，如：深圳、shenzhen"
                                    style={{ width: '100%' }}
                                />
                            </Form.Item>
                            <SwapRightOutlined style={{ fontSize: 20, color: '#999', flexShrink: 0 }} />
                            <Form.Item name="destination" rules={[{ required: true, message: '请输入收货地址' }]} style={{ marginBottom: 0, flex: 1 }}>
                                <AddressInput
                                    placeholder="收货地址，如：洛杉矶、LA"
                                    style={{ width: '100%' }}
                                />
                            </Form.Item>
                        </div>
                    </div>

                    <Divider style={{ margin: '12px 0' }} />

                    {/* 第二行: 运输类型、运输模式和货物信息 */}
                    <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', alignItems: 'flex-end' }}>
                        {/* 运输类型选择器 - 下拉选择 */}
                        <Form.Item name="transport_type" label="运输类型" initialValue="sea" style={{ marginBottom: 0 }}>
                            <Select
                                style={{ width: 140 }}
                                onChange={(value) => {
                                    setTransportType(value);
                                    if (value === 'air') {
                                        setTransportMode('lcl');
                                        form.setFieldValue('transport_mode', 'lcl');
                                    }
                                }}
                            >
                                <Select.Option value="sea">🚢 海运</Select.Option>
                                <Select.Option value="air">✈️ 空运</Select.Option>
                                <Select.Option value="land">🚚 陆运</Select.Option>
                                <Select.Option value="multimodal">🔄 多式联运</Select.Option>
                            </Select>
                        </Form.Item>

                        {/* 运输模式 - 空运无FCL概念 */}
                        {transportType !== 'air' && (
                            <Form.Item name="transport_mode" label="运输模式" initialValue="lcl" style={{ marginBottom: 0 }}>
                                <Radio.Group
                                    onChange={(e) => setTransportMode(e.target.value)}
                                    buttonStyle="solid"
                                >
                                    <Radio.Button value="lcl">📦 零担 LCL</Radio.Button>
                                    {transportType === 'land' ? (
                                        <Radio.Button value="ftl">🚚 整车 FTL</Radio.Button>
                                    ) : (
                                        <Radio.Button value="fcl">🚢 整柜 FCL</Radio.Button>
                                    )}
                                </Radio.Group>
                            </Form.Item>
                        )}

                        {/* 柜型 - 仅FCL模式显示 */}
                        {transportMode === 'fcl' && transportType !== 'air' && transportType !== 'land' && (
                            <Form.Item name="container_type" label="柜型" initialValue="20GP" style={{ marginBottom: 0 }}>
                                <Select style={{ width: 120 }}>
                                    <Select.Option value="20GP">20GP</Select.Option>
                                    <Select.Option value="40GP">40GP</Select.Option>
                                    <Select.Option value="40HQ">40HQ</Select.Option>
                                </Select>
                            </Form.Item>
                        )}

                        {/* 体积和件数 - LCL/空运模式显示 */}
                        {(transportMode === 'lcl' || transportType === 'air') && (
                            <>
                                <Form.Item name="volume_cbm" label="体积(CBM)" initialValue={1} style={{ marginBottom: 0 }}>
                                    <InputNumber min={0.1} step={0.5} style={{ width: 100 }} />
                                </Form.Item>
                                <Form.Item name="quantity" label="件数" initialValue={1} style={{ marginBottom: 0 }}>
                                    <InputNumber min={1} style={{ width: 80 }} />
                                </Form.Item>
                            </>
                        )}

                        <Form.Item name="weight_kg" label="重量(KG)" initialValue={500} style={{ marginBottom: 0 }}>
                            <InputNumber min={1} style={{ width: 100 }} />
                        </Form.Item>

                        <Form.Item name="cargo_type" label="货物类型" initialValue="general" style={{ marginBottom: 0 }}>
                            <Select style={{ width: 100 }}>
                                <Select.Option value="general">普货</Select.Option>
                                <Select.Option value="dangerous">危险品</Select.Option>
                                <Select.Option value="cold_chain">冷链</Select.Option>
                            </Select>
                        </Form.Item>

                        <Form.Item name="currency" label="货币" initialValue="CNY" style={{ marginBottom: 0, minWidth: 120 }}>
                            <Select style={{ width: 110 }} popupMatchSelectWidth={false}>
                                <Select.Option value="CNY">🇨🇳 CNY</Select.Option>
                                <Select.Option value="USD">🇺🇸 USD</Select.Option>
                            </Select>
                        </Form.Item>

                        <Form.Item style={{ marginBottom: 0 }}>
                            <Button type="primary" htmlType="submit" loading={loading} icon={<ThunderboltOutlined />}>
                                🔍 智能规划
                            </Button>
                        </Form.Item>
                    </div>
                </Form>
            </Card>

            {/* 结果区域 */}
            {loading ? (
                <Card style={{ textAlign: 'center', padding: 60, marginTop: 16 }}>
                    <Spin size="large" tip="正在计算最优路径..." />
                </Card>
            ) : result ? (
                <>
                    {/* 地图区域 - 展示完整路线 */}
                    <Card style={{ marginTop: 16, padding: 0 }} bodyStyle={{ padding: 0 }}>
                        <div style={{ height: 400 }}>
                            <MapContainer
                                center={[20, 0]}
                                zoom={2}
                                style={{ height: '100%', width: '100%' }}
                                zoomControl={false}
                            >
                                <ZoomControl position="topright" />
                                <TileLayer url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png" />
                                <MapFitBounds bounds={mapBounds} />

                                {/* 起点 */}
                                {currentRoute && currentRoute.segments.length > 0 && (
                                    <Marker position={[currentRoute.segments[0].from_lat, currentRoute.segments[0].from_lng]} icon={createIcon('#52c41a', 16)}>
                                        <Popup><strong>📍 发货地</strong><br />{result.origin}</Popup>
                                    </Marker>
                                )}

                                {/* 动态渲染路径段标记和线段 */}
                                {currentRoute?.segments.map((segment, idx) => {
                                    const segmentColors: Record<string, string> = {
                                        first_mile: '#52c41a',
                                        line_haul: '#1890ff',
                                        last_mile: '#f5222d',
                                    };
                                    const color = segmentColors[segment.type] || '#1890ff';
                                    const isOcean = segment.mode === 'ocean';

                                    return (
                                        <React.Fragment key={idx}>
                                            {/* 终点标记 (不包括起点，因为已经在第一个段落的from处渲染) */}
                                            <Marker position={[segment.to_lat, segment.to_lng]} icon={createIcon(color, idx === currentRoute.segments.length - 1 ? 16 : 12)}>
                                                <Popup>
                                                    <strong>{segment.type === 'last_mile' ? '📍 收货地' : segment.mode === 'ocean' ? '🚢 目的港' : '📍 中转点'}</strong>
                                                    <br />{segment.to}
                                                </Popup>
                                            </Marker>
                                            {/* 路径线 */}
                                            <Polyline
                                                positions={[[segment.from_lat, segment.from_lng], [segment.to_lat, segment.to_lng]]}
                                                color={color}
                                                weight={3}
                                                dashArray={isOcean ? '8,4' : undefined}
                                            />
                                        </React.Fragment>
                                    );
                                })}
                            </MapContainer>
                        </div>
                    </Card>

                    {/* 路径选择 */}
                    <Card style={{ marginTop: 16 }} size="small">
                        <Radio.Group value={selectedRoute} onChange={(e) => { setSelectedRoute(e.target.value); setSelectedSegmentIndex(null); }} buttonStyle="solid">
                            {result.routes.map((route) => (
                                <Radio.Button key={route.type} value={route.type}>
                                    <Space>
                                        {route.type === 'fastest' && <ThunderboltOutlined />}
                                        {route.type === 'cheapest' && <DollarOutlined />}
                                        {route.type === 'safest' && <SafetyOutlined />}
                                        <span>{route.label}</span>
                                        <Tag color={route.type === 'fastest' ? 'red' : route.type === 'cheapest' ? 'green' : 'blue'}>
                                            {route.total_days}天 / {currencySymbol}{route.total_cost.toLocaleString()}
                                        </Tag>
                                        {route.device_coverage && <Tag color="purple">{route.device_coverage}</Tag>}
                                    </Space>
                                </Radio.Button>
                            ))}
                        </Radio.Group>
                    </Card>

                    {/* 路径详情 */}
                    {currentRoute && (
                        <Card
                            title={`${currentRoute.label} - 总计 ${currentRoute.total_days}天 / ${currencySymbol}${currentRoute.total_cost.toLocaleString()}`}
                            style={{ marginTop: 16 }}
                            size="small"
                            extra={<Space><Button size="small">导出</Button><Button size="small" type="primary" onClick={handleApplyToShipment}>应用到运单</Button></Space>}
                        >
                            <Table
                                dataSource={currentRoute.segments}
                                columns={segmentColumns}
                                rowKey={(_, index) => `segment-${index}`}
                                pagination={false}
                                size="small"
                                onRow={(_, index) => ({
                                    onClick: () => {
                                        // 点击同一行则取消选中（恢复全景），否则选中该行
                                        setSelectedSegmentIndex(prev => prev === index ? null : (index ?? null));
                                    },
                                    style: {
                                        cursor: 'pointer',
                                        backgroundColor: selectedSegmentIndex === index ? '#e6f7ff' : undefined,
                                    }
                                })}
                            />
                        </Card>
                    )}
                </>
            ) : (
                <Card style={{ marginTop: 16, textAlign: 'center', padding: 60 }}>
                    <Empty
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                        description={<span style={{ color: '#666' }}>输入起止地址，系统将自动推荐 <strong>最快</strong>、<strong>经济</strong>、<strong>安全</strong> 三种路径</span>}
                    />
                </Card>
            )}
        </div>
    );
};

export default RoutePlanning;
