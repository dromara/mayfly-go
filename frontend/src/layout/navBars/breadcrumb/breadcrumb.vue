<template>
    <div class="flex flex-1 h-inherit items-center pl-4" v-show="themeConfig.isBreadcrumb">
        <SvgIcon
            class="cursor-pointer text-18px mr-4 text-[var(--bg-topBarColor)]"
            :name="themeConfig.isCollapse ? 'expand' : 'fold'"
            @click="onThemeConfigChange"
        />
        <el-breadcrumb class="layout-navbars-breadcrumb-hide">
            <transition-group name="breadcrumb">
                <el-breadcrumb-item v-for="(v, k) in state.breadcrumbList" :key="v.meta.title">
                    <span v-if="k === state.breadcrumbList.length - 1 || (!v.redirect && !v.component)" class="opacity-70 text-[var(--bg-topBarColor)]">
                        <SvgIcon :name="v.meta.icon" class="text-14px mr-1.25" v-if="themeConfig.isBreadcrumbIcon" />
                        {{ $t(v.meta.title) }}
                    </span>
                    <a v-else @click.prevent="onBreadcrumbClick(v)" class="opacity-100 text-[var(--bg-topBarColor)] hover:opacity-100">
                        <SvgIcon :name="v.meta.icon" class="text-14px mr-1.25" v-if="themeConfig.isBreadcrumbIcon" />
                        {{ $t(v.meta.title) }}
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

const { themeConfig } = storeToRefs(useThemeConfig());
const { routesList } = storeToRefs(useRoutesList());
const route = useRoute();
const router = useRouter();
const state = reactive({
    breadcrumbList: [] as any[],
});

// 面包屑点击时
const onBreadcrumbClick = (v: any) => {
    const { redirect, path } = v;
    if (redirect) {
        router.push(redirect);
        return;
    }
    if (v.component) {
        router.push(path);
    }
};
// 展开/收起左侧菜单点击
const onThemeConfigChange = () => {
    themeConfig.value.isCollapse = !themeConfig.value.isCollapse;
};

// 根据当前路径生成面包屑列表
const generateBreadcrumbList = (currentPath: string) => {
    if (!themeConfig.value.isBreadcrumb) {
        return;
    }

    // 初始化面包屑列表，包含首页
    const homeRoute = routesList.value.length > 0 ? routesList.value[0] : null;
    const breadcrumbList = homeRoute ? [homeRoute] : [];

    // 查找匹配的路由及其所有父级路由（除了首页）
    if (homeRoute && currentPath !== homeRoute.path) {
        const matchedRoutes = findMatchedRoutes(routesList.value, currentPath);
        // 如果找到匹配的路由，添加到面包屑列表中（排除首页，避免重复）
        if (matchedRoutes.length > 0) {
            // 过滤掉首页路由，避免重复添加
            const filteredRoutes = matchedRoutes.filter((r) => r !== homeRoute);
            breadcrumbList.push(...filteredRoutes);
        }
    }

    state.breadcrumbList = breadcrumbList;
};

// 在路由树中查找匹配当前路径的路由，并返回该路由及其所有父级路由
const findMatchedRoutes = (routes: any[], currentPath: string): any[] => {
    for (const route of routes) {
        // 精确匹配
        if (route.path === currentPath) {
            return [route];
        }

        // 前缀匹配且有子路由
        if (currentPath.startsWith(route.path + '/') && route.children) {
            const matchedChildren = findMatchedRoutes(route.children, currentPath);
            if (matchedChildren.length > 0) {
                return [route, ...matchedChildren];
            }
        }

        // 处理子路由匹配但当前路由是根路径的情况
        if (route.path === '/' && route.children) {
            const matchedChildren = findMatchedRoutes(route.children, currentPath);
            if (matchedChildren.length > 0) {
                return [route, ...matchedChildren];
            }
        }

        // 递归查找子路由
        if (route.children) {
            const matchedChildren = findMatchedRoutes(route.children, currentPath);
            if (matchedChildren.length > 0) {
                return [route, ...matchedChildren];
            }
        }
    }

    return [];
};

// 页面加载时
onMounted(() => {
    generateBreadcrumbList(route.path);
});
// 路由更新时
onBeforeRouteUpdate((to) => {
    generateBreadcrumbList(to.path);
});
</script>

<style scoped>
::v-deep(.el-breadcrumb__separator) {
    opacity: 0.7;
    color: var(--bg-topBarColor);
}
</style>
