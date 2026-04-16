import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist',
    lib: {
      entry: 'src/index.tsx',
      formats: ['es'],
      fileName: 'index'
    },
    rollupOptions: {
      external: ['react', 'react-dom', 'decky-frontend-lib'],
      output: {
        globals: {
          'react': 'React',
          'react-dom': 'ReactDOM',
          'decky-frontend-lib': 'deckyFrontendLib'
        }
      }
    },
    // Minimizing helps keep the plugin zip small
    minify: 'esbuild'
  }
});
