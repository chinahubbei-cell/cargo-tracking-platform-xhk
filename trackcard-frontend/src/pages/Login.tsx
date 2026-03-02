import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Card, message, Typography, Tabs, Modal, Radio, Alert } from 'antd';
import { UserOutlined, LockOutlined, MobileOutlined, CopyOutlined } from '@ant-design/icons';
import api from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { UserOrg } from '../types';

const { Title, Text } = Typography;

const Login: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [codeLoading, setCodeLoading] = useState(false);
    const [countdown, setCountdown] = useState(0);
    const [orgSelectVisible, setOrgSelectVisible] = useState(false);
    const [orgs, setOrgs] = useState<UserOrg[]>([]);
    const [selectedOrg, setSelectedOrg] = useState<string>('');
    const [debugCode, setDebugCode] = useState<string>('');

    const navigate = useNavigate();
    const { setAuth } = useAuthStore();
    const [smsForm] = Form.useForm();

    const onPasswordLogin = async (values: { email: string; password: string }) => {
        setLoading(true);
        try {
            const response: any = await api.login(values);
            const data = response.data || response;
            if (data && data.token) {
                setAuth(data.token, data.user);
                message.success('登录成功');
                navigate('/dashboard');
            } else {
                throw new Error('登录响应格式错误');
            }
        } catch (error: any) {
            message.error(error.response?.data?.error || '登录失败，请检查邮箱和密码');
        } finally {
            setLoading(false);
        }
    };

    const sendCode = async () => {
        const phone = smsForm.getFieldValue('phone_number');
        if (!phone) {
            message.warning('请先输入手机号');
            return;
        }
        setCodeLoading(true);
        try {
            const res: any = await api.sendSMSCode({ phone_number: phone, scene: 'login' });
            const data = res.data || res;
            if (data?.debug_code) {
                setDebugCode(data.debug_code);
                smsForm.setFieldValue('code', data.debug_code);
                message.success(`验证码已发送（测试模式）：${data.debug_code}`);
            } else {
                setDebugCode('');
                message.success('验证码已发送');
            }
            setCountdown(60);
            const timer = setInterval(() => {
                setCountdown((prev) => {
                    if (prev <= 1) {
                        clearInterval(timer);
                        return 0;
                    }
                    return prev - 1;
                });
            }, 1000);
        } catch (error: any) {
            message.error(error.response?.data?.error || '验证码发送失败');
        } finally {
            setCodeLoading(false);
        }
    };

    const onSMSLogin = async (values: { phone_number: string; code: string }) => {
        setLoading(true);
        try {
            const response: any = await api.loginBySMS(values);
            const data = response.data || response;

            if (data.need_select_org) {
                setOrgs(data.orgs || []);
                if (data.orgs?.length) setSelectedOrg(data.orgs[0].id);
                if (data.token_temp) {
                    localStorage.setItem('token', data.token_temp);
                }
                setOrgSelectVisible(true);
                return;
            }

            if (data.token) {
                setAuth(data.token, data.user);
                message.success('登录成功');
                navigate('/dashboard');
            }
        } catch (error: any) {
            message.error(error.response?.data?.error || '登录失败，请检查手机号和验证码');
        } finally {
            setLoading(false);
        }
    };

    const confirmSelectOrg = async () => {
        if (!selectedOrg) return;
        setLoading(true);
        try {
            const response: any = await api.selectOrg(selectedOrg);
            const data = response.data || response;
            if (data.token) {
                localStorage.setItem('token', data.token);
                const me: any = await api.getCurrentUser();
                const meData = me.data || me;
                setAuth(data.token, meData);
                setOrgSelectVisible(false);
                message.success('机构选择成功');
                navigate('/dashboard');
            }
        } catch (error: any) {
            message.error(error.response?.data?.error || '机构选择失败');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' }}>
            <Card style={{ width: 420, boxShadow: '0 8px 32px rgba(0,0,0,0.2)', borderRadius: 16 }}>
                <div style={{ textAlign: 'center', marginBottom: 20 }}>
                    <Title level={2} style={{ marginBottom: 8 }}>📍 TrackCard</Title>
                    <Text type="secondary">全球货物追踪平台</Text>
                </div>

                <Tabs
                    defaultActiveKey="password"
                    items={[
                        {
                            key: 'password',
                            label: '邮箱密码登录',
                            children: (
                                <Form name="pwd-login" onFinish={onPasswordLogin} autoComplete="off" size="large">
                                    <Form.Item name="email" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '请输入有效邮箱' }]}>
                                        <Input prefix={<UserOutlined />} placeholder="邮箱" />
                                    </Form.Item>
                                    <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }, { min: 6, message: '密码至少6位' }]}>
                                        <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                                    </Form.Item>
                                    {debugCode && (
                                        <Alert
                                            type="info"
                                            showIcon
                                            style={{ marginBottom: 12 }}
                                            message={
                                                <span>
                                                    测试验证码：<b>{debugCode}</b>
                                                    <Button
                                                        type="link"
                                                        size="small"
                                                        icon={<CopyOutlined />}
                                                        onClick={async () => {
                                                            try {
                                                                await navigator.clipboard.writeText(debugCode);
                                                                message.success('验证码已复制');
                                                            } catch {
                                                                message.warning('复制失败，请手动复制');
                                                            }
                                                        }}
                                                        style={{ marginLeft: 8, padding: 0, height: 'auto' }}
                                                    >复制</Button>
                                                </span>
                                            }
                                            description="仅测试环境展示，已自动填入验证码输入框。"
                                        />
                                    )}
                                    <Form.Item><Button type="primary" htmlType="submit" loading={loading} block>登录</Button></Form.Item>
                                </Form>
                            ),
                        },
                        {
                            key: 'sms',
                            label: '手机号验证码登录',
                            children: (
                                <Form form={smsForm} name="sms-login" onFinish={onSMSLogin} autoComplete="off" size="large">
                                    <Form.Item name="phone_number" rules={[{ required: true, message: '请输入手机号' }, { pattern: /^1\d{10}$/, message: '请输入11位大陆手机号' }]}>
                                        <Input prefix={<MobileOutlined />} placeholder="手机号（+86）" />
                                    </Form.Item>
                                    <Form.Item>
                                        <Input.Group compact>
                                            <Form.Item name="code" noStyle rules={[{ required: true, message: '请输入验证码' }, { len: 6, message: '验证码6位' }]}>
                                                <Input style={{ width: '60%' }} placeholder="验证码" />
                                            </Form.Item>
                                            <Button style={{ width: '40%' }} onClick={sendCode} disabled={countdown > 0} loading={codeLoading}>
                                                {countdown > 0 ? `${countdown}s后重试` : '发送验证码'}
                                            </Button>
                                        </Input.Group>
                                    </Form.Item>
                                    {debugCode && (
                                        <Alert
                                            type="info"
                                            showIcon
                                            style={{ marginBottom: 12 }}
                                            message={
                                                <span>
                                                    测试验证码：<b>{debugCode}</b>
                                                    <Button
                                                        type="link"
                                                        size="small"
                                                        icon={<CopyOutlined />}
                                                        onClick={async () => {
                                                            try {
                                                                await navigator.clipboard.writeText(debugCode);
                                                                message.success('验证码已复制');
                                                            } catch {
                                                                message.warning('复制失败，请手动复制');
                                                            }
                                                        }}
                                                        style={{ marginLeft: 8, padding: 0, height: 'auto' }}
                                                    >复制</Button>
                                                </span>
                                            }
                                            description="仅测试环境展示，已自动填入验证码输入框。"
                                        />
                                    )}
                                    <Form.Item><Button type="primary" htmlType="submit" loading={loading} block>登录</Button></Form.Item>
                                </Form>
                            ),
                        },
                    ]}
                />

                <div style={{ textAlign: 'center' }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>默认账号：admin@trackcard.com / admin123</Text>
                </div>
            </Card>

            <Modal title="选择登录机构" open={orgSelectVisible} onOk={confirmSelectOrg} onCancel={() => setOrgSelectVisible(false)} confirmLoading={loading} okText="确认进入">
                <Radio.Group value={selectedOrg} onChange={(e) => setSelectedOrg(e.target.value)} style={{ width: '100%' }}>
                    {orgs.map((org) => (
                        <div key={org.id} style={{ marginBottom: 8 }}>
                            <Radio value={org.id}>{org.name} {org.is_primary ? '(主机构)' : ''}</Radio>
                        </div>
                    ))}
                </Radio.Group>
            </Modal>
        </div>
    );
};

export default Login;
