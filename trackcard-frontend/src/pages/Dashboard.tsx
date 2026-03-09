import React, { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Spin, message, Modal, Input, Dropdown, Badge, AutoComplete } from 'antd';
import {
    RocketOutlined,
    WarningOutlined,
    GlobalOutlined,
    CheckCircleOutlined,
    SyncOutlined,
    SearchOutlined,
    BellOutlined,
} from '@ant-design/icons';
import api from '../api/client';
import type { DashboardStats, Alert, Device, Shipment } from '../types';

// 组件导入
import StatsCard from '../components/Dashboard/StatsCard';
import GlobalMap from '../components/Dashboard/GlobalMap';
import AlertList from '../components/Dashboard/AlertList';
import TrendChart from '../components/Dashboard/TrendChart';
import DonutChart from '../components/Dashboard/DonutChart';
import ShipmentTable from '../components/Dashboard/ShipmentTable';

const TYPE_LABELS: Record<string, string> = {
    eta_delay: 'ETA延误',
    temp_high: '温度过高',
    temp_low: '温度过低',
    free_time_expiring: '免箱期即将到期',
    customs_hold: '海关扣留',
    shock_detected: '检测到震动',
    route_deviation: '路线偏离',
    battery_low: '电量低',
    offline: '设备离线',
    geofence_enter: '进入围栏',
    geofence_exit: '离开围栏',
    vessel_delay: '船期延误',
    carrier_stale: '船司数据过时',
    humidity_high: '高湿预警',
    tilt_detected: '货物倾斜',
    door_open: '非法开箱/光感告警',
    etd_not_departed: '货物未按时发出',
    free_time_expired: '免租期已逾期',
    eta_change: 'ETA大幅变更',
    missing_coordinates: '坐标信息缺失',
    no_origin_track: '设备首次上报即在围栏外',
};

const Dashboard: React.FC = () => {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [stats, setStats] = useState<DashboardStats | null>(null);
    const [alerts, setAlerts] = useState<Alert[]>([]);
    const [devices, setDevices] = useState<Device[]>([]);
    const [shipments, setShipments] = useState<Shipment[]>([]);
    // 告警详情弹窗状态
    const [alertModalVisible, setAlertModalVisible] = useState(false);
    const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);

    const handleAlertClick = (alert: Alert) => {
        setSelectedAlert(alert);
        setAlertModalVisible(true);
    };

    // 搜索模态框状态
    const [searchModalVisible, setSearchModalVisible] = useState(false);
    const [searchKeyword, setSearchKeyword] = useState('');
    const [searchSuggestions, setSearchSuggestions] = useState<Array<{ value: string; label: React.ReactNode; type: string; id: string }>>([]);
    const [searchLoading, setSearchLoading] = useState(false);

    // 搜索建议获取
    const fetchSearchSuggestions = useCallback(async (keyword: string) => {
        if (!keyword || keyword.length < 2) {
            setSearchSuggestions([]);
            return;
        }
        setSearchLoading(true);
        try {
            const result = await api.searchSuggestions(keyword);
            const options = result.suggestions.map((item) => ({
                value: item.label,
                type: item.type,
                id: item.id,
                label: (
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                            <span style={{ fontWeight: 500 }}>{item.label}</span>
                            {item.subLabel && <span style={{ color: '#888', marginLeft: 8, fontSize: 12 }}>{item.subLabel}</span>}
                        </div>
                        <span style={{ fontSize: 11, color: item.type === 'shipment' ? '#1890ff' : '#52c41a', background: item.type === 'shipment' ? '#e6f7ff' : '#f6ffed', padding: '2px 6px', borderRadius: 4 }}>
                            {item.type === 'shipment' ? '📦 运单' : '📡 设备'}
                        </span>
                    </div>
                ),
            }));
            setSearchSuggestions(options);
        } catch (error) {
            console.error('Failed to fetch suggestions:', error);
        } finally {
            setSearchLoading(false);
        }
    }, []);

    const handleSearch = (value?: string) => {
        const keyword = value || searchKeyword;
        if (keyword.trim()) {
            // 根据关键词跳转到对应页面
            if (keyword.startsWith('8681') || keyword.includes('设备')) {
                navigate(`/devices?search=${encodeURIComponent(keyword)}`);
            } else {
                navigate(`/shipments?search=${encodeURIComponent(keyword)}`);
            }
            setSearchModalVisible(false);
            setSearchKeyword('');
            setSearchSuggestions([]);
        }
    };

    const handleSelectSuggestion = (_value: string, option: any) => {
        if (option.type === 'shipment') {
            navigate(`/shipments?search=${encodeURIComponent(option.id)}`);
        } else {
            navigate(`/devices?search=${encodeURIComponent(option.id)}`);
        }
        setSearchModalVisible(false);
        setSearchKeyword('');
        setSearchSuggestions([]);
    };

    // 通知下拉菜单项
    const notificationItems = alerts.slice(0, 5).map((alert, index) => ({
        key: String(index),
        label: (
            <div
                style={{ padding: '8px 0', maxWidth: 280, cursor: 'pointer' }}
                onClick={() => {
                    handleAlertClick(alert);
                }}
            >
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{
                        width: 8,
                        height: 8,
                        borderRadius: '50%',
                        background: alert.severity === 'critical' ? '#ef4444' : alert.severity === 'warning' ? '#f59e0b' : '#3b82f6'
                    }} />
                    <span style={{ fontWeight: 500, fontSize: 13 }}>{alert.title}</span>
                </div>
                <div style={{ fontSize: 12, color: '#9ca3af', marginTop: 4, marginLeft: 16 }}>
                    {alert.message?.slice(0, 40)}...
                </div>
            </div>
        ),
    }));

    // 添加"查看全部"选项
    if (alerts.length > 0) {
        notificationItems.push({
            key: 'divider',
            label: <div style={{ borderTop: '1px solid #e5e7eb', margin: '8px 0' }} />,
        });
        notificationItems.push({
            key: 'viewAll',
            label: (
                <div
                    style={{ textAlign: 'center', color: '#2563eb', fontWeight: 500, cursor: 'pointer' }}
                    onClick={() => navigate('/alerts')}
                >
                    查看全部告警
                </div>
            ),
        });
    } else {
        notificationItems.push({
            key: 'empty',
            label: <div style={{ textAlign: 'center', color: '#9ca3af', padding: '16px 0' }}>暂无新告警</div>,
        });
    }

    // 生成模拟设备数据（全球运输概览用）
    const generateMockDevices = (): Device[] => {
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
        const providers = ['kuaihuoyun', 'sinoiov'];

        return Array.from({ length: 100 }, (_, i) => {
            const city = cities[i % cities.length];
            const isOnline = Math.random() > 0.2;
            return {
                id: `DEV-${String(i + 1).padStart(5, '0')}`,
                name: `设备-${city.name}-${i + 1}`,
                type: 'container',
                provider: providers[i % 2],
                status: isOnline ? 'online' : 'offline',
                battery: Math.floor(Math.random() * 100),
                latitude: city.lat + (Math.random() - 0.5) * 2,
                longitude: city.lng + (Math.random() - 0.5) * 2,
                external_device_id: `EXT${868120342395115 + i}`,
                speed: isOnline ? Math.floor(Math.random() * 80) : 0,
                direction: Math.floor(Math.random() * 360),
                temperature: 15 + Math.random() * 20,
                humidity: 40 + Math.random() * 40,
                last_update: new Date(Date.now() - Math.random() * 86400000).toISOString(),
                created_at: new Date(Date.now() - Math.random() * 86400000 * 30).toISOString(),
            } as Device;
        });
    };

    const loadData = useCallback(async (showMessage = false) => {
        try {
            if (showMessage) setRefreshing(true);

            const [statsRes, alertsRes, devicesRes, shipmentsRes] = await Promise.all([
                api.getDashboardStats(),
                api.getAlerts({ status: 'pending' }),
                api.getDashboardLocations(),
                api.getShipments({ limit: 5 }),
            ]);

            if (statsRes.data) setStats(statsRes.data);
            if (alertsRes.data) setAlerts(alertsRes.data);
            if (shipmentsRes.data) setShipments(shipmentsRes.data);

            // 始终使用模拟数据，如果API有真实数据则合并
            const mockData = generateMockDevices();
            if (devicesRes.data && devicesRes.data.length > 0) {
                setDevices([...devicesRes.data, ...mockData]);
            } else {
                setDevices(mockData);
            }

            if (showMessage) {
                message.success('数据刷新成功');
            }
        } catch (error) {
            console.error('加载数据失败:', error);
            // 加载失败时也使用模拟数据
            setDevices(generateMockDevices());
            if (showMessage) {
                message.error('数据刷新失败');
            }
        } finally {
            setLoading(false);
            setRefreshing(false);
        }
    }, []);

    useEffect(() => {
        loadData();
        const interval = setInterval(() => loadData(false), 30000);
        return () => clearInterval(interval);
    }, [loadData]);

    // 状态分布数据
    const shipmentStatusData = [
        { name: '运输中', value: stats?.shipments?.in_transit || 12, color: '#2563eb' },
        { name: '已送达', value: stats?.shipments?.delivered || 45, color: '#10b981' },
        { name: '待发运', value: stats?.shipments?.pending || 8, color: '#f59e0b' },
    ];

    if (loading) {
        return (
            <div className="dashboard-container" style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                minHeight: '80vh'
            }}>
                <div style={{ textAlign: 'center' }}>
                    <Spin size="large" />
                    <p style={{ marginTop: 16, color: 'var(--text-muted)' }}>正在加载数据...</p>
                </div>
            </div>
        );
    }

    return (
        <div className="dashboard-container">
            {/* ... headers ... */}
            <header style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 24,
                flexWrap: 'wrap',
                gap: 16
            }}>
                <div>
                    <h1 style={{
                        fontSize: 24,
                        fontWeight: 700,
                        color: 'var(--text-primary)',
                        margin: 0
                    }}>
                        数据概览
                    </h1>
                    <p style={{
                        fontSize: 14,
                        color: 'var(--text-tertiary)',
                        margin: '4px 0 0 0'
                    }}>
                        全球物流实时监控与数据分析
                    </p>
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    {/* 搜索框 */}
                    <div
                        onClick={() => setSearchModalVisible(true)}
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                            padding: '8px 14px',
                            background: 'var(--bg-primary)',
                            border: '1px solid var(--border-light)',
                            borderRadius: 8,
                            color: 'var(--text-muted)',
                            fontSize: 14,
                            minWidth: 200,
                            cursor: 'pointer',
                            transition: 'all 0.2s'
                        }}
                        onMouseEnter={(e) => e.currentTarget.style.borderColor = '#2563eb'}
                        onMouseLeave={(e) => e.currentTarget.style.borderColor = 'var(--border-light)'}
                    >
                        <SearchOutlined />
                        <span>搜索运单、设备...</span>
                    </div>

                    {/* 通知按钮 */}
                    <Dropdown
                        menu={{ items: notificationItems }}
                        placement="bottomRight"
                        trigger={['click']}
                    >
                        <Badge count={alerts.length} size="small">
                            <button
                                className="btn btn-secondary"
                                style={{ padding: 10, position: 'relative' }}
                                title="通知消息"
                            >
                                <BellOutlined style={{ fontSize: 18 }} />
                            </button>
                        </Badge>
                    </Dropdown>

                    {/* 刷新按钮 */}
                    <button
                        className="btn btn-primary"
                        onClick={() => loadData(true)}
                        disabled={refreshing}
                        style={{ display: 'flex', alignItems: 'center', gap: 8 }}
                    >
                        <SyncOutlined className={refreshing ? 'animate-spin' : ''} />
                        刷新数据
                    </button>
                </div>
            </header>

            {/* 统计卡片行 */}
            <section style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
                gap: 20,
                marginBottom: 24
            }}>
                <StatsCard
                    title="总运单数"
                    value={stats?.shipments?.total || 24500}
                    icon={<RocketOutlined />}
                    status="blue"
                    trend={12}
                    trendLabel="较上月"
                    delay={0}
                    onClick={() => navigate('/business/shipments')}
                />
                <StatsCard
                    title="准时送达率"
                    value={98.2}
                    suffix="%"
                    icon={<CheckCircleOutlined />}
                    status="green"
                    trend={1.5}
                    trendLabel="改善"
                    delay={100}
                />
                <StatsCard
                    title="在线设备"
                    value={stats?.devices?.online || 1250}
                    icon={<GlobalOutlined />}
                    status="blue"
                    delay={200}
                    onClick={() => navigate('/devices')}
                />
                <StatsCard
                    title="平均运输时间"
                    value={3.5}
                    suffix=" 天"
                    icon={<WarningOutlined />}
                    status="orange"
                    trend={-0.2}
                    trendLabel="缩短"
                    delay={300}
                />
            </section>

            {/* 地图和告警区域 */}
            <section style={{
                display: 'grid',
                gridTemplateColumns: '1fr 380px',
                gap: 20,
                marginBottom: 24,
                minHeight: 500
            }}>
                <GlobalMap devices={devices} height="calc(100vh - 380px)" />
                <AlertList
                    alerts={alerts}
                    maxItems={5}
                    onViewAll={() => navigate('/alerts')}
                    onAlertClick={handleAlertClick}
                />
            </section>

            {/* 图表区域 */}
            <section style={{
                display: 'grid',
                gridTemplateColumns: '1fr 320px',
                gap: 20,
                marginBottom: 24
            }}>
                <TrendChart
                    title="月度运输趋势"
                    subtitle="Monthly Shipment Volume"
                />
                <DonutChart
                    title="运单状态分布"
                    subtitle="Shipment Status Distribution"
                    data={shipmentStatusData}
                />
            </section>

            {/* 运单表格 */}
            <section>
                <ShipmentTable
                    shipments={shipments}
                    maxRows={4}
                    onViewAll={() => navigate('/business/shipments')}
                />
            </section>

            {/* 告警详情弹窗 */}
            <Modal
                title="告警详情"
                open={alertModalVisible}
                onCancel={() => setAlertModalVisible(false)}
                footer={null}
            >
                {selectedAlert && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                            <span className={`alert-badge ${selectedAlert.severity}`} style={{ fontSize: 14, padding: '4px 12px' }}>
                                {selectedAlert.severity === 'critical' ? '紧急' : selectedAlert.severity === 'warning' ? '警告' : '消息'}
                            </span>
                            <span style={{ fontSize: 16, fontWeight: 600 }}>{selectedAlert.title}</span>
                        </div>

                        <div style={{ background: '#f5f5f5', padding: 12, borderRadius: 8 }}>
                            <div style={{ fontSize: 13, color: '#666', marginBottom: 4 }}>告警内容</div>
                            <div style={{ fontSize: 15 }}>{selectedAlert.message}</div>
                        </div>

                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                            <div>
                                <div style={{ fontSize: 13, color: '#666' }}>关联运单</div>
                                <div>{selectedAlert.shipment_id || '-'}</div>
                            </div>
                            <div>
                                <div style={{ fontSize: 13, color: '#666' }}>关联设备</div>
                                <div>{selectedAlert.device?.external_device_id || selectedAlert.device_id || '-'}</div>
                            </div>
                            <div>
                                <div style={{ fontSize: 13, color: '#666' }}>发生时间</div>
                                <div>{new Date(selectedAlert.created_at).toLocaleString('zh-CN')}</div>
                            </div>
                            <div>
                                <div style={{ fontSize: 13, color: '#666' }}>告警类型</div>
                                <div>{TYPE_LABELS[selectedAlert.type] || selectedAlert.type}</div>
                            </div>
                        </div>
                    </div>
                )}
            </Modal>

            <Modal
                title="全局搜索"
                open={searchModalVisible}
                onCancel={() => {
                    setSearchModalVisible(false);
                    setSearchKeyword('');
                    setSearchSuggestions([]);
                }}
                footer={null}
                width={500}
            >
                <div style={{ padding: '12px 0' }}>
                    <AutoComplete
                        style={{ width: '100%' }}
                        options={searchSuggestions}
                        onSearch={fetchSearchSuggestions}
                        onSelect={handleSelectSuggestion}
                        value={searchKeyword}
                        onChange={setSearchKeyword}
                    >
                        <Input.Search
                            placeholder="输入运单号、设备ID或关键词..."
                            size="large"
                            loading={searchLoading}
                            onSearch={handleSearch}
                            enterButton="搜索"
                        />
                    </AutoComplete>
                    <div style={{ marginTop: 16, color: '#9ca3af', fontSize: 13 }}>
                        <div>提示：</div>
                        <ul style={{ marginTop: 8, paddingLeft: 20 }}>
                            <li>输入运单号或包含"运单"关键词将跳转到运单管理</li>
                            <li>输入小黑卡设备号(以8681开头)将跳转到设备管理</li>
                        </ul>
                    </div>
                </div>
            </Modal>
        </div>
    );
};

export default Dashboard;
