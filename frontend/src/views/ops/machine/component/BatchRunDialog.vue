<template>
    <el-dialog v-model="visible" :title="$t('machine.batchRunCmd')" width="720px" :close-on-click-modal="false">
        <!-- 已选机器 -->
        <div class="mb-2">
            <div class="mb-1 text-xs text-gray-400">{{ $t('machine.batchSelectedMachines') }}({{ machines.length }})</div>
            <el-tag v-for="m in machines" :key="m.id" class="ml-0.5" size="small">{{ m.name }}</el-tag>
        </div>

        <el-input v-model="cmd" type="textarea" :rows="4" :disabled="running" :placeholder="$t('machine.batchCmdPlaceholder')" />
        <div class="mt-1 text-xs text-gray-400">{{ $t('machine.batchCmdTips') }}</div>

        <div class="mt-2">
            <el-button type="primary" :disabled="!cmd.trim()" :loading="running" @click="onRun">{{ $t('machine.execute') }}</el-button>
        </div>

        <template v-if="results.length">
            <el-divider content-position="left">{{ $t('machine.execResult') }}</el-divider>
            <el-table :data="results" max-height="360" size="small" stripe>
                <el-table-column type="expand">
                    <template #default="{ row }">
                        <pre class="batch-run-output">{{ row.output || row.error }}</pre>
                    </template>
                </el-table-column>
                <el-table-column prop="name" :label="$t('common.name')" min-width="110" show-overflow-tooltip />
                <el-table-column :label="$t('machine.ipAndPort')" min-width="130" show-overflow-tooltip>
                    <template #default="{ row }">{{ `${row.ip}:${row.port}` }}</template>
                </el-table-column>
                <el-table-column :label="$t('common.status')" width="80" align="center">
                    <template #default="{ row }">
                        <el-tag v-if="row.success" type="success" effect="plain">{{ $t('machine.batchSuccess') }}</el-tag>
                        <el-tag v-else type="danger" effect="plain">{{ row.timeout ? $t('machine.batchTimeout') : $t('machine.batchFail') }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column :label="$t('machine.execTime')" width="85">
                    <template #default="{ row }">{{ `${row.costMs}ms` }}</template>
                </el-table-column>
                <el-table-column :label="$t('machine.policyNotice')" width="80" align="center">
                    <template #default="{ row }">
                        <el-tooltip v-if="row.policyNotice" :content="row.policyNotice" placement="top">
                            <el-tag type="warning" effect="plain">{{ $t('machine.policyNoticeHit') }}</el-tag>
                        </el-tooltip>
                        <span v-else>-</span>
                    </template>
                </el-table-column>
            </el-table>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import { batchExecApi } from '../api';
import type { BatchCmdResult, MachineVO } from '../types';

const props = defineProps({
    /** 已选中的机器列表（打开对话框时由父组件传入） */
    machines: { type: Array as () => MachineVO[], default: () => [] },
});

const visible = defineModel<boolean>('visible', { default: false });

const cmd = ref('');
const running = ref(false);
const results = ref<BatchCmdResult[]>([]);

// 每次打开重置上次结果（选中机器可能已变化），命令保留便于重复执行
watch(visible, (val) => {
    if (val) {
        results.value = [];
    }
});

const onRun = async () => {
    running.value = true;
    try {
        results.value = await batchExecApi.run.request({
            machineIds: props.machines.map((m) => m.id),
            cmd: cmd.value,
        });
    } finally {
        running.value = false;
    }
};
</script>

<style scoped>
.batch-run-output {
    margin: 0;
    padding: 8px 12px;
    font-family: var(--font-mono, ui-monospace, monospace);
    font-size: 12px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-all;
}
</style>
