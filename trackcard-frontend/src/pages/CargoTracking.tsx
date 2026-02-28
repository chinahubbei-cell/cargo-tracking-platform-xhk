import React, { useEffect, useState, useMemo, lazy, Suspense } from 'react';
import { Input, Spin, Tag, Card, Space, Select } from 'antd';
import {
    SearchOutlined,
    EnvironmentOutlined,
    ClockCircleOutlined,
    RightOutlined,
} from '@ant-design/icons';
import api from '../api/client';
import { useLocation, useNavigate } from 'react-router-dom';
import type { Shipment, OrganizationTreeNode } from '../types';
import './CargoTracking.css';

// 懒加载地图组件
const ShipmentMap = lazy(() => import('../components/ShipmentMap'));
import TransportNodeTimeline from '../components/TransportNodeTimeline';
import DeviceStopRecords from '../components/DeviceStopRecords';


const CargoTracking: React.FC = () => {
    const location = useLocation();
    const navigate = useNavigate();
    const initialShipmentId = (location.state as any)?.shipmentId as string | undefined;
    const [loading, setLoading] = useState(true);
    const [shipments, setShipments] = useState<Shipment[]>([]);
    // 如果从运单管理跳转过来，将运单号设置为初始搜索值
    const [search, setSearch] = useState(initialShipmentId || '');
    const [selectedShipment, setSelectedShipment] = useState<Shipment | null>(null);
    // 组织筛选相关
    const [filterOrgId, setFilterOrgId] = useState<string>('');
    const [organizations, setOrganizations] = useState<OrganizationTreeNode[]>([]);

    // 展平组织树结构，用于筛选下拉框
    const flattenOrgs = (orgs: OrganizationTreeNode[], level = 0): { id: string; name: string; level: number }[] => {
        const result: { id: string; name: string; level: number }[] = [];
        for (const org of orgs) {
            result.push({ id: org.id, name: org.name, level });
            if (org.children && org.children.length > 0) {
                result.push(...flattenOrgs(org.children, level + 1));
            }
        }
        return result;
    };
    const flatOrganizations = flattenOrgs(organizations);

    // 从运单管理跳转过来时，清除state防止刷新后重复设置搜索框
    useEffect(() => {
        if (initialShipmentId) {
            navigate(location.pathname, { replace: true, state: null });
        }
    }, []);

    useEffect(() => {
        loadData();
    }, [search, filterOrgId]); // search或组织变化时重新加载数据

    const loadData = async () => {
        setLoading(true);
        try {
            // 调用后端搜索，传递组织筛选参数
            const [shipmentsRes, orgsRes] = await Promise.all([
                api.getShipments({ search, org_id: filterOrgId }),
                organizations.length === 0 ? api.getOrganizations({ tree: true }) : Promise.resolve(null),
            ]);
            const data = shipmentsRes.data || [];
            if (orgsRes) setOrganizations(orgsRes);

            setShipments(data);
            if (data.length > 0) {
                // 从运单管理跳转过来时，自动选中搜索到的运单显示详情面板
                if (initialShipmentId && search === initialShipmentId) {
                    const targetShipment = data.find(s => s.id === initialShipmentId);
                    if (targetShipment) {
                        setSelectedShipment(targetShipment);
                        return; // 已选中，直接返回
                    }
                }
                if (selectedShipment && data.find(s => s.id === selectedShipment.id)) {
                    // 保持当前选中（用户手动选择的）
                    // 不做任何操作
                } else {
                    // 未指定运单且没有当前选中，默认不选中任何运单，显示全部
                    setSelectedShipment(null);
                }
            } else {
                setSelectedShipment(null);
            }
        } catch {
            // 加载失败
            setShipments([]);
        } finally {
            setLoading(false);
        }
    };

    // 计算ETA倒计时
    const getETACountdown = (eta: string | undefined) => {
        if (!eta) return '-';
        const diff = new Date(eta).getTime() - Date.now();
        if (diff <= 0) return '已到达';
        const days = Math.floor(diff / (24 * 3600000));
        const hours = Math.floor((diff % (24 * 3600000)) / 3600000);
        return `${days}天 ${hours}小时`;
    };

    // 获取当前里程碑索引


    const statusColors: Record<string, string> = {
        pending: '#faad14',
        in_transit: '#1890ff',
        delivered: '#52c41a',
        cancelled: '#ff4d4f',
    };

    const statusLabels: Record<string, string> = {
        pending: '待发货',
        in_transit: '运输中',
        delivered: '已到达',
        cancelled: '已取消',
    };

    // 过滤运单并将选中运单置顶
    const filteredShipments = useMemo(() => {
        let result = shipments;
        if (search) {
            result = result.filter(s =>
                s.id.toLowerCase().includes(search.toLowerCase()) ||
                s.origin.toLowerCase().includes(search.toLowerCase()) ||
                s.destination.toLowerCase().includes(search.toLowerCase())
            );
        }
        // 选中运单置顶显示
        if (selectedShipment) {
            const selectedIndex = result.findIndex(s => s.id === selectedShipment.id);
            if (selectedIndex > 0) {
                const selected = result[selectedIndex];
                result = [selected, ...result.slice(0, selectedIndex), ...result.slice(selectedIndex + 1)];
            }
        }
        return result;
    }, [shipments, search, selectedShipment]);

    if (loading) {
        return (
            <div className="shipments-loading">
                <Spin size="large" />
                <p>加载运单数据...</p>
            </div>
        );
    }

    return (
        <Card
            title="货物追踪"
            headStyle={{ fontSize: 16, fontWeight: 600 }}
            extra={
                <Space>
                    <Select
                        value={filterOrgId || 'all'}
                        onChange={(v) => setFilterOrgId(v === 'all' ? '' : v)}
                        style={{ width: 180 }}
                        placeholder="组织机构"
                    >
                        <Select.Option value="all">全部组织</Select.Option>
                        {flatOrganizations.map(org => (
                            <Select.Option key={org.id} value={org.id}>
                                {org.level > 0 ? '└ '.repeat(org.level) : ''}{org.name}
                            </Select.Option>
                        ))}
                    </Select>
                    <Input
                        placeholder="搜索运单号..."
                        prefix={<SearchOutlined />}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        style={{ width: 200 }}
                        allowClear
                    />
                </Space>
            }
        >
            <div className="shipments-container">
                {/* 左侧地图区域 */}
                <div className="shipments-map-section">
                    <Suspense fallback={<div className="map-loading"><Spin size="large" /></div>}>
                        <ShipmentMap
                            shipments={filteredShipments}
                            selectedShipment={selectedShipment}
                            onSelectShipment={setSelectedShipment}
                        />
                    </Suspense>

                    {/* 底部面板 - 选中运单时显示详情，未选中时显示占位 */}
                    {selectedShipment ? (
                        <div className="shipment-detail-panel">
                            {/* 第一行：运单信息、设备指标、路线、总耗时 */}
                            <div className="detail-header">
                                <div className="detail-header-left capsule-info">
                                    <div className="detail-id">
                                        <span className="id-icon">📦</span>
                                        <span
                                            className="id-text clickable"
                                            onClick={() => navigate('/business/shipments', { state: { shipmentId: selectedShipment.id } })}
                                            title="点击查看运单详情"
                                        >
                                            {selectedShipment.id}
                                        </span>
                                        <Tag color={statusColors[selectedShipment.status] || '#999'}>
                                            {statusLabels[selectedShipment.status] || selectedShipment.status}
                                        </Tag>
                                    </div>

                                    <div className="detail-header-metrics">
                                        <span className="header-metric" title="温度">
                                            🌡️ {selectedShipment.device?.temperature != null ? `${selectedShipment.device.temperature}°C` : '--'}
                                        </span>
                                        <span className="header-metric" title="湿度">
                                            💧 {selectedShipment.device?.humidity != null ? `${selectedShipment.device.humidity}%` : '--'}
                                        </span>
                                        <span className="header-metric" title="电量">
                                            🔋 {selectedShipment.device?.battery ?? '--'}%
                                        </span>
                                        <span className="header-metric header-metric-eta" title="ETA">
                                            <ClockCircleOutlined /> {getETACountdown(selectedShipment.eta)}
                                        </span>
                                    </div>
                                </div>

                                <div className="detail-header-center">
                                    <div className="route-inline-compact">
                                        <div className="route-point origin">
                                            <EnvironmentOutlined />
                                            <span>{selectedShipment.origin}</span>
                                        </div>
                                        <div className="route-arrow">
                                            <RightOutlined />
                                        </div>
                                        <div className="route-point dest">
                                            <EnvironmentOutlined />
                                            <span>{selectedShipment.destination}</span>
                                        </div>
                                    </div>
                                </div>

                                <div className="detail-header-right">
                                    <span className="total-duration-text">
                                        {(selectedShipment as any).total_duration || '--'}
                                    </span>
                                </div>
                            </div>

                            {/* 第二行：运输环节 */}
                            <div className="detail-timeline-section">
                                <div className="timeline-wrapper">
                                    <TransportNodeTimeline
                                        shipmentId={selectedShipment.id}
                                        compact={true}
                                    />
                                </div>
                            </div>

                            {/* 第三行：设备停留记录 */}
                            <div className="detail-stop-records">
                                <DeviceStopRecords
                                    shipmentId={selectedShipment.id}
                                    refreshInterval={30000}
                                    maxRecords={100}
                                />
                            </div>
                        </div>
                    ) : (
                        <div className="shipment-detail-panel empty-panel">
                            <div className="empty-panel-content">
                                <EnvironmentOutlined style={{ fontSize: 24, opacity: 0.3 }} />
                                <span style={{ marginLeft: 8, color: 'var(--text-muted)', fontSize: 14 }}>
                                    点击右侧运单卡片查看详情
                                </span>
                            </div>
                        </div>
                    )}
                </div>

                {/* 右侧运单列表 */}
                <div className="shipments-list-section">
                    {/* 头部 */}
                    <div className="list-header">
                        <h2>运单列表</h2>
                    </div>

                    {/* 运单卡片列表 */}
                    <div className="shipment-cards">
                        {filteredShipments.map((shipment) => (
                            <div
                                key={shipment.id}
                                className={`shipment-card ${selectedShipment?.id === shipment.id ? 'selected' : ''}`}
                                onClick={() => setSelectedShipment(shipment)}
                            >
                                {/* 卡片头部 */}
                                <div className="card-header">
                                    <span className="shipment-id">运单号：{shipment.id}</span>
                                </div>

                                {/* 路线 */}
                                <div className="card-route">
                                    <EnvironmentOutlined className="route-icon origin" />
                                    <span className="route-text">{shipment.origin}</span>
                                    <RightOutlined className="route-arrow" />
                                    <EnvironmentOutlined className="route-icon dest" />
                                    <span className="route-text">{shipment.destination}</span>
                                </div>

                                {/* 进度条 */}
                                <div className="card-progress">
                                    <div className="progress-bar">
                                        <div
                                            className="progress-fill"
                                            style={{
                                                width: `${shipment.progress || 0}%`,
                                                backgroundColor: statusColors[shipment.status]
                                            }}
                                        />
                                    </div>
                                    <span className="progress-text">进度：{shipment.progress || 0}%</span>
                                </div>

                                {/* 底部信息 */}
                                <div className="card-footer">
                                    <div className="eta-info">
                                        <ClockCircleOutlined />
                                        <span>预计到达：{getETACountdown(shipment.eta)}</span>
                                    </div>
                                    <Tag color={statusColors[shipment.status]}>
                                        {statusLabels[shipment.status] || shipment.status}
                                    </Tag>
                                </div>
                            </div>
                        ))}

                        {filteredShipments.length === 0 && (
                            <div className="no-data">
                                <EnvironmentOutlined style={{ fontSize: 48, opacity: 0.3 }} />
                                <p>暂无运单数据</p>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </Card>
    );
};

export default CargoTracking;
