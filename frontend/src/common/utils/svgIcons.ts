import { nextTick } from 'vue';
import * as svg from '@element-plus/icons-vue';
import SvgIcon from '@/components/svgIcon/index.vue';
import { getLocalIcons } from '@/assets/icon/icon.js';

/**
 * 导出全局注册 element plus svg 图标
 * @param app vue 实例
 * @description 使用：https://element-plus.gitee.io/zh-CN/component/icon.html
 */
export function registElSvgIcon(app: any) {
    const icons = svg as any;
    for (const i in icons) {
        app.component(`${icons[i].name}`, icons[i]);
    }
    app.component('SvgIcon', SvgIcon);
}

// 初始化获取 css 样式，获取 element plus 自带图标
const elementPlusIconfont = () => {
    return new Promise((resolve, reject) => {
        nextTick(() => {
            const icons = svg as any;
            const sheetsIconList = [];
            for (const i in icons) {
                sheetsIconList.push(`${icons[i].name}`);
            }
            if (sheetsIconList.length > 0) resolve(sheetsIconList);
            else reject('未获取到值，请刷新重试');
        });
    });
};

// 定义导出方法集合
const initIconfont = {
    ele: () => {
        return elementPlusIconfont();
    },
    other: () => {
        return getLocalIcons();
    },
};

// 导出方法
export default initIconfont;
