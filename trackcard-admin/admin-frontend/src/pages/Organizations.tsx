import React, { useEffect, useState } from 'react';
import {
    Table, Button, Space, Tag, Form, Input, Select,
    message, Typography, Row, Col, Popconfirm, Progress, Tabs
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined, DollarOutlined } from '@ant-design/icons';
import { orgApi } from '../services/api';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import CreateCustomerModal from '../components/CreateCustomerModal';
import RenewModal from '../components/RenewModal';

const { Title, Text } = Typography;

const statusColors: Record<string, string> = {
    trial: 'blue', active: 'green', suspended: 'default', expired: 'red',
};
const statusLabels: Record<string, string> = {
    trial: '试用', active: '正常', suspended: '暂停', expired: '已过期',
};

const Organizations: React.FC = () => {
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [createVisible, setCreateVisible] = useState(false);
    const [renewVisible, setRenewVisible] = useState(false);
    const [renewOrg, setRenewOrg] = useState<any>(null);
    const [statusFilter, setStatusFilter] = useState('');
    const [keyword, setKeyword] = useState('');
    const navigate = useNavigate();

    useEffect(() => {
        fetchData();
    }, [statusFilter, keyword]);

    const fetchData = async () => {
        setLoading(true);
        try {
            const params: any = {};
            if (statusFilter) params.status = statusFilter;
            if (keyword) params.keyword = keyword;
            const res = await orgApi.list(params);
            if (res.data.success) {
                setData(res.data.data || []);
            }
        } finally {
            setLoading(false);
        }
    };

    const handleDelete = async (id: string) => {
        try {
            await orgApi.delete(id);
            message.success('删除成功');
            fetchData();
        } catch {
            message.error('删除失败');
        }
    };

    const openRenew = (record: any) => {
        setRenewOrg(record);
        setRenewVisible(true);
    };

    const parentOptions = data.filter(org => org.level === 1);

    const getDaysUntilExpiry = (endDate: string) => {
        if (!endDate) return null;
        const days = dayjs(endDate).diff(dayjs(), 'day');
        return days;
    };

    const columns = [
        {
            title: '客户名称',
            dataIndex: 'name',
            key: 'name',
            width: 220,
            render: (text: string, record: any) => (
                <Space>
                    {record.level === 1 || !record.level ? (
                        <Tag color="blue">主账号</Tag>
                    ) : (
                        <Tag color="cyan">子账号</Tag>
                    )}
                    <a onClick={() => navigate(`/orgs/${record.id}`)}>{text}</a>
                </Space>
            )
        },
        {
            title: '公司名称',
            dataIndex: 'company_name',
            key: 'company_name',
            width: 200,
            ellipsis: true,
            render: (t: string) => t || '-',
        },
        { title: '联系人', dataIndex: 'contact_name', key: 'contact_name', width: 100 },
        { title: '联系电话', dataIndex: 'contact_phone', key: 'contact_phone', width: 130 },
        {
            title: '服务状态',
            dataIndex: 'service_status',
            key: 'service_status',
            width: 110,
            render: (status: string, record: any) => {
                if (record.level > 1) return <Tag>继承主账号</Tag>;
                const days = getDaysUntilExpiry(record.service_end);
                if (status === 'active' && days !== null && days <= 30 && days >= 0) {
                    return <Tag color="orange">即将到期</Tag>;
                }
                return <Tag color={statusColors[status]}>{statusLabels[status] || status}</Tag>;
            },
        },
        {
            title: '到期日期',
            dataIndex: 'service_end',
            key: 'service_end',
            width: 140,
            render: (t: string, record: any) => {
                if (record.level > 1) return '-';
                if (!t) return '-';
                const days = getDaysUntilExpiry(t);
                return (
                    <Space direction="vertical" size={0}>
                        <Text>{dayjs(t).format('YYYY-MM-DD')}</Text>
                        {days !== null && days <= 30 && days >= 0 && (
                            <Text type="danger" style={{ fontSize: 12 }}>剩余{days}天</Text>
                        )}
                    </Space>
                );
            },
        },
        {
            title: '设备配额',
            key: 'device_quota',
            width: 130,
            render: (_: any, record: any) => {
                if (record.level > 1) return '-';
                const used = record.device_count || 0;
                const max = record.max_devices || 10;
                const percent = Math.min(Math.round((used / max) * 100), 100);
                return (
                    <Space direction="vertical" size={0} style={{ width: '100%' }}>
                        <Text style={{ fontSize: 12 }}>{used} / {max}</Text>
                        <Progress percent={percent} size="small" showInfo={false}
                            strokeColor={percent > 90 ? '#ff4d4f' : '#1890ff'} />
                    </Space>
                );
            },
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 120,
            render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD') : '-',
        },
        {
            title: '操作',
            key: 'action',
            width: 200,
            render: (_: any, record: any) => (
                <Space>
                    <Button size="small" icon={<EditOutlined />} onClick={() => navigate(`/orgs/${record.id}`)}>详情</Button>
                    {record.level <= 1 && (
                        <Button size="small" icon={<DollarOutlined />} onClick={() => openRenew(record)}>续费</Button>
                    )}
                    <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
                        <Button size="small" danger icon={<DeleteOutlined />} />
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    const statusTabs = [
        { key: '', label: '全部' },
        { key: 'trial', label: '试用' },
        { key: 'active', label: '正常' },
        { key: 'expired', label: '已过期' },
        { key: 'suspended', label: '暂停' },
    ];

    return (
        <div>
            <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>客户管理</Title>
                <Space>
                    <Input.Search
                        placeholder="搜索客户名称/联系人/电话"
                        allowClear
                        style={{ width: 260 }}
                        onSearch={val => setKeyword(val)}
                    />
                    <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
                        新增客户
                    </Button>
                </Space>
            </Row>

            <Tabs
                activeKey={statusFilter}
                onChange={key => setStatusFilter(key)}
                items={statusTabs}
                style={{ marginBottom: 8 }}
            />

            <Table
                dataSource={data}
                columns={columns}
                rowKey="id"
                loading={loading}
                pagination={{ pageSize: 15 }}
                scroll={{ x: 1400 }}
                size="middle"
            />

            <CreateCustomerModal
                visible={createVisible}
                parentOptions={parentOptions}
                onClose={() => setCreateVisible(false)}
                onSuccess={() => { setCreateVisible(false); fetchData(); }}
            />

            <RenewModal
                visible={renewVisible}
                org={renewOrg}
                onClose={() => setRenewVisible(false)}
                onSuccess={() => { setRenewVisible(false); fetchData(); }}
            />
        </div>
    );
};

export default Organizations;
