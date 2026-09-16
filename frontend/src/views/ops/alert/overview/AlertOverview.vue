<template>
    <div class="alert-overview p-4">
        <!-- 页面标题 -->
        <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-medium">{{ $t('alert.overview') }}</h2>
            <el-button link type="primary" icon="refresh" @click="loadOverview">{{ $t('common.refresh') }}</el-button>
        </div>

        <!-- 事件状态概览 -->
        <el-row :gutter="16" class="stat-cards mb-4">
            <el-col :xs="12" :sm="8" :md="4">
                <div class="stat-card firing">
                    <div class="stat-value">{{ overview.firingCount }}</div>
                    <div class="stat-label">{{ $t('alert.firing') }}</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="8" :md="4">
                <div class="stat-card acknowledged">
                    <div class="stat-value">{{ overview.acknowledgedCount }}</div>
                    <div class="stat-label">{{ $t('alert.acknowledged') }}</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="8" :md="4">
                <div class="stat-card recovered">
                    <div class="stat-value">{{ overview.recoveredCount }}</div>
                    <div class="stat-label">{{ $t('alert.recovered') }}</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="8" :md="4">
                <div class="stat-card closed">
                    <div class="stat-value">{{ overview.closedCount }}</div>
                    <div class="stat-label">{{ $t('alert.closed') }}</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="8" :md="4">
                <div class="stat-card mttr">
                    <div class="stat-value">{{ formatDuration(overview.avgRecoveryTime) }}</div>
                    <div class="stat-label">MTTR</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="8" :md="4">
                <div class="stat-card today">
                    <div class="stat-value">{{ overview.todayNotifyCount }}</div>
                    <div class="stat-label">{{ $t('alert.todayNotifications') }}</div>
                </div>
            </el-col>
        </el-row>

        <!-- 规则统计 + 通知统计 -->
        <el-row :gutter="16" class="mb-4">
            <el-col :xs="24" :md="12">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.ruleStats') }}</h3>
                    <div class="stats-grid">
                        <div class="stat-item">
                            <span class="stat-num">{{ overview.ruleStats?.total || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.totalRules') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-success">{{ overview.ruleStats?.enabled || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.enabled') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-muted">{{ overview.ruleStats?.disabled || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.disabled') }}</span>
                        </div>
                    </div>
                    <!-- 按优先级分布 -->
                    <div class="sub-section" v-if="overview.ruleStats?.byPriority">
                        <div class="sub-title">{{ $t('alert.byPriority') }}</div>
                        <div class="priority-bars">
                            <div v-for="(count, key) in overview.ruleStats.byPriority" :key="key" class="priority-bar">
                                <span class="priority-label">{{ key }}</span>
                                <div class="bar-container">
                                    <div class="bar-fill" :style="{ width: getPriorityBarWidth(count) + '%' }"></div>
                                </div>
                                <span class="bar-count">{{ count }}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </el-col>
            <el-col :xs="24" :md="12">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.notifyStats') }}</h3>
                    <div class="stats-grid">
                        <div class="stat-item">
                            <span class="stat-num">{{ overview.notifyStats?.todaySent || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.todaySent') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-success">{{ formatPercent(overview.notifyStats?.successRate) }}%</span>
                            <span class="stat-desc">{{ $t('alert.successRate') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num">{{ overview.notifyStats?.policyMatched || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.policyMatched') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-danger">{{ overview.notifyStats?.unmatchedCount || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.unmatchedCount') }}</span>
                        </div>
                    </div>
                    <!-- 策略匹配率 -->
                    <div class="sub-section">
                        <div class="sub-title">{{ $t('alert.policyMatchRate') }}</div>
                        <el-progress :percentage="overview.notifyStats?.successRate || 0" :stroke-width="8" />
                    </div>
                    <!-- 阶段一：渠道分布 -->
                    <div class="sub-section" v-if="overview.notifyStats?.channelDistribution">
                        <div class="sub-title">{{ $t('alert.channelDistribution') }}</div>
                        <div class="channel-bars">
                            <div v-for="(count, channel) in overview.notifyStats.channelDistribution" :key="channel" class="channel-bar">
                                <span class="channel-label">{{ channel }}</span>
                                <div class="bar-container">
                                    <div class="bar-fill channel-fill" :style="{ width: getChannelBarWidth(count) + '%' }"></div>
                                </div>
                                <span class="bar-count">{{ count }}</span>
                            </div>
                        </div>
                    </div>
                    <!-- 阶段二：精确统计 -->
                    <div class="sub-section" v-if="overview.notifyStats?.todaySuccess !== undefined">
                        <div class="sub-title">{{ $t('alert.preciseStats') }}</div>
                        <div class="stats-grid">
                            <div class="stat-item">
                                <span class="stat-num text-success">{{ overview.notifyStats?.todaySuccess || 0 }}</span>
                                <span class="stat-desc">{{ $t('alert.todaySuccess') }}</span>
                            </div>
                            <div class="stat-item">
                                <span class="stat-num text-danger">{{ overview.notifyStats?.todayFailed || 0 }}</span>
                                <span class="stat-desc">{{ $t('alert.todayFailed') }}</span>
                            </div>
                        </div>
                    </div>
                    <!-- 阶段二：渠道维度统计 -->
                    <div class="sub-section" v-if="overview.notifyStats?.byChannel && Object.keys(overview.notifyStats.byChannel).length > 0">
                        <div class="sub-title">{{ $t('alert.byChannelStats') }}</div>
                        <div class="channel-stats-list">
                            <div v-for="(stat, channel) in overview.notifyStats.byChannel" :key="channel" class="channel-stat-item">
                                <span class="channel-name">{{ channel }}</span>
                                <span class="channel-stat">{{ stat.success }}/{{ stat.sent }} ({{ formatPercent(stat.rate) }}%)</span>
                            </div>
                        </div>
                    </div>
                </div>
            </el-col>
        </el-row>

        <!-- 静默/抑制/升级统计 -->
        <el-row :gutter="16" class="mb-4">
            <el-col :xs="24" :md="8">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.silenceStats') }}</h3>
                    <div class="stats-grid">
                        <div class="stat-item">
                            <span class="stat-num">{{ overview.silenceStats?.total || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.totalRules') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-warning">{{ overview.silenceStats?.active || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.activeNow') }}</span>
                        </div>
                    </div>
                </div>
            </el-col>
            <el-col :xs="24" :md="8">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.inhibitionStats') }}</h3>
                    <div class="stats-grid">
                        <div class="stat-item">
                            <span class="stat-num">{{ overview.inhibitionStats?.total || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.totalRules') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-warning">{{ overview.inhibitionStats?.active || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.activeNow') }}</span>
                        </div>
                    </div>
                </div>
            </el-col>
            <el-col :xs="24" :md="8">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.escalationStats') }}</h3>
                    <div class="stats-grid">
                        <div class="stat-item">
                            <span class="stat-num">{{ overview.escalationStats?.total || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.totalRules') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-warning">{{ overview.escalationStats?.active || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.activeNow') }}</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-num text-danger">{{ overview.escalationStats?.escalatedCount || 0 }}</span>
                            <span class="stat-desc">{{ $t('alert.escalated') }}</span>
                        </div>
                    </div>
                </div>
            </el-col>
        </el-row>

        <!-- Top 告警 -->
        <el-row :gutter="16" class="mb-4">
            <el-col :xs="24" :md="8">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.topFrequent') }}</h3>
                    <div class="top-list">
                        <div v-for="(event, idx) in overview.topFrequent" :key="event.id" class="top-item">
                            <span class="rank">{{ idx + 1 }}</span>
                            <span class="name">{{ event.ruleName }}</span>
                            <span class="count">{{ event.triggerCount }}{{ $t('alert.times') }}</span>
                        </div>
                        <div v-if="!overview.topFrequent?.length" class="empty-text">{{ $t('alert.noData') }}</div>
                    </div>
                </div>
            </el-col>
            <el-col :xs="24" :md="8">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.topLongest') }}</h3>
                    <div class="top-list">
                        <div v-for="(event, idx) in overview.topLongest" :key="event.id" class="top-item">
                            <span class="rank">{{ idx + 1 }}</span>
                            <span class="name">{{ event.ruleName }}</span>
                            <span class="count">{{ getEventDuration(event) }}</span>
                        </div>
                        <div v-if="!overview.topLongest?.length" class="empty-text">{{ $t('alert.noData') }}</div>
                    </div>
                </div>
            </el-col>
            <el-col :xs="24" :md="8">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.topNotified') }}</h3>
                    <div class="top-list">
                        <div v-for="(event, idx) in overview.topNotified" :key="event.id" class="top-item">
                            <span class="rank">{{ idx + 1 }}</span>
                            <span class="name">{{ event.ruleName }}</span>
                            <span class="count">{{ event.notifyCount }}{{ $t('alert.times') }}</span>
                        </div>
                        <div v-if="!overview.topNotified?.length" class="empty-text">{{ $t('alert.noData') }}</div>
                    </div>
                </div>
            </el-col>
        </el-row>

        <!-- 通知趋势 + 质量报告 -->
        <el-row :gutter="16" class="mb-4">
            <el-col :xs="24" :md="16">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.notifyTrend') }}</h3>
                    <div class="trend-chart" v-if="overview.notifyTrend?.length">
                        <div class="trend-bars">
                            <div v-for="point in overview.notifyTrend" :key="point.hour" class="trend-bar-item">
                                <div class="trend-bar-wrapper">
                                    <div class="trend-bar success-bar" :style="{ height: getTrendBarHeight(point.success) + '%' }" :title="$t('alert.todaySuccess') + ': ' + point.success"></div>
                                    <div class="trend-bar failed-bar" :style="{ height: getTrendBarHeight(point.failed) + '%' }" :title="$t('alert.todayFailed') + ': ' + point.failed"></div>
                                </div>
                                <span class="trend-label">{{ point.hour }}</span>
                            </div>
                        </div>
                        <div class="trend-legend">
                            <span class="legend-item"><span class="legend-dot success-dot"></span>{{ $t('alert.todaySuccess') }}</span>
                            <span class="legend-item"><span class="legend-dot failed-dot"></span>{{ $t('alert.todayFailed') }}</span>
                        </div>
                    </div>
                    <div v-else class="empty-text">{{ $t('alert.noData') }}</div>
                </div>
            </el-col>
            <el-col :xs="24" :md="8">
                <div class="overview-card">
                    <h3 class="card-title">{{ $t('alert.qualityReport') }}</h3>
                    <div class="quality-stats">
                        <div class="quality-item">
                            <span class="quality-label">{{ $t('alert.avgSendDelay') }}</span>
                            <span class="quality-value">{{ overview.qualityReport?.avgSendDelay || 0 }}s</span>
                        </div>
                        <div class="quality-item">
                            <span class="quality-label">{{ $t('alert.maxSendDelay') }}</span>
                            <span class="quality-value">{{ overview.qualityReport?.maxSendDelay || 0 }}s</span>
                        </div>
                        <div class="quality-item">
                            <span class="quality-label">{{ $t('alert.weeklySuccess') }}</span>
                            <span class="quality-value text-success">{{ formatPercent(overview.qualityReport?.weeklySuccess) }}%</span>
                        </div>
                        <div class="quality-item">
                            <span class="quality-label">{{ $t('alert.monthlySuccess') }}</span>
                            <span class="quality-value text-success">{{ formatPercent(overview.qualityReport?.monthlySuccess) }}%</span>
                        </div>
                    </div>
                    <div class="sub-section" v-if="overview.qualityReport?.topFailReasons?.length">
                        <div class="sub-title">{{ $t('alert.topFailReasons') }}</div>
                        <div class="fail-reasons-list">
                            <div v-for="(reason, idx) in overview.qualityReport.topFailReasons" :key="idx" class="fail-reason-item">
                                <span class="fail-reason-text">{{ reason.reason }}</span>
                                <span class="fail-reason-count">{{ reason.count }}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </el-col>
        </el-row>

        <!-- 最近告警事件 -->
        <div class="overview-card">
            <div class="flex items-center justify-between mb-3">
                <h3 class="card-title mb-0">{{ $t('alert.recentEvents') }}</h3>
            </div>
            <page-table :columns="columns" :data="recentEventsList" :pageable="false" :tool-button="false" border>
                <template #resourceName="{ data }">
                    {{ data.resourceName || `${getResourceTypeLabel(data.resourceType)}#${data.resourceId}` }}
                </template>

                <template #metric="{ data }">
                    {{ getMetricLabel(data.metric) }}
                </template>

                <template #currentValue="{ data }">
                    {{ formatMetricValue(data.metric, data.currentValue) }}
                </template>

                <template #priority="{ data }">
                    <enum-tag :enums="AlertPriorityEnum" :value="data.priority" />
                </template>

                <template #status="{ data }">
                    <enum-tag :enums="AlertEventStatusEnum" :value="data.status" />
                </template>
            </page-table>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { alertOverviewApi } from '../api';
import type { AlertOverviewData, AlertEventVO } from '../types';
import { AlertEventStatusEnum, AlertPriorityEnum } from '../enums';
import { formatMetricValue, getResourceTypeLabel, getMetricLabel } from '../utils';
import EnumTag from '@/components/enum-tag/EnumTag.vue';

const REFRESH_INTERVAL = 60 * 1000;

const overview = ref<AlertOverviewData>({
    firingCount: 0,
    acknowledgedCount: 0,
    recoveredCount: 0,
    closedCount: 0,
    avgRecoveryTime: 0,
    todayNotifyCount: 0,
    ruleStats: { total: 0, enabled: 0, disabled: 0, byPriority: {}, byResourceType: {} },
    notifyStats: {
        todaySent: 0, successRate: 0, policyMatched: 0, totalEvents: 0,
        unmatchedCount: 0, channelDistribution: {},
        todaySuccess: 0, todayFailed: 0, byChannel: {},
    },
    silenceStats: { total: 0, active: 0, silencedCount: 0 },
    inhibitionStats: { total: 0, active: 0, inhibitedCount: 0 },
    escalationStats: { total: 0, active: 0, escalatedCount: 0, maxLevelReached: 0 },
    topFrequent: [],
    topLongest: [],
    topNotified: [],
    recentEvents: { total: 0, list: [] },
    notifyTrend: [],
    qualityReport: {
        avgSendDelay: 0, maxSendDelay: 0, queueLength: 0,
        weeklySuccess: 0, monthlySuccess: 0, topFailReasons: [],
    },
});

const recentEventsList = computed(() => overview.value.recentEvents?.list ?? []);

const columns = [
    TableColumn.new('ruleName', 'alert.ruleName'),
    TableColumn.new('resourceName', 'alert.resourceName').isSlot(),
    TableColumn.new('metric', 'alert.metricName').isSlot(),
    TableColumn.new('currentValue', 'alert.currentValue').isSlot().alignCenter(),
    TableColumn.new('priority', 'alert.priority').isSlot().alignCenter(),
    TableColumn.new('status', 'alert.status').isSlot().alignCenter(),
    TableColumn.new('firstTriggerTime', 'alert.firstTriggerTime').isTime(),
    TableColumn.new('lastTriggerTime', 'alert.lastTriggerTime').isTime(),
];

const loadOverview = async () => {
    try {
        overview.value = await alertOverviewApi.overview.request({});
    } catch (e) {
        console.error('load overview error', e);
    }
};

// 格式化持续时间（秒转为可读格式）
const formatDuration = (seconds: number): string => {
    if (!seconds || seconds <= 0) return '0s';
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0) return `${h}h${m}m`;
    if (m > 0) return `${m}m${s}s`;
    return `${s}s`;
};

// 计算事件持续时间
const getEventDuration = (event: AlertEventVO): string => {
    const start = new Date(event.firstTriggerTime).getTime();
    const end = event.recoverTime ? new Date(event.recoverTime).getTime() : Date.now();
    const seconds = Math.floor((end - start) / 1000);
    return formatDuration(seconds);
};

// 格式化百分比
const formatPercent = (value: number): string => {
    if (!value) return '0';
    return value.toFixed(1);
};

// 计算优先级条形图宽度
const getPriorityBarWidth = (count: number): number => {
    const total = overview.value.ruleStats?.total || 1;
    return Math.max(5, (count / total) * 100);
};

// 计算渠道条形图宽度
const getChannelBarWidth = (count: number): number => {
    const distribution = overview.value.notifyStats?.channelDistribution || {};
    const maxCount = Math.max(...Object.values(distribution), 1);
    return Math.max(5, (count / maxCount) * 100);
};

// 计算趋势条形图高度（基于所有趋势点中的最大值）
const getTrendBarHeight = (value: number): number => {
    const trend = overview.value.notifyTrend || [];
    const maxVal = Math.max(...trend.map(p => Math.max(p.success, p.failed)), 1);
    return Math.max(2, (value / maxVal) * 100);
};

let refreshTimer: ReturnType<typeof setInterval> | null = null;

onMounted(() => {
    loadOverview();
    refreshTimer = setInterval(loadOverview, REFRESH_INTERVAL);
});

onBeforeUnmount(() => {
    if (refreshTimer) {
        clearInterval(refreshTimer);
        refreshTimer = null;
    }
});
</script>

<style lang="scss" scoped>
.alert-overview {
    // 统一行内卡片高度
    :deep(.el-row) {
        display: flex;
        flex-wrap: wrap;

        .el-col {
            display: flex;
        }
    }

    .stat-card {
        background: var(--el-bg-color);
        border-radius: 8px;
        padding: 20px 16px;
        text-align: center;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
        transition: all 0.2s ease-out;
        margin-bottom: 16px;
        width: 100%;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        min-height: 90px;

        &:hover {
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
            transform: translateY(-2px);
        }

        .stat-value {
            font-size: 28px;
            font-weight: 600;
            line-height: 1.3;
        }

        .stat-label {
            font-size: 13px;
            color: var(--el-text-color-secondary);
            margin-top: 6px;
        }

        &.firing .stat-value {
            color: var(--el-color-danger);
        }
        &.acknowledged .stat-value {
            color: var(--el-color-warning);
        }
        &.recovered .stat-value {
            color: var(--el-color-success);
        }
        &.closed .stat-value {
            color: var(--el-text-color-secondary);
        }
        &.mttr .stat-value {
            color: var(--el-color-primary);
        }
        &.today .stat-value {
            color: var(--el-color-info);
        }
    }

    .overview-card {
        background: var(--el-bg-color);
        border-radius: 8px;
        padding: 20px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
        margin-bottom: 16px;
        width: 100%;
        display: flex;
        flex-direction: column;

        .card-title {
            font-size: 14px;
            font-weight: 500;
            color: var(--el-text-color-primary);
            margin: 0 0 16px 0;
        }
    }

    .stats-grid {
        display: flex;
        justify-content: space-around;
        gap: 16px;
        flex: 1;

        .stat-item {
            text-align: center;
            flex: 1;

            .stat-num {
                display: block;
                font-size: 24px;
                font-weight: 600;
                color: var(--el-text-color-primary);
            }

            .stat-desc {
                display: block;
                font-size: 12px;
                color: var(--el-text-color-secondary);
                margin-top: 4px;
            }
        }
    }

    .sub-section {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid var(--el-border-color-lighter);

        .sub-title {
            font-size: 12px;
            color: var(--el-text-color-secondary);
            margin-bottom: 12px;
        }
    }

    .priority-bars {
        .priority-bar {
            display: flex;
            align-items: center;
            margin-bottom: 8px;

            .priority-label {
                width: 30px;
                font-size: 12px;
                font-weight: 500;
            }

            .bar-container {
                flex: 1;
                height: 16px;
                background: var(--el-fill-color-light);
                border-radius: 4px;
                overflow: hidden;
                margin: 0 8px;

                .bar-fill {
                    height: 100%;
                    background: var(--el-color-primary);
                    border-radius: 4px;
                    transition: width 0.3s ease-out;
                }
            }

            .bar-count {
                width: 30px;
                text-align: right;
                font-size: 12px;
                color: var(--el-text-color-regular);
            }
        }
    }

    .top-list {
        flex: 1;
        min-height: 120px;

        .top-item {
            display: flex;
            align-items: center;
            padding: 8px 0;
            border-bottom: 1px solid var(--el-border-color-extra-light);

            &:last-child {
                border-bottom: none;
            }

            .rank {
                width: 20px;
                height: 20px;
                border-radius: 50%;
                background: var(--el-fill-color);
                color: var(--el-text-color-secondary);
                font-size: 11px;
                display: flex;
                align-items: center;
                justify-content: center;
                margin-right: 12px;
                flex-shrink: 0;
            }

            &:nth-child(1) .rank {
                background: var(--el-color-danger-light-7);
                color: var(--el-color-danger);
            }
            &:nth-child(2) .rank {
                background: var(--el-color-warning-light-7);
                color: var(--el-color-warning);
            }
            &:nth-child(3) .rank {
                background: var(--el-color-primary-light-7);
                color: var(--el-color-primary);
            }

            .name {
                flex: 1;
                font-size: 13px;
                color: var(--el-text-color-regular);
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
            }

            .count {
                font-size: 12px;
                color: var(--el-text-color-secondary);
                margin-left: 8px;
                flex-shrink: 0;
            }
        }

        .empty-text {
            text-align: center;
            color: var(--el-text-color-placeholder);
            font-size: 13px;
            padding: 20px 0;
        }
    }

    .text-success {
        color: var(--el-color-success) !important;
    }
    .text-warning {
        color: var(--el-color-warning) !important;
    }
    .text-danger {
        color: var(--el-color-danger) !important;
    }
    .text-muted {
        color: var(--el-text-color-secondary) !important;
    }

    // 趋势图表
    .trend-chart {
        .trend-bars {
            display: flex;
            align-items: flex-end;
            gap: 2px;
            height: 120px;
            padding: 0 4px;
        }
        .trend-bar-item {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            min-width: 0;
        }
        .trend-bar-wrapper {
            width: 100%;
            height: 100px;
            display: flex;
            align-items: flex-end;
            justify-content: center;
            gap: 1px;
        }
        .trend-bar {
            width: 40%;
            min-height: 2px;
            border-radius: 2px 2px 0 0;
            transition: height 0.3s ease;
        }
        .success-bar {
            background-color: var(--el-color-success);
        }
        .failed-bar {
            background-color: var(--el-color-danger);
        }
        .trend-label {
            font-size: 10px;
            color: var(--el-text-color-placeholder);
            margin-top: 4px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            max-width: 100%;
            text-align: center;
        }
        .trend-legend {
            display: flex;
            justify-content: center;
            gap: 16px;
            margin-top: 8px;
            font-size: 12px;
            color: var(--el-text-color-regular);
        }
        .legend-item {
            display: flex;
            align-items: center;
            gap: 4px;
        }
        .legend-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
        }
        .success-dot { background-color: var(--el-color-success); }
        .failed-dot { background-color: var(--el-color-danger); }
    }

    // 质量报告
    .quality-stats {
        .quality-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 6px 0;
            border-bottom: 1px solid var(--el-border-color-lighter);
            &:last-child { border-bottom: none; }
        }
        .quality-label {
            font-size: 13px;
            color: var(--el-text-color-regular);
        }
        .quality-value {
            font-size: 14px;
            font-weight: 500;
        }
    }
    .fail-reasons-list {
        .fail-reason-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 4px 0;
            font-size: 12px;
        }
        .fail-reason-text {
            color: var(--el-text-color-regular);
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            flex: 1;
            margin-right: 8px;
        }
        .fail-reason-count {
            color: var(--el-color-danger);
            font-weight: 500;
            flex-shrink: 0;
        }
    }
}
</style>
