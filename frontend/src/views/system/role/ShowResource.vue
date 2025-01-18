<template>
    <div>
        <el-dialog @close="closeDialog" :title="title" :before-close="closeDialog" v-model="dialogVisible" width="400px">
            <el-tree style="height: 50vh; overflow: auto" :data="resources" node-key="id" :props="defaultProps">
                <template #default="{ node, data }">
                    <span class="custom-tree-node">
                        <SvgIcon :name="getMenuIcon(data)" class="mb3 mr3" />

                        <span v-if="data.type == ResourceTypeEnum.Menu.value">{{ $t(node.label) }}</span>
                        <span v-if="data.type == ResourceTypeEnum.Permission.value" style="color: #67c23a">{{ $t(node.label) }}</span>

                        <el-popover :show-after="500" placement="right-start" :title="$t('system.role.permissionInfo')" trigger="hover" :width="300">
                            <template #reference>
                                <el-link style="margin-left: 25px" icon="InfoFilled" type="info" :underline="false" />
                            </template>
                            <template #default>
                                <el-descriptions :column="1" size="small">
                                    <el-descriptions-item :label="$t('common.name')">
                                        {{ $t(data.name) }}
                                    </el-descriptions-item>
                                    <el-descriptions-item :label="$t('system.role.assigner')">
                                        {{ data.creator }}
                                    </el-descriptions-item>
                                    <el-descriptions-item :label="$t('system.role.allocateTime')">
                                        {{ formatDate(data.createTime) }}
                                    </el-descriptions-item>
                                </el-descriptions>
                            </template>
                        </el-popover>
                    </span>
                </template>
            </el-tree>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch } from 'vue';
import { ResourceTypeEnum } from '../enums';
import { formatDate } from '@/common/utils/format';
import { getMenuIcon } from '../resource/index';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    resources: {
        type: Array,
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'update:resources']);

const defaultProps = {
    children: 'children',
    label: 'name',
};

const state = reactive({
    dialogVisible: false,
});
const { dialogVisible } = toRefs(state);

watch(
    () => props.visible,
    (newValue) => {
        state.dialogVisible = newValue;
    }
);

const closeDialog = () => {
    emit('update:visible', false);
    emit('update:resources', []);
};
</script>

<style></style>
