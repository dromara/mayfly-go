<template>
    <div class="component-container">
        <el-space>
        <el-select size="small" v-model="selectedCollection" style="min-width: 200px" @change="loadList" filterable clearable :teleported="false">
            <el-option v-for="item in collections" :key="item" :label="item" :value="item" />
        </el-select>

        <el-button type="primary" size="small" icon="plus" @click="handleCreate">
            {{ $t('milvus.createPartition') }}
        </el-button>
        <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
    </el-space>

    <el-table :data="list">
        <el-table-column prop="id" label="id" />
        <el-table-column prop="name" :label="$t('milvus.partitionName')" />
        <el-table-column prop="createTime" :label="$t('common.createTime')" />
        <el-table-column :label="$t('common.operation')" width="200">
            <template #default="{ row }">
                <el-button size="small" type="warning" plain @click="handleRelease(row)">{{ $t('milvus.release') }}</el-button>
                <el-button size="small" type="danger" @click="handleDrop(row)">{{ $t('common.delete') }}</el-button>
            </template>
        </el-table-column>
        </el-table>
    </div>

    <el-dialog v-model="createDialog.visible" :title="$t('milvus.createPartition')" width="500px">
        <auto-form ref="createFormRef" v-model="createForm" :items="createItems" label-width="auto" />
        <template #footer>
            <el-button @click="createDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitCreate" :loading="createLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { storeToRefs } from 'pinia';
import { onMounted, ref, useTemplateRef, watch } from 'vue';
import { milvusApi } from '../api';
import type { IPartition } from '../types';

const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');
const { collections, selectedCollection } = storeToRefs(milvusStore);

const list = ref<IPartition[]>([]);
const createDialog = ref({
    visible: false,
});
const createFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('createFormRef');
const loading = ref(false);
const createLoading = ref(false);
const createForm = ref({
    name: '',
});

/** 建分区表单声明 */
const createItems: AutoFormItem[] = [{ prop: 'name', label: 'milvus.partitionName', required: true, placeholder: 'milvus.partitionNamePlaceholder' }];

const loadList = async () => {
    loading.value = true;
    // 需要先选择 collection，这里暂时加载所有分区
    try {
        const res = await milvusApi.listPartitions(props.milvusId, milvusStore.selectedCollection);
        list.value = res || [];
    } finally {
        loading.value = false;
    }
};

const handleCreate = () => {
    createForm.value = { name: '' };
    createDialog.value.visible = true;
};

const submitCreate = async () => {
    if (!createFormRef.value) return;

    await createFormRef.value?.validate();

    createLoading.value = true;
    try {
        await milvusApi.createPartition(props.milvusId, milvusStore.selectedCollection, createForm.value);
        Msg.success('milvus.createdSuccess');
        createDialog.value.visible = false;
        await loadList();
    } finally {
        createLoading.value = false;
    }
};

const handleDrop = async (row: IPartition) => {
    await useI18nConfirm('milvus.confirmDeletePartition', { name: row.name });
    await milvusApi.dropPartition(props.milvusId, milvusStore.selectedCollection, row.name);
    Msg.success('milvus.deletedSuccess');
    await loadList();
};

const handleRelease = async (row: IPartition) => {
    await milvusApi.releasePartition(props.milvusId, milvusStore.selectedCollection, row.name);
    Msg.success('milvus.releasedSuccess');
};

onMounted(() => {
    loadList();
});

watch(
    [() => props.milvusId, () => milvusStore.authCertName],
    () => {
        list.value = [];
        loadList();
        milvusStore.clear();
    }
);
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
</style>
