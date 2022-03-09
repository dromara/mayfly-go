<template>
    <div>
        <el-dialog @close="closeDialog" :title="title" :before-close="closeDialog" v-model="dialogVisible" width="400px">
            <el-tree style="height: 50vh; overflow: auto" :data="resources" node-key="id" :props="defaultProps">
                <template #default="{ node, data }">
                    <span class="custom-tree-node">
                        <span v-if="data.type == enums.ResourceTypeEnum.MENU.value">{{ node.label }}</span>
                        <span v-if="data.type == enums.ResourceTypeEnum.PERMISSION.value" style="color: #67c23a">{{ node.label }}</span>

                        <el-link @click.prevent="info(data)" style="margin-left: 25px" icon="el-icon-view" type="info" :underline="false" />
                    </span>
                </template>
            </el-tree>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { getCurrentInstance, toRefs, reactive, watch, defineComponent } from 'vue';
import { ElMessageBox } from 'element-plus';
import enums from '../enums';

export default defineComponent({
    name: 'ShowResource',
    props: {
        visible: {
            type: Boolean,
        },
        resources: {
            type: Array,
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const { proxy } = getCurrentInstance() as any;
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

        const info = (info: any) => {
            ElMessageBox.alert(
                '<strong style="margin-right: 18px">资源名称:</strong>' +
                    info.name +
                    ' <br/><strong style="margin-right: 18px">分配账号:</strong>' +
                    info.creator +
                    ' <br/><strong style="margin-right: 18px">分配时间:</strong>' +
                    proxy.$filters.dateFormat(info.createTime) +
                    '',
                '分配信息',
                {
                    type: 'info',
                    dangerouslyUseHTMLString: true,
                    closeOnClickModal: true,
                    showConfirmButton: false,
                }
            ).catch(() => {});
            return;
        };

        const closeDialog = () => {
            emit('update:visible', false);
            emit('update:resources', []);
        };

        return {
            ...toRefs(state),
            enums,
            info,
            closeDialog,
        };
    },
});
</script>

<style>
</style>
