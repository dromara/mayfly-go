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
                    <el-button type="primary" link size="small" @click="showRunLog(data)">
                        {{ $t('db.transferRunLog') }}
                    </el-button>
                </template>
            </page-table>

            <!-- 运行日志弹窗（使用通用 LogViewer） -->
            <el-dialog v-model="runLogVisible" :title="$t('db.transferRunLog')" width="900px" :destroy-on-close="true" @close="runLogUserClosed = true">
                <LogViewer
                    v-if="runLogLines.length > 0 || runLogLoading"
                    :lines="runLogLines"
                    :loading="runLogLoading"
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
import { DbTransferLogStatusEnum, DbTransferLogPurposeEnum } from '@/views/ops/db/transfer/enums';
import type { DbTransferLogListVO } from '@/views/ops/db/types';
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
    TableColumn.new('purpose', 'db.transferPurpose').alignCenter().typeTag(DbTransferLogPurposeEnum).setMinWidth(90),
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
const runLogLoading = ref(false);
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

const showRunLog = (data: DbTransferLogListVO) => {
    runLogVisible.value = true;
    runLogUserClosed.value = false; // 用户主动打开，重置关闭标记
    loadRunLog(data.id);
};

// 拉取单条执行日志的运行日志内容（列表接口不返回该大文本，避免日志较多时响应体膨胀），返回日志文本供调用方判断是否展示
const loadRunLog = async (logId: number): Promise<string> => {
    if (!logId) return '';
    runLogLoading.value = true;
    try {
        const res = await dbTransferApi.dbTransferTaskLogRun.request({ logId });
        currentRunLog.value = res?.runLog || '';
        return currentRunLog.value;
    } finally {
        runLogLoading.value = false;
    }
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
        // 实时模式下，获取最新一条执行日志的运行日志内容并自动更新展示
        if (state.realTime && state.query.taskId) {
            const res = (await dbTransferApi.dbTransferTaskLogs.request({
                taskId: state.query.taskId,
                pageNum: 1,
                pageSize: 1,
            })) as PageResult<DbTransferLogListVO>;
            if (res?.list?.length > 0) {
                const latestLog = res.list[0];
                // 用户未手动关闭过运行日志弹窗时，按需拉取最新日志的运行内容，有内容则自动弹出
                if (!runLogUserClosed.value && (await loadRunLog(latestLog.id))) {
                    if (!runLogVisible.value) {
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
