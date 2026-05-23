import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      },
      '/v1': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      },
      '/.well-known': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      },
      '/downloads': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      },
      '/shasums': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      },
    },
  },
})
