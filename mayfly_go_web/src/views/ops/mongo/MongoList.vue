<template>
    <div>
        <page-table ref="pageTableRef" :query="state.queryConfig" v-model:query-form="query" :show-selection="true"
            v-model:selection-data="selectionData" :data="list" :columns="state.columns" :total="total"
            v-model:page-size="query.pageSize" v-model:page-num="query.pageNum" @pageChange="search()">

            <template #tagPathSelect>
                <el-select @focus="getTags" v-model="query.tagPath" placeholder="请选择标签" @clear="search" filterable clearable
                    style="width: 200px">
                    <el-option v-for="item in tags" :key="item" :label="item" :value="item"> </el-option>
                </el-select>
            </template>

            <template #queryRight>
                <el-button type="primary" icon="plus" @click="editMongo(true)" plain>添加</el-button>
                <el-button type="primary" icon="edit" :disabled="selectionData.length != 1" @click="editMongo(false)"
                    plain>编辑
                </el-button>
                <el-button type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteMongo" plain>删除
                </el-button>
            </template>

            <template #tagPath="{ data }">
                <tag-info :tag-path="data.tagPath" />
                <span class="ml5">
                    {{ data.tagPath }}
                </span>
            </template>

            <template #action="{ data }">
                <el-link type="primary" @click="showDatabases(data.id)" plain size="small" :underline="false">数据库</el-link>
            </template>
        </page-table>

        <el-dialog width="800px" :title="databaseDialog.title" v-model="databaseDialog.visible">
            <el-table :data="databaseDialog.data" size="small">
                <el-table-column min-width="130" property="Name" label="库名" />
                <el-table-column min-width="90" property="SizeOnDisk" label="size">
                    <template #default="scope">
                        {{ formatByteSize(scope.row.SizeOnDisk) }}
                    </template>
                </el-table-column>
                <el-table-column min-width="80" property="Empty" label="是否为空" />

                <el-table-column min-width="150" label="操作">
                    <template #default="scope">
                        <el-link type="success" @click="showDatabaseStats(scope.row.Name)" plain size="small"
                            :underline="false">stats</el-link>
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-link type="primary" @click="showCollections(scope.row.Name)" plain size="small"
                            :underline="false">集合</el-link>
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
                        <el-link type="success" @click="showCollectionStats(scope.row.name)" plain size="small"
                            :underline="false">stats</el-link>
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-popconfirm @confirm="onDeleteCollection(scope.row.name)" title="确定删除该集合?">
                            <template #reference>
                                <el-link type="danger" plain size="small" :underline="false">删除</el-link>
                            </template>
                        </el-popconfirm>
                    </template>
                </el-table-column>
            </el-table>

            <el-dialog width="700px" :title="collectionsDialog.statsDialog.title"
                v-model="collectionsDialog.statsDialog.visible">
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

        <mongo-edit @val-change="valChange" :title="mongoEditDialog.title" v-model:visible="mongoEditDialog.visible"
            v-model:mongo="mongoEditDialog.data"></mongo-edit>
    </div>
</template>

<script lang="ts" setup>
import { mongoApi } from './api';
import { ref, toRefs, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { tagApi } from '../tag/api';
import MongoEdit from './MongoEdit.vue';
import { formatByteSize } from '@/common/utils/format';
import TagInfo from '../component/TagInfo.vue';
import PageTable from '@/components/pagetable/PageTable.vue'
import { TableColumn, TableQuery } from '@/components/pagetable';

const pageTableRef: any = ref(null)

const state = reactive({
    tags: [],
    dbOps: {
        dbId: 0,
        db: '',
    },
    list: [],
    total: 0,
    selectionData: [],
    query: {
        pageNum: 1,
        pageSize: 10,
        tagPath: null,
    },
    queryConfig: [
        TableQuery.slot("tagPath", "标签", "tagPathSelect"),
    ],
    columns: [
        TableColumn.new("tagPath", "标签路径").setSlot("tagPath").setAddWidth(20),
        TableColumn.new("name", "名称"),
        TableColumn.new("uri", "连接uri"),
        TableColumn.new("createTime", "创建时间").isTime(),
        TableColumn.new("creator", "创建人"),
        TableColumn.new("action", "操作").setSlot("action").setMinWidth(100).fixedRight(),
    ],
    mongoEditDialog: {
        visible: false,
        data: null as any,
        title: '新增mongo',
    },
    databaseDialog: {
        visible: false,
        data: [],
        title: '',
        statsDialog: {
            visible: false,
            data: {} as any,
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
            data: {} as any,
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

const {
    tags,
    list,
    total,
    selectionData,
    query,
    mongoEditDialog,
    databaseDialog,
    collectionsDialog,
    createCollectionDialog,
} = toRefs(state)

onMounted(async () => {
    search();
});

const showDatabases = async (id: number) => {
    // state.query.tagPath = row.tagPath
    state.dbOps.dbId = id

    state.databaseDialog.data = (await mongoApi.databases.request({ id })).Databases;
    state.databaseDialog.title = `数据库列表`;
    state.databaseDialog.visible = true;
};

const showDatabaseStats = async (dbName: string) => {
    state.databaseDialog.statsDialog.data = await mongoApi.runCommand.request({
        id: state.dbOps.dbId,
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
    const res = await mongoApi.collections.request({ id: state.dbOps.dbId, database });
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
        id: state.dbOps.dbId,
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
        id: state.dbOps.dbId,
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
        id: state.dbOps.dbId,
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
        await ElMessageBox.confirm(`确定删除【${state.selectionData.map((x: any) => x.name).join(", ")}】mongo信息?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await mongoApi.deleteMongo.request({ id: state.selectionData.map((x: any) => x.id).join(",") });
        ElMessage.success('删除成功');
        search();
    } catch (err) { }
};

const search = async () => {
    try {
        pageTableRef.value.loading(true);
        const res = await mongoApi.mongoList.request(state.query);
        state.list = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false);
    }
};

const getTags = async () => {
    state.tags = await tagApi.getAccountTags.request(null);
};

const editMongo = async (isAdd = false) => {
    if (isAdd) {
        state.mongoEditDialog.data = null;
        state.mongoEditDialog.title = '新增mongo';
    } else {
        state.mongoEditDialog.data = state.selectionData[0];
        state.mongoEditDialog.title = '修改mongo';
    }
    state.mongoEditDialog.visible = true;
};

const valChange = () => {
    search();
};

</script>

<style></style>
