<template>
    <div class="db-transfer-logs">
        <el-dialog v-model="dialogVisible" :before-close="cancel" :destroy-on-close="false" width="1200px">
            <template #header>
                <span class="mr-2">{{ $t('db.log') }}</span>
                <el-switch v-model="realTime" @change="watchPolling" inline-prompt :active-text="$t('db.realTime')" :inactive-text="$t('db.noRealTime')" />
                <el-button @click="search" icon="Refresh" circle size="small" :loading="realTime" class="ml-2"></el-button>
            </template>

            <page-table ref="logTableRef" :page-api="dbTransferApi.dbTransferTaskLogs" v-model:query-form="query" :tool-button="false" :columns="columns" size="small">
                <template #durationMs="{ data }">
                    <span :class="{ 'text-red-500': data.durationMs > 30000 }">{{ data.durationMs ? `${data.durationMs} ms` : '-' }}</span>
                </template>
                <template #runLog="{ data }">
                    <el-button v-if="data.runLog" type="primary" link size="small" @click="showRunLog(data)">
                        {{ $t('db.transferRunLog') }}
                    </el-button>
                    <span v-else class="text-gray-400">-</span>
                </template>
            </page-table>

            <!-- 运行日志弹窗（使用通用 LogViewer） -->
            <el-dialog v-model="runLogVisible" :title="$t('db.transferRunLog')" width="900px" :destroy-on-close="true" @close="runLogUserClosed = true">
                <LogViewer
                    v-if="runLogLines.length > 0"
                    :lines="runLogLines"
                    :finished="!state.realTime"
                    :total-lines="runLogLines.length"
                    :theme="runLogTheme"
                    style="height: 500px"
                    @download="downloadRunLog"
                />
                <el-empty v-else :description="$t('db.transferRunLogEmpty')" />
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, Ref, ref, toRefs, watch, nextTick } from 'vue';

import PageTable from '@/components/page-table/PageTable.vue';
import { TableColumn } from '@/components/page-table';
import { LogViewer, SyncRunLogParser, type ParsedLogLine } from '@/components/log-viewer';
import { dbTransferApi } from '@/views/ops/db/transfer/api';
import { DbTransferLogStatusEnum } from '@/views/ops/db/transfer/enums';
import type { DbTransferLog } from '@/views/ops/db/types';
import type { PageResult } from '@/types/common';

const props = defineProps({
    taskId: {
        type: Number,
    },
    running: {
        type: Boolean,
        default: false,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const columns = ref([
    TableColumn.new('status', 'common.status').alignCenter().typeTag(DbTransferLogStatusEnum).setMinWidth(80),
    TableColumn.new('createTime', 'Time').alignCenter().isTime().setMinWidth(160),
    TableColumn.new('durationMs', 'db.transferDuration').alignCenter().isSlot().setMinWidth(100),
    TableColumn.new('totalRows', 'db.transferTotalRows').alignCenter().setMinWidth(100),
    TableColumn.new('tableCount', 'db.transferTableCount').alignCenter().setMinWidth(90),
    TableColumn.new('runLog', 'db.transferRunLog').alignCenter().isSlot().setMinWidth(100),
]);

// 运行日志解析器
const runLogParser = new SyncRunLogParser();

// 运行日志弹窗状态
const runLogVisible = ref(false);
const runLogUserClosed = ref(false); // 用户是否手动关闭过运行日志弹窗
const currentRunLog = ref('');
const runLogLines = computed<ParsedLogLine[]>(() => {
    if (!currentRunLog.value) return [];
    return runLogParser.parseAll(currentRunLog.value, 0);
});

// 运行日志主题
const runLogTheme = computed(() => ({
    showLineNumbers: false,
    showTimestamp: true,
    showLevelIcon: true,
}));

const showRunLog = (data: DbTransferLog) => {
    currentRunLog.value = data.runLog || '';
    runLogVisible.value = true;
    runLogUserClosed.value = false; // 用户主动打开，重置关闭标记
};

// 下载运行日志
const downloadRunLog = () => {
    if (!currentRunLog.value) return;
    const blob = new Blob([currentRunLog.value], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `transfer-run-log-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
};

watch(dialogVisible, (newValue: any) => {
    if (!newValue) {
        state.polling = false;
        watchPolling(false);
        return;
    }

    // 重新打开日志弹窗时重置用户关闭标记，允许自动弹出运行日志
    runLogUserClosed.value = false;
    runLogVisible.value = false;

    state.query.taskId = props.taskId!;
    search();
    state.realTime = props.running;
    watchPolling(props.running);
});

const startPolling = () => {
    if (!state.polling) {
        state.polling = true;
        state.pollingIndex = setInterval(search, 1000);
    }
};
const stopPolling = () => {
    if (state.polling) {
        state.polling = false;
        clearInterval(state.pollingIndex);
    }
};

const watchPolling = (polling: boolean) => {
    if (polling) {
        startPolling();
    } else {
        stopPolling();
    }
};

const logTableRef: Ref<any> = ref(null);

const search = async () => {
    try {
        logTableRef.value?.search();
        // 实时模式下，获取最新日志的运行日志内容并自动更新展示
        if (state.realTime && state.query.taskId) {
            const res = (await dbTransferApi.dbTransferTaskLogs.request({
                taskId: state.query.taskId,
                pageNum: 1,
                pageSize: 1,
            })) as PageResult<DbTransferLog>;
            if (res?.list?.length > 0) {
                const latestLog = res.list[0];
                if (latestLog.runLog) {
                    currentRunLog.value = latestLog.runLog;
                    // 仅在任务运行中且用户未手动关闭过运行日志弹窗时自动打开
                    if (!runLogVisible.value && !runLogUserClosed.value) {
                        nextTick(() => {
                            runLogVisible.value = true;
                        });
                    }
                }
                // 任务已完成（status 为成功或失败），停止实时模式并关闭运行日志弹窗
                if (latestLog.status === DbTransferLogStatusEnum.Success.value || latestLog.status === DbTransferLogStatusEnum.Fail.value) {
                    state.realTime = false;
                    watchPolling(false);
                    runLogVisible.value = false;
                    // 任务完成后刷新列表以显示最新状态和数据
                    logTableRef.value?.search();
                }
            }
        }
    } catch (e) {
        /* empty */
    }
};

const emit = defineEmits<{
    cancel: [];
}>();
const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
    watchPolling(false);
};

const state = reactive({
    polling: false,
    pollingIndex: 0 as any,
    realTime: props.running,
    query: {
        taskId: 0,
        pageNum: 1,
        pageSize: 0,
    },
});

const { query, realTime } = toRefs(state);
</script>
