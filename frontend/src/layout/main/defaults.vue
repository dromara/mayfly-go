<template>
    <el-container class="layout-container">
        <Aside />
        <el-container class="flex-center layout-backtop">
            <Header v-if="isFixedHeader" />
            <Header v-if="!isFixedHeader" />
            <Main />
        </el-container>
    </el-container>
</template>

<script lang="ts" setup name="layoutDefaults">
import { computed, defineAsyncComponent, getCurrentInstance, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useThemeConfig } from '@/store/themeConfig';

const Aside = defineAsyncComponent(() => import('@/layout/component/aside.vue'));
const Header = defineAsyncComponent(() => import('@/layout/component/header.vue'));
const Main = defineAsyncComponent(() => import('@/layout/component/main.vue'));

const { proxy } = getCurrentInstance() as any;
const route = useRoute();
const isFixedHeader = computed(() => {
    return useThemeConfig().themeConfig.isFixedHeader;
});
// 监听路由的变化
watch(
    () => route.path,
    () => {
        try {
            proxy.$refs.layoutScrollbarRef.wrapRef.scrollTop = 0;
        } catch (e) {}
    }
);
</script>
