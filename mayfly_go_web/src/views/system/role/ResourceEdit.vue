<template>
    <div>
        <el-dialog :title="'分配“' + roleInfo?.name + '”菜单&权限'" v-model="dialogVisible" :before-close="cancel" :show-close="false" width="400px">
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
                        <span v-if="data.type == ResourceTypeEnum.Menu.value">{{ node.label }}</span>
                        <span v-if="data.type == ResourceTypeEnum.Permission.value" style="color: #67c23a">{{ node.label }}</span>
                    </span>
                </template>
            </el-tree>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel">取 消</el-button>
                    <el-button type="primary" @click="btnOk">确 定</el-button>
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

const props = defineProps({
    visible: {
        type: Boolean,
    },
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

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const defaultProps = {
    children: 'children',
    label: 'name',
};

const menuTree: any = ref(null);

const state = reactive({
    dialogVisible: false,
    roleInfo: null as any,
});

const { dialogVisible, roleInfo } = toRefs(state);

watch(
    () => props.visible,
    (newValue) => {
        state.dialogVisible = newValue;
        state.roleInfo = props.role;
    }
);

const btnOk = async () => {
    let menuIds = menuTree.value.getCheckedKeys();
    let halfMenuIds = menuTree.value.getHalfCheckedKeys();
    let resources = [].concat(menuIds, halfMenuIds).join(',');
    await roleApi.saveResources.request({
        id: props.role!.id,
        resourceIds: resources,
    });
    ElMessage.success('保存成功!');
    emit('cancel');
};

const cancel = () => {
    // 更新父组件visible prop对应的值为false
    emit('update:visible', false);
    emit('cancel');
};
</script>

<style></style>
