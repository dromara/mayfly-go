<template>
    <div class="h100">
        <router-view v-slot="{ Component }">
            <transition :name="setTransitionName" mode="out-in">
                <keep-alive :include="state.keepAliveNameList">
                    <component :is="Component" :key="state.refreshRouterViewKey" class="w100" />
                </keep-alive>
            </transition>
        </router-view>
    </div>
</template>

<script lang="ts" setup name="layoutParentView">
import { computed, reactive, onBeforeMount, onUnmounted, nextTick } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useKeepALiveNames } from '@/store/keepAliveNames';
import mittBus from '@/common/utils/mitt';

const route = useRoute();
const { themeConfig } = storeToRefs(useThemeConfig());
const { keepAliveNames } = storeToRefs(useKeepALiveNames());
const state: any = reactive({
    refreshRouterViewKey: null,
    keepAliveNameList: [],
    keepAliveNameNewList: [],
});
// 监听路由的变化，动态赋值给refreshRouterViewKey
// onBeforeRouteUpdate((to: any) => {
// 	state.refreshRouterViewKey = decodeURI(to.fullPath);
// });
// 页面加载前，处理缓存，页面刷新时路由缓存处理
onBeforeMount(() => {
    state.keepAliveNameList = keepAliveNames.value;
    mittBus.on('onTagsViewRefreshRouterView', (path: string) => {
        if (decodeURI(route.fullPath) !== path) return false;
        state.keepAliveNameList = keepAliveNames.value.filter((name: string) => route.name !== name);
        state.refreshRouterViewKey = route.path;
        nextTick(() => {
            state.refreshRouterViewKey = null;
            state.keepAliveNameList = keepAliveNames.value;
        });
    });
});
// 设置主界面切换动画
const setTransitionName = computed(() => {
    return themeConfig.value.animation;
});
// 页面卸载时
onUnmounted(() => {
    mittBus.off('onTagsViewRefreshRouterView');
});
</script>
