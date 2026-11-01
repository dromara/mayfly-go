<template>
    <div class="db-table h-full flex flex-col gap-1">
        <el-row class="mb-1">
            <!-- 数据字典：导出结构事实（表/列/索引/注释），不含数据；有勾选表则只导这些表 -->
            <el-dropdown class="ml-1" @command="exportDataDict">
                <el-button type="primary" plain size="small">{{ $t('db.dataDict') }}</el-button>
                <template #dropdown>
                    <el-dropdown-menu>
                        <el-dropdown-item command="csv">CSV</el-dropdown-item>
                        <el-dropdown-item command="md">Markdown</el-dropdown-item>
                    </el-dropdown-menu>
                </template>
            </el-dropdown>

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
                    <!-- 被触发策略拦下时，用户可以改走审批：工单批准后凭工单下载同一次导出 -->
                    <el-button @click="applyExportTicket()" size="small">{{ $t('db.exportApplyTicket') }}</el-button>
                    <el-button @click="dump(db)" type="success" size="small">{{ $t('common.confirm') }}</el-button>
                </div>
            </el-popover>

            <!-- 库级只读诊断：与「数据字典」同属「看这个库的全貌」，放在文字按钮区而不是顶部图标行 -->
            <el-button size="small" type="primary" plain @click="emit('open-diagnostics')">{{ $t('db.sessionList') }}</el-button>
            <el-button size="small" @click="showErDiagram()">ER{{ $t('db.diagram') }}</el-button>
            <el-button type="primary" size="small" @click="openEditTable(false)">{{ $t('db.createTable') }}</el-button>
            <!-- 表级权限的配置入口放在表清单旁：管理员看到「有哪些表」的地方才是决定「保护哪张表」的地方 -->
            <el-button v-auth="'db:save'" size="small" @click="aclVisible = true">{{ $t('db.tableAcl.entry') }}</el-button>
            <el-button size="small" @click="sensitiveVisible = true">{{ $t('db.sensitive.entry') }}</el-button>
        </el-row>

        <DbTableAclDialog v-model:visible="aclVisible" :db-id="dbId" :db-name="db" />
        <DbSensitiveScanDialog v-model:visible="sensitiveVisible" :db-id="dbId" :db="db" />

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
            <el-table-column v-if="dialectCapabilities.mysqlCompatible" property="createTime" :label="$t('common.createTime')" min-width="150">
            </el-table-column>
            <el-table-column :label="$t('common.more')" min-width="160">
                <template #default="scope">
                    <el-link @click.prevent="showColumns(scope.row)" type="primary">{{ $t('db.column') }}</el-link>
                    <el-link class="ml-1" @click.prevent="showTableIndex(scope.row)" type="success">{{ $t('db.index') }}</el-link>
                    <el-link class="ml-1" v-if="dialectCapabilities.supportsTableEdit" @click.prevent="openEditTable(scope.row)" type="warning">
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

        <el-dialog width="40%" :title="`${chooseTableName} ${$t('db.column')}`" v-model="columnDialog.visible" append-to-body>
            <el-table border stripe :data="columnDialog.columns" size="small">
                <el-table-column prop="columnName" :label="$t('db.columnName')" show-overflow-tooltip> </el-table-column>
                <el-table-column width="120" prop="columnType" :label="$t('common.type')" show-overflow-tooltip> </el-table-column>
                <el-table-column width="80" prop="nullable" :label="$t('db.nullable')" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnComment" :label="$t('db.comment')" show-overflow-tooltip> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="40%" :title="`${chooseTableName} ${$t('db.index')}`" v-model="indexDialog.visible" append-to-body>
            <el-table border stripe :data="indexDialog.indexes" size="small">
                <el-table-column prop="indexName" :label="$t('common.name')" min-width="120" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnName" :label="$t('db.columnName')" min-width="120" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="seqInIndex" :label="$t('db.seqInIndex')" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="indexType" :label="$t('common.type')"> </el-table-column>
                <el-table-column prop="indexComment" :label="$t('db.comment')" min-width="130" show-overflow-tooltip> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="55%" :title="`'${chooseTableName}' DDL`" v-model="ddlDialog.visible" append-to-body>
            <monaco-editor height="400px" language="sql" v-model="ddlDialog.ddl" :options="{ readOnly: true }" />
        </el-dialog>

        <!-- 画布要拖拽平移：鼠标按下后很可能拖出弹窗范围、在遮罩上松开，
             默认的点击遮罩关闭会把整个图关掉，所以这里必须关掉它 -->
        <el-dialog
            :title="$t('db.erDiagram')"
            v-model="erDialog.visible"
            width="90%"
            top="5vh"
            destroy-on-close
            append-to-body
            :close-on-click-modal="false"
            :close-on-press-escape="false"
        >
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
import { computed, defineAsyncComponent, nextTick, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import { dbApi } from '@/views/ops/db/api';
import { openTicket } from '@/views/flow/ticketService';
import SqlExecBox from '../sql-editor/SqlExecBox';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { isTrue } from '@/common/assert';
import { getDbDialect, getDialectCapabilities } from '../dialect/index';
import { DbInst } from '../db';
import { declaredColumnArgs } from '../core/columnDefinition';
import type { DbTableInfo, ColumnMetadata, DbTableIndex, TableOpData } from '../types';
import { formatSql } from '../sql-editor/utils/formatSql';
import { fuzzyMatchField } from '@/common/utils/string';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import ErDiagram from './ErDiagram.vue';
import type { TableDefinition, TableColumnDefinition, TableIndexDefinition, IndexColumnDefinition } from '../types/schema';

const DbTableOp = defineAsyncComponent(() => import('./DbTableOp.vue'));
const DbTableAclDialog = defineAsyncComponent(() => import('./DbTableAclDialog.vue'));
const DbSensitiveScanDialog = defineAsyncComponent(() => import('./DbSensitiveScanDialog.vue'));
// DDL 查看是低频动作，编辑器主体（约 3.9M）随弹窗首次打开才加载
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

// 库级诊断入口：由宿主（DbDataOp）打开诊断页签，本组件只负责触发
const emit = defineEmits<{ (e: 'open-diagnostics'): void }>();

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

// 表访问控制弹窗的开关状态：低频动作，弹窗内才加载角色与规则
const aclVisible = ref(false);
// 敏感列扫描同样低频，弹窗打开才跑（一次扫描要逐表取列元数据）
const sensitiveVisible = ref(false);

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
        indexes: [] as Record<string, unknown>[],
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
            indexes: [],
            columns: [],
        } as TableOpData,
    },
    filterDb: {
        param: '',
        cache: [],
        list: [],
    },
});

const { loading, tableNameSearch, tableCommentSearch, dumpInfo, chooseTableName, columnDialog, indexDialog, ddlDialog, erDialog, tableCreateDialog } =
    toRefs(state);

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
 * 申请导出工单
 *
 * 与即席导出的区别：策略型拦截（需审批/禁止）可以走这条通道；表/列权限命中不行——
 * 那是硬边界，审批不能替代「这个人本来就不该看到这张表」。因此这里不做前置判定，
 * 由后端在下载时再判一次权限（工单只能解决「要不要人批」）。
 */
const applyExportTicket = () => {
    isTrue(state.dumpInfo.tables.length > 0, 'db.selectExportTable');
    openTicket({
        bizType: 'db_export_flow',
        bizForm: {
            dbId: props.dbId,
            dbName: props.db,
            tables: state.dumpInfo.tables.map((x: DbTableInfo) => x.tableName),
            dumpType: String(state.dumpInfo.type),
            reason: '',
        },
        remark: `${props.db} 导出申请`,
    });
    state.dumpInfo.visible = false;
};

/**
 * 数据库信息导出
 */
/** 数据字典导出：勾选了表就只导这些表，否则整库 */
const exportDataDict = (format: string) => {
    const tableNames = state.dumpInfo.tables.map((x: DbTableInfo) => x.tableName);
    const tablesParam = tableNames.length > 0 ? `&tables=${tableNames.join(',')}` : '';
    const a = document.createElement('a');
    a.setAttribute(
        'href',
        `${config.baseApiUrl}/dbs/${props.dbId}/data-dict?db=${props.db}&format=${format}${tablesParam}&${joinClientParams()}`
    );
    a.click();
};

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
    state.indexDialog.indexes = await dbApi.tableIndex.request({
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

            const colDefs: TableColumnDefinition[] = columns.map((c: ColumnMetadata) => {
                const { length, numScale } = declaredColumnArgs(c);
                return {
                    name: c.columnName,
                    type: c.dataType || 'varchar',
                    length: String(length ?? ''),
                    numScale: String(numScale ?? ''),
                    nullable: c.nullable ?? false,
                    isPrimaryKey: c.isPrimaryKey || false,
                    autoIncrement: c.autoIncrement || false,
                    value: c.columnDefault ? String(c.columnDefault).replace(/^'|'$/g, '') : '',
                    comment: c.columnComment || '',
                };
            });

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
    if (!(await useI18nDeleteConfirm(tableName))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
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
//
// 顺序不可颠倒：**先把 data 备好再置 visible**。子组件以 visible 的 false→true 作为表单初始化时机，
// 若先开抽屉再异步取列元数据，watch 会带着旧 data 跑一遍（实测：创建表成功后自动进入编辑态时，
// 表单留着创建时的脏行/空白表单，oldFields 与显示内容都对不上）。
// 抽屉已开着（创建表成功后的自动进入编辑态）时还要先关一次，否则 visible 不变、watch 不会重跑。
const openEditTable = async (row: DbTableInfo | { tableName: string } | false) => {
    if (state.tableCreateDialog.visible) {
        state.tableCreateDialog.visible = false;
        await nextTick();
    }
    state.tableCreateDialog.activeName = '1';

    if (row === false) {
        state.tableCreateDialog.data = { edit: false, row: {}, indexes: [], columns: [] };
        state.tableCreateDialog.title = useI18nCreateTitle('db.table');
    }

    if (row && row.tableName) {
        state.tableCreateDialog.title = useI18nEditTitle('db.table');
        let indexes = await dbApi.tableIndex.request({
            id: props.dbId,
            db: props.db,
            tableName: row.tableName,
        });
        let columns = await dbApi.columnMetadata.request({
            id: props.dbId,
            db: props.db,
            tableName: row.tableName,
        });
        state.tableCreateDialog.data = { edit: true, row: { ...row }, indexes, columns };
    }

    state.tableCreateDialog.visible = true;
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
