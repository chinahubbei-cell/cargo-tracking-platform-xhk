import React, { useEffect, useState } from 'react';
import {
    Table, Button, Space, Tag, Modal, Form, Input, InputNumber,
    DatePicker, Select, message, Typography, Card, Row, Col, Popconfirm
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { orgApi } from '../services/api';
import dayjs from 'dayjs';

const { Title } = Typography;

const Organizations: React.FC = () => {
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalVisible, setModalVisible] = useState(false);
    const [editingOrg, setEditingOrg] = useState<any>(null);
    const [form] = Form.useForm();

    useEffect(() => {
        fetchData();
    }, []);

    const fetchData = async () => {
        setLoading(true);
        try {
            const res = await orgApi.list();
            if (res.data.success) {
                setData(res.data.data || []);
            }
        } finally {
            setLoading(false);
        }
    };

    const handleCreate = () => {
        setEditingOrg(null);
        form.resetFields();
        setModalVisible(true);
    };

    const handleEdit = (record: any) => {
        setEditingOrg(record);
        form.setFieldsValue({
            ...record,
            service_start: record.service_start ? dayjs(record.service_start) : null,
            service_end: record.service_end ? dayjs(record.service_end) : null,
        });
        setModalVisible(true);
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

    const handleSubmit = async () => {
        const values = await form.validateFields();
        const payload = {
            ...values,
            service_start: values.service_start?.toISOString(),
            service_end: values.service_end?.toISOString(),
        };

        try {
            if (editingOrg) {
                await orgApi.update(editingOrg.id, payload);
                message.success('更新成功');
            } else {
                await orgApi.create(payload);
                message.success('创建成功');
            }
            setModalVisible(false);
            fetchData();
        } catch {
            message.error('操作失败');
        }
    };

    const columns = [
        { title: '组织名称', dataIndex: 'name', key: 'name', width: 200 },
        { title: '联系人', dataIndex: 'contact_name', key: 'contact_name', width: 100 },
        { title: '联系电话', dataIndex: 'contact_phone', key: 'contact_phone', width: 130 },
        {
            title: '服务状态',
            dataIndex: 'service_status',
            key: 'service_status',
            width: 100,
            render: (status: string) => {
                const colors: Record<string, string> = {
                    trial: 'blue', active: 'green', suspended: 'orange', expired: 'red',
                };
                const labels: Record<string, string> = {
                    trial: '试用', active: '正常', suspended: '暂停', expired: '已过期',
                };
                return <Tag color={colors[status]}>{labels[status] || status}</Tag>;
            },
        },
        {
            title: '到期时间',
            dataIndex: 'service_end',
            key: 'service_end',
            width: 120,
            render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD') : '-',
        },
        { title: '最大设备数', dataIndex: 'max_devices', key: 'max_devices', width: 100 },
        {
            title: '操作',
            key: 'action',
            width: 150,
            render: (_: any, record: any) => (
                <Space>
                    <Button size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                    <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
                        <Button size="small" danger icon={<DeleteOutlined />} />
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    return (
        <div>
            <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>组织管理</Title>
                <Space>
                    <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
                    <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
                        新增组织
                    </Button>
                </Space>
            </Row>

            <Table
                dataSource={data}
                columns={columns}
                rowKey="id"
                loading={loading}
                pagination={{ pageSize: 10 }}
                scroll={{ x: 1000 }}
            />

            <Modal
                title={editingOrg ? '编辑组织' : '新增组织'}
                open={modalVisible}
                onOk={handleSubmit}
                onCancel={() => setModalVisible(false)}
                width={600}
            >
                <Form form={form} layout="vertical">
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="name" label="组织名称" rules={[{ required: true }]}>
                                <Input />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="short_name" label="简称">
                                <Input />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="contact_name" label="联系人">
                                <Input />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="contact_phone" label="联系电话">
                                <Input />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Form.Item name="contact_email" label="邮箱">
                        <Input />
                    </Form.Item>
                    <Form.Item name="address" label="地址">
                        <Input.TextArea rows={2} />
                    </Form.Item>
                    <Row gutter={16}>
                        <Col span={8}>
                            <Form.Item name="service_status" label="服务状态">
                                <Select>
                                    <Select.Option value="trial">试用</Select.Option>
                                    <Select.Option value="active">正常</Select.Option>
                                    <Select.Option value="suspended">暂停</Select.Option>
                                    <Select.Option value="expired">已过期</Select.Option>
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="service_start" label="服务开始">
                                <DatePicker style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="service_end" label="服务到期">
                                <DatePicker style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={8}>
                            <Form.Item name="max_devices" label="最大设备数">
                                <InputNumber min={1} style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="max_users" label="最大用户数">
                                <InputNumber min={1} style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="max_shipments" label="月运单配额">
                                <InputNumber min={1} style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                    </Row>
                </Form>
            </Modal>
        </div>
    );
};

export default Organizations;
