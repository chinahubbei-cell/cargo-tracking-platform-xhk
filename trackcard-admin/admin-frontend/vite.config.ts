import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
    plugins: [react()],
    server: {
        host: '127.0.0.1',
        port: 5181,
        proxy: {
            '/api/admin': {
                target: 'http://localhost:8001',
                changeOrigin: true,
            },
            '/api': {
                target: 'http://localhost:5051',
                changeOrigin: true,
            },
        },
    },
})
