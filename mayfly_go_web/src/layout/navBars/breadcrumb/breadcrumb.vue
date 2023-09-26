<template>
    <div class="layout-navbars-breadcrumb" v-show="themeConfig.isBreadcrumb">
        <SvgIcon class="layout-navbars-breadcrumb-icon" :name="themeConfig.isCollapse ? 'expand' : 'fold'"
            @click="onThemeConfigChange" />
        <el-breadcrumb class="layout-navbars-breadcrumb-hide">
            <transition-group name="breadcrumb" mode="out-in">
                <el-breadcrumb-item v-for="(v, k) in state.breadcrumbList" :key="v.meta.title">
                    <span v-if="k === state.breadcrumbList.length - 1" class="layout-navbars-breadcrumb-span">
                        <SvgIcon :name="v.meta.icon" class="layout-navbars-breadcrumb-iconfont"
                            v-if="themeConfig.isBreadcrumbIcon" />
                        {{ v.meta.title }}
                    </span>
                    <a v-else @click.prevent="onBreadcrumbClick(v)">
                        <SvgIcon :name="v.meta.icon" class="layout-navbars-breadcrumb-iconfont"
                            v-if="themeConfig.isBreadcrumbIcon" />
                        {{ v.meta.title }}
                    </a>
                </el-breadcrumb-item>
            </transition-group>
        </el-breadcrumb>
    </div>
</template>

<script lang="ts" setup name="layoutBreadcrumb">
import { reactive, onMounted } from 'vue';
import { onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useRoutesList } from '@/store/routesList';
import mittBus from '@/common/utils/mitt';

const { themeConfig } = storeToRefs(useThemeConfig());
const { routesList } = storeToRefs(useRoutesList());
const route = useRoute();
const router = useRouter();
const state: any = reactive({
    breadcrumbList: [],
    routeSplit: [],
    routeSplitFirst: '',
    routeSplitIndex: 1,
});

// 面包屑点击时
const onBreadcrumbClick = (v: any) => {
    const { redirect, path } = v;
    if (redirect) router.push(redirect);
    else router.push(path);
};
// 展开/收起左侧菜单点击
const onThemeConfigChange = () => {
    mittBus.emit('onMenuClick');
    themeConfig.value.isCollapse = !themeConfig.value.isCollapse;
};
// 处理面包屑数据
const getBreadcrumbList = (arr: Array<object>) => {
    arr.map((item: any) => {
        state.routeSplit.map((v: any, k: number, arrs: any) => {
            if (state.routeSplitFirst === item.path) {
                state.routeSplitFirst += `/${arrs[state.routeSplitIndex]}`;
                state.breadcrumbList.push(item);
                state.routeSplitIndex++;
                if (item.children) getBreadcrumbList(item.children);
            }
        });
    });
};
// 当前路由字符串切割成数组，并删除第一项空内容
const initRouteSplit = (path: string) => {
    if (!themeConfig.value.isBreadcrumb) return false;
    state.breadcrumbList = [routesList.value[0]];
    state.routeSplit = path.split('/');
    state.routeSplit.shift();
    state.routeSplitFirst = `/${state.routeSplit[0]}`;
    state.routeSplitIndex = 1;
    getBreadcrumbList(routesList.value);
};
// 页面加载时
onMounted(() => {
    initRouteSplit(route.path);
});
// 路由更新时
onBeforeRouteUpdate((to) => {
    initRouteSplit(to.path);
});
</script>

<style scoped lang="scss">
.layout-navbars-breadcrumb {
    flex: 1;
    height: inherit;
    display: flex;
    align-items: center;
    padding-left: 15px;

    .layout-navbars-breadcrumb-icon {
        cursor: pointer;
        font-size: 18px;
        margin-right: 15px;
        color: var(--bg-topBarColor);
    }

    .layout-navbars-breadcrumb-span {
        opacity: 0.7;
        color: var(--bg-topBarColor);
    }

    .layout-navbars-breadcrumb-iconfont {
        font-size: 14px;
        margin-right: 5px;
    }

    ::v-deep(.el-breadcrumb__separator) {
        opacity: 0.7;
        color: var(--bg-topBarColor);
    }
}
</style>
