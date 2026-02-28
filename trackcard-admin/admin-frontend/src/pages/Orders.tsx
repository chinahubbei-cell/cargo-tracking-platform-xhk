import React, { useEffect, useState } from 'react';
import {
    Table, Button, Space, Tag, Modal, Form, Input, InputNumber,
    Select, message, Typography, Row, Col, Popconfirm, Descriptions
} from 'antd';
import { PlusOutlined, EyeOutlined, CheckOutlined, SendOutlined, CloseOutlined, ReloadOutlined } from '@ant-design/icons';
import { orderApi, orgApi } from '../services/api';
import dayjs from 'dayjs';

const { Title } = Typography;

const Orders: React.FC = () => {
    const [data, setData] = useState<any[]>([]);
    const [orgs, setOrgs] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalVisible, setModalVisible] = useState(false);
    const [detailVisible, setDetailVisible] = useState(false);
    const [shipVisible, setShipVisible] = useState(false);
    const [currentOrder, setCurrentOrder] = useState<any>(null);
    const [form] = Form.useForm();
    const [shipForm] = Form.useForm();

    useEffect(() => {
        fetchData();
        fetchOrgs();
    }, []);

    const fetchData = async () => {
        setLoading(true);
        try {
            const res = await orderApi.list();
            if (res.data.success) {
                setData(res.data.data || []);
            }
        } finally {
            setLoading(false);
        }
    };

    const fetchOrgs = async () => {
        const res = await orgApi.list();
        if (res.data.success) {
            setOrgs(res.data.data || []);
        }
    };

    const handleCreate = () => {
        form.resetFields();
        setModalVisible(true);
    };

    const handleSubmit = async () => {
        const values = await form.validateFields();
        const org = orgs.find(o => o.id === values.org_id);
        const payload = {
            ...values,
            org_name: org?.name || '',
        };

        try {
            await orderApi.create(payload);
            message.success('创建成功');
            setModalVisible(false);
            fetchData();
        } catch {
            message.error('创建失败');
        }
    };

    const handleConfirm = async (id: string) => {
        try {
            await orderApi.confirm(id);
            message.success('订单已确认');
            fetchData();
        } catch {
            message.error('操作失败');
        }
    };

    const handleShip = (record: any) => {
        setCurrentOrder(record);
        shipForm.resetFields();
        setShipVisible(true);
    };

    const handleShipSubmit = async () => {
        const values = await shipForm.validateFields();
        try {
            await orderApi.ship(currentOrder.id, values);
            message.success('发货成功');
            setShipVisible(false);
            fetchData();
        } catch {
            message.error('操作失败');
        }
    };

    const handleComplete = async (id: string) => {
        try {
            await orderApi.complete(id);
            message.success('订单已完成');
            fetchData();
        } catch {
            message.error('操作失败');
        }
    };

    const handleCancel = async (id: string) => {
        try {
            await orderApi.cancel(id);
            message.success('订单已取消');
            fetchData();
        } catch {
            message.error('操作失败');
        }
    };

    const statusColors: Record<string, string> = {
        pending: 'orange', confirmed: 'blue', processing: 'cyan',
        shipped: 'purple', completed: 'green', cancelled: 'red',
    };
    const statusLabels: Record<string, string> = {
        pending: '待确认', confirmed: '已确认', processing: '处理中',
        shipped: '已发货', completed: '已完成', cancelled: '已取消',
    };

    const columns = [
        { title: '订单号', dataIndex: 'order_no', key: 'order_no', width: 180 },
        { title: '客户', dataIndex: 'org_name', key: 'org_name', width: 150 },
        {
            title: '类型',
            dataIndex: 'order_type',
            key: 'order_type',
            width: 80,
            render: (t: string) => {
                const labels: Record<string, string> = { purchase: '购买', renew: '续费', upgrade: '升级' };
                return labels[t] || t;
            },
        },
        { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 60 },
        {
            title: '金额',
            dataIndex: 'total_amount',
            key: 'total_amount',
            width: 100,
            render: (v: number) => `¥${v?.toFixed(2) || '0.00'}`
        },
        {
            title: '状态',
            dataIndex: 'order_status',
            key: 'order_status',
            width: 90,
            render: (s: string) => <Tag color={statusColors[s]}>{statusLabels[s] || s}</Tag>,
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 150,
            render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm'),
        },
        {
            title: '操作',
            key: 'action',
            width: 200,
            render: (_: any, record: any) => (
                <Space size="small">
                    <Button size="small" icon={<EyeOutlined />} onClick={() => { setCurrentOrder(record); setDetailVisible(true); }} />
                    {record.order_status === 'pending' && (
                        <Popconfirm title="确认此订单？" onConfirm={() => handleConfirm(record.id)}>
                            <Button size="small" type="primary" icon={<CheckOutlined />}>确认</Button>
                        </Popconfirm>
                    )}
                    {(record.order_status === 'confirmed' || record.order_status === 'processing') && (
                        <Button size="small" type="primary" icon={<SendOutlined />} onClick={() => handleShip(record)}>发货</Button>
                    )}
                    {record.order_status === 'shipped' && (
                        <Popconfirm title="确认完成？" onConfirm={() => handleComplete(record.id)}>
                            <Button size="small" type="primary">完成</Button>
                        </Popconfirm>
                    )}
                    {!['shipped', 'completed', 'cancelled'].includes(record.order_status) && (
                        <Popconfirm title="确认取消？" onConfirm={() => handleCancel(record.id)}>
                            <Button size="small" danger icon={<CloseOutlined />} />
                        </Popconfirm>
                    )}
                </Space>
            ),
        },
    ];

    return (
        <div>
            <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>订单管理</Title>
                <Space>
                    <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
                    <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新增订单</Button>
                </Space>
            </Row>

            <Table dataSource={data} columns={columns} rowKey="id" loading={loading} scroll={{ x: 1200 }} />

            {/* 新增订单 */}
            <Modal title="新增订单" open={modalVisible} onOk={handleSubmit} onCancel={() => setModalVisible(false)} width={500}>
                <Form form={form} layout="vertical">
                    <Form.Item name="org_id" label="客户" rules={[{ required: true }]}>
                        <Select showSearch optionFilterProp="children">
                            {orgs.map(org => <Select.Option key={org.id} value={org.id}>{org.name}</Select.Option>)}
                        </Select>
                    </Form.Item>
                    <Form.Item name="order_type" label="订单类型" rules={[{ required: true }]}>
                        <Select>
                            <Select.Option value="purchase">购买</Select.Option>
                            <Select.Option value="renew">续费</Select.Option>
                            <Select.Option value="upgrade">升级</Select.Option>
                        </Select>
                    </Form.Item>
                    <Row gutter={16}>
                        <Col span={8}><Form.Item name="quantity" label="数量"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={8}><Form.Item name="unit_price" label="单价"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={8}><Form.Item name="total_amount" label="总金额"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                    </Row>
                    <Form.Item name="contact_name" label="联系人"><Input /></Form.Item>
                    <Form.Item name="phone" label="电话"><Input /></Form.Item>
                    <Form.Item name="shipping_address" label="收货地址"><Input.TextArea rows={2} /></Form.Item>
                </Form>
            </Modal>

            {/* 发货 */}
            <Modal title="发货" open={shipVisible} onOk={handleShipSubmit} onCancel={() => setShipVisible(false)}>
                <Form form={shipForm} layout="vertical">
                    <Form.Item name="tracking_company" label="快递公司"><Input placeholder="如：顺丰、圆通" /></Form.Item>
                    <Form.Item name="tracking_no" label="快递单号"><Input /></Form.Item>
                </Form>
            </Modal>

            {/* 详情 */}
            <Modal title="订单详情" open={detailVisible} onCancel={() => setDetailVisible(false)} footer={null} width={600}>
                {currentOrder && (
                    <Descriptions column={2} bordered size="small">
                        <Descriptions.Item label="订单号">{currentOrder.order_no}</Descriptions.Item>
                        <Descriptions.Item label="状态"><Tag color={statusColors[currentOrder.order_status]}>{statusLabels[currentOrder.order_status]}</Tag></Descriptions.Item>
                        <Descriptions.Item label="客户">{currentOrder.org_name}</Descriptions.Item>
                        <Descriptions.Item label="类型">{currentOrder.order_type}</Descriptions.Item>
                        <Descriptions.Item label="数量">{currentOrder.quantity}</Descriptions.Item>
                        <Descriptions.Item label="金额">¥{currentOrder.total_amount?.toFixed(2)}</Descriptions.Item>
                        <Descriptions.Item label="收货地址" span={2}>{currentOrder.shipping_address}</Descriptions.Item>
                        <Descriptions.Item label="快递公司">{currentOrder.tracking_company || '-'}</Descriptions.Item>
                        <Descriptions.Item label="快递单号">{currentOrder.tracking_no || '-'}</Descriptions.Item>
                    </Descriptions>
                )}
            </Modal>
        </div>
    );
};

export default Orders;
