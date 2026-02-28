import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
    plugins: [react()],
    server: {
        host: '::',
        port: 5181,
        proxy: {
            '/api': {
                target: 'http://localhost:5051',
                changeOrigin: true,
            },
        },
    },
})
