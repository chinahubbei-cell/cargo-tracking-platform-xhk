import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Card, message, Typography } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import api from '../api/client';
import { useAuthStore } from '../store/authStore';

const { Title, Text } = Typography;

const Login: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();
    const { setAuth } = useAuthStore();

    const onFinish = async (values: { email: string; password: string }) => {
        setLoading(true);
        try {
            const response: any = await api.login(values);
            // 后端返回 { success: true, data: { token, user } }
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

    return (
        <div
            style={{
                minHeight: '100vh',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            }}
        >
            <Card
                style={{
                    width: 400,
                    boxShadow: '0 8px 32px rgba(0,0,0,0.2)',
                    borderRadius: 16,
                }}
            >
                <div style={{ textAlign: 'center', marginBottom: 32 }}>
                    <Title level={2} style={{ marginBottom: 8 }}>
                        📍 TrackCard
                    </Title>
                    <Text type="secondary">全球货物追踪平台</Text>
                </div>

                <Form
                    name="login"
                    onFinish={onFinish}
                    autoComplete="off"
                    size="large"
                >
                    <Form.Item
                        name="email"
                        rules={[
                            { required: true, message: '请输入邮箱' },
                            { type: 'email', message: '请输入有效的邮箱地址' },
                        ]}
                    >
                        <Input
                            prefix={<UserOutlined />}
                            placeholder="邮箱"
                        />
                    </Form.Item>

                    <Form.Item
                        name="password"
                        rules={[
                            { required: true, message: '请输入密码' },
                            { min: 6, message: '密码至少6位' },
                        ]}
                    >
                        <Input.Password
                            prefix={<LockOutlined />}
                            placeholder="密码"
                        />
                    </Form.Item>

                    <Form.Item>
                        <Button type="primary" htmlType="submit" loading={loading} block>
                            登录
                        </Button>
                    </Form.Item>
                </Form>

                <div style={{ textAlign: 'center' }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                        默认账号：admin@trackcard.com / admin123
                    </Text>
                </div>
            </Card>
        </div>
    );
};

export default Login;
