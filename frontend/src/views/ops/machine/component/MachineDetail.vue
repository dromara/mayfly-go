<template>
    <div>
        <el-popover placement="right" width="auto" title="机器详情" trigger="click">
            <template #reference>
                <el-link @click="getMachineDetail" type="primary">{{ props.code }}</el-link>
            </template>

            <el-descriptions v-loading="state.loading" :column="3" border>
                <el-descriptions-item :span="1" label="机器id">{{ state.machineDetail.id }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="编号">{{ state.machineDetail.code }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="名称">{{ state.machineDetail.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="关联标签"><ResourceTags :tags="state.machineDetail.tags" /></el-descriptions-item>

                <el-descriptions-item :span="2" label="IP">{{ state.machineDetail.ip }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="端口">{{ state.machineDetail.port }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="备注">{{ state.machineDetail.remark }}</el-descriptions-item>

                <el-descriptions-item :span="1.5" label="SSH隧道">{{ state.machineDetail.sshTunnelMachineId > 0 ? '是' : '否' }} </el-descriptions-item>
                <el-descriptions-item :span="1.5" label="终端回放">{{ state.machineDetail.enableRecorder == 1 ? '是' : '否' }} </el-descriptions-item>

                <el-descriptions-item :span="2" label="创建时间">{{ formatDate(state.machineDetail.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="创建者">{{ state.machineDetail.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="更新时间">{{ formatDate(state.machineDetail.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="修改者">{{ state.machineDetail.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-popover>
    </div>
</template>

<script lang="ts" setup>
import { reactive } from 'vue';
import { machineApi } from '../api';
import { formatDate } from '@/common/utils/format';
import ResourceTags from '../../component/ResourceTags.vue';

const props = defineProps({
    code: {
        type: [String],
        requierd: true,
    },
});

const state = reactive({
    loading: false,
    machineDetail: {} as any,
});

const getMachineDetail = async () => {
    try {
        state.machineDetail = {};
        state.loading = true;
        const res = await machineApi.list.request({
            code: props.code,
        });
        if (res.total == 0) {
            return;
        }

        state.machineDetail = res.list?.[0];
    } finally {
        state.loading = false;
    }
};
</script>

<style></style>
