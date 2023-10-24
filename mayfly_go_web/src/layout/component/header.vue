<template>
    <el-header class="layout-header" :height="setHeaderHeight">
        <NavBarsIndex />
    </el-header>
</template>

<script lang="ts">
import { computed } from 'vue';
import NavBarsIndex from '@/layout/navBars/index.vue';
import { useThemeConfig } from '@/store/themeConfig';
export default {
    name: 'layoutHeader',
    components: { NavBarsIndex },
    setup() {
        // 设置 header 的高度
        const setHeaderHeight = computed(() => {
            let { isTagsview, layout } = useThemeConfig().themeConfig;
            if (isTagsview && layout !== 'classic') return '84px';
            else return '50px';
        });
        return {
            setHeaderHeight,
        };
    },
};
</script>
