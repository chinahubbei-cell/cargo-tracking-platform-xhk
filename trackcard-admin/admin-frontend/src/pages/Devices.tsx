import React, { useEffect, useState } from 'react';
import {
    Table, Button, Space, Tag, Modal, Form, Input, Select,
    message, Typography, Row, Popconfirm, Card, Statistic, Col
} from 'antd';
import { PlusOutlined, ReloadOutlined, ExportOutlined, ImportOutlined } from '@ant-design/icons';
import { deviceApi, orgApi } from '../services/api';
import dayjs from 'dayjs';

const { Title } = Typography;

const Devices: React.FC = () => {
    const [data, setData] = useState<any[]>([]);
    const [orgs, setOrgs] = useState<any[]>([]);
    const [stats, setStats] = useState<any>({});
    const [loading, setLoading] = useState(false);
    const [modalVisible, setModalVisible] = useState(false);
    const [allocateVisible, setAllocateVisible] = useState(false);
    const [currentDevice, setCurrentDevice] = useState<any>(null);
    const [form] = Form.useForm();
    const [allocateForm] = Form.useForm();

    useEffect(() => {
        fetchData();
        fetchOrgs();
        fetchStats();
    }, []);

    const fetchData = async () => {
        setLoading(true);
        try {
            const res = await deviceApi.list();
            if (res.data.success) {
                setData(res.data.data || []);
            }
        } finally {
            setLoading(false);
        }
    };

    const fetchOrgs = async () => {
        const res = await orgApi.list({ page_size: 1000 });
        if (res.data.success) {
            setOrgs(res.data.data || []);
        }
    };

    const fetchStats = async () => {
        const res = await deviceApi.stats();
        if (res.data.success) {
            setStats(res.data.data);
        }
    };

    const handleCreate = () => {
        form.resetFields();
        setModalVisible(true);
    };

    const handleSubmit = async () => {
        const values = await form.validateFields();
        try {
            await deviceApi.create(values);
            message.success('入库成功');
            setModalVisible(false);
            fetchData();
            fetchStats();
        } catch {
            message.error('操作失败');
        }
    };

    const handleAllocate = (record: any) => {
        setCurrentDevice(record);
        allocateForm.resetFields();
        setAllocateVisible(true);
    };

    const handleAllocateSubmit = async () => {
        const values = await allocateForm.validateFields();
        const org = orgs.find(o => o.id === values.org_id);
        try {
            await deviceApi.allocate(currentDevice.id, {
                ...values,
                org_name: org?.name || '',
            });
            message.success('分配成功');
            setAllocateVisible(false);
            fetchData();
            fetchStats();
        } catch {
            message.error('操作失败');
        }
    };

    const handleReturn = async (id: string) => {
        try {
            await deviceApi.return(id, { reason: '手动退回' });
            message.success('退回成功');
            fetchData();
            fetchStats();
        } catch {
            message.error('操作失败');
        }
    };

    const statusColors: Record<string, string> = {
        in_stock: 'green', allocated: 'blue', activated: 'purple', returned: 'orange', damaged: 'red',
    };
    const statusLabels: Record<string, string> = {
        in_stock: '库存中', allocated: '已分配', activated: '已激活', returned: '已退回', damaged: '已损坏',
    };

    const columns = [
        { title: '序号', key: 'index', width: 60, render: (_t: any, _r: any, index: number) => index + 1 },
        {
            title: '操作',
            key: 'action',
            width: 150,
            fixed: 'left' as const,
            render: (_: any, record: any) => (
                <Space size="small">
                    {record.status === 'in_stock' && (
                        <Button size="small" type="primary" icon={<ExportOutlined />} onClick={() => handleAllocate(record)}>
                            分配
                        </Button>
                    )}
                    {(record.status === 'allocated' || record.status === 'activated') && (
                        <Popconfirm title="确认退回？" onConfirm={() => handleReturn(record.id)}>
                            <Button size="small" icon={<ImportOutlined />}>退回</Button>
                        </Popconfirm>
                    )}
                </Space>
            ),
        },
        { title: '设备型号', dataIndex: 'device_model', key: 'device_model', width: 120 },
        { title: '设备号', dataIndex: 'imei', key: 'imei', width: 140, render: (t: string, r: any) => t || r.sn || r.id },
        {
            title: '设备状态',
            dataIndex: 'service_status',
            key: 'service_status',
            width: 90,
            render: (s: string) => {
                const map: Record<string, string> = { 'active': '在线', 'inactive': '离线', 'suspended': '停用' };
                const colorMap: Record<string, string> = { 'active': 'success', 'inactive': 'default', 'suspended': 'error' };
                return s ? <Tag color={colorMap[s] || 'default'}>{map[s] || s}</Tag> : '-';
            }
        },
        { title: '电量', dataIndex: 'battery', key: 'battery', width: 80, render: (b: number) => b !== undefined && b !== null ? `${b}%` : '-' },
        {
            title: '定位模式',
            dataIndex: 'locate_type',
            key: 'locate_type',
            width: 90,
            render: (t: number) => {
                const map: Record<number, string> = { 0: 'LBS', 1: 'GPS', 2: 'WIFI', 3: '北斗' };
                return t !== undefined && t !== null ? map[t] || t : '-';
            }
        },
        { title: '定位周期', key: 'locate_period', width: 90, render: () => '-' },
        {
            title: '最后一次定位时间',
            dataIndex: 'last_update',
            key: 'last_update',
            width: 150,
            render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-',
        },
        {
            title: '库存状态',
            dataIndex: 'status',
            key: 'status',
            width: 90,
            render: (s: string) => <Tag color={statusColors[s]}>{statusLabels[s] || s}</Tag>,
        },
        { title: '账号', key: 'account', width: 100, render: () => '-' },
        {
            title: '归属公司',
            dataIndex: 'org_name',
            key: 'org_name',
            width: 150,
            render: (text: string, record: any) => {
                const org = orgs.find((o: any) => o.id === record.org_id);
                const displayName = org?.name || text;
                if (!displayName) return '-';

                let levelTag: { color: string; label: string } | null = null;
                if (org?.level === 1) {
                    levelTag = { color: 'blue', label: '一级机构' };
                } else if (org?.level === 2) {
                    levelTag = { color: 'cyan', label: '二级机构' };
                }

                return (
                    <Space direction="vertical" size={0}>
                        <span>{displayName}</span>
                        {levelTag ? <Tag color={levelTag.color}>{levelTag.label}</Tag> : null}
                    </Space>
                );
            }
        },
        { title: 'ICCID', key: 'iccid', width: 150, render: () => '-' },
        { title: 'SIM卡号', key: 'sim', width: 120, render: () => '-' },
        { title: '物联卡状态', key: 'iot_card_status', width: 100, render: () => '-' },
        { title: '发卡日期', key: 'issue_date', width: 120, render: () => '-' },
        { title: '激活日期', key: 'activation_date', width: 120, render: () => '-' },
        { title: '最晚激活日期', key: 'latest_activation_date', width: 120, render: () => '-' },
        {
            title: '服务期止',
            dataIndex: 'service_end_at',
            key: 'service_end_at',
            width: 120,
            render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD') : '-',
        },
        { title: '套餐总量(MB)', key: 'total_data', width: 120, render: () => '-' },
        { title: '已使用流量(MB)', key: 'used_data', width: 120, render: () => '-' },
        { title: '剩余流量(MB)', key: 'remaining_data', width: 120, render: () => '-' },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 150,
            render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-',
        },
        { title: '出库时间', key: 'outbound_time', width: 120, render: () => '-' },
        { title: '到期时间', key: 'expiry_time', width: 120, render: () => '-' },
        { title: 'MCU_VER版本', key: 'mcu_ver', width: 120, render: () => '-' },
        { title: 'MTK_VER版本', key: 'mtk_ver', width: 120, render: () => '-' },
    ];

    return (
        <div>
            <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>设备管理</Title>
                <Space>
                    <Button icon={<ReloadOutlined />} onClick={() => { fetchData(); fetchStats(); }}>刷新</Button>
                    <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>入库设备</Button>
                </Space>
            </Row>

            {/* 统计卡片 */}
            <Row gutter={16} style={{ marginBottom: 16 }}>
                <Col span={4}><Card size="small"><Statistic title="总计" value={stats.total || 0} /></Card></Col>
                <Col span={4}><Card size="small"><Statistic title="库存中" value={stats.in_stock || 0} valueStyle={{ color: '#52c41a' }} /></Card></Col>
                <Col span={4}><Card size="small"><Statistic title="已分配" value={stats.allocated || 0} valueStyle={{ color: '#1890ff' }} /></Card></Col>
                <Col span={4}><Card size="small"><Statistic title="已激活" value={stats.activated || 0} valueStyle={{ color: '#722ed1' }} /></Card></Col>
                <Col span={4}><Card size="small"><Statistic title="已退回" value={stats.returned || 0} valueStyle={{ color: '#faad14' }} /></Card></Col>
                <Col span={4}><Card size="small"><Statistic title="已损坏" value={stats.damaged || 0} valueStyle={{ color: '#ff4d4f' }} /></Card></Col>
            </Row>

            <Table dataSource={data} columns={columns} rowKey="id" loading={loading} scroll={{ x: 'max-content' }} />

            {/* 入库 */}
            <Modal title="入库设备" open={modalVisible} onOk={handleSubmit} onCancel={() => setModalVisible(false)}>
                <Form form={form} layout="vertical">
                    <Form.Item name="imei" label="设备ID号" rules={[{ required: true }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name="device_model" label="设备名称" rules={[{ required: true }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name="device_type" label="设备类型" rules={[{ required: true }]}>
                        <Select>
                            <Select.Option value="tracker">追踪器</Select.Option>
                            <Select.Option value="sensor">传感器</Select.Option>
                            <Select.Option value="gateway">网关</Select.Option>
                            <Select.Option value="container">集装箱</Select.Option>
                        </Select>
                    </Form.Item>
                    <Form.Item name="provider" label="供应商">
                        <Input defaultValue="kuaihuoyun" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 分配 */}
            <Modal title="分配设备" open={allocateVisible} onOk={handleAllocateSubmit} onCancel={() => setAllocateVisible(false)}>
                <Form form={allocateForm} layout="vertical">
                    <Form.Item name="org_id" label="分配给组织" rules={[{ required: true }]}>
                        <Select showSearch optionFilterProp="children">
                            {orgs.map(org => <Select.Option key={org.id} value={org.id}>{org.name}</Select.Option>)}
                        </Select>
                    </Form.Item>
                    <Form.Item name="sub_account_name" label="子账号（可选）">
                        <Input placeholder="如：某分公司、某仓库" />
                    </Form.Item>
                    <Form.Item name="remark" label="备注">
                        <Input.TextArea rows={2} />
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    );
};

export default Devices;
