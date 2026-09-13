<template>
    <div class="db-table h-full flex flex-col gap-1">
        <el-row class="mb-1">
            <el-popover v-model:visible="state.dumpInfo.visible" trigger="click" :width="470" placement="right">
                <template #reference>
                    <el-button v-auth="'db:data:export'" :disabled="state.dumpInfo.tables?.length == 0" class="ml-1" type="success" size="small">
                        {{ $t('db.dump') }}
                    </el-button>
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
                :sort-method="(a: DbTableInfo, b: DbTableInfo) => parseInt(String(a.tableRows)) - parseInt(String(b.tableRows))"
            ></el-table-column>
            <el-table-column
                property="dataLength"
                :label="$t('db.dataSize')"
                sortable
                :sort-method="(a: DbTableInfo, b: DbTableInfo) => parseInt(String(a.dataLength)) - parseInt(String(b.dataLength))"
            >
                <template #default="scope">
                    {{ formatByteSize(scope.row.dataLength) }}
                </template>
            </el-table-column>
            <el-table-column
                property="indexLength"
                :label="$t('db.indexSize')"
                sortable
                :sort-method="(a: DbTableInfo, b: DbTableInfo) => parseInt(String(a.indexLength)) - parseInt(String(b.indexLength))"
            >
                <template #default="scope">
                    {{ formatByteSize(scope.row.indexLength) }}
                </template>
            </el-table-column>
            <el-table-column v-if="dialectCapabilities.mysqlCompatible" property="createTime" :label="$t('common.createTime')" min-width="150"> </el-table-column>
            <el-table-column :label="$t('common.more')" min-width="160">
                <template #default="scope">
                    <el-link @click.prevent="showColumns(scope.row)" type="primary">{{ $t('db.column') }}</el-link>
                    <el-link class="ml-1" @click.prevent="showTableIndex(scope.row)" type="success">{{ $t('db.index') }}</el-link>
                    <el-link class="ml-1" v-if="dialectCapabilities.supportsTableEdit" @click.prevent="openEditTable(scope.row)" type="warning">
                        {{ $t('db.editTable') }}
                    </el-link>
                    <el-link class="ml-1" @click.prevent="showCreateDdl(scope.row)" type="info">DDL</el-link>
                    <el-link class="ml-1" @click.prevent="showErDiagram()" type="primary">ER{{ $t('db.diagram') }}</el-link>
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

        <el-dialog :title="$t('db.erDiagram')" v-model="erDialog.visible" width="90%" top="5vh" destroy-on-close>
            <div v-loading="erDialog.loading" style="height: 70vh">
                <ErDiagram v-if="erDialog.tables.length > 0" :tables="erDialog.tables" @table-click="onErTableClick" />
                <el-empty v-else-if="!erDialog.loading" :description="$t('db.erDiagramEmpty')" />
            </div>
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
import SqlExecBox from '../sql-editor/SqlExecBox';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { isTrue } from '@/common/assert';
import { getDbDialect, getDialectCapabilities } from '../dialect/index';
import { DbInst } from '../db';
import type { DbTableInfo, ColumnMetadata, DbTableIndex, TableOpData } from '../types';
import { formatSql } from '../sql-editor/utils/formatSql';
import { fuzzyMatchField } from '@/common/utils/string';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import ErDiagram from './ErDiagram.vue';
import type { TableDefinition, ColumnDefinition, TableIndexDefinition, IndexColumnDefinition } from '../types/schema';

const DbTableOp = defineAsyncComponent(() => import('./DbTableOp.vue'));
// DDL 查看是低频动作，编辑器主体（约 3.9M）随弹窗首次打开才加载
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

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
    tables: [] as DbTableInfo[],
    tableNameSearch: '',
    tableCommentSearch: '',
    dumpInfo: {
        visible: false,
        id: 0,
        db: '',
        type: 3,
        tables: [] as DbTableInfo[],
    },
    chooseTableName: '',
    columnDialog: {
        visible: false,
        columns: [] as ColumnMetadata[],
    },
    indexDialog: {
        visible: false,
        indexs: [] as Record<string, unknown>[],
    },
    ddlDialog: {
        visible: false,
        ddl: '',
    },
    erDialog: {
        visible: false,
        loading: false,
        tables: [] as TableDefinition[],
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
        } as TableOpData,
    },
    filterDb: {
        param: '',
        cache: [],
        list: [],
    },
});

const { loading, tableNameSearch, tableCommentSearch, dumpInfo, chooseTableName, columnDialog, indexDialog, ddlDialog, erDialog, tableCreateDialog } = toRefs(state);

/** 方言能力：MySQL 专属列与表编辑入口的唯一判据，新增方言无需改动本文件 */
const dialectCapabilities = computed(() => getDialectCapabilities(getDbDialect(props.dbType)));

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
        return fuzzyMatchField(tableNameSearch, tables, (table: DbTableInfo) => table.tableName);
    }
    return fuzzyMatchField(tableCommentSearch, tables, (table: DbTableInfo) => table.tableComment);
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
const handleDumpTableSelectionChange = (vals: DbTableInfo[]) => {
    state.dumpInfo.tables = vals;
};

/**
 * 数据库信息导出
 */
const dump = (db: string) => {
    isTrue(state.dumpInfo.tables.length > 0, 'db.selectExportTable');
    const tableNames = state.dumpInfo.tables.map((x: DbTableInfo) => x.tableName);
    const a = document.createElement('a');
    a.setAttribute(
        'href',
        `${config.baseApiUrl}/dbs/${props.dbId}/dump?db=${db}&type=${state.dumpInfo.type}&tables=${tableNames.join(',')}&${joinClientParams()}`
    );
    a.click();
    state.dumpInfo.visible = false;
};

const showColumns = async (row: DbTableInfo) => {
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

const showTableIndex = async (row: DbTableInfo) => {
    state.chooseTableName = row.tableName;
    state.indexDialog.indexs = await dbApi.tableIndex.request({
        id: props.dbId,
        db: props.db,
        tableName: row.tableName,
    });

    state.indexDialog.visible = true;
};

const showCreateDdl = async (row: DbTableInfo) => {
    state.chooseTableName = row.tableName;
    const res = await dbApi.tableDdl.request({
        id: props.dbId,
        db: props.db,
        tableName: row.tableName,
    });

    state.ddlDialog.ddl = await formatSql(res, getDbDialect(props.dbType).getInfo().formatSqlDialect);
    state.ddlDialog.visible = true;
};

/** 打开 ER 图对话框，加载所有表的元数据 */
const showErDiagram = async () => {
    state.erDialog.visible = true;
    state.erDialog.loading = true;
    state.erDialog.tables = [];

    try {
        const tables: TableDefinition[] = [];
        for (const tableInfo of state.tables) {
            const [columns, indexes] = await Promise.all([
                dbApi.columnMetadata.request({ id: props.dbId, db: props.db, tableName: tableInfo.tableName }),
                dbApi.tableIndex.request({ id: props.dbId, db: props.db, tableName: tableInfo.tableName }).catch(() => []),
            ]);

            const colDefs: ColumnDefinition[] = columns.map((c: ColumnMetadata) => ({
                name: c.columnName,
                type: c.dataType || 'varchar',
                length: c.showLength ? String(c.showLength) : '',
                numScale: c.showScale ? String(c.showScale) : '',
                notNull: !c.nullable,
                pri: c.isPrimaryKey || false,
                auto_increment: c.autoIncrement || false,
                value: c.columnDefault ? String(c.columnDefault).replace(/^'|'$/g, '') : '',
                remark: c.columnComment || '',
            }));

            // 后端 t-index 下发的是扁平结构（DbTableIndex），此处转为表编辑子系统的领域模型
            const idxDefs: TableIndexDefinition[] = (indexes as DbTableIndex[])
                .filter((idx) => idx.indexName !== 'PRIMARY')
                .map((idx) => ({
                    name: idx.indexName,
                    columns: (idx.columnName || '')
                        .split(',')
                        .filter(Boolean)
                        .map((n): IndexColumnDefinition => ({ name: n.trim() })),
                    unique: !!idx.isUnique,
                    type: (idx.indexType || 'BTREE') as TableIndexDefinition['type'],
                }));

            tables.push({
                name: tableInfo.tableName,
                comment: tableInfo.tableComment || '',
                columns: colDefs,
                indexes: idxDefs,
                constraints: [],
            });
        }
        state.erDialog.tables = tables;
    } catch (e) {
        //
    } finally {
        state.erDialog.loading = false;
    }
};

/** ER 图点击表名，打开编辑表对话框 */
const onErTableClick = async (tableName: string) => {
    state.erDialog.visible = false;
    const row = state.tables.find((t) => t.tableName === tableName);
    if (row) {
        await openEditTable(row);
    }
};

/**
 * 删除表
 */
const dropTable = async (row: DbTableInfo) => {
    const tableName = row.tableName;
    try {
        await useI18nDeleteConfirm(tableName);
    } catch {
        return; // 用户取消
    }

    const dialect = getDbDialect(props.dbType);
    SqlExecBox({
        // 删表 DDL 由方言生成：schema 型方言须带 schema 限定，否则会删到默认 schema 下的同名表
        sql: dialect.getDropTableSql(props.db as string, tableName),
        dbId: props.dbId as number,
        db: props.db as string,
        formatDialect: dialect.getInfo().formatSqlDialect,
        runSuccessCallback: async () => {
            await getTables();
        },
    });
};

// 打开编辑表
const openEditTable = async (row: DbTableInfo | { tableName: string } | false) => {
    state.tableCreateDialog.visible = true;
    state.tableCreateDialog.activeName = '1';

    if (row === false) {
        state.tableCreateDialog.data = { edit: false, row: {}, indexs: [], columns: [] };
        state.tableCreateDialog.title = useI18nCreateTitle('db.table');
    }

    if (row && row.tableName) {
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
        state.tableCreateDialog.data = { edit: true, row: { ...row }, indexs, columns };
    }
};

const onSubmitSql = async (row: { tableName: string }) => {
    await openEditTable(row);
    await getTables();
};
</script>
<style lang="scss">
.db-table {
    > .el-table {
        flex: 1;
        min-height: 0;
    }
}
</style>
