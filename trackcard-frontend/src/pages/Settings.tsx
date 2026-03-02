import React, { useEffect, useState } from 'react';
import { Form, Input, Button, message, Spin, Tag, Alert, Menu, Switch, Card, Result, Radio, Space, Descriptions, Avatar } from 'antd';
import type { MenuProps } from 'antd';
import {
    SaveOutlined,
    ThunderboltOutlined,
    CloudOutlined,
    EnvironmentOutlined,
    BellOutlined,
    SafetyOutlined,
    CheckCircleOutlined,
    CloseCircleOutlined,
    LoadingOutlined,
    InfoCircleOutlined,
    TeamOutlined,
    LockOutlined,
    DollarOutlined,
    UserOutlined,
    MailOutlined,
    SettingOutlined
} from '@ant-design/icons';
import { useLocation } from 'react-router-dom';
import api from '../api/client';
import { useAuthStore } from '../store/authStore';
import { useCurrencyStore, CURRENCIES, type CurrencyCode } from '../store/currencyStore';
import SecurityConfig from '../components/SecurityConfig';
import NotificationConfig from '../components/NotificationConfig';
import PermissionConfig from '../components/PermissionConfig';
import ShipmentFieldSettings from '../components/ShipmentFieldSettings';

type SettingKey = 'global' | 'api' | 'geofence' | 'notification' | 'security' | 'permissions' | 'profile' | 'password' | 'shipment_fields';

const Settings: React.FC = () => {
    const location = useLocation();
    const { user } = useAuthStore();
    const { currency, setCurrency } = useCurrencyStore();
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [testing, setTesting] = useState(false);
    const [activeKey, setActiveKey] = useState<SettingKey>('profile');
    const [apiStatus, setApiStatus] = useState<'configured' | 'not_configured' | 'checking'>('checking');
    const [apiInfo, setApiInfo] = useState<{ cid?: string; baseUrl?: string }>({});
    const [passwordForm] = Form.useForm();
    const [form] = Form.useForm();

    // 判断当前是个人设置还是系统设置
    const isProfileSettings = location.pathname === '/settings/profile';
    const isAdmin = user?.role === 'admin';

    useEffect(() => {
        // 只在系统设置页面加载配置
        if (!isProfileSettings) {
            loadConfig();
            checkApiStatus();
        }
        // 个人设置默认显示个人信息，并同步最新用户信息
        if (isProfileSettings) {
            setActiveKey('profile');
            syncCurrentUser();
        }
    }, [isProfileSettings]);

    const syncCurrentUser = async () => {
        try {
            const res = await api.getCurrentUser();
            if (res.data) {
                useAuthStore.setState((state) => ({
                    ...state,
                    user: {
                        ...(state.user || {}),
                        ...res.data,
                    } as any,
                }));
            }
        } catch {
            // 静默失败，避免打扰用户
        }
    };

    const loadConfig = async () => {
        setLoading(true);
        try {
            const res = await api.getConfig();
            if (res.data) {
                form.setFieldsValue(res.data);
            }
        } catch (error) {
            message.error('加载配置失败');
        } finally {
            setLoading(false);
        }
    };

    const checkApiStatus = async () => {
        setApiStatus('checking');
        try {
            const res = await api.getConfig();
            if (res.data) {
                const cid = res.data.kuaihuoyun_cid;
                const secretKey = res.data.kuaihuoyun_secret_key;
                setApiInfo({
                    cid: cid,
                    baseUrl: 'bapi.kuaihuoyun.com'
                });
                // 如果有CID和密钥，认为已配置
                if (cid && secretKey && secretKey !== '****') {
                    setApiStatus('configured');
                } else {
                    setApiStatus('not_configured');
                }
            }
        } catch (error) {
            setApiStatus('not_configured');
        }
    };

    const handleSave = async (values: Record<string, string>) => {
        setSaving(true);
        try {
            // 保存围栏相关的配置
            const safeValues: Record<string, string> = {};
            // 保存自动状态识别开关
            if (values.auto_status_enabled !== undefined) {
                safeValues.auto_status_enabled = values.auto_status_enabled ? 'true' : 'false';
            }
            if (values.geofence_radius) {
                safeValues.geofence_radius = values.geofence_radius;
            }
            if (values.port_geofence_radius) {
                safeValues.port_geofence_radius = values.port_geofence_radius;
            }
            await api.updateConfig(safeValues);
            message.success('配置保存成功');
        } catch (error) {
            message.error('保存失败');
        } finally {
            setSaving(false);
        }
    };

    const handleTestAPI = async () => {
        setTesting(true);
        try {
            const res = await api.testKuaihuoyunAPI();
            if (res.success) {
                message.success('API连接测试成功');
                setApiStatus('configured');
            } else {
                message.error('API连接测试失败: ' + (res.error || '未知错误'));
            }
        } catch (error: any) {
            message.error('API连接测试失败: ' + (error.response?.data?.error || '请检查服务器配置'));
        } finally {
            setTesting(false);
        }
    };

    const menuItems: MenuProps['items'] = [
        // 个人设置菜单
        ...(isProfileSettings ? [
            {
                key: 'profile',
                icon: <UserOutlined />,
                label: '个人信息',
            },
            {
                key: 'password',
                icon: <LockOutlined />,
                label: '修改密码',
            },
        ] : []),
        // 系统设置菜单（仅管理员）
        ...(!isProfileSettings ? [
            {
                key: 'global',
                icon: <DollarOutlined />,
                label: '货币设置',
            },
            {
                key: 'geofence',
                icon: <EnvironmentOutlined />,
                label: '围栏设置',
            },
            {
                key: 'shipment_fields',
                icon: <SettingOutlined />,
                label: '运单字段',
            },
            {
                key: 'api',
                icon: <CloudOutlined />,
                label: '快货运API',
            },
            {
                key: 'permissions',
                icon: <TeamOutlined />,
                label: '权限管理',
            },
            {
                key: 'notification',
                icon: <BellOutlined />,
                label: '通知设置',
            },
            {
                key: 'security',
                icon: <SafetyOutlined />,
                label: '账户安全',
            },
        ] : []),
    ];

    const renderApiConfig = () => (
        <div>
            <h3 style={{ marginBottom: 24, fontSize: 18, fontWeight: 600 }}>快货运API配置</h3>

            {/* 连接状态卡片 */}
            <div className="card" style={{ marginBottom: 24, padding: 20 }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
                    <span style={{ fontSize: 14, color: 'var(--text-tertiary)' }}>连接状态</span>
                    {apiStatus === 'checking' ? (
                        <Tag icon={<LoadingOutlined spin />} color="processing">检测中</Tag>
                    ) : apiStatus === 'configured' ? (
                        <Tag icon={<CheckCircleOutlined />} color="success">已配置</Tag>
                    ) : (
                        <Tag icon={<CloseCircleOutlined />} color="error">未配置</Tag>
                    )}
                </div>

                <div style={{ display: 'grid', gap: 12 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <span style={{ color: 'var(--text-tertiary)', fontSize: 14 }}>客户ID (CID)</span>
                        <span style={{ fontWeight: 500 }}>{apiInfo.cid || '-'}</span>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <span style={{ color: 'var(--text-tertiary)', fontSize: 14 }}>API地址</span>
                        <span style={{ fontWeight: 500 }}>{apiInfo.baseUrl || '-'}</span>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <span style={{ color: 'var(--text-tertiary)', fontSize: 14 }}>API密钥</span>
                        <span style={{ fontWeight: 500, color: 'var(--text-muted)' }}>••••••••••••</span>
                    </div>
                </div>
            </div>

            {/* 提示信息 */}
            <Alert
                message="API配置由服务器环境变量管理"
                description={
                    <div style={{ marginTop: 8 }}>
                        <p style={{ marginBottom: 8 }}>如需修改API配置，请联系系统管理员修改以下环境变量：</p>
                        <ul style={{ margin: 0, paddingLeft: 20 }}>
                            <li><code>KUAIHUOYUN_CID</code> - 客户ID</li>
                            <li><code>KUAIHUOYUN_SECRET_KEY</code> - API密钥</li>
                            <li><code>KUAIHUOYUN_BASE_URL</code> - API地址</li>
                        </ul>
                    </div>
                }
                type="info"
                showIcon
                icon={<InfoCircleOutlined />}
                style={{ marginBottom: 24 }}
            />

            {/* 测试连接按钮 */}
            <Button
                type="primary"
                icon={<ThunderboltOutlined />}
                onClick={handleTestAPI}
                loading={testing}
                disabled={apiStatus === 'not_configured'}
            >
                测试连接
            </Button>
        </div>
    );

    // 个人设置 - 修改密码
    const handlePasswordChange = async (values: any) => {
        if (values.new_password !== values.confirm_password) {
            message.error('两次输入的密码不一致');
            return;
        }

        setSaving(true);
        try {
            await api.changePassword({
                current_password: values.old_password,
                new_password: values.new_password,
            });
            message.success('密码修改成功，请重新登录');
            passwordForm.resetFields();
        } catch (error: any) {
            message.error(error.response?.data?.error || '密码修改失败');
        } finally {
            setSaving(false);
        }
    };

    const renderProfileSettings = () => (
        <div>
            <h3 style={{ marginBottom: 24, fontSize: 18, fontWeight: 600 }}>个人信息</h3>

            {/* 个人信息卡片 */}
            <Card style={{ marginBottom: 24 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
                    <Avatar size={80} style={{ background: '#1890ff', fontSize: 32 }}>
                        {user?.name?.[0] || 'U'}
                    </Avatar>
                    <div>
                        <div style={{ fontSize: 20, fontWeight: 600, marginBottom: 8 }}>
                            {user?.name || '-'}
                        </div>
                        <div style={{ color: '#666', marginBottom: 4 }}>
                            <UserOutlined style={{ marginRight: 8 }} />
                            {user?.role === 'admin' ? '管理员' : user?.role === 'operator' ? '操作员' : '查看者'}
                        </div>
                        <div style={{ color: '#666', marginBottom: 4 }}>
                            <UserOutlined style={{ marginRight: 8 }} />
                            {user?.phone_number ? `${user?.phone_country_code || '+86'} ${user?.phone_number}` : '手机号未设置'}
                        </div>
                        {user?.organizations && user.organizations.length > 0 && (
                            <div style={{ color: '#666' }}>
                                <TeamOutlined style={{ marginRight: 8 }} />
                                {user.organizations.map((org: any) => org.name).join('、')}
                            </div>
                        )}
                    </div>
                </div>
            </Card>

            {/* 详细信息 */}
            <Descriptions title="账号信息" bordered column={1} style={{ marginBottom: 24 }}>
                <Descriptions.Item label="用户ID">{user?.id || '-'}</Descriptions.Item>
                <Descriptions.Item label="姓名">{user?.name || '-'}</Descriptions.Item>
                <Descriptions.Item label="邮箱">
                    <MailOutlined style={{ marginRight: 8 }} />
                    {user?.email || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="手机号">
                    <UserOutlined style={{ marginRight: 8 }} />
                    {user?.phone_number ? `${user?.phone_country_code || '+86'} ${user?.phone_number}` : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="角色">
                    <Tag color={user?.role === 'admin' ? 'red' : user?.role === 'operator' ? 'blue' : 'default'}>
                        {user?.role === 'admin' ? '管理员' : user?.role === 'operator' ? '操作员' : '查看者'}
                    </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                    <Tag color={user?.status === 'active' ? 'success' : 'error'}>
                        {user?.status === 'active' ? '正常' : '禁用'}
                    </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="注册时间">
                    {user?.created_at ? new Date(user.created_at).toLocaleString('zh-CN') : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="最后登录">
                    {user?.last_login ? new Date(user.last_login).toLocaleString('zh-CN') : '-'}
                </Descriptions.Item>
            </Descriptions>
        </div>
    );

    const renderPasswordSettings = () => (
        <div>
            <h3 style={{ marginBottom: 24, fontSize: 18, fontWeight: 600 }}>修改密码</h3>

            <Alert
                message="安全提示"
                description="为了您的账号安全，建议定期修改密码。密码长度至少6位，建议包含字母、数字和特殊字符。"
                type="info"
                showIcon
                style={{ marginBottom: 24 }}
            />

            <Form
                form={passwordForm}
                layout="vertical"
                onFinish={handlePasswordChange}
                style={{ maxWidth: 500 }}
            >
                <Form.Item
                    name="old_password"
                    label="当前密码"
                    rules={[{ required: true, message: '请输入当前密码' }]}
                >
                    <Input.Password placeholder="请输入当前密码" />
                </Form.Item>

                <Form.Item
                    name="new_password"
                    label="新密码"
                    rules={[
                        { required: true, message: '请输入新密码' },
                        { min: 6, message: '密码长度至少6位' },
                    ]}
                >
                    <Input.Password placeholder="请输入新密码（至少6位）" />
                </Form.Item>

                <Form.Item
                    name="confirm_password"
                    label="确认新密码"
                    dependencies={['new_password']}
                    rules={[
                        { required: true, message: '请再次输入新密码' },
                        ({ getFieldValue }) => ({
                            validator(_, value) {
                                if (!value || getFieldValue('new_password') === value) {
                                    return Promise.resolve();
                                }
                                return Promise.reject(new Error('两次输入的密码不一致'));
                            },
                        }),
                    ]}
                >
                    <Input.Password placeholder="请再次输入新密码" />
                </Form.Item>

                <Form.Item>
                    <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
                        修改密码
                    </Button>
                </Form.Item>
            </Form>
        </div>
    );

    const renderGlobalConfig = () => (
        <div>
            <h3 style={{ marginBottom: 24, fontSize: 18, fontWeight: 600 }}>货币设置</h3>

            {/* 货币设置 */}
            <div className="card" style={{ marginBottom: 24, padding: 20 }}>
                <div style={{ marginBottom: 16 }}>
                    <span style={{ fontSize: 14, fontWeight: 500 }}>货币设置</span>
                    <p style={{ fontSize: 13, color: 'var(--text-tertiary)', marginTop: 4, marginBottom: 16 }}>
                        设置系统默认显示货币，切换后全系统统一使用该货币（线路规划模块除外）
                    </p>
                </div>

                <Radio.Group
                    value={currency}
                    onChange={(e) => {
                        setCurrency(e.target.value as CurrencyCode);
                        message.success(`货币已切换为${CURRENCIES[e.target.value as CurrencyCode].name}`);
                    }}
                    size="large"
                >
                    <Space direction="vertical" size="middle">
                        <Radio value="CNY" style={{ fontSize: 15 }}>
                            <span style={{ marginLeft: 8 }}>¥ 人民币 (CNY)</span>
                            {currency === 'CNY' && <Tag color="blue" style={{ marginLeft: 12 }}>当前</Tag>}
                        </Radio>
                        <Radio value="USD" style={{ fontSize: 15 }}>
                            <span style={{ marginLeft: 8 }}>$ 美元 (USD)</span>
                            {currency === 'USD' && <Tag color="blue" style={{ marginLeft: 12 }}>当前</Tag>}
                        </Radio>
                    </Space>
                </Radio.Group>
            </div>

            <Alert
                message="提示"
                description="货币设置将影响运单管理、预警中心等模块中的金额显示。线路规划模块拥有独立的货币选择，不受此设置影响。"
                type="info"
                showIcon
                icon={<InfoCircleOutlined />}
            />
        </div>
    );

    const renderGeofenceConfig = () => (
        <div>
            <h3 style={{ marginBottom: 24, fontSize: 18, fontWeight: 600 }}>围栏设置</h3>
            <Form form={form} layout="vertical" onFinish={handleSave} style={{ maxWidth: 500 }}>
                <Form.Item
                    name="auto_status_enabled"
                    label="发运状态自动识别"
                    valuePropName="checked"
                    extra="开启后，系统将根据定位器位置自动判断运单状态（离开发货地→运输中，到达目的地→已到达）"
                >
                    <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                </Form.Item>

                <Form.Item
                    name="geofence_radius"
                    label="收发货地围栏半径"
                    rules={[{ required: true, message: '请输入围栏半径' }]}
                    extra="定位器距离发货地/目的地超过此半径时触发状态变更"
                >
                    <Input type="number" placeholder="1000" suffix="米" />
                </Form.Item>

                <Form.Item
                    name="port_geofence_radius"
                    label="港口码头围栏半径"
                    rules={[{ required: true, message: '请输入港口围栏半径' }]}
                    extra="港口、码头等大型场站的围栏识别范围，建议设置较大值"
                >
                    <Input type="number" placeholder="5000" suffix="米" />
                </Form.Item>

                <Form.Item>
                    <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
                        保存设置
                    </Button>
                </Form.Item>
            </Form>
        </div>
    );

    const renderNotificationConfig = () => <NotificationConfig />;

    const renderSecurityConfig = () => <SecurityConfig />;

    const renderPermissionsConfig = () => <PermissionConfig />;

    const renderShipmentFieldSettings = () => <ShipmentFieldSettings />;

    const renderContent = () => {
        // 个人设置
        if (isProfileSettings) {
            switch (activeKey) {
                case 'profile':
                    return renderProfileSettings();
                case 'password':
                    return renderPasswordSettings();
                default:
                    return renderProfileSettings();
            }
        }

        // 系统设置（需要管理员权限）
        if (!isAdmin) {
            return (
                <Result
                    status="403"
                    title="无操作权限"
                    subTitle="系统设置仅限管理员访问，如需修改系统配置，请联系管理员。"
                    icon={<LockOutlined style={{ color: '#faad14' }} />}
                />
            );
        }

        switch (activeKey) {
            case 'global':
                return renderGlobalConfig();
            case 'geofence':
                return renderGeofenceConfig();
            case 'shipment_fields':
                return renderShipmentFieldSettings();
            case 'api':
                return renderApiConfig();
            case 'permissions':
                return renderPermissionsConfig();
            case 'notification':
                return renderNotificationConfig();
            case 'security':
                return renderSecurityConfig();
            default:
                return renderGlobalConfig();
        }
    };

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}>
                <Spin size="large" />
            </div>
        );
    }

    return (
        <Card title={isProfileSettings ? '个人设置' : '系统设置'} headStyle={{ fontSize: 16, fontWeight: 600 }}>
            <div style={{ display: 'flex', gap: 24 }}>
                {/* 左侧菜单 */}
                <div style={{ width: 200, flexShrink: 0 }}>
                    <Menu
                        mode="inline"
                        selectedKeys={[activeKey]}
                        items={menuItems}
                        onClick={({ key }) => setActiveKey(key as SettingKey)}
                        style={{ border: 'none', background: 'transparent' }}
                    />
                </div>

                {/* 右侧内容 */}
                <div style={{ flex: 1, paddingLeft: 24, borderLeft: '1px solid #f0f0f0' }}>
                    {renderContent()}
                </div>
            </div>
        </Card>
    );
};

export default Settings;
