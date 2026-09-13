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
                escapeTags: ['el-config-provider'],
            }),
            progress(),
        ],
        optimizeDeps: {
            // logicflow 仅被懒加载的 flow 路由组件引用，启动扫描发现不了；首次进入流程菜单时才在运行时被识别，
            // 触发依赖重新预构建使旧 chunk 失效，浏览器报 504 Outdated Optimize Dep 并导致路由动态 import 失败。
            // 显式 include 使其随启动预构建，消除运行时重新优化。
            include: ['element-plus/es/locale/lang/zh-cn', '@logicflow/core', '@logicflow/extension'],
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
            sourcemap: false,
            chunkSizeWarningLimit: 1500,
            rolldownOptions: {
                output: {
                    entryFileNames: `assets/js/[hash]-[name].js`,
                    chunkFileNames: `assets/js/[hash]-[name].js`,
                    assetFileNames: `assets/[ext]/[hash]-[name].[ext]`,
                    hashCharacters: 'hex',
                    advancedChunks: {
                        groups: [
                            // 各分组必须显式给 priority：rolldown 未标注优先级的组会输给标注过的组，
                            // 一旦 preload-helper 落到别组，它又会被并入体积最大的 monaco（见下方注释）。
                            // Vite 的动态导入预加载 helper 会被每个含 import() 的 chunk 静态引用。
                            // 不单独分组时它会被并入体积最大的共享 chunk（monaco），
                            // 于是 login/main/layout 等 60 余个与编辑器无关的页面都被迫加载整份 monaco。
                            { name: 'preload-helper', test: /vite\/preload-helper/, priority: 140 },
                            { name: 'vue-vendor', test: /[\\/]node_modules[\\/](vue|@vue|vue-router|pinia|vue-i18n|@intlify)[\\/]/, priority: 130 },
                            { name: 'charts', test: /[\\/]node_modules[\\/](echarts)[\\/]/, priority: 130 },
                            // monaco 语言包是「只给 globalThis 赋消息表」的纯副作用模块，须先于 monaco 求值（见 main.ts 首行），
                            // 无法挪到 monaco 的加载路径里。不单独分组时它会被并入整份 monaco，使启动页被迫下载 4M+ 编辑器代码。
                            { name: 'monaco-nls', test: /[\\/]node_modules[\\/]monaco-editor[\\/]esm[\\/]vs[\\/]nls[\\/]/, priority: 120 },
                            { name: 'monaco', test: /[\\/]node_modules[\\/]monaco-editor[\\/]/, priority: 110 },
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
                            charset: (atRule: { name: string; remove: () => void }) => {
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
