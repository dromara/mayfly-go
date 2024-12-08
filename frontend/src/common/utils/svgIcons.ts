import { nextTick } from 'vue';
import * as svg from '@element-plus/icons-vue';
import iconfontJson from '@/assets/iconfont/iconfont.json';
import SvgIcon from '@/components/svgIcon/index.vue';

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

// 获取阿里字体图标
const getAlicdnIconfont = () => {
    return new Promise((resolve, reject) => {
        nextTick(() => {
            const styles: any = document.styleSheets;
            let sheetsList = [];
            let sheetsIconList = [];
            for (let i = 0; i < styles.length; i++) {
                console.log(styles[i]);
                if (styles[i].href && styles[i].href.indexOf('iconfont') > -1) {
                    sheetsList.push(styles[i]);
                }
            }
            for (let i = 0; i < sheetsList.length; i++) {
                for (let j = 0; j < sheetsList[i].cssRules.length; j++) {
                    if (sheetsList[i].cssRules[j].selectorText && sheetsList[i].cssRules[j].selectorText.indexOf('.icon-') > -1) {
                        sheetsIconList.push(
                            `${sheetsList[i].cssRules[j].selectorText.substring(1, sheetsList[i].cssRules[j].selectorText.length).replace(/\:\:before/gi, '')}`
                        );
                    }
                }
            }
            if (sheetsIconList.length > 0) resolve(sheetsIconList);
            else reject('未获取到值，请刷新重试');
        });
    });
};

// 获取本地阿里icons
const getLocalAliIconfont = () => {
    return new Promise((resolve, reject) => {
        nextTick(() => {
            const prefix = iconfontJson.css_prefix_text;
            resolve(iconfontJson.glyphs.map((x: any) => prefix + x.font_class));
        });
    });
};

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

// 初始化获取 css 样式，这里使用 fontawesome 的图标
const awesomeIconfont = () => {
    return new Promise((resolve, reject) => {
        nextTick(() => {
            const styles: any = document.styleSheets;
            let sheetsList = [];
            let sheetsIconList = [];
            for (let i = 0; i < styles.length; i++) {
                if (styles[i].href && styles[i].href.indexOf('netdna.bootstrapcdn.com') > -1) {
                    sheetsList.push(styles[i]);
                }
            }
            for (let i = 0; i < sheetsList.length; i++) {
                for (let j = 0; j < sheetsList[i].cssRules.length; j++) {
                    if (
                        sheetsList[i].cssRules[j].selectorText &&
                        sheetsList[i].cssRules[j].selectorText.indexOf('.fa-') === 0 &&
                        sheetsList[i].cssRules[j].selectorText.indexOf(',') === -1
                    ) {
                        sheetsIconList.push(
                            `${sheetsList[i].cssRules[j].selectorText.substring(1, sheetsList[i].cssRules[j].selectorText.length).replace(/\:\:before/gi, '')}`
                        );
                    }
                }
            }
            if (sheetsIconList.length > 0) resolve(sheetsIconList);
            else reject('未获取到值，请刷新重试');
        });
    });
};

// 定义导出方法集合
const initIconfont = {
    ali: () => {
        return getLocalAliIconfont();
    },
    ele: () => {
        return elementPlusIconfont();
    },
    // awe: () => {
    // 	return awesomeIconfont();
    // },
};

// 导出方法
export default initIconfont;
