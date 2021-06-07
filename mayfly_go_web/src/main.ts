import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import { store, key } from './store';
import { directive } from '@/common/utils/directive.ts';
import { globalComponentSize } from '@/common/utils/componentSize.ts';
import { dateStrFormat } from '@/common/utils/date.ts'

import ElementPlus from 'element-plus';
import 'element-plus/lib/theme-chalk/index.css';
import '@/theme/index.scss';
import mitt from 'mitt';
import { ElMessage } from 'element-plus';
import locale from 'element-plus/lib/locale/lang/zh-cn'

const app = createApp(App);

app.use(router)
    .use(store, key)
    .use(ElementPlus, { size: globalComponentSize, locale: locale })
    .mount('#app');


// 自定义全局过滤器
app.config.globalProperties.$filters = {
    dateFormat(value: any) {
        if (!value) {
            return ""
        }
        return dateStrFormat('yyyy-MM-dd HH:mm:ss', value)
    }
}

// 全局error处理
app.config.errorHandler = function (err: any, vm, info) {
    // 如果是断言错误，则进行提示即可
    if (err.name == 'AssertError') {
        ElMessage.error(err.message)
    } else {
        console.error(err, info)
    }
}

app.config.globalProperties.mittBus = mitt();

directive(app);
