<template>
    <div>
        <el-popover placement="right" width="auto" :title="$t('common.detail')" trigger="click">
            <template #reference>
                <el-link @click="getMachineDetail" type="primary">{{ props.code }}</el-link>
            </template>

            <el-descriptions v-loading="state.loading" :column="3" border>
                <el-descriptions-item :span="1" label="ID">{{ state.machineDetail.id }}</el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.code')">{{ state.machineDetail.code }}</el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.name')">{{ state.machineDetail.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><ResourceTags :tags="state.machineDetail.tags" /></el-descriptions-item>

                <el-descriptions-item :span="2" label="IP">{{ state.machineDetail.ip }}</el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('machine.port')">{{ state.machineDetail.port }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('common.remark')">{{ state.machineDetail.remark }}</el-descriptions-item>

                <el-descriptions-item :span="1.5" :label="$t('machine.sshTunnel')">
                    {{ state.machineDetail.sshTunnelMachineId > 0 ? $t('common.yes') : $t('common.no') }}
                </el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('machine.terminalPlayback')">
                    {{ state.machineDetail.enableRecorder == 1 ? $t('common.yes') : $t('common.no') }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.createTime')">{{ formatDate(state.machineDetail.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.creator')">{{ state.machineDetail.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.updateTime')">{{ formatDate(state.machineDetail.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.modifier')">{{ state.machineDetail.modifier }}</el-descriptions-item>
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
