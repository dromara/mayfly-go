<template>
    <div class="db-table">
        <el-row class="mb-1">
            <el-popover v-model:visible="state.dumpInfo.visible" trigger="click" :width="470" placement="right">
                <template #reference>
                    <el-button :disabled="state.dumpInfo.tables?.length == 0" class="ml-1" type="success" size="small">{{ $t('db.dump') }}</el-button>
                </template>
                <el-form-item :label="$t('db.exportContent')">
                    <el-radio-group v-model="dumpInfo.type">
                        <el-radio :value="1" size="small">{{ $t('db.structure') }}</el-radio>
                        <el-radio :value="2" size="small">{{ $t('db.data') }}</el-radio>
                        <el-radio :value="3" size="small">{{ $t('db.structure') }} ＋ {{ $t('db.data') }}</el-radio>
                    </el-radio-group>
                </el-form-item>

                <el-form-item>
                    <el-table :data="state.dumpInfo.tables" :empty-text="$t('db.selectExportTable')" max-height="300" size="small">
                        <el-table-column property="tableName" :label="$t('db.table')" min-width="150" show-overflow-tooltip> </el-table-column>
                        <el-table-column property="tableComment" :label="$t('db.comment')" min-width="150" show-overflow-tooltip> </el-table-column>
                    </el-table>
                </el-form-item>

                <div style="text-align: right">
                    <el-button @click="state.dumpInfo.visible = false" size="small">{{ $t('common.cancel') }}</el-button>
                    <el-button @click="dump(db)" type="success" size="small">{{ $t('common.confirm') }}</el-button>
                </div>
            </el-popover>

            <el-button type="primary" size="small" @click="openEditTable(false)">{{ $t('db.createTable') }}</el-button>
        </el-row>

        <el-table v-loading="loading" @selection-change="handleDumpTableSelectionChange" border stripe :data="filterTableInfos" size="small" :height="height">
            <el-table-column type="selection" width="30" />

            <el-table-column property="tableName" :label="$t('db.table')" min-width="150" show-overflow-tooltip>
                <template #header>
                    <el-input v-model="tableNameSearch" size="small" :placeholder="$t('db.tableNamePlaceholder')" clearable />
                </template>
            </el-table-column>
            <el-table-column property="tableComment" :label="$t('db.comment')" min-width="150" show-overflow-tooltip>
                <template #header>
                    <el-input v-model="tableCommentSearch" size="small" :placeholder="$t('db.commentPlaceholder')" clearable />
                </template>
            </el-table-column>
            <el-table-column
                prop="tableRows"
                label="Rows"
                min-width="70"
                sortable
                :sort-method="(a: any, b: any) => parseInt(a.tableRows) - parseInt(b.tableRows)"
            ></el-table-column>
            <el-table-column
                property="dataLength"
                :label="$t('db.dataSize')"
                sortable
                :sort-method="(a: any, b: any) => parseInt(a.dataLength) - parseInt(b.dataLength)"
            >
                <template #default="scope">
                    {{ formatByteSize(scope.row.dataLength) }}
                </template>
            </el-table-column>
            <el-table-column
                property="indexLength"
                :label="$t('db.indexSize')"
                sortable
                :sort-method="(a: any, b: any) => parseInt(a.indexLength) - parseInt(b.indexLength)"
            >
                <template #default="scope">
                    {{ formatByteSize(scope.row.indexLength) }}
                </template>
            </el-table-column>
            <el-table-column v-if="compatibleMysql(dbType)" property="createTime" :label="$t('common.createTime')" min-width="150"> </el-table-column>
            <el-table-column :label="$t('common.more')" min-width="160">
                <template #default="scope">
                    <el-link @click.prevent="showColumns(scope.row)" type="primary">{{ $t('db.column') }}</el-link>
                    <el-link class="ml-1" @click.prevent="showTableIndex(scope.row)" type="success">{{ $t('db.index') }}</el-link>
                    <el-link class="ml-1" v-if="editDbTypes.indexOf(dbType) > -1" @click.prevent="openEditTable(scope.row)" type="warning">
                        {{ $t('db.editTable') }}
                    </el-link>
                    <el-link class="ml-1" @click.prevent="showCreateDdl(scope.row)" type="info">DDL</el-link>
                </template>
            </el-table-column>
            <el-table-column :label="$t('common.operation')" min-width="80">
                <template #default="scope">
                    <el-link @click.prevent="dropTable(scope.row)" type="danger">{{ $t('common.delete') }}</el-link>
                </template>
            </el-table-column>
        </el-table>

        <el-dialog width="40%" :title="`${chooseTableName} ${$t('db.column')}`" v-model="columnDialog.visible">
            <el-table border stripe :data="columnDialog.columns" size="small">
                <el-table-column prop="columnName" :label="$t('db.columnName')" show-overflow-tooltip> </el-table-column>
                <el-table-column width="120" prop="columnType" :label="$t('common.type')" show-overflow-tooltip> </el-table-column>
                <el-table-column width="80" prop="nullable" :label="$t('db.nullable')" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnComment" :label="$t('db.comment')" show-overflow-tooltip> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="40%" :title="`${chooseTableName} ${$t('db.index')}`" v-model="indexDialog.visible">
            <el-table border stripe :data="indexDialog.indexs" size="small">
                <el-table-column prop="indexName" :label="$t('common.name')" min-width="120" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnName" :label="$t('db.columnName')" min-width="120" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="seqInIndex" :label="$t('db.seqInIndex')" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="indexType" :label="$t('common.type')"> </el-table-column>
                <el-table-column prop="indexComment" :label="$t('db.comment')" min-width="130" show-overflow-tooltip> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="55%" :title="`'${chooseTableName}' DDL`" v-model="ddlDialog.visible">
            <monaco-editor height="400px" language="sql" v-model="ddlDialog.ddl" :options="{ readOnly: true }" />
        </el-dialog>

        <db-table-op
            :title="tableCreateDialog.title"
            :active-name="tableCreateDialog.activeName"
            :dbId="dbId"
            :db="db"
            :dbType="dbType"
            :data="tableCreateDialog.data"
            v-model:visible="tableCreateDialog.visible"
            @submit-sql="onSubmitSql"
        >
        </db-table-op>
    </div>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent, onMounted, reactive, toRefs, watch } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import { dbApi } from '@/views/ops/db/api';
import SqlExecBox from '../sqleditor/SqlExecBox';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { isTrue } from '@/common/assert';
import { compatibleMysql, editDbTypes, getDbDialect } from '../../dialect/index';
import { DbInst } from '../../db';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { format as sqlFormatter } from 'sql-formatter';
import { fuzzyMatchField } from '@/common/utils/string';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';

const DbTableOp = defineAsyncComponent(() => import('./DbTableOp.vue'));

const props = defineProps({
    height: {
        type: [String],
        default: '65vh',
    },
    dbId: {
        type: [Number],
        required: true,
    },
    db: {
        type: [String],
        required: true,
    },
    dbType: {
        type: [String],
        required: true,
    },
});

const state = reactive({
    row: {},
    loading: false,
    tables: [],
    tableNameSearch: '',
    tableCommentSearch: '',
    dumpInfo: {
        visible: false,
        id: 0,
        db: '',
        type: 3,
        tables: [],
    },
    chooseTableName: '',
    columnDialog: {
        visible: false,
        columns: [],
    },
    indexDialog: {
        visible: false,
        indexs: [],
    },
    ddlDialog: {
        visible: false,
        ddl: '',
    },
    tableCreateDialog: {
        title: '',
        visible: false,
        activeName: '1',
        type: '',
        data: {
            // 修改表时，传递修改数据
            edit: false,
            row: {},
            indexs: [],
            columns: [],
        },
    },
    filterDb: {
        param: '',
        cache: [],
        list: [],
    },
});

const { loading, tableNameSearch, tableCommentSearch, dumpInfo, chooseTableName, columnDialog, indexDialog, ddlDialog, tableCreateDialog } = toRefs(state);

onMounted(async () => {
    getTables();
});

watch(props, async () => {
    await getTables();
});

const filterTableInfos = computed(() => {
    const tables = state.tables;
    const tableNameSearch = state.tableNameSearch;
    const tableCommentSearch = state.tableCommentSearch;
    if (!tableNameSearch && !tableCommentSearch) {
        return tables;
    }

    if (tableNameSearch) {
        return fuzzyMatchField(tableNameSearch, tables, (table: any) => table.tableName);
    }
    return fuzzyMatchField(tableCommentSearch, tables, (table: any) => table.tableComment);
});

const getTables = async () => {
    state.loading = true;
    try {
        state.tables = [];
        state.tables = await dbApi.tableInfos.request({ id: props.dbId, db: props.db });
    } catch (e) {
        //
    } finally {
        state.loading = false;
    }
};

/**
 * 选择导出数据库表
 */
const handleDumpTableSelectionChange = (vals: any) => {
    state.dumpInfo.tables = vals;
};

/**
 * 数据库信息导出
 */
const dump = (db: string) => {
    isTrue(state.dumpInfo.tables.length > 0, 'db.selectExportTable');
    const tableNames = state.dumpInfo.tables.map((x: any) => x.tableName);
    const a = document.createElement('a');
    a.setAttribute(
        'href',
        `${config.baseApiUrl}/dbs/${props.dbId}/dump?db=${db}&type=${state.dumpInfo.type}&tables=${tableNames.join(',')}&${joinClientParams()}`
    );
    a.click();
    state.dumpInfo.visible = false;
};

const showColumns = async (row: any) => {
    state.chooseTableName = row.tableName;
    const columns = await dbApi.columnMetadata.request({
        id: props.dbId,
        db: props.db,
        tableName: row.tableName,
    });
    DbInst.initColumns(columns);
    state.columnDialog.columns = columns;

    state.columnDialog.visible = true;
};

const showTableIndex = async (row: any) => {
    state.chooseTableName = row.tableName;
    state.indexDialog.indexs = await dbApi.tableIndex.request({
        id: props.dbId,
        db: props.db,
        tableName: row.tableName,
    });

    state.indexDialog.visible = true;
};

const showCreateDdl = async (row: any) => {
    state.chooseTableName = row.tableName;
    const res = await dbApi.tableDdl.request({
        id: props.dbId,
        db: props.db,
        tableName: row.tableName,
    });

    state.ddlDialog.ddl = sqlFormatter(res, { language: getDbDialect(props.dbType).getInfo().formatSqlDialect as any });
    state.ddlDialog.visible = true;
};

/**
 * 删除表
 */
const dropTable = async (row: any) => {
    try {
        const tableName = row.tableName;
        await useI18nDeleteConfirm(tableName);
        SqlExecBox({
            sql: `DROP TABLE ${tableName}`,
            dbId: props.dbId as any,
            db: props.db as any,
            runSuccessCallback: async () => {
                await getTables();
            },
        });
    } catch (err) {
        //
    }
};

// 打开编辑表
const openEditTable = async (row: any) => {
    state.tableCreateDialog.visible = true;
    state.tableCreateDialog.activeName = '1';

    if (!row === false) {
        state.tableCreateDialog.data = { edit: false, row: {}, indexs: [], columns: [] };
        state.tableCreateDialog.title = useI18nCreateTitle('db.table');
    }

    if (row.tableName) {
        state.tableCreateDialog.title = useI18nEditTitle('db.table');
        let indexs = await dbApi.tableIndex.request({
            id: props.dbId,
            db: props.db,
            tableName: row.tableName,
        });
        let columns = await dbApi.columnMetadata.request({
            id: props.dbId,
            db: props.db,
            tableName: row.tableName,
        });
        state.tableCreateDialog.data = { edit: true, row, indexs, columns };
    }
};

const onSubmitSql = async (row: { tableName: string }) => {
    await openEditTable(row);
    await getTables();
};
</script>
<style lang="scss"></style>
