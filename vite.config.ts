import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000, // Frontend dev server port
    proxy: {
      '/api': {
        target: 'http://localhost:8080', // Backend server address
        changeOrigin: true,
        // rewrite: (path) => path.replace(/^\/api/, '/api'), // Usually not needed if backend prefix is also /api
      },
    },
  },
})
