import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';
import { defineConfig } from 'vitest/config';

export default defineConfig({
    plugins: [vue()],
    css: {
        // 与 vite.config 保持一致：内联空配置，禁用自动加载遗留的 CJS postcss.config.js（type:module 下无法加载）
        postcss: {},
    },
    test: {
        environment: 'happy-dom',
        globals: true,
    },
    resolve: {
        alias: {
            '@': resolve(__dirname, 'src/'),
        },
    },
    define: {
        __VUE_I18N_LEGACY_API__: JSON.stringify(false),
        __VUE_I18N_FULL_INSTALL__: JSON.stringify(false),
        __INTLIFY_PROD_DEVTOOLS__: JSON.stringify(false),
    },
});
