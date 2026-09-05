<template>
    <div class="component-container">
        <div>
            <el-button size="small" icon="plus" type="primary" @click="handleCreate">{{ $t('milvus.createResourceGroup') }}</el-button>
            <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
        </div>

        <el-table :data="list">
        <el-table-column prop="name" :label="$t('milvus.resourceGroupName')" />
        <el-table-column :label="$t('common.operation')" width="250">
            <template #default="{ row }">
                <el-button size="small" @click="handleDescribe(row)">{{ $t('milvus.detail') }}</el-button>
                <el-button size="small" type="danger" @click="handleDrop(row)">{{ $t('common.delete') }}</el-button>
            </template>
        </el-table-column>
        </el-table>
    </div>

    <el-dialog v-model="createDialog.visible" :title="$t('milvus.createResourceGroup')" width="500px">
        <auto-form ref="createFormRef" v-model="createForm" :items="createItems" label-width="auto" />
        <template #footer>
            <el-button @click="createDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitCreate" :loading="createLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { onMounted, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { milvusApi } from '../api';

const { t } = useI18n();
const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');

const list = ref<{ name: string }[]>([]);
const createDialog = ref({
    visible: false,
});
const createFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('createFormRef');
const loading = ref(false);
const createLoading = ref(false);
const createForm = ref({
    name: '',
});

/** 建资源组表单声明 */
const createItems: AutoFormItem[] = [{ prop: 'name', label: 'milvus.resourceGroupName', required: true, placeholder: 'milvus.resourceGroupNamePlaceholder' }];
const detailData = ref<Record<string, unknown>>({});

const loadList = async () => {
    loading.value = true;
    try {
        const res = await milvusApi.listResourceGroups(props.milvusId);
        list.value = (res || []).map((item: string) => ({ name: item }));
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
        await milvusApi.createResourceGroup(props.milvusId, createForm.value);
        Msg.success('milvus.createdSuccess');
        createDialog.value.visible = false;
        loadList();
    } finally {
        createLoading.value = false;
    }
};

const handleDescribe = async (row: { name: string }) => {
    const res = await milvusApi.describeResourceGroup(props.milvusId, row.name);
    MonacoEditorBox({
        content: JSON.stringify(res, null, 2),
        title: t('milvus.resourceGroup'),
        language: 'json',
        showConfirmButton: false,
        canChangeLang: true,
        closeFn: () => {},
        useDrawer: true,
        drawerSize: '50%',
        options: {
            readOnly: true,
        },
    });
};

const handleDrop = async (row: { name: string }) => {
    await useI18nConfirm('milvus.confirmDeleteResourceGroup', { name: row.name });
    await milvusApi.dropResourceGroup(props.milvusId, row.name);
    Msg.success('milvus.deletedSuccess');
    await loadList();
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
