<template>
    <div class="component-container">
        <el-space>
            <el-tooltip :content="t('db.selectDbPlaceholder')" placement="top">
                <el-select size="small" v-model="selectedDb" style="width: 150px" @change="onChangeDb">
                    <el-option v-for="item in dbs" :key="item.name" :value="item.name">{{ item.name }}</el-option>
                </el-select>
            </el-tooltip>
            <el-button size="small" type="primary" @click="handleCreate" icon="plus">{{ $t('milvus.createCollection') }}</el-button>
            <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
        </el-space>

        <el-table :data="list">
            <el-table-column prop="name" :label="$t('common.name')" sortable width="220">
                <template #default="{ row }">
                    <el-link type="primary" underline="never" @click="handleDataOperation(row.name)">{{ row.name }}</el-link>
                </template>
            </el-table-column>
            <el-table-column prop="created_time" :label="$t('common.createTime')" sortable width="160" />
            <el-table-column :label="$t('milvus.loadStatus')" width="100">
                <template #default="{ row }">
                    <el-tag v-if="row.Loaded" type="success" @click="handleReleaseClick(row)" style="cursor: pointer">
                        {{ $t('milvus.loaded') }}
                    </el-tag>
                    <el-tag v-else-if="row.LoadedPercentage > 0 && row.LoadedPercentage < 100" type="warning">
                        {{ $t('milvus.loading') }} {{ row.LoadedPercentage }}%
                    </el-tag>
                    <el-tag v-else type="info" @click="handleLoadClick(row)" style="cursor: pointer">
                        {{ $t('milvus.unloaded') }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column :label="$t('milvus.aliases')" min-width="200">
                <template #default="{ row }">
                    <div class="alias-container">
                        <el-tag v-for="alias in row.aliases || []" :key="alias" size="small" closable @close="handleDeleteAlias(row, alias)" class="alias-tag">
                            {{ alias }}
                        </el-tag>
                        <el-button size="small" text @click="handleAddAlias(row)" icon="plus" class="add-alias-btn">
                            {{ $t('milvus.addAlias') }}
                        </el-button>
                    </div>
                </template>
            </el-table-column>
            <el-table-column :label="$t('common.operation')" width="250">
                <template #default="{ row }">
                    <el-space>
                        <el-button size="small" type="warning" @click="handleEdit(row)" text>{{ $t('common.edit') }}</el-button>
                        <el-button size="small" type="primary" text @click="handleCopy(row)">{{ $t('common.copy') }}</el-button>
                        <el-button size="small" @click="handleDescribe(row)" text>{{ $t('milvus.detail') }}</el-button>
                        <el-button size="small" type="danger" @click="handleDrop(row)" text>{{ $t('common.delete') }}</el-button>
                    </el-space>
                </template>
            </el-table-column>
        </el-table>
    </div>

    <CollectionsCreate v-model:visible="createDrawerVisible" :milvus-id="milvusId" :mode="drawerMode" :edit-data="editData" @success="loadList" />

    <el-dialog v-model="aliasDialogVisible" :title="$t('milvus.addAlias')" width="400px">
        <el-form @submit.prevent="submitAddAlias">
            <el-form-item :label="$t('milvus.aliasName')">
                <el-input v-model="newAlias" :placeholder="$t('milvus.aliasPlaceholder')" />
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="aliasDialogVisible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitAddAlias" :loading="aliasLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { ElSpace } from 'element-plus';
import { storeToRefs } from 'pinia';
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { milvusApi } from '../api';
import type { ICollection, IMilvusCollectionDetail } from '../types';
import CollectionsCreate from './CollectionsCreate.vue';

const { t } = useI18n();
const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');
const { dbs, selectedDb } = storeToRefs(milvusStore);
const emit = defineEmits(['changeTab']);

const list = ref<ICollection[]>([]);
const createDrawerVisible = ref(false);
const loading = ref(false);
const drawerMode = ref<'create' | 'edit' | 'copy'>('create');
const editData = ref<IMilvusCollectionDetail | null>(null);

// 别名相关
const aliasDialogVisible = ref(false);
const newAlias = ref('');
const aliasLoading = ref(false);
const currentCollectionForAlias = ref<ICollection | null>(null);

// 轮询定时器
let pollingTimer: number | null = null;

const loadList = async () => {
    loading.value = true;
    try {
        const res = await milvusApi.listCollections(props.milvusId);
        // 以name排序
        res.sort((a, b) => {
            return a.name.localeCompare(b.name);
        });
        list.value = res || [];
        milvusStore.setCollections(res.map((a) => a.name));

        // 加载每个集合的别名
        await loadAllAliases();

        // 开始轮询加载中的集合
        startPolling();
    } finally {
        loading.value = false;
    }
};

// 加载所有集合的别名
const loadAllAliases = async () => {
    const promises = list.value.map(async (item) => {
        try {
            const aliases = await milvusApi.listAliases(props.milvusId, item.name);
            item.aliases = aliases || [];
        } catch (e) {
            item.aliases = [];
        }
    });
    await Promise.all(promises);
};

// 轮询加载中的集合
const startPolling = () => {
    stopPolling();

    const loadingItems = list.value.filter((item) => (item.LoadedPercentage ?? 0) > 0 && (item.LoadedPercentage ?? 0) < 100);
    if (loadingItems.length === 0) return;

    pollingTimer = window.setInterval(async () => {
        const stillLoading = list.value.filter((item) => (item.LoadedPercentage ?? 0) > 0 && (item.LoadedPercentage ?? 0) < 100);
        if (stillLoading.length === 0) {
            stopPolling();
            return;
        }

        // 刷新列表
        await loadList();
    }, 2000);
};

const stopPolling = () => {
    if (pollingTimer !== null) {
        globalThis.clearInterval(pollingTimer);
        pollingTimer = null;
    }
};

const handleCreate = () => {
    drawerMode.value = 'create';
    editData.value = null;
    createDrawerVisible.value = true;
};

const handleEdit = async (row: ICollection) => {
    const data = await milvusApi.describeCollection(props.milvusId, row.name);
    drawerMode.value = 'edit';
    editData.value = data;
    createDrawerVisible.value = true;
};

const handleDataOperation = (name: string) => {
    milvusStore.setSelectedCollection(name);
    emit('changeTab', 'data');
};

// 为每个字段加载索引信息
const handleCopy = async (row: ICollection) => {
    const data = await milvusApi.describeCollection(props.milvusId, row.name);
    drawerMode.value = 'copy';
    editData.value = data;
    createDrawerVisible.value = true;
};

const handleDescribe = async (row: ICollection) => {
    const res = await milvusApi.describeCollection(props.milvusId, row.name);
    MonacoEditorBox({
        content: JSON.stringify(res, null, 2),
        title: t('milvus.collectionDetail'),
        language: 'json',
        showConfirmButton: false,
        canChangeLang: false,
        closeFn: () => {},
        useDrawer: true,
        drawerSize: '50%',
        options: {
            readOnly: true,
        },
    });
};

// 点击加载状态 tag - 未加载时提示加载
const handleLoadClick = async (row: ICollection) => {
    await useI18nConfirm('milvus.confirmLoadCollection', { name: row.name });
    handleLoad(row);
};

// 点击加载状态 tag - 已加载时提示释放
const handleReleaseClick = async (row: ICollection) => {
    await useI18nConfirm('milvus.confirmReleaseCollection', { name: row.name });
    handleRelease(row);
};

const handleLoad = (row: ICollection) => {
    milvusApi
        .loadCollection(props.milvusId, row.name, { async: true })
        .then(() => {
            Msg.success('milvus.loadedSuccess');
            loadList();
        })
        .catch((error: unknown) => {
            Msg.error(error instanceof Error ? error.message : String(error));
        });
};

const handleRelease = (row: ICollection) => {
    milvusApi
        .releaseCollection(props.milvusId, row.name)
        .then(() => {
            Msg.success('milvus.releasedSuccess');
            loadList();
        })
        .catch((error: unknown) => {
            Msg.error(error instanceof Error ? error.message : String(error));
        });
};

const handleDrop = async (row: ICollection) => {
    await useI18nConfirm('milvus.confirmDeleteCollection', { name: row.name });

    await milvusApi.dropCollection(props.milvusId, row.name);
    Msg.success('milvus.deletedSuccess');
    await loadList();
};

// 别名操作
const handleAddAlias = (row: ICollection) => {
    currentCollectionForAlias.value = row;
    newAlias.value = '';
    aliasDialogVisible.value = true;
};

const submitAddAlias = async () => {
    if (!newAlias.value || !currentCollectionForAlias.value) return;

    aliasLoading.value = true;
    try {
        await milvusApi.createAlias(props.milvusId, currentCollectionForAlias.value.name, newAlias.value);
        Msg.success('milvus.addedAliasSuccess');
        aliasDialogVisible.value = false;
        await loadList();
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message : String(error));
    } finally {
        aliasLoading.value = false;
    }
};

const handleDeleteAlias = async (row: ICollection, alias: string) => {
    await useI18nConfirm('milvus.confirmDeleteAlias', { name: alias });

    try {
        await milvusApi.dropAlias(props.milvusId, alias);
        Msg.success('milvus.deletedAliasSuccess');
        await loadList();
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message : String(error));
    }
};

const onChangeDb = (v: string) => {
    milvusStore.selectedCollection = '';
    milvusStore.setSelectedDb(v);
    milvusApi.useDatabase(props.milvusId, v);
    loadList();
};

onMounted(loadList);

onBeforeUnmount(() => {
    stopPolling();
});

watch([() => props.milvusId, () => milvusStore.authCertName], async () => {
    list.value = [];
    milvusStore.clear();
    await milvusStore.refreshDbs(props.milvusId);
    milvusApi.useDatabase(props.milvusId, 'default');
    await loadList();
});
</script>

<style scoped>
.component-container {
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.component-container :deep(.el-table) {
    flex: 1;
    min-height: 0;
}

pre {
    background: #f5f7fa;
    padding: 15px;
    border-radius: 4px;
    max-height: 500px;
    overflow: auto;
}

.alias-container {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 4px;
}

.alias-tag {
    margin-right: 4px;
}

.add-alias-btn {
    padding: 0 4px;
    font-size: 12px;
}
</style>
