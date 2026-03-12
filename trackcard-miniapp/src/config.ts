/**
 * Global Configuration
 *
 * 优先读取环境变量 TARO_APP_API_BASE_URL，避免小程序和PC指向不同后端导致“数据不同步”。
 * - 本地开发默认: http://127.0.0.1:5051/api
 * - 生产默认:     https://trackcard.kuaihuoyun.com/api
 */

const DEFAULT_DEV_API = 'http://127.0.0.1:5051/api'
const DEFAULT_PROD_API = 'https://trackcard.kuaihuoyun.com/api'

const isDev = process.env.NODE_ENV !== 'production'
const envApiBase = process.env.TARO_APP_API_BASE_URL

export const config = {
    BASE_URL: envApiBase || (isDev ? DEFAULT_DEV_API : DEFAULT_PROD_API),

    // Timeout for requests (10s)
    TIMEOUT: 10000,
}

export default config
