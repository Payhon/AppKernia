import react from '@vitejs/plugin-react'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [tanstackRouter({ target: 'react', autoCodeSplitting: true }), react()],
  server: {
    port: 4173,
    proxy: {
      '/admin-api': 'http://127.0.0.1:8080',
    },
    strictPort: true,
  },
  build: {
    manifest: true,
    sourcemap: true,
    target: 'es2023',
  },
})
