import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'gl-matrix': ['./src/utils/gl_matrix.js'],
          'renderer': ['./src/renderer/particle_renderer.js'],
          'camera': ['./src/camera/quaternion_camera.js', './src/camera/mouse_controls.js', './src/camera/keyboard_controls.js']
        }
      }
    }
  }
});
