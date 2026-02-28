import React, { useState } from 'react';
import { Form, Input, Button, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

const Login: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const { login } = useAuth();
    const navigate = useNavigate();

    const onFinish = async (values: { username: string; password: string }) => {
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

    return (
        <div style={{
            minHeight: '100vh',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        }}>
            <Card
                style={{ width: 400, boxShadow: '0 8px 32px rgba(0,0,0,0.15)' }}
                bordered={false}
            >
                <div style={{ textAlign: 'center', marginBottom: 32 }}>
                    <h1 style={{ fontSize: 24, fontWeight: 600, color: '#1890ff' }}>
                        TrackCard 管理后台
                    </h1>
                    <p style={{ color: '#8c8c8c' }}>请登录您的管理员账号</p>
                </div>

                <Form
                    name="login"
                    onFinish={onFinish}
                    autoComplete="off"
                    size="large"
                >
                    <Form.Item
                        name="username"
                        rules={[{ required: true, message: '请输入用户名' }]}
                    >
                        <Input
                            prefix={<UserOutlined />}
                            placeholder="用户名"
                        />
                    </Form.Item>

                    <Form.Item
                        name="password"
                        rules={[{ required: true, message: '请输入密码' }]}
                    >
                        <Input.Password
                            prefix={<LockOutlined />}
                            placeholder="密码"
                        />
                    </Form.Item>

                    <Form.Item>
                        <Button
                            type="primary"
                            htmlType="submit"
                            loading={loading}
                            block
                        >
                            登录
                        </Button>
                    </Form.Item>
                </Form>

                <div style={{ textAlign: 'center', color: '#bfbfbf', fontSize: 12 }}>
                    默认账号: admin / admin123
                </div>
            </Card>
        </div>
    );
};

export default Login;
