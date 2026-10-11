<template>
    <div class="db-sql-exec-log h-full">
        <page-table
            ref="pageTableRef"
            :page-api="dbApi.getSqlExecs"
            :lazy="true"
            height="100%"
            :search-items="searchItems"
            v-model:query-form="query"
            :columns="columns"
        >
            <template #dbSelect>
                <el-select v-model="query.db" :placeholder="$t('db.selectDbPlaceholder')" filterable clearable>
                    <el-option v-for="item in dbs" :key="item" :label="`${item}`" :value="item"> </el-option>
                </el-select>
            </template>

            <template #action="{ data }">
                <el-link
                    v-if="canRestoreRollback(data)"
                    type="primary"
                    plain
                    size="small"
                    underline="never"
                    @click="onShowRollbackSql(data)"
                >
                    {{ $t('db.restoreSql') }}</el-link
                >
            </template>
        </page-table>

        <el-dialog width="55%" :title="$t('db.restoreSql')" v-model="rollbackDialog.visible">
            <!-- 行数据采集被截断、未做并发校验等说明由服务端给出：不展示就等于让用户以为「全都能改回去」 -->
            <el-alert v-if="rollbackDialog.note" :title="rollbackDialog.note" type="warning" :closable="false" show-icon class="mb-2" />
            <!-- 未生成回滚的行逐行带原因列出：只给条数用户没法对账 -->
            <el-alert v-if="rollbackDialog.conflicts.length" type="warning" :closable="false" show-icon class="mb-2">
                <template #title>{{ $t('db.rollbackConflicts', { count: rollbackDialog.conflicts.length }) }}</template>
                <ul class="m-0 pl-4">
                    <li v-for="conflict in rollbackDialog.conflicts" :key="`${conflict.key}|${conflict.reason}`" class="text-[12px] leading-5">
                        <span class="font-mono">{{ conflict.key }}</span>
                        <span> · {{ rollbackConflictReasonText(conflict.reason) }}</span>
                    </li>
                </ul>
            </el-alert>
            <div class="mb-2 text-[12px] text-gray-500">{{ $t('db.rollbackGenerated', { rows: rollbackDialog.rows }) }}</div>
            <el-input type="textarea" :autosize="{ minRows: 15, maxRows: 30 }" :model-value="rollbackDialog.sql" readonly size="small"></el-input>
            <div class="mt-2 text-[12px] text-gray-400">{{ $t('db.rollbackTextOnly') }}</div>
            <template #footer>
                <el-button :disabled="!rollbackDialog.sql" @click="copyRollbackSql">{{ $t('common.copy') }}</el-button>
                <el-button type="primary" @click="rollbackDialog.visible = false">{{ $t('common.close') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, toRefs, useTemplateRef, watch, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import { dbApi } from '../api';
import { DbSqlExecTypeEnum, DbSqlExecStatusEnum } from '../enums';
import PageTable from '@/components/page-table/PageTable.vue';
import { TableColumn } from '@/components/page-table';
import { SearchItem } from '@/components/page-table/SearchForm';
import { formatDate } from '@/common/utils/format';
import { copyToClipboard } from '@/common/utils/string';
import type { DbSqlExec } from '../types';
import { useRollbackSqlDialog } from './useRollbackSql';

const props = defineProps({
    dbId: {
        type: [Number],
        required: true,
    },
    dbs: {
        type: Array as PropType<string[]>,
        required: true,
    },
});

const searchItems = [
    SearchItem.slot('db', 'db.db', 'dbSelect'),
    SearchItem.input('table', 'db.table'),
    SearchItem.select('type', 'db.stmtType').withEnum(DbSqlExecTypeEnum),
    // 状态必须能单筛：默认不带状态时列表含「待执行/不执行」的记录，而排查失败时只想要失败
    SearchItem.select('status', 'common.status').withEnum(DbSqlExecStatusEnum),
    SearchItem.input('keyword', 'db.sqlKeyword'),
    SearchItem.datePicker('execTimeRange', 'db.execTime')
        .withSpan(2)
        .withOneProps('type', 'datetimerange')
        .withOneProps('format', 'YYYY-MM-DD HH:mm:ss')
        .withOneProps('value-format', 'YYYY-MM-DD HH:mm:ss')
        .bindEvent('change', (value: string | [string, string] | null) => {
            if (!value) {
                state.query.startTime = '';
                state.query.endTime = '';
                return;
            }
            state.query.startTime = formatDate(value[0]);
            state.query.endTime = formatDate(value[1]);
        }),
];

const columns = ref([
    TableColumn.new('db', 'db.db'),
    TableColumn.new('table', 'db.table'),
    TableColumn.new('type', 'db.stmtType').typeTag(DbSqlExecTypeEnum),
    TableColumn.new('creator', 'db.execUser'),
    TableColumn.new('sql', 'SQL').canBeautify(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('status', 'common.status').typeTag(DbSqlExecStatusEnum),
    TableColumn.new('res', 'db.execRes'),
    TableColumn.new('createTime', 'db.execTime').isTime(),
    TableColumn.new('oldValue', 'db.oldValue').canBeautify(),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(90).fixedRight().alignCenter(),
]);

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const state = reactive({
    dbs: [],
    query: {
        dbId: 0,
        db: '',
        table: '',
        status: '' as number | '',
        // 默认不筛状态：写死「成功+失败」会让「待执行/不执行」的记录在审计列表里永远看不见
        type: null,
        keyword: '',
        startTime: '',
        endTime: '',
        pageNum: 1,
        pageSize: 10,
    },
    filterDb: {
        param: '',
        cache: [],
        list: [],
    },
});

// 回滚弹窗的取数与状态单点收敛至 composable（可单独测到「拒绝时不开空弹窗」）
const { dialog: rollbackDialog, show: showRollbackSql } = useRollbackSqlDialog();

const { t } = useI18n();

/** 可生成回滚的记录：成功执行过的 UPDATE/DELETE/INSERT（各类带回滚所需的行数据） */
const restorableTypes: number[] = [DbSqlExecTypeEnum.Update.value, DbSqlExecTypeEnum.Delete.value, DbSqlExecTypeEnum.Insert.value];
const canRestoreRollback = (data: DbSqlExec) => data.status == DbSqlExecStatusEnum.Success.value && restorableTypes.includes(data.type);

// 冲突原因枚举 → 文案 key；后端新增枚举而前端未跟上时原样展示枚举值，而不是吞掉
const rollbackConflictReasonKeys: Record<string, string> = {
    'concurrent-modified': 'db.rollbackConflictConcurrentModified',
    'row-missing': 'db.rollbackConflictRowMissing',
    'unknown-position': 'db.rollbackConflictUnknownPosition',
    'ambiguous-match': 'db.rollbackConflictAmbiguousMatch',
};
const rollbackConflictReasonText = (reason: string) => {
    const key = rollbackConflictReasonKeys[reason];
    return key ? t(key) : reason;
};

const { query } = toRefs(state);

onMounted(async () => {
    state.query.dbId = props.dbId;
    state.query.pageNum = 1;
    await searchSqlExecLog();
});

watch(props, async () => {
    state.query.dbId = props.dbId;
    state.query.pageNum = 1;
    await searchSqlExecLog();
});

const searchSqlExecLog = async () => {
    if (state.query.dbId) {
        pageTableRef.value?.search();
    }
};

const onShowRollbackSql = (sqlExecLog: DbSqlExec) => showRollbackSql(sqlExecLog.id);

// 成功/失败提示由 copyToClipboard 内部统一弹出，这里不再重复提示
const copyRollbackSql = () => {
    void copyToClipboard(rollbackDialog.sql);
};
</script>
<style lang="scss"></style>
