<template>
    <div class="db-table">
        <el-row class="mb5">
            <el-popover v-model:visible="showDumpInfo" :width="470" placement="right" trigger="click">
                <template #reference>
                    <el-button class="ml5" type="success" size="small">导出</el-button>
                </template>
                <el-form-item label="导出内容: ">
                    <el-radio-group v-model="dumpInfo.type">
                        <el-radio :label="1" size="small">结构</el-radio>
                        <el-radio :label="2" size="small">数据</el-radio>
                        <el-radio :label="3" size="small">结构＋数据</el-radio>
                    </el-radio-group>
                </el-form-item>

                <el-form-item label="导出表: ">
                    <el-table @selection-change="handleDumpTableSelectionChange" max-height="300" size="small" :data="tables">
                        <el-table-column type="selection" width="45" />
                        <el-table-column property="tableName" label="表名" min-width="150" show-overflow-tooltip> </el-table-column>
                        <el-table-column property="tableComment" label="备注" min-width="150" show-overflow-tooltip> </el-table-column>
                    </el-table>
                </el-form-item>

                <div style="text-align: right">
                    <el-button @click="showDumpInfo = false" size="small">取消</el-button>
                    <el-button @click="dump(db)" type="success" size="small">确定</el-button>
                </div>
            </el-popover>

            <el-button type="primary" size="small" @click="openEditTable(false)">创建表</el-button>
        </el-row>

        <el-table v-loading="loading" border stripe :data="filterTableInfos" size="small" :height="height">
            <el-table-column property="tableName" label="表名" min-width="150" show-overflow-tooltip>
                <template #header>
                    <el-input v-model="tableNameSearch" size="small" placeholder="表名: 输入可过滤" clearable />
                </template>
            </el-table-column>
            <el-table-column property="tableComment" label="备注" min-width="150" show-overflow-tooltip>
                <template #header>
                    <el-input v-model="tableCommentSearch" size="small" placeholder="备注: 输入可过滤" clearable />
                </template>
            </el-table-column>
            <el-table-column
                prop="tableRows"
                label="Rows"
                min-width="70"
                sortable
                :sort-method="(a: any, b: any) => parseInt(a.tableRows) - parseInt(b.tableRows)"
            ></el-table-column>
            <el-table-column property="dataLength" label="数据大小" sortable :sort-method="(a: any, b: any) => parseInt(a.dataLength) - parseInt(b.dataLength)">
                <template #default="scope">
                    {{ formatByteSize(scope.row.dataLength) }}
                </template>
            </el-table-column>
            <el-table-column
                property="indexLength"
                label="索引大小"
                sortable
                :sort-method="(a: any, b: any) => parseInt(a.indexLength) - parseInt(b.indexLength)"
            >
                <template #default="scope">
                    {{ formatByteSize(scope.row.indexLength) }}
                </template>
            </el-table-column>
            <el-table-column v-if="compatibleMysql(dbType)" property="createTime" label="创建时间" min-width="150"> </el-table-column>
            <el-table-column label="更多信息" min-width="160">
                <template #default="scope">
                    <el-link @click.prevent="showColumns(scope.row)" type="primary">字段</el-link>
                    <el-link class="ml5" @click.prevent="showTableIndex(scope.row)" type="success">索引</el-link>
                    <el-link class="ml5" v-if="editDbTypes.indexOf(dbType) > -1" @click.prevent="openEditTable(scope.row)" type="warning">编辑表</el-link>
                    <el-link class="ml5" @click.prevent="showCreateDdl(scope.row)" type="info">DDL</el-link>
                </template>
            </el-table-column>
            <el-table-column label="操作" min-width="80">
                <template #default="scope">
                    <el-link @click.prevent="dropTable(scope.row)" type="danger">删除</el-link>
                </template>
            </el-table-column>
        </el-table>

        <el-dialog width="40%" :title="`${chooseTableName} 字段信息`" v-model="columnDialog.visible">
            <el-table border stripe :data="columnDialog.columns" size="small">
                <el-table-column prop="columnName" label="名称" show-overflow-tooltip> </el-table-column>
                <el-table-column width="120" prop="columnType" label="类型" show-overflow-tooltip> </el-table-column>
                <el-table-column width="80" prop="nullable" label="是否可为空" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnComment" label="备注" show-overflow-tooltip> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="40%" :title="`${chooseTableName} 索引信息`" v-model="indexDialog.visible">
            <el-table border stripe :data="indexDialog.indexs" size="small">
                <el-table-column prop="indexName" label="索引名" min-width="120" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnName" label="列名" min-width="120" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="seqInIndex" label="列序列号" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="indexType" label="类型"> </el-table-column>
                <el-table-column prop="indexComment" label="备注" min-width="130" show-overflow-tooltip> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="55%" :title="`${chooseTableName} Create-DDL`" v-model="ddlDialog.visible">
            <el-input disabled type="textarea" :autosize="{ minRows: 15, maxRows: 30 }" v-model="ddlDialog.ddl" size="small"> </el-input>
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
import { ElMessageBox } from 'element-plus';
import { formatByteSize } from '@/common/utils/format';
import { dbApi } from '@/views/ops/db/api';
import SqlExecBox from '../sqleditor/SqlExecBox';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { isTrue } from '@/common/assert';
import { compatibleMysql, DbType, editDbTypes } from '../../dialect/index';

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
    showDumpInfo: false,
    dumpInfo: {
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
        title: '创建表',
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

const {
    loading,
    tables,
    tableNameSearch,
    tableCommentSearch,
    showDumpInfo,
    dumpInfo,
    chooseTableName,
    columnDialog,
    indexDialog,
    ddlDialog,
    tableCreateDialog,
} = toRefs(state);

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
    return tables.filter((data: any) => {
        let tnMatch = true;
        let tcMatch = true;
        if (tableNameSearch) {
            tnMatch = data.tableName.toLowerCase().includes(tableNameSearch.toLowerCase());
        }
        if (tableCommentSearch) {
            tcMatch = data.tableComment.includes(tableCommentSearch);
        }
        return tnMatch && tcMatch;
    });
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
    state.dumpInfo.tables = vals.map((x: any) => x.tableName);
};

/**
 * 数据库信息导出
 */
const dump = (db: string) => {
    isTrue(state.dumpInfo.tables.length > 0, '请选择要导出的表');
    const a = document.createElement('a');
    a.setAttribute(
        'href',
        `${config.baseApiUrl}/dbs/${props.dbId}/dump?db=${db}&type=${state.dumpInfo.type}&tables=${state.dumpInfo.tables.join(',')}&${joinClientParams()}`
    );
    a.click();
    state.showDumpInfo = false;
};

const showColumns = async (row: any) => {
    state.chooseTableName = row.tableName;
    state.columnDialog.columns = await dbApi.columnMetadata.request({
        id: props.dbId,
        db: props.db,
        tableName: row.tableName,
    });

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
    state.ddlDialog.ddl = res;
    state.ddlDialog.visible = true;
};

/**
 * 删除表
 */
const dropTable = async (row: any) => {
    try {
        const tableName = row.tableName;
        await ElMessageBox.confirm(`确定删除'${tableName}'表?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
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

    if (row === false) {
        state.tableCreateDialog.data = { edit: false, row: {}, indexs: [], columns: [] };
        state.tableCreateDialog.title = '创建表';
    }

    if (row.tableName) {
        state.tableCreateDialog.title = '修改表';
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
