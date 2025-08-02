<template>
    <el-main class="layout-main !h-full">
        <el-scrollbar ref="layoutScrollbarRef" view-class="!h-full">
            <LayoutParentView />
        </el-scrollbar>
    </el-main>

    <el-footer v-if="themeConfig.isFooter">
        <Footer />
    </el-footer>
</template>

<script setup lang="ts" name="layoutMain">
import { getCurrentInstance, watch, defineAsyncComponent } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';

const LayoutParentView = defineAsyncComponent(() => import('@/layout/routerView/parent.vue'));
const Footer = defineAsyncComponent(() => import('@/layout/footer/index.vue'));

const { proxy } = getCurrentInstance() as any;
const { themeConfig } = storeToRefs(useThemeConfig());
const route = useRoute();

// 监听 themeConfig 配置文件的变化，更新菜单 el-scrollbar 的高度
watch(themeConfig.value, (val) => {
    if (val.isFixedHeaderChange !== val.isFixedHeader) {
        if (!proxy.$refs.layoutScrollbarRef) return false;
        proxy.$refs.layoutScrollbarRef.update();
    }
});
// 监听路由的变化
watch(
    () => route.path,
    () => {
        proxy.$refs.layoutScrollbarRef.wrapRef.scrollTop = 0;
    }
);
</script>
