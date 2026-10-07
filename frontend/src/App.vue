<template>
    <el-config-provider
        :size="getGlobalComponentSize"
        :locale="getGlobalI18n"
        :button="{ autoInsertSpace: false, round: true }"
        :dialog="{ alignCenter: true, transition: 'dialog-bounce' }"
    >
        <el-watermark
            :zIndex="1"
            :width="210"
            v-if="themeConfig.isWatermark"
            :font="{ color: 'rgba(180, 180, 180, 0.3)' }"
            :content="themeConfig.watermarkText"
            class="h-full!"
        >
            <router-view />
        </el-watermark>
        <router-view v-if="!themeConfig.isWatermark" />

        <Setings />

        <!-- 全局系统通知悬浮按钮 -->
        <GlobalNotificationFab />

        <!-- 全局 AI 助手悬浮球（仅系统已配置 AI 模型时渲染） -->
        <AiAssistantFab />
    </el-config-provider>
</template>

<script setup vapor lang="ts" name="app">
import { onMounted, nextTick, watch, computed, defineAsyncComponent } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useIntervalFn } from '@vueuse/core';
import { useI18n } from 'vue-i18n';
import EnumValue from './common/Enum';
import { I18nEnum } from './common/commonEnum';
import { saveThemeConfig } from './common/utils/storage';
import { useEscapeClosesDrawer } from '@/hooks/useEscapeClosesDrawer';
import GlobalNotificationFab from '@/components/system-message/GlobalNotificationFab.vue';

const Setings = defineAsyncComponent(() => import('@/layout/navBars/breadcrumb/setings.vue'));
const AiAssistantFab = defineAsyncComponent(() => import('@/views/ai/AiAssistantFab.vue'));

const route = useRoute();

const themeConfigStores = useThemeConfig();
const { themeConfig } = storeToRefs(themeConfigStores);

// 定义变量内容
const { locale, t } = useI18n();

// 焦点停在 select 上时 Escape 会被 select 掐掉，抽屉因此关不掉（全站共用一条接缝）
useEscapeClosesDrawer();

// 页面加载时
onMounted(() => {
    nextTick(() => {
        // 初始化系统主题
        themeConfigStores.initThemeConfig();
    });
});

// 监听 themeConfig isWartermark配置文件的变化
watch(
    () => themeConfig.value.isWatermark,
    (val) => {
        if (val) {
            setTimeout(() => {
                setWatermarkContent();
                refreshWatermarkTime();
                resume();
            }, 500);
        } else {
            pause();
        }
    }
);

watch(
    () => themeConfig.value.globalI18n,
    (val) => {
        locale.value = val;
    }
);

watch(
    themeConfig,
    (val) => {
        saveThemeConfig(val);
    },
    { deep: true }
);

// 获取全局组件大小
const getGlobalComponentSize = computed(() => {
    return themeConfig.value.globalComponentSize;
});

// 获取全局 i18n
const getGlobalI18n = computed(() => {
    return EnumValue.getEnumByValue(I18nEnum, locale.value)?.extra.el;
});

// 刷新水印时间
const { pause, resume } = useIntervalFn(() => {
    if (!themeConfig.value.isWatermark) {
        pause();
    }
    refreshWatermarkTime();
}, 60000);

const setWatermarkContent = () => {
    themeConfigStores.setWatermarkUser();
};

/**
 * 刷新水印时间
 */
const refreshWatermarkTime = () => {
    themeConfigStores.setWatermarkNowTime();
};

// 监听路由与语言的变化，设置网站标题
//
// 语言也得进依赖：标题是翻译出来的，只监听 route.path 时切完语言标签标题还停在旧语言，
// 要等下一次跳转才更新
watch(
    [() => route.path, locale],
    ([path, language]) => {
        // 读屏软件靠它选发音规则，写死 zh_CN 会让英文界面下也用中文引擎念
        document.documentElement.lang = language;
        if (path) {
            nextTick(() => {
                document.title = `${t((route.meta.title as string) || '')} - ${themeConfig.value.globalTitle}` || themeConfig.value.globalTitle;
            });
        }
    },
    { immediate: true }
);
</script>
