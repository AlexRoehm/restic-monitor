import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:4711',
        changeOrigin: true,
      },
      '/agents': {
        target: 'http://localhost:4711',
        changeOrigin: true,
      },
      '/policies': {
        target: 'http://localhost:4711',
        changeOrigin: true,
      }
    }
  }
})
