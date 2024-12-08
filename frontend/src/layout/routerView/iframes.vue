<template>
    <div>
        <div class="layout-view-bg-white flex h100" v-loading="iframeLoading">
            <iframe :src="iframeUrl" frameborder="0" height="100%" width="100%" id="iframe" v-show="!iframeLoading"></iframe>
        </div>
    </div>
</template>

<script lang="ts">
import { defineComponent, reactive, toRefs, onMounted, onBeforeMount, onUnmounted, nextTick } from 'vue';
import { useRoute } from 'vue-router';
import mittBus from '@/common/utils/mitt';
export default defineComponent({
    name: 'layoutIfameView',
    props: {
        meta: {
            type: Object,
            default: () => {},
        },
    },
    setup(props, { emit }) {
        const route = useRoute();
        const state = reactive({
            iframeLoading: true,
            iframeUrl: '',
        });
        // 初始化页面加载 loading
        const initIframeLoad = () => {
            nextTick(() => {
                state.iframeLoading = true;
                const iframe = document.getElementById('iframe');
                if (!iframe) return false;
                iframe.onload = () => {
                    state.iframeLoading = false;
                };
            });
        };
        // 页面加载前
        onBeforeMount(() => {
            state.iframeUrl = props.meta.link;
            mittBus.on('onTagsViewRefreshRouterView', (path: string) => {
                if (route.path !== path) return false;
                emit('getCurrentRouteMeta');
            });
        });
        // 页面加载时
        onMounted(() => {
            initIframeLoad();
        });
        // 页面卸载时
        onUnmounted(() => {
            mittBus.off('onTagsViewRefreshRouterView', () => {});
        });
        return {
            ...toRefs(state),
        };
    },
});
</script>
