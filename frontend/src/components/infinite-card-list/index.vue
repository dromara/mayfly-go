<template>
    <div class="infinite-card-list">
        <!-- 搜索工具栏：字段配置化 + 右侧 actions 插槽（SearchToolbar） -->
        <div v-if="searchFields?.length || $slots.actions" class="icl-toolbar">
            <div class="icl-toolbar__fields">
                <template v-for="field in searchFields" :key="field.key">
                    <el-input
                        v-if="field.type === 'input'"
                        :model-value="searchValues?.[field.key]"
                        :placeholder="field.placeholder"
                        clearable
                        class="icl-toolbar__input"
                        @update:model-value="onFieldChange(field.key, $event)"
                        @keyup.enter="onSearch?.()"
                        @clear="onSearch?.()"
                    />
                    <el-select
                        v-else
                        :model-value="searchValues?.[field.key]"
                        :placeholder="field.placeholder"
                        clearable
                        class="icl-toolbar__select"
                        @update:model-value="onFieldChange(field.key, $event)"
                        @change="onSearch?.()"
                    >
                        <el-option v-for="opt in field.options" :key="opt.value" :label="opt.label" :value="opt.value" />
                    </el-select>
                </template>
                <!-- 搜索 / 重置按钮 -->
                <el-button icon="search" plain @click="onSearch?.()" />
                <el-button plain @click="onReset?.()">{{ $t('common.reset') }}</el-button>
            </div>
            <div class="icl-toolbar__actions">
                <slot name="actions" />
            </div>
        </div>

        <!-- 列表区域：仅此区域滚动（滚动职责单一化） -->
        <div class="icl-body">
            <!-- 首屏加载骨架：匹配网格布局 -->
            <div v-if="loading" class="icl-grid" :class="gridClass">
                <div v-for="i in skeletonCount" :key="i" class="icl-skeleton">
                    <el-skeleton animated :rows="4" />
                </div>
            </div>

            <!-- 空状态 -->
            <div v-else-if="!items?.length" class="icl-empty">
                <slot name="empty">
                    <el-icon :size="42" color="var(--el-text-color-placeholder)"><Box /></el-icon>
                    <div class="icl-empty__title">{{ emptyText || $t('common.empty') }}</div>
                    <div v-if="emptyDesc" class="icl-empty__desc">{{ emptyDesc }}</div>
                </slot>
            </div>

            <template v-else>
                <div class="icl-grid" :class="gridClass">
                    <slot v-for="(item, index) in items" :key="getRowKey(item, index)" :item="item" :index="index" />
                </div>

                <!-- 滚动哨兵：接近底部（rootMargin 200px 提前触发）自动加载下一页 -->
                <div ref="sentinelRef" class="icl-sentinel">
                    <span v-if="loadingMore" class="icl-sentinel__loading">
                        <el-icon class="is-loading"><Loading /></el-icon>
                        {{ $t('common.loadingMore') }}
                    </span>
                    <span v-else-if="!hasMore" class="icl-sentinel__nomore">{{ $t('common.noMore') }}</span>
                </div>
            </template>
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * InfiniteCardList - 通用无限滚动卡片列表
 *
 * 数据状态经 useInfiniteScroll 的 bindCardList 一次性展开传入；
 * 支持配置化搜索栏（input/select）+ actions 插槽 + 滚动哨兵自动加载 + 骨架屏 + 空状态。
 * 卡片渲染经默认作用域插槽下发（slot props: item/index）。
 */
import { useIntersectionObserver } from '@vueuse/core';
import { Box, Loading } from '@element-plus/icons-vue';
import { ref } from 'vue';

/** 搜索栏字段定义 */
export interface SearchField {
    key: string;
    /** input 文本输入 / select 下拉 */
    type: 'input' | 'select';
    placeholder?: string;
    /** select 选项 */
    options?: { label: string; value: any }[];
}

const props = withDefaults(
    defineProps<{
        items: any[];
        loading?: boolean;
        loadingMore?: boolean;
        hasMore?: boolean;
        loadMore?: () => void;
        searchFields?: SearchField[];
        searchValues?: Record<string, any>;
        onSearch?: () => void;
        onSearchChange?: (values: Record<string, any>) => void;
        onReset?: () => void;
        /** 行 key 字段名或取值函数 */
        rowKey?: string | ((item: any, index: number) => string | number);
        /** 网格布局 class（追加到默认网格样式上） */
        gridClass?: string;
        emptyText?: string;
        emptyDesc?: string;
        /** 骨架卡片数量 */
        skeletonCount?: number;
    }>(),
    {
        items: () => [],
        loading: false,
        loadingMore: false,
        hasMore: true,
        skeletonCount: 6,
    },
);

const getRowKey = (item: any, index: number): string | number => {
    if (typeof props.rowKey === 'function') return props.rowKey(item, index);
    return item[props.rowKey ?? 'id'] ?? index;
};

const onFieldChange = (key: string, value: any) => {
    props.onSearchChange?.({ ...props.searchValues, [key]: value });
};

// 滚动哨兵：hasMore 且非加载中才启用
const sentinelRef = ref<HTMLElement>();
useIntersectionObserver(
    sentinelRef,
    ([entry]) => {
        if (entry?.isIntersecting) props.loadMore?.();
    },
    { rootMargin: '200px' },
);
</script>

<style lang="scss" scoped>
.infinite-card-list {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;

    .icl-toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 8px;
        margin-bottom: 12px;

        .icl-toolbar__fields {
            display: flex;
            align-items: center;
            gap: 8px;

            .icl-toolbar__input {
                width: 220px;
            }

            .icl-toolbar__select {
                width: 140px;
            }
        }

        .icl-toolbar__actions {
            display: flex;
            align-items: center;
            gap: 0;
        }
    }

    .icl-body {
        flex: 1;
        min-height: 0;
        overflow: auto;
    }

    .icl-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 12px;
        align-content: start;
    }

    .icl-skeleton {
        padding: 14px;
        border: 1px solid var(--el-border-color-light);
        border-radius: 8px;
    }

    .icl-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 6px;
        padding: 48px 0;

        .icl-empty__title {
            font-size: 14px;
            color: var(--el-text-color-primary);
            margin-top: 8px;
        }

        .icl-empty__desc {
            font-size: 12px;
            color: var(--el-text-color-secondary);
        }
    }

    .icl-sentinel {
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 14px 0;
        min-height: 24px;

        .icl-sentinel__loading {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 13px;
            color: var(--el-text-color-secondary);
        }

        .icl-sentinel__nomore {
            font-size: 12px;
            color: var(--el-text-color-placeholder);
        }
    }
}
</style>
