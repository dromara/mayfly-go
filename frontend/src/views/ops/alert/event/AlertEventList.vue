<template>
    <div class="alert-event-page">
        <!-- 统计卡片 -->
        <el-row :gutter="16" class="stat-cards mb-4">
            <el-col :xs="12" :sm="6">
                <div class="stat-card firing">
                    <div class="stat-value">{{ overview.firingCount }}</div>
                    <div class="stat-label">{{ $t('alert.firing') }}</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="6">
                <div class="stat-card acknowledged">
                    <div class="stat-value">{{ overview.acknowledgedCount }}</div>
                    <div class="stat-label">{{ $t('alert.acknowledged') }}</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="6">
                <div class="stat-card recovered">
                    <div class="stat-value">{{ overview.recoveredCount }}</div>
                    <div class="stat-label">{{ $t('alert.recovered') }}</div>
                </div>
            </el-col>
            <el-col :xs="12" :sm="6">
                <div class="stat-card closed">
                    <div class="stat-value">{{ overview.closedCount }}</div>
                    <div class="stat-label">{{ $t('alert.closed') }}</div>
                </div>
            </el-col>
        </el-row>

        <!-- 告警事件列表 -->
        <div class="table-container">
            <page-table ref="pageTableRef" :page-api="alertEventApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns">
            <template #resourceName="{ data }">
                {{ data.resourceName || `${getResourceTypeLabel(data.resourceType)}#${data.resourceId}` }}
            </template>

            <template #metric="{ data }">
                {{ getMetricLabel(data.metric) }}
            </template>

            <template #currentValue="{ data }">
                {{ formatMetricValue(data.metric, data.currentValue) }}
            </template>

            <template #threshold="{ data }">
                {{ formatThreshold(data.threshold) }}
            </template>

            <template #priority="{ data }">
                <enum-tag :enums="AlertPriorityEnum" :value="data.priority" />
            </template>

            <template #status="{ data }">
                <enum-tag :enums="AlertEventStatusEnum" :value="data.status" />
            </template>

            <template #action="{ data }">
                <el-button link v-if="data.status === AlertEventStatusFiring" @click="onAck(data)" type="primary">{{ $t('alert.confirm') }}</el-button>
                <el-button
                    link
                    v-if="data.status === AlertEventStatusFiring || data.status === AlertEventStatusAcknowledged || data.status === AlertEventStatusRecovered"
                    @click="onClose(data)"
                    type="warning"
                >
                    {{ $t('common.close') }}
                </el-button>
                <el-button
                    link
                    v-if="data.status === AlertEventStatusRecovered || data.status === AlertEventStatusClosed"
                    @click="onDelete(data)"
                    type="danger"
                >
                    {{ $t('common.delete') }}
                </el-button>
            </template>
        </page-table>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { onBeforeUnmount, onMounted, ref, useTemplateRef } from 'vue';
import { alertEventApi, alertOverviewApi } from '../api';
import { AlertEventStatusEnum, AlertPriorityEnum } from '../enums';
import type { AlertEventVO, AlertOverviewData } from '../types';
import { formatMetricValue, getResourceTypeLabel, getMetricLabel, formatThreshold } from '../utils';
import EnumTag from '@/components/enum-tag/EnumTag.vue';

// 概览为只读统计，定时刷新避免用户长时间停留在页面上看到过期数据
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
        todaySent: 0,
        successRate: 0,
        policyMatched: 0,
        totalEvents: 0,
        unmatchedCount: 0,
        channelDistribution: {},
        todaySuccess: 0,
        todayFailed: 0,
        byChannel: {},
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
        avgSendDelay: 0,
        maxSendDelay: 0,
        queueLength: 0,
        weeklySuccess: 0,
        monthlySuccess: 0,
        topFailReasons: [],
    },
});

const loadOverview = async () => {
    try {
        overview.value = await alertOverviewApi.overview.request({});
    } catch (e) {
        console.error('load overview error', e);
    }
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

// 状态字面量集中引用，避免模板中与后端枚举漂移的魔法数字
const AlertEventStatusFiring = AlertEventStatusEnum.Firing.value;
const AlertEventStatusAcknowledged = AlertEventStatusEnum.Acknowledged.value;
const AlertEventStatusRecovered = AlertEventStatusEnum.Recovered.value;
const AlertEventStatusClosed = AlertEventStatusEnum.Closed.value;

const searchItems = [
    SearchItem.input('keyword', 'alert.ruleName').withPlaceholder('common.keyword'),
    SearchItem.select('status', 'alert.status').withOptions(Object.values(AlertEventStatusEnum).map((e) => ({ value: e.value, label: e.label }))),
    SearchItem.select('priority', 'alert.priority').withOptions(Object.values(AlertPriorityEnum).map((e) => ({ value: e.value, label: e.label }))),
];

const columns = [
    TableColumn.new('ruleName', 'alert.ruleName'),
    TableColumn.new('resourceName', 'alert.resourceName').isSlot(),
    TableColumn.new('metric', 'alert.metricName').isSlot(),
    TableColumn.new('currentValue', 'alert.currentValue').isSlot().alignCenter(),
    TableColumn.new('threshold', 'alert.thresholdDesc').isSlot(),
    TableColumn.new('priority', 'alert.priority').isSlot().alignCenter(),
    TableColumn.new('status', 'alert.status').isSlot().alignCenter(),
    TableColumn.new('triggerCount', 'alert.triggerCountLabel').alignCenter(),
    TableColumn.new('notifyCount', 'alert.notifyCount').alignCenter(),
    TableColumn.new('escalationLevel', 'alert.escalationLevelCount').alignCenter(),
    TableColumn.new('firstTriggerTime', 'alert.firstTriggerTime').isTime(),
    TableColumn.new('lastTriggerTime', 'alert.lastTriggerTime').isTime(),
    TableColumn.new('recoverTime', 'alert.recoverTime').isTime(),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(120).alignCenter(),
];

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const query = ref({
    keyword: '',
    status: undefined as number | undefined,
    priority: undefined as number | undefined,
    pageNum: 1,
    pageSize: 0,
});

const onAck = async (row: AlertEventVO) => {
    await useI18nConfirm('alert.confirmAck');
    await alertEventApi.ack.request({ id: row.id });
    Msg.operateSuccess();
    pageTableRef.value?.search();
};

const onClose = async (row: AlertEventVO) => {
    await useI18nConfirm('alert.confirmClose');
    await alertEventApi.close.request({ id: row.id });
    Msg.operateSuccess();
    pageTableRef.value?.search();
};

const onDelete = async (row: AlertEventVO) => {
    await useI18nConfirm('alert.confirmDeleteEvent');
    await alertEventApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    pageTableRef.value?.search();
};
</script>

<style lang="scss" scoped>
.alert-event-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-width: 0;
    overflow: hidden;
}

.stat-cards {
    flex-shrink: 0;
}

.table-container {
    flex: 1;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
}

.stat-card {
    background: #fff;
    border-radius: 8px;
    padding: 20px 16px;
    text-align: center;
    box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.06);
    transition: all 0.3s;

    &:hover {
        box-shadow: 0 4px 16px 0 rgba(0, 0, 0, 0.1);
        transform: translateY(-2px);
    }

    .stat-value {
        font-size: 32px;
        font-weight: 700;
        line-height: 1.2;
    }

    .stat-label {
        font-size: 13px;
        color: #909399;
        margin-top: 6px;
    }
}

.firing .stat-value {
    color: #f56c6c;
}
.acknowledged .stat-value {
    color: #e6a23c;
}
.recovered .stat-value {
    color: #67c23a;
}
.closed .stat-value {
    color: #909399;
}
</style>
