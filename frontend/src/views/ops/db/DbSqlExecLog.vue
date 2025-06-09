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
                    v-if="
                        data.oldValue != '' &&
                        data.status == DbSqlExecStatusEnum.Success.value &&
                        (data.type == DbSqlExecTypeEnum.Update.value || data.type == DbSqlExecTypeEnum.Delete.value)
                    "
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

        <el-dialog width="55%" :title="$t('db.restoreSql')" v-model="rollbackSqlDialog.visible">
            <el-input type="textarea" :autosize="{ minRows: 15, maxRows: 30 }" v-model="rollbackSqlDialog.sql" size="small"> </el-input>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, Ref, ref, toRefs, watch } from 'vue';
import { dbApi } from './api';
import { DbSqlExecTypeEnum, DbSqlExecStatusEnum } from './enums';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import { formatDate } from '@/common/utils/format';

const props = defineProps({
    dbId: {
        type: [Number],
        required: true,
    },
    dbs: {
        type: [Array<String>],
        required: true,
    },
});

const searchItems = [
    SearchItem.slot('db', 'db.db', 'dbSelect'),
    SearchItem.input('table', 'db.table'),
    SearchItem.select('type', 'db.stmtType').withEnum(DbSqlExecTypeEnum),
    SearchItem.input('keyword', 'common.keyword'),
    SearchItem.datePicker('execTimeRange', 'db.execTime')
        .withSpan(2)
        .withOneProps('type', 'datetimerange')
        .withOneProps('format', 'YYYY-MM-DD HH:mm:ss')
        .withOneProps('value-format', 'YYYY-MM-DD HH:mm:ss')
        .bindEvent('change', (value: any) => {
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
    TableColumn.new('type', 'db.stmtType').typeTag(DbSqlExecTypeEnum).setAddWidth(10),
    TableColumn.new('creator', 'db.execUser'),
    TableColumn.new('sql', 'SQL').canBeautify(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('status', 'common.status').typeTag(DbSqlExecStatusEnum),
    TableColumn.new('res', 'db.execRes'),
    TableColumn.new('createTime', 'db.execTime').isTime(),
    TableColumn.new('oldValue', 'db.oldValue').canBeautify(),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(90).fixedRight().alignCenter(),
]);

const pageTableRef: Ref<any> = ref(null);

const state = reactive({
    dbs: [],
    query: {
        dbId: 0,
        db: '',
        table: '',
        status: [DbSqlExecStatusEnum.Success.value, DbSqlExecStatusEnum.Fail.value].join(','),
        type: null,
        keyword: '',
        startTime: '',
        endTime: '',
        pageNum: 1,
        pageSize: 10,
    },
    rollbackSqlDialog: {
        visible: false,
        sql: '',
    },
    filterDb: {
        param: '',
        cache: [],
        list: [],
    },
});

const { query, rollbackSqlDialog } = toRefs(state);

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
        pageTableRef.value.search();
    }
};

const onShowRollbackSql = async (sqlExecLog: any) => {
    const columns = await dbApi.columnMetadata.request({ id: sqlExecLog.dbId, db: sqlExecLog.db, tableName: sqlExecLog.table });
    const primaryKey = getPrimaryKey(columns);
    const oldValue = JSON.parse(sqlExecLog.oldValue);

    let schema = '';
    let dbArr = sqlExecLog.db.split('/');
    if (dbArr.length == 2) {
        schema = dbArr[1] + '.';
    }

    const rollbackSqls = [];
    if (sqlExecLog.type == DbSqlExecTypeEnum.Update.value) {
        for (let ov of oldValue) {
            const setItems = [];
            for (let key in ov) {
                if (key == primaryKey) {
                    continue;
                }
                setItems.push(`${key} = ${wrapValue(ov[key])}`);
            }
            rollbackSqls.push(`UPDATE ${schema}${sqlExecLog.table} SET ${setItems.join(', ')} WHERE ${primaryKey} = ${wrapValue(ov[primaryKey])};`);
        }
    } else if (sqlExecLog.type == DbSqlExecTypeEnum.Delete.value) {
        const columnNames = columns.map((c: any) => c.columnName);
        for (let ov of oldValue) {
            const values = [];
            for (let column of columnNames) {
                values.push(wrapValue(ov[column]));
            }
            rollbackSqls.push(`INSERT INTO ${schema}${sqlExecLog.table} (${columnNames.join(', ')}) VALUES (${values.join(', ')});`);
        }
    }

    state.rollbackSqlDialog.sql = rollbackSqls.join('\n');
    state.rollbackSqlDialog.visible = true;
};

const getPrimaryKey = (columns: any) => {
    const col = columns.find((c: any) => c.isPrimaryKey);
    if (col) {
        return col.columnName;
    }
    return columns[0].columnName;
};

/**
 * 包装值，如果值类型为number则直接返回，其他则需要使用''包装
 */
const wrapValue = (val: any) => {
    if (typeof val == 'number') {
        return val;
    }
    return `'${val}'`;
};
</script>
<style lang="scss"></style>
