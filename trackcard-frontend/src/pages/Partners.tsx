import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Button, Space, Input, Select, Tag, Modal, Form, message, Rate, Statistic, Row, Col, Tooltip, Popconfirm, Tabs, Divider } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, TeamOutlined, BarChartOutlined, SearchOutlined, ReloadOutlined, GlobalOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { api } from '../api/client';
import type { Partner, PartnerType, PartnerCreateRequest, PartnerUpdateRequest, LogisticsStage } from '../types';
import { PartnerTypeNames, LogisticsStageNames, PartnerTypesByStage } from '../types';
import './Partners.css';

const { Option, OptGroup } = Select;

const Partners: React.FC = () => {
    const [partners, setPartners] = useState<Partner[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalVisible, setModalVisible] = useState(false);
    const [editingPartner, setEditingPartner] = useState<Partner | null>(null);
    const [statsVisible, setStatsVisible] = useState(false);
    const [statsData, setStatsData] = useState<{ partner_id: string; total_collabs: number; completed_collabs: number; completion_rate: number; avg_duration_hrs: number } | null>(null);
    const [selectedPartner, setSelectedPartner] = useState<Partner | null>(null);
    const [filters, setFilters] = useState<{ type?: string; status?: string; search?: string; stage?: string; country?: string }>({});
    const [activeStage, setActiveStage] = useState<string>('all');
    const [form] = Form.useForm();

    const fetchPartners = useCallback(async () => {
        setLoading(true);
        try {
            const queryFilters = { ...filters };
            if (activeStage !== 'all') {
                queryFilters.stage = activeStage;
            }
            const response = await api.getPartners(queryFilters);
            if (response.success) {
                setPartners(response.data || []);
            }
        } catch (error) {
            message.error('获取合作伙伴列表失败');
        } finally {
            setLoading(false);
        }
    }, [filters, activeStage]);

    useEffect(() => {
        fetchPartners();
    }, [fetchPartners]);

    const handleCreate = () => {
        setEditingPartner(null);
        form.resetFields();
        setModalVisible(true);
    };

    const handleEdit = (partner: Partner) => {
        setEditingPartner(partner);
        form.setFieldsValue(partner);
        setModalVisible(true);
    };

    const handleDelete = async (id: string) => {
        try {
            await api.deletePartner(id);
            message.success('删除成功');
            fetchPartners();
        } catch (error) {
            message.error('删除失败');
        }
    };

    const handleSubmit = async () => {
        try {
            const values = await form.validateFields();
            if (editingPartner) {
                const updateData: PartnerUpdateRequest = {
                    name: values.name,
                    sub_type: values.sub_type,
                    contact_name: values.contact_name,
                    phone: values.phone,
                    email: values.email,
                    address: values.address,
                    country: values.country,
                    region: values.region,
                    service_ports: values.service_ports,
                    service_routes: values.service_routes,
                    payment_terms: values.payment_terms,
                };
                await api.updatePartner(editingPartner.id, updateData);
                message.success('更新成功');
            } else {
                const createData: PartnerCreateRequest = {
                    name: values.name,
                    code: values.code,
                    type: values.type,
                    stage: values.stage,
                    sub_type: values.sub_type,
                    contact_name: values.contact_name,
                    phone: values.phone,
                    email: values.email,
                    address: values.address,
                    country: values.country,
                    region: values.region,
                    service_ports: values.service_ports,
                    service_routes: values.service_routes,
                    payment_terms: values.payment_terms,
                };
                await api.createPartner(createData);
                message.success('创建成功');
            }
            setModalVisible(false);
            fetchPartners();
        } catch (error) {
            message.error('操作失败');
        }
    };

    const handleViewStats = async (partner: Partner) => {
        setSelectedPartner(partner);
        try {
            const response = await api.getPartnerStats(partner.id);
            if (response.success && response.data) {
                setStatsData(response.data);
                setStatsVisible(true);
            }
        } catch (error) {
            message.error('获取统计数据失败');
        }
    };

    const getTypeColor = (type: PartnerType): string => {
        const colors: Record<string, string> = {
            // 前程运输 - 蓝色系
            drayage_origin: '#1890ff',
            consolidator: '#40a9ff',
            inspector: '#69c0ff',
            // 起运港 - 绿色系
            booking_agent: '#52c41a',
            customs_export: '#73d13d',
            terminal: '#95de64',
            // 干线运输 - 紫色系
            vocc: '#722ed1',
            nvocc: '#9254de',
            airline: '#b37feb',
            // 目的港 - 橙色系
            customs_import: '#fa8c16',
            drayage_dest: '#ffa940',
            chassis_provider: '#ffc069',
            bonded_warehouse: '#ffd591',
            // 末端配送 - 红色系
            overseas_3pl: '#f5222d',
            courier: '#ff4d4f',
            platform_warehouse: '#ff7875',
            // 兼容旧类型
            forwarder: '#1890ff',
            broker: '#52c41a',
            trucker: '#fa8c16',
            warehouse: '#722ed1',
        };
        return colors[type] || '#666';
    };

    const getStageColor = (stage?: LogisticsStage): string => {
        const colors: Record<string, string> = {
            first_mile: '#1890ff',
            origin_port: '#52c41a',
            main_leg: '#722ed1',
            dest_port: '#fa8c16',
            last_mile: '#f5222d',
        };
        return stage ? colors[stage] || '#666' : '#666';
    };

    const columns: ColumnsType<Partner> = [
        {
            title: '编号',
            dataIndex: 'code',
            key: 'code',
            width: 100,
        },
        {
            title: '名称',
            dataIndex: 'name',
            key: 'name',
            render: (name: string, record: Partner) => (
                <Space>
                    <TeamOutlined style={{ color: getTypeColor(record.type) }} />
                    <span style={{ fontWeight: 500 }}>{name}</span>
                </Space>
            ),
        },
        {
            title: '物流环节',
            dataIndex: 'stage',
            key: 'stage',
            width: 100,
            render: (stage: LogisticsStage) => stage ? (
                <Tag color={getStageColor(stage)}>{LogisticsStageNames[stage]}</Tag>
            ) : '-',
        },
        {
            title: '类型',
            dataIndex: 'type',
            key: 'type',
            width: 110,
            render: (type: PartnerType) => (
                <Tag color={getTypeColor(type)}>{PartnerTypeNames[type] || type}</Tag>
            ),
        },
        {
            title: '服务区域',
            key: 'region',
            width: 120,
            render: (_: unknown, record: Partner) => (
                <Space size="small">
                    {record.country && <GlobalOutlined />}
                    <span>{record.country || '-'}</span>
                </Space>
            ),
        },
        {
            title: '联系人',
            dataIndex: 'contact_name',
            key: 'contact_name',
            width: 100,
        },
        {
            title: '电话',
            dataIndex: 'phone',
            key: 'phone',
            width: 130,
        },
        {
            title: '评分',
            dataIndex: 'rating',
            key: 'rating',
            width: 140,
            render: (rating: number) => (
                <Rate disabled value={rating} allowHalf style={{ fontSize: 12 }} />
            ),
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 70,
            render: (status: string) => (
                <Tag color={status === 'active' ? 'green' : 'default'}>
                    {status === 'active' ? '启用' : '停用'}
                </Tag>
            ),
        },
        {
            title: '操作',
            key: 'action',
            width: 140,
            render: (_, record) => (
                <Space size="small">
                    <Tooltip title="统计">
                        <Button type="text" size="small" icon={<BarChartOutlined />} onClick={() => handleViewStats(record)} />
                    </Tooltip>
                    <Tooltip title="编辑">
                        <Button type="text" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                    </Tooltip>
                    <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
                        <Tooltip title="删除">
                            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                        </Tooltip>
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    const stageTabItems = [
        { key: 'all', label: '全部' },
        { key: 'first_mile', label: '🚚 前程运输' },
        { key: 'origin_port', label: '🏗️ 起运港' },
        { key: 'main_leg', label: '🚢 干线运输' },
        { key: 'dest_port', label: '🛃 目的港' },
        { key: 'last_mile', label: '📦 末端配送' },
    ];

    // 根据选中的stage过滤可用的type选项
    const getTypeOptionsByStage = (stage?: LogisticsStage) => {
        if (!stage) {
            // 返回所有类型，按环节分组
            return (
                <>
                    <OptGroup label="前程运输">
                        {PartnerTypesByStage.first_mile.map(t => (
                            <Option key={t} value={t}>{PartnerTypeNames[t]}</Option>
                        ))}
                    </OptGroup>
                    <OptGroup label="起运港">
                        {PartnerTypesByStage.origin_port.map(t => (
                            <Option key={t} value={t}>{PartnerTypeNames[t]}</Option>
                        ))}
                    </OptGroup>
                    <OptGroup label="干线运输">
                        {PartnerTypesByStage.main_leg.map(t => (
                            <Option key={t} value={t}>{PartnerTypeNames[t]}</Option>
                        ))}
                    </OptGroup>
                    <OptGroup label="目的港">
                        {PartnerTypesByStage.dest_port.map(t => (
                            <Option key={t} value={t}>{PartnerTypeNames[t]}</Option>
                        ))}
                    </OptGroup>
                    <OptGroup label="末端配送">
                        {PartnerTypesByStage.last_mile.map(t => (
                            <Option key={t} value={t}>{PartnerTypeNames[t]}</Option>
                        ))}
                    </OptGroup>
                    <OptGroup label="通用">
                        <Option value="forwarder">货代</Option>
                        <Option value="broker">报关行</Option>
                        <Option value="trucker">拖车行</Option>
                        <Option value="warehouse">仓库</Option>
                    </OptGroup>
                </>
            );
        }
        // 返回该环节对应的类型
        const types = PartnerTypesByStage[stage] || [];
        return types.map(t => (
            <Option key={t} value={t}>{PartnerTypeNames[t]}</Option>
        ));
    };

    return (
        <div className="partners-page">
            <Card
                title={
                    <Space>
                        <TeamOutlined />
                        <span>合作伙伴管理</span>
                        <Tag color="blue">{partners.length} 个</Tag>
                    </Space>
                }
                extra={
                    <Space>
                        <Input
                            placeholder="搜索名称/编号"
                            prefix={<SearchOutlined />}
                            style={{ width: 180 }}
                            onChange={(e) => setFilters({ ...filters, search: e.target.value })}
                            onPressEnter={fetchPartners}
                            allowClear
                        />
                        <Select
                            placeholder="类型"
                            style={{ width: 120 }}
                            allowClear
                            onChange={(value) => setFilters({ ...filters, type: value })}
                        >
                            {getTypeOptionsByStage()}
                        </Select>
                        <Select
                            placeholder="状态"
                            style={{ width: 90 }}
                            allowClear
                            onChange={(value) => setFilters({ ...filters, status: value })}
                        >
                            <Option value="active">启用</Option>
                            <Option value="inactive">停用</Option>
                        </Select>
                        <Button icon={<ReloadOutlined />} onClick={fetchPartners}>刷新</Button>
                        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新增伙伴</Button>
                    </Space>
                }
            >
                <Tabs
                    activeKey={activeStage}
                    onChange={setActiveStage}
                    items={stageTabItems}
                    style={{ marginBottom: 16 }}
                />
                <Table
                    columns={columns}
                    dataSource={partners}
                    rowKey="id"
                    loading={loading}
                    size="small"
                    pagination={{ pageSize: 10, showTotal: (total) => `共 ${total} 条` }}
                />
            </Card>

            {/* 创建/编辑对话框 */}
            <Modal
                title={editingPartner ? '编辑合作伙伴' : '新增合作伙伴'}
                open={modalVisible}
                onOk={handleSubmit}
                onCancel={() => setModalVisible(false)}
                width={700}
            >
                <Form form={form} layout="vertical">
                    <Divider orientation={"left" as any} plain>基本信息</Divider>
                    <Row gutter={16}>
                        <Col span={8}>
                            <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
                                <Input placeholder="请输入合作伙伴名称" />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="code" label="编号" rules={[{ required: true, message: '请输入编号' }]}>
                                <Input placeholder="请输入唯一编号" disabled={!!editingPartner} />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="stage" label="物流环节">
                                <Select placeholder="选择物流环节" allowClear disabled={!!editingPartner}>
                                    <Option value="first_mile">前程运输</Option>
                                    <Option value="origin_port">起运港</Option>
                                    <Option value="main_leg">干线运输</Option>
                                    <Option value="dest_port">目的港</Option>
                                    <Option value="last_mile">末端配送</Option>
                                </Select>
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={8}>
                            <Form.Item name="type" label="类型" rules={[{ required: true, message: '请选择类型' }]}>
                                <Select placeholder="请选择类型" disabled={!!editingPartner}>
                                    {getTypeOptionsByStage(form.getFieldValue('stage'))}
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="country" label="服务国家">
                                <Input placeholder="如: 中国, 美国" />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="region" label="服务区域">
                                <Input placeholder="如: 华东, 西海岸" />
                            </Form.Item>
                        </Col>
                    </Row>

                    <Divider orientation={"left" as any} plain>联系方式</Divider>
                    <Row gutter={16}>
                        <Col span={8}>
                            <Form.Item name="contact_name" label="联系人">
                                <Input placeholder="请输入联系人姓名" />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="phone" label="电话">
                                <Input placeholder="请输入联系电话" />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="email" label="邮箱">
                                <Input placeholder="请输入邮箱地址" />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Form.Item name="address" label="地址">
                        <Input.TextArea placeholder="请输入地址" rows={2} />
                    </Form.Item>

                    <Divider orientation={"left" as any} plain>业务信息</Divider>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="service_ports" label="服务港口">
                                <Input placeholder="多个港口用逗号分隔，如: CNSHA,CNNBO" />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="service_routes" label="服务航线">
                                <Input placeholder="多个航线用逗号分隔" />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="payment_terms" label="账期">
                                <Select placeholder="选择账期" allowClear>
                                    <Option value="PREPAID">预付</Option>
                                    <Option value="NET15">15天账期</Option>
                                    <Option value="NET30">30天账期</Option>
                                    <Option value="NET45">45天账期</Option>
                                    <Option value="NET60">60天账期</Option>
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="sub_type" label="细分类型">
                                <Input placeholder="如: 美线专家, FBA专线" />
                            </Form.Item>
                        </Col>
                    </Row>
                </Form>
            </Modal>

            {/* 统计对话框 */}
            <Modal
                title={`${selectedPartner?.name || ''} - 绩效统计`}
                open={statsVisible}
                onCancel={() => setStatsVisible(false)}
                footer={null}
                width={500}
            >
                {statsData && (
                    <Row gutter={16}>
                        <Col span={12}>
                            <Statistic title="总协作次数" value={statsData.total_collabs} />
                        </Col>
                        <Col span={12}>
                            <Statistic title="已完成" value={statsData.completed_collabs} />
                        </Col>
                        <Col span={12} style={{ marginTop: 16 }}>
                            <Statistic title="完成率" value={statsData.completion_rate} suffix="%" precision={1} />
                        </Col>
                        <Col span={12} style={{ marginTop: 16 }}>
                            <Statistic title="平均完成时间" value={statsData.avg_duration_hrs || 0} suffix="小时" precision={1} />
                        </Col>
                    </Row>
                )}
            </Modal>
        </div>
    );
};

export default Partners;
