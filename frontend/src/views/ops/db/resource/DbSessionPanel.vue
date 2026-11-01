<template>
    <div class="db-session-panel h-full flex flex-col gap-1">
        <div class="flex items-center gap-2 flex-shrink-0">
            <el-button size="small" :loading="state.loading" icon="Refresh" @click="refresh">{{ t('common.refresh') }}</el-button>
            <el-tag size="small" type="info" effect="plain">{{ t('db.sessionContextDb') }}：{{ dbName }}</el-tag>
            <el-switch v-if="state.tab === 'sessions'" v-model="state.autoRefresh" size="small" :active-text="t('db.sessionAutoRefresh')" />
            <span class="text-[12px] text-gray-400">{{ t('db.sessionUpdatedAt', { time: state.updatedAt }) }}</span>
        </div>

        <el-tabs v-model="state.tab" class="session-tabs flex-1 min-h-0 flex flex-col">
            <!-- 活动会话 -->
            <el-tab-pane :label="t('db.sessionTabSessions')" name="sessions">
                <div class="h-full flex flex-col gap-1">
                    <!-- 不支持的方言：明确说明原因，而不是给一张看起来「没有会话」的空表 -->
                    <el-alert v-if="state.unsupported" :title="t('db.sessionUnsupported')" type="info" :closable="false" show-icon class="flex-shrink-0" />

                    <el-table v-loading="state.loading && !state.sessions.length" :data="state.sessions" size="small" height="100%" class="flex-1 min-h-0">
                        <el-table-column prop="id" :label="t('db.sessionId')" width="90" />
                        <el-table-column prop="user" :label="t('db.sessionUser')" width="130" show-overflow-tooltip />
                        <el-table-column prop="db" :label="t('db.dbName')" width="140" show-overflow-tooltip />
                        <el-table-column :label="t('db.sessionElapsed')" width="100">
                            <template #default="{ row }">{{ formatElapsed(row.elapsedMs) }}</template>
                        </el-table-column>
                        <el-table-column :label="t('db.sessionLockWait')" width="90">
                            <template #default="{ row }">
                                <el-tag v-if="row.lockWait" type="danger" size="small">{{ t('common.yes') }}</el-tag>
                                <span v-else>-</span>
                            </template>
                        </el-table-column>
                        <el-table-column prop="state" :label="t('db.sessionState')" width="150" show-overflow-tooltip />
                        <el-table-column prop="query" :label="t('db.sessionQuery')" min-width="260" show-overflow-tooltip />
                        <el-table-column :label="t('common.operation')" width="90" fixed="right">
                            <template #default="{ row }">
                                <el-button v-auth="'db:session:cancel'" type="warning" link size="small" :disabled="state.canceling" @click="cancelSession(row, 'sessions')">
                                    {{ t('db.sessionCancel') }}
                                </el-button>
                            </template>
                        </el-table-column>
                        <template #empty>
                            <span class="text-[12px] text-gray-400">{{ state.unsupported ? t('db.sessionUnsupported') : t('db.sessionEmpty') }}</span>
                        </template>
                    </el-table>
                </div>
            </el-tab-pane>

            <!-- 慢查询聚合 -->
            <el-tab-pane :label="t('db.sessionTabSlow')" name="slow">
                <div class="h-full flex flex-col gap-1">
                    <!-- 拿不到统计源时给出原因与处置方向，而不是一张空表 -->
                    <el-alert v-if="state.slowError" :title="state.slowError" type="info" :closable="false" show-icon class="flex-shrink-0" />
                    <el-table v-loading="state.slowLoading" :data="state.slowQueries" size="small" height="100%" class="flex-1 min-h-0">
                        <el-table-column prop="sql" :label="t('db.sessionQuery')" min-width="320" show-overflow-tooltip />
                        <el-table-column prop="schema" :label="t('db.dbName')" width="130" show-overflow-tooltip />
                        <el-table-column prop="calls" :label="t('db.slowCalls')" width="90" align="right" />
                        <el-table-column :label="t('db.slowAvg')" width="100" align="right">
                            <template #default="{ row }">{{ formatMs(row.avgMs) }}</template>
                        </el-table-column>
                        <el-table-column :label="t('db.slowMax')" width="100" align="right">
                            <template #default="{ row }">{{ formatMs(row.maxMs) }}</template>
                        </el-table-column>
                        <el-table-column :label="t('db.slowTotal')" width="110" align="right">
                            <template #default="{ row }">{{ formatMs(row.totalMs) }}</template>
                        </el-table-column>
                        <el-table-column prop="rowsAvg" :label="t('db.slowRows')" width="90" align="right" />
                        <el-table-column :label="t('common.operation')" width="80" fixed="right">
                            <template #default="{ row }">
                                <el-link type="primary" underline="never" @click="copySlowSql(row.sql)">{{ t('common.copy') }}</el-link>
                            </template>
                        </el-table-column>
                        <template #empty>
                            <span class="text-[12px] text-gray-400">{{ state.slowError ? t('db.slowUnavailable') : t('db.slowEmpty') }}</span>
                        </template>
                    </el-table>
                </div>
            </el-tab-pane>

            <!-- 锁等待链 -->
            <el-tab-pane :label="t('db.sessionTabLocks')" name="locks">
                <div class="h-full flex flex-col gap-1">
                    <el-alert v-if="state.lockError" :title="state.lockError" type="info" :closable="false" show-icon class="flex-shrink-0" />
                    <el-table v-loading="state.lockLoading" :data="state.lockWaits" size="small" height="100%" class="flex-1 min-h-0">
                        <el-table-column :label="t('db.lockWaiter')" min-width="200">
                            <template #default="{ row }">
                                <div class="flex flex-col">
                                    <span>{{ row.waitSessionId }} · {{ row.waitUser || '-' }} · {{ formatElapsed(row.waitMs) }}</span>
                                    <span class="text-[11px] text-gray-400">{{ truncate(row.waitQuery) }}</span>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column :label="t('db.lockBlockedBy')" min-width="200">
                            <template #default="{ row }">
                                <div class="flex flex-col">
                                    <span>{{ row.blockSessionId }} · {{ row.blockUser || '-' }}</span>
                                    <span class="text-[11px] text-gray-400">{{ truncate(row.blockQuery) }}</span>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column prop="lockType" :label="t('db.lockType')" min-width="180" show-overflow-tooltip />
                        <el-table-column :label="t('common.operation')" width="120" fixed="right">
                            <template #default="{ row }">
                                <!-- 处置的是阻塞源：只停等待方不解决问题，源头还在持锁 -->
                                <el-button
                                    v-auth="'db:session:cancel'"
                                    type="warning"
                                    link
                                    size="small"
                                    :disabled="state.canceling"
                                    @click="cancelSession({ id: row.blockSessionId, query: row.blockQuery }, 'locks')"
                                >
                                    {{ t('db.lockKillBlocker') }}
                                </el-button>
                            </template>
                        </el-table-column>
                        <template #empty>
                            <span class="text-[12px] text-gray-400">{{ state.lockError ? t('db.lockUnavailable') : t('db.lockEmpty') }}</span>
                        </template>
                    </el-table>
                </div>
            </el-tab-pane>
            <!-- 指标趋势：来自平台采样表，库不可达时历史曲线仍可看 -->
            <el-tab-pane :label="t('db.sessionTabMetrics')" name="metrics">
                <div class="h-full flex flex-col gap-1">
                    <div class="flex items-center gap-2 flex-shrink-0">
                        <el-select v-model="state.metric" size="small" class="w-52!" @change="loadMetrics">
                            <el-option v-for="item in metricOptions" :key="item.value" :label="t(item.label)" :value="item.value" />
                        </el-select>
                        <el-select v-model="state.metricMinutes" size="small" class="w-36!" @change="loadMetrics">
                            <el-option v-for="item in rangeOptions" :key="item.value" :label="t(item.label)" :value="item.value" />
                        </el-select>
                        <span class="text-xs text-gray-400">{{ t('db.metricSampleTip') }}</span>
                    </div>
                    <el-alert v-if="state.metricError" :title="state.metricError" type="info" :closable="false" show-icon class="flex-shrink-0" />
                    <div v-loading="state.metricLoading" class="flex-1 min-h-0">
                        <ECharts v-if="metricOption" :option="metricOption" height="360" />
                        <div v-else class="h-full flex items-center justify-center text-xs text-gray-400">{{ t('db.metricEmpty') }}</div>
                    </div>
                </div>
            </el-tab-pane>
            <!-- 恢复能力：PITR 前提是数据库侧开关，平台只能如实探测；不可用时必须给原因 -->
            <el-tab-pane :label="t('db.sessionTabRecovery')" name="recovery">
                <div v-loading="state.recoveryLoading" class="text-sm">
                    <el-alert v-if="state.recoveryError" :title="state.recoveryError" type="warning" :closable="false" show-icon />
                    <template v-else-if="state.recovery">
                        <el-alert
                            :title="state.recovery.prepared ? t('db.recoveryPrepared') : t('db.recoveryNotPrepared')"
                            :type="state.recovery.prepared ? 'success' : 'warning'"
                            :closable="false"
                            show-icon
                        />
                        <el-descriptions :column="1" border class="mt-2">
                            <el-descriptions-item :label="t('db.recoveryMode')">{{ recoverModeText }}</el-descriptions-item>
                            <el-descriptions-item v-if="state.recovery.reason" :label="t('db.recoveryReason')">
                                {{ state.recovery.reason }}
                            </el-descriptions-item>
                            <el-descriptions-item v-if="state.recovery.oldestPoint" :label="t('db.recoveryOldest')">
                                {{ state.recovery.oldestPoint }}
                            </el-descriptions-item>
                            <el-descriptions-item v-if="state.recovery.latestPoint" :label="t('db.recoveryLatest')">
                                {{ state.recovery.latestPoint }}
                            </el-descriptions-item>
                            <el-descriptions-item v-if="state.recovery.retention" :label="t('db.recoveryRetention')">
                                {{ state.recovery.retention }}
                            </el-descriptions-item>
                        </el-descriptions>
                        <div class="mt-2 text-xs text-gray-400">{{ t('db.recoveryTip') }}</div>
                    </template>
                </div>
            </el-tab-pane>
            <!-- 账号盘点：只读。平台不代改账号，只把「有哪些、哪些高危」摊开 -->
            <el-tab-pane :label="t('db.sessionTabAccounts')" name="accounts">
                <div v-loading="state.accountsLoading" class="h-full flex flex-col gap-1">
                    <el-alert v-if="state.accountsError" :title="state.accountsError" type="warning" :closable="false" show-icon />
                    <template v-else-if="state.accounts">
                        <div class="flex items-center gap-2 flex-shrink-0">
                            <span class="text-xs text-gray-500">{{ t('db.accountTotal', { total: state.accounts.accounts.length }) }}</span>
                            <el-tag v-if="highRisks.length" type="danger" size="small">{{ t('db.accountRiskHighCount', { count: highRisks.length }) }}</el-tag>
                            <el-tag v-if="warnRisks.length" type="warning" size="small">{{ t('db.accountRiskWarnCount', { count: warnRisks.length }) }}</el-tag>
                        </div>
                        <el-table :data="accountRows" size="small" border height="100%" class="flex-1">
                            <el-table-column prop="name" :label="t('db.accountName')" min-width="140" />
                            <el-table-column prop="host" :label="t('db.accountHost')" width="120">
                                <template #default="{ row }">{{ row.host || '-' }}</template>
                            </el-table-column>
                            <el-table-column :label="t('db.accountFlags')" min-width="260">
                                <template #default="{ row }">
                                    <el-tag v-if="row.superuser" type="danger" size="small" class="mr-1">{{ t('db.accountSuper') }}</el-tag>
                                    <el-tag v-if="row.canGrant" type="warning" size="small" class="mr-1">{{ t('db.accountGrant') }}</el-tag>
                                    <el-tag v-if="row.passwordExpired" type="danger" size="small" class="mr-1">{{ t('db.accountPwdExpired') }}</el-tag>
                                    <el-tag v-if="row.locked" type="info" size="small" class="mr-1">{{ t('db.accountLocked') }}</el-tag>
                                    <el-tag v-if="row.neverExpire && row.canLogin" type="warning" size="small" class="mr-1">{{ t('db.accountNeverExpire') }}</el-tag>
                                    <el-tag v-if="!row.canLogin" type="info" size="small">{{ t('db.accountNoLogin') }}</el-tag>
                                </template>
                            </el-table-column>
                            <el-table-column prop="lastChanged" :label="t('db.accountLastChanged')" width="180">
                                <template #default="{ row }">{{ row.lastChanged || '-' }}</template>
                            </el-table-column>
                        </el-table>
                    </template>
                </div>
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script lang="ts" setup>
/**
 * 数据库诊断面板（库工作区的一个页签）：活动会话 / 慢查询聚合 / 锁等待链。
 *
 * 存在理由：前端 abort 只断 HTTP，长查询在数据库侧仍在跑；没有会话视图就既看不到「谁在跑什么」，
 * 也无法停掉它。三个页签的数据全部来自库侧系统表/统计表（由方言的可选能力给出），本组件不做推断。
 *
 * 为什么慢查询与锁等待不自动刷新：两者都要读 performance_schema 的聚合表，轮询代价远高于
 * processlist，且变化不像会话那样秒级；只在切到页签与点刷新时查一次。
 */
import { computed, onBeforeUnmount, reactive, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useIntervalFn } from '@vueuse/core';
import { dbApi } from '../api';
import ECharts from '@/components/echarts/ECharts.vue';
import { createPanelLoader, toErrorMessage } from './composables/panelLoader';
import { copyToClipboard } from '@/common/utils/string';
import type {
    DbAccountOverview,
    DbLockWait,
    DbMetricSeries,
    DbRecoveryReadiness,
    DbSessionInfo,
    DbSlowQuery,
} from '../types';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { formatTime } from '@/common/utils/format';

const props = defineProps<{
    dbId: number;
    dbName: string;
}>();

const { t } = useI18n();

const state = reactive({
    tab: 'sessions',
    sessions: [] as DbSessionInfo[],
    loading: false,
    // 失败原因是否为「该方言不支持会话内省」：为 true 时给出说明，而不是留一张看起来「没有会话」的空表
    unsupported: false,
    autoRefresh: true,
    updatedAt: '',
    // 已有取消流程在走（包括确认框未关）：不允许再开第二个，否则会叠出多个确认框而旧框已陈旧
    canceling: false,

    slowQueries: [] as DbSlowQuery[],
    slowLoading: false,
    // 慢查询拿不到数据源时的原因（未开启采集/版本不支持），与「没有慢查询」必须区分
    slowError: '',

    lockWaits: [] as DbLockWait[],
    lockLoading: false,
    lockError: '',

    metric: 'sessions_active',
    metricMinutes: 60,
    accounts: null as DbAccountOverview | null,
    accountsLoading: false,
    accountsError: '',

    recovery: null as DbRecoveryReadiness | null,
    recoveryLoading: false,
    recoveryError: '',

    metricPoints: [] as { ts: number; value: number }[],
    metricLoading: false,
    metricError: '',
});

// 自动刷新：面板卸载即停（onBeforeUnmount），不会在后台反复查库侧系统表。
// load 内有 loading 互斥，首轮加载未完成时轮询到的一轮会被直接跳过，不会并发打库
const { pause, resume } = useIntervalFn(() => void load(true), 3000, { immediate: true });

watch(
    () => [props.dbId, props.dbName],
    () => void load(),
    { immediate: true }
);

watch(
    () => state.autoRefresh,
    (on) => (on ? resume() : pause())
);

/** 按当前页签刷新：顶栏只有一个刷新入口，避免三个页签各挂一个按钮 */
function refresh() {
    if (state.tab === 'sessions') {
        void load();
        return;
    }
    if (state.tab === 'slow') {
        void loadSlowQueries();
        return;
    }
    if (state.tab === 'metrics') {
        void loadMetrics();
        return;
    }
    if (state.tab === 'recovery') {
        void loadRecovery();
        return;
    }
    if (state.tab === 'accounts') {
        void loadAccounts();
        return;
    }
    void loadLockWaits();
}

/**
 * 拉取会话列表
 *
 * @param silent 自动刷新时不铺 loading、不弹错误提示（否则每 3 秒闪一次）
 */
async function load(silent = false) {
    if (state.loading) {
        return;
    }
    state.loading = true;
    try {
        const sessions = await dbApi.dbSessions.request({ dbId: props.dbId, db: props.dbName });
        state.sessions = sessions ?? [];
        state.unsupported = false;
        state.updatedAt = new Date().toLocaleTimeString();
    } catch (e: unknown) {
        // 能力缺失与真实查询失败要分开呈现：前者是「这个库看不了」，后者必须让人看到错误本身
        state.unsupported = isUnsupportedErr(e);
        state.sessions = [];
        if (!silent && !state.unsupported) {
            Msg.error(e instanceof Error ? e.message : String(e));
        }
    } finally {
        state.loading = false;
    }
}

/** 慢查询：拿不到统计源时把服务端原因原样展示（它已经说明该开哪个开关） */
const loadSlowQueries = createPanelLoader<DbSlowQuery[]>({
    isLoading: () => state.slowLoading,
    setLoading: (value) => (state.slowLoading = value),
    setError: (message) => (state.slowError = message),
    fetch: () => dbApi.dbSlowQueries.request({ id: props.dbId, db: props.dbName, limit: 20 }),
    onSuccess: (rows) => {
        state.slowQueries = rows ?? [];
        state.updatedAt = new Date().toLocaleTimeString();
    },
    onError: (error) => {
        state.slowQueries = [];
        state.slowError = toErrorMessage(error);
    },
});

/** 锁等待链：同样把「不可用」与「没有锁等待」分开 */
const loadLockWaits = createPanelLoader<DbLockWait[]>({
    isLoading: () => state.lockLoading,
    setLoading: (value) => (state.lockLoading = value),
    setError: (message) => (state.lockError = message),
    fetch: () => dbApi.dbLockWaits.request({ id: props.dbId, db: props.dbName }),
    onSuccess: (rows) => {
        state.lockWaits = rows ?? [];
        state.updatedAt = new Date().toLocaleTimeString();
    },
    onError: (error) => {
        state.lockWaits = [];
        state.lockError = toErrorMessage(error);
    },
});

/** 账号风险等级权重：高危在前，无风险最后——列表默认顺序会让高危埋在中间 */
const accountRiskWeight = computed(() => {
    const weight = new Map<string, number>();
    for (const risk of state.accounts?.risks ?? []) {
        const current = weight.get(risk.account) ?? 3;
        const value = risk.level === 'high' ? 0 : risk.level === 'warn' ? 1 : 2;
        weight.set(risk.account, Math.min(current, value));
    }
    return weight;
});

/** 按风险排序后的账号行：排序只影响展示，不改后端返回 */
const accountRows = computed(() => {
    const rows = [...(state.accounts?.accounts ?? [])];
    return rows.sort((left, right) => {
        const diff = (accountRiskWeight.value.get(left.name) ?? 3) - (accountRiskWeight.value.get(right.name) ?? 3);
        return diff !== 0 ? diff : left.name.localeCompare(right.name);
    });
});

const highRisks = computed(() => state.accounts?.risks.filter((risk) => risk.level === 'high') ?? []);
const warnRisks = computed(() => state.accounts?.risks.filter((risk) => risk.level === 'warn') ?? []);

/** 账号盘点取数：只读，不改任何账号 */
const loadAccounts = createPanelLoader<DbAccountOverview>({
    isLoading: () => state.accountsLoading,
    setLoading: (value) => (state.accountsLoading = value),
    setError: (message) => (state.accountsError = message),
    fetch: () => dbApi.dbAccounts.request({ id: props.dbId, db: props.dbName }),
    onSuccess: (overview) => {
        state.accounts = overview;
        state.updatedAt = new Date().toLocaleTimeString();
    },
    onError: (error) => {
        state.accounts = null;
        // 读不到 mysql.user 很常见（权限不足）：原因照实显示，不给空表
        state.accountsError = toErrorMessage(error);
    },
});

/** 恢复方式文案：none 表示数据库侧没开归档，平台只能恢复到逻辑备份时刻 */
const recoverModeText = computed(() => {
    const mode = state.recovery?.mode ?? 'none';
    return t(`db.recoveryMode_${mode}`);
});

/** 恢复能力取数：读数据库侧的 binlog / WAL 归档事实 */
const loadRecovery = createPanelLoader<DbRecoveryReadiness>({
    isLoading: () => state.recoveryLoading,
    setLoading: (value) => (state.recoveryLoading = value),
    setError: (message) => (state.recoveryError = message),
    fetch: () => dbApi.dbRecoveryReadiness.request({ id: props.dbId, db: props.dbName }),
    onSuccess: (readiness) => {
        state.recovery = readiness;
        state.updatedAt = new Date().toLocaleTimeString();
    },
    onError: (error) => {
        state.recovery = null;
        state.recoveryError = toErrorMessage(error);
    },
});

/** 指标名与展示文案：与后端 metricNames 白名单一一对应 */
const metricOptions = [
    { value: 'sessions_active', label: 'db.metricSessionsActive' },
    { value: 'session_max_elapsed', label: 'db.metricSessionMaxElapsed' },
    { value: 'lock_wait_count', label: 'db.metricLockWaitCount' },
    { value: 'ping_fail', label: 'db.metricPingFail' },
    { value: 'tls_cert_days', label: 'db.metricTLSCertDays' },
    { value: 'schema_drift_tables', label: 'db.metricSchemaDrift' },
    { value: 'pitr_ready', label: 'db.metricPitrReady' },
];

const rangeOptions = [
    { value: 60, label: 'db.metricRange1h' },
    { value: 360, label: 'db.metricRange6h' },
    { value: 1440, label: 'db.metricRange24h' },
    { value: 10080, label: 'db.metricRange7d' },
];

/** 趋势图配置：数据点由后端抽稀（≤720 点），这里只负责呈现 */
const metricOption = computed(() => {
    if (!state.metricPoints.length) {
        return null;
    }
    return {
        grid: { left: 8, right: 16, top: 24, bottom: 8, containLabel: true },
        // as const：ECharts 的 trigger 是字面量联合类型，裸字符串会被推成 string
        tooltip: { trigger: 'axis' as const },
        xAxis: { type: 'time' as const },
        yAxis: { type: 'value' as const, scale: true },
        series: [
            {
                name: t(metricOptions.find((item) => item.value === state.metric)?.label ?? 'db.sessionTabMetrics'),
                type: 'line' as const,
                showSymbol: false,
                smooth: true,
                areaStyle: { opacity: 0.12 },
                data: state.metricPoints.map((point) => [point.ts, point.value]),
            },
        ],
    };
});

/** 指标趋势取数：读平台采样表，不连接目标库 */
const loadMetrics = createPanelLoader<DbMetricSeries>({
    isLoading: () => state.metricLoading,
    setLoading: (value) => (state.metricLoading = value),
    setError: (message) => (state.metricError = message),
    fetch: () => dbApi.dbMetrics.request({ id: props.dbId, metric: state.metric, minutes: state.metricMinutes }),
    onSuccess: (series) => {
        state.metricPoints = series?.points ?? [];
        state.updatedAt = new Date().toLocaleTimeString();
    },
    onError: (error) => {
        state.metricPoints = [];
        state.metricError = toErrorMessage(error);
    },
});

watch(
    () => state.tab,
    (tab) => {
        if (tab === 'slow') {
            void loadSlowQueries();
        }
        if (tab === 'locks') {
            void loadLockWaits();
        }
        if (tab === 'metrics') {
            void loadMetrics();
        }
        if (tab === 'recovery') {
            void loadRecovery();
        }
        if (tab === 'accounts') {
            void loadAccounts();
        }
    }
);

function isUnsupportedErr(e: unknown): boolean {
    return (e instanceof Error ? e.message : String(e)).toLowerCase().includes('unsupported');
}

/** 已执行时长：ms 交给既有的秒级人类可读格式化（8s / 1m20s），不在此重写单位换算 */
function formatElapsed(ms: number): string {
    return formatTime(Math.round((ms ?? 0) / 1000), 's').trim() || '-';
}

/**
 * 终止会话
 *
 * @param source 触发来源：锁等待页签终止的是阻塞源，成功后要刷新锁列表而不是会话列表
 */
async function cancelSession(row: { id: string; query?: string }, source: 'sessions' | 'locks') {
    // 同时只允许一个取消确认框：列表在自动刷新，连着点两行会叠出多个确认框，
    // 而旧框里的会话可能已经结束，误点就是把错误的语句被中断
    if (state.canceling) {
        return;
    }
    state.canceling = true;
    try {
        // 取消是危险动作（会中断别人的业务语句）：确认框里必须看清是哪条语句
        if (!(await useI18nConfirm('db.sessionCancelConfirm', { id: row.id, query: truncate(row.query) }))) {
            return;
        }
        await dbApi.cancelDbSession.request({ id: props.dbId, db: props.dbName, sessionId: row.id });
        Msg.success('db.sessionCancelSuccess');
    } catch {
        // 失败文案由请求中心层 toast（服务端已把库侧拒绝原因包成人话），这里不叠第二条；
        // 不 catch 则成了未处理的 rejection，界面看起来就是点了没反应
    } finally {
        state.canceling = false;
        // 无论成败都重取一次：失败往往正因为该行已是陈旧数据
        if (source === 'locks') {
            void loadLockWaits();
        } else {
            void load(true);
        }
    }
}

/**
 * 慢查询耗时格式化（毫秒档）。
 *
 * 不能用 formatElapsed：它按秒取整，几十到几百毫秒的语句会变成 0 秒再被压成 "-"，
 * 于是"最慢的语句"看起来像"没有耗时"。
 */
function formatMs(ms?: number): string {
    const value = Math.max(0, Math.round(ms ?? 0));
    if (!value) {
        return '-';
    }
    if (value < 1000) {
        return `${value}ms`;
    }
    if (value < 60000) {
        return `${(value / 1000).toFixed(1)}s`;
    }
    return formatTime(Math.round(value / 1000), 's').trim() || `${value}ms`;
}

/** 复制慢查询语句：诊断之后最常见的动作就是拿去优化或看执行计划（提示由公共工具统一弹） */
function copySlowSql(sql?: string) {
    if (!sql) {
        return;
    }
    void copyToClipboard(sql);
}

function truncate(text?: string): string {
    const value = (text ?? '').trim();
    return value.length > 80 ? `${value.slice(0, 80)}...` : value;
}

onBeforeUnmount(pause);
</script>

<style lang="scss" scoped>
.db-session-panel {
    padding: 4px;
}

/* 表格要撑满剩余高度：页签内容区本身是 flex 容器，不显式传高度会塌成内容高 */
.session-tabs {
    :deep(.el-tabs__content) {
        flex: 1;
        min-height: 0;
    }

    :deep(.el-tab-pane) {
        height: 100%;
    }
}
</style>

/* 不可用提示带「启用步骤」第二行：保留换行，否则会挤成一句长文 */
.db-session-panel :deep(.el-alert__title) {
    white-space: pre-line;
}
