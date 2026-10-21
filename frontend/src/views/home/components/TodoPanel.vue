<template>
    <SectionCard :title="$t('home.todoTasks')" icon="Tickets" :count="total">
        <template #actions>
            <el-link v-if="viewable" type="primary" underline="never" class="panel-link" @click="$emit('view-all')">
                {{ $t('home.viewAll') }}
                <SvgIcon name="ArrowRight" :size="12" />
            </el-link>
        </template>

        <div class="todo-body">
            <div v-if="loading" class="feed-skeleton" aria-hidden="true">
                <span v-for="row of 4" :key="row" class="feed-skeleton__row"></span>
            </div>

            <div v-else-if="failed" class="feed-failed">
                <span>{{ $t('home.loadFailed') }}</span>
                <el-link type="primary" underline="never" @click="$emit('retry')">{{ $t('home.retry') }}</el-link>
            </div>

            <div v-else-if="tasks.length" class="feed-list">
                <button
                    v-for="task of tasks"
                    :key="task.id"
                    type="button"
                    class="feed-item"
                    :class="{ 'is-static': !viewable }"
                    @click="viewable && $emit('view-all')"
                >
                    <span class="feed-item__dot"></span>
                    <span class="feed-item__body">
                        <span class="feed-item__title">{{ task.nodeName || $t('home.todoTasks') }}</span>
                        <span class="feed-item__meta">
                            {{ task.bizKey || $t('home.taskFallback', { id: task.procinstId }) }}
                            <template v-if="task.creator">
                                <span class="feed-item__sep">·</span>{{ task.creator }}
                            </template>
                        </span>
                    </span>
                    <span class="feed-item__side">
                        <span class="feed-item__time">{{ formatFeedTime(task.createTime) }}</span>
                        <span class="feed-item__action">
                            {{ $t('home.viewAction') }}
                            <SvgIcon name="ArrowRight" :size="12" />
                        </span>
                    </span>
                </button>
            </div>

            <HomeEmpty v-else icon="CircleCheckFilled" :title="$t('home.noTodo')" :desc="$t('home.noTodoDesc')" />
        </div>
    </SectionCard>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import SectionCard from '@/components/section-card/SectionCard.vue';
import type { ProcinstTask } from '@/views/flow/types';
import { formatFeedTime } from '../utils';
import HomeEmpty from './HomeEmpty.vue';

withDefaults(
    defineProps<{
        tasks: ProcinstTask[];
        total: number | null;
        loading?: boolean;
        failed?: boolean;
        /** 当前账号是否有待办菜单权限：无权限时只展示数据，不提供跳转入口 */
        viewable?: boolean;
    }>(),
    {
        loading: false,
        failed: false,
        viewable: true,
    }
);

defineEmits<{
    (e: 'view-all'): void;
    (e: 'retry'): void;
}>();
</script>

<style scoped lang="scss">
.panel-link {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    font-size: 12px;
}

/* 列表随内容高度：待办条目少时卡片就矮，并在左列下方接快捷入口，不再靠空白撑满 */
.todo-body {
    display: flex;
    flex-direction: column;
}

.feed-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px;
}

.feed-item {
    display: flex;
    align-items: center;
    gap: 10px;
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

    /* 无菜单权限时整行不可点，也不给 hover 反馈 */
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

.feed-item__dot {
    flex-shrink: 0;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--el-color-warning);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-warning) 16%, transparent);
}

.feed-item__body {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
}

.feed-item__title {
    font-size: 13px;
    font-weight: 500;
    color: var(--el-text-color-primary);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.feed-item__meta {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.feed-item__sep {
    margin: 0 4px;
}

.feed-item__side {
    display: flex;
    align-items: center;
    gap: 14px;
    flex-shrink: 0;
}

.feed-item__time {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    font-variant-numeric: tabular-nums;
}

.feed-item__action {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding-left: 2px;
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
        width: 72%;
    }

    &:nth-child(3n) {
        width: 86%;
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
