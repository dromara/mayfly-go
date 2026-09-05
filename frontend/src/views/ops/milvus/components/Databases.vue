<template>
    <div class="component-container">
        <div>
            <el-button size="small" type="primary" @click="handleCreate" icon="plus">{{ $t('milvus.createDatabase') }}</el-button>
            <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
        </div>

        <el-table :data="dbs" style="width: 100%">
        <el-table-column prop="name" :label="$t('milvus.dbName')" sortable>
            <template #default="{ row }">
                <el-link type="primary" underline="never" @click="handleUse(row)">{{ row.name }}</el-link>
            </template>
        </el-table-column>
        <el-table-column prop="create_time" :label="$t('common.createTime')" sortable />
        <el-table-column :label="$t('common.operation')" width="200">
            <template #default="{ row }">
                <el-button type="warning" size="small" plain @click="handleConfig(row)">{{ $t('milvus.config') }}</el-button>
                <el-button type="danger" size="small" @click="handleDrop(row)">{{ $t('common.delete') }}</el-button>
            </template>
        </el-table-column>
        </el-table>
    </div>

    <el-dialog v-model="createDialog.visible" :title="$t('milvus.createDatabase')" width="500px">
        <auto-form ref="createFormRef" v-model="createForm" :items="createItems" label-width="auto" />
        <template #footer>
            <el-button @click="createDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitCreate" :loading="createLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>

    <el-dialog v-model="configDialog.visible" :title="$t('milvus.databaseProperties')" width="500px">
        <auto-form ref="configFormRef" v-model="configForm" :items="configItems" label-width="auto" />
        <template #footer>
            <el-button @click="configDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitConfig" :loading="configLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { storeToRefs } from 'pinia';
import { onMounted, ref, useTemplateRef, watch } from 'vue';
import { milvusApi, timezones } from '../api';
import type { IDatabase } from '../types';

const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');
const { dbs } = storeToRefs(milvusStore);

const emits = defineEmits(['use']);

const createDialog = ref({
    visible: false,
});
const createFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('createFormRef');
const loading = ref(false);
const createLoading = ref(false);
const createForm = ref({
    name: '',
    timezone: '',
});

/** 建库表单声明 */
const createItems: AutoFormItem[] = [
    { prop: 'name', label: 'milvus.databaseName', required: true, placeholder: 'milvus.databaseNamePlaceholder' },
    { prop: 'timezone', label: 'milvus.timezone', type: 'select', required: true, options: timezones, props: { filterable: true, clearable: true }, placeholder: 'milvus.timezonePlaceholder' },
];

const configDialog = ref({
    visible: false,
    currentDb: '',
});
const configFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('configFormRef');
const configLoading = ref(false);
const configForm = ref({
    timezone: '',
});

/** 库属性表单声明 */
const configItems: AutoFormItem[] = [
    { prop: 'timezone', label: 'milvus.timezone', type: 'select', required: true, options: timezones, props: { filterable: true }, placeholder: 'milvus.timezonePlaceholder' },
];

const loadList = async () => {
    loading.value = true;
    try {
        if (props.milvusId == 0) {
            return;
        }
        await milvusStore.refreshDbs(props.milvusId);
    } finally {
        loading.value = false;
    }
};

const handleCreate = () => {
    createForm.value = { name: '', timezone: 'Asia/Shanghai' };
    createDialog.value.visible = true;
};

const submitCreate = async () => {
    if (!createFormRef.value) return;

    await createFormRef.value?.validate();

    createLoading.value = true;
    try {
        // 构建 properties 对象
        const properties: Record<string, string> = {};
        if (configForm.value.timezone) {
            properties.timezone = configForm.value.timezone;
        }
        await milvusApi.createDatabase(props.milvusId, { ...createForm.value, properties });
        Msg.success('milvus.createdSuccess');
        createDialog.value.visible = false;
        await loadList();
    } finally {
        createLoading.value = false;
    }
};

const handleDrop = async (row: IDatabase) => {
    await useI18nConfirm('milvus.confirmDeleteDatabase', { name: row.name });
    await milvusApi.dropDatabase(props.milvusId, row.name);
    Msg.success('milvus.deletedSuccess');
    await loadList();
};

const handleConfig = async (row: IDatabase) => {
    configDialog.value.currentDb = row.name;
    // 获取当前数据库的配置
    const res = await milvusApi.describeDatabase(props.milvusId, row.name);
    // 解析 properties，获取 timezone
    configForm.value.timezone = res?.Properties?.timezone || '';
    configDialog.value.visible = true;
};

const submitConfig = async () => {
    configLoading.value = true;
    try {
        // 构建 properties 对象
        const properties: Record<string, string> = {};
        if (configForm.value.timezone) {
            properties.timezone = configForm.value.timezone;
        }
        await milvusApi.alterDatabase(props.milvusId, configDialog.value.currentDb, { properties, name: configDialog.value.currentDb });
        Msg.success('milvus.savedSuccess');
        configDialog.value.visible = false;
        await loadList();
    } finally {
        configLoading.value = false;
    }
};

const handleDetail = (_row: Record<string, unknown>) => {};

const handleUse = async (row: IDatabase) => {
    milvusStore.setSelectedDb(row.name);
    emits('use', row.name);
    milvusApi.useDatabase(props.milvusId, row.name);
};

watch(
    [() => props.milvusId, () => milvusStore.authCertName],
    () => {
        milvusStore.setDbs([]);
        loadList();
        milvusStore.clear();
    }
);
onMounted(loadList);
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
