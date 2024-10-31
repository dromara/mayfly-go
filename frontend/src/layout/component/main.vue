<template>
    <el-main class="layout-main">
        <el-scrollbar ref="layoutScrollbarRef" v-show="!state.currentRouteMeta.link && state.currentRouteMeta.linkType != 1">
            <LayoutParentView />
        </el-scrollbar>

        <Link class="h100" :meta="state.currentRouteMeta" v-if="state.currentRouteMeta.link && state.currentRouteMeta.linkType == 2" />

        <Iframes
            class="h100"
            :meta="state.currentRouteMeta"
            v-if="state.currentRouteMeta.link && state.currentRouteMeta.linkType == 1 && state.isShowLink"
            @getCurrentRouteMeta="onGetCurrentRouteMeta"
        />
    </el-main>

    <el-footer v-if="themeConfig.isFooter">
        <Footer />
    </el-footer>
</template>

<script setup lang="ts" name="layoutMain">
import { reactive, getCurrentInstance, watch, onBeforeMount } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import LayoutParentView from '@/layout/routerView/parent.vue';
import Footer from '@/layout/footer/index.vue';
import Link from '@/layout/routerView/link.vue';
import Iframes from '@/layout/routerView/iframes.vue';

const { proxy } = getCurrentInstance() as any;
const { themeConfig } = storeToRefs(useThemeConfig());
const route = useRoute();
const state = reactive({
    headerHeight: '',
    currentRouteMeta: {} as any,
    isShowLink: false,
});

// 子组件触发更新
const onGetCurrentRouteMeta = () => {
    initCurrentRouteMeta(route.meta);
};
// 初始化当前路由 meta 信息
const initCurrentRouteMeta = (meta: object) => {
    state.isShowLink = false;
    state.currentRouteMeta = meta;
    setTimeout(() => {
        state.isShowLink = true;
    }, 100);
};
// 设置 main 的高度
const initHeaderHeight = () => {
    let { isTagsview } = themeConfig.value;
    if (isTagsview) return (state.headerHeight = `77px`);
    else return (state.headerHeight = `50px`);
};
// 页面加载前
onBeforeMount(() => {
    initCurrentRouteMeta(route.meta);
    initHeaderHeight();
});
// 监听 themeConfig 配置文件的变化，更新菜单 el-scrollbar 的高度
watch(themeConfig.value, (val) => {
    state.headerHeight = val.isTagsview ? '77px' : '50px';
    if (val.isFixedHeaderChange !== val.isFixedHeader) {
        if (!proxy.$refs.layoutScrollbarRef) return false;
        proxy.$refs.layoutScrollbarRef.update();
    }
});
// 监听路由的变化
watch(
    () => route.path,
    () => {
        initCurrentRouteMeta(route.meta);
        proxy.$refs.layoutScrollbarRef.wrapRef.scrollTop = 0;
    }
);
</script>
