<template>
    <router-view v-slot="{ Component }">
        <transition appear :name="themeConfig.animation" mode="out-in">
            <keep-alive :include="getKeepAliveNames">
                <component :is="Component" :key="state.refreshRouterViewKey" v-show="!isIframePage" />
            </keep-alive>
        </transition>
    </router-view>

    <transition :name="themeConfig.animation" mode="out-in">
        <Iframes class="w-full" v-show="isIframePage" :refreshKey="state.iframeRefreshKey" :name="themeConfig.animation" :list="state.iframes" />
    </transition>
</template>

<script lang="ts" setup name="layoutParentView">
import { computed, watch, reactive, onBeforeMount, onMounted, nextTick, defineAsyncComponent } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useKeepALiveNames } from '@/store/keepAliveNames';
import { getTagViews } from '@/common/utils/storage';
import { useTagsViews } from '@/store/tagsViews';
import { LinkTypeEnum } from '@/common/commonEnum';

const Iframes = defineAsyncComponent(() => import('@/layout/routerView/iframes.vue'));

const route = useRoute();
const router = useRouter();
const { themeConfig } = storeToRefs(useThemeConfig());
const { keepAliveNames, cachedViews } = storeToRefs(useKeepALiveNames());

const state = reactive({
    refreshRouterViewKey: '',
    keepAliveNameList: [] as any[],
    iframeRefreshKey: '', // iframe tagsview 右键菜单刷新时
    iframes: [] as any[],
});

const { currentRefreshPath } = storeToRefs(useTagsViews());

// 获取组件缓存列表(name值)
const getKeepAliveNames = computed(() => {
    return themeConfig.value.isTagsview ? cachedViews.value : state.keepAliveNameList;
});

// 设置 iframe 显示/隐藏
const isIframePage = computed(() => {
    return route.meta.linkType == LinkTypeEnum.Iframes.value;
});

watch(currentRefreshPath, (path) => {
    if (decodeURI(route.fullPath) !== path) {
        return;
    }
    state.keepAliveNameList = keepAliveNames.value.filter((name: string) => route.name !== name);
    state.refreshRouterViewKey = '';
    state.iframeRefreshKey = '';
    nextTick(() => {
        state.refreshRouterViewKey = path;
        state.iframeRefreshKey = path;
        state.keepAliveNameList = keepAliveNames.value;
    });
    useTagsViews().setCurrentRefreshPath('');
});

// 页面加载前，处理缓存，页面刷新时路由缓存处理
onBeforeMount(() => {
    state.keepAliveNameList = keepAliveNames.value;
});

// 页面加载时
onMounted(() => {
    getIframesRoutes();
    nextTick(() => {
        setTimeout(() => {
            if (themeConfig.value.isCacheTagsView) {
                let tagsViewArr: any = getTagViews() || [];
                cachedViews.value = tagsViewArr.filter((item: any) => item?.isKeepAlive).map((item: any) => item.name as string);
            }
        }, 0);
    });
});

// 监听路由变化，防止 tagsView 多标签时，切换动画消失
watch(
    () => route.fullPath,
    () => {
        state.refreshRouterViewKey = decodeURI(route.fullPath);
    },
    {
        immediate: true,
    }
);

// 获取 iframe 组件列表(未进行渲染)
const getIframesRoutes = async () => {
    router.getRoutes().forEach((v) => {
        if (v.meta.linkType === LinkTypeEnum.Iframes.value) {
            v.meta.isIframeOpen = false;
            v.meta.loading = true;
            state.iframes.push({ ...v });
        }
    });
};
</script>
