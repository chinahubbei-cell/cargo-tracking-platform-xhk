import React, { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import { Table, Card, Tag, Button, Space, Select, message, Dropdown, Modal } from 'antd';
import type { MenuProps } from 'antd';
import { CheckOutlined, ReloadOutlined, SettingOutlined, EyeOutlined, DeleteOutlined, DownloadOutlined, CameraOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import * as XLSX from 'xlsx';
import html2canvas from 'html2canvas';
import { downloadExcel, downloadPng } from '../utils/downloadUtils';
import api from '../api/client';
import type { Alert } from '../types';
import { useAuthStore } from '../store/authStore';
import { useNavigate } from 'react-router-dom';
import './Alerts.css';

// 常量提取到组件外部，避免每次渲染重建
const SEVERITY_COLORS: Record<string, string> = {
    critical: 'red',
    warning: 'orange',
    info: 'blue',
};

const SEVERITY_LABELS: Record<string, string> = {
    critical: '严重',
    warning: '警告',
    info: '提示',
};

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

// 安全的日期格式化函数
const formatDateTime = (dateStr: string | undefined): string => {
    if (!dateStr) return '-';
    try {
        const date = new Date(dateStr);
        if (isNaN(date.getTime())) return '-';
        return date.toLocaleString('zh-CN');
    } catch {
        return '-';
    }
};

// 预警详情组件 - 拆分提高可读性
interface AlertDetailProps {
    alert: Alert;
}

const AlertDetail: React.FC<AlertDetailProps> = ({ alert }) => (
    <div className="alert-detail">
        <div className="alert-detail-grid">
            <div className="alert-detail-item">
                <div className="alert-detail-label">预警ID</div>
                <div className="alert-detail-value">{alert.id}</div>
            </div>
            <div className="alert-detail-item">
                <div className="alert-detail-label">状态</div>
                <Tag color={alert.status === 'pending' ? 'orange' : 'green'}>
                    {alert.status === 'pending' ? '待处理' : '已处理'}
                </Tag>
            </div>
        </div>
        <div className="alert-detail-section">
            <div className="alert-detail-label">预警标题</div>
            <div className="alert-detail-title">{alert.title}</div>
        </div>
        <div className="alert-detail-section">
            <div className="alert-detail-label">预警内容</div>
            <div className="alert-detail-message">
                {alert.message || '无详细信息'}
            </div>
        </div>
        <div className="alert-detail-grid">
            <div className="alert-detail-item">
                <div className="alert-detail-label">预警类型</div>
                <div>{TYPE_LABELS[alert.type] || alert.type}</div>
            </div>
            <div className="alert-detail-item">
                <div className="alert-detail-label">严重程度</div>
                <Tag color={SEVERITY_COLORS[alert.severity]}>
                    {SEVERITY_LABELS[alert.severity] || alert.severity}
                </Tag>
            </div>
            <div className="alert-detail-item">
                <div className="alert-detail-label">关联运单</div>
                <div>{alert.shipment_id || '-'}</div>
            </div>
            <div className="alert-detail-item">
                <div className="alert-detail-label">关联设备</div>
                <div>{alert.device?.external_device_id || alert.device_id || '-'}</div>
            </div>
            <div className="alert-detail-item">
                <div className="alert-detail-label">创建时间</div>
                <div>{formatDateTime(alert.created_at)}</div>
            </div>
            {alert.resolved_at && (
                <div className="alert-detail-item">
                    <div className="alert-detail-label">处理时间</div>
                    <div>{formatDateTime(alert.resolved_at)}</div>
                </div>
            )}
        </div>
    </div>
);

const Alerts: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [alerts, setAlerts] = useState<Alert[]>([]);
    const [statusFilter, setStatusFilter] = useState<string>('');
    const [severityFilter, setSeverityFilter] = useState<string>('');
    const [typeFilter, setTypeFilter] = useState<string>('');
    const [detailModalVisible, setDetailModalVisible] = useState(false);
    const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);
    const [currentPage, setCurrentPage] = useState(1);
    const [pageSize, setPageSize] = useState(50);
    const tableRef = useRef<HTMLDivElement>(null);

    const { user } = useAuthStore();
    const navigate = useNavigate();

    const canEdit = user?.role === 'admin' || user?.role === 'operator';

    // 使用useCallback优化loadAlerts，避免不必要的重建
    const loadAlerts = useCallback(async () => {
        setLoading(true);
        try {
            const res = await api.getAlerts({
                status: statusFilter || undefined,
                severity: severityFilter || undefined,
            });
            if (res.data) {
                setAlerts(res.data);
            }
        } catch (error) {
            console.error('加载预警列表失败:', error);
            message.error('加载预警列表失败');
        } finally {
            setLoading(false);
        }
    }, [statusFilter, severityFilter]);

    useEffect(() => {
        loadAlerts();
    }, [loadAlerts]);

    // 筛选后的数据
    const filteredAlerts = useMemo(() => {
        return alerts.filter(alert => {
            if (typeFilter && alert.type !== typeFilter) return false;
            return true;
        });
    }, [alerts, typeFilter]);

    // Excel导出功能
    const handleExportExcel = useCallback(() => {
        if (filteredAlerts.length === 0) {
            message.warning('没有可导出的数据');
            return;
        }

        const exportData = filteredAlerts.map((alert, index) => ({
            '序号': index + 1,
            '运单号': alert.shipment_id || '-',
            '预警标题': alert.title,
            '预警类型': TYPE_LABELS[alert.type] || alert.type,
            '严重程度': SEVERITY_LABELS[alert.severity] || alert.severity,
            '状态': alert.status === 'resolved' ? '已处理' : '待处理',
            '关联设备': alert.device?.external_device_id || alert.device_id || '-',
            '预警内容': alert.message || '-',
            '创建时间': formatDateTime(alert.created_at),
            '处理时间': formatDateTime(alert.resolved_at),
        }));

        const ws = XLSX.utils.json_to_sheet(exportData);
        const wb = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(wb, ws, '预警数据');

        // 设置列宽
        ws['!cols'] = [
            { wch: 6 },   // 序号
            { wch: 20 },  // 运单号
            { wch: 30 },  // 预警标题
            { wch: 15 },  // 预警类型
            { wch: 10 },  // 严重程度
            { wch: 10 },  // 状态
            { wch: 20 },  // 关联设备
            { wch: 40 },  // 预警内容
            { wch: 20 },  // 创建时间
            { wch: 20 },  // 处理时间
        ];

        // 生成Excel文件并下载
        const fileName = `alerts_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}`;
        const excelBuffer = XLSX.write(wb, { bookType: 'xlsx', type: 'array' });
        downloadExcel(excelBuffer, fileName);

        message.success(`成功导出 ${filteredAlerts.length} 条预警数据`);
    }, [filteredAlerts]);

    // 截屏功能
    const handleScreenshot = useCallback(async () => {
        if (!tableRef.current) {
            message.error('无法获取表格内容');
            return;
        }

        try {
            message.loading({ content: '正在生成截图...', key: 'screenshot' });



            const canvas = await html2canvas(tableRef.current, {
                backgroundColor: '#ffffff',
                scale: 2,
                useCORS: true,
                logging: false,
            });

            canvas.toBlob((blob) => {
                if (blob) {
                    const fileName = `alerts_screenshot_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}`;
                    downloadPng(blob, fileName);
                    message.success({ content: '截图已保存', key: 'screenshot' });
                }
            }, 'image/png');
        } catch (error) {
            console.error('截图失败:', error);
            message.error({ content: '截图失败', key: 'screenshot' });
        }
    }, []);

    const handleResolve = useCallback(async (id: string) => {
        try {
            await api.resolveAlert(id);
            message.success('预警已处理');
            loadAlerts();
        } catch (error) {
            console.error('处理预警失败:', error);
            message.error('处理失败');
        }
    }, [loadAlerts]);

    const handleViewDetail = useCallback((record: Alert) => {
        setSelectedAlert(record);
        setDetailModalVisible(true);
    }, []);

    const handleCloseDetail = useCallback(() => {
        setDetailModalVisible(false);
        setSelectedAlert(null);
    }, []);

    const handleResolveFromDetail = useCallback(() => {
        if (selectedAlert) {
            handleResolve(selectedAlert.id);
            setDetailModalVisible(false);
            setSelectedAlert(null);
        }
    }, [selectedAlert, handleResolve]);

    // 获取所有预警类型用于筛选
    const alertTypes = useMemo(() => {
        const types = new Set(alerts.map(a => a.type));
        return Array.from(types).map(type => ({
            value: type,
            label: TYPE_LABELS[type] || type
        }));
    }, [alerts]);

    // 使用useMemo优化columns定义
    const columns: ColumnsType<Alert> = useMemo(() => [
        {
            title: '序号',
            key: 'index',
            width: 60,
            fixed: 'left',
            align: 'center' as const,
            // 修复：根据分页计算正确的序号
            render: (_, __, index) => (currentPage - 1) * pageSize + index + 1,
        },
        {
            title: '操作',
            key: 'action',
            width: 60,
            fixed: 'left',
            align: 'center' as const,
            render: (_, record) => {
                const menuItems: MenuProps['items'] = [
                    {
                        key: 'view',
                        icon: <EyeOutlined />,
                        label: '查看详情',
                        onClick: () => handleViewDetail(record),
                    },
                    ...(canEdit && record.status === 'pending' ? [
                        {
                            key: 'resolve',
                            icon: <CheckOutlined />,
                            label: '标记已处理',
                            onClick: () => {
                                Modal.confirm({
                                    title: '确认处理',
                                    content: `确定标记预警 "${record.title}" 为已处理吗？`,
                                    okText: '确定',
                                    cancelText: '取消',
                                    onOk: () => handleResolve(record.id),
                                });
                            },
                        },
                    ] : []),
                    ...(canEdit ? [
                        { type: 'divider' as const },
                        {
                            key: 'delete',
                            icon: <DeleteOutlined />,
                            label: '删除预警',
                            danger: true,
                            disabled: true, // 修复：禁用未实现的功能
                            onClick: () => message.info('删除功能开发中'),
                        },
                    ] : []),
                ];

                return (
                    <Dropdown
                        menu={{ items: menuItems }}
                        trigger={['hover']}
                        placement="bottomRight"
                    >
                        <Button
                            type="text"
                            icon={<SettingOutlined className="action-icon" />}
                            className="action-button"
                        />
                    </Dropdown>
                );
            },
        },
        {
            title: '运单号',
            dataIndex: 'shipment_id',
            key: 'shipment_id',
            width: 140,
            render: (shipmentId: string | undefined) =>
                shipmentId ? (
                    <a
                        onClick={() => navigate('/business/shipments', { state: { shipmentId } })}
                        className="shipment-link"
                    >
                        {shipmentId}
                    </a>
                ) : '-',
        },
        {
            title: '预警标题',
            dataIndex: 'title',
            key: 'title',
            ellipsis: true, // 添加文本溢出处理
        },
        {
            title: '类型',
            dataIndex: 'type',
            key: 'type',
            width: 120,
            render: (type: string) => TYPE_LABELS[type] || type,
        },
        {
            title: '严重程度',
            dataIndex: 'severity',
            key: 'severity',
            width: 100,
            render: (severity: string) => (
                <Tag color={SEVERITY_COLORS[severity]}>{SEVERITY_LABELS[severity] || severity}</Tag>
            ),
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 80,
            render: (status: string) => (
                <Tag color={status === 'resolved' ? 'green' : 'orange'}>
                    {status === 'resolved' ? '已处理' : '待处理'}
                </Tag>
            ),
        },
        {
            title: '关联设备',
            dataIndex: 'device_id',
            key: 'device_id',
            width: 160,
            render: (_: string | undefined, record: Alert) =>
                record.device?.external_device_id || record.device_id || '-',
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            render: (time: string) => formatDateTime(time),
        },
    ], [canEdit, currentPage, pageSize, handleResolve, handleViewDetail, navigate]);

    // Modal footer使用useMemo优化
    const modalFooter = useMemo(() => {
        const buttons = [
            <Button key="close" onClick={handleCloseDetail}>关闭</Button>
        ];

        if (canEdit && selectedAlert?.status === 'pending') {
            buttons.push(
                <Button key="resolve" type="primary" onClick={handleResolveFromDetail}>
                    标记已处理
                </Button>
            );
        }

        return buttons;
    }, [canEdit, selectedAlert?.status, handleCloseDetail, handleResolveFromDetail]);

    return (
        <>
            <Card
                title="预警中心"
                extra={
                    <Space>
                        <Select
                            placeholder="状态"
                            allowClear
                            style={{ width: 120 }}
                            value={statusFilter || undefined}
                            onChange={setStatusFilter}
                        >
                            <Select.Option value="">全部状态</Select.Option>
                            <Select.Option value="pending">待处理</Select.Option>
                            <Select.Option value="resolved">已处理</Select.Option>
                        </Select>
                        <Select
                            placeholder="严重程度"
                            allowClear
                            style={{ width: 120 }}
                            value={severityFilter || undefined}
                            onChange={setSeverityFilter}
                        >
                            <Select.Option value="">全部等级</Select.Option>
                            <Select.Option value="critical">严重</Select.Option>
                            <Select.Option value="warning">警告</Select.Option>
                            <Select.Option value="info">提示</Select.Option>
                        </Select>
                        <Select
                            placeholder="预警类型"
                            allowClear
                            style={{ width: 150 }}
                            value={typeFilter || undefined}
                            onChange={setTypeFilter}
                            options={[
                                { value: '', label: '全部类型' },
                                ...alertTypes
                            ]}
                        />
                        <Button icon={<ReloadOutlined />} onClick={loadAlerts} loading={loading}>
                            刷新
                        </Button>
                    </Space>
                }
            >
                <div ref={tableRef}>
                    <Table
                        columns={columns}
                        dataSource={filteredAlerts}
                        rowKey="id"
                        loading={loading}
                        scroll={{ x: 1200, y: 'calc(100vh - 280px)' }}
                        size="small"
                        pagination={{
                            current: currentPage,
                            pageSize,
                            pageSizeOptions: ['10', '20', '50', '100'],
                            showSizeChanger: true,
                            showQuickJumper: true,
                            showTotal: (total, range) => (
                                <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                                    <Button
                                        type="default"
                                        size="small"
                                        icon={<DownloadOutlined />}
                                        onClick={handleExportExcel}
                                    >
                                        导出Excel
                                    </Button>
                                    <Button
                                        type="default"
                                        size="small"
                                        icon={<CameraOutlined />}
                                        onClick={handleScreenshot}
                                    >
                                        截屏
                                    </Button>
                                    <span>第 {range[0]}-{range[1]} 条，共 {total} 条</span>
                                </div>
                            ),
                            onChange: (page, size) => {
                                setCurrentPage(page);
                                if (size !== pageSize) setPageSize(size);
                            },
                        }}
                    />
                </div>
            </Card>

            {/* 预警详情弹窗 */}
            <Modal
                title="预警详情"
                open={detailModalVisible}
                onCancel={handleCloseDetail}
                footer={modalFooter}
                width={600}
                destroyOnClose
            >
                {selectedAlert && <AlertDetail alert={selectedAlert} />}
            </Modal>
        </>
    );
};

export default Alerts;
