<template>
    <el-header class="layout-header" :height="setHeaderHeight">
        <NavBarsIndex />
    </el-header>
</template>

<script lang="ts">
import { computed } from 'vue';
import { useStore } from '@/store/index.ts';
import NavBarsIndex from '@/views/layout/navBars/index.vue';
export default {
    name: 'layoutHeader',
    components: { NavBarsIndex },
    setup() {
        const store = useStore();
        // 设置 header 的高度
        const setHeaderHeight = computed(() => {
            let { isTagsview, layout } = store.state.themeConfig.themeConfig;
            if (isTagsview && layout !== 'classic') return '84px';
            else return '50px';
        });
        return {
            setHeaderHeight,
        };
    },
};
</script>
