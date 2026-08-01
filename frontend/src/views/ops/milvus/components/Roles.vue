<template>
    <div class="component-container">
        <div>
            <el-button size="small" icon="plus" type="primary" @click="handleCreate">{{ $t('milvus.createRole') }}</el-button>
            <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
        </div>

        <el-table :data="list">
        <el-table-column prop="roleName" :label="$t('milvus.roleName')" />
        <el-table-column :label="$t('common.operation')" width="250">
            <template #default="{ row }">
                <el-button size="small" @click="handleGrantPrivilege(row)" :disabled="row.roleName === 'public' || row.roleName === 'admin'">
                    {{ $t('milvus.grantPrivilege') }}
                </el-button>
                <el-button size="small" type="danger" @click="handleDrop(row)">{{ $t('common.delete') }}</el-button>
            </template>
        </el-table-column>
        </el-table>
    </div>

    <!-- 创建角色弹窗 -->
    <el-dialog v-model="createDialog.visible" :title="$t('milvus.createRole')" width="500px">
        <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="auto">
            <el-form-item :label="$t('milvus.roleName')" prop="roleName">
                <el-input v-model="createForm.roleName" :placeholder="$t('milvus.roleNamePlaceholder')"></el-input>
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="createDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitCreate" :loading="createLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>

    <!-- 授权弹窗 -->
    <RolesGrantPrivilege ref="grantPrivilegeRef" :milvus-id="milvusId" @privilege-saved="loadList" />
</template>

<script setup lang="ts">
import { Rules } from '@/common/rule';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { FormInstance } from 'element-plus';
import { onMounted, ref, watch } from 'vue';
import { milvusApi } from '../api';
import RolesGrantPrivilege from './RolesGrantPrivilege.vue';

const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');

const list = ref<{ roleName: string; privileges: unknown[] }[]>([]);
const createDialog = ref({
    visible: false,
});
const createFormRef = ref<FormInstance>();
const loading = ref(false);
const createLoading = ref(false);
const createForm = ref({
    roleName: '',
});

const createRules = {
    roleName: [Rules.requiredInput('milvus.roleName')],
};

const grantPrivilegeRef = ref<InstanceType<typeof RolesGrantPrivilege>>();

const loadList = async () => {
    loading.value = true;
    try {
        const res = await milvusApi.listRoles(props.milvusId);
        list.value = (res || []).map((a: string) => ({ roleName: a, privileges: [] }));
    } finally {
        loading.value = false;
    }
};

const handleCreate = () => {
    createForm.value = { roleName: '' };
    createDialog.value.visible = true;
};

const submitCreate = async () => {
    if (!createFormRef.value) return;

    await createFormRef.value?.validate(async (valid) => {
        if (!valid) return;

        createLoading.value = true;
        try {
            await milvusApi.createRole(props.milvusId, createForm.value);
            Msg.success('milvus.createdSuccess');
            createDialog.value.visible = false;
            loadList();
        } finally {
            createLoading.value = false;
        }
    });
};

const handleGrantPrivilege = async (row: { roleName: string }) => {
    grantPrivilegeRef.value?.handleGrantPrivilege(row);
};

const handleDrop = async (row: { roleName: string }) => {
    await useI18nConfirm('milvus.confirmDeleteRole', { name: row.roleName });
    await milvusApi.dropRole(props.milvusId, row.roleName);
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
