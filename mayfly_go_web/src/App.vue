<template>
    <div class="h100">
        <el-watermark
            :zIndex="10000000"
            :width="210"
            v-if="themeConfig.isWatermark"
            :font="{ color: 'rgba(180, 180, 180, 0.5)' }"
            :content="themeConfig.watermarkText"
            class="h100"
        >
            <router-view v-show="themeConfig.lockScreenTime !== 0" />
        </el-watermark>
        <router-view v-if="!themeConfig.isWatermark" v-show="themeConfig.lockScreenTime !== 0" />

        <LockScreen v-if="themeConfig.isLockScreen" />
        <Setings ref="setingsRef" v-show="themeConfig.lockScreenTime !== 0" />
    </div>
</template>

<script setup lang="ts" name="app">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import LockScreen from '@/layout/lockScreen/index.vue';
import Setings from '@/layout/navBars/breadcrumb/setings.vue';
import mittBus from '@/common/utils/mitt';
import { useIntervalFn } from '@vueuse/core';

const setingsRef = ref();
const route = useRoute();

const themeConfigStores = useThemeConfig();
const { themeConfig } = storeToRefs(themeConfigStores);

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
            document.title = `${route.meta.title} - ${themeConfig.value.globalTitle}` || themeConfig.value.globalTitle;
        });
    }
);
</script>
