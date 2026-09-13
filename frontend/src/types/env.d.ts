declare module '*.vue' {
    import type { DefineComponent } from 'vue';
    const component: DefineComponent<{}, {}, any>;
    export default component;
}

// vue-i18n globalInjection: true 注入的全局方法类型声明
// 确保所有组件模板中的 $t / $d / $n / $tm / $rt / $i18n 有正确类型
import type { ComposerTranslation, ComposerDateTimeFormat, ComposerNumberFormat, ComposerTranslatePlural } from 'vue-i18n';

declare module 'vue' {
    interface ComponentCustomProperties {
        $t: ComposerTranslation;
        $rt: ComposerTranslation;
        $tm: (key: string) => unknown;
        $d: ComposerDateTimeFormat;
        $n: ComposerNumberFormat;
        $i18n: {
            locale: string;
            messages: Record<string, unknown>;
        };
    }
}
