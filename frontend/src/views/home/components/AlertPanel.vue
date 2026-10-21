<template>
    <SectionCard :title="$t('home.activeAlerts')" icon="Bell" :count="total">
        <template #actions>
            <el-link v-if="viewable" type="primary" underline="never" class="panel-link" @click="$emit('view-all')">
                {{ $t('home.viewAll') }}
                <SvgIcon name="ArrowRight" :size="12" />
            </el-link>
        </template>

        <div v-if="loading" class="feed-skeleton" aria-hidden="true">
            <span v-for="row of 4" :key="row" class="feed-skeleton__row"></span>
        </div>

        <div v-else-if="failed" class="feed-failed">
            <span>{{ $t('home.loadFailed') }}</span>
            <el-link type="primary" underline="never" @click="$emit('retry')">{{ $t('home.retry') }}</el-link>
        </div>

        <div v-else-if="alerts.length" class="feed-list">
            <button
                v-for="alert of alerts"
                :key="alert.id"
                type="button"
                class="feed-item"
                :class="{ 'is-static': !viewable }"
                @click="viewable && $emit('view-all')"
            >
                <span class="feed-item__bar" :class="`is-${getPriorityTagType(alert.priority)}`"></span>
                <span class="feed-item__body">
                    <span class="feed-item__head">
                        <span class="feed-item__title">{{ alert.ruleName }}</span>
                        <span class="feed-item__priority" :class="`is-${getPriorityTagType(alert.priority)}`">
                            {{ getPriorityLabel(alert.priority) }}
                        </span>
                    </span>
                    <span class="feed-item__meta">
                        {{ resourceTextOf(alert) }}
                        <span class="feed-item__sep">·</span>
                        {{ metricTextOf(alert) }}
                    </span>
                </span>
                <span class="feed-item__side">
                    <span class="feed-item__time">{{ formatFeedTime(alert.lastTriggerTime) }}</span>
                    <span class="feed-item__action">
                        {{ $t('home.viewAction') }}
                        <SvgIcon name="ArrowRight" :size="12" />
                    </span>
                </span>
            </button>
        </div>

        <HomeEmpty v-else icon="CircleCheckFilled" :title="$t('home.noAlert')" :desc="$t('home.noAlertDesc')" />
    </SectionCard>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import SectionCard from '@/components/section-card/SectionCard.vue';
import type { AlertEventVO } from '@/views/ops/alert/types';
import { formatMetricValue, getMetricLabel, getPriorityLabel, getPriorityTagType, getResourceTypeLabel } from '@/views/ops/alert/utils';
import { useI18n } from 'vue-i18n';
import { formatFeedTime } from '../utils';
import HomeEmpty from './HomeEmpty.vue';

withDefaults(
    defineProps<{
        alerts: AlertEventVO[];
        total: number | null;
        loading?: boolean;
        failed?: boolean;
        /** 当前账号是否有告警菜单权限：无权限时只展示数据，不提供跳转入口 */
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

const { t } = useI18n();

/** 资源名可能为空（资源被删除后只留 id），回退到「类型#id」与告警事件列表同口径 */
const resourceTextOf = (alert: AlertEventVO) => {
    return alert.resourceName || `${getResourceTypeLabel(alert.resourceType)}#${alert.resourceId}`;
};

/** status 指标的标签本身就是「在线状态」，再拼「：在线」会重复；只保留值，读作「资源 · 在线」 */
const metricTextOf = (alert: AlertEventVO) => {
    const value = formatMetricValue(alert.metric, alert.currentValue);
    if (alert.metric === 'status') {
        return value;
    }
    return t('home.metricValue', { metric: getMetricLabel(alert.metric), value });
};
</script>

<style scoped lang="scss">
.panel-link {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    font-size: 12px;
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

.feed-item__bar {
    --bar-tone: var(--el-color-danger);
    flex-shrink: 0;
    width: 3px;
    height: 28px;
    border-radius: 3px;
    background: var(--bar-tone);

    &.is-warning {
        --bar-tone: var(--el-color-warning);
    }

    &.is-info {
        --bar-tone: var(--el-color-info);
    }

    &.is-success {
        --bar-tone: var(--el-color-success);
    }
}

.feed-item__body {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
}

.feed-item__head {
    display: flex;
    align-items: center;
    gap: 8px;
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

.feed-item__priority {
    --tag-tone: var(--el-color-danger);
    flex-shrink: 0;
    padding: 0 6px;
    font-size: 10px;
    font-weight: 600;
    line-height: 16px;
    border-radius: 4px;
    color: var(--tag-tone);
    background: color-mix(in srgb, var(--tag-tone) 12%, transparent);

    &.is-warning {
        --tag-tone: var(--el-color-warning);
    }

    &.is-info {
        --tag-tone: var(--el-color-info);
    }

    &.is-success {
        --tag-tone: var(--el-color-success);
    }
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
