<template>
    <el-aside class="layout-aside" :class="setCollapseWidth" v-if="state.clientWidth > 1000">
        <Logo v-if="setShowLogo" />
        <el-scrollbar class="flex-auto" ref="layoutAsideScrollbarRef">
            <Vertical :menuList="state.menuList" :class="setCollapseWidth" />
        </el-scrollbar>
    </el-aside>
    <el-drawer v-model="themeConfig.isCollapse" :with-header="false" direction="ltr" size="220px" v-else>
        <el-aside class="layout-aside w100 h100">
            <Logo v-if="setShowLogo" />
            <el-scrollbar class="flex-auto" ref="layoutAsideScrollbarRef">
                <Vertical :menuList="state.menuList" />
            </el-scrollbar>
        </el-aside>
    </el-drawer>
</template>

<script lang="ts" setup name="layoutAside">
import { reactive, computed, watch, getCurrentInstance, onBeforeMount, onUnmounted } from 'vue';
import pinia from '@/store/index';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useRoutesList } from '@/store/routesList';
import Logo from '@/layout/logo/index.vue';
import Vertical from '@/layout/navMenu/vertical.vue';
import mittBus from '@/common/utils/mitt';

const { proxy } = getCurrentInstance() as any;

const { themeConfig } = storeToRefs(useThemeConfig());
const { routesList } = storeToRefs(useRoutesList());

const state: any = reactive({
    menuList: [],
    clientWidth: '',
});

// 设置菜单展开/收起时的宽度
const setCollapseWidth = computed(() => {
    let { layout, isCollapse, menuBar } = themeConfig.value;
    let asideBrColor = menuBar === '#FFFFFF' || menuBar === '#FFF' || menuBar === '#fff' || menuBar === '#ffffff' ? 'layout-el-aside-br-color' : '';
    if (layout === 'columns') {
        // 分栏布局，菜单收起时宽度给 1px
        if (isCollapse) {
            return ['layout-aside-width1', asideBrColor];
        } else {
            return ['layout-aside-width-default', asideBrColor];
        }
    } else {
        // 其它布局给 64px
        if (isCollapse) {
            return ['layout-aside-width64', asideBrColor];
        } else {
            return ['layout-aside-width-default', asideBrColor];
        }
    }
});

// 设置显示/隐藏 logo
const setShowLogo = computed(() => {
    let { layout, isShowLogo } = themeConfig.value;
    return (isShowLogo && layout === 'defaults') || (isShowLogo && layout === 'columns');
});

// 设置/过滤路由（非静态路由/是否显示在菜单中）
const setFilterRoutes = () => {
    if (themeConfig.value.layout === 'columns') return false;
    state.menuList = filterRoutesFun(routesList.value);
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
// 设置菜单导航是否固定（移动端）
const initMenuFixed = (clientWidth: number) => {
    state.clientWidth = clientWidth;
};

// 监听 themeConfig 配置文件的变化，更新菜单 el-scrollbar 的高度
watch(themeConfig.value, (val) => {
    if (val.isShowLogoChange !== val.isShowLogo) {
        if (!proxy.$refs.layoutAsideScrollbarRef) return false;
        proxy.$refs.layoutAsideScrollbarRef.update();
    }
});

// 监听路由的变化，动态赋值给菜单中
watch(pinia.state, (val) => {
    if (val.routesList.routesList.length === state.menuList.length) return false;
    let { layout, isClassicSplitMenu } = val.themeConfig.themeConfig;
    if (layout === 'classic' && isClassicSplitMenu) return false;
    setFilterRoutes();
});

// 页面加载前
onBeforeMount(() => {
    initMenuFixed(document.body.clientWidth);
    setFilterRoutes();
    mittBus.on('setSendColumnsChildren', (res: any) => {
        state.menuList = res.children;
    });
    mittBus.on('setSendClassicChildren', (res: any) => {
        let { layout, isClassicSplitMenu } = themeConfig.value;
        if (layout === 'classic' && isClassicSplitMenu) {
            state.menuList = [];
            state.menuList = res.children;
        }
    });
    mittBus.on('getBreadcrumbIndexSetFilterRoutes', () => {
        setFilterRoutes();
    });
    mittBus.on('layoutMobileResize', (res: any) => {
        initMenuFixed(res.clientWidth);
    });
});
// 页面卸载时
onUnmounted(() => {
    mittBus.off('setSendColumnsChildren');
    mittBus.off('setSendClassicChildren');
    mittBus.off('getBreadcrumbIndexSetFilterRoutes');
    mittBus.off('layoutMobileResize');
});
</script>
