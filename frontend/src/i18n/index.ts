// 定义语言国际化内容
/**
 * 说明：
 * 注意国际化定义的字段，不要与原有的定义字段相同。
 * /src/i18n/(zh-cn、en...)/module.ts 下的 ts 为各模块国际化内容。
 */
import { createI18n } from 'vue-i18n';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import pinia from '@/store';
import { I18nEnum } from '@/common/commonEnum';

const modules: Record<string, any> = import.meta.glob('./**/*.ts', { eager: true });

// 读取 pinia 默认语言
const { themeConfig } = storeToRefs(useThemeConfig(pinia));

function initI18n() {
    // 定义变量内容
    const messages: any = {};
    const itemizeMap = new Map<string, any[]>();

    // 对自动引入的 modules 进行分类 en、zh-cn
    // https://vitejs.cn/vite3-cn/guide/features.html#glob-import
    for (const path in modules) {
        const parts = path.split('/');
        const i18n = parts[1];

        const msgs = modules[path].default;
        if (itemizeMap.get(i18n)) {
            itemizeMap.get(i18n)?.push(modules[path].default);
        } else {
            itemizeMap.set(i18n, [msgs]);
        }
    }

    // 处理最终格式
    itemizeMap.forEach((value, key) => {
        messages[key] = Object.assign({}, ...value);
    });

    // https://vue-i18n.intlify.dev/guide/essentials/fallback.html#explicit-fallback-with-one-locale
    return createI18n({
        legacy: false,
        silentTranslationWarn: true,
        missingWarn: false,
        silentFallbackWarn: true,
        fallbackWarn: false,
        locale: themeConfig.value.globalI18n,
        fallbackLocale: I18nEnum.ZhCn.value,
        messages,
    });
}

// 导出语言国际化
export const i18n = initI18n();
