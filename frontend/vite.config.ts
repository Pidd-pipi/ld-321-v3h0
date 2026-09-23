import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 18621,
    proxy: {
      '/api': 'http://localhost:19621',
      '/ws': { target: 'ws://localhost:19621', ws: true },
    },
  },
  preview: { host: '0.0.0.0', port: 18621 },
});
