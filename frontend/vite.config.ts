import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://127.0.0.1:8080',
        ws: true,
      },
    },
  },
  build: {
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        manualChunks: {
          // Core React runtime — tiny, always cached
          'vendor-react': ['react', 'react-dom'],
          // Charting library — large, but changes rarely
          'vendor-charts': ['recharts'],
          // Icon set — large SVG bundle
          'vendor-icons': ['lucide-react'],
          // State management
          'vendor-state': ['zustand'],
        },
      },
    },
  },
});
