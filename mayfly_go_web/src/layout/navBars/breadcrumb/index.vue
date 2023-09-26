<template>
    <div class="layout-navbars-breadcrumb-index">
        <Logo v-if="setIsShowLogo" />
        <Breadcrumb />
        <Horizontal :menuList="state.menuList" v-if="isLayoutTransverse" />
        <User />
    </div>
</template>

<script lang="ts" setup name="layoutBreadcrumbIndex">
import { computed, reactive, onMounted, onUnmounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import pinia from '@/store/index';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useRoutesList } from '@/store/routesList';
import Breadcrumb from '@/layout/navBars/breadcrumb/breadcrumb.vue';
import User from '@/layout/navBars/breadcrumb/user.vue';
import Logo from '@/layout/logo/index.vue';
import Horizontal from '@/layout/navMenu/horizontal.vue';
import mittBus from '@/common/utils/mitt';

const { themeConfig } = storeToRefs(useThemeConfig());
const { routesList } = storeToRefs(useRoutesList());
const route = useRoute();
const state: any = reactive({
    menuList: [],
});

// 设置 logo 显示/隐藏
const setIsShowLogo = computed(() => {
    let { isShowLogo, layout } = themeConfig.value;
    return (isShowLogo && layout === 'classic') || (isShowLogo && layout === 'transverse');
});
// 设置是否显示横向导航菜单
const isLayoutTransverse = computed(() => {
    let { layout, isClassicSplitMenu } = themeConfig.value;
    return layout === 'transverse' || (isClassicSplitMenu && layout === 'classic');
});
// 设置/过滤路由（非静态路由/是否显示在菜单中）
const setFilterRoutes = () => {
    let { layout, isClassicSplitMenu } = themeConfig.value;
    if (layout === 'classic' && isClassicSplitMenu) {
        state.menuList = delClassicChildren(filterRoutesFun(routesList.value));
        const resData = setSendClassicChildren(route.path);
        mittBus.emit('setSendClassicChildren', resData);
    } else {
        state.menuList = filterRoutesFun(routesList.value);
    }
};
// 设置了分割菜单时，删除底下 children
const delClassicChildren = (arr: Array<object>) => {
    arr.map((v: any) => {
        if (v.children) delete v.children;
    });
    return arr;
};
// 路由过滤递归函数
const filterRoutesFun = (arr: Array<object>) => {
    return arr
        .filter((item: any) => !item.meta.isHide)
        .map((item: any) => {
            item = Object.assign({}, item);
            if (item.children) item.children = filterRoutesFun(item.children);
            return item;
        });
};
// 传送当前子级数据到菜单中
const setSendClassicChildren = (path: string) => {
    const currentPathSplit = path.split('/');
    let currentData: any = {};
    filterRoutesFun(routesList.value).map((v, k) => {
        if (v.path === `/${currentPathSplit[1]}`) {
            v['k'] = k;
            currentData['item'] = [{ ...v }];
            currentData['children'] = [{ ...v }];
            if (v.children) currentData['children'] = v.children;
        }
    });
    return currentData;
};
// 监听路由的变化，动态赋值给菜单中
watch(pinia.state, (val) => {
    if (val.routesList.routesList.length === state.menuList.length) return false;
    setFilterRoutes();
});
// 页面加载时
onMounted(() => {
    setFilterRoutes();
    mittBus.on('getBreadcrumbIndexSetFilterRoutes', () => {
        setFilterRoutes();
    });
});
// 页面卸载时
onUnmounted(() => {
    mittBus.off('getBreadcrumbIndexSetFilterRoutes');
});
</script>

<style scoped lang="scss">
.layout-navbars-breadcrumb-index {
    height: 50px;
    display: flex;
    align-items: center;
    padding-right: 15px;
    background: var(--bg-topBar);
    overflow: hidden;
    border-bottom: 1px solid var(--el-border-color-light, #ebeef5);
}
</style>
