import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  css: {
    preprocessorOptions: {
      scss: {
        // example: additionalData: `@use "@/assets/styles/element-variables.scss" as *;`,
      },
    },
  },
  server: {
    port: 3000, // Development server port
    proxy: {
      // Proxy API requests to the backend (running on port 8080)
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        // No rewrite needed if backend routes also start with /api
      },
    },
  },
})
