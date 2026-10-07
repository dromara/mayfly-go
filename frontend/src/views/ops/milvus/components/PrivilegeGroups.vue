<template>
    <div class="component-container">
        <!-- 工具栏 -->
        <div class="mb5" style="display: flex; align-items: center">
            <el-button size="small" type="primary" icon="plus" @click="handleCreate">{{ $t('milvus.addPrivilegeGroup') }}</el-button>
            <el-button size="small" icon="edit" :disabled="!canEditSelected" @click="handleEdit">{{ $t('milvus.editPrivilegeGroup') }}</el-button>
            <el-button size="small" type="danger" icon="delete" :disabled="!canDeleteSelected" @click="handleDelete" plain>{{
                $t('milvus.deletePrivilegeGroup')
            }}</el-button>
        </div>

        <!-- 权限组表格 -->
        <el-table :data="privilegeGroups" @selection-change="handleSelectionChange" border stripe>
            <el-table-column type="selection" width="45" />
            <el-table-column :label="$t('milvus.privilegeGroupName')" prop="GroupName" width="200" />
            <el-table-column :label="$t('milvus.privileges')">
                <template #default="{ row }">
                    <el-tag v-for="p in row.Privileges" :key="p" type="success" class="mr5 mb5" effect="dark" round>
                        {{ p }}
                    </el-tag>
                </template>
            </el-table-column>
        </el-table>

        <!-- 编辑弹窗 -->
        <PrivilegeGroupEdit v-model:visible="editDialogVisible" :milvus-id="milvusId" :privilege-group="editingGroup" @saved="loadList" />
    </div>
</template>

<script setup lang="ts">
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { milvusApi } from '../api';
import { IPrivilegeGroup } from '../types';
import PrivilegeGroupEdit from './PrivilegeGroupEdit.vue';

const { t } = useI18n();

const props = defineProps<{
    milvusId: number;
}>();

const BUILTIN_PRIVILEGE_GROUPS = [
    'ClusterReadOnly',
    'ClusterReadWrite',
    'ClusterAdmin',
    'DatabaseReadOnly',
    'DatabaseReadWrite',
    'DatabaseAdmin',
    'CollectionReadOnly',
    'CollectionReadWrite',
    'CollectionAdmin',
];

const privilegeGroups = ref<IPrivilegeGroup[]>([]);
const selectedGroups = ref<IPrivilegeGroup[]>([]);
const editDialogVisible = ref(false);
const editingGroup = ref<IPrivilegeGroup | null>(null);

const isBuiltinGroup = (name: string) => {
    return BUILTIN_PRIVILEGE_GROUPS.includes(name);
};

const canEditSelected = computed(() => {
    return selectedGroups.value.length === 1 && !isBuiltinGroup(selectedGroups.value[0].GroupName);
});

const canDeleteSelected = computed(() => {
    return selectedGroups.value.length > 0 && selectedGroups.value.every((g) => !isBuiltinGroup(g.GroupName));
});

const loadList = async () => {
    const res = await milvusApi.getPrivilegeGroups(props.milvusId);
    privilegeGroups.value = res || [];
};

const handleSelectionChange = (selection: IPrivilegeGroup[]) => {
    selectedGroups.value = selection;
};

const handleCreate = () => {
    editingGroup.value = null;
    editDialogVisible.value = true;
};

const handleEdit = () => {
    if (selectedGroups.value.length === 1) {
        editingGroup.value = selectedGroups.value[0];
        editDialogVisible.value = true;
    }
};

const handleDelete = async () => {
    const names = selectedGroups.value.map((g) => g.GroupName).join(', ');
    if (!(await useI18nConfirm('milvus.confirmDeletePrivilegeGroup', { name: names }))) {
        return;
    }
    for (const group of selectedGroups.value) {
        await milvusApi.dropPrivilegeGroup(props.milvusId, group.GroupName);
    }
    Msg.success('milvus.privilegeGroupDeleteSuccess');
    await loadList();
};

onMounted(() => {
    loadList();
});

watch(
    () => props.milvusId,
    () => {
        privilegeGroups.value = [];
        loadList();
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
