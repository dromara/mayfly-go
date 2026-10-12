<template>
    <div class="component-container">
        <data-table :data="list" :pageable="false" :columns="columns" :card="false">
            <template #tableHeader>
                <el-space>
                    <el-tooltip :content="t('db.selectDbPlaceholder')" placement="top">
                        <el-select size="small" v-model="selectedDb" style="width: 150px" @change="onChangeDb">
                            <el-option v-for="item in dbs" :key="item.name" :value="item.name">{{ item.name }}</el-option>
                        </el-select>
                    </el-tooltip>
                    <el-button size="small" type="primary" @click="handleCreate" icon="plus">{{ $t('milvus.createCollection') }}</el-button>
                    <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
                </el-space>
            </template>
            <template #name="{ data }">
                <el-link type="primary" underline="never" @click="handleDataOperation(data.name)">{{ data.name }}</el-link>
            </template>
            <template #loadStatus="{ data }">
                <el-tag v-if="data.Loaded" type="success" @click="handleReleaseClick(data)" style="cursor: pointer">{{ $t('milvus.loaded') }}</el-tag>
                <el-tag v-else-if="data.LoadedPercentage > 0 && data.LoadedPercentage < 100" type="warning">
                    {{ $t('milvus.loading') }} {{ data.LoadedPercentage }}%
                </el-tag>
                <el-tag v-else type="info" @click="handleLoadClick(data)" style="cursor: pointer">{{ $t('milvus.unloaded') }}</el-tag>
            </template>
            <template #aliases="{ data }">
                <div class="alias-container">
                    <el-tag v-for="alias in data.aliases || []" :key="alias" size="small" closable @close="handleDeleteAlias(data, alias)" class="alias-tag">
                        {{ alias }}
                    </el-tag>
                    <el-button size="small" text @click="handleAddAlias(data)" icon="plus" class="add-alias-btn">
                        {{ $t('milvus.addAlias') }}
                    </el-button>
                </div>
            </template>
        </data-table>
    </div>

    <CollectionsCreate v-model:visible="createDrawerVisible" :milvus-id="milvusId" :mode="drawerMode" :edit-data="editData" @success="loadList" />

    <el-dialog v-model="aliasDialogVisible" :title="$t('milvus.addAlias')" width="400px">
        <auto-form v-model="aliasForm" :items="aliasItems" label-width="auto" />
        <template #footer>
            <el-button @click="aliasDialogVisible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitAddAlias" :loading="aliasLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import DataTable from '@/components/data-table/DataTable.vue';
import { TableColumn } from '@/components/data-table';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { ElSpace } from 'element-plus';
import { storeToRefs } from 'pinia';
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
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

// 声明式列：name/加载状态/别名走插槽；操作列由 setActions 统一渲染（text 外观 + 4 格全直显，保留原观感）
const columns = [
    TableColumn.new('name', 'common.name').isSlot('name').setSortable(),
    TableColumn.new('created_time', 'common.createTime').setSortable(),
    TableColumn.new('loadStatus', 'milvus.loadStatus').isSlot('loadStatus'),
    // 别名列不截断（多 tag 换行）：换行态 DOM 回填量不到溢出，故用 setSlotText 量真实 tag 文本 + setAddWidth 为「添加别名」按钮预留
    TableColumn.new('aliases', 'milvus.aliases')
        .isSlot('aliases')
        .noShowOverflowTooltip()
        .setSlotText((row: ICollection) => (row.aliases ?? []).join(' '))
        .setAddWidth(80),
    TableColumn.new('action', 'common.operation').setActions<ICollection>(
        [
            { key: 'edit', label: 'common.edit', type: 'warning', onClick: (row) => handleEdit(row) },
            { key: 'copy', label: 'common.copy', type: 'primary', onClick: (row) => handleCopy(row) },
            { key: 'detail', label: 'milvus.detail', onClick: (row) => handleDescribe(row) },
            {
                key: 'delete',
                label: 'common.delete',
                type: 'danger',
                confirm: { title: 'milvus.confirmDeleteCollection', params: (row) => ({ name: row.name }) },
                onClick: (row) => dropCollection(row),
            },
        ],
        { appearance: 'text', maxInline: 4, size: 'small' }
    ),
];
const createDrawerVisible = ref(false);
const loading = ref(false);
const drawerMode = ref<'create' | 'edit' | 'copy'>('create');
const editData = ref<IMilvusCollectionDetail | null>(null);

// 别名相关
const aliasDialogVisible = ref(false);
const aliasForm = reactive({ name: '' });
/** 添加别名表单声明 */
const aliasItems: AutoFormItem[] = [{ prop: 'name', label: 'milvus.aliasName', required: true, placeholder: 'milvus.aliasPlaceholder' }];
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
    if (!(await useI18nConfirm('milvus.confirmLoadCollection', { name: row.name }))) {
        return;
    }
    handleLoad(row);
};

// 点击加载状态 tag - 已加载时提示释放
const handleReleaseClick = async (row: ICollection) => {
    if (!(await useI18nConfirm('milvus.confirmReleaseCollection', { name: row.name }))) {
        return;
    }
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

const dropCollection = async (row: ICollection) => {
    await milvusApi.dropCollection(props.milvusId, row.name);
    Msg.success('milvus.deletedSuccess');
    await loadList();
};

// 别名操作
const handleAddAlias = (row: ICollection) => {
    currentCollectionForAlias.value = row;
    aliasForm.name = '';
    aliasDialogVisible.value = true;
};

const submitAddAlias = async () => {
    if (!aliasForm.name || !currentCollectionForAlias.value) return;

    aliasLoading.value = true;
    try {
        await milvusApi.createAlias(props.milvusId, currentCollectionForAlias.value.name, aliasForm.name);
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
    if (!(await useI18nConfirm('milvus.confirmDeleteAlias', { name: alias }))) {
        return;
    }

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
