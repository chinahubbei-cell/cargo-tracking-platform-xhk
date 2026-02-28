import React, { useState } from 'react'
import { View, Text } from '@tarojs/components'
import Taro from '@tarojs/taro'
import { Button, Input, Toast, Form } from '@nutui/nutui-react-taro'
import { AuthService } from '../../services/api'
import './index.css'

function Login() {
    const [showBind, setShowBind] = useState(false)
    const [loading, setLoading] = useState(false)

    // Bind form state
    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')

    const handleWeChatLogin = async () => {
        setLoading(true)
        try {
            const loginRes = await Taro.login()
            console.log('WeChat Code:', loginRes.code)

            const res: any = await AuthService.login(loginRes.code)
            console.log('Auth Res:', res)

            if (res.code === 'USER_NOT_FOUND') {
                Toast.show('请绑定已有账号')
                // No need to store openid, backend handles it via code
                setShowBind(true)
            } else {
                // Support both flat and nested structure
                const token = res.token || res.data?.token
                const user = res.user || res.data?.user

                if (token) {
                    Taro.setStorageSync('token', token)
                    Taro.setStorageSync('user', user)
                    Toast.show({ title: '登录成功', icon: 'success' })
                    setTimeout(() => {
                        Taro.reLaunch({ url: '/pages/index/index' })
                    }, 1000)
                }
            }
        } catch (err) {
            console.error(err)
            // Error handling is managed by request interceptor/wrapper
        } finally {
            setLoading(false)
        }
    }

    const handleBind = async () => {
        if (!email || !password) {
            Toast.show('请填写完整')
            return
        }
        setLoading(true)
        try {
            // Get fresh code for binding (backend needs it for Code2Session)
            const loginRes = await Taro.login()
            console.log('Bind Code:', loginRes.code)

            // Send code, username (email), password.
            const res: any = await AuthService.bind({
                code: loginRes.code,
                username: email,
                password
            })

            console.log('Bind Res:', res)

            // Support both flat and nested structure
            const token = res.token || res.data?.token
            const user = res.user || res.data?.user

            if (token) {
                console.log('Bind Success, Token:', token)
                Taro.setStorageSync('token', token)
                Taro.setStorageSync('user', user)
                Toast.show({ title: '绑定成功', icon: 'success' })
                setTimeout(() => {
                    // Switch tab because index is a tab bar page now
                    Taro.switchTab({ url: '/pages/index/index' })
                        .catch(() => {
                            // Fallback if not a tab bar page
                            Taro.reLaunch({ url: '/pages/index/index' })
                        })
                }, 1000)
            } else {
                console.warn('Bind Success but NO Token found in response:', res)
                Toast.show('绑定异常，请重试')
            }
        } catch (err: any) {
            console.error('Bind Error:', err)
            // request.ts throws Error with message from backend
            if (err.message && (err.message.includes('Invalid email') || err.message.includes('password'))) {
                Toast.show('账号或密码错误')
            } else {
                Toast.show('绑定失败，请重试')
            }
        } finally {
            setLoading(false)
        }
    }

    return (
        <View className="login-container">
            <View className="login-header">
                <Text className="login-title">Cargo Tracking</Text>
            </View>

            {!showBind ? (
                <Button
                    type="primary"
                    block
                    loading={loading}
                    onClick={handleWeChatLogin}
                >
                    微信一键登录
                </Button>
            ) : (
                <View>
                    <View className="bind-header">
                        <Text>绑定已有账号</Text>
                    </View>
                    <Form>
                        <Form.Item label="邮箱" name="email">
                            <Input
                                placeholder="请输入邮箱"
                                value={email}
                                onChange={(val: any) => setEmail(String(val))}
                            />
                        </Form.Item>
                        <Form.Item label="密码" name="password">
                            <Input
                                placeholder="请输入密码"
                                type="password"
                                value={password}
                                onChange={(val: any) => setPassword(String(val))}
                            />
                        </Form.Item>
                    </Form>

                    <Button
                        type="info"
                        block
                        className="bind-confirm-btn"
                        loading={loading}
                        onClick={handleBind}
                    >
                        确认绑定
                    </Button>
                    <Button
                        fill="none"
                        block
                        className="bind-back-btn"
                        onClick={() => setShowBind(false)}
                    >
                        返回
                    </Button>
                </View>
            )}
            <Toast />
        </View>
    )
}

export default Login
