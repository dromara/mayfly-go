import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';
import type { UserConfig } from 'vite';
import { loadEnv } from './src/common/utils/viteBuild';
import { CodeInspectorPlugin } from 'code-inspector-plugin';
import progress from 'vite-plugin-progress';
import tailwindcss from '@tailwindcss/vite';

const pathResolve = (dir: string): any => {
    return resolve(__dirname, '.', dir);
};

const { VITE_PORT, VITE_OPEN, VITE_PUBLIC_PATH, VITE_EDITOR } = loadEnv();

const isProd = process.env.NODE_ENV === 'production';

const alias: Record<string, string> = {
    '@': pathResolve('src/'),
};

const viteConfig: UserConfig = {
    plugins: [
        vue(),
        tailwindcss(),
        CodeInspectorPlugin({
            bundler: 'vite',
            editor: VITE_EDITOR as any,
        }),
        progress(),
    ],
    root: process.cwd(),
    resolve: {
        alias,
    },
    base: isProd ? VITE_PUBLIC_PATH : './',
    optimizeDeps: {
        include: ['element-plus/es/locale/lang/zh-cn'],
    },
    server: {
        host: '0.0.0.0',
        port: VITE_PORT,
        open: VITE_OPEN,
        proxy: {
            '/api': {
                target: 'http://localhost:18888',
                ws: true,
                changeOrigin: true,
            },
        },
    },
    build: {
        outDir: 'dist',
        chunkSizeWarningLimit: 1500,
        rollupOptions: {
            output: {
                entryFileNames: `assets/[hash]-[name].js`,
                chunkFileNames: `assets/[hash]-[name].js`,
                assetFileNames: `assets/[hash]-[name].[ext]`,
                hashCharacters: 'hex',
                advancedChunks: {
                    groups: [
                        { name: 'vue-vendor', test: /[\\/]node_modules[\\/](vue|@vue|vue-router|pinia)[\\/]/ },
                        { name: 'echarts', test: /(echarts)/i },
                        { name: 'monaco', test: /(monaco-editor)/i },
                    ],
                },
            },
        },
    },
    define: {
        __VUE_I18N_LEGACY_API__: JSON.stringify(false),
        __VUE_I18N_FULL_INSTALL__: JSON.stringify(false),
        __INTLIFY_PROD_DEVTOOLS__: JSON.stringify(false),
    },
    css: {
        postcss: {
            plugins: [
                {
                    postcssPlugin: 'internal:charset-removal',
                    AtRule: {
                        charset: (atRule) => {
                            if (atRule.name === 'charset') {
                                atRule.remove();
                            }
                        },
                    },
                },
            ],
        },
    },
};

export default viteConfig;
