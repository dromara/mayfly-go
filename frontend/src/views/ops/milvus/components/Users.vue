<template>
    <div class="component-container">
        <div>
            <el-button size="small" icon="plus" type="primary" @click="handleCreate">{{ $t('milvus.createUser') }}</el-button>
            <el-button size="small" text icon="refresh" @click="loadList" :loading="loading" />
        </div>

        <el-table :data="list">
        <el-table-column prop="name" :label="$t('common.username')" />
        <el-table-column :label="$t('common.operation')" width="350">
            <template #default="{ row }">
                <el-button size="small" @click="handleEditRoles(row)">{{ $t('milvus.editRole') }}</el-button>
                <el-button size="small" @click="handleChangePassword(row)">{{ $t('login.changePassword') }}</el-button>
                <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
            </template>
        </el-table-column>
        </el-table>
    </div>

    <el-dialog v-model="createDialog.visible" :title="$t('milvus.createUser')" width="500px">
        <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="auto">
            <el-form-item :label="$t('common.username')" prop="username">
                <el-input v-model="createForm.username" :placeholder="$t('common.username')"></el-input>
            </el-form-item>
            <el-form-item :label="$t('common.password')" prop="password">
                <el-input
                    type="password"
                    :minlength="6"
                    :maxlength="72"
                    show-password
                    v-model="createForm.password"
                    :placeholder="$t('common.password')"
                ></el-input>
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="createDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitCreate" :loading="createLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>

    <el-dialog v-model="passwordDialog.visible" :title="$t('milvus.changePassword')" width="500px">
        <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="auto">
            <el-form-item :label="$t('login.oldPassword')" prop="oldPassword">
                <el-input type="password" show-password v-model="passwordForm.oldPassword"></el-input>
            </el-form-item>
            <el-form-item :label="$t('login.newPassword')" prop="newPassword">
                <el-input type="password" show-password v-model="passwordForm.newPassword"></el-input>
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="passwordDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitPassword" :loading="passwordLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>

    <el-dialog v-model="roleDialog.visible" :title="$t('milvus.editRole')" width="700px">
        <div style="margin-bottom: 10px">
            <div style="font-weight: bold; margin-bottom: 8px">{{ $t('milvus.selectedRoles') }}</div>
            <el-checkbox-group v-model="roleDialog.selectedRoles">
                <el-checkbox v-for="role in allRoles" :key="role" :value="role" :label="role" />
            </el-checkbox-group>
        </div>
        <template #footer>
            <el-button @click="roleDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitRoles" :loading="roleLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import { Rules } from '@/common/rule';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { FormInstance } from 'element-plus';
import { onMounted, ref, watch } from 'vue';
import { milvusApi } from '../api';
import type { IUser } from '../types';

const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');

const list = ref<IUser[]>([]);
const createDialog = ref({
    visible: false,
});
const createFormRef = ref<FormInstance>();
const loading = ref(false);
const createLoading = ref(false);
const createForm = ref({
    username: '',
    password: '',
});

const createRules = {
    username: [Rules.requiredInput('common.username')],
    password: [Rules.requiredInput('common.password')],
};

const passwordDialog = ref({
    visible: false,
    currentUser: '',
});
const passwordFormRef = ref<FormInstance>();
const passwordLoading = ref(false);
const passwordForm = ref({
    oldPassword: '',
    newPassword: '',
});

const passwordRules = {
    oldPassword: [Rules.requiredInput('login.oldPassword')],
    newPassword: [Rules.requiredInput('login.newPassword')],
};

const roleDialog = ref({
    visible: false,
    currentUser: '',
    originalRoles: [] as string[],
    selectedRoles: [] as string[],
});
const roleLoading = ref(false);
const allRoles = ref<string[]>([]);

const loadList = async () => {
    loading.value = true;
    try {
        const res = await milvusApi.listUsers(props.milvusId);
        list.value = res || [];
        // 根据name排序
        list.value.sort((a, b) => a.name.localeCompare(b.name));

        // 加载所有角色列表
        const roles = await milvusApi.listRoles(props.milvusId);
        allRoles.value = roles || [];
    } finally {
        loading.value = false;
    }
};

const handleCreate = () => {
    createForm.value = { username: '', password: '' };
    createDialog.value.visible = true;
};

const submitCreate = async () => {
    if (!createFormRef.value) return;

    await createFormRef.value?.validate(async (valid) => {
        if (!valid) return;

        createLoading.value = true;
        try {
            await milvusApi.createUser(props.milvusId, createForm.value);
            Msg.success('milvus.createdSuccess');
            createDialog.value.visible = false;
            await loadList();
        } finally {
            createLoading.value = false;
        }
    });
};

const handleChangePassword = (row: IUser) => {
    passwordForm.value = { oldPassword: '', newPassword: '' };
    passwordDialog.value.currentUser = row.name;
    passwordDialog.value.visible = true;
};

const submitPassword = async () => {
    if (!passwordFormRef.value) return;

    await passwordFormRef.value?.validate(async (valid) => {
        if (!valid) return;

        passwordLoading.value = true;
        try {
            await milvusApi.updatePassword(props.milvusId, passwordDialog.value.currentUser, passwordForm.value);
            Msg.success('milvus.savedSuccess');
            passwordDialog.value.visible = false;
        } catch (error: unknown) {
            passwordLoading.value = false;
        }
    });
};

const handleDelete = async (row: IUser) => {
    await useI18nConfirm('milvus.confirmDeleteUser', { name: row.name });
    await milvusApi.deleteUser(props.milvusId, row.name);
    Msg.success('milvus.deletedSuccess');
    await loadList();
};

const handleEditRoles = (row: IUser) => {
    roleDialog.value.currentUser = row.name;
    roleDialog.value.originalRoles = [...(row.roles || [])];
    roleDialog.value.selectedRoles = [...(row.roles || [])];
    roleDialog.value.visible = true;
};

const submitRoles = async () => {
    roleLoading.value = true;
    try {
        const { selectedRoles, originalRoles, currentUser } = roleDialog.value;

        // 找出需要新增的角色（在selectedRoles中但不在originalRoles中）
        const rolesToAdd = selectedRoles.filter((role) => !originalRoles.includes(role));

        // 找出需要移除的角色（在originalRoles中但不在selectedRoles中）
        const rolesToRemove = originalRoles.filter((role) => !selectedRoles.includes(role));

        // 执行授权操作
        for (const role of rolesToAdd) {
            await milvusApi.grantRole(props.milvusId, currentUser, role);
        }

        // 执行撤销操作
        for (const role of rolesToRemove) {
            await milvusApi.revokeRole(props.milvusId, currentUser, role);
        }

        Msg.success('milvus.savedSuccess');
        roleDialog.value.visible = false;
        await loadList();
    } finally {
        roleLoading.value = false;
    }
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
