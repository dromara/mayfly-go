<template>
    <router-view v-slot="{ Component }">
        <transition appear :name="setTransitionName" mode="out-in">
            <keep-alive :include="getKeepAliveNames">
                <component :is="Component" :key="state.refreshRouterViewKey" />
            </keep-alive>
        </transition>
    </router-view>
</template>

<script lang="ts" setup name="layoutParentView">
import { computed, watch, reactive, onBeforeMount, onMounted, onUnmounted, nextTick } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useKeepALiveNames } from '@/store/keepAliveNames';
import mittBus from '@/common/utils/mitt';
import { getTagViews } from '@/common/utils/storage';

const route = useRoute();
const { themeConfig } = storeToRefs(useThemeConfig());
const { keepAliveNames, cachedViews } = storeToRefs(useKeepALiveNames());

const state: any = reactive({
    refreshRouterViewKey: null,
    keepAliveNameList: [],
});

// 获取组件缓存列表(name值)
const getKeepAliveNames = computed(() => {
    return themeConfig.value.isTagsview ? cachedViews.value : state.keepAliveNameList;
});

// 页面加载前，处理缓存，页面刷新时路由缓存处理
onBeforeMount(() => {
    state.keepAliveNameList = keepAliveNames.value;
    mittBus.on('onTagsViewRefreshRouterView', (path: string) => {
        if (decodeURI(route.fullPath) !== path) return false;
        state.keepAliveNameList = keepAliveNames.value.filter((name: string) => route.name !== name);
        state.refreshRouterViewKey = '';
        nextTick(() => {
            state.refreshRouterViewKey = path;
            state.keepAliveNameList = keepAliveNames.value;
        });
    });
});
// 页面加载时
onMounted(() => {
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
// 设置主界面切换动画
const setTransitionName = computed(() => {
    return themeConfig.value.animation;
});
// 页面卸载时
onUnmounted(() => {
    mittBus.off('onTagsViewRefreshRouterView');
});
</script>
