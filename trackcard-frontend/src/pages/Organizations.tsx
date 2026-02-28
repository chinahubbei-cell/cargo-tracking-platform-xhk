import React, { useEffect, useState, useCallback } from 'react';
import {
    Card, Tree, Button, Space, Input, Modal, Form, Select, message, Descriptions,
    Tag, Table, Empty, Badge, Dropdown, Spin, Row, Col, Avatar, InputNumber
} from 'antd';
import type { MenuProps, TreeDataNode } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
    PlusOutlined, SearchOutlined, DeleteOutlined, EditOutlined, SettingOutlined,
    TeamOutlined, ApartmentOutlined, UserOutlined,
    BankOutlined, BranchesOutlined, HomeOutlined, SwapOutlined, ExclamationCircleOutlined
} from '@ant-design/icons';
import api from '../api/client';
import type { Organization, OrganizationTreeNode, OrganizationType, User } from '../types';

// 组织类型配置
const orgTypeConfig: Record<OrganizationType, { label: string; color: string; icon: React.ReactNode }> = {
    group: { label: '集团', color: '#722ed1', icon: <BankOutlined /> },
    company: { label: '公司', color: '#1890ff', icon: <HomeOutlined /> },
    branch: { label: '分公司', color: '#13c2c2', icon: <BranchesOutlined /> },
    dept: { label: '部门', color: '#52c41a', icon: <ApartmentOutlined /> },
    team: { label: '团队', color: '#fa8c16', icon: <TeamOutlined /> },
};

interface OrgUser {
    id: number;
    user_id: string;
    organization_id: string;
    organization_name: string;
    is_primary: boolean;
    position: string;
    joined_at: string;
    user: User;
}

const Organizations: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [treeData, setTreeData] = useState<OrganizationTreeNode[]>([]);
    const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
    const [selectedOrg, setSelectedOrg] = useState<Organization | null>(null);
    const [orgUsers, setOrgUsers] = useState<OrgUser[]>([]);
    const [usersLoading, setUsersLoading] = useState(false);
    const [search, setSearch] = useState('');

    // 弹窗状态
    const [createModalVisible, setCreateModalVisible] = useState(false);
    const [editModalVisible, setEditModalVisible] = useState(false);
    const [moveModalVisible, setMoveModalVisible] = useState(false);
    const [addUserModalVisible, setAddUserModalVisible] = useState(false);
    const [allUsers, setAllUsers] = useState<User[]>([]);

    const [form] = Form.useForm();
    const [moveForm] = Form.useForm();
    const [addUserForm] = Form.useForm();

    // 加载组织树
    const loadOrganizations = useCallback(async () => {
        setLoading(true);
        try {
            const data = await api.getOrganizations({ tree: true, search });
            setTreeData(data || []);
            // 默认展开第一级
            if (data && data.length > 0) {
                setExpandedKeys(data.map((item: OrganizationTreeNode) => item.id));
            }
        } catch (error) {
            message.error('加载组织列表失败');
        } finally {
            setLoading(false);
        }
    }, [search]);

    // 加载组织用户
    const loadOrgUsers = useCallback(async (orgId: string) => {
        setUsersLoading(true);
        try {
            const data = await api.getOrganizationUsers(orgId, { include_sub: false });
            setOrgUsers(data || []);
        } catch (error) {
            message.error('加载用户列表失败');
        } finally {
            setUsersLoading(false);
        }
    }, []);

    // 加载所有用户（用于添加用户弹窗）
    const loadAllUsers = useCallback(async () => {
        try {
            const res = await api.getUsers({});
            if (res.data) {
                setAllUsers(res.data);
            }
        } catch (error) {
            console.error('加载用户列表失败', error);
        }
    }, []);

    useEffect(() => {
        loadOrganizations();
        loadAllUsers();
    }, [loadOrganizations, loadAllUsers]);

    // 选中组织时加载详情和用户
    const handleSelectOrg = async (selectedKeys: React.Key[]) => {
        if (selectedKeys.length === 0) {
            setSelectedOrg(null);
            setOrgUsers([]);
            return;
        }

        const orgId = selectedKeys[0] as string;
        try {
            const data = await api.getOrganization(orgId);
            setSelectedOrg(data);
            loadOrgUsers(orgId);
        } catch (error) {
            message.error('加载组织详情失败');
        }
    };

    // 转换为Tree组件需要的数据格式
    const convertToTreeData = (nodes: OrganizationTreeNode[]): TreeDataNode[] => {
        return nodes.map(node => {
            const typeInfo = orgTypeConfig[node.type as OrganizationType] || orgTypeConfig.dept;
            return {
                key: node.id,
                title: (
                    <Space>
                        <span style={{ color: typeInfo.color }}>{typeInfo.icon}</span>
                        <span>{node.name}</span>
                        <Tag color={typeInfo.color} style={{ marginLeft: 4, fontSize: 10 }}>
                            {typeInfo.label}
                        </Tag>
                        {node.user_count > 0 && (
                            <Badge count={node.user_count} style={{ backgroundColor: '#52c41a' }} size="small" />
                        )}
                    </Space>
                ),
                children: node.children ? convertToTreeData(node.children) : undefined,
            };
        });
    };

    // 创建组织
    const handleCreate = async (values: any) => {
        try {
            const data = {
                ...values,
                parent_id: selectedOrg?.id || undefined,
            };
            await api.createOrganization(data);
            message.success('创建成功');
            setCreateModalVisible(false);
            form.resetFields();
            loadOrganizations();
        } catch (error: any) {
            message.error(error.response?.data?.error || '创建失败');
        }
    };

    // 编辑组织
    const handleEdit = async (values: any) => {
        if (!selectedOrg) return;
        try {
            await api.updateOrganization(selectedOrg.id, values);
            message.success('更新成功');
            setEditModalVisible(false);
            loadOrganizations();
            // 刷新选中的组织
            const orgData = await api.getOrganization(selectedOrg.id);
            setSelectedOrg(orgData);
        } catch (error: any) {
            message.error(error.response?.data?.error || '更新失败');
        }
    };

    // 删除组织
    const handleDelete = () => {
        if (!selectedOrg) return;
        Modal.confirm({
            title: '确认删除',
            icon: <ExclamationCircleOutlined />,
            content: `确定要删除组织 "${selectedOrg.name}" 吗？删除后不可恢复。`,
            okText: '删除',
            okType: 'danger',
            cancelText: '取消',
            onOk: async () => {
                try {
                    await api.deleteOrganization(selectedOrg.id);
                    message.success('删除成功');
                    setSelectedOrg(null);
                    setOrgUsers([]);
                    loadOrganizations();
                } catch (error: any) {
                    message.error(error.response?.data?.error || '删除失败');
                }
            },
        });
    };

    // 移动组织
    const handleMove = async (values: any) => {
        if (!selectedOrg) return;
        try {
            await api.moveOrganization(selectedOrg.id, values);
            message.success('移动成功');
            setMoveModalVisible(false);
            moveForm.resetFields();
            loadOrganizations();
        } catch (error: any) {
            message.error(error.response?.data?.error || '移动失败');
        }
    };

    // 添加用户到组织
    const handleAddUser = async (values: any) => {
        if (!selectedOrg) return;
        try {
            await api.addUserToOrganization(selectedOrg.id, values);
            message.success('添加成功');
            setAddUserModalVisible(false);
            addUserForm.resetFields();
            loadOrgUsers(selectedOrg.id);
            loadOrganizations(); // 刷新用户数
        } catch (error: any) {
            message.error(error.response?.data?.error || '添加失败');
        }
    };

    // 移除用户
    const handleRemoveUser = (user: OrgUser) => {
        if (!selectedOrg) return;
        Modal.confirm({
            title: '确认移除',
            content: `确定要将用户 "${user.user.name}" 从组织中移除吗？`,
            okText: '移除',
            okType: 'danger',
            onOk: async () => {
                try {
                    await api.removeUserFromOrganization(selectedOrg.id, user.user_id);
                    message.success('移除成功');
                    loadOrgUsers(selectedOrg.id);
                    loadOrganizations();
                } catch (error: any) {
                    message.error(error.response?.data?.error || '移除失败');
                }
            },
        });
    };

    // 切换主部门
    const handleTogglePrimary = async (user: OrgUser) => {
        if (!selectedOrg) return;
        try {
            await api.updateUserOrganization(selectedOrg.id, user.user_id, {
                is_primary: !user.is_primary,
            });
            message.success(user.is_primary ? '已取消主部门' : '已设为主部门');
            loadOrgUsers(selectedOrg.id);
        } catch (error: any) {
            message.error(error.response?.data?.error || '操作失败');
        }
    };

    // 用户表格列配置
    const userColumns: ColumnsType<OrgUser> = [
        {
            title: '用户',
            key: 'user',
            render: (_, record) => (
                <Space>
                    <Avatar icon={<UserOutlined />} size="small" />
                    <span>{record.user?.name || '-'}</span>
                </Space>
            ),
        },
        {
            title: '邮箱',
            key: 'email',
            render: (_, record) => record.user?.email || '-',
        },
        {
            title: '职位',
            dataIndex: 'position',
            key: 'position',
            render: (text) => text || '-',
        },
        {
            title: '主部门',
            dataIndex: 'is_primary',
            key: 'is_primary',
            width: 80,
            render: (isPrimary: boolean) => (
                isPrimary ? <Tag color="blue">主</Tag> : <Tag>兼</Tag>
            ),
        },
        {
            title: '加入时间',
            dataIndex: 'joined_at',
            key: 'joined_at',
            width: 160,
            render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            render: (_, record) => {
                const menuItems: MenuProps['items'] = [
                    {
                        key: 'toggle',
                        label: record.is_primary ? '取消主部门' : '设为主部门',
                        onClick: () => handleTogglePrimary(record),
                    },
                    { type: 'divider' },
                    {
                        key: 'remove',
                        label: '移除用户',
                        danger: true,
                        onClick: () => handleRemoveUser(record),
                    },
                ];
                return (
                    <Dropdown menu={{ items: menuItems }} trigger={['click']}>
                        <Button type="text" icon={<SettingOutlined />} />
                    </Dropdown>
                );
            },
        },
    ];

    // 获取可选的父组织（用于移动弹窗）
    const getParentOptions = (nodes: OrganizationTreeNode[], currentId: string, level: number = 0): { value: string; label: string; disabled: boolean }[] => {
        const options: { value: string; label: string; disabled: boolean }[] = [];
        for (const node of nodes) {
            // 不能选择自己或自己的子节点
            const isDisabled = node.id === currentId || node.level >= 2; // 3级限制
            options.push({
                value: node.id,
                label: '　'.repeat(level) + node.name,
                disabled: isDisabled,
            });
            if (node.children) {
                options.push(...getParentOptions(node.children, currentId, level + 1));
            }
        }
        return options;
    };

    return (
        <div style={{ display: 'flex', height: 'calc(100vh - 120px)', gap: 16 }}>
            {/* 左侧组织树 */}
            <Card
                title={
                    <Space>
                        <ApartmentOutlined />
                        <span>组织架构</span>
                    </Space>
                }
                style={{ width: 360, flexShrink: 0 }}
                bodyStyle={{ padding: '12px 0', height: 'calc(100% - 57px)', overflow: 'auto' }}
                extra={
                    <Button
                        type="primary"
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={() => {
                            form.resetFields();
                            setCreateModalVisible(true);
                        }}
                    >
                        新建
                    </Button>
                }
            >
                <div style={{ padding: '0 12px 12px' }}>
                    <Input
                        placeholder="搜索组织..."
                        prefix={<SearchOutlined />}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        onPressEnter={() => loadOrganizations()}
                        allowClear
                    />
                </div>
                <Spin spinning={loading}>
                    {treeData.length > 0 ? (
                        <Tree
                            treeData={convertToTreeData(treeData)}
                            expandedKeys={expandedKeys}
                            onExpand={(keys) => setExpandedKeys(keys)}
                            onSelect={handleSelectOrg}
                            selectedKeys={selectedOrg ? [selectedOrg.id] : []}
                            blockNode
                            showLine={{ showLeafIcon: false }}
                            style={{ padding: '0 12px' }}
                            draggable
                            onDrop={async (info) => {
                                const dragNodeId = info.dragNode.key as string;
                                const dropNodeId = info.node.key as string;

                                // 简化的拖拽：移动到目标节点下
                                if (info.dropToGap) {
                                    // 移动到同级
                                    message.info('拖拽排序功能开发中');
                                } else {
                                    // 移动到子节点
                                    try {
                                        await api.moveOrganization(dragNodeId, { parent_id: dropNodeId });
                                        message.success('移动成功');
                                        loadOrganizations();
                                    } catch (error: any) {
                                        message.error(error.response?.data?.error || '移动失败');
                                    }
                                }
                            }}
                        />
                    ) : (
                        <Empty description="暂无组织" style={{ marginTop: 40 }} />
                    )}
                </Spin>
            </Card>

            {/* 右侧详情区域 */}
            <Card
                title={
                    selectedOrg ? (
                        <Space>
                            {orgTypeConfig[selectedOrg.type as OrganizationType]?.icon}
                            <span>{selectedOrg.name}</span>
                            <Tag color={orgTypeConfig[selectedOrg.type as OrganizationType]?.color}>
                                {orgTypeConfig[selectedOrg.type as OrganizationType]?.label}
                            </Tag>
                        </Space>
                    ) : (
                        '组织详情'
                    )
                }
                style={{ flex: 1 }}
                bodyStyle={{ height: 'calc(100% - 57px)', overflow: 'auto' }}
                extra={
                    selectedOrg && (
                        <Space>
                            <Button icon={<EditOutlined />} onClick={() => {
                                form.setFieldsValue(selectedOrg);
                                setEditModalVisible(true);
                            }}>编辑</Button>
                            <Button icon={<SwapOutlined />} onClick={() => {
                                moveForm.resetFields();
                                setMoveModalVisible(true);
                            }}>移动</Button>
                            <Button danger icon={<DeleteOutlined />} onClick={handleDelete}>
                                删除
                            </Button>
                        </Space>
                    )
                }
            >
                {selectedOrg ? (
                    <div>
                        {/* 基本信息 */}
                        <Descriptions title="基本信息" bordered size="small" column={2} style={{ marginBottom: 24 }}>
                            <Descriptions.Item label="组织ID">{selectedOrg.id}</Descriptions.Item>
                            <Descriptions.Item label="组织编码">{selectedOrg.code}</Descriptions.Item>
                            <Descriptions.Item label="组织名称">{selectedOrg.name}</Descriptions.Item>
                            <Descriptions.Item label="组织类型">
                                <Tag color={orgTypeConfig[selectedOrg.type as OrganizationType]?.color}>
                                    {orgTypeConfig[selectedOrg.type as OrganizationType]?.label}
                                </Tag>
                            </Descriptions.Item>
                            <Descriptions.Item label="层级">第 {selectedOrg.level} 级</Descriptions.Item>
                            <Descriptions.Item label="状态">
                                <Tag color={selectedOrg.status === 'active' ? 'green' : 'default'}>
                                    {selectedOrg.status === 'active' ? '正常' : '已禁用'}
                                </Tag>
                            </Descriptions.Item>
                            <Descriptions.Item label="负责人">
                                {selectedOrg.leader_name || '-'}
                            </Descriptions.Item>
                            <Descriptions.Item label="排序号">{selectedOrg.sort}</Descriptions.Item>
                            <Descriptions.Item label="创建时间" span={2}>
                                {selectedOrg.created_at ? new Date(selectedOrg.created_at).toLocaleString('zh-CN') : '-'}
                            </Descriptions.Item>
                            {selectedOrg.description && (
                                <Descriptions.Item label="描述" span={2}>
                                    {selectedOrg.description}
                                </Descriptions.Item>
                            )}
                        </Descriptions>

                        {/* 用户列表 */}
                        <Card
                            title={
                                <Space>
                                    <TeamOutlined />
                                    <span>组织成员</span>
                                    <Badge count={orgUsers.length} style={{ backgroundColor: '#52c41a' }} />
                                </Space>
                            }
                            size="small"
                            extra={
                                <Button
                                    type="primary"
                                    size="small"
                                    icon={<PlusOutlined />}
                                    onClick={() => {
                                        addUserForm.resetFields();
                                        setAddUserModalVisible(true);
                                    }}
                                >
                                    添加成员
                                </Button>
                            }
                        >
                            <Table
                                columns={userColumns}
                                dataSource={orgUsers}
                                rowKey="id"
                                loading={usersLoading}
                                pagination={false}
                                size="small"
                                locale={{ emptyText: <Empty description="暂无成员" /> }}
                            />
                        </Card>

                        {/* 子组织列表 */}
                        {selectedOrg.children && selectedOrg.children.length > 0 && (
                            <Card
                                title={
                                    <Space>
                                        <BranchesOutlined />
                                        <span>子组织</span>
                                        <Badge count={selectedOrg.children.length} style={{ backgroundColor: '#1890ff' }} />
                                    </Space>
                                }
                                size="small"
                                style={{ marginTop: 16 }}
                            >
                                <Row gutter={[16, 16]}>
                                    {selectedOrg.children.map(child => (
                                        <Col span={8} key={child.id}>
                                            <Card
                                                size="small"
                                                hoverable
                                                onClick={() => handleSelectOrg([child.id])}
                                            >
                                                <Space>
                                                    {orgTypeConfig[(child as any).type as OrganizationType]?.icon}
                                                    <span>{child.name}</span>
                                                </Space>
                                            </Card>
                                        </Col>
                                    ))}
                                </Row>
                            </Card>
                        )}
                    </div>
                ) : (
                    <Empty description="请选择左侧组织查看详情" style={{ marginTop: 100 }} />
                )}
            </Card>

            {/* 创建组织弹窗 */}
            <Modal
                title={selectedOrg ? `在 "${selectedOrg.name}" 下创建子组织` : '创建根组织'}
                open={createModalVisible}
                onCancel={() => setCreateModalVisible(false)}
                onOk={() => form.submit()}
            >
                <Form form={form} layout="vertical" onFinish={handleCreate}>
                    <Form.Item name="name" label="组织名称" rules={[{ required: true, message: '请输入组织名称' }]}>
                        <Input placeholder="如：华东分公司" />
                    </Form.Item>
                    <Form.Item name="code" label="组织编码" rules={[{ required: true, message: '请输入组织编码' }]}>
                        <Input placeholder="如：EAST" />
                    </Form.Item>
                    <Form.Item name="type" label="组织类型">
                        <Select placeholder="选择类型（可选，将自动根据层级设置）">
                            {Object.entries(orgTypeConfig).map(([key, config]) => (
                                <Select.Option key={key} value={key}>
                                    <Space>
                                        {config.icon}
                                        <span>{config.label}</span>
                                    </Space>
                                </Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                    <Form.Item name="leader_id" label="负责人">
                        <Select placeholder="选择负责人（可选）" allowClear showSearch optionFilterProp="label">
                            {allUsers.map(user => (
                                <Select.Option key={user.id} value={user.id} label={user.name}>
                                    {user.name} ({user.email})
                                </Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                    <Form.Item name="sort" label="排序号">
                        <InputNumber min={0} placeholder="数字越小越靠前" style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="description" label="描述">
                        <Input.TextArea rows={2} placeholder="可选" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 编辑组织弹窗 */}
            <Modal
                title="编辑组织"
                open={editModalVisible}
                onCancel={() => setEditModalVisible(false)}
                onOk={() => form.submit()}
            >
                <Form form={form} layout="vertical" onFinish={handleEdit}>
                    <Form.Item name="name" label="组织名称" rules={[{ required: true }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name="code" label="组织编码" rules={[{ required: true }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name="type" label="组织类型">
                        <Select>
                            {Object.entries(orgTypeConfig).map(([key, config]) => (
                                <Select.Option key={key} value={key}>
                                    <Space>
                                        {config.icon}
                                        <span>{config.label}</span>
                                    </Space>
                                </Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                    <Form.Item name="status" label="状态">
                        <Select>
                            <Select.Option value="active">正常</Select.Option>
                            <Select.Option value="disabled">已禁用</Select.Option>
                        </Select>
                    </Form.Item>
                    <Form.Item name="leader_id" label="负责人">
                        <Select allowClear showSearch optionFilterProp="label">
                            {allUsers.map(user => (
                                <Select.Option key={user.id} value={user.id} label={user.name}>
                                    {user.name} ({user.email})
                                </Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                    <Form.Item name="sort" label="排序号">
                        <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="description" label="描述">
                        <Input.TextArea rows={2} />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 移动组织弹窗 */}
            <Modal
                title="移动组织"
                open={moveModalVisible}
                onCancel={() => setMoveModalVisible(false)}
                onOk={() => moveForm.submit()}
            >
                <Form form={moveForm} layout="vertical" onFinish={handleMove}>
                    <Form.Item name="parent_id" label="移动到">
                        <Select placeholder="选择目标位置（留空则移动到根级）" allowClear>
                            {selectedOrg && getParentOptions(treeData, selectedOrg.id).map(opt => (
                                <Select.Option key={opt.value} value={opt.value} disabled={opt.disabled}>
                                    {opt.label}
                                </Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                    <Form.Item name="sort" label="新排序号">
                        <Input type="number" placeholder="可选，数字越小越靠前" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 添加用户弹窗 */}
            <Modal
                title={`添加成员到 "${selectedOrg?.name}"`}
                open={addUserModalVisible}
                onCancel={() => setAddUserModalVisible(false)}
                onOk={() => addUserForm.submit()}
            >
                <Form form={addUserForm} layout="vertical" onFinish={handleAddUser}>
                    <Form.Item name="user_id" label="选择用户" rules={[{ required: true, message: '请选择用户' }]}>
                        <Select
                            placeholder="搜索并选择用户"
                            showSearch
                            optionFilterProp="label"
                        >
                            {allUsers.filter(u => !orgUsers.some(ou => ou.user_id === u.id)).map(user => (
                                <Select.Option key={user.id} value={user.id} label={user.name}>
                                    <Space>
                                        <Avatar icon={<UserOutlined />} size="small" />
                                        <span>{user.name}</span>
                                        <span style={{ color: '#999' }}>({user.email})</span>
                                    </Space>
                                </Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                    <Form.Item name="position" label="职位">
                        <Input placeholder="如：经理、工程师" />
                    </Form.Item>
                    <Form.Item name="is_primary" label="设为主部门" valuePropName="checked">
                        <Select defaultValue={false}>
                            <Select.Option value={true}>是（主部门）</Select.Option>
                            <Select.Option value={false}>否（兼职部门）</Select.Option>
                        </Select>
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    );
};

export default Organizations;
