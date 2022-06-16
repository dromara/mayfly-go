<template>
    <div>
        <el-dialog :title="'分配“' + role.name + '”菜单&权限'" v-model="dialogVisible" :before-close="cancel" :show-close="false" width="400px">
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
                        <span v-if="data.type == enums.ResourceTypeEnum.MENU.value">{{ node.label }}</span>
                        <span v-if="data.type == enums.ResourceTypeEnum.PERMISSION.value" style="color: #67c23a">{{ node.label }}</span>
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

<script lang="ts">
import { toRefs, reactive, watch, defineComponent, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { roleApi } from '../api';
import enums from '../enums';

export default defineComponent({
    name: 'ResourceEdit',
    props: {
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
    },
    setup(props: any, { emit }) {
        const menuTree: any = ref(null);

        const state = reactive({
            dialogVisible: false,
            defaultProps: {
                children: 'children',
                label: 'name',
            },
        });

        watch(
            () => props.visible,
            (newValue) => {
                state.dialogVisible = newValue;
            }
        );

        /**
         * 获取所有菜单树的叶子节点
         * @param {Object} trees  菜单树列表
         */
        const getAllLeafIds = (trees: any) => {
            let leafIds: any = [];
            for (let tree of trees) {
                setLeafIds(tree, leafIds);
            }
            return leafIds;
        };

        const setLeafIds = (tree: any, ids: any) => {
            if (tree.children !== null) {
                for (let t of tree.children) {
                    setLeafIds(t, ids);
                }
            } else {
                ids.push(tree.id);
            }
        };

        const btnOk = async () => {
            let menuIds = menuTree.value.getCheckedKeys();
            let halfMenuIds = menuTree.value.getHalfCheckedKeys();
            let resources = [].concat(menuIds, halfMenuIds).join(',');
            await roleApi.saveResources.request({
                id: props.role['id'],
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

        return {
            ...toRefs(state),
            enums,
            menuTree,
            btnOk,
            getAllLeafIds,
            cancel,
        };
    },
});
</script>

<style>
</style>
