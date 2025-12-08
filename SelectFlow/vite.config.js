import path from 'path';
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import cssInjectedByJsPlugin from 'vite-plugin-css-injected-by-js';
// Output to orchestrator's assets folder for direct deployment
var orchestratorAssetsPath = path.resolve(__dirname, '../microservices/orchestrator/assets');
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
        // Output to orchestrator assets folder
        outDir: orchestratorAssetsPath,
        emptyOutDir: false, // Don't clear other files in assets
        lib: {
            entry: path.resolve(__dirname, './src/main.ts'),
            name: 'SelectFlow',
            formats: ['iife'],
            fileName: function () { return 'selectflow.js'; }
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
});
