<template>
    <div class="h100">
        <router-view v-slot="{ Component }">
            <transition :name="setTransitionName" mode="out-in">
                <keep-alive :include="keepAliveNameList">
                    <component :is="Component" :key="refreshRouterViewKey" class="w100" />
                </keep-alive>
            </transition>
        </router-view>
    </div>
</template>

<script lang="ts">
import { computed, defineComponent, toRefs, reactive, getCurrentInstance, onBeforeMount, onUnmounted, nextTick, ref } from 'vue';
import { useRoute, onBeforeRouteUpdate } from 'vue-router';
import { useStore } from '@/store/index.ts';
export default defineComponent({
    name: 'layoutParentView',
    setup() {
        const { proxy } = getCurrentInstance() as any;
        const route = useRoute();
        const store = useStore();
        const state: any = reactive({
            refreshRouterViewKey: null,
            keepAliveNameList: [],
            keepAliveNameNewList: [],
        });
        // 监听路由的变化，动态赋值给refreshRouterViewKey
        // onBeforeRouteUpdate((to: any) => {
        // 	state.refreshRouterViewKey = decodeURI(to.fullPath);
        // });
        // 设置主界面切换动画
        const setTransitionName = computed(() => {
            return store.state.themeConfig.themeConfig.animation;
        });
        // 获取布局配置信息
        const getThemeConfig = computed(() => {
            return store.state.themeConfig.themeConfig;
        });
        // 获取组件缓存列表(name值)
        const getKeepAliveNames = computed(() => {
            return store.state.keepAliveNames.keepAliveNames;
        });
        // 页面加载前，处理缓存，页面刷新时路由缓存处理
        onBeforeMount(() => {
            state.keepAliveNameList = getKeepAliveNames.value;
            proxy.mittBus.on('onTagsViewRefreshRouterView', (path: string) => {
                if (decodeURI(route.fullPath) !== path) return false;
                state.keepAliveNameList = getKeepAliveNames.value.filter((name: string) => route.name !== name);
                state.refreshRouterViewKey = route.path;
                nextTick(() => {
                    state.refreshRouterViewKey = null;
                    state.keepAliveNameList = getKeepAliveNames.value;
                });
            });
        });
        // 页面卸载时
        onUnmounted(() => {
            proxy.mittBus.off('onTagsViewRefreshRouterView');
        });
        return {
            getThemeConfig,
            getKeepAliveNames,
            setTransitionName,
            ...toRefs(state),
        };
    },
});
</script>
