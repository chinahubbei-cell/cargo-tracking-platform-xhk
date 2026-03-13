import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, List, Typography, Tag, Button, Space } from 'antd';
import {
    TeamOutlined,
    HddOutlined,
    WarningOutlined,
    ClockCircleOutlined,
    DollarOutlined,
} from '@ant-design/icons';
import { dashboardApi } from '../services/api';
import RenewModal from '../components/RenewModal';
import dayjs from 'dayjs';

const { Title } = Typography;

interface DashboardData {
    organizations: {
        total: number;
        active: number;
        expiring: number;
        expired: number;
    };
    devices: {
        total: number;
        in_stock: number;
        allocated: number;
        activated: number;
    };
    expiring_orgs: any[];
}

const Dashboard: React.FC = () => {
    const [data, setData] = useState<DashboardData | null>(null);
    const [loading, setLoading] = useState(true);
    const [renewVisible, setRenewVisible] = useState(false);
    const [renewOrg, setRenewOrg] = useState<any>(null);

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

    const openRenew = (org: any) => {
        setRenewOrg(org);
        setRenewVisible(true);
    };

    return (
        <div>
            <Title level={4} style={{ marginBottom: 24 }}>控制台</Title>

            {/* 统计卡片 */}
            <Row gutter={[16, 16]}>
                <Col xs={24} sm={12} lg={8}>
                    <Card>
                        <Statistic
                            title="活跃账号"
                            value={data?.organizations.active || 0}
                            prefix={<TeamOutlined style={{ color: '#52c41a' }} />}
                            suffix={`/ ${data?.organizations.total || 0}`}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={8}>
                    <Card>
                        <Statistic
                            title="即将到期"
                            value={data?.organizations.expiring || 0}
                            prefix={<WarningOutlined style={{ color: '#faad14' }} />}
                            valueStyle={{ color: data?.organizations.expiring ? '#faad14' : undefined }}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={8}>
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
                {/* 即将到期 */}
                <Col xs={24} lg={12}>
                    <Card
                        title={<span><ClockCircleOutlined /> 即将到期账号</span>}
                        size="small"
                    >
                        <List
                            dataSource={data?.expiring_orgs || []}
                            loading={loading}
                            renderItem={(org: any) => (
                                <List.Item
                                    actions={[
                                        <Button
                                            key="renew"
                                            type="primary"
                                            size="small"
                                            icon={<DollarOutlined />}
                                            onClick={() => openRenew(org)}
                                        >
                                            快速续费
                                        </Button>
                                    ]}
                                >
                                    <List.Item.Meta
                                        title={org.name}
                                        description={
                                            <Space>
                                                <span>到期: {dayjs(org.service_end).format('YYYY-MM-DD')}</span>
                                                {org.contact_name && <span>· {org.contact_name}</span>}
                                                {org.contact_phone && <span>· {org.contact_phone}</span>}
                                            </Space>
                                        }
                                    />
                                    <Tag color="warning">
                                        剩余 {dayjs(org.service_end).diff(dayjs(), 'day')} 天
                                    </Tag>
                                </List.Item>
                            )}
                            locale={{ emptyText: '暂无即将到期的账号' }}
                        />
                    </Card>
                </Col>

                {/* 设备统计 */}
                <Col xs={24} lg={12}>
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

            <RenewModal
                visible={renewVisible}
                org={renewOrg}
                onClose={() => setRenewVisible(false)}
                onSuccess={() => { setRenewVisible(false); fetchData(); }}
            />
        </div>
    );
};

export default Dashboard;
