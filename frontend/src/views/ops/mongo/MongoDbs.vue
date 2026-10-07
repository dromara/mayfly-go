<template>
    <div>
        <el-dialog v-model="visible" width="860px" :title="$t('mongo.dbList')" :before-close="close">
            <div class="mb-2 flex items-center gap-2">
                <el-button v-auth="perms.ddlSave" type="primary" icon="plus" size="small" @click="openCreateDb">
                    {{ $t('mongo.createDbAndColl') }}
                </el-button>
                <el-button size="small" icon="refresh" :loading="loading" @click="showDatabases">{{ $t('common.refresh') }}</el-button>
                <span class="text-[12px] text-gray-400">{{ $t('mongo.collLocateTip') }}</span>
            </div>

            <!-- 集合就地展开在所属库下面：旧实现是「库弹窗 → 集合弹窗 → 统计弹窗」三层嵌套，
                 关掉一层就忘了自己在哪，而集合统计/索引已经归到操作视图的集合信息面板里 -->
            <el-table v-loading="loading" :data="databases" :max-height="480" row-key="name" @expand-change="onExpandChange">
                <el-table-column type="expand">
                    <template #default="{ row }">
                        <div class="px-8 py-2">
                            <div class="mb-2 flex items-center gap-2">
                                <el-button v-auth="perms.ddlSave" size="small" icon="plus" @click="openCreateCollection(row.name)">
                                    {{ $t('mongo.createColl') }}
                                </el-button>
                            </div>
                            <el-table v-loading="collLoading === row.name" :data="collectionsOf(row.name)" size="small" max-height="240">
                                <el-table-column min-width="200" prop="name" :label="$t('common.name')" show-overflow-tooltip />
                                <el-table-column min-width="90" :label="$t('common.type')">
                                    <template #default="{ row: coll }">
                                        {{ coll.type === 'collection' ? $t('mongo.coll') : coll.type }}
                                    </template>
                                </el-table-column>
                                <el-table-column min-width="180" :label="$t('common.operation')">
                                    <template #default="{ row: coll }">
                                        <el-button link type="primary" @click="onOpenData(row.name, coll)">
                                            {{ $t('mongo.openData') }}
                                        </el-button>
                                        <el-button v-auth="perms.ddlDel" link type="danger" @click="onDeleteCollection(row.name, coll.name)">
                                            {{ $t('common.delete') }}
                                        </el-button>
                                    </template>
                                </el-table-column>
                                <template #empty>
                                    <span class="text-[13px] text-gray-400">{{ $t('mongo.noColl') }}</span>
                                </template>
                            </el-table>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column min-width="180" prop="name" :label="$t('common.name')" show-overflow-tooltip />
                <el-table-column min-width="100" :label="$t('mongo.dbSize')">
                    <template #default="{ row }">{{ formatByteSize(row.sizeOnDisk) }}</template>
                </el-table-column>
                <el-table-column min-width="80" :label="$t('mongo.isEmpty')">
                    <template #default="{ row }">{{ row.empty ? $t('common.yes') : $t('common.no') }}</template>
                </el-table-column>
                <el-table-column min-width="150" :label="$t('common.operation')">
                    <template #default="{ row }">
                        <el-button link type="success" @click="showDatabaseStats(row.name)">dbStats</el-button>
                        <el-button v-auth="perms.ddlDel" link type="danger" @click="onDeleteDb(row.name)">{{ $t('common.delete') }}</el-button>
                    </template>
                </el-table-column>
                <template #empty>
                    <span class="text-[13px] text-gray-400">{{ $t('mongo.noDb') }}</span>
                </template>
            </el-table>
        </el-dialog>

        <el-dialog v-model="dbStatsDialog.visible" width="700px" :title="dbStatsDialog.title">
            <el-descriptions :column="3" border>
                <el-descriptions-item v-for="item in statsItems(dbStatsDialog.data)" :key="item.label" :label="item.label" label-align="right" align="center">
                    {{ item.value }}
                </el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <el-dialog v-model="createDbDialog.visible" width="420px" :title="$t('mongo.createDbAndColl')" destroy-on-close>
            <auto-form v-model="createDbDialog.form" :items="createDbItems" label-width="auto" />
            <template #footer>
                <el-button @click="createDbDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="onCreateDb">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>

        <el-dialog v-model="createCollDialog.visible" width="420px" :title="$t('mongo.createColl')" destroy-on-close>
            <auto-form v-model="createCollDialog.form" :items="createCollItems" label-width="auto" />
            <template #footer>
                <el-button @click="createCollDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="onCreateCollection">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, toRefs, watch } from 'vue';

import { formatByteSize } from '@/common/utils/format';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { Msg } from '@/hooks/useI18n';
import { mongoApi } from './api';
import { confirmByName } from './confirm';
import { openMongoCollection } from './resource';
import { perms } from './perms';
import type { MongoCollection, MongoDatabase } from './types';

const props = defineProps<{ id: number; name: string; code: string }>();

const visible = defineModel<boolean>('visible', { default: false });

const state = reactive({
    loading: false,
    databases: [] as MongoDatabase[],
    /** 集合按库名缓存：展开哪个库就取哪个，全量预取会让「上千集合」的实例一次卡住 */
    collections: {} as Record<string, MongoCollection[]>,
    collLoading: '',
    dbStatsDialog: { visible: false, title: '', data: {} as Record<string, unknown> },
    createDbDialog: { visible: false, form: { dbName: '', collectionName: '' } },
    /** 新建集合归属的库，展开行里的按钮带进来 */
    createCollDialog: { visible: false, database: '', form: { collection: '' } },
});

const { loading, collLoading, dbStatsDialog, createDbDialog, createCollDialog, collections, databases } = toRefs(state);

/** 建库&建集合表单声明：Mongo 的库在首个集合创建时才真正存在，因此两个名称一起收集 */
const createDbItems: AutoFormItem[] = [
    { prop: 'dbName', label: 'mongo.dbName', required: true },
    { prop: 'collectionName', label: 'mongo.collName', required: true },
];

const createCollItems: AutoFormItem[] = [{ prop: 'collection', label: 'mongo.collName', required: true }];

watch(visible, (open) => {
    if (open) {
        showDatabases();
    }
});

function close() {
    visible.value = false;
}

async function showDatabases() {
    state.loading = true;
    try {
        state.databases = (await mongoApi.databases.request({ id: props.id })) ?? [];
        // 换了一轮库列表就把集合缓存丢掉：集合可能被别人增删，留着会显示已经不存在的名
        state.collections = {};
    } finally {
        state.loading = false;
    }
}

function collectionsOf(database: string): MongoCollection[] {
    return state.collections[database] ?? [];
}

async function loadCollections(database: string) {
    state.collLoading = database;
    try {
        state.collections[database] = (await mongoApi.collections.request({ id: props.id, database })) ?? [];
    } finally {
        state.collLoading = '';
    }
}

/**
 * 展开某个库时才取它的集合。
 *
 * 每次展开都重取：集合是别人可能刚改过的东西，缓存到上一轮会显示已删除的集合名；
 * 集合数上千的实例也不会因为打开这个弹窗就把全部集合拉一遍。
 */
function onExpandChange(row: MongoDatabase, expandedRows: MongoDatabase[]) {
    if ((expandedRows ?? []).some((item) => item.name === row.name)) {
        void loadCollections(row.name);
    }
}

async function showDatabaseStats(database: string) {
    state.dbStatsDialog.data = await mongoApi.runCommand.request({ id: props.id, database, command: { dbStats: 1 } });
    state.dbStatsDialog.title = `${database} stats`;
    state.dbStatsDialog.visible = true;
}

/**
 * 统计读数。
 *
 * 按 dbStats 的字节类字段统一格式化，其余原样显示：逐字段写死一版布局会随服务端版本漏字段，
 * 这里按返回内容渲染反而稳定。集合级统计不走这里，它在集合信息面板里与索引同进退。
 */
function statsItems(data: Record<string, unknown>) {
    const byteFields = [
        'sizeOnDisk',
        'size',
        'storageSize',
        'freeStorageSize',
        'totalSize',
        'dataSize',
        'indexSize',
        'fsTotalSize',
        'fsUsedSize',
        'totalIndexSize',
    ];
    const numericFields = ['count', 'nindexes', 'indexes', 'collections', 'objects', 'avgObjSize'];

    return Object.keys(data ?? {})
        .filter((key) => typeof data[key] === 'number' || typeof data[key] === 'string')
        .map((key) => ({
            label: key,
            value: byteFields.includes(key) ? formatByteSize(Number(data[key])) : numericFields.includes(key) ? String(data[key]) : String(data[key]),
        }));
}

function onOpenData(database: string, coll: MongoCollection) {
    openMongoCollection({ code: props.code, id: props.id, instName: props.name, database, collection: coll.name, readOnly: coll.readOnly });
    close();
}

async function onDeleteCollection(database: string, collection: string) {
    if (!(await confirmByName(collection))) {
        return;
    }
    await mongoApi.dropCollection.request({ id: props.id, database, collection, confirmName: collection });
    Msg.deleteSuccess();
    await loadCollections(database);
}

async function onDeleteDb(database: string) {
    if (!(await confirmByName(database))) {
        return;
    }
    await mongoApi.dropDatabase.request({ id: props.id, database, confirmName: database });
    Msg.deleteSuccess();
    await showDatabases();
}

function openCreateCollection(database: string) {
    state.createCollDialog.database = database;
    state.createCollDialog.form = { collection: '' };
    state.createCollDialog.visible = true;
}

async function onCreateCollection() {
    const { collection } = state.createCollDialog.form;
    const database = state.createCollDialog.database;
    if (!collection.trim()) {
        Msg.error('mongo.collNameRequired');
        return;
    }
    await mongoApi.createCollection.request({ id: props.id, database, collection });
    Msg.saveSuccess();
    state.createCollDialog.visible = false;
    await loadCollections(database);
}

function openCreateDb() {
    state.createDbDialog.form = { dbName: '', collectionName: '' };
    state.createDbDialog.visible = true;
}

async function onCreateDb() {
    const { dbName, collectionName } = state.createDbDialog.form;
    if (!dbName.trim() || !collectionName.trim()) {
        Msg.error('mongo.dbNameRequired');
        return;
    }
    // Mongo 没有 createDatabase 命令，库在首次写入时才存在，因此以「在新库下建集合」实现
    await mongoApi.createCollection.request({ id: props.id, database: dbName, collection: collectionName });
    Msg.saveSuccess();
    state.createDbDialog.visible = false;
    await showDatabases();
}
</script>

<style lang="scss" scoped></style>
