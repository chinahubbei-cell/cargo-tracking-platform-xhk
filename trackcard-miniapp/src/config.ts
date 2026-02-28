/**
 * Global Configuration
 */

// Toggle this for production vs development
const IS_DEV = true

export const config = {
    // Use 127.0.0.1 for local dev to avoid localhost resolution issues in WeChat DevTools
    BASE_URL: IS_DEV ? 'http://127.0.0.1:5051/api' : 'https://api.yourdomain.com/api',

    // Timeout for requests (10s)
    TIMEOUT: 10000,
}

export default config
