import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';
import type { UserConfig } from 'vite';
import { loadEnv } from './src/common/utils/viteBuild.ts';

const pathResolve = (dir: string): any => {
	return resolve(__dirname, '.', dir);
};

const { VITE_PORT, VITE_OPEN, VITE_PUBLIC_PATH } = loadEnv();

const alias: Record<string, string> = {
	'@': pathResolve('src/'),
};

const viteConfig: UserConfig = {
	plugins: [vue()],
	root: process.cwd(),
	resolve: { alias },
	base: process.env.NODE_ENV === 'production' ? VITE_PUBLIC_PATH : './',
	optimizeDeps: {
		include: ['element-plus/es/locale/lang/zh-cn'],
	},
	server: {
		host: '0.0.0.0',
		port: VITE_PORT,
		open: VITE_OPEN,
		proxy: {
			'/api': {
				target: 'http://localhost:8888',
				ws: true,
				changeOrigin: true,
			},
		},
	},
	build: {
		outDir: 'dist',
		minify: 'esbuild',
		sourcemap: false,
        chunkSizeWarningLimit: 1500,
		rollupOptions: {
			output: {
				entryFileNames: `assets/[name].${new Date().getTime()}.js`,
				chunkFileNames: `assets/[name].${new Date().getTime()}.js`,
				assetFileNames: `assets/[name].${new Date().getTime()}.[ext]`,
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
                }
              }
            }
          ]
        }
    },
};

export default viteConfig;
