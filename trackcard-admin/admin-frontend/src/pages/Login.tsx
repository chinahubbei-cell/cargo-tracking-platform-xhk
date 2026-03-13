import React, { useState } from 'react';
import { Form, Input, Button, Card, message, Tabs, Modal, Radio } from 'antd';
import { UserOutlined, LockOutlined, MobileOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

const Login: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [codeLoading, setCodeLoading] = useState(false);
    const [countdown, setCountdown] = useState(0);
    const [orgs, setOrgs] = useState<Array<{ id: string; name: string; is_primary?: boolean }>>([]);
    const [selectedOrg, setSelectedOrg] = useState('');
    const [orgModal, setOrgModal] = useState(false);

    const { login, loginBySMS, sendSMSCode, selectOrg } = useAuth();
    const navigate = useNavigate();
    const [smsForm] = Form.useForm();

    const onPasswordLogin = async (values: { username: string; password: string }) => {
        setLoading(true);
        const success = await login(values.username, values.password);
        setLoading(false);
        if (success) {
            message.success('登录成功');
            navigate('/');
        } else {
            message.error('用户名或密码错误');
        }
    };

    const onSMSLogin = async (values: { phone: string; code: string }) => {
        setLoading(true);
        const res = await loginBySMS(values.phone, values.code);
        setLoading(false);
        if (!res.success) {
            message.error('手机号或验证码错误');
            return;
        }
        if (res.needSelectOrg) {
            setOrgs(res.orgs || []);
            if (res.orgs?.length) setSelectedOrg(res.orgs[0].id);
            setOrgModal(true);
            return;
        }
        message.success('登录成功');
        navigate('/');
    };

    const handleSendCode = async () => {
        const phone = smsForm.getFieldValue('phone');
        if (!phone) {
            message.warning('请先输入手机号');
            return;
        }
        setCodeLoading(true);
        const res = await sendSMSCode(phone, 'login');
        setCodeLoading(false);
        if (!res.ok) {
            message.error('验证码发送失败');
            return;
        }
        if (res.debugCode) {
            message.success(`验证码已发送（开发模式）：${res.debugCode}`);
        } else {
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
    };

    const confirmOrg = async () => {
        if (!selectedOrg) return;
        setLoading(true);
        const ok = await selectOrg(selectedOrg);
        setLoading(false);
        if (ok) {
            setOrgModal(false);
            message.success('机构选择成功');
            navigate('/');
        } else {
            message.error('机构选择失败');
        }
    };

    return (
        <div style={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' }}>
            <Card style={{ width: 420, boxShadow: '0 8px 32px rgba(0,0,0,0.15)' }} bordered={false}>
                <div style={{ textAlign: 'center', marginBottom: 20 }}>
                    <h1 style={{ fontSize: 24, fontWeight: 600, color: '#1890ff' }}>TrackCard 管理后台</h1>
                    <p style={{ color: '#8c8c8c' }}>请登录您的管理员账号</p>
                </div>

                <Tabs
                    defaultActiveKey="pwd"
                    items={[
                        {
                            key: 'pwd',
                            label: '用户名密码',
                            children: (
                                <Form name="login-pwd" onFinish={onPasswordLogin} autoComplete="off" size="large">
                                    <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
                                        <Input prefix={<UserOutlined />} placeholder="用户名/邮箱" />
                                    </Form.Item>
                                    <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
                                        <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                                    </Form.Item>
                                    <Form.Item><Button type="primary" htmlType="submit" loading={loading} block>登录</Button></Form.Item>
                                </Form>
                            ),
                        },
                        {
                            key: 'sms',
                            label: '手机号验证码',
                            children: (
                                <Form form={smsForm} name="login-sms" onFinish={onSMSLogin} autoComplete="off" size="large">
                                    <Form.Item name="phone" rules={[{ required: true, message: '请输入手机号' }, { pattern: /^1\d{10}$/, message: '请输入11位大陆手机号' }]}>
                                        <Input prefix={<MobileOutlined />} placeholder="手机号（+86）" />
                                    </Form.Item>
                                    <Form.Item>
                                        <Input.Group compact>
                                            <Form.Item name="code" noStyle rules={[{ required: true, message: '请输入验证码' }, { len: 6, message: '验证码6位' }]}>
                                                <Input style={{ width: '60%' }} placeholder="验证码" />
                                            </Form.Item>
                                            <Button style={{ width: '40%' }} onClick={handleSendCode} disabled={countdown > 0} loading={codeLoading}>
                                                {countdown > 0 ? `${countdown}s后重试` : '发送验证码'}
                                            </Button>
                                        </Input.Group>
                                    </Form.Item>
                                    <Form.Item><Button type="primary" htmlType="submit" loading={loading} block>登录</Button></Form.Item>
                                </Form>
                            ),
                        },
                    ]}
                />
            </Card>

            <Modal title="选择登录机构" open={orgModal} onOk={confirmOrg} onCancel={() => setOrgModal(false)} confirmLoading={loading}>
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
