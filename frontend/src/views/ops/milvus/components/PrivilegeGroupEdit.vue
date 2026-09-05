<template>
    <el-dialog
        :title="isEdit ? $t('milvus.editPrivilegeGroup') : $t('milvus.addPrivilegeGroup')"
        v-model="visible"
        :close-on-click-modal="false"
        width="750px"
    >
        <!-- 权限组名称 -->
        <auto-form v-model="form" :items="formItems" label-width="auto" />

        <!-- 权限选择区域 -->
        <div class="privilege-select-area">
            <div v-for="group in PRIVILEGE_GROUPS" :key="group.label" class="privilege-group-section">
                <div class="privilege-group-header">
                    <el-checkbox v-model="group.checkAll" :indeterminate="group.indeterminate" @change="handleCheckAllChange(group)">
                        {{ group.label }}
                    </el-checkbox>
                </div>
                <el-checkbox-group v-model="form.privileges" @change="handlePrivilegeChange(group)">
                    <el-checkbox v-for="p in group.privileges" :key="p" :label="p" :value="p">
                        {{ p }}
                    </el-checkbox>
                </el-checkbox-group>
            </div>
        </div>

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import type { AutoFormItem } from '@/components/auto-form';
import type { PropType } from 'vue';
import { computed, reactive, ref, watch } from 'vue';
import { milvusApi } from '../api';
import type { IPrivilegeGroup } from '../types';

const props = defineProps({
    milvusId: { type: Number, required: true },
    privilegeGroup: { type: Object as PropType<IPrivilegeGroup | null>, default: null },
});

const emit = defineEmits(['saved']);

const visible = defineModel<boolean>('visible', { default: false });

const isEdit = computed(() => props.privilegeGroup !== null);

/** 权限组表单声明（编辑模式禁用 groupName） */
const formItems: AutoFormItem[] = [
    { prop: 'groupName', label: 'milvus.privilegeGroupName', required: true, disabled: () => isEdit.value, placeholder: 'milvus.privilegeGroupName' },
];

const form = reactive({
    groupName: '',
    privileges: [] as string[],
});

const saving = ref(false);

interface PrivilegeGroupItem {
    label: string;
    privileges: string[];
    checkAll: boolean;
    indeterminate: boolean;
}

const PRIVILEGE_GROUPS = reactive<PrivilegeGroupItem[]>([
    {
        label: 'Collection 相关权限',
        privileges: [
            'CreateCollection',
            'DescribeCollection',
            'ShowCollections',
            'DropCollection',
            'RenameCollection',
            'CreateAlias',
            'DescribeAlias',
            'DropAlias',
            'ListAliases',
            'Load',
            'GetLoadingProgress',
            'GetLoadState',
            'Release',
            'Flush',
            'GetFlushState',
            'GetStatistics',
            'Compaction',
            'FlushAll',
        ],
        checkAll: false,
        indeterminate: false,
    },
    {
        label: 'Partition 相关权限',
        privileges: ['CreatePartition', 'DropPartition', 'ShowPartitions', 'HasPartition'],
        checkAll: false,
        indeterminate: false,
    },
    {
        label: 'Index 相关权限',
        privileges: ['CreateIndex', 'DropIndex', 'IndexDetail'],
        checkAll: false,
        indeterminate: false,
    },
    {
        label: 'Entity 相关权限',
        privileges: ['Query', 'Insert', 'Upsert', 'Delete', 'Search', 'Import'],
        checkAll: false,
        indeterminate: false,
    },
    {
        label: 'User 相关权限',
        privileges: ['CreateUser', 'DropUser', 'UpdateUser', 'SelectUser'],
        checkAll: false,
        indeterminate: false,
    },
    {
        label: 'Database 相关权限',
        privileges: ['CreateDatabase', 'DropDatabase', 'ListDatabase', 'DescribeDatabase'],
        checkAll: false,
        indeterminate: false,
    },
    {
        label: 'ResourceGroup 相关权限',
        privileges: ['DescribeResourceGroup', 'ListResourceGroup', 'UpdateResourceGroups', 'CreateResourceGroup', 'DropResourceGroup'],
        checkAll: false,
        indeterminate: false,
    },
]);

const updateGroupState = (group: PrivilegeGroupItem) => {
    const checkedCount = group.privileges.filter((p) => form.privileges.includes(p)).length;
    group.checkAll = checkedCount === group.privileges.length;
    group.indeterminate = checkedCount > 0 && checkedCount < group.privileges.length;
};

const updateAllGroupStates = () => {
    PRIVILEGE_GROUPS.forEach((group) => updateGroupState(group));
};

const handleCheckAllChange = (group: PrivilegeGroupItem) => {
    if (group.checkAll) {
        // 全选：添加该分组中尚未选中的权限
        group.privileges.forEach((p) => {
            if (!form.privileges.includes(p)) {
                form.privileges.push(p);
            }
        });
    } else {
        // 取消全选：移除该分组中的所有权限
        form.privileges = form.privileges.filter((p) => !group.privileges.includes(p));
    }
    group.indeterminate = false;
};

const handlePrivilegeChange = (group: PrivilegeGroupItem) => {
    updateGroupState(group);
};

const handleSave = async () => {
    if (!form.groupName) {
        Msg.warning('milvus.privilegeGroupName');
        return;
    }
    saving.value = true;
    try {
        await milvusApi.savePrivilegeGroup(props.milvusId, {
            groupName: form.groupName,
            privileges: form.privileges,
        });
        Msg.success('milvus.privilegeGroupSaveSuccess');
        emit('saved');
        visible.value = false;
    } finally {
        saving.value = false;
    }
};

watch(visible, (val) => {
    if (val) {
        if (props.privilegeGroup) {
            // 编辑模式：回显数据
            form.groupName = props.privilegeGroup.GroupName;
            form.privileges = [...(props.privilegeGroup.Privileges || [])];
        } else {
            // 新建模式：清空表单
            form.groupName = '';
            form.privileges = [];
        }
        updateAllGroupStates();
    }
});
</script>

<style scoped>
.privilege-select-area {
    max-height: 500px;
    overflow-y: auto;
    padding: 10px;
}
.privilege-group-section {
    margin-bottom: 15px;
    padding: 10px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
}
.privilege-group-header {
    margin-bottom: 8px;
    font-weight: bold;
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding-bottom: 8px;
}
</style>
