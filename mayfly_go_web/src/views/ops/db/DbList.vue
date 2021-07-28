<template>
    <div class="db-list">
        <div class="toolbar">
            <el-row>
                <el-col>
                    <el-form class="search-form" label-position="right" :inline="true" label-width="60px" size="small">
                        <el-form-item prop="project" label="项目">
                            <el-select v-model="query.projectId" placeholder="请选择项目" filterable clearable>
                                <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id">
                                </el-option>
                            </el-select>
                        </el-form-item>

                        <el-form-item label="数据库">
                            <el-input v-model="query.database" auto-complete="off" clearable></el-input>
                        </el-form-item>
                        <el-button type="primary" icon="el-icon-search" size="mini" @click="search()">查询</el-button>
                    </el-form>
                </el-col>
            </el-row>

            <el-row class="mt5">
                <el-col>
                    <el-button v-auth="permissions.saveDb" type="primary" icon="el-icon-plus" size="mini" @click="editDb(true)">添加</el-button>
                    <el-button
                        v-auth="permissions.saveDb"
                        :disabled="chooseId == null"
                        @click="editDb(false)"
                        type="primary"
                        icon="el-icon-edit"
                        size="mini"
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
                </el-col>
            </el-row>
        </div>
        <el-table :data="datas" border ref="table" @current-change="choose" show-overflow-tooltip>
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
        </el-table>
        <el-pagination
            @current-change="handlePageChange"
            style="text-align: center"
            background
            layout="prev, pager, next, total, jumper"
            :total="total"
            v-model:current-page="query.pageNum"
            :page-size="query.pageSize"
        />

        <db-edit @val-change="valChange" :projects="projects" :title="dbEditDialog.title" v-model:visible="dbEditDialog.visible" v-model:db="dbEditDialog.data"></db-edit>
    </div>
</template>

<script lang='ts'>
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import ProjectEnvSelect from '../component/ProjectEnvSelect.vue';
import DbEdit from './DbEdit.vue';
import { dbApi } from './api';
import { projectApi } from '../project/api.ts';
export default defineComponent({
    name: 'DbList',
    components: {
        ProjectEnvSelect,
        DbEdit,
    },
    setup() {
        const state = reactive({
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
            dbEditDialog: {
                visible: false,
                data: null,
                title: '新增数据库',
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

        return {
            ...toRefs(state),
            // enums,
            search,
            choose,
            handlePageChange,
            editDb,
            valChange,
            deleteDb,
        };
    },
});
</script>
<style lang="scss">
</style>
