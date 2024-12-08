import { createApp } from 'vue';
import App from '@/App.vue';

import router from './router';
import pinia from '@/store/index';
import { directive } from '@/directive/index';
import { registElSvgIcon } from '@/common/utils/svgIcons';

import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import 'element-plus/theme-chalk/dark/css-vars.css';
import { ElMessage } from 'element-plus';
import { i18n } from '@/i18n/index';

import 'splitpanes/dist/splitpanes.css';

import '@/theme/index.scss';
import '@/assets/font/font.css';
import '@/assets/iconfont/iconfont.js';
import { getThemeConfig } from './common/utils/storage';

const app = createApp(App);

registElSvgIcon(app);
directive(app);

app.use(pinia).use(router).use(i18n).use(ElementPlus, { size: getThemeConfig()?.globalComponentSize }).mount('#app');

// 屏蔽警告信息
app.config.warnHandler = () => null;
// 全局error处理
app.config.errorHandler = function (err: any, vm, info) {
    // 如果是断言错误，则进行提示即可
    if (err.name == 'AssertError') {
        ElMessage.error(err.message);
    } else {
        console.error(err, info);
    }
};
