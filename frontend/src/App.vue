<template>
    <el-config-provider :size="getGlobalComponentSize" :locale="getGlobalI18n">
        <div class="h-full">
            <el-watermark
                :zIndex="10000000"
                :width="210"
                v-if="themeConfig.isWatermark"
                :font="{ color: 'rgba(180, 180, 180, 0.3)' }"
                :content="themeConfig.watermarkText"
                class="!h-full"
            >
                <router-view v-show="themeConfig.lockScreenTime !== 0" />
            </el-watermark>
            <router-view v-if="!themeConfig.isWatermark" v-show="themeConfig.lockScreenTime !== 0" />

            <LockScreen v-if="themeConfig.isLockScreen" />
            <Setings ref="setingsRef" v-show="themeConfig.lockScreenTime !== 0" />
        </div>
    </el-config-provider>
</template>

<script setup lang="ts" name="app">
import { ref, onMounted, onUnmounted, nextTick, watch, computed } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import LockScreen from '@/layout/lockScreen/index.vue';
import Setings from '@/layout/navBars/breadcrumb/setings.vue';
import mittBus from '@/common/utils/mitt';
import { useIntervalFn } from '@vueuse/core';
import { useI18n } from 'vue-i18n';
import EnumValue from './common/Enum';
import { I18nEnum } from './common/commonEnum';
import { saveThemeConfig } from './common/utils/storage';

const setingsRef = ref();
const route = useRoute();

const themeConfigStores = useThemeConfig();
const { themeConfig } = storeToRefs(themeConfigStores);

// 定义变量内容
const { locale, t } = useI18n();

// 布局配置弹窗打开
const openSetingsDrawer = () => {
    setingsRef.value.openDrawer();
};

// 页面加载时
onMounted(() => {
    nextTick(() => {
        // 监听布局配置弹窗点击打开
        mittBus.on('openSetingsDrawer', () => {
            openSetingsDrawer();
        });

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

// 页面销毁时，关闭监听布局配置
onUnmounted(() => {
    mittBus.off('openSetingsDrawer', () => {});
});

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
