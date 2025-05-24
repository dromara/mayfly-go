<template>
    <div>
        <el-dialog
            :title="$t('system.role.allocateMenuTitle', { roleName: roleInfo?.name })"
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
                <div class="dialog-footer">
                    <el-button :loading="state.submiting" @click="onCancel">{{ $t('common.cancel') }}</el-button>
                    <el-button :loading="state.submiting" type="primary" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { roleApi } from '../api';
import { ResourceTypeEnum } from '../enums';
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/svgIcon/index.vue';
import { getMenuIcon } from '../resource';

const { t } = useI18n();

const props = defineProps({
    title: {
        type: String,
    },
    role: {
        type: Object,
    },
    // 默认勾选的节点
    defaultCheckedKeys: {
        type: Array,
    },
    // 所有资源树
    resources: {
        type: Array,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const defaultProps = {
    children: 'children',
    label: 'name',
};

const menuTree: any = ref(null);

const state = reactive({
    roleInfo: null as any,
    submiting: false,
});

const { roleInfo } = toRefs(state);

watch(
    () => visible,
    (newValue) => {
        state.roleInfo = props.role;
    }
);

const onConfirm = async () => {
    let menuIds = menuTree.value.getCheckedKeys();
    let halfMenuIds = menuTree.value.getHalfCheckedKeys();
    let resources = [].concat(menuIds, halfMenuIds).join(',');
    try {
        state.submiting = true;
        await roleApi.saveResources.request({
            id: props.role!.id,
            resourceIds: resources,
        });
        ElMessage.success(t('common.saveSuccess'));
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
