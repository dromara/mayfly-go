<template>
    <div class="component-container">
        <data-table :data="list" :pageable="false" :columns="columns" :card="false">
            <template #tableHeader>
                <el-button size="small" icon="plus" type="primary" @click="handleCreate">{{ $t('milvus.createRole') }}</el-button>
                <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
            </template>
        </data-table>
    </div>

    <!-- 创建角色弹窗 -->
    <el-dialog v-model="createDialog.visible" :title="$t('milvus.createRole')" width="500px">
        <auto-form ref="createFormRef" v-model="createForm" :items="createItems" label-width="auto" />
        <template #footer>
            <el-button @click="createDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitCreate" :loading="createLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>

    <!-- 授权弹窗 -->
    <RolesGrantPrivilege ref="grantPrivilegeRef" :milvus-id="milvusId" @privilege-saved="loadList" />
</template>

<script setup lang="ts">
import { Msg } from '@/hooks/useI18n';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import DataTable from '@/components/data-table/DataTable.vue';
import { TableColumn } from '@/components/data-table';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { onMounted, ref, useTemplateRef, watch } from 'vue';
import { milvusApi } from '../api';
import RolesGrantPrivilege from './RolesGrantPrivilege.vue';

const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');

/** 角色行（listRoles 返回字符串，映射成带 privileges 占位的行对象） */
interface RoleRow {
    roleName: string;
    privileges: unknown[];
}

const list = ref<RoleRow[]>([]);

// 声明式列：操作列由 setActions 统一渲染并同源算宽（不再手写 width=250），行类型绑 RoleRow
const columns = [
    TableColumn.new('roleName', 'milvus.roleName'),
    TableColumn.new('action', 'common.operation').setActions<RoleRow>([
        {
            key: 'grant',
            label: 'milvus.grantPrivilege',
            disabled: (row) => row.roleName === 'public' || row.roleName === 'admin',
            onClick: (row) => handleGrantPrivilege(row),
        },
        {
            key: 'delete',
            label: 'common.delete',
            type: 'danger',
            confirm: { title: 'milvus.confirmDeleteRole', params: (row) => ({ name: row.roleName }) },
            onClick: (row) => dropRole(row),
        },
    ], { size: 'small' }),
];
const createDialog = ref({
    visible: false,
});
const createFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('createFormRef');
const loading = ref(false);
const createLoading = ref(false);
const createForm = ref({
    roleName: '',
});

/** 建角色表单声明 */
const createItems: AutoFormItem[] = [{ prop: 'roleName', label: 'milvus.roleName', required: true, placeholder: 'milvus.roleNamePlaceholder' }];

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

    await createFormRef.value?.validate();

    createLoading.value = true;
    try {
        await milvusApi.createRole(props.milvusId, createForm.value);
        Msg.success('milvus.createdSuccess');
        createDialog.value.visible = false;
        loadList();
    } finally {
        createLoading.value = false;
    }
};

const handleGrantPrivilege = async (row: { roleName: string }) => {
    grantPrivilegeRef.value?.handleGrantPrivilege(row);
};

const dropRole = async (row: RoleRow) => {
    await milvusApi.dropRole(props.milvusId, row.roleName);
    Msg.success('milvus.deletedSuccess');
    await loadList();
};

onMounted(() => {
    loadList();
});

watch([() => props.milvusId, () => milvusStore.authCertName], () => {
    list.value = [];
    loadList();
    milvusStore.clear();
});
</script>

<style scoped>
.component-container {
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
}
</style>
