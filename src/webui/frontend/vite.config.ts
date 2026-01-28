import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../dist',
    emptyOutDir: true,
  },
  clearScreen: false,
  server: {
    // Configure HMR to connect directly to Vite when accessed via Go proxy
    hmr: {
      host: 'localhost',
      port: 5173,
      protocol: 'ws',
    },
    proxy: {
      // Proxy Connect RPC requests to the Go API server
      '/critic.v1.CriticService': {
        target: 'http://localhost:65432',
        changeOrigin: true,
      },
      // Proxy WebSocket connections (for app, not HMR)
      '/ws': {
        target: 'ws://localhost:65432',
        ws: true,
      },
    },
  },
})
