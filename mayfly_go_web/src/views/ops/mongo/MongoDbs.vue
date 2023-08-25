<template>
    <div>
        <el-dialog width="800px" title="数据库列表" :before-close="close" v-model="databaseDialog.visible">
            <div class="mb5">
                <el-button @click="showCreateDbDialog" type="primary" icon="plus" size="small">新建</el-button>
            </div>
            <el-table :data="databaseDialog.data" :max-height="500">
                <el-table-column min-width="130" property="Name" label="库名" />
                <el-table-column min-width="90" property="SizeOnDisk" label="size">
                    <template #default="scope">
                        {{ formatByteSize(scope.row.SizeOnDisk) }}
                    </template>
                </el-table-column>
                <el-table-column min-width="80" property="Empty" label="是否为空" />

                <el-table-column min-width="150" label="操作">
                    <template #default="scope">
                        <el-link type="success" @click="showDatabaseStats(scope.row.Name)" plain size="small" :underline="false">stats</el-link>
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-link type="primary" @click="showCollections(scope.row.Name)" plain size="small" :underline="false">集合</el-link>
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-popconfirm @confirm="onDeleteDb(scope.row.Name)" title="确定删除该库?">
                            <template #reference>
                                <el-link type="danger" plain size="small" :underline="false">删除</el-link>
                            </template>
                        </el-popconfirm>
                    </template>
                </el-table-column>
            </el-table>

            <el-dialog width="700px" :title="databaseDialog.statsDialog.title" v-model="databaseDialog.statsDialog.visible">
                <el-descriptions title="库状态信息" :column="3" border>
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
            <div class="mb5">
                <el-button @click="showCreateCollectionDialog" type="primary" icon="plus" size="small">新建</el-button>
            </div>
            <el-table stripe :data="collectionsDialog.data" :max-height="500">
                <el-table-column prop="name" label="名称" show-overflow-tooltip> </el-table-column>
                <el-table-column min-width="80" label="操作">
                    <template #default="scope">
                        <el-link type="success" @click="showCollectionStats(scope.row.name)" plain size="small" :underline="false">stats</el-link>
                        <el-divider direction="vertical" border-style="dashed" />
                        <el-popconfirm @confirm="onDeleteCollection(scope.row.name)" width="160" title="确定删除该集合?">
                            <template #reference>
                                <el-link type="danger" plain size="small" :underline="false">删除</el-link>
                            </template>
                        </el-popconfirm>
                    </template>
                </el-table-column>
            </el-table>

            <el-dialog width="700px" :title="collectionsDialog.statsDialog.title" v-model="collectionsDialog.statsDialog.visible">
                <el-descriptions title="集合状态信息" :column="3" border>
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

        <el-dialog width="400px" title="新建库&集合" v-model="createDbDialog.visible" :destroy-on-close="true">
            <el-form :model="createDbDialog.form" label-width="auto">
                <el-form-item prop="dbName" label="库名" required>
                    <el-input v-model="createDbDialog.form.dbName" clearable></el-input>
                </el-form-item>
                <el-form-item prop="collectionName" label="集合名" required>
                    <el-input v-model="createDbDialog.form.collectionName" clearable></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div>
                    <el-button @click="createDbDialog.visible = false">取 消</el-button>
                    <el-button @click="onCreateDb" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog width="400px" title="新建集合" v-model="createCollectionDialog.visible" :destroy-on-close="true">
            <el-form :model="createCollectionDialog.form" label-width="auto">
                <el-form-item prop="name" label="集合名" required>
                    <el-input v-model="createCollectionDialog.form.name" clearable></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div>
                    <el-button @click="createCollectionDialog.visible = false">取 消</el-button>
                    <el-button @click="onCreateCollection" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { mongoApi } from './api';
import { watch, toRefs, reactive } from 'vue';
import { ElMessage } from 'element-plus';
import { formatByteSize } from '@/common/utils/format';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    id: {
        type: [Number],
        required: true,
    },
});

//定义事件
const emit = defineEmits(['update:visible']);

const state = reactive({
    databaseDialog: {
        visible: false,
        data: [],
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
    createDbDialog: {
        visible: false,
        form: {
            dbName: '',
            collectionName: '',
        },
    },
});

const { databaseDialog, collectionsDialog, createCollectionDialog, createDbDialog } = toRefs(state);

watch(props, async (newValue: any) => {
    if (!newValue.visible) {
        state.databaseDialog.visible = false;
        return;
    }

    showDatabases();
});

const close = () => {
    emit('update:visible', false);
};

const showDatabases = async () => {
    state.databaseDialog.data = (await mongoApi.databases.request({ id: props.id })).Databases;
    state.databaseDialog.visible = true;
};

const showDatabaseStats = async (dbName: string) => {
    state.databaseDialog.statsDialog.data = await mongoApi.runCommand.request({
        id: props.id,
        database: dbName,
        command: [
            {
                dbStats: 1,
            },
        ],
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
    const res = await mongoApi.collections.request({ id: props.id, database });
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
        id: props.id,
        database: state.collectionsDialog.database,
        command: [
            {
                collStats: collection,
            },
        ],
    });
    state.collectionsDialog.statsDialog.title = `'${collection}' stats`;
    state.collectionsDialog.statsDialog.visible = true;
};

/**
 * 删除集合
 */
const onDeleteCollection = async (collection: string) => {
    await mongoApi.runCommand.request({
        id: props.id,
        database: state.collectionsDialog.database,
        command: [
            {
                drop: collection,
            },
        ],
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
        id: props.id,
        database: state.collectionsDialog.database,
        command: [
            {
                create: form.name,
            },
        ],
    });
    ElMessage.success('集合创建成功');
    state.createCollectionDialog.visible = false;
    state.createCollectionDialog.form = {} as any;
    setCollections(state.collectionsDialog.database);
};

const showCreateDbDialog = () => {
    state.createDbDialog.visible = true;
};

const onCreateDb = async () => {
    const form = state.createDbDialog.form;
    await mongoApi.runCommand.request({
        id: props.id,
        database: form.dbName,
        command: [
            {
                create: form.collectionName,
            },
        ],
    });
    ElMessage.success('数据库与集合创建成功');
    state.createDbDialog.visible = false;
    state.createDbDialog.form = {} as any;
    showDatabases();
};

const onDeleteDb = async (db: string) => {
    await mongoApi.runCommand.request({
        id: props.id,
        database: db,
        command: [
            {
                dropDatabase: 1,
            },
        ],
    });
    ElMessage.success('库删除成功');
    showDatabases();
};
</script>

<style></style>
