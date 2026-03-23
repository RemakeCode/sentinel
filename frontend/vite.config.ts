import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import wails from '@wailsio/runtime/plugins/vite';

export default defineConfig({
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
