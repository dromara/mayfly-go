<template>
    <el-popover v-if="state.total > 0" placement="bottom" :width="650" trigger="hover" :show-after="300">
        <template #reference>
            <el-card shadow="hover" class="cursor-pointer transition-all hover:-translate-y-1" @click="onCardClick">
                <div class="flex flex-col items-center">
                    <SvgIcon :size="32" :name="resourceIcon" :color="resourceColor" />
                    <div class="text-xs text-gray-600 text-center">{{ resourceLabel }}</div>
                    <div class="text-lg font-bold text-(--el-color-primary)">{{ state.total }}</div>
                </div>
            </el-card>
        </template>

        <!-- 资源悬浮展示 -->
        <div class="resource-popover">
            <div class="flex justify-between items-center mb-3 pb-2 border-b border-(--el-border-color-lighter) font-medium text-sm">
                <span>{{ resourceLabel }}</span>
                <span class="text-gray-500">({{ state.total }})</span>
            </div>

            <el-scrollbar ref="scrollbarRef" :max-height="180" @scroll="handleScroll">
                <div v-loading="state.loading" class="min-w-0">
                    <div class="grid grid-cols-2 gap-2.5">
                        <div v-for="resource in resources" :key="resource.id" @click="navigateToResource(resource)">
                            <slot name="item" :resource="resource"></slot>
                        </div>
                    </div>

                    <!-- 加载更多提示 -->
                    <div v-if="state.hasMore" class="text-center py-3 text-xs text-gray-500">
                        <span class="italic">{{ $t('home.loadMore') }}</span>
                    </div>
                    <div v-else-if="resources.length > 0" class="text-center py-3 text-xs text-gray-400">
                        <span class="italic">{{ $t('home.loadedAll') }}</span>
                    </div>
                </div>
            </el-scrollbar>
        </div>
    </el-popover>
</template>

<script lang="ts" setup generic="T extends { id: number; code?: string; name?: string }">
import SvgIcon from '@/components/svg-icon/index.vue';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { tagApi } from '@/views/ops/tag/api';
import type { ElScrollbar } from 'element-plus';
import { onMounted, reactive, ref, shallowRef } from 'vue';
import { useRouter } from 'vue-router';

/** 资源列表请求方法（方法声明，参数双变兼容各 Api 实例） */
interface ApiMethod<R> {
    request(param?: Record<string, unknown>): Promise<{ list: R[]; total: number }>;
}

const props = defineProps<{
    resourceLabel: string;
    resourceIcon: string;
    resourceColor: string;
    apiMethod: ApiMethod<T>;
}>();

const router = useRouter();

const scrollbarRef = ref<InstanceType<typeof ElScrollbar>>();

// 资源列表（shallowRef 避免泛型深层解包，列表始终整体替换）
const resources = shallowRef<T[]>([]);

// 内部状态
const state = reactive({
    currentPage: 1,
    pageSize: 6,
    loading: false,
    hasMore: true,
    total: 0,
    initialized: false,
});

onMounted(() => {
    initLoad();
});

// 通用数据加载方法
const fetchData = async (pageNum: number) => {
    try {
        state.loading = true;
        const res = await props.apiMethod.request({ pageSize: state.pageSize, pageNum });
        return res;
    } catch (error) {
        console.error(`Failed to load ${props.resourceLabel}:`, error);
        return { list: [], total: 0 };
    } finally {
        state.loading = false;
    }
};

// 初始化加载数据
const initLoad = async () => {
    if (state.initialized) return;
    const res = await fetchData(1);
    const items = res?.list || [];

    state.total = res.total;
    resources.value = items.slice(0, state.pageSize);
    state.hasMore = resources.value.length < state.total;
    state.initialized = true;
};

// 加载更多数据
const loadMore = async () => {
    if (state.loading || !state.hasMore) return;

    const nextPage = state.currentPage + 1;
    const res = await fetchData(nextPage);

    const newItems = res?.list || [];
    resources.value = [...resources.value, ...newItems];
    state.currentPage = nextPage;
    state.hasMore = resources.value.length < res.total;
};

// 处理滚动
const handleScroll = () => {
    if (!scrollbarRef.value) return;

    const wrapRef = scrollbarRef.value?.wrapRef;
    if (!wrapRef) return;

    // 滚动到底部时加载更多
    if (wrapRef.scrollHeight - wrapRef.scrollTop - wrapRef.clientHeight < 20) {
        loadMore();
    }
};

// 跳转到资源
const navigateToResource = async (resource: T) => {
    const tagResources = await tagApi.listByQuery.request({ codes: resource?.code });
    useAutoOpenResource().setCodePath(tagResources?.[0]?.codePath);
    router.push({ path: '/my-resource' });
};

// 点击卡片
const onCardClick = () => {
    navigateToResource(resources.value[0]);
};

// 暴露方法给父组件
defineExpose({
    initLoad,
    state,
});
</script>

<style scoped lang="scss"></style>
