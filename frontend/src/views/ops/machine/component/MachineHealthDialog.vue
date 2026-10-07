<template>
    <el-dialog v-model="visible" :title="$t('machine.healthOverview')" width="1080px" :close-on-click-modal="false">
        <div class="mb-3 flex items-center justify-between gap-3">
            <div class="flex flex-wrap items-center gap-2">
                <template v-if="abnormalCount || offlineCount || unknownCount">
                    <el-tag v-if="offlineCount" type="danger" effect="dark" size="small">
                        {{ $t('machine.healthSummaryOffline', { n: offlineCount }) }}
                    </el-tag>
                    <el-tag v-if="abnormalCount" type="warning" effect="dark" size="small">
                        {{ $t('machine.healthSummaryAbnormal', { n: abnormalCount }) }}
                    </el-tag>
                    <el-tag v-if="unknownCount" type="info" effect="plain" size="small">
                        {{ $t('machine.healthSummaryUnknown', { n: unknownCount }) }}
                    </el-tag>
                </template>
                <el-tag v-else type="success" effect="plain" size="small">{{ $t('machine.healthAllOk') }}</el-tag>
            </div>
            <div class="flex shrink-0 items-center gap-3">
                <el-checkbox v-model="onlyAbnormal" size="small">{{ $t('machine.healthOnlyAbnormal') }}</el-checkbox>
                <el-button icon="refresh" circle plain size="small" @click="load" />
            </div>
        </div>

        <el-table v-loading="loading" :data="rows" size="small" stripe max-height="480">
            <el-table-column :label="$t('machine.healthTriageColumn')" width="96" align="center">
                <template #default="{ row }">
                    <el-tag :type="triageTagType(row)" effect="light" size="small">{{ triageLabel(row) }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="name" :label="$t('machine.name')" min-width="120" show-overflow-tooltip />
            <el-table-column :label="$t('machine.ipAndPort')" min-width="150">
                <template #default="{ row }">
                    <span class="font-mono text-xs">{{ `${row.ip}:${row.port}` }}</span>
                </template>
            </el-table-column>
            <el-table-column :label="$t('machine.status')" width="80" align="center">
                <template #default="{ row }">
                    <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="plain" size="small">
                        {{ row.status === 1 ? $t('machine.healthOnline') : $t('machine.healthOffline') }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column label="CPU" min-width="150">
                <template #default="{ row }">
                    <el-progress :percentage="pct(row.cpuUsage)" :status="metricStatus(row, 'cpu_rate')" :stroke-width="10" />
                </template>
            </el-table-column>
            <el-table-column :label="$t('machine.memory')" min-width="150">
                <template #default="{ row }">
                    <el-progress :percentage="pct(row.memUsage)" :status="metricStatus(row, 'mem_rate')" :stroke-width="10" />
                </template>
            </el-table-column>
            <el-table-column :label="$t('machine.disk')" min-width="150">
                <template #default="{ row }">
                    <el-progress :percentage="pct(row.diskUsage)" :status="metricStatus(row, 'disk_usage')" :stroke-width="10" />
                </template>
            </el-table-column>
            <el-table-column :label="$t('machine.lastCollect')" min-width="170">
                <template #default="{ row }">
                    <span v-if="row.collectTime">{{ formatDate(row.collectTime) }}</span>
                    <span v-else class="text-xs text-gray-400">{{ $t('machine.healthNoCollection') }}</span>
                </template>
            </el-table-column>
            <el-table-column :label="$t('common.operation')" width="100" align="center" fixed="right">
                <template #default="{ row }">
                    <el-button type="primary" link @click="emit('view-trend', row)">{{ $t('machine.healthViewTrend') }}</el-button>
                </template>
            </el-table-column>
            <template #expand="{ row }">
                <div v-if="row.hits?.length" class="px-6 py-2">
                    <div class="mb-1 text-xs text-gray-500">{{ $t('machine.healthHitTip') }}</div>
                    <div v-for="(hit, i) in row.hits" :key="i" class="text-xs leading-6">
                        <el-tag size="small" effect="plain">{{ metricLabel(hit.metric) }}</el-tag>
                        <span class="ml-2 font-mono">{{ `${compareSymbol(hit.compare)} ${fmtNum(hit.threshold)}` }}</span>
                        <span class="ml-2 text-gray-500">{{ `→ ${fmtNum(hit.current)}` }}</span>
                        <span class="ml-3">{{ hit.ruleName }}</span>
                        <el-tag class="ml-2" size="small" :type="priorityTagType(hit.priority)" effect="light">
                            {{ priorityLabel(hit.priority) }}
                        </el-tag>
                    </div>
                </div>
            </template>
        </el-table>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { metricApi } from '../api';
import { formatDate } from '@/common/utils/format';
import type { MachineHealth, MachineHealthHit } from '../types';

const { t } = useI18n();

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits<{ 'view-trend': [health: MachineHealth] }>();

const loading = ref(false);
const list = ref<MachineHealth[]>([]);
const onlyAbnormal = ref(false);

/**
 * 分诊序：离线最前，其次按命中规则优先级（P0 最优先），再次是「未判定」，最后是正常。
 *
 * 「未判定」排在正常之前：取不到当前值不等于健康，排在后面会被当成没问题扫过去。
 */
const rank = (row: MachineHealth) => {
    if (row.status !== 1) return 0;
    if (row.priority >= 0) return 1 + row.priority;
    return row.triaged === false ? 5 : 6;
};

const sorted = computed(() => [...list.value].sort((a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name)));

const rows = computed(() => (onlyAbnormal.value ? sorted.value.filter((r) => rank(r) <= 5) : sorted.value));

const offlineCount = computed(() => list.value.filter((r) => r.status !== 1).length);
const abnormalCount = computed(() => list.value.filter((r) => r.status === 1 && r.priority >= 0).length);
const unknownCount = computed(() => list.value.filter((r) => r.status === 1 && r.priority < 0 && r.triaged === false).length);

// 使用率百分比取整（0-100），供 el-progress 使用；接口数值可能因 json-bigint 回传成字符串，先收敛为 number
const pct = (v: number) => {
    const n = Number(v);
    return Math.max(0, Math.min(100, Math.round(Number.isFinite(n) ? n : 0)));
};

const hitOf = (row: MachineHealth, metric: string): MachineHealthHit | undefined =>
    row.hits?.find((h) => h.metric === metric);

// 越线的指标标红，未判定的标黄：颜色只表达「要不要看」，具体阈值在展开行里
const metricStatus = (row: MachineHealth, metric: string): 'exception' | 'warning' | undefined => {
    if (hitOf(row, metric)) return 'exception';
    return row.triaged === false ? 'warning' : undefined;
};

const triageLabel = (row: MachineHealth) => {
    if (row.status !== 1) return t('machine.healthOffline');
    if (row.priority >= 0) return priorityLabel(row.priority);
    return t('machine.healthNormal');
};

type TagType = 'danger' | 'warning' | 'info' | 'success';

const priorityTagType = (priority: number): TagType => (priority <= 1 ? 'danger' : priority === 2 ? 'warning' : 'info');

const triageTagType = (row: MachineHealth): TagType => {
    if (row.status !== 1) return 'danger';
    if (row.priority >= 0) return priorityTagType(row.priority);
    return row.triaged === false ? 'info' : 'success';
};

const priorityLabel = (priority: number) =>
    [t('alert.priorityCritical'), t('alert.priorityHigh'), t('alert.priorityMedium'), t('alert.priorityLow')][priority] ?? '';

// 指标展示名与告警侧同源，避免总览与告警规则页对同一指标叫法不一致
const metricLabel = (metric: string) =>
    ({ cpu_rate: t('alert.metricCpuRate'), mem_rate: t('alert.metricMemRate'), disk_usage: t('alert.metricDiskUsage'), status: t('alert.metricStatus') })[
        metric
    ] ?? metric;

const compareSymbol = (compare: string) =>
    ({ gt: '>', gte: '≥', lt: '<', lte: '≤', eq: '=', neq: '≠' })[compare] ?? compare;

const fmtNum = (v: number) => {
    const n = Number(v);
    return Number.isFinite(n) ? String(Math.round(n * 10) / 10) : '-';
};

const load = async () => {
    loading.value = true;
    try {
        list.value = (await metricApi.healthOverview.request()) || [];
    } finally {
        loading.value = false;
    }
};

watch(visible, (val) => {
    if (val) load();
});
</script>
