import { createApp } from 'vue';
import App from '@/App.vue';

import router from './router';
import pinia from '@/store/index';
import { directive } from '@/directive/index';
import { registElSvgIcon } from '@/common/utils/svgIcons';

import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import 'element-plus/theme-chalk/dark/css-vars.css';
import { i18n } from '@/i18n/index';

import '@/theme/index.scss';
import '@/theme/tailwind.css';
import '@/assets/font/font.css';
import '@/assets/icon/icon.js';
import { getThemeConfig } from './common/utils/storage';
import { initSysMsgs } from './common/sysmsgs';

const app = createApp(App);

registElSvgIcon(app);
directive(app);
initSysMsgs();

app.use(pinia).use(router).use(i18n).use(ElementPlus, { size: getThemeConfig()?.globalComponentSize }).mount('#app');

// 屏蔽警告信息
app.config.warnHandler = () => null;
