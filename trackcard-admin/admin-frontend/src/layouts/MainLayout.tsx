import React, { useState } from 'react';
import { Layout, Menu, Dropdown, Avatar, Button } from 'antd';
import {
    DashboardOutlined,
    TeamOutlined,
    ShoppingCartOutlined,
    HddOutlined,
    UserOutlined,
    LogoutOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
} from '@ant-design/icons';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

const { Header, Sider, Content } = Layout;

const MainLayout: React.FC = () => {
    const [collapsed, setCollapsed] = useState(false);
    const { user, logout } = useAuth();
    const navigate = useNavigate();
    const location = useLocation();

    const menuItems = [
        {
            key: '/',
            icon: <DashboardOutlined />,
            label: '控制台',
        },
        {
            key: '/orgs',
            icon: <TeamOutlined />,
            label: '组织管理',
        },
        {
            key: '/orders',
            icon: <ShoppingCartOutlined />,
            label: '订单管理',
        },
        {
            key: '/devices',
            icon: <HddOutlined />,
            label: '设备管理',
        },
    ];

    const userMenuItems = [
        {
            key: 'logout',
            icon: <LogoutOutlined />,
            label: '退出登录',
            onClick: () => {
                logout();
                navigate('/login');
            },
        },
    ];

    return (
        <Layout style={{ minHeight: '100vh' }}>
            <Sider
                trigger={null}
                collapsible
                collapsed={collapsed}
                theme="dark"
                style={{
                    overflow: 'auto',
                    height: '100vh',
                    position: 'fixed',
                    left: 0,
                    top: 0,
                    bottom: 0,
                }}
            >
                <div style={{
                    height: 64,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: '#fff',
                    fontSize: collapsed ? 16 : 18,
                    fontWeight: 600,
                    background: 'rgba(255,255,255,0.1)',
                }}>
                    {collapsed ? 'TC' : 'TrackCard 管理'}
                </div>
                <Menu
                    theme="dark"
                    mode="inline"
                    selectedKeys={[location.pathname]}
                    items={menuItems}
                    onClick={({ key }) => navigate(key)}
                />
            </Sider>

            <Layout style={{ marginLeft: collapsed ? 80 : 200, transition: 'margin-left 0.2s' }}>
                <Header style={{
                    padding: '0 24px',
                    background: '#fff',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
                    position: 'sticky',
                    top: 0,
                    zIndex: 100,
                }}>
                    <Button
                        type="text"
                        icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                        onClick={() => setCollapsed(!collapsed)}
                    />

                    <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
                        <div style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}>
                            <Avatar icon={<UserOutlined />} style={{ backgroundColor: '#1890ff' }} />
                            <span>{user?.name || user?.username}</span>
                        </div>
                    </Dropdown>
                </Header>

                <Content style={{
                    margin: 24,
                    padding: 24,
                    background: '#fff',
                    borderRadius: 8,
                    minHeight: 'calc(100vh - 112px)',
                }}>
                    <Outlet />
                </Content>
            </Layout>
        </Layout>
    );
};

export default MainLayout;
