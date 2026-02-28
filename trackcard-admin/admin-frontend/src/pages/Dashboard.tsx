import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, List, Typography } from 'antd';
import {
    TeamOutlined,
    ShoppingCartOutlined,
    HddOutlined,
    WarningOutlined,
    ClockCircleOutlined,
    CheckCircleOutlined,
} from '@ant-design/icons';
import { dashboardApi } from '../services/api';
import dayjs from 'dayjs';

const { Title } = Typography;

interface DashboardData {
    organizations: {
        total: number;
        active: number;
        expiring: number;
        expired: number;
    };
    orders: {
        pending: number;
        processing: number;
        completed: number;
    };
    devices: {
        total: number;
        in_stock: number;
        allocated: number;
        activated: number;
    };
    recent_orders: any[];
    expiring_orgs: any[];
}

const Dashboard: React.FC = () => {
    const [data, setData] = useState<DashboardData | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetchData();
    }, []);

    const fetchData = async () => {
        try {
            const res = await dashboardApi.getStats();
            if (res.data.success) {
                setData(res.data.data);
            }
        } finally {
            setLoading(false);
        }
    };

    const orderColumns = [
        { title: '订单号', dataIndex: 'order_no', key: 'order_no', width: 180 },
        { title: '客户', dataIndex: 'org_name', key: 'org_name' },
        {
            title: '状态',
            dataIndex: 'order_status',
            key: 'order_status',
            render: (status: string) => {
                const colors: Record<string, string> = {
                    pending: 'orange',
                    confirmed: 'blue',
                    shipped: 'cyan',
                    completed: 'green',
                    cancelled: 'red',
                };
                const labels: Record<string, string> = {
                    pending: '待确认',
                    confirmed: '已确认',
                    shipped: '已发货',
                    completed: '已完成',
                    cancelled: '已取消',
                };
                return <Tag color={colors[status]}>{labels[status] || status}</Tag>;
            },
        },
        {
            title: '时间',
            dataIndex: 'created_at',
            key: 'created_at',
            render: (t: string) => dayjs(t).format('MM-DD HH:mm'),
        },
    ];

    return (
        <div>
            <Title level={4} style={{ marginBottom: 24 }}>控制台</Title>

            {/* 统计卡片 */}
            <Row gutter={[16, 16]}>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="活跃组织"
                            value={data?.organizations.active || 0}
                            prefix={<TeamOutlined style={{ color: '#52c41a' }} />}
                            suffix={`/ ${data?.organizations.total || 0}`}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="即将到期"
                            value={data?.organizations.expiring || 0}
                            prefix={<WarningOutlined style={{ color: '#faad14' }} />}
                            valueStyle={{ color: data?.organizations.expiring ? '#faad14' : undefined }}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="待处理订单"
                            value={(data?.orders.pending || 0) + (data?.orders.processing || 0)}
                            prefix={<ShoppingCartOutlined style={{ color: '#1890ff' }} />}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="设备库存"
                            value={data?.devices.in_stock || 0}
                            prefix={<HddOutlined style={{ color: '#722ed1' }} />}
                            suffix={`/ ${data?.devices.total || 0}`}
                        />
                    </Card>
                </Col>
            </Row>

            <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
                {/* 最近订单 */}
                <Col xs={24} lg={14}>
                    <Card title="最近订单" size="small">
                        <Table
                            dataSource={data?.recent_orders || []}
                            columns={orderColumns}
                            rowKey="id"
                            size="small"
                            pagination={false}
                            loading={loading}
                        />
                    </Card>
                </Col>

                {/* 即将到期 */}
                <Col xs={24} lg={10}>
                    <Card
                        title={<span><ClockCircleOutlined /> 即将到期组织</span>}
                        size="small"
                    >
                        <List
                            dataSource={data?.expiring_orgs || []}
                            loading={loading}
                            renderItem={(org: any) => (
                                <List.Item>
                                    <List.Item.Meta
                                        title={org.name}
                                        description={`到期时间: ${dayjs(org.service_end).format('YYYY-MM-DD')}`}
                                    />
                                    <Tag color="warning">
                                        剩余 {dayjs(org.service_end).diff(dayjs(), 'day')} 天
                                    </Tag>
                                </List.Item>
                            )}
                            locale={{ emptyText: '暂无即将到期的组织' }}
                        />
                    </Card>
                </Col>
            </Row>

            {/* 设备统计 */}
            <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
                <Col span={24}>
                    <Card title="设备统计">
                        <Row gutter={16}>
                            <Col xs={12} sm={6}>
                                <Statistic
                                    title="库存中"
                                    value={data?.devices.in_stock || 0}
                                    valueStyle={{ color: '#52c41a' }}
                                />
                            </Col>
                            <Col xs={12} sm={6}>
                                <Statistic
                                    title="已分配"
                                    value={data?.devices.allocated || 0}
                                    valueStyle={{ color: '#1890ff' }}
                                />
                            </Col>
                            <Col xs={12} sm={6}>
                                <Statistic
                                    title="已激活"
                                    value={data?.devices.activated || 0}
                                    valueStyle={{ color: '#722ed1' }}
                                />
                            </Col>
                            <Col xs={12} sm={6}>
                                <Statistic
                                    title="总计"
                                    value={data?.devices.total || 0}
                                />
                            </Col>
                        </Row>
                    </Card>
                </Col>
            </Row>
        </div>
    );
};

export default Dashboard;
