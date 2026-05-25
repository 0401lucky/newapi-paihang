import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  build: {
    target: 'es2020',
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'index.html'),
        embed: path.resolve(__dirname, 'embed.html'),
        admin: path.resolve(__dirname, 'admin.html'),
      },
    },
    assetsInlineLimit: 1024,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/admin': 'http://localhost:8080',
    },
  },
})
