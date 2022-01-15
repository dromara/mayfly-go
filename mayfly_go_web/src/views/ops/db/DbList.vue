<template>
    <div class="db-list">
        <el-card>
            <el-button v-auth="permissions.saveDb" type="primary" icon="el-icon-plus" size="mini" @click="editDb(true)">添加</el-button>
            <el-button v-auth="permissions.saveDb" :disabled="chooseId == null" @click="editDb(false)" type="primary" icon="el-icon-edit" size="mini"
                >编辑</el-button
            >
            <el-button
                v-auth="permissions.delDb"
                :disabled="chooseId == null"
                @click="deleteDb(chooseId)"
                type="danger"
                icon="el-icon-delete"
                size="mini"
                >删除</el-button
            >
            <div style="float: right">
                <el-form class="search-form" label-position="right" :inline="true" label-width="60px" size="small">
                    <el-form-item prop="project">
                        <el-select v-model="query.projectId" placeholder="请选择项目" filterable clearable>
                            <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                        </el-select>
                    </el-form-item>

                    <el-form-item>
                        <el-input v-model="query.database" placeholder="请输入数据库" auto-complete="off" clearable></el-input>
                    </el-form-item>
                    <el-button v-waves type="primary" icon="el-icon-search" size="mini" @click="search()">查询</el-button>
                </el-form>
            </div>
            <el-table :data="datas" ref="table" @current-change="choose" show-overflow-tooltip>
                <el-table-column label="选择" width="50px">
                    <template #default="scope">
                        <el-radio v-model="chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="project" label="项目" min-width="100"></el-table-column>
                <el-table-column prop="env" label="环境" min-width="100"></el-table-column>
                <el-table-column prop="name" label="名称" min-width="200"></el-table-column>
                <el-table-column min-width="160" label="host:port">
                    <template #default="scope">
                        {{ `${scope.row.host}:${scope.row.port}` }}
                    </template>
                </el-table-column>
                <el-table-column prop="type" label="类型" min-width="80"></el-table-column>
                <el-table-column prop="database" label="数据库" min-width="120"></el-table-column>
                <el-table-column prop="username" label="用户名" min-width="100"></el-table-column>

                <el-table-column min-width="115" prop="creator" label="创建账号"></el-table-column>
                <el-table-column min-width="160" prop="createTime" label="创建时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>

                <el-table-column fixed="right" label="更多信息" min-width="100">
                    <template #default="scope">
                        <el-link @click.prevent="tableInfo(scope.row)" type="success">表信息</el-link>
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination
                    style="text-align: right"
                    @current-change="handlePageChange"
                    :total="total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"
                ></el-pagination>
            </el-row>
        </el-card>

        <el-dialog
            width="75%"
            :title="`${chooseData ? chooseData.database : ''} 表信息`"
            :before-close="closeTableInfo"
            v-model="tableInfoDialog.visible"
        >
            <el-row class="mb10">
                <el-button type="primary" size="mini" @click="tableCreateDialog.visible = true">创建表</el-button>
            </el-row>
            <el-table border :data="tableInfoDialog.infos" size="small">
                <el-table-column property="tableName" label="表名" min-width="150" show-overflow-tooltip></el-table-column>
                <el-table-column property="tableComment" label="备注" min-width="150" show-overflow-tooltip></el-table-column>
                <el-table-column
                    prop="tableRows"
                    label="Rows"
                    min-width="70"
                    sortable
                    :sort-method="(a, b) => parseInt(a.tableRows) - parseInt(b.tableRows)"
                ></el-table-column>
                <el-table-column
                    property="dataLength"
                    label="数据大小"
                    sortable
                    :sort-method="(a, b) => parseInt(a.dataLength) - parseInt(b.dataLength)"
                >
                    <template #default="scope">
                        {{ formatByteSize(scope.row.dataLength) }}
                    </template>
                </el-table-column>
                <el-table-column
                    property="indexLength"
                    label="索引大小"
                    sortable
                    :sort-method="(a, b) => parseInt(a.indexLength) - parseInt(b.indexLength)"
                >
                    <template #default="scope">
                        {{ formatByteSize(scope.row.indexLength) }}
                    </template>
                </el-table-column>
                <el-table-column property="createTime" label="创建时间" min-width="150"> </el-table-column>
                <el-table-column label="更多信息" min-width="100">
                    <template #default="scope">
                        <el-link @click.prevent="showColumns(scope.row)" type="primary">字段</el-link>
                        <el-link class="ml5" @click.prevent="showTableIndex(scope.row)" type="success">索引</el-link>
                        <el-link class="ml5" @click.prevent="showCreateDdl(scope.row)" type="info">SQL</el-link>
                    </template>
                </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="40%" :title="`${chooseTableName} 字段信息`" v-model="columnDialog.visible">
            <el-table border :data="columnDialog.columns" size="mini">
                <el-table-column prop="columnName" label="名称" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnComment" label="备注" show-overflow-tooltip> </el-table-column>
                <el-table-column width="120" prop="columnType" label="类型" show-overflow-tooltip> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="40%" :title="`${chooseTableName} 索引信息`" v-model="indexDialog.visible">
            <el-table border :data="indexDialog.indexs" size="mini">
                <el-table-column prop="indexName" label="索引名" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="columnName" label="列名" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="seqInIndex" label="列序列号" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="indexType" label="类型"> </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog width="55%" :title="`${chooseTableName} Create-DDL`" v-model="ddlDialog.visible">
            <el-input disabled type="textarea" :autosize="{ minRows: 15, maxRows: 30 }" v-model="ddlDialog.ddl"> </el-input>
        </el-dialog>

        <db-edit
            @val-change="valChange"
            :projects="projects"
            :title="dbEditDialog.title"
            v-model:visible="dbEditDialog.visible"
            v-model:db="dbEditDialog.data"
        ></db-edit>
        <create-table :dbId="dbId" v-model:visible="tableCreateDialog.visible"></create-table>
    </div>
</template>

<script lang='ts'>
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { formatByteSize } from '@/common/utils/format';
import DbEdit from './DbEdit.vue';
import CreateTable from '../component/Table/CreateTable.vue';
import { dbApi } from './api';
import { projectApi } from '../project/api.ts';
export default defineComponent({
    name: 'DbList',
    components: {
        DbEdit,
        CreateTable,
    },
    setup() {
        const state = reactive({
            dbId: 0,
            permissions: {
                saveDb: 'db:save',
                delDb: 'db:del',
            },
            projects: [],
            chooseId: null,
            /**
             * 选中的数据
             */
            chooseData: null,
            /**
             * 查询条件
             */
            query: {
                pageNum: 1,
                pageSize: 10,
            },
            datas: [],
            total: 0,

            chooseTableName: '',
            tableInfoDialog: {
                visible: false,
                infos: [],
            },
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
            dbEditDialog: {
                visible: false,
                data: null,
                title: '新增数据库',
            },
            tableCreateDialog: {
                visible: false,
            },
        });

        onMounted(async () => {
            search();
            state.projects = (await projectApi.projects.request({ pageNum: 1, pageSize: 100 })).list;
        });

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.chooseId = item.id;
            state.chooseData = item;
        };

        const search = async () => {
            let res: any = await dbApi.dbs.request(state.query);
            state.datas = res.list;
            state.total = res.total;
        };

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            search();
        };

        const editDb = (isAdd = false) => {
            if (isAdd) {
                state.dbEditDialog.data = null;
                state.dbEditDialog.title = '新增数据库';
            } else {
                state.dbEditDialog.data = state.chooseData;
                state.dbEditDialog.title = '修改数据库';
            }
            state.dbEditDialog.visible = true;
        };

        const valChange = () => {
            search();
        };

        const deleteDb = async (id: number) => {
            try {
                await ElMessageBox.confirm(`确定删除该库?`, '提示', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                });
                await dbApi.deleteDb.request({ id });
                ElMessage.success('删除成功');
                state.chooseData = null;
                state.chooseId = null;
                search();
            } catch (err) {}
        };

        const tableInfo = async (row: any) => {
            state.tableInfoDialog.infos = await dbApi.tableInfos.request({ id: row.id });
            state.dbId = row.id;
            state.tableInfoDialog.visible = true;
        };

        const closeTableInfo = () => {
            state.tableInfoDialog.visible = false;
            state.tableInfoDialog.infos = [];
        };

        const showColumns = async (row: any) => {
            state.chooseTableName = row.tableName;
            state.columnDialog.columns = await dbApi.columnMetadata.request({
                id: state.chooseId,
                tableName: row.tableName,
            });

            state.columnDialog.visible = true;
        };

        const showTableIndex = async (row: any) => {
            state.chooseTableName = row.tableName;
            state.indexDialog.indexs = await dbApi.tableIndex.request({
                id: state.chooseId,
                tableName: row.tableName,
            });

            state.indexDialog.visible = true;
        };

        const showCreateDdl = async (row: any) => {
            state.chooseTableName = row.tableName;
            const res = await dbApi.tableDdl.request({
                id: state.chooseId,
                tableName: row.tableName,
            });
            state.ddlDialog.ddl = res[0]['Create Table'];
            console.log(state.ddlDialog);
            state.ddlDialog.visible = true;
        };

        return {
            ...toRefs(state),
            // enums,
            search,
            choose,
            handlePageChange,
            editDb,
            valChange,
            deleteDb,
            tableInfo,
            closeTableInfo,
            showColumns,
            showTableIndex,
            showCreateDdl,
            formatByteSize,
        };
    },
});
</script>
<style lang="scss">
</style>
