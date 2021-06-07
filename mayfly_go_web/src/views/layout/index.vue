<template>
    <Defaults v-if="getThemeConfig.layout === 'defaults'" />
    <Classic v-else-if="getThemeConfig.layout === 'classic'" />
    <Transverse v-else-if="getThemeConfig.layout === 'transverse'" />
    <Columns v-else-if="getThemeConfig.layout === 'columns'" />
</template>

<script lang="ts">
import { computed, onBeforeMount, onUnmounted, getCurrentInstance } from 'vue';
import { useStore } from '@/store/index.ts';
import { getLocal, setLocal } from '@/common/utils/storage.ts';
import Defaults from '@/views/layout/main/defaults.vue';
import Classic from '@/views/layout/main/classic.vue';
import Transverse from '@/views/layout/main/transverse.vue';
import Columns from '@/views/layout/main/columns.vue';
export default {
    name: 'layout',
    components: { Defaults, Classic, Transverse, Columns },
    setup() {
        const { proxy } = getCurrentInstance() as any;
        const store = useStore();
        // 获取布局配置信息
        const getThemeConfig = computed(() => {
            return store.state.themeConfig.themeConfig;
        });
        // 窗口大小改变时(适配移动端)
        const onLayoutResize = () => {
            if (!getLocal('oldLayout')) setLocal('oldLayout', getThemeConfig.value.layout);
            const clientWidth = document.body.clientWidth;
            if (clientWidth < 1000) {
                getThemeConfig.value.isCollapse = false;
                proxy.mittBus.emit('layoutMobileResize', {
                    layout: 'defaults',
                    clientWidth,
                });
            } else {
                proxy.mittBus.emit('layoutMobileResize', {
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
        return {
            getThemeConfig,
        };
    },
};
</script>
