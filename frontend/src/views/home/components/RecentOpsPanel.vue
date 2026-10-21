<template>
    <SectionCard :title="$t('home.recentOperations')" icon="Clock" :count="logs.length || null">
        <div v-if="loading" class="feed-skeleton" aria-hidden="true">
            <span v-for="row of 5" :key="row" class="feed-skeleton__row"></span>
        </div>

        <div v-else-if="failed" class="feed-failed">
            <span>{{ $t('home.loadFailed') }}</span>
            <el-link type="primary" underline="never" @click="$emit('retry')">{{ $t('home.retry') }}</el-link>
        </div>

        <div v-else-if="logs.length" class="op-list">
            <button
                v-for="log of logs"
                :key="log.id"
                type="button"
                class="op-item"
                :class="{ 'is-static': !linkable }"
                @click="linkable && $emit('navigate', log.codePath)"
            >
                <span class="op-item__time">{{ formatFeedTime(log.createTime) }}</span>
                <span class="op-item__path">
                    <TagCodePath :path="log.codePath" />
                </span>
                <span v-if="linkable" class="op-item__action">
                    {{ $t('home.openResource') }}
                    <SvgIcon name="ArrowRight" :size="12" />
                </span>
            </button>
        </div>

        <HomeEmpty v-else icon="Document" :title="$t('home.noOpRecord')" tone="primary" />
    </SectionCard>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import SectionCard from '@/components/section-card/SectionCard.vue';
import type { ResourceOpLog } from '@/views/ops/tag/types';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { formatFeedTime } from '../utils';
import HomeEmpty from './HomeEmpty.vue';

withDefaults(
    defineProps<{
        logs: ResourceOpLog[];
        loading?: boolean;
        failed?: boolean;
        /** 当前账号是否有资源页菜单权限：无权限时只展示路径，不提供跳转入口 */
        linkable?: boolean;
    }>(),
    {
        loading: false,
        failed: false,
        linkable: true,
    }
);

defineEmits<{
    (e: 'navigate', codePath: string): void;
    (e: 'retry'): void;
}>();
</script>

<style scoped lang="scss">
.op-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px;
}

/* 操作列表信息短、行宽大：宽屏改为双列，一行不再只占左侧一小段 */
@media (min-width: 1200px) {
    .op-list {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 2px 16px;
    }
}

.op-item {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 9px 10px;
    font: inherit;
    text-align: left;
    border: 0;
    border-radius: 10px;
    background: transparent;
    cursor: pointer;
    transition: background-color 0.18s ease;

    &:hover {
        background: var(--el-fill-color-light);
    }

    /* 无资源页权限时行不可点，也不给 hover 反馈 */
    &.is-static {
        cursor: default;

        &:hover {
            background: transparent;
        }
    }

    &:focus-visible {
        outline: 2px solid var(--el-color-primary);
        outline-offset: -2px;
    }
}

.op-item__time {
    flex-shrink: 0;
    width: 82px;
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    font-variant-numeric: tabular-nums;
}

.op-item__path {
    flex: 1;
    min-width: 0;
    overflow: hidden;
}

.op-item__action {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    flex-shrink: 0;
    font-size: 12px;
    font-weight: 500;
    color: var(--el-color-primary);
}

.feed-skeleton {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px 18px;
}

.feed-skeleton__row {
    height: 14px;
    border-radius: 6px;
    background: var(--el-fill-color);
    animation: feed-pulse 1.2s ease-in-out infinite;

    &:nth-child(2n) {
        width: 78%;
    }

    &:nth-child(3n) {
        width: 64%;
    }
}

.feed-failed {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 36px 20px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

@media (max-width: 640px) {
    .op-item__time {
        width: 72px;
    }
}

@keyframes feed-pulse {
    0%,
    100% {
        opacity: 0.45;
    }
    50% {
        opacity: 1;
    }
}

@media (prefers-reduced-motion: reduce) {
    .feed-skeleton__row {
        animation: none;
    }
}
</style>
