import React, { useState, useEffect, useCallback } from 'react';
import {
    Table,
    Card,
    Button,
    Input,
    Space,

    Modal,
    Form,
    message,
    Tabs,
    Popconfirm,
    Tooltip
} from 'antd';
import {
    PlusOutlined,
    SearchOutlined,
    EditOutlined,
    DeleteOutlined,
    UserOutlined,
    TeamOutlined,
    ReloadOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useAuthStore } from '../store/authStore';
import api from '../api/client';
import type { Customer, CustomerType } from '../types';
import AddressInput from '../components/AddressInput';
import type { AddressData } from '../components/AddressInput';

const Customers: React.FC = () => {
    const [activeTab, setActiveTab] = useState<CustomerType>('sender');
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<Customer[]>([]);
    const [modalVisible, setModalVisible] = useState(false);
    const [editingId, setEditingId] = useState<string | null>(null);
    const [form] = Form.useForm();
    const [searchText, setSearchText] = useState('');
    const { token } = useAuthStore();

    // 加载数据
    const fetchData = useCallback(async () => {
        if (!token) return;
        setLoading(true);
        try {
            // 目前后端API没有提供搜索参数，这里先获取全部后前端过滤，或者等后端完善Search接口
            // 使用 List 接口并带上 type 参数
            const res = await api.getCustomers({ type: activeTab });
            if (res.data) {
                // 如果有搜索词，前端简单的过滤一下
                let list = res.data as Customer[];
                if (searchText) {
                    const lowerKey = searchText.toLowerCase();
                    list = list.filter(item =>
                        item.name.toLowerCase().includes(lowerKey) ||
                        item.phone.includes(lowerKey) ||
                        (item.company && item.company.toLowerCase().includes(lowerKey))
                    );
                }
                setData(list);
            }
        } catch (error) {
            console.error('获取客户列表失败:', error);
            message.error('获取客户列表失败');
        } finally {
            setLoading(false);
        }
    }, [token, activeTab, searchText]);

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    // 处理提交
    const handleSubmit = async () => {
        try {
            const values = await form.validateFields();
            // 确保类型正确
            const payload = { ...values, type: activeTab };

            if (editingId) {
                await api.updateCustomer(editingId, payload);
                message.success('更新成功');
            } else {
                await api.createCustomer(payload);
                message.success('创建成功');
            }
            setModalVisible(false);
            form.resetFields();
            setEditingId(null);
            fetchData();
        } catch (error: any) {
            console.error('保存失败:', error);
            // 处理后端返回的 specific error (比如手机号重复)
            if (error.response?.data?.error) {
                message.error(error.response.data.error);
            } else {
                message.error(`保存失败: ${error.message || '未知错误'}`);
            }
        }
    };

    // 处理删除
    const handleDelete = async (id: string) => {
        try {
            await api.deleteCustomer(id);
            message.success('删除成功');
            fetchData();
        } catch (error) {
            console.error('删除失败:', error);
            message.error('删除失败');
        }
    };

    // 打开编辑
    const handleEdit = (record: Customer) => {
        setEditingId(record.id);
        form.setFieldsValue(record);
        setModalVisible(true);
    };

    // 打开新增
    const handleAdd = () => {
        setEditingId(null);
        form.resetFields();
        setModalVisible(true);
    };

    const columns: ColumnsType<Customer> = [
        {
            title: '姓名',
            dataIndex: 'name',
            key: 'name',
            width: 150,
            fixed: 'left',
            render: (text) => <span style={{ fontWeight: 500 }}>{text}</span>
        },
        {
            title: '手机号',
            dataIndex: 'phone',
            key: 'phone',
            width: 180,
            render: (text) => <div style={{ whiteSpace: 'nowrap' }}>{text}</div>
        },
        {
            title: '公司',
            dataIndex: 'company',
            key: 'company',
            width: 200,
            ellipsis: true,
        },
        {
            title: '地址',
            dataIndex: 'address',
            key: 'address',
            ellipsis: true,
        },
        {
            title: '城市',
            dataIndex: 'city',
            key: 'city',
            width: 120,
        },
        {
            title: '国家',
            dataIndex: 'country',
            key: 'country',
            width: 120,
        },
        {
            title: '操作',
            key: 'action',
            width: 150,
            fixed: 'right',
            render: (_, record) => (
                <Space size="middle">
                    <Tooltip title="编辑">
                        <Button
                            type="text"
                            icon={<EditOutlined />}
                            onClick={() => handleEdit(record)}
                        />
                    </Tooltip>
                    <Popconfirm
                        title={`确定要删除该${activeTab === 'sender' ? '发货人' : '收货人'}吗？`}
                        onConfirm={() => handleDelete(record.id)}
                        okText="确定"
                        cancelText="取消"
                    >
                        <Tooltip title="删除">
                            <Button type="text" danger icon={<DeleteOutlined />} />
                        </Tooltip>
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    const items = [
        {
            key: 'sender',
            label: (
                <span>
                    <UserOutlined /> 发货人管理
                </span>
            ),
        },
        {
            key: 'receiver',
            label: (
                <span>
                    <TeamOutlined /> 收货人管理
                </span>
            ),
        },
    ];

    return (
        <div style={{ padding: 24 }}>
            <div style={{ marginBottom: 16 }}>
                <h2 style={{ fontSize: 24, fontWeight: 600, color: '#1f2937', marginBottom: 24 }}>
                    客户管理
                </h2>
            </div>

            <Card bordered={false} bodyStyle={{ padding: 0 }}>
                <Tabs
                    activeKey={activeTab}
                    onChange={(key) => setActiveTab(key as CustomerType)}
                    items={items}
                    tabBarStyle={{ padding: '0 24px' }}
                />

                <div style={{ padding: 24 }}>
                    <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
                        <Space>
                            <Input
                                placeholder="搜索姓名/手机号/公司"
                                prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
                                value={searchText}
                                onChange={e => setSearchText(e.target.value)}
                                style={{ width: 250 }}
                                allowClear
                            />
                            <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
                        </Space>
                        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                            新增{activeTab === 'sender' ? '发货人' : '收货人'}
                        </Button>
                    </div>

                    <Table
                        columns={columns}
                        dataSource={data}
                        rowKey="id"
                        loading={loading}
                        pagination={{
                            defaultPageSize: 10,
                            showSizeChanger: true,
                            showTotal: (total) => `共 ${total} 条`
                        }}
                    />
                </div>
            </Card>

            <Modal
                title={`${editingId ? '编辑' : '新增'}${activeTab === 'sender' ? '发货人' : '收货人'}`}
                open={modalVisible}
                onOk={handleSubmit}
                onCancel={() => {
                    setModalVisible(false);
                    form.resetFields();
                    setEditingId(null);
                }}
                destroyOnClose
            >
                <Form
                    form={form}
                    layout="vertical"
                >
                    <Form.Item
                        name="name"
                        label="姓名"
                        rules={[{ required: true, message: '请输入姓名' }]}
                    >
                        <Input placeholder="请输入姓名" />
                    </Form.Item>
                    <Form.Item
                        name="phone"
                        label="手机号"
                        rules={[
                            { required: true, message: '请输入手机号' },
                            { pattern: /^[0-9+\-\s]+$/, message: '请输入有效的电话号码' }
                        ]}
                        extra="同一机构下，同类型的客户手机号必须唯一"
                    >
                        <Input placeholder="请输入手机号" />
                    </Form.Item>
                    <Form.Item name="company" label="公司名称">
                        <Input placeholder="请输入公司名称（选填）" />
                    </Form.Item>
                    <Form.Item name="address" label="详细地址">
                        <AddressInput
                            placeholder="请输入详细地址"
                            onAddressSelect={(data: AddressData) => {
                                form.setFieldsValue({
                                    address: data.address,
                                    city: data.city || data.shortName,
                                    country: data.country || (data.isOversea ? '' : '中国'),
                                    latitude: data.lat,
                                    longitude: data.lng
                                });
                            }}
                        />
                    </Form.Item>
                    <Form.Item name="latitude" hidden>
                        <Input />
                    </Form.Item>
                    <Form.Item name="longitude" hidden>
                        <Input />
                    </Form.Item>
                    <Space style={{ display: 'flex' }} align="baseline">
                        <Form.Item name="city" label="城市">
                            <Input placeholder="例如：深圳" />
                        </Form.Item>
                        <Form.Item name="country" label="国家">
                            <Input placeholder="例如：中国" />
                        </Form.Item>
                    </Space>
                </Form>
            </Modal>
        </div>
    );
};

export default Customers;
