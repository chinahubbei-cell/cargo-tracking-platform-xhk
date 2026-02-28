import Taro from '@tarojs/taro'
import { config } from '../config'

// Define standard response structures
interface ApiResponse<T = any> {
    code?: number | string
    message?: string
    error?: string
    data?: T
    token?: string
    user?: any
    [key: string]: any // Fallback for various backend formats
}

// Interceptor to add token
const interceptor = function (chain) {
    const requestParams = chain.requestParams
    const token = Taro.getStorageSync('token')

    if (token) {
        requestParams.header = {
            ...requestParams.header,
            Authorization: `Bearer ${token}`
        }
    }

    return chain.proceed(requestParams).then(res => {
        // Handle 401 Unauthorized globally
        if (res.statusCode === 401) {
            Taro.removeStorageSync('token')
            // Don't redirect immediately if already on login page to avoid loops
            const pages = Taro.getCurrentPages()
            const currentPage = pages[pages.length - 1]
            if (currentPage && currentPage.route !== 'pages/login/index') {
                Taro.reLaunch({ url: '/pages/login/index' })
                Taro.showToast({ title: '登录已过期', icon: 'none' })
            }
        }
        return res
    })
}

Taro.addInterceptor(interceptor)

export const request = async <T = any>(
    url: string,
    method: 'GET' | 'POST' | 'PUT' | 'DELETE',
    data?: any
): Promise<T> => {
    try {
        const res = await Taro.request({
            url: `${config.BASE_URL}${url}`,
            method,
            data,
            header: {
                'Content-Type': 'application/json'
            },
            timeout: config.TIMEOUT,
        })

        if (res.statusCode >= 200 && res.statusCode < 300) {
            // Check for business logic errors in 200 responses if any
            // Some backends return 200 but with { code: "ERROR", ... }
            const body = res.data as ApiResponse
            if (body && body.code && body.code !== 200 && body.code !== 'SUCCESS') {
                // But wait, our login handler returns { code: "USER_NOT_FOUND" } which is valid logic flow, not exception.
                // So we should return the body and let caller handle specific codes.
                return res.data as T
            }
            return res.data as T
        } else {
            // Handle HTTP errors
            const body = res.data as ApiResponse
            // Extract the most relevant error message
            // Priority: error field > message field > standard status text
            const errMsg = body?.error || body?.message || `请求失败 (${res.statusCode})`

            Taro.showToast({
                title: errMsg,
                icon: 'none',
                duration: 2000
            })

            // Throw error with message for caller to catch if needed
            throw new Error(errMsg)
        }
    } catch (err) {
        console.error('Request Error:', err)
        // If it's a network error (not thrown by above logic)
        if (err.message && err.message.indexOf('request:fail') !== -1) {
            Taro.showToast({
                title: '网络连接失败',
                icon: 'none'
            })
        }
        throw err
    }
}

export default request
