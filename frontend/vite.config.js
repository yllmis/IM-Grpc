import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api/user': {
        target: 'http://127.0.0.1:8888',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/user/, '')
      },
      '/api/social': {
        target: 'http://127.0.0.1:8881',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/social/, '')
      },
      '/api/im': {
        target: 'http://127.0.0.1:8882',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/im/, '')
      },
      '/ws': {
        target: 'ws://127.0.0.1:10090',
        ws: true,
        changeOrigin: true
      }
    }
  }
})
