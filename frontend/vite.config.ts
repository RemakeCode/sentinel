import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import wails from '@wailsio/runtime/plugins/vite';

export default defineConfig({
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true
  },
  plugins: [react(), wails('./bindings')],
  resolve: {
    tsconfigPaths: true
  },
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              test: /node_modules\/react/,
              name: 'react'
            },
            {
              test: /node_modules\/react-dom/,
              name: 'react-dom'
            },
            {
              test: /node_modules\/@wailsio\/runtime/,
              name: 'wails-runtime'
            }
          ]
        }
      }
    }
  }
});
