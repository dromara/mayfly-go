import { createApp } from 'vue';
import App from '@/App.vue';
import router from './router';
import pinia from '@/store/index';
import { directive } from '@/directive/index';
import { globalComponentSize } from '@/common/utils/componentSize';

import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import '@/theme/index.scss';
import { ElMessage } from 'element-plus';

import * as svg from '@element-plus/icons-vue';
import SvgIcon from '@/components/svgIcon/index.vue';
import '@/assets/font/font.css'

const app = createApp(App);

/**
 * 导出全局注册 element plus svg 图标
 * @param app vue 实例
 * @description 使用：https://element-plus.gitee.io/zh-CN/component/icon.html
 */
function elSvg(app: any) {
    const icons = svg as any;
    for (const i in icons) {
        app.component(`${icons[i].name}`, icons[i]);
    }
    app.component('SvgIcon', SvgIcon);
}

elSvg(app);
directive(app);

app.use(pinia)
    .use(router)
    .use(ElementPlus, { size: globalComponentSize, locale: zhCn })
    .mount('#app');

// 屏蔽警告信息
app.config.warnHandler = () => null;
// 全局error处理
app.config.errorHandler = function (err: any, vm, info) {
    // 如果是断言错误，则进行提示即可
    if (err.name == 'AssertError') {
        ElMessage.error(err.message)
    } else {
        console.error(err, info)
    }
}
