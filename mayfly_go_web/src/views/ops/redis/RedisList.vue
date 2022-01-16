<template>
    <div>
        <el-card>
            <el-button type="primary" icon="plus" @click="editRedis(true)" plain>添加</el-button>
            <el-button type="primary" icon="edit" :disabled="currentId == null" @click="editRedis(false)" plain>编辑</el-button>
            <el-button type="danger" icon="delete" :disabled="currentId == null" @click="deleteRedis" plain>删除</el-button>
            <div style="float: right">
                <!-- <el-input placeholder="host"  style="width: 140px" v-model="query.host" @clear="search" plain clearable></el-input>
                <el-select v-model="params.clusterId"  clearable placeholder="集群选择">
                    <el-option v-for="item in clusters" :key="item.id" :value="item.id" :label="item.name"></el-option>
                </el-select> -->
                <el-select v-model="query.projectId" placeholder="请选择项目" filterable clearable>
                    <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                </el-select>
                <el-button class="ml5" @click="search" type="success" icon="search"></el-button>
            </div>
            <el-table :data="redisTable" style="width: 100%" @current-change="choose">
                <el-table-column label="选择" width="60px">
                    <template #default="scope">
                        <el-radio v-model="currentId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="project" label="项目" width></el-table-column>
                <el-table-column prop="env" label="环境" width></el-table-column>
                <el-table-column prop="host" label="host:port" width></el-table-column>
                <el-table-column prop="createTime" label="创建时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="creator" label="创建人"></el-table-column>
                <el-table-column label="操作" width>
                    <template #default="scope">
                        <el-button type="primary" @click="info(scope.row)" icon="tickets" plain size="small">info</el-button>
                        <!-- <el-button type="success" @click="manage(scope.row)" :ref="scope.row"  plain>数据管理</el-button> -->
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

        <info v-model:visible="infoDialog.visible" :title="infoDialog.title" :info="infoDialog.info"></info>

        <redis-edit
            @val-change="valChange"
            :projects="projects"
            :title="redisEditDialog.title"
            v-model:visible="redisEditDialog.visible"
            v-model:redis="redisEditDialog.data"
        ></redis-edit>
    </div>
</template>

<script lang="ts">
import Info from './Info.vue';
import { redisApi } from './api';
import { toRefs, reactive, defineComponent, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { projectApi } from '../project/api.ts';
import RedisEdit from './RedisEdit.vue';

export default defineComponent({
    name: 'RedisList',
    components: {
        Info,
        RedisEdit,
    },
    setup() {
        const state = reactive({
            projects: [],
            redisTable: [],
            total: 0,
            currentId: null,
            currentData: null,
            query: {
                pageNum: 1,
                pageSize: 10,
                prjectId: null,
                clusterId: null,
            },
            redisInfo: {
                url: '',
            },
            clusters: [
                {
                    id: 0,
                    name: '单机',
                },
            ],
            infoDialog: {
                title: '',
                visible: false,
                info: {
                    Server: {},
                    Keyspace: {},
                    Clients: {},
                    CPU: {},
                    Memory: {},
                },
            },
            redisEditDialog: {
                visible: false,
                data: null,
                title: '新增redis',
            },
        });

        onMounted(async () => {
            search();
            state.projects = (await projectApi.projects.request({ pageNum: 1, pageSize: 100 })).list;
        });

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            search();
        };

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.currentId = item.id;
            state.currentData = item;
        };

        // connect() {
        //   Req.post('/open/redis/connect', this.form, res => {
        //     this.redisInfo = res
        //   })
        // }

        const deleteRedis = async () => {
            try {
                await ElMessageBox.confirm(`确定删除该redis?`, '提示', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                });
                await redisApi.delRedis.request({ id: state.currentId });
                ElMessage.success('删除成功');
                state.currentData = null;
                state.currentId = null;
                search();
            } catch (err) {}
        };

        const info = (redis: any) => {
            redisApi.redisInfo.request({ id: redis.id }).then((res: any) => {
                state.infoDialog.info = res;
                state.infoDialog.title = `'${redis.host}' info`;
                state.infoDialog.visible = true;
            });
        };

        const search = async () => {
            const res = await redisApi.redisList.request(state.query);
            state.redisTable = res.list;
            state.total = res.total;
        };

        const editRedis = (isAdd = false) => {
            if (isAdd) {
                state.redisEditDialog.data = null;
                state.redisEditDialog.title = '新增redis';
            } else {
                state.redisEditDialog.data = state.currentData;
                state.redisEditDialog.title = '修改redis';
            }
            state.redisEditDialog.visible = true;
        };

        const valChange = () => {
            search();
        };

        return {
            ...toRefs(state),
            search,
            handlePageChange,
            choose,
            info,
            deleteRedis,
            editRedis,
            valChange,
        };
    },
});
</script>

<style>
</style>
