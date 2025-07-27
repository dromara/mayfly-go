import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';
import { CodeInspectorPlugin } from 'code-inspector-plugin';
import progress from 'vite-plugin-progress';
import tailwindcss from '@tailwindcss/vite';
import { ConfigEnv, defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }: ConfigEnv) => {
    const env = loadEnv(mode, process.cwd(), '');
    const isProd = process.env.NODE_ENV === 'production';
    const pathResolve = (dir: string): any => {
        return resolve(__dirname, '.', dir);
    };

    return {
        base: isProd ? env.VITE_PUBLIC_PATH : './',
        resolve: {
            alias: {
                '@': pathResolve('src/'),
            },
        },
        plugins: [
            vue(),
            tailwindcss(),
            CodeInspectorPlugin({
                bundler: 'vite',
                editor: env.VITE_EDITOR as any,
            }),
            progress(),
        ],
        optimizeDeps: {
            include: ['element-plus/es/locale/lang/zh-cn'],
        },
        server: {
            host: '0.0.0.0',
            port: Number.parseInt(env.VITE_PORT) || 8889,
            open: env.VITE_OPEN === 'true',
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
                    entryFileNames: `assets/js/[hash]-[name].js`,
                    chunkFileNames: `assets/js/[hash]-[name].js`,
                    assetFileNames: `assets/[ext]/[hash]-[name].[ext]`,
                    hashCharacters: 'hex',
                    advancedChunks: {
                        groups: [
                            { name: 'vue-vendor', test: /[\\/]node_modules[\\/](vue|@vue|vue-router|pinia)[\\/]/ },
                            { name: 'charts', test: /[\\/]node_modules[\\/](echarts)[\\/]/ },
                            { name: 'monaco', test: /[\\/]node_modules[\\/]monaco-editor[\\/]/ },
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
});
