import path from 'path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import cssInjectedByJsPlugin from 'vite-plugin-css-injected-by-js'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [vue(), cssInjectedByJsPlugin()],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src'),
        },
    },
    build: {
        cssCodeSplit: false,
        lib: {
            entry: path.resolve(__dirname, './src/main.ts'),
            name: 'SelectFlow',
            formats: ['iife'],
            fileName: () => 'selectflow.js'
        },
        rollupOptions: {
            output: {
                inlineDynamicImports: true,
            }
        },
        minify: true
    },
    define: {
        'process.env.NODE_ENV': JSON.stringify('production')
    }
})
