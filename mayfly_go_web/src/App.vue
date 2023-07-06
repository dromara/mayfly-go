<template>
    <router-view v-show="themeConfig.lockScreenTime !== 0" />
    <LockScreen v-if="themeConfig.isLockScreen" />
    <Setings ref="setingsRef" v-show="themeConfig.lockScreenTime !== 0" />
</template>

<script setup lang="ts" name="app">
import { ref, onBeforeMount, onMounted, onUnmounted, nextTick, watch } from 'vue';
import { useRoute } from 'vue-router';
// import { useTagsViewRoutes } from '@/store/tagsViewRoutes';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { getLocal } from '@/common/utils/storage';
import LockScreen from '@/views/layout/lockScreen/index.vue';
import Setings from '@/views/layout/navBars/breadcrumb/setings.vue';
import Watermark from '@/common/utils/wartermark';
import mittBus from '@/common/utils/mitt';

const setingsRef = ref();
const route = useRoute();

const themeConfigStores = useThemeConfig();
const { themeConfig } = storeToRefs(themeConfigStores);

// 布局配置弹窗打开
const openSetingsDrawer = () => {
    setingsRef.value.openDrawer();
};

// 设置初始化，防止刷新时恢复默认
onBeforeMount(() => {
    // 设置批量第三方 icon 图标
    // setIntroduction.cssCdn();
    // // 设置批量第三方 js
    // setIntroduction.jsCdn();
});

// 页面加载时
onMounted(() => {
    nextTick(() => {
        // 监听布局配置弹窗点击打开
        mittBus.on('openSetingsDrawer', () => {
            openSetingsDrawer();
        });
        // 获取缓存中的布局配置
        if (getLocal('themeConfig')) {
            themeConfigStores.setThemeConfig({ themeConfig: getLocal('themeConfig') });
            document.documentElement.style.cssText = getLocal('themeConfigStyle');
        }
    });
});

// 页面销毁时，关闭监听布局配置
onUnmounted(() => {
    mittBus.off('openSetingsDrawer', () => {});
});

// 监听路由的变化，设置网站标题
watch(
    () => route.path,
    () => {
        nextTick(() => {
            // 路由变化更新水印
            Watermark.use();
            document.title = `${route.meta.title} - ${themeConfig.value.globalTitle}` || themeConfig.value.globalTitle;
        });
    }
);
</script>
