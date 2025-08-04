<template>
    <el-config-provider :size="getGlobalComponentSize" :locale="getGlobalI18n">
        <el-watermark
            :zIndex="100000"
            :width="210"
            v-if="themeConfig.isWatermark"
            :font="{ color: 'rgba(180, 180, 180, 0.3)' }"
            :content="themeConfig.watermarkText"
            class="!h-full"
        >
            <router-view />
        </el-watermark>
        <router-view v-if="!themeConfig.isWatermark" />

        <Setings />
    </el-config-provider>
</template>

<script setup lang="ts" name="app">
import { onMounted, nextTick, watch, computed, defineAsyncComponent } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useIntervalFn } from '@vueuse/core';
import { useI18n } from 'vue-i18n';
import EnumValue from './common/Enum';
import { I18nEnum } from './common/commonEnum';
import { saveThemeConfig } from './common/utils/storage';

const Setings = defineAsyncComponent(() => import('@/layout/navBars/breadcrumb/setings.vue'));

const route = useRoute();

const themeConfigStores = useThemeConfig();
const { themeConfig } = storeToRefs(themeConfigStores);

// 定义变量内容
const { locale, t } = useI18n();

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

// 监听路由的变化，设置网站标题
watch(
    () => route.path,
    () => {
        nextTick(() => {
            document.title = `${t((route.meta.title as string) || '')} - ${themeConfig.value.globalTitle}` || themeConfig.value.globalTitle;
        });
    }
);
</script>
