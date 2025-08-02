<template>
    <div class="w-[64px] h-full bg-[var(--bg-columnsMenuBar)]">
        <el-scrollbar>
            <ul class="relative">
                <li
                    v-for="(v, k) in state.columnsAsideList"
                    :key="k"
                    @click="onColumnsAsideMenuClick(v, k)"
                    :ref="
                        (el) => {
                            if (el) columnsAsideOffsetTopRefs[k] = el;
                        }
                    "
                    :class="[
                        { 'text-white': state.liIndex === k },
                        'color-[var(--bg-columnsMenuBarColor)] w-full h-[50px] text-center flex cursor-pointer relative z-[1] transition-[color] duration-300 ease-in-out',
                    ]"
                    :title="$t(v.meta.title)"
                >
                    <div class="mx-auto my-auto" v-if="!v.meta.link || (v.meta.link && v.meta.linkType == 1)">
                        <i :class="v.meta.icon"></i>
                        <div class="pt-[1px] !text-[12px]">
                            {{ $t(v.meta.title) && $t(v.meta.title).length >= 4 ? $t(v.meta.title).substring(0, 4) : $t(v.meta.title) }}
                        </div>
                    </div>
                    <div class="mx-auto my-auto" v-else>
                        <a :href="v.meta.link" target="_blank" class="no-underline color-[var(--bg-columnsMenuBarColor)]">
                            <i :class="v.meta.icon"></i>
                            <div class="pt-[1px] !text-[12px]">
                                {{ $t(v.meta.title) && $t(v.meta.title).length >= 4 ? $t(v.meta.title).substring(0, 4) : $t(v.meta.title) }}
                            </div>
                        </a>
                    </div>
                </li>
                <div
                    ref="columnsAsideActiveRef"
                    :class="[
                        'absolute z-[0] bg-[var(--el-color-primary)] text-white transition-all duration-300 ease-in-out',
                        setColumnsAsideStyle === 'columnsRound'
                            ? 'left-1/2 top-[2px] h-[44px] w-[58px] -translate-x-1/2 rounded-[5px]'
                            : 'left-0 top-0 h-[50px] w-full rounded-[0]',
                    ]"
                ></div>
            </ul>
        </el-scrollbar>
    </div>
</template>

<script lang="ts" setup name="layoutColumnsAside">
import { reactive, ref, computed, onMounted, nextTick, watch, inject } from 'vue';
import { useRoute, useRouter, onBeforeRouteUpdate } from 'vue-router';
import pinia from '@/store/index';
import { useThemeConfig } from '@/store/themeConfig';
import { useRoutesList } from '@/store/routesList';

const columnsAsideOffsetTopRefs: any = ref([]);
const columnsAsideActiveRef = ref();
const route = useRoute();
const router = useRouter();
const state = reactive({
    columnsAsideList: [] as any[],
    liIndex: 0,
    difference: 0,
    routeSplit: [] as any[],
});

// 注入 columnsMenuData
const columnsMenuData: any = inject('columnsMenuData');

// 设置高亮样式
const setColumnsAsideStyle = computed(() => {
    return useThemeConfig().themeConfig.columnsAsideStyle;
});

// 设置菜单高亮位置移动
const setColumnsAsideMove = (k: number) => {
    state.liIndex = k;
    columnsAsideActiveRef.value.style.top = `${columnsAsideOffsetTopRefs.value[k].offsetTop + state.difference}px`;
};

// 菜单高亮点击事件
const onColumnsAsideMenuClick = (v: any, k: number) => {
    setColumnsAsideMove(k);
    if (v.children && v.children.length > 0) {
        router.push(v.children[0].path);
    } else {
        router.push(v.path);
    }
    // if (redirect) {
    //     router.push(redirect);
    // } else {
    //     router.push(path);
    // }
};
// 设置高亮动态位置
const onColumnsAsideDown = (k: number) => {
    nextTick(() => {
        setColumnsAsideMove(k);
    });
};
// 设置/过滤路由（非静态路由/是否显示在菜单中）
const setFilterRoutes = () => {
    state.columnsAsideList = filterRoutesFun(useRoutesList().routesList);
    const resData: any = setSendChildren(route.path);
    onColumnsAsideDown(resData.item[0].k);
    if (columnsMenuData) {
        columnsMenuData.value = resData;
    }
};
// 传送当前子级数据到菜单中
const setSendChildren = (path: string) => {
    let currentData: any = {};
    const result = findRootRoute(state.columnsAsideList, path);

    if (result) {
        const k = state.columnsAsideList.findIndex((v: any) => v === result);
        if (k !== -1) {
            result['k'] = k;
            currentData['item'] = [{ ...result }];
            currentData['children'] = [{ ...result }];
            if (result.children) currentData['children'] = result.children;
        }
    }

    return currentData;
};

// 路由过滤递归函数
const filterRoutesFun = (arr: Array<object>) => {
    return arr
        .filter((item: any) => !item.meta.isHide)
        .map((item: any) => {
            item = Object.assign({}, item);
            if (item.children) {
                item.children = filterRoutesFun(item.children);
            }
            return item;
        });
};

// tagsView 点击时，根据路由查找下标 columnsAsideList，实现左侧菜单高亮
const setColumnsMenuHighlight = (path: string) => {
    const rootRoute = findRootRoute(state.columnsAsideList, path);
    if (rootRoute) {
        // 延迟拿值，防止取不到
        setTimeout(() => {
            onColumnsAsideDown(rootRoute.k);
        }, 0);
    }
};

// 递归查找路由并返回根节点
const findRootRoute = (routes: any[], currentPath: string): any => {
    for (const route of routes) {
        // 直接匹配
        if (route.path === currentPath) {
            return route;
        }

        // 在子路由中查找
        if (route.children && route.children.length > 0) {
            const found = findRootRoute(route.children, currentPath);
            if (found) {
                // 如果在子路由中找到了，返回根节点
                return route;
            }
        }
    }
    return null;
};

// 监听路由的变化，动态赋值给菜单中
watch(pinia.state, (val) => {
    val.themeConfig.themeConfig.columnsAsideStyle === 'columnsRound' ? (state.difference = 3) : (state.difference = 0);
    if (val.routesList.routesList.length === state.columnsAsideList.length) {
        return;
    }
    setFilterRoutes();
});

// 页面加载时
onMounted(() => {
    setFilterRoutes();
});

// 路由更新时
onBeforeRouteUpdate((to) => {
    setColumnsMenuHighlight(to.path);

    if (columnsMenuData) {
        columnsMenuData.value = setSendChildren(to.path);
    }
});
</script>
