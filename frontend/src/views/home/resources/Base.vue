<template>
    <el-popover v-if="state.total > 0" placement="bottom" :width="650" trigger="hover" :show-after="300">
        <template #reference>
            <div class="resource-tile" :style="{ '--tile-color': resourceColor }" role="button" tabindex="0" @click="onCardClick" @keydown.enter="onCardClick">
                <span class="resource-tile__label">{{ resourceLabel }}</span>
                <span class="resource-tile__stats">
                    <span class="resource-tile__icon">
                        <SvgIcon :size="16" :name="resourceIcon" />
                    </span>
                    <span class="resource-tile__value">{{ state.total }}</span>
                </span>
            </div>
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
                        <!-- flex 容器 + 卡片 h-full：同一行的资源卡统一撑满行高，避免离线实例（无监控行）比别人矮一截 -->
                        <div v-for="resource in resources" :key="resource.id" class="min-w-0 flex" @click="navigateToResource(resource)">
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

    <!-- 无资源时仍保留卡片位置：直接隐藏会让概览区出现空洞，也读不出哪类资源尚未纳管 -->
    <div v-else class="resource-tile is-empty" :style="{ '--tile-color': resourceColor }">
        <span class="resource-tile__label">{{ resourceLabel }}</span>
        <span class="resource-tile__stats">
            <span class="resource-tile__icon">
                <SvgIcon :size="16" :name="resourceIcon" />
            </span>
            <span class="resource-tile__value">{{ state.total }}</span>
        </span>
    </div>
</template>

<script lang="ts" setup generic="T extends { id: number; code?: string; name?: string }">
import SvgIcon from '@/components/svg-icon/index.vue';
import { hasPerm } from '@/components/auth/auth';
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

const emit = defineEmits<{
    /** 总数变化：首页指标卡据此汇总「资源总数」，避免另发一轮只取 total 的请求 */
    (e: 'loaded', total: number): void;
}>();

const router = useRouter();

// 没有资源页基础权限时，资源卡仍可悬浮查看列表，但点击不跳转
const canOpenResource = hasPerm('resource:base');

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
    emit('loaded', state.total);
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
    emit('loaded', state.total);
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
    if (!canOpenResource) {
        return;
    }
    const tagResources = await tagApi.listByQuery.request({ codes: resource?.code });
    useAutoOpenResource().setCodePath(tagResources?.[0]?.codePath);
    router.push({ path: '/my-resource' });
};

// 点击卡片
const onCardClick = () => {
    const first = resources.value[0];
    if (first) {
        navigateToResource(first);
    }
};

/** 首页刷新：重置分页后重新拉取（含总数），供父级通过模板引用统一驱动 */
const reload = async () => {
    state.currentPage = 1;
    state.hasMore = true;
    state.initialized = false;
    await initLoad();
};

// 暴露方法给父组件
defineExpose({
    initLoad,
    reload,
    state,
});
</script>

<style scoped lang="scss">
.resource-tile {
    --tile-color: var(--el-color-primary);
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 14px;
    background: var(--el-fill-color-lighter);
    cursor: pointer;
    transition:
        border-color 0.2s ease,
        background-color 0.2s ease,
        transform 0.2s ease;

    &:hover {
        transform: translateY(-1px);
        border-color: color-mix(in srgb, var(--tile-color) 38%, var(--el-border-color-lighter));
        background: color-mix(in srgb, var(--tile-color) 7%, var(--el-fill-color-lighter));
    }

    &:focus-visible {
        outline: 2px solid var(--tile-color);
        outline-offset: 2px;
    }

    &.is-empty {
        cursor: default;
        opacity: 0.62;

        &:hover {
            transform: none;
            border-color: var(--el-border-color-lighter);
            background: var(--el-fill-color-lighter);
        }
    }
}

/* 名称独占一行：6/12 宽度下 4 列卡片只有 130-220px，横向排布会把 Elasticsearch 这类长名截断 */
.resource-tile__label {
    font-size: 13px;
    font-weight: 500;
    color: var(--el-text-color-regular);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.resource-tile__stats {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
}

.resource-tile__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    border-radius: 10px;
    color: var(--tile-color);
    background: color-mix(in srgb, var(--tile-color) 14%, transparent);
}

.resource-tile__value {
    font-size: 22px;
    font-weight: 700;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
    color: var(--el-text-color-primary);
}

@media (prefers-reduced-motion: reduce) {
    .resource-tile:hover {
        transform: none;
    }
}
</style>
