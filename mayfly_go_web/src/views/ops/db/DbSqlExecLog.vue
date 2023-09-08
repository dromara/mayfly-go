<template>
    <div class="db-sql-exec-log">
        <page-table
            height="100%"
            ref="sqlExecDialogPageTableRef"
            :query="queryConfig"
            v-model:query-form="query"
            :data="data"
            :columns="columns"
            :total="total"
            v-model:page-size="query.pageSize"
            v-model:page-num="query.pageNum"
            @pageChange="searchSqlExecLog()"
        >
            <template #dbSelect>
                <el-select v-model="query.db" placeholder="请选择数据库" style="width: 200px" filterable clearable>
                    <el-option v-for="item in dbs" :key="item" :label="`${item}`" :value="item"> </el-option>
                </el-select>
            </template>

            <template #action="{ data }">
                <el-link
                    v-if="data.type == DbSqlExecTypeEnum.Update.value || data.type == DbSqlExecTypeEnum.Delete.value"
                    type="primary"
                    plain
                    size="small"
                    :underline="false"
                    @click="onShowRollbackSql(data)"
                >
                    还原SQL</el-link
                >
            </template>
        </page-table>

        <el-dialog width="55%" :title="`还原SQL`" v-model="rollbackSqlDialog.visible">
            <el-input type="textarea" :autosize="{ minRows: 15, maxRows: 30 }" v-model="rollbackSqlDialog.sql" size="small"> </el-input>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs,watch, reactive, computed, onMounted, defineAsyncComponent } from 'vue';
import { dbApi } from './api';
import { DbSqlExecTypeEnum } from './enums';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn, TableQuery } from '@/components/pagetable';

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

const queryConfig = [
    TableQuery.slot('db', '数据库', 'dbSelect'),
    TableQuery.text('table', '表名'),
    TableQuery.select('type', '操作类型').setOptions(Object.values(DbSqlExecTypeEnum)),
];

const columns = [
    TableColumn.new('db', '数据库'),
    TableColumn.new('table', '表'),
    TableColumn.new('type', '类型').typeTag(DbSqlExecTypeEnum).setAddWidth(10),
    TableColumn.new('creator', '执行人'),
    TableColumn.new('sql', 'SQL').canBeautify(),
    TableColumn.new('oldValue', '原值').canBeautify(),
    TableColumn.new('createTime', '执行时间').isTime(),
    TableColumn.new('remark', '备注'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(90).fixedRight().alignCenter(),
];

const state = reactive({
    data: [],
    total: 0,
    dbs: [],
    query: {
        dbId: 0,
        db: '',
        table: '',
        type: null,
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

const { data, query, total, rollbackSqlDialog } = toRefs(state);

onMounted(async () => {
    searchSqlExecLog();
});

watch(props, async (newValue: any) => {
    await searchSqlExecLog();
});


const searchSqlExecLog = async () => {
    state.query.dbId = props.dbId
    const res = await dbApi.getSqlExecs.request(state.query);
    state.data = res.list;
    state.total = res.total;
};

const onShowRollbackSql = async (sqlExecLog: any) => {
    const columns = await dbApi.columnMetadata.request({ id: sqlExecLog.dbId, db: sqlExecLog.db, tableName: sqlExecLog.table });
    const primaryKey = getPrimaryKey(columns);
    const oldValue = JSON.parse(sqlExecLog.oldValue);

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
            rollbackSqls.push(`UPDATE ${sqlExecLog.table} SET ${setItems.join(', ')} WHERE ${primaryKey} = ${wrapValue(ov[primaryKey])};`);
        }
    } else if (sqlExecLog.type == DbSqlExecTypeEnum.Delete.value) {
        const columnNames = columns.map((c: any) => c.columnName);
        for (let ov of oldValue) {
            const values = [];
            for (let column of columnNames) {
                values.push(wrapValue(ov[column]));
            }
            rollbackSqls.push(`INSERT INTO ${sqlExecLog.table} (${columnNames.join(', ')}) VALUES (${values.join(', ')});`);
        }
    }

    state.rollbackSqlDialog.sql = rollbackSqls.join('\n');
    state.rollbackSqlDialog.visible = true;
};

const getPrimaryKey = (columns: any) => {
    const col = columns.find((c: any) => c.columnKey == 'PRI');
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
