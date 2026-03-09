import React, { useState, useCallback, useEffect } from 'react';
import { Form, Input, Button, Card, Tag, Switch, Select, Alert, message, Modal, Spin } from 'antd';
import { ExclamationCircleOutlined, LoadingOutlined } from '@ant-design/icons';

import api from '../api/client';
import './SecuritySettings.css';

// 类型定义
interface LoginRecord {
    id: number | string;
    time: string;
    ip: string;
    device: string;
    location: string;
    status: 'success' | 'failed';
}

interface ActiveSession {
    id: string;
    device: string;
    ip: string;
    location: string;
    lastActive: string;
    isCurrent: boolean;
}

interface SecuritySettings {
    abnormalLoginDetection: boolean;
    newDeviceNotification: boolean;
    loginFailureLock: boolean;
    sessionDuration: string;
}

// 密码强度检测
const checkPasswordStrength = (password: string): { level: 'weak' | 'medium' | 'strong'; message: string } => {
    if (!password) return { level: 'weak', message: '' };

    const hasLetter = /[a-zA-Z]/.test(password);
    const hasNumber = /\d/.test(password);
    const hasSpecial = /[!@#$%^&*(),.?":{}|<>]/.test(password);
    const isLongEnough = password.length >= 8;
    const isVeryLong = password.length >= 12;

    const score = [hasLetter, hasNumber, hasSpecial, isLongEnough, isVeryLong].filter(Boolean).length;

    if (score <= 2) return { level: 'weak', message: '弱：建议使用更复杂的密码' };
    if (score <= 3) return { level: 'medium', message: '中：可以更安全' };
    return { level: 'strong', message: '强：密码安全性良好' };
};

// 密码强度指示器组件
const PasswordStrength: React.FC<{ password: string }> = ({ password }) => {
    const strength = checkPasswordStrength(password);
    const colorMap = { weak: '#ff4d4f', medium: '#faad14', strong: '#52c41a' };

    if (!password) return null;

    return (
        <div className="password-strength">
            <div className="strength-bars">
                <div className={`strength-bar ${strength.level !== 'weak' ? 'active' : ''}`}
                    style={{ backgroundColor: colorMap[strength.level] }} />
                <div className={`strength-bar ${strength.level === 'medium' || strength.level === 'strong' ? 'active' : ''}`}
                    style={{ backgroundColor: strength.level !== 'weak' ? colorMap[strength.level] : '#e8e8e8' }} />
                <div className={`strength-bar ${strength.level === 'strong' ? 'active' : ''}`}
                    style={{ backgroundColor: strength.level === 'strong' ? colorMap[strength.level] : '#e8e8e8' }} />
            </div>
            <span className="strength-text" style={{ color: colorMap[strength.level] }}>
                {strength.message}
            </span>
        </div>
    );
};

// 修改密码表单组件
const ChangePasswordForm: React.FC = () => {
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);
    const [newPassword, setNewPassword] = useState('');

    const handleSubmit = useCallback(async (values: { currentPassword: string; newPassword: string; confirmPassword: string }) => {
        // 验证新密码强度
        const strength = checkPasswordStrength(values.newPassword);
        if (strength.level === 'weak') {
            message.error('密码强度不足，请使用至少8位包含字母和数字的密码');
            return;
        }

        // 验证两次密码一致
        if (values.newPassword !== values.confirmPassword) {
            message.error('两次输入的新密码不一致');
            return;
        }

        setLoading(true);
        try {
            // 调用API修改密码
            await api.changePassword({
                current_password: values.currentPassword,
                new_password: values.newPassword,
            });
            message.success('密码修改成功，请重新登录');
            form.resetFields();
            setNewPassword('');
        } catch (error: any) {
            const errorMsg = error?.response?.data?.error || '密码修改失败，请检查当前密码是否正确';
            message.error(errorMsg);
        } finally {
            setLoading(false);
        }
    }, [form]);

    return (
        <Card size="small" title="修改密码" className="security-card">
            <Form form={form} layout="vertical" onFinish={handleSubmit}>
                <Form.Item
                    name="currentPassword"
                    label="当前密码"
                    rules={[{ required: true, message: '请输入当前密码' }]}
                    className="form-item"
                >
                    <Input.Password placeholder="请输入当前密码" />
                </Form.Item>
                <Form.Item
                    name="newPassword"
                    label="新密码"
                    rules={[
                        { required: true, message: '请输入新密码' },
                        { min: 8, message: '密码至少8位' },
                        {
                            pattern: /^(?=.*[A-Za-z])(?=.*\d)/,
                            message: '密码必须包含字母和数字'
                        }
                    ]}
                    className="form-item"
                >
                    <Input.Password
                        placeholder="请输入新密码（至少8位，含字母和数字）"
                        onChange={(e) => setNewPassword(e.target.value)}
                    />
                </Form.Item>
                <PasswordStrength password={newPassword} />
                <Form.Item
                    name="confirmPassword"
                    label="确认新密码"
                    dependencies={['newPassword']}
                    rules={[
                        { required: true, message: '请再次输入新密码' },
                        ({ getFieldValue }) => ({
                            validator(_, value) {
                                if (!value || getFieldValue('newPassword') === value) {
                                    return Promise.resolve();
                                }
                                return Promise.reject(new Error('两次输入的密码不一致'));
                            },
                        }),
                    ]}
                    className="form-item"
                >
                    <Input.Password placeholder="请再次输入新密码" />
                </Form.Item>
                <Button type="primary" htmlType="submit" loading={loading}>
                    修改密码
                </Button>
            </Form>
        </Card>
    );
};

// 登录安全设置组件
const LoginSecuritySettings: React.FC<{
    settings: SecuritySettings;
    onChange: (key: keyof SecuritySettings, value: boolean | string) => void;
    onSave: () => void;
    saving: boolean;
}> = ({ settings, onChange, onSave, saving }) => (
    <Card size="small" title="登录安全" className="security-card">
        <div className="setting-row">
            <span>异常登录检测</span>
            <Switch
                checked={settings.abnormalLoginDetection}
                onChange={(checked) => onChange('abnormalLoginDetection', checked)}
            />
        </div>
        <div className="setting-row">
            <span>新设备登录通知</span>
            <Switch
                checked={settings.newDeviceNotification}
                onChange={(checked) => onChange('newDeviceNotification', checked)}
            />
        </div>
        <div className="setting-row">
            <span>登录失败锁定（5次失败后锁定30分钟）</span>
            <Switch
                checked={settings.loginFailureLock}
                onChange={(checked) => onChange('loginFailureLock', checked)}
            />
        </div>
        <div className="setting-row last">
            <span>登录会话有效期</span>
            <Select
                value={settings.sessionDuration}
                onChange={(value) => onChange('sessionDuration', value)}
                style={{ width: 150 }}
            >
                <Select.Option value="1">1小时</Select.Option>
                <Select.Option value="8">8小时</Select.Option>
                <Select.Option value="24">24小时</Select.Option>
                <Select.Option value="168">7天</Select.Option>
                <Select.Option value="720">30天</Select.Option>
            </Select>
        </div>
        <Button type="primary" onClick={onSave} loading={saving} style={{ marginTop: 12 }}>
            保存设置
        </Button>
    </Card>
);

// 在线设备管理组件
const ActiveSessionsCard: React.FC<{
    sessions: ActiveSession[];
    onLogout: (sessionId: string) => void;
    loading: boolean;
}> = ({ sessions, onLogout, loading }) => {
    const handleLogout = useCallback((session: ActiveSession) => {
        Modal.confirm({
            title: '确认下线设备',
            icon: <ExclamationCircleOutlined />,
            content: `确定要强制下线设备 "${session.device}" 吗？该设备将需要重新登录。`,
            okText: '确认下线',
            okType: 'danger',
            cancelText: '取消',
            onOk: () => onLogout(session.id),
        });
    }, [onLogout]);

    if (loading) {
        return (
            <Card size="small" title="在线设备" className="security-card">
                <div className="loading-container">
                    <Spin indicator={<LoadingOutlined />} />
                </div>
            </Card>
        );
    }

    return (
        <Card size="small" title="在线设备" className="security-card">
            {sessions.length === 0 ? (
                <div className="empty-text">暂无在线设备</div>
            ) : (
                sessions.map(session => (
                    <div key={session.id} className="session-item">
                        <div className="session-info">
                            <div className="session-device">
                                {session.device}
                                {session.isCurrent && <Tag color="green" className="current-tag">当前</Tag>}
                            </div>
                            <div className="session-meta">
                                {session.ip} · {session.location} · {session.lastActive}
                            </div>
                        </div>
                        {!session.isCurrent && (
                            <Button size="small" danger onClick={() => handleLogout(session)}>
                                下线
                            </Button>
                        )}
                    </div>
                ))
            )}
        </Card>
    );
};

// 登录记录表格组件
const LoginHistoryTable: React.FC<{ records: LoginRecord[]; loading: boolean }> = ({ records, loading }) => (
    <Card size="small" title="最近登录记录" className="security-card">
        {loading ? (
            <div className="loading-container">
                <Spin indicator={<LoadingOutlined />} />
            </div>
        ) : (
            <table className="login-history-table">
                <thead>
                    <tr>
                        <th>时间</th>
                        <th>IP地址</th>
                        <th>设备</th>
                        <th>位置</th>
                        <th>状态</th>
                    </tr>
                </thead>
                <tbody>
                    {records.length === 0 ? (
                        <tr>
                            <td colSpan={5} className="empty-text">暂无登录记录</td>
                        </tr>
                    ) : (
                        records.map(log => (
                            <tr key={log.id}>
                                <td>{log.time}</td>
                                <td>{log.ip}</td>
                                <td>{log.device}</td>
                                <td>{log.location}</td>
                                <td>
                                    <Tag color={log.status === 'success' ? 'green' : 'red'}>
                                        {log.status === 'success' ? '成功' : '失败'}
                                    </Tag>
                                </td>
                            </tr>
                        ))
                    )}
                </tbody>
            </table>
        )}
    </Card>
);

// 主组件
const SecurityConfig: React.FC = () => {

    const [loadingHistory, setLoadingHistory] = useState(false);
    const [loadingSessions, setLoadingSessions] = useState(false);
    const [savingSettings, setSavingSettings] = useState(false);

    // 登录安全设置状态
    const [securitySettings, setSecuritySettings] = useState<SecuritySettings>({
        abnormalLoginDetection: true,
        newDeviceNotification: true,
        loginFailureLock: true,
        sessionDuration: '24',
    });

    // 模拟数据 - 实际应从API获取
    const [loginHistory, setLoginHistory] = useState<LoginRecord[]>([]);
    const [activeSessions, setActiveSessions] = useState<ActiveSession[]>([]);

    // 加载数据
    useEffect(() => {
        loadLoginHistory();
        loadActiveSessions();
    }, []);

    const loadLoginHistory = useCallback(async () => {
        setLoadingHistory(true);
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 500));
            setLoginHistory([
                { id: 1, time: '2026-01-19 09:45:23', ip: '192.168.1.100', device: 'Chrome / macOS', location: '深圳市', status: 'success' },
                { id: 2, time: '2026-01-18 21:12:08', ip: '192.168.1.100', device: 'Chrome / macOS', location: '深圳市', status: 'success' },
                { id: 3, time: '2026-01-18 14:30:45', ip: '10.0.0.15', device: 'Safari / iOS', location: '上海市', status: 'success' },
                { id: 4, time: '2026-01-17 10:22:11', ip: '192.168.1.55', device: 'Firefox / Windows', location: '北京市', status: 'failed' },
            ]);
        } catch (error) {
            message.error('加载登录记录失败');
        } finally {
            setLoadingHistory(false);
        }
    }, []);

    const loadActiveSessions = useCallback(async () => {
        setLoadingSessions(true);
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 300));
            setActiveSessions([
                { id: 's1', device: 'Chrome / macOS', ip: '192.168.1.100', location: '深圳市', lastActive: '当前设备', isCurrent: true },
                { id: 's2', device: 'Safari / iOS', ip: '10.0.0.15', location: '上海市', lastActive: '2小时前', isCurrent: false },
            ]);
        } catch (error) {
            message.error('加载在线设备失败');
        } finally {
            setLoadingSessions(false);
        }
    }, []);

    const handleSettingChange = useCallback((key: keyof SecuritySettings, value: boolean | string) => {
        setSecuritySettings(prev => ({ ...prev, [key]: value }));
    }, []);

    const handleSaveSettings = useCallback(async () => {
        setSavingSettings(true);
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 500));
            message.success('安全设置已保存');
        } catch (error) {
            message.error('保存设置失败');
        } finally {
            setSavingSettings(false);
        }
    }, [securitySettings]);

    const handleLogoutSession = useCallback(async (sessionId: string) => {
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 300));
            setActiveSessions(prev => prev.filter(s => s.id !== sessionId));
            message.success('已强制下线该设备');
        } catch (error) {
            message.error('下线失败，请重试');
        }
    }, []);

    const handleDeleteAccount = useCallback(() => {
        Modal.confirm({
            title: '确认注销账户',
            icon: <ExclamationCircleOutlined />,
            content: (
                <div>
                    <p style={{ color: '#ff4d4f', fontWeight: 500 }}>警告：此操作不可撤销！</p>
                    <p>注销账户后，您的所有数据将被永久删除，包括：</p>
                    <ul>
                        <li>个人信息和设置</li>
                        <li>操作历史记录</li>
                        <li>相关业务数据</li>
                    </ul>
                </div>
            ),
            okText: '确认注销',
            okType: 'danger',
            cancelText: '取消',
            onOk: async () => {
                message.info('账户注销功能暂未开放');
            },
        });
    }, []);

    return (
        <div className="security-config">
            <h3 className="security-title">账户安全</h3>

            <ChangePasswordForm />

            <LoginSecuritySettings
                settings={securitySettings}
                onChange={handleSettingChange}
                onSave={handleSaveSettings}
                saving={savingSettings}
            />

            <ActiveSessionsCard
                sessions={activeSessions}
                onLogout={handleLogoutSession}
                loading={loadingSessions}
            />

            <LoginHistoryTable
                records={loginHistory}
                loading={loadingHistory}
            />

            {/* 危险操作 */}
            <Card size="small" title="危险操作" className="security-card danger-zone">
                <Alert
                    message="注销账户"
                    description="注销账户后，所有数据将被永久删除且无法恢复。"
                    type="error"
                    showIcon
                    action={
                        <Button danger onClick={handleDeleteAccount}>
                            注销账户
                        </Button>
                    }
                />
            </Card>
        </div>
    );
};

export default SecurityConfig;
