<template>
    <el-dialog v-model="visible" :title="$t('machine.batchFile')" width="720px" :close-on-click-modal="false">
        <!-- 已选机器 -->
        <div class="mb-2">
            <div class="mb-1 text-xs text-gray-400">{{ $t('machine.batchSelectedMachines') }}({{ machines.length }})</div>
            <el-tag v-for="m in machines" :key="m.id" class="ml-0.5" size="small">{{ m.name }}</el-tag>
        </div>

        <div class="mb-2 flex items-center gap-2">
            <el-upload action="#" :auto-upload="false" :limit="1" :show-file-list="false" :on-change="onFileChange" :disabled="running">
                <el-button icon="upload" plain :disabled="running">{{ $t('machine.selectFile') }}</el-button>
            </el-upload>
            <span class="text-sm">{{ file ? file.name : $t('machine.noFileSelected') }}</span>
        </div>

        <el-input v-model="remotePath" :disabled="running" :placeholder="$t('machine.remotePathPlaceholder')" class="mb-1" />
        <div class="mt-1 text-xs text-gray-400">{{ $t('machine.batchFileTips') }}</div>

        <div class="mt-2">
            <el-button type="primary" :disabled="!file || !remotePath.trim()" :loading="running" @click="onDispatch">
                {{ $t('machine.dispatchFile') }}
            </el-button>
        </div>

        <template v-if="results.length">
            <el-divider content-position="left">{{ $t('machine.execResult') }}</el-divider>
            <el-table :data="results" max-height="360" size="small" stripe>
                <el-table-column prop="name" :label="$t('common.name')" min-width="110" show-overflow-tooltip />
                <el-table-column :label="$t('machine.ipAndPort')" min-width="130" show-overflow-tooltip>
                    <template #default="{ row }">{{ `${row.ip}:${row.port}` }}</template>
                </el-table-column>
                <el-table-column :label="$t('common.status')" width="80" align="center">
                    <template #default="{ row }">
                        <el-tag v-if="row.success" type="success" effect="plain">{{ $t('machine.batchSuccess') }}</el-tag>
                        <el-tag v-else type="danger" effect="plain">{{ $t('machine.batchFail') }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column :label="$t('machine.size')" width="100">
                    <template #default="{ row }">{{ row.success ? formatByteSize(row.bytes) : '-' }}</template>
                </el-table-column>
                <el-table-column :label="$t('machine.execTime')" width="85">
                    <template #default="{ row }">{{ `${row.costMs}ms` }}</template>
                </el-table-column>
                <el-table-column :label="$t('machine.errorReason')" min-width="160" show-overflow-tooltip>
                    <template #default="{ row }">{{ row.error || '-' }}</template>
                </el-table-column>
            </el-table>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import { Msg } from '@/hooks/useI18n';
import { batchFileApi, uploadFileToStore } from '../api';
import type { BatchFileResult, MachineVO } from '../types';
import type { UploadFile } from 'element-plus';

const props = defineProps({
    /** 已选中的机器列表（打开对话框时由父组件传入） */
    machines: { type: Array as () => MachineVO[], default: () => [] },
});

const visible = defineModel<boolean>('visible', { default: false });

const file = ref<File | null>(null);
const remotePath = ref('');
const running = ref(false);
const results = ref<BatchFileResult[]>([]);

// 每次打开重置上次结果与所选文件（选中机器可能已变化）
watch(visible, (val) => {
    if (val) {
        results.value = [];
        file.value = null;
    }
});

const onFileChange = (uploadFile: UploadFile) => {
    file.value = uploadFile.raw ?? null;
};

const onDispatch = async () => {
    if (!file.value || !remotePath.value.trim()) return;
    running.value = true;
    try {
        // 先上传到平台文件服务得到 fileKey，再逐台分发（reader 单次消费，必须落库中转）
        const fileKey = await uploadFileToStore(file.value);
        results.value = await batchFileApi.dispatch.request({
            machineIds: props.machines.map((m) => m.id),
            fileKey,
            remotePath: remotePath.value,
        });
    } catch (e) {
        Msg.error((e as Error).message);
    } finally {
        running.value = false;
    }
};
</script>
