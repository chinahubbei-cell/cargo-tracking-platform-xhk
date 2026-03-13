import React, { useEffect, useState } from 'react';
import {
    Typography, Tabs, Descriptions, Tag, Button, Card, Row, Col,
    Progress, Table, Space, message, Popconfirm, Form, Input, Select, Spin
} from 'antd';
import {
    ArrowLeftOutlined, EditOutlined, SaveOutlined, CloseOutlined,
    DollarOutlined, PauseCircleOutlined, PlayCircleOutlined
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { orgApi } from '../services/api';
import RenewModal from '../components/RenewModal';
import dayjs from 'dayjs';

const { Title, Text } = Typography;

const statusColors: Record<string, string> = {
    trial: 'blue', active: 'green', suspended: 'default', expired: 'red',
};
const statusLabels: Record<string, string> = {
    trial: '试用', active: '正常', suspended: '暂停', expired: '已过期',
};

const CustomerDetail: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [org, setOrg] = useState<any>(null);
    const [stats, setStats] = useState<any>(null);
    const [renewals, setRenewals] = useState<any[]>([]);
    const [devices, setDevices] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);
    const [editing, setEditing] = useState(false);
    const [form] = Form.useForm();
    const [renewVisible, setRenewVisible] = useState(false);

    useEffect(() => {
        if (id) loadAll();
    }, [id]);

    const loadAll = async () => {
        setLoading(true);
        try {
            const [orgRes, statsRes, renewalsRes, devicesRes] = await Promise.all([
                orgApi.get(id!),
                orgApi.getStats(id!),
                orgApi.getRenewals(id!),
                orgApi.getDevices(id!),
            ]);
            if (orgRes.data.success) setOrg(orgRes.data.data);
            if (statsRes.data.success) setStats(statsRes.data.data);
            if (renewalsRes.data.success) setRenewals(renewalsRes.data.data || []);
            if (devicesRes.data.success) setDevices(devicesRes.data.data || []);
        } finally {
            setLoading(false);
        }
    };

    const startEdit = () => {
        form.setFieldsValue({
            name: org.name,
            company_name: org.company_name,
            credit_code: org.credit_code,
            short_name: org.short_name,
            contact_name: org.contact_name,
            contact_phone: org.contact_phone,
            contact_email: org.contact_email,
            address: org.address,
            remark: org.remark,
        });
        setEditing(true);
    };

    const saveEdit = async () => {
        const values = await form.validateFields();
        try {
            await orgApi.update(id!, values);
            message.success('保存成功');
            setEditing(false);
            loadAll();
        } catch {
            message.error('保存失败');
        }
    };

    const toggleService = async (newStatus: string) => {
        try {
            await orgApi.setService(id!, { service_status: newStatus });
            message.success(newStatus === 'suspended' ? '已暂停服务' : '已恢复服务');
            loadAll();
        } catch {
            message.error('操作失败');
        }
    };

    if (loading) return <Spin style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} size="large" />;
    if (!org) return <div>客户不存在</div>;

    const isMainAccount = org.level <= 1;
    const daysLeft = org.service_end ? dayjs(org.service_end).diff(dayjs(), 'day') : null;

    // Tab 1: 基本信息
    const renderInfo = () => (
        <Card
            title="基本信息"
            extra={
                editing ? (
                    <Space>
                        <Button icon={<SaveOutlined />} type="primary" onClick={saveEdit}>保存</Button>
                        <Button icon={<CloseOutlined />} onClick={() => setEditing(false)}>取消</Button>
                    </Space>
                ) : (
                    <Button icon={<EditOutlined />} onClick={startEdit}>编辑</Button>
                )
            }
        >
            {editing ? (
                <Form form={form} layout="vertical">
                    <Row gutter={16}>
                        <Col span={8}><Form.Item name="name" label="客户名称" rules={[{ required: true }]}><Input /></Form.Item></Col>
                        <Col span={8}><Form.Item name="company_name" label="公司名称"><Input /></Form.Item></Col>
                        <Col span={8}><Form.Item name="credit_code" label="社会信用代码"><Input maxLength={18} /></Form.Item></Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={8}><Form.Item name="short_name" label="简称"><Input /></Form.Item></Col>
                        <Col span={8}><Form.Item name="contact_name" label="联系人"><Input /></Form.Item></Col>
                        <Col span={8}><Form.Item name="contact_phone" label="联系电话"><Input /></Form.Item></Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={8}><Form.Item name="contact_email" label="邮箱"><Input /></Form.Item></Col>
                        <Col span={16}><Form.Item name="address" label="地址"><Input /></Form.Item></Col>
                    </Row>
                    <Form.Item name="remark" label="备注"><Input.TextArea rows={2} /></Form.Item>
                </Form>
            ) : (
                <Descriptions bordered column={3}>
                    <Descriptions.Item label="客户名称">{org.name}</Descriptions.Item>
                    <Descriptions.Item label="公司名称">{org.company_name || '-'}</Descriptions.Item>
                    <Descriptions.Item label="社会信用代码">{org.credit_code || '-'}</Descriptions.Item>
                    <Descriptions.Item label="简称">{org.short_name || '-'}</Descriptions.Item>
                    <Descriptions.Item label="联系人">{org.contact_name || '-'}</Descriptions.Item>
                    <Descriptions.Item label="联系电话">{org.contact_phone || '-'}</Descriptions.Item>
                    <Descriptions.Item label="邮箱">{org.contact_email || '-'}</Descriptions.Item>
                    <Descriptions.Item label="地址" span={2}>{org.address || '-'}</Descriptions.Item>
                    <Descriptions.Item label="账号类型">
                        {isMainAccount ? <Tag color="blue">主账号</Tag> : <Tag color="cyan">子账号</Tag>}
                    </Descriptions.Item>
                    <Descriptions.Item label="备注" span={2}>{org.remark || '-'}</Descriptions.Item>
                </Descriptions>
            )}
        </Card>
    );

    // Tab 2: 服务与配额
    const renderQuotaCard = (title: string, used: number, max: number) => {
        const percent = max > 0 ? Math.min(Math.round((used / max) * 100), 100) : 0;
        return (
            <Card size="small" style={{ textAlign: 'center' }}>
                <Text type="secondary" style={{ fontSize: 13 }}>{title}</Text>
                <div style={{ margin: '12px 0' }}>
                    <Progress
                        type="dashboard"
                        percent={percent}
                        size={100}
                        strokeColor={percent > 90 ? '#ff4d4f' : percent > 70 ? '#faad14' : '#52c41a'}
                        format={() => `${used}/${max}`}
                    />
                </div>
            </Card>
        );
    };

    const renderService = () => {
        if (!isMainAccount) {
            return (
                <Card>
                    <div style={{ textAlign: 'center', padding: 40 }}>
                        <Tag color="cyan" style={{ fontSize: 16, padding: '4px 16px' }}>子账号</Tag>
                        <div style={{ marginTop: 16 }}>
                            <Text type="secondary">子账号的服务状态、服务期限、配额均继承自主账号，无需单独配置。</Text>
                        </div>
                    </div>
                </Card>
            );
        }
        return (
            <>
                <Card title="服务信息" extra={
                    <Space>
                        <Button type="primary" icon={<DollarOutlined />} onClick={() => setRenewVisible(true)}>续费</Button>
                        {org.service_status === 'active' && (
                            <Popconfirm title="确定暂停服务？" onConfirm={() => toggleService('suspended')}>
                                <Button icon={<PauseCircleOutlined />}>暂停服务</Button>
                            </Popconfirm>
                        )}
                        {org.service_status === 'suspended' && (
                            <Popconfirm title="确定恢复服务？" onConfirm={() => toggleService('active')}>
                                <Button type="primary" icon={<PlayCircleOutlined />}>恢复服务</Button>
                            </Popconfirm>
                        )}
                    </Space>
                }>
                    <Descriptions bordered column={2}>
                        <Descriptions.Item label="服务状态">
                            <Tag color={statusColors[org.service_status]} style={{ fontSize: 14, padding: '2px 12px' }}>
                                {statusLabels[org.service_status] || org.service_status}
                            </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="到期倒计时">
                            {daysLeft !== null ? (
                                <Text type={daysLeft <= 30 ? 'danger' : undefined} strong>
                                    {daysLeft > 0 ? `还有 ${daysLeft} 天到期` : daysLeft === 0 ? '今日到期' : `已过期 ${Math.abs(daysLeft)} 天`}
                                </Text>
                            ) : '未设置'}
                        </Descriptions.Item>
                        <Descriptions.Item label="服务期限" span={2}>
                            {org.service_start ? dayjs(org.service_start).format('YYYY-MM-DD') : '未设置'}
                            {' ~ '}
                            {org.service_end ? dayjs(org.service_end).format('YYYY-MM-DD') : '未设置'}
                        </Descriptions.Item>
                    </Descriptions>
                </Card>

                <Row gutter={16} style={{ marginTop: 16 }}>
                    <Col span={8}>
                        {renderQuotaCard('设备数', stats?.device_count || 0, stats?.max_devices || org.max_devices)}
                    </Col>
                    <Col span={8}>
                        {renderQuotaCard('用户数', stats?.user_count || 0, stats?.max_users || org.max_users)}
                    </Col>
                    <Col span={8}>
                        {renderQuotaCard('本月运单', stats?.shipment_count || 0, stats?.max_shipments || org.max_shipments)}
                    </Col>
                </Row>
            </>
        );
    };

    // Tab 3: 续费记录
    const renewalColumns = [
        {
            title: '续费类型', dataIndex: 'renewal_type', width: 100,
            render: (t: string) => <Tag>{t === 'manual' ? '手动续费' : t === 'auto' ? '自动续费' : t}</Tag>
        },
        { title: '续费月数', dataIndex: 'period_months', width: 100 },
        { title: '续费金额', dataIndex: 'amount', width: 100, render: (t: number) => t ? `¥${t.toFixed(2)}` : '-' },
        { title: '变更前到期', dataIndex: 'old_end_date', width: 120, render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD') : '-' },
        { title: '变更后到期', dataIndex: 'new_end_date', width: 120, render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD') : '-' },
        { title: '操作人', dataIndex: 'created_by_name', width: 100, render: (t: string) => t || '-' },
        { title: '时间', dataIndex: 'created_at', width: 160, render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-' },
    ];

    const renderRenewals = () => (
        <Table
            dataSource={renewals}
            columns={renewalColumns}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            locale={{ emptyText: '暂无续费记录' }}
        />
    );

    // Tab 4: 关联设备
    const deviceColumns = [
        { title: 'IMEI', dataIndex: 'imei', width: 160 },
        { title: '设备型号', dataIndex: 'model', width: 100 },
        {
            title: '状态', dataIndex: 'status', width: 100,
            render: (s: string) => {
                const map: Record<string, { color: string; label: string }> = {
                    in_stock: { color: 'default', label: '库存' },
                    allocated: { color: 'blue', label: '已分配' },
                    activated: { color: 'green', label: '已激活' },
                };
                const item = map[s] || { color: 'default', label: s };
                return <Tag color={item.color}>{item.label}</Tag>;
            }
        },
        { title: '分配时间', dataIndex: 'allocated_at', width: 160, render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-' },
    ];

    const renderDevices = () => (
        <Table
            dataSource={devices}
            columns={deviceColumns}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            locale={{ emptyText: '暂无关联设备' }}
        />
    );

    const tabItems = [
        { key: 'info', label: '基本信息', children: renderInfo() },
        { key: 'service', label: '服务与配额', children: renderService() },
        ...(isMainAccount ? [
            { key: 'renewals', label: `续费记录 (${renewals.length})`, children: renderRenewals() },
        ] : []),
        { key: 'devices', label: `关联设备 (${devices.length})`, children: renderDevices() },
    ];

    return (
        <div>
            <Space style={{ marginBottom: 16 }}>
                <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/orgs')}>返回列表</Button>
                <Title level={4} style={{ margin: 0 }}>{org.name}</Title>
                {isMainAccount ? <Tag color="blue">主账号</Tag> : <Tag color="cyan">子账号</Tag>}
                {isMainAccount && (
                    <Tag color={statusColors[org.service_status]}>
                        {statusLabels[org.service_status] || org.service_status}
                    </Tag>
                )}
            </Space>

            <Tabs items={tabItems} />

            <RenewModal
                visible={renewVisible}
                org={org}
                onClose={() => setRenewVisible(false)}
                onSuccess={() => { setRenewVisible(false); loadAll(); }}
            />
        </div>
    );
};

export default CustomerDetail;
