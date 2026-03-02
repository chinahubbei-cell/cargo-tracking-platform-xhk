import React, { useEffect, useState } from 'react';
import { Table, Card, Tag, Button, Space, Input, Modal, Form, Select, message, Dropdown, Descriptions, Tooltip, Result } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, SearchOutlined, DeleteOutlined, EditOutlined, SettingOutlined, EyeOutlined, KeyOutlined, StopOutlined, ApartmentOutlined, LockOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import api from '../api/client';
import type { User } from '../types';
import { useAuthStore } from '../store/authStore';

const Users: React.FC = () => {
    const { user: currentUser } = useAuthStore();
    const [loading, setLoading] = useState(false);
    const [users, setUsers] = useState<User[]>([]);
    const [search, setSearch] = useState('');
    const [modalVisible, setModalVisible] = useState(false);
    const [editingUser, setEditingUser] = useState<User | null>(null);
    const [detailModalVisible, setDetailModalVisible] = useState(false);
    const [viewingUser, setViewingUser] = useState<User | null>(null);
    const [form] = Form.useForm();

    // 权限检查：非管理员无权访问
    const isAdmin = currentUser?.role === 'admin';

    // 如果不是管理员，显示无权限提示
    if (!isAdmin) {
        return (
            <Card>
                <Result
                    status="403"
                    title="无操作权限"
                    subTitle="用户管理仅限管理员访问，如需管理用户，请联系管理员。"
                    icon={<LockOutlined style={{ color: '#faad14' }} />}
                />
            </Card>
        );
    }

    useEffect(() => {
        loadUsers();
    }, []);

    const loadUsers = async () => {
        setLoading(true);
        try {
            const res = await api.getUsers({ search });
            if (res.data) {
                setUsers(res.data);
            }
        } catch (error: any) {
            if (error.response?.status === 403) {
                message.error('无操作权限，仅管理员可访问用户管理');
            } else {
                message.error('加载用户列表失败');
            }
        } finally {
            setLoading(false);
        }
    };

    const handleCreate = () => {
        setEditingUser(null);
        form.resetFields();
        setModalVisible(true);
    };

    const handleEdit = (user: User) => {
        setEditingUser(user);
        form.setFieldsValue(user);
        setModalVisible(true);
    };

    const handleViewDetail = (user: User) => {
        setViewingUser(user);
        setDetailModalVisible(true);
    };

    const handleDelete = async (id: string) => {
        try {
            await api.deleteUser(id);
            message.success('删除成功');
            loadUsers();
        } catch (error: any) {
            message.error(error.response?.data?.error || '删除失败');
        }
    };

    const handleResetPassword = async (user: User) => {
        Modal.confirm({
            title: '重置密码',
            content: `确定要重置用户 ${user.name} 的密码吗？新密码将被设置为: 123456`,
            okText: '确定重置',
            cancelText: '取消',
            onOk: async () => {
                try {
                    await api.resetUserPassword(user.id, '123456');
                    message.success('密码已重置为: 123456');
                } catch (error: any) {
                    message.error(error.response?.data?.error || '重置密码失败');
                }
            },
        });
    };

    const handleToggleStatus = async (user: User) => {
        const newStatus = user.status === 'active' ? 'disabled' : 'active';
        const actionText = newStatus === 'active' ? '启用' : '禁用';
        try {
            await api.updateUser(user.id, { status: newStatus } as any);
            message.success(`${actionText}成功`);
            loadUsers();
        } catch (error: any) {
            message.error(error.response?.data?.error || `${actionText}失败`);
        }
    };

    const handleSubmit = async (values: any) => {
        try {
            if (editingUser) {
                await api.updateUser(editingUser.id, values);
                message.success('更新成功');
            } else {
                await api.createUser(values);
                message.success('创建成功');
            }
            setModalVisible(false);
            loadUsers();
        } catch (error: any) {
            message.error(error.response?.data?.error || '操作失败');
        }
    };

    const roleLabels: Record<string, string> = {
        admin: '管理员',
        operator: '操作员',
        viewer: '查看者',
    };


    const columns: ColumnsType<User> = [
        {
            title: '序号',
            key: 'index',
            width: 60,
            fixed: 'left',
            align: 'center' as const,
            render: (_, __, index) => index + 1,
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
                    {
                        key: 'edit',
                        icon: <EditOutlined />,
                        label: '编辑用户',
                        onClick: () => handleEdit(record),
                    },
                    {
                        key: 'resetPassword',
                        icon: <KeyOutlined />,
                        label: '重置密码',
                        onClick: () => handleResetPassword(record),
                    },
                    { type: 'divider' },
                    {
                        key: 'disable',
                        icon: <StopOutlined />,
                        label: record.status === 'active' ? '禁用用户' : '启用用户',
                        onClick: () => handleToggleStatus(record),
                    },
                    {
                        key: 'delete',
                        icon: <DeleteOutlined />,
                        label: '删除用户',
                        danger: true,
                        onClick: () => {
                            Modal.confirm({
                                title: '确认删除',
                                content: `确定要删除用户 ${record.name} 吗？`,
                                okText: '删除',
                                okType: 'danger',
                                onOk: () => handleDelete(record.id),
                            });
                        },
                    },
                ];

                return (
                    <Dropdown
                        menu={{ items: menuItems }}
                        trigger={['hover']}
                        placement="bottomRight"
                    >
                        <Button
                            type="text"
                            icon={<SettingOutlined style={{ fontSize: 16, color: '#6b7280' }} />}
                            style={{ padding: 4 }}
                        />
                    </Dropdown>
                );
            },
        },
        {
            title: '用户ID',
            dataIndex: 'id',
            key: 'id',
            width: 120,
        },
        {
            title: '姓名',
            dataIndex: 'name',
            key: 'name',
        },
        {
            title: '邮箱',
            dataIndex: 'email',
            key: 'email',
        },
        {
            title: '手机号',
            key: 'phone_number',
            width: 130,
            render: (_, record) => (record.phone_number ? `${record.phone_country_code || '+86'} ${record.phone_number}` : '-'),
        },
        {
            title: '角色',
            dataIndex: 'role',
            key: 'role',
            width: 100,
            render: (role: string) => {
                const colors: Record<string, string> = {
                    admin: 'red',
                    operator: 'blue',
                    viewer: 'default',
                };
                return <Tag color={colors[role]}>{roleLabels[role] || role}</Tag>;
            },
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 80,
            render: (status: string) => (
                <Tag color={status === 'active' ? 'green' : 'default'}>
                    {status === 'active' ? '正常' : '已禁用'}
                </Tag>
            ),
        },
        {
            title: '组织机构',
            dataIndex: 'primary_org_name',
            key: 'primary_org_name',
            width: 150,
            render: (orgName: string, record: User) => {
                if (!record.organizations || record.organizations.length === 0) {
                    return <span style={{ color: '#999' }}>未分配</span>;
                }
                const count = record.organizations.length;
                if (count === 1) {
                    return <Tag icon={<ApartmentOutlined />}>{record.organizations[0].name}</Tag>;
                }
                return (
                    <Tooltip title={record.organizations.map(o => `${o.name}${o.is_primary ? '(主)' : ''}`).join(', ')}>
                        <Tag icon={<ApartmentOutlined />}>
                            {orgName || record.organizations[0].name}
                            <span style={{ marginLeft: 4, fontSize: 11, color: '#666' }}>+{count - 1}</span>
                        </Tag>
                    </Tooltip>
                );
            },
        },
        {
            title: '最后登录',
            dataIndex: 'last_login',
            key: 'last_login',
            width: 160,
            render: (time: string | undefined) => (time ? new Date(time).toLocaleString('zh-CN') : '-'),
        },
    ];

    return (
        <Card
            title="用户管理"
            headStyle={{ fontSize: 16, fontWeight: 600 }}
            extra={
                <Space>
                    <Input
                        placeholder="搜索用户"
                        prefix={<SearchOutlined />}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        onPressEnter={() => loadUsers()}
                        style={{ width: 200 }}
                    />
                    <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
                        添加用户
                    </Button>
                </Space>
            }
        >
            <Table
                columns={columns}
                dataSource={users}
                rowKey="id"
                loading={loading}
                pagination={{ pageSize: 10, showTotal: (total) => `共 ${total} 条` }}
            />

            <Modal
                title={editingUser ? '编辑用户' : '添加用户'}
                open={modalVisible}
                onCancel={() => setModalVisible(false)}
                onOk={() => form.submit()}
            >
                <Form form={form} layout="vertical" onFinish={handleSubmit}>
                    <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item
                        name="email"
                        label="邮箱"
                        rules={[
                            { required: true, message: '请输入邮箱' },
                            { type: 'email', message: '请输入有效邮箱' },
                        ]}
                    >
                        <Input disabled={!!editingUser} />
                    </Form.Item>
                    {!editingUser && (
                        <Form.Item
                            name="password"
                            label="密码"
                            rules={[
                                { required: true, message: '请输入密码' },
                                { min: 6, message: '密码至少6位' },
                            ]}
                        >
                            <Input.Password />
                        </Form.Item>
                    )}
                    <Form.Item name="role" label="角色">
                        <Select>
                            <Select.Option value="admin">管理员</Select.Option>
                            <Select.Option value="operator">操作员</Select.Option>
                            <Select.Option value="viewer">查看者</Select.Option>
                        </Select>
                    </Form.Item>
                    {editingUser && (
                        <Form.Item name="status" label="状态">
                            <Select>
                                <Select.Option value="active">正常</Select.Option>
                                <Select.Option value="disabled">已禁用</Select.Option>
                            </Select>
                        </Form.Item>
                    )}
                </Form>
            </Modal>

            {/* 用户详情弹窗 */}
            <Modal
                title={<span><EyeOutlined style={{ marginRight: 8 }} />用户详情</span>}
                open={detailModalVisible}
                onCancel={() => setDetailModalVisible(false)}
                footer={[
                    <Button key="close" onClick={() => setDetailModalVisible(false)}>关闭</Button>,
                    <Button key="edit" type="primary" onClick={() => {
                        setDetailModalVisible(false);
                        if (viewingUser) handleEdit(viewingUser);
                    }}>编辑用户</Button>
                ]}
                width={500}
            >
                {viewingUser && (
                    <Descriptions column={1} bordered size="small">
                        <Descriptions.Item label="用户ID">{viewingUser.id}</Descriptions.Item>
                        <Descriptions.Item label="姓名">{viewingUser.name}</Descriptions.Item>
                        <Descriptions.Item label="邮箱">{viewingUser.email}</Descriptions.Item>
                        <Descriptions.Item label="手机号">{viewingUser.phone_number ? `${viewingUser.phone_country_code || '+86'} ${viewingUser.phone_number}` : '-'}</Descriptions.Item>
                        <Descriptions.Item label="角色">
                            <Tag color={viewingUser.role === 'admin' ? 'red' : viewingUser.role === 'operator' ? 'blue' : 'default'}>
                                {roleLabels[viewingUser.role] || viewingUser.role}
                            </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="状态">
                            <Tag color={viewingUser.status === 'active' ? 'green' : 'default'}>
                                {viewingUser.status === 'active' ? '正常' : '已禁用'}
                            </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="组织机构">
                            {viewingUser.organizations && viewingUser.organizations.length > 0 ? (
                                <Space wrap>
                                    {viewingUser.organizations.map((org, index) => (
                                        <Tag key={index} icon={<ApartmentOutlined />} color={org.is_primary ? 'blue' : 'default'}>
                                            {org.name}{org.is_primary && '(主)'}
                                            {org.position && <span style={{ marginLeft: 4, fontSize: 11 }}>- {org.position}</span>}
                                        </Tag>
                                    ))}
                                </Space>
                            ) : (
                                <span style={{ color: '#999' }}>未分配组织</span>
                            )}
                        </Descriptions.Item>
                        <Descriptions.Item label="创建时间">
                            {viewingUser.created_at ? new Date(viewingUser.created_at).toLocaleString('zh-CN') : '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label="最后登录">
                            {viewingUser.last_login ? new Date(viewingUser.last_login).toLocaleString('zh-CN') : '从未登录'}
                        </Descriptions.Item>
                    </Descriptions>
                )}
            </Modal>
        </Card>
    );
};

export default Users;
