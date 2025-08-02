<template>
    <div class="h-full">
        <div class="w-full h-full relative" v-for="v in setIframeList" :key="v.path">
            <transition-group :name="name">
                <div
                    class="absolute top-0 left-0 w-full h-full flex justify-center items-center bg-white z-[100]"
                    v-if="v.meta.loading"
                    :key="`${v.path}-loading`"
                >
                    <div class="flex flex-col items-center text-gray-500">
                        <i class="el-icon-loading"></i>
                        <div class="mt-2.5 text-sm">loading...</div>
                    </div>
                </div>
                <iframe
                    :src="v.meta.link"
                    :key="v.path"
                    frameborder="0"
                    height="100%"
                    width="100%"
                    style="position: absolute"
                    :data-url="v.path"
                    v-show="getRoutePath === v.path"
                    ref="iframeRef"
                />
            </transition-group>
        </div>
    </div>
</template>

<script setup lang="ts" name="layoutIframeView">
import { computed, watch, ref, nextTick } from 'vue';
import { useRoute } from 'vue-router';

// 定义父组件传过来的值
const props = defineProps({
    // 刷新 iframe
    refreshKey: {
        type: String,
        default: () => '',
    },
    // 过渡动画 name
    name: {
        type: String,
        default: () => 'slide-right',
    },
    // iframe 列表
    list: {
        type: Array,
        default: () => [],
    },
});

const iframeRef = ref();
const route = useRoute();

// 处理 list 列表，当打开时，才进行加载
const setIframeList = computed(() => {
    return props.list.filter((v: any) => v.meta?.isIframeOpen) as any[];
});

// 获取 iframe 当前路由 path
const getRoutePath = computed(() => {
    return route.path;
});

// 关闭 iframe loading
const closeIframeLoading = (val: string, item: any) => {
    nextTick(() => {
        if (!iframeRef.value) return false;
        iframeRef.value.forEach((v: HTMLElement) => {
            if (v.dataset.url === val) {
                v.onload = () => {
                    if (item.meta?.isIframeOpen && item.meta.loading) item.meta.loading = false;
                };
            }
        });
    });
};

// 监听路由变化，初始化 iframe 数据，防止多个 iframe 时，切换不生效
watch(
    () => route.fullPath,
    (val) => {
        const item: any = props.list.find((v: any) => v.path === val);
        if (!item) return false;
        if (!item.meta.isIframeOpen) item.meta.isIframeOpen = true;
        closeIframeLoading(val, item);
    },
    {
        immediate: true,
    }
);

// 监听 iframe refreshKey 变化，用于 tagsview 右键菜单刷新
watch(
    () => props.refreshKey,
    () => {
        const item: any = props.list.find((v: any) => v.path === route.path);
        if (!item) return false;
        if (item.meta.isIframeOpen) item.meta.isIframeOpen = false;
        setTimeout(() => {
            item.meta.isIframeOpen = true;
            item.meta.loading = true;
            closeIframeLoading(route.fullPath, item);
        });
    },
    {
        deep: true,
    }
);
</script>

<style scoped></style>
