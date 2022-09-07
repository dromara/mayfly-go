<template>
    <div>
        <el-card>
            <el-button type="primary" icon="plus" @click="editMongo(true)" plain>添加</el-button>
            <el-button type="primary" icon="edit" :disabled="currentId == null" @click="editMongo(false)" plain>编辑</el-button>
            <el-button type="danger" icon="delete" :disabled="currentId == null" @click="deleteMongo" plain>删除</el-button>
            <div style="float: right">
                <el-select @focus="getProjects" v-model="query.projectId" placeholder="请选择项目" filterable clearable>
                    <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                </el-select>
                <el-button class="ml5" @click="search" type="success" icon="search"></el-button>
            </div>
            <el-table :data="list" style="width: 100%" @current-change="choose" stripe>
                <el-table-column label="选择" width="60px">
                    <template #default="scope">
                        <el-radio v-model="currentId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="project" label="项目" width></el-table-column>
                <el-table-column prop="env" label="环境" width></el-table-column>
                <el-table-column prop="name" label="名称" width></el-table-column>
                <el-table-column prop="uri" label="连接uri" min-width="150" show-overflow-tooltip>
                    <template #default="scope">
                        {{ scope.row.uri.split('@')[1] }}
                    </template>
                </el-table-column>
                <el-table-column prop="createTime" label="创建时间" min-width="150">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="creator" label="创建人"></el-table-column>

                <el-table-column label="操作" width>
                    <template #default="scope">
                        <el-link type="primary" @click="showDatabases(scope.row.id)" plain size="small" :underline="false">数据库</el-link>
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

        <el-dialog width="800px" :title="databaseDialog.title" v-model="databaseDialog.visible">
            <el-table :data="databaseDialog.data" size="small">
                <el-table-column min-width="130" property="Name" label="库名" />
                <el-table-column min-width="90" property="SizeOnDisk" label="size">
                    <template #default="scope">
                        {{ formatByteSize(scope.row.SizeOnDisk) }}
                    </template>
                </el-table-column>
                <el-table-column min-width="80" property="Empty" label="是否为空" />

                <el-table-column min-width="80" label="操作">
                    <template #default="scope">
                        <el-link type="success" @click="showDatabaseStats(scope.row.Name)" plain size="small" :underline="false">stats</el-link>
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-link type="primary" @click="showCollections(scope.row.Name)" plain size="small" :underline="false">集合</el-link>
                    </template>
                </el-table-column>
            </el-table>

            <el-dialog width="700px" :title="databaseDialog.statsDialog.title" v-model="databaseDialog.statsDialog.visible">
                <el-descriptions title="库状态信息" :column="3" border size="small">
                    <el-descriptions-item label="db" label-align="right" align="center">
                        {{ databaseDialog.statsDialog.data.db }}
                    </el-descriptions-item>
                    <el-descriptions-item label="collections" label-align="right" align="center">
                        {{ databaseDialog.statsDialog.data.collections }}
                    </el-descriptions-item>
                    <el-descriptions-item label="objects" label-align="right" align="center">
                        {{ databaseDialog.statsDialog.data.objects }}
                    </el-descriptions-item>
                    <el-descriptions-item label="indexes" label-align="right" align="center">
                        {{ databaseDialog.statsDialog.data.indexes }}
                    </el-descriptions-item>

                    <el-descriptions-item label="avgObjSize" label-align="right" align="center">
                        {{ formatByteSize(databaseDialog.statsDialog.data.avgObjSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="dataSize" label-align="right" align="center">
                        {{ formatByteSize(databaseDialog.statsDialog.data.dataSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="totalSize" label-align="right" align="center">
                        {{ formatByteSize(databaseDialog.statsDialog.data.totalSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="storageSize" label-align="right" align="center">
                        {{ formatByteSize(databaseDialog.statsDialog.data.storageSize) }}
                    </el-descriptions-item>

                    <el-descriptions-item label="fsTotalSize" label-align="right" align="center">
                        {{ formatByteSize(databaseDialog.statsDialog.data.fsTotalSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="fsUsedSize" label-align="right" align="center">
                        {{ formatByteSize(databaseDialog.statsDialog.data.fsUsedSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="indexSize" label-align="right" align="center">
                        {{ formatByteSize(databaseDialog.statsDialog.data.indexSize) }}
                    </el-descriptions-item>
                </el-descriptions>
            </el-dialog>
        </el-dialog>

        <el-dialog width="600px" :title="collectionsDialog.title" v-model="collectionsDialog.visible">
            <div>
                <el-button @click="showCreateCollectionDialog" type="primary" icon="plus" size="small">新建</el-button>
            </div>
            <el-table border stripe :data="collectionsDialog.data" size="small">
                <el-table-column prop="name" label="名称" show-overflow-tooltip> </el-table-column>
                <el-table-column min-width="80" label="操作">
                    <template #default="scope">
                        <el-link type="success" @click="showCollectionStats(scope.row.name)" plain size="small" :underline="false">stats</el-link>
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-popconfirm @confirm="onDeleteCollection(scope.row.name)" title="确定删除该集合?">
                            <template #reference>
                                <el-link type="danger" plain size="small" :underline="false">删除</el-link>
                            </template>
                        </el-popconfirm>
                    </template>
                </el-table-column>
            </el-table>

            <el-dialog width="700px" :title="collectionsDialog.statsDialog.title" v-model="collectionsDialog.statsDialog.visible">
                <el-descriptions title="集合状态信息" :column="3" border size="small">
                    <el-descriptions-item label="ns" label-align="right" :span="2" align="center">
                        {{ collectionsDialog.statsDialog.data.ns }}
                    </el-descriptions-item>
                    <el-descriptions-item label="count" label-align="right" align="center">
                        {{ collectionsDialog.statsDialog.data.count }}
                    </el-descriptions-item>
                    <el-descriptions-item label="avgObjSize" label-align="right" align="center">
                        {{ formatByteSize(collectionsDialog.statsDialog.data.avgObjSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="nindexes" label-align="right" align="center">
                        {{ collectionsDialog.statsDialog.data.nindexes }}
                    </el-descriptions-item>

                    <el-descriptions-item label="size" label-align="right" align="center">
                        {{ formatByteSize(collectionsDialog.statsDialog.data.size) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="totalSize" label-align="right" align="center">
                        {{ formatByteSize(collectionsDialog.statsDialog.data.totalSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="storageSize" label-align="right" align="center">
                        {{ formatByteSize(collectionsDialog.statsDialog.data.storageSize) }}
                    </el-descriptions-item>
                    <el-descriptions-item label="freeStorageSize" label-align="right" align="center">
                        {{ formatByteSize(collectionsDialog.statsDialog.data.freeStorageSize) }}
                    </el-descriptions-item>
                </el-descriptions>
            </el-dialog>
        </el-dialog>

        <el-dialog width="400px" title="新建集合" v-model="createCollectionDialog.visible" :destroy-on-close="true">
            <el-form :model="createCollectionDialog.form" label-width="70px">
                <el-form-item prop="name" label="集合名" required>
                    <el-input v-model="createCollectionDialog.form.name" clearable></el-input>
                </el-form-item>
                <!-- <el-form-item label="描述:">
                    <el-input v-model="showEnvDialog.envForm.remark" auto-complete="off"></el-input>
                </el-form-item> -->
            </el-form>
            <template #footer>
                <div>
                    <el-button @click="createCollectionDialog.visible = false">取 消</el-button>
                    <el-button @click="onCreateCollection" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <mongo-edit
            @val-change="valChange"
            :projects="projects"
            :title="mongoEditDialog.title"
            v-model:visible="mongoEditDialog.visible"
            v-model:mongo="mongoEditDialog.data"
        ></mongo-edit>
    </div>
</template>

<script lang="ts">
import { mongoApi } from './api';
import { toRefs, reactive, defineComponent, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { projectApi } from '../project/api.ts';
import MongoEdit from './MongoEdit.vue';
import { formatByteSize } from '@/common/utils/format';

export default defineComponent({
    name: 'MongoList',
    components: {
        MongoEdit,
    },
    setup() {
        const state = reactive({
            projects: [],
            list: [],
            total: 0,
            currentId: null,
            currentData: null,
            query: {
                pageNum: 1,
                pageSize: 10,
                prjectId: null,
                clusterId: null,
            },
            mongoEditDialog: {
                visible: false,
                data: null,
                title: '新增mongo',
            },
            databaseDialog: {
                visible: false,
                data: [],
                title: '',
                statsDialog: {
                    visible: false,
                    data: {},
                    title: '',
                },
            },
            collectionsDialog: {
                database: '',
                visible: false,
                data: [],
                title: '',
                statsDialog: {
                    visible: false,
                    data: {},
                    title: '',
                },
            },
            createCollectionDialog: {
                visible: false,
                form: {
                    name: '',
                },
            },
        });

        onMounted(async () => {
            search();
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

        const showDatabases = async (id: number) => {
            state.databaseDialog.data = (await mongoApi.databases.request({ id })).Databases;
            state.databaseDialog.title = `数据库列表`;
            state.databaseDialog.visible = true;
        };

        const showDatabaseStats = async (dbName: string) => {
            state.databaseDialog.statsDialog.data = await mongoApi.runCommand.request({
                id: state.currentId,
                database: dbName,
                command: {
                    dbStats: 1,
                },
            });
            state.databaseDialog.statsDialog.title = `'${dbName}' stats`;
            state.databaseDialog.statsDialog.visible = true;
        };

        const showCollections = async (database: string) => {
            state.collectionsDialog.database = database;
            state.collectionsDialog.data = [];
            setCollections(database);
            state.collectionsDialog.title = `'${database}' 集合`;
            state.collectionsDialog.visible = true;
        };

        const setCollections = async (database: string) => {
            const res = await mongoApi.collections.request({ id: state.currentId, database });
            const collections = [] as any;
            for (let r of res) {
                collections.push({ name: r });
            }
            state.collectionsDialog.data = collections;
        };

        /**
         * 显示集合状态
         */
        const showCollectionStats = async (collection: string) => {
            state.collectionsDialog.statsDialog.data = await mongoApi.runCommand.request({
                id: state.currentId,
                database: state.collectionsDialog.database,
                command: {
                    collStats: collection,
                },
            });
            state.collectionsDialog.statsDialog.title = `'${collection}' stats`;
            state.collectionsDialog.statsDialog.visible = true;
        };

        /**
         * 删除集合
         */
        const onDeleteCollection = async (collection: string) => {
            await mongoApi.runCommand.request({
                id: state.currentId,
                database: state.collectionsDialog.database,
                command: {
                    drop: collection,
                },
            });
            ElMessage.success('集合删除成功');
            setCollections(state.collectionsDialog.database);
        };

        const showCreateCollectionDialog = () => {
            state.createCollectionDialog.visible = true;
        };

        const onCreateCollection = async () => {
            const form = state.createCollectionDialog.form;
            await mongoApi.runCommand.request({
                id: state.currentId,
                database: state.collectionsDialog.database,
                command: {
                    create: form.name,
                },
            });
            ElMessage.success('集合创建成功');
            state.createCollectionDialog.visible = false;
            state.createCollectionDialog.form = {} as any;
            setCollections(state.collectionsDialog.database);
        };

        const deleteMongo = async () => {
            try {
                await ElMessageBox.confirm(`确定删除该mongo?`, '提示', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                });
                await mongoApi.deleteMongo.request({ id: state.currentId });
                ElMessage.success('删除成功');
                state.currentData = null;
                state.currentId = null;
                search();
            } catch (err) {}
        };

        const search = async () => {
            const res = await mongoApi.mongoList.request(state.query);
            state.list = res.list;
            state.total = res.total;
        };

        const getProjects = async () => {
            state.projects = await projectApi.accountProjects.request(null);
        };

        const editMongo = async (isAdd = false) => {
            await getProjects();
            if (isAdd) {
                state.mongoEditDialog.data = null;
                state.mongoEditDialog.title = '新增mongo';
            } else {
                state.mongoEditDialog.data = state.currentData;
                state.mongoEditDialog.title = '修改mongo';
            }
            state.mongoEditDialog.visible = true;
        };

        const valChange = () => {
            state.currentId = null;
            state.currentData = null;
            search();
        };

        return {
            ...toRefs(state),
            getProjects,
            search,
            handlePageChange,
            choose,
            showDatabases,
            showDatabaseStats,
            showCollections,
            showCollectionStats,
            onDeleteCollection,
            showCreateCollectionDialog,
            onCreateCollection,
            formatByteSize,
            deleteMongo,
            editMongo,
            valChange,
        };
    },
});
</script>

<style>
</style>
