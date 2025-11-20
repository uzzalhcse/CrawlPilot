import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  build: {
    outDir: '../internal/browser/dist',
    assetsDir: '',
    cssCodeSplit: false,
    lib: {
      entry: path.resolve(__dirname, './src/main.ts'),
      name: 'CrawlifySelector',
      formats: ['iife'],
      fileName: () => 'selector-overlay.js'
    },
    rollupOptions: {
      output: {
        inlineDynamicImports: true,
        assetFileNames: 'selector-overlay.[ext]'
      }
    },
    minify: true
  },
  define: {
    'process.env.NODE_ENV': JSON.stringify('production')
  }
})
