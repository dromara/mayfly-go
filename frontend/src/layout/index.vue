<template>
    <Defaults v-if="themeConfig.layout === 'defaults'" />
    <Classic v-else-if="themeConfig.layout === 'classic'" />
    <Transverse v-else-if="themeConfig.layout === 'transverse'" />
    <Columns v-else-if="themeConfig.layout === 'columns'" />
</template>

<script setup lang="ts" name="layout">
import { onBeforeMount, onUnmounted } from 'vue';
import { getLocal, setLocal } from '@/common/utils/storage';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import Defaults from '@/layout/main/defaults.vue';
import Classic from '@/layout/main/classic.vue';
import Transverse from '@/layout/main/transverse.vue';
import Columns from '@/layout/main/columns.vue';
import mittBus from '@/common/utils/mitt';

const { themeConfig } = storeToRefs(useThemeConfig());

// 窗口大小改变时(适配移动端)
const onLayoutResize = () => {
    if (!getLocal('oldLayout')) setLocal('oldLayout', themeConfig.value.layout);
    const clientWidth = document.body.clientWidth;
    if (clientWidth < 1000) {
        themeConfig.value.isCollapse = false;
        mittBus.emit('layoutMobileResize', {
            layout: 'defaults',
            clientWidth,
        });
    } else {
        mittBus.emit('layoutMobileResize', {
            layout: getLocal('oldLayout') ? getLocal('oldLayout') : 'defaults',
            clientWidth,
        });
    }
};
// 页面加载前
onBeforeMount(() => {
    onLayoutResize();
    window.addEventListener('resize', onLayoutResize);
});
// 页面卸载时
onUnmounted(() => {
    window.removeEventListener('resize', onLayoutResize);
});
</script>
