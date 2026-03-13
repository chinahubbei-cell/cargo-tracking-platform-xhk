import React, { useState, useEffect } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Avatar, Dropdown, Modal, Descriptions, Tag, Button, Space, message } from 'antd';
import type { MenuProps } from 'antd';
import {
    DashboardOutlined,
    ShoppingOutlined,
    GlobalOutlined,
    MobileOutlined,
    TeamOutlined,
    DollarOutlined,
    SettingOutlined,
    LogoutOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    ClockCircleOutlined,
    CarOutlined,
    AimOutlined,
    NodeIndexOutlined,
    ApartmentOutlined,
    UserOutlined,
    AlertOutlined,
    CheckOutlined,
} from '@ant-design/icons';

// 自定义图标：港口/码头 (船锚 - 描边风格，匹配 Ant Design Outlined)
const PortIcon = () => (
    <span role="img" aria-label="port" className="anticon">
        <svg viewBox="0 0 1024 1024" width="1em" height="1em" fill="none" stroke="currentColor" strokeWidth="64" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="512" cy="160" r="80" />
            <line x1="512" y1="240" x2="512" y2="700" />
            <line x1="320" y1="400" x2="704" y2="400" />
            <path d="M320 700 Q320 860 512 860 Q704 860 704 700" />
        </svg>
    </span>
);

// 自定义图标：机场 (飞机 - 描边风格，匹配 Ant Design Outlined)
const AirportIcon = () => (
    <span role="img" aria-label="airport" className="anticon">
        <svg viewBox="0 0 1024 1024" width="1em" height="1em" fill="none" stroke="currentColor" strokeWidth="64" strokeLinecap="round" strokeLinejoin="round">
            <path d="M512 128 L512 400" />
            <path d="M192 480 L512 400 L832 480" />
            <path d="M512 400 L512 750" />
            <path d="M352 680 L512 750 L672 680" />
            <circle cx="512" cy="850" r="50" />
        </svg>
    </span>
);
import { useAuthStore } from '../../store/authStore';
import api from '../../api/client';

const { Header, Content, Sider } = Layout;

const menuItems: MenuProps['items'] = [
    // 1. 数据概览（独立）
    {
        key: '/dashboard',
        icon: <DashboardOutlined />,
        label: '数据概览',
    },

    // 2. 货运业务（包含预警中心）
    {
        key: '/business',
        icon: <ShoppingOutlined />,
        label: '货运业务',
        children: [
            {
                key: '/business/shipments',
                icon: <CarOutlined />,
                label: '运单管理',
            },
            {
                key: '/business/tracking',
                icon: <AimOutlined />,
                label: '货物追踪',
            },
            {
                key: '/business/route-planning',
                icon: <NodeIndexOutlined />,
                label: '线路规划',
            },
            {
                key: '/business/alerts',
                icon: <AlertOutlined />,
                label: '预警中心',
            },
        ],
    },

    // 3. 基础数据
    {
        key: '/resources',
        icon: <GlobalOutlined />,
        label: '基础数据',
        children: [
            {
                key: '/resources/ports',
                icon: <PortIcon />,
                label: '全球港口',
            },
            {
                key: '/resources/airports',
                icon: <AirportIcon />,
                label: '全球机场',
            },
            {
                key: '/resources/partners',
                icon: <TeamOutlined />,
                label: '合作伙伴',
            },
            {
                key: '/resources/customers',
                icon: <TeamOutlined />,
                label: '客户管理',
            },
        ],
    },

    // 4. 设备管理
    {
        key: '/devices',
        icon: <MobileOutlined />,
        label: '设备管理',
        children: [
            {
                key: '/devices/list',
                icon: <MobileOutlined />,
                label: '设备管理',
            },
        ],
    },

    // 5. 组织管理
    {
        key: '/organization',
        icon: <ApartmentOutlined />,
        label: '组织管理',
        children: [
            {
                key: '/organization/users',
                icon: <UserOutlined />,
                label: '用户管理',
            },
            {
                key: '/organization/departments',
                icon: <ApartmentOutlined />,
                label: '组织机构',
            },
        ],
    },

    // 6. 商务管理
    {
        key: '/business-mgmt',
        icon: <DollarOutlined />,
        label: '商务管理',
        children: [
            {
                key: '/business-mgmt/rates',
                icon: <DollarOutlined />,
                label: '运价管理',
            },
        ],
    },


    // 7. 系统设置
    {
        key: '/settings',
        icon: <SettingOutlined />,
        label: '系统设置',
        children: [
            {
                key: '/settings/profile',
                icon: <SettingOutlined />,
                label: '个人设置',
            },
            {
                key: '/settings/system',
                icon: <SettingOutlined />,
                label: '系统设置',
            },
        ],
    },
];

const MainLayout: React.FC = () => {
    const [collapsed, setCollapsed] = useState(false);
    const [beijingTime, setBeijingTime] = useState('');
    const [profileModalVisible, setProfileModalVisible] = useState(false);
    const navigate = useNavigate();
    const location = useLocation();
    const { user, logout, currentOrg, currentOrgId, setCurrentOrg } = useAuthStore();


    const syncCurrentUser = async () => {
        try {
            const res = await api.getCurrentUser();
            if (res.data) {
                useAuthStore.setState((state) => ({
                    ...state,
                    user: {
                        ...(state.user || {} as any),
                        ...res.data,
                    } as any,
                }));
                localStorage.setItem('user', JSON.stringify({
                    ...(user || {}),
                    ...res.data,
                }));
            }
        } catch {
            // 静默失败，避免打扰用户
        }
    };

    // 默认展开的菜单项（货运业务）
    const defaultOpenKeys = ['/business'];

    // 北京时间更新
    useEffect(() => {
        const updateTime = () => {
            const now = new Date();
            const beijingOffset = 8 * 60; // UTC+8
            const localOffset = now.getTimezoneOffset();
            const beijingDate = new Date(now.getTime() + (beijingOffset + localOffset) * 60000);

            // 格式: 2026-1-14 23:58:59
            const year = beijingDate.getFullYear();
            const month = beijingDate.getMonth() + 1;
            const day = beijingDate.getDate();
            const hours = String(beijingDate.getHours()).padStart(2, '0');
            const minutes = String(beijingDate.getMinutes()).padStart(2, '0');
            const seconds = String(beijingDate.getSeconds()).padStart(2, '0');

            setBeijingTime(`${year}-${month}-${day} ${hours}:${minutes}:${seconds}`);
        };

        updateTime();
        const timer = setInterval(updateTime, 1000);
        return () => clearInterval(timer);
    }, []);

    useEffect(() => {
        syncCurrentUser();
    }, []);

    const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
        // 父菜单不跳转（如 /business, /resources 等）
        if (!key.includes('/') || key === '/business' || key === '/resources' ||
            key === '/devices' || key === '/organization' ||
            key === '/business-mgmt' || key === '/settings') {
            return;
        }
        navigate(key);
    };

    // 构建用户菜单（包含组织切换）
    const userMenuItems: MenuProps['items'] = [
        // 当前组织信息显示在菜单顶部
        ...(currentOrg ? [
            {
                key: 'current-org',
                icon: <ApartmentOutlined style={{ color: '#1890ff' }} />,
                label: (
                    <span style={{ fontWeight: 500 }}>
                        {currentOrg.name}
                        {user?.organizations && user.organizations.length > 1 && (
                            <span style={{ marginLeft: 6, fontSize: 11, color: '#999' }}>▼</span>
                        )}
                    </span>
                ),
                // 如果有多个组织，点击可切换
                ...(user?.organizations && user.organizations.length > 1 ? {
                    children: user.organizations.map(org => ({
                        key: `org-${org.id}`,
                        icon: currentOrgId === org.id ? <CheckOutlined style={{ color: '#1890ff' }} /> : <ApartmentOutlined />,
                        label: (
                            <span style={{ color: currentOrgId === org.id ? '#1890ff' : undefined }}>
                                {org.name}{org.is_primary ? ' (主)' : ''}
                            </span>
                        ),
                    })),
                } : {}),
            },
            {
                type: 'divider' as const,
            },
        ] : []),
        {
            key: 'profile',
            icon: <UserOutlined />,
            label: '个人中心',
        },
        {
            type: 'divider',
        },
        {
            key: 'logout',
            icon: <LogoutOutlined />,
            label: '退出登录',
            danger: true,
        },
    ];

    const handleUserMenuClick: MenuProps['onClick'] = ({ key }) => {
        if (key === 'logout') {
            logout();
            navigate('/login');
        } else if (key === 'profile') {
            syncCurrentUser();
            setProfileModalVisible(true);
        } else if (key.startsWith('org-')) {
            const orgId = key.replace('org-', '');
            if (orgId !== currentOrgId) {
                setCurrentOrg(orgId);
                const org = user?.organizations?.find(o => o.id === orgId);
                message.success(`已切换到：${org?.name || orgId}`);
                // 刷新页面以应用新的组织数据
                window.location.reload();
            }
        }
    };

    return (
        <Layout style={{ minHeight: '100vh' }}>
            <Sider
                trigger={null}
                collapsible
                collapsed={collapsed}
                width={240}
                style={{
                    background: '#ffffff',
                    borderRight: '1px solid #e5e7eb',
                }}
            >
                {/* Logo区域 */}
                <div
                    style={{
                        height: 64,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: collapsed ? 'center' : 'flex-start',
                        padding: collapsed ? 0 : '0 20px',
                        borderBottom: '1px solid #e5e7eb',
                    }}
                >
                    <div style={{
                        width: 36,
                        height: 36,
                        borderRadius: 10,
                        background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        color: 'white',
                        fontSize: 18,
                        fontWeight: 'bold',
                        flexShrink: 0,
                    }}>
                        T
                    </div>
                    {!collapsed && (
                        <span style={{
                            marginLeft: 12,
                            fontSize: 16,
                            fontWeight: 600,
                            color: '#111827',
                        }}>
                            TrackCard
                        </span>
                    )}
                </div>

                {/* 导航菜单 */}
                <Menu
                    mode="inline"
                    selectedKeys={[location.pathname]}
                    defaultOpenKeys={defaultOpenKeys}
                    items={menuItems}
                    onClick={handleMenuClick}
                    style={{
                        border: 'none',
                        padding: '12px 8px',
                    }}
                />
            </Sider>

            <Layout style={{ background: '#f9fafb' }}>
                {/* 顶部导航栏 */}
                <Header
                    style={{
                        padding: '0 24px',
                        background: '#ffffff',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        borderBottom: '1px solid #e5e7eb',
                        height: 64,
                    }}
                >
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <button
                            onClick={() => setCollapsed(!collapsed)}
                            style={{
                                border: 'none',
                                background: 'transparent',
                                cursor: 'pointer',
                                padding: 8,
                                borderRadius: 6,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                color: '#6b7280',
                                transition: 'all 0.2s',
                            }}
                            onMouseEnter={(e) => {
                                e.currentTarget.style.background = '#f3f4f6';
                                e.currentTarget.style.color = '#111827';
                            }}
                            onMouseLeave={(e) => {
                                e.currentTarget.style.background = 'transparent';
                                e.currentTarget.style.color = '#6b7280';
                            }}
                        >
                            {collapsed ? (
                                <MenuUnfoldOutlined style={{ fontSize: 18 }} />
                            ) : (
                                <MenuFoldOutlined style={{ fontSize: 18 }} />
                            )}
                        </button>

                        {/* 北京时间 - 左上角 */}
                        <div style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                            marginLeft: 16,
                            color: '#6b7280',
                            fontSize: 13,
                        }}>
                            <ClockCircleOutlined style={{ fontSize: 14 }} />
                            <span style={{ fontFamily: 'monospace' }}>
                                {beijingTime}
                            </span>
                        </div>
                    </div>

                    {/* 右侧区域：用户信息 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>

                        {/* 用户信息 */}
                        <Dropdown
                            menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
                            placement="bottomRight"
                        >
                            <div style={{
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                gap: 10,
                                padding: '6px 12px',
                                borderRadius: 8,
                                transition: 'background 0.2s',
                            }}
                                onMouseEnter={(e) => {
                                    e.currentTarget.style.background = '#f3f4f6';
                                }}
                                onMouseLeave={(e) => {
                                    e.currentTarget.style.background = 'transparent';
                                }}
                            >
                                <Avatar
                                    size={32}
                                    style={{
                                        background: '#dbeafe',
                                        color: '#2563eb'
                                    }}
                                >
                                    {user?.name?.[0] || 'U'}
                                </Avatar>
                                <div style={{ lineHeight: 1.2 }}>
                                    <div style={{
                                        fontSize: 14,
                                        fontWeight: 500,
                                        color: '#111827'
                                    }}>
                                        {user?.name || '用户'}
                                    </div>
                                    <div style={{
                                        fontSize: 12,
                                        color: '#9ca3af',
                                    }}>
                                        {user?.role === 'admin' ? '管理员' : user?.role === 'operator' ? '操作员' : '查看者'}
                                    </div>
                                </div>
                            </div>
                        </Dropdown>
                    </div>
                </Header>

                {/* 主内容区域 */}
                <Content
                    style={{
                        margin: 0,
                        padding: 0,
                        minHeight: 280,
                        overflow: 'auto',
                        background: '#f9fafb',
                    }}
                >
                    <Outlet />
                </Content>
            </Layout>

            {/* 个人中心弹窗 */}
            <Modal
                title={<span><UserOutlined style={{ marginRight: 8 }} />个人中心</span>}
                open={profileModalVisible}
                onCancel={() => setProfileModalVisible(false)}
                footer={[
                    <Button key="close" onClick={() => setProfileModalVisible(false)}>关闭</Button>
                ]}
                width={450}
            >
                {user && (
                    <Descriptions column={1} bordered size="small">
                        <Descriptions.Item label="用户名">{user.name || '-'}</Descriptions.Item>
                        <Descriptions.Item label="邮箱">{user.email || '-'}</Descriptions.Item>
                        <Descriptions.Item label="手机号">
                            {user.phone_number ? (
                                <span>{user.phone_country_code || '+86'} {user.phone_number}</span>
                            ) : (
                                <span style={{ color: '#999' }}>未设置</span>
                            )}
                        </Descriptions.Item>
                        <Descriptions.Item label="角色">
                            <Tag color={user.role === 'admin' ? 'red' : user.role === 'operator' ? 'blue' : 'default'}>
                                {user.role === 'admin' ? '管理员' : user.role === 'operator' ? '操作员' : '查看者'}
                            </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="组织机构">
                            {user.organizations && user.organizations.length > 0 ? (
                                <Space wrap>
                                    {user.organizations.map((org: any, index: number) => (
                                        <Tag key={index} color={org.is_primary ? 'blue' : 'default'}>
                                            {org.name}{org.is_primary && '(主)'}
                                            {org.position && <span style={{ marginLeft: 4, fontSize: 11 }}>- {org.position}</span>}
                                        </Tag>
                                    ))}
                                </Space>
                            ) : (
                                <span style={{ color: '#999' }}>未分配组织</span>
                            )}
                        </Descriptions.Item>
                        <Descriptions.Item label="用户ID">{user.id || '-'}</Descriptions.Item>
                    </Descriptions>
                )}
            </Modal>
        </Layout>
    );
};

export default MainLayout;
