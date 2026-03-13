import React, { useState } from 'react'
import { View, Text } from '@tarojs/components'
import Taro from '@tarojs/taro'
import { Button, Input, Form } from '@nutui/nutui-react-taro'
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
            console.log('[Login] Taro.login code:', loginRes.code)

            const res: any = await AuthService.login(loginRes.code)
            console.log('[Login] AuthService.login response:', JSON.stringify(res))

            if (res.code === 'USER_NOT_FOUND') {
                Taro.showToast({ title: '请绑定已有账号', icon: 'none', duration: 2000 })
                setShowBind(true)
            } else {
                // Support both flat and nested structure
                const token = res.token || res.data?.token
                const user = res.user || res.data?.user

                if (token) {
                    Taro.setStorageSync('token', token)
                    Taro.setStorageSync('user', user)
                    Taro.showToast({ title: '登录成功', icon: 'success', duration: 1500 })
                    setTimeout(() => {
                        Taro.reLaunch({ url: '/pages/index/index' })
                    }, 1500)
                } else {
                    console.error('[Login] No token in response:', res)
                    Taro.showToast({ title: '登录异常，请重试', icon: 'none', duration: 2000 })
                }
            }
        } catch (err: any) {
            console.error('[Login] Error:', err)
            Taro.showToast({
                title: err?.message || '登录失败，请检查网络',
                icon: 'none',
                duration: 2000
            })
        } finally {
            setLoading(false)
        }
    }

    const handleBind = async () => {
        if (!email || !password) {
            Taro.showToast({ title: '请填写完整', icon: 'none' })
            return
        }
        setLoading(true)
        try {
            // Get fresh code for binding (backend needs it for Code2Session)
            const loginRes = await Taro.login()
            console.log('[Bind] Taro.login code:', loginRes.code)

            // Send code, username (email), password.
            const res: any = await AuthService.bind({
                code: loginRes.code,
                username: email,
                password
            })
            console.log('[Bind] AuthService.bind response:', JSON.stringify(res))

            // Support both flat and nested structure
            const token = res.token || res.data?.token
            const user = res.user || res.data?.user

            if (token) {
                Taro.setStorageSync('token', token)
                Taro.setStorageSync('user', user)
                Taro.showToast({ title: '绑定成功', icon: 'success', duration: 1500 })
                setTimeout(() => {
                    Taro.reLaunch({ url: '/pages/index/index' })
                }, 1500)
            } else {
                console.error('[Bind] No token in response:', res)
                Taro.showToast({ title: '绑定异常，请重试', icon: 'none', duration: 2000 })
            }
        } catch (err: any) {
            console.error('[Bind] Error:', err)
            if (err.message && (err.message.includes('Invalid email') || err.message.includes('password'))) {
                Taro.showToast({ title: '账号或密码错误', icon: 'none', duration: 2000 })
            } else {
                Taro.showToast({ title: err?.message || '绑定失败，请重试', icon: 'none', duration: 2000 })
            }
        } finally {
            setLoading(false)
        }
    }

    return (
        <View className="login-container">
            <View className="login-header">
                <Text className="login-title">全球货物追踪平台</Text>
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
                        <Text>首次使用，请绑定已有账号</Text>
                    </View>
                    <Form>
                        <Form.Item label="账号" name="email">
                            <Input
                                placeholder="请输入账号"
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
                        确认登录
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
        </View>
    )
}

export default Login
