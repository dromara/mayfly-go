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
        // 挂载真实 Element Plus 组件的用例在全量并跑时会超过默认 5s（隔离跑 1.5s，全量偶发超时），
        // 抬高上限是为了消掉这类负载抖动误报，而不是容忍慢用例
        testTimeout: 15_000,
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
