<template>
    <div class="sql-exec-res h-full!">
        <el-tabs class="h-full! w-full!" v-if="execResTabs.length > 0" @tab-remove="onRemoveTab" @tab-change="onTabChange" :model-value="activeTab">
            <el-tab-pane class="h-full! flex flex-col" closable v-for="dt in execResTabs" :label="dt.id" :name="dt.id" :key="dt.id">
                <template #label>
                    <el-popover :show-after="1000" placement="top-start" :title="$t('db.execInfo')" trigger="hover" :width="300">
                        <template #reference>
                            <div>
                                <span>
                                    <span v-if="dt.loading">
                                        <SvgIcon class="mb-0.5! is-loading" name="Loading" color="var(--el-color-primary)" />
                                    </span>
                                    <span v-else>
                                        <SvgIcon class="mb-0.5!" v-if="dt.notice && !dt.errorMsg" name="Warning" color="var(--el-color-warning)" />
                                        <SvgIcon class="mb-0.5!" v-else-if="dt.errorMsg" name="CircleClose" color="var(--el-color-error)" />
                                        <SvgIcon class="mb-0.5!" v-else name="CircleCheck" color="var(--el-color-success)" />
                                    </span>
                                </span>

                                <span> {{ $t('db.result') }}-{{ dt.id }} </span>
                            </div>
                        </template>
                        <template #default>
                            <el-descriptions v-if="dt.sql" :column="1" size="small">
                                <el-descriptions-item>
                                    <div style="width: 280px">
                                        <el-text size="small" truncated :title="dt.sql"> {{ dt.sql }} </el-text>
                                    </div>
                                </el-descriptions-item>
                                <el-descriptions-item :label="`${$t('db.times')} :`"> {{ dt.execTime }}ms </el-descriptions-item>
                                <el-descriptions-item :label="`${$t('db.resultSet')} :`">
                                    {{ dt.data?.length }}
                                </el-descriptions-item>
                            </el-descriptions>
                        </template>
                    </el-popover>
                </template>

                <!-- 批量执行时逐条标注哪些语句被要求审批，并给出直接提单入口 -->
                <div v-if="needApprovalOf(dt).length" class="approval-tip">
                    <span>{{ $t('db.needApprovalTip', { count: needApprovalOf(dt).length }) }}</span>
                    <el-button link type="primary" @click="emit('submitTicket', needApprovalOf(dt).join(';\n'), dt)">
                        {{ $t('flow.submitTicket') }}
                    </el-button>
                </div>

                <!-- 截断必须说出口并给全量出口：只写「共 N 行」会被读成整张表就这些，
                     用户会拿一份缺尾巴的结果去做判断甚至导出去。颜色走 token 不写死色值 -->
                <div v-if="dt.truncated" class="approval-tip text-[var(--el-color-warning)]">
                    <span>{{ $t('db.resultTruncated', { n: dt.rowLimit }) }}</span>
                    <el-button v-if="dt.exportSql" link type="primary" :loading="exportingTab === dt.id" @click="exportFull(dt)">
                        {{ $t('db.exportAll') }}
                    </el-button>
                </div>

                <!-- 有结果行才能切图表：无数据时图表只会是一张空坐标系，切换本身也没意义 -->
                <div v-if="!dt.errorMsg && !dt.notice && dt.data?.length" class="mt-2 mb-1 ml-1 flex shrink-0 items-center gap-4">
                    <el-radio-group v-model="views[dt.id]" size="small">
                        <el-radio-button value="table">{{ $t('db.viewTable') }}</el-radio-button>
                        <el-radio-button value="chart">{{ $t('db.viewChart') }}</el-radio-button>
                    </el-radio-group>
                    <!-- 就地编辑的提交/取消紧跟视图切换右侧（左对齐不推到最右）：同行出现不顶动下方、也不与右下角悬浮球相撞 -->
                    <span v-if="dt.hasUpdatedFields" class="flex items-center">
                        <el-link type="success" underline="never" @click="emit('submitUpdateFields', dt)"
                            ><span style="font-size: 12px">{{ $t('common.submit') }}</span></el-link
                        >
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-link type="warning" underline="never" @click="emit('cancelUpdateFields', dt)"
                            ><span style="font-size: 12px">{{ $t('common.cancel') }}</span></el-link
                        >
                    </span>
                </div>

                <SqlResultChart
                    v-if="!dt.errorMsg && !dt.notice && views[dt.id] === 'chart'"
                    :columns="chartColumns(dt)"
                    :rows="dt.data"
                    class="ml-1 flex-1 min-h-0"
                />

                <db-table-data
                    v-else-if="!dt.errorMsg && !dt.notice"
                    class="shrink-0"
                    :ref="(el: unknown) => setDbTableRef(dt, el)"
                    :db-id="dbId"
                    :db="db"
                    :data="dt.data"
                    :table="dt.table"
                    :columns="dt.tableColumn"
                    :column-more-actions="['fixed']"
                    :loading="dt.loading"
                    :abort-fn="dt.abortFn"
                    :height="tableDataHeight"
                    :empty-text="tableDataEmptyText"
                    @change-updated-field="emit('changeUpdatedField', $event, dt)"
                    @data-delete="emit('dataDelete', $event, dt)"
                ></db-table-data>

                <!-- 命中「仅提醒」但选择不执行：黄色提醒而不是红色报错，因为它根本没失败 -->
                <el-result v-else-if="dt.notice" icon="warning" :title="$t('flow.warnAckTitle')" :sub-title="dt.notice">
                    <template #extra>
                        <el-button v-if="needApprovalOf(dt).length" type="primary" @click="onSubmitTicket(dt)">
                            {{ $t('flow.submitTicket') }}
                        </el-button>
                    </template>
                </el-result>

                <el-result v-else icon="error" :title="$t('db.execFail')" :sub-title="dt.errorMsg">
                    <template #extra>
                        <!-- 只有策略要求审批的失败才给提单入口：语法错误、权限不足提单也没用 -->
                        <el-button v-if="needApprovalOf(dt).length" type="primary" @click="onSubmitTicket(dt)">
                            {{ $t('flow.submitTicket') }}
                        </el-button>
                    </template>
                </el-result>
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, reactive, ref } from 'vue';
import SvgIcon from '@/components/svg-icon/index.vue';
import DbTableData from '@/views/ops/db/data-grid/DbTableData.vue';
import { exportQueryResult } from '../core/queryExport';
import type { ExecResTabLike, ExecResTabState, DbTableDataRef } from './composables/useSqlExec';

// 图表按需异步加载：echarts 体积大，静态引入会把它拽进 SQL 编辑器的首屏 chunk
const SqlResultChart = defineAsyncComponent(() => import('./SqlResultChart.vue'));

const props = defineProps<{
    execResTabs: ExecResTabState[];
    activeTab: number;
    dbId: number;
    db: string;
    tableDataHeight: string;
    tableDataEmptyText: string;
}>();

const emit = defineEmits<{
    tabRemove: [targetId: number];
    submitTicket: [sql: string, dt: ExecResTabLike];
    tabChange: [];
    submitUpdateFields: [dt: ExecResTabLike];
    cancelUpdateFields: [dt: ExecResTabLike];
    changeUpdatedField: [hasUpdatedFields: boolean, dt: ExecResTabLike];
    dataDelete: [deleteDatas: Record<string, unknown>[], dt: ExecResTabLike];
}>();

const onRemoveTab = (targetId: number) => {
    emit('tabRemove', targetId);
};

const onTabChange = () => {
    emit('tabChange');
};

// needApprovalOf 该结果里被触发策略要求审批的语句（与「已被禁止执行」互斥）
const needApprovalOf = (dt: ExecResTabState) => dt.needApprovalSqls ?? [];

// 正在导出的结果 tab id：导出是服务端重跑整条语句，可能比页面执行久得多，必须逐 tab 分别显态
const exportingTab = ref<number | null>(null);

const exportFull = async (dt: ExecResTabState) => {
    if (!dt.exportSql || exportingTab.value !== null) {
        return;
    }
    exportingTab.value = dt.id;
    try {
        await exportQueryResult({ dbId: props.dbId, db: props.db, sql: dt.exportSql, format: 'csv' });
    } finally {
        exportingTab.value = null;
    }
};

const onSubmitTicket = (dt: ExecResTabState) => {
    const sqls = needApprovalOf(dt);
    if (sqls.length) {
        // 多条需审批语句合并进一张工单（后端审批通过后拆句全部回放），不必逐条提单
        emit('submitTicket', sqls.join(';\n'), dt);
    }
};

const setDbTableRef = (dt: ExecResTabState, el: unknown) => {
    dt.dbTableRef = el as DbTableDataRef | null;
};

/** 每个结果页签各自的表格/图表选择，默认表格（不改变原有阅读路径） */
const views = reactive<Record<number, 'table' | 'chart'>>({});

/**
 * 图表取数列一律用 key、展示用 columnName。
 *
 * 结果集里存在「标题≠数据键」的合成列（执行全部的 SQL 列标题为 'SQL'、数据键为 'sql'），
 * 拿标题去取行数据会整列都是 undefined
 */
const chartColumns = (dt: ExecResTabState) => (dt.tableColumn ?? []).map((col) => ({ key: col.key ?? col.columnName, label: col.columnName }));
</script>
