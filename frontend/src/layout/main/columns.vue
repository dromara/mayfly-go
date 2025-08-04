<template>
    <el-container class="layout-container">
        <ColumnsAside />
        <div class="layout-columns-warp">
            <Aside />
            <el-container class="flex-center layout-backtop">
                <Header v-if="isFixedHeader" />
                <Header v-if="!isFixedHeader" />
                <Main />
            </el-container>
        </div>
    </el-container>
</template>

<script lang="ts" setup name="layoutColumns">
import { computed, defineAsyncComponent, provide, ref } from 'vue';
import { useThemeConfig } from '@/store/themeConfig';

const Aside = defineAsyncComponent(() => import('@/layout/component/aside.vue'));
const Header = defineAsyncComponent(() => import('@/layout/component/header.vue'));
const Main = defineAsyncComponent(() => import('@/layout/component/main.vue'));
const ColumnsAside = defineAsyncComponent(() => import('@/layout/component/columnsAside.vue'));

// 提供响应式数据给子组件
const columnsMenuData = ref<any>(null);
provide('columnsMenuData', columnsMenuData);

const isFixedHeader = computed(() => {
    return useThemeConfig().themeConfig.isFixedHeader;
});
</script>
