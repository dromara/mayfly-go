<template>
    <el-dialog v-model="dialogVisible" :title="$t('common.detail')" width="600px" :destroy-on-close="true">
        <el-descriptions v-loading="loading" :column="3" border>
            <el-descriptions-item :span="1" label="ID">{{ machineDetail.id }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.code')">{{ machineDetail.code }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.name')">{{ machineDetail.name }}</el-descriptions-item>

            <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><TagCodePath :code="machineDetail.code" /></el-descriptions-item>

            <el-descriptions-item :span="2" label="IP">{{ machineDetail.ip }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('machine.port')">{{ machineDetail.port }}</el-descriptions-item>

            <el-descriptions-item :span="3" :label="$t('common.remark')">{{ machineDetail.remark }}</el-descriptions-item>

            <el-descriptions-item :span="1.5" :label="$t('machine.sshTunnel')">
                {{ machineDetail.sshTunnelMachineId > 0 ? $t('common.yes') : $t('common.no') }}
            </el-descriptions-item>
            <el-descriptions-item :span="1.5" :label="$t('machine.terminalPlayback')">
                {{ machineDetail.enableRecorder == 1 ? $t('common.yes') : $t('common.no') }}
            </el-descriptions-item>

            <el-descriptions-item :span="2" :label="$t('common.createTime')">{{ formatDate(machineDetail.createTime) }} </el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.creator')">{{ machineDetail.creator }}</el-descriptions-item>

            <el-descriptions-item :span="2" :label="$t('common.updateTime')">{{ formatDate(machineDetail.updateTime) }} </el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.modifier')">{{ machineDetail.modifier }}</el-descriptions-item>
        </el-descriptions>
    </el-dialog>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { ref, watch } from 'vue';
import { machineApi } from '../../api';
import type { MachineVO } from '../../types';

const props = defineProps({
    code: {
        type: String,
        required: true,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });
const loading = ref(false);
const machineDetail = ref<MachineVO>({} as MachineVO);

const getMachineDetail = async () => {
    try {
        machineDetail.value = {} as MachineVO;
        loading.value = true;
        const res = await machineApi.list.request({
            code: props.code,
        });
        if (res.total == 0) {
            return;
        }

        machineDetail.value = res.list?.[0] ?? ({} as MachineVO);
    } finally {
        loading.value = false;
    }
};

// 监听 visible 变化，打开时加载数据
watch(
    dialogVisible,
    (val: boolean) => {
        if (val) {
            getMachineDetail();
        }
    },
    { immediate: true }
);
</script>
