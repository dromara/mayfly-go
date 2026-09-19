<template>
    <div class="sync-task-logs">
        <el-dialog v-model="dialogVisible" :before-close="cancel" :destroy-on-close="false" width="1400px">
            <template #header>
                <span class="mr-2">{{ $t('db.log') }}</span>
                <el-switch v-model="realTime" @change="watchPolling" inline-prompt :active-text="$t('db.realTime')" :inactive-text="$t('db.noRealTime')" />
                <el-button @click="search" icon="Refresh" circle size="small" :loading="realTime" class="ml-2"></el-button>
            </template>

            <!-- Phase 4: 同步指标概览卡片 -->
            <div v-if="state.latestLog" class="sync-metrics-summary mb-3">
                <el-row :gutter="12">
                    <el-col :span="5">
                        <el-statistic :title="$t('db.syncDuration')" :value="state.latestLog.durationMs || 0">
                            <template #suffix>ms</template>
                        </el-statistic>
                    </el-col>
                    <el-col :span="5">
                        <el-statistic :title="$t('db.syncThroughput')" :value="state.latestLog.throughput || 0">
                            <template #suffix>rows/s</template>
                        </el-statistic>
                    </el-col>
                    <el-col :span="3.5">
                        <el-statistic :title="$t('db.syncInsertCount')" :value="state.latestLog.insertCount || 0"></el-statistic>
                    </el-col>
                    <el-col :span="3.5">
                        <el-statistic :title="$t('db.syncUpdateCount')" :value="state.latestLog.updateCount || 0"></el-statistic>
                    </el-col>
                    <el-col :span="3.5">
                        <el-statistic :title="$t('db.syncDeleteCount')" :value="state.latestLog.deleteCount || 0"></el-statistic>
                    </el-col>
                    <el-col :span="3.5">
                        <el-statistic :title="$t('db.syncSkipCount')" :value="state.latestLog.skipCount || 0"></el-statistic>
                    </el-col>
                </el-row>
            </div>

            <page-table ref="logTableRef" :page-api="dbSyncApi.datasyncLogs" v-model:query-form="query" :tool-button="false" :columns="columns" size="small">
                <!-- Phase 4: 指标列自定义渲染 -->
                <template #durationMs="{ data }">
                    <span :class="{ 'text-red-500': data.durationMs > 30000 }">{{ data.durationMs ? `${data.durationMs} ms` : '-' }}</span>
                </template>
                <template #throughput="{ data }">
                    <span :class="{ 'text-green-500': data.throughput > 1000 }">{{ data.throughput || '-' }}</span>
                </template>
                <template #runLog="{ data }">
                    <el-button v-if="data.runLog" type="primary" link size="small" @click="showRunLog(data)">
                        {{ $t('db.syncRunLog') }}
                    </el-button>
                    <span v-else class="text-gray-400">-</span>
                </template>
            </page-table>

            <!-- 运行日志弹窗（使用通用 LogViewer） -->
            <el-dialog v-model="runLogVisible" :title="$t('db.syncRunLog')" width="900px" :destroy-on-close="true">
                <LogViewer
                    v-if="runLogLines.length > 0"
                    :lines="runLogLines"
                    :finished="!state.realTime"
                    :total-lines="runLogLines.length"
                    :theme="runLogTheme"
                    style="height: 500px"
                    @download="downloadRunLog"
                />
                <el-empty v-else :description="$t('db.syncRunLogEmpty')" />
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, Ref, ref, toRefs, watch, nextTick } from 'vue';

import PageTable from '@/components/page-table/PageTable.vue';
import { TableColumn } from '@/components/page-table';
import { LogViewer, SyncRunLogParser, type ParsedLogLine } from '@/components/log-viewer';
import { dbSyncApi } from '@/views/ops/db/sync/api';
import { DbDataSyncLogStatusEnum } from '@/views/ops/db/sync/enums';
import type { DataSyncLog } from '@/views/ops/db/types';
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
    TableColumn.new('status', 'common.status').alignCenter().typeTag(DbDataSyncLogStatusEnum).setMinWidth(80),
    TableColumn.new('createTime', 'Time').alignCenter().isTime().setMinWidth(160),
    TableColumn.new('durationMs', 'db.syncDuration').alignCenter().isSlot().setMinWidth(100),
    TableColumn.new('throughput', 'db.syncThroughput').alignCenter().isSlot().setMinWidth(100),
    TableColumn.new('insertCount', 'db.syncInsertCount').alignCenter().setMinWidth(90),
    TableColumn.new('updateCount', 'db.syncUpdateCount').alignCenter().setMinWidth(90),
    TableColumn.new('deleteCount', 'db.syncDeleteCount').alignCenter().setMinWidth(90),
    TableColumn.new('skipCount', 'db.syncSkipCount').alignCenter().setMinWidth(90),
    TableColumn.new('runLog', 'db.syncRunLog').alignCenter().isSlot().setMinWidth(100),
]);

// 运行日志解析器
const runLogParser = new SyncRunLogParser();

// 运行日志弹窗状态
const runLogVisible = ref(false);
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

const showRunLog = (data: DataSyncLog) => {
    currentRunLog.value = data.runLog || '';
    runLogVisible.value = true;
};

// 下载运行日志
const downloadRunLog = () => {
    if (!currentRunLog.value) return;
    const blob = new Blob([currentRunLog.value], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `run-log-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
};

watch(dialogVisible, (newValue: any) => {
    if (!newValue) {
        state.polling = false;
        watchPolling(false);
        return;
    }

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
            const res = (await dbSyncApi.datasyncLogs.request({ taskId: state.query.taskId, pageNum: 1, pageSize: 1 })) as PageResult<DataSyncLog>;
            if (res?.list?.length > 0) {
                const latestLog = res.list[0];
                state.latestLog = latestLog;
                if (latestLog.runLog) {
                    currentRunLog.value = latestLog.runLog;
                    // 任务运行中时自动打开运行日志弹窗
                    if (!runLogVisible.value) {
                        nextTick(() => {
                            runLogVisible.value = true;
                        });
                    }
                }
                // 任务已完成（durationMs > 0 表示 endRunning 已执行），停止实时模式
                if (latestLog.durationMs > 0) {
                    state.realTime = false;
                    watchPolling(false);
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
    latestLog: null as DataSyncLog | null,
    query: {
        taskId: 0,
        name: null,
        pageNum: 1,
        pageSize: 0,
    },
});

const { query, realTime } = toRefs(state);
</script>
<style lang="scss">
.sync-metrics-summary {
    padding: 12px;
    background: var(--el-fill-color-lighter);
    border-radius: 8px;
}
</style>
