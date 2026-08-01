<template>
    <div>
        <el-dialog
            :title="$t('system.role.allocateMenuTitle', { roleName: props.role?.name })"
            v-model="visible"
            :before-close="onCancel"
            :show-close="false"
            width="400px"
        >
            <el-tree
                style="height: 50vh; overflow: auto"
                ref="menuTree"
                :data="resources"
                show-checkbox
                node-key="id"
                :default-checked-keys="defaultCheckedKeys"
                :props="defaultProps"
            >
                <template #default="{ node, data }">
                    <span class="custom-tree-node">
                        <SvgIcon :name="getMenuIcon(data)" class="mb-0.5 mr-0.5" />
                        <span v-if="data.type == ResourceTypeEnum.Menu.value">{{ $t(node.label) }}</span>
                        <span v-if="data.type == ResourceTypeEnum.Permission.value" style="color: #67c23a">{{ $t(node.label) }}</span>
                    </span>
                </template>
            </el-tree>
            <template #footer>
                <el-button :loading="state.submiting" @click="onCancel">{{ $t('common.cancel') }}</el-button>
                <el-button :loading="state.submiting" type="primary" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg } from '@/hooks/useI18n';
import { reactive, ref, type PropType } from 'vue';
import { ElTree } from 'element-plus';
import { roleApi } from '../api';
import { ResourceTypeEnum } from '../enums';
import { getMenuIcon } from '../resource';
import type { SysResource, SysRole } from '../types';

const props = defineProps({
    title: {
        type: String,
    },
    role: {
        type: Object as () => SysRole | null,
    },
    // 默认勾选的节点
    defaultCheckedKeys: {
        type: Array as PropType<number[]>,
    },
    // 所有资源树
    resources: {
        type: Array as PropType<SysResource[]>,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const defaultProps = {
    children: 'children',
    label: 'name',
};

const menuTree = ref<InstanceType<typeof ElTree> | null>(null);

const state = reactive({
    submiting: false,
});

const onConfirm = async () => {
    const menuIds = menuTree.value?.getCheckedKeys() ?? [];
    const halfMenuIds = menuTree.value?.getHalfCheckedKeys() ?? [];
    const resources = [...menuIds, ...halfMenuIds].join(',');
    try {
        state.submiting = true;
        await roleApi.saveResources.request({
            id: props.role!.id,
            resourceIds: resources,
        });
        Msg.saveSuccess();
        emit('cancel');
    } finally {
        state.submiting = false;
    }
};

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>

<style></style>
