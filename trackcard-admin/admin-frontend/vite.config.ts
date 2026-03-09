import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
    plugins: [react()],
    server: {
        host: '::',
        port: 5181,
        proxy: {
            '/api/admin': {
                target: 'http://localhost:8001',
                changeOrigin: true,
            },
            '/api': {
                target: 'http://localhost:5052',
                changeOrigin: true,
            },
        },
    },
})
