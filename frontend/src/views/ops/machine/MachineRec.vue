<template>
    <div id="terminalRecDialog">
        <!-- 终端操作记录列表 -->
        <el-dialog
            modal-penetrable
            :modal="false"
            draggable
            :title="title"
            v-model="visible"
            :before-close="handleClose"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            width="1000"
            @open="getTermOps()"
        >
            <page-table ref="pageTableRef" :page-api="machineApi.termOpRecs" :lazy="true" height="100%" v-model:query-form="query" :columns="columns">
                <template #fileKey="{ data }">
                    <FileInfo :fileKey="data.fileKey" show-file-size />
                </template>

                <template #action="{ data }">
                    <el-button @click="playRec(data)" loading-icon="loading" :loading="data.playRecLoding" type="primary" link>
                        {{ $t('machine.playback') }}
                    </el-button>
                    <el-button @click="showExecCmds(data)" type="primary" link>{{ $t('machine.cmd') }}</el-button>
                </template>
            </page-table>
        </el-dialog>

        <!-- 全屏终端回放 -->
        <Teleport to="body">
            <Transition name="rec-fade">
                <div v-if="playerVisible" class="rec-player-fullscreen">
                    <div class="rec-player__header">
                        <span class="rec-player__title">{{ title }}</span>
                        <el-button text @click="closePlayer">
                            <el-icon :size="18"><Close /></el-icon>
                        </el-button>
                    </div>
                    <div class="rec-player__body" ref="playerRef"></div>
                </div>
            </Transition>
        </Teleport>

        <!-- 执行命令记录弹窗 -->
        <el-dialog :title="$t('machine.execCmdRecord')" v-model="execCmdsDialogVisible" :destroy-on-close="true" width="500">
            <el-empty v-if="state.execCmds.length === 0" :description="$t('machine.noCmdRecord')" />
            <el-table v-else :data="state.execCmds" max-height="480" stripe size="small">
                <el-table-column prop="cmd" :label="$t('machine.cmd')" show-overflow-tooltip min-width="150px"> </el-table-column>
                <el-table-column prop="time" :label="$t('machine.execTime')" min-width="80" show-overflow-tooltip>
                    <template #default="scope">
                        {{ formatDate(new Date(scope.row.time * 1000).toString()) }}
                    </template>
                </el-table-column>
            </el-table>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, watch, ref, reactive, nextTick } from 'vue';
import { Close } from '@element-plus/icons-vue';
import { machineApi } from './api';
import * as AsciinemaPlayer from 'asciinema-player';
import 'asciinema-player/dist/bundle/asciinema-player.css';
import PageTable from '@/components/page-table/PageTable.vue';
import { TableColumn } from '@/components/page-table';
import { formatDate } from '@/common/utils/format';
import { getFileUrl } from '@/common/request';
import FileInfo from '@/components/file/FileInfo.vue';
import type { MachineTermOp } from './types';

const props = defineProps({
    machineId: { type: Number },
    title: { type: String },
});

const emit = defineEmits(['cancel']);

const visible = defineModel<boolean>('visible', { default: false });

const columns = [
    TableColumn.new('creator', 'machine.operator').setMinWidth(120),
    TableColumn.new('createTime', 'machine.beginTime').isTime().setMinWidth(150),
    TableColumn.new('endTime', 'machine.endTime').isTime().setMinWidth(150),
    TableColumn.new('fileKey', 'machine.file').isSlot(),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(120).fixedRight().alignCenter(),
];

const playerRef = ref<HTMLElement | null>(null);
const pageTableRef = ref<InstanceType<typeof PageTable> | null>(null);
const state = reactive({
    title: '',
    query: {
        pageNum: 1,
        pageSize: 10,
        machineId: 0,
    },
    playerVisible: false,
    execCmdsDialogVisible: false,
    execCmds: [],
});

const { query, playerVisible, execCmdsDialogVisible } = toRefs(state);

watch(
    [visible, () => props.machineId],
    async ([newVisible, newMachineId]) => {
        if (newVisible) {
            state.query.machineId = newMachineId || 0;
            state.title = props.title || '';
        }
    },
    { immediate: true }
);

const getTermOps = async () => {
    pageTableRef.value?.search();
};

const showExecCmds = (data: MachineTermOp) => {
    state.execCmds = data.execCmds ? JSON.parse(data.execCmds) : [];
    state.execCmdsDialogVisible = true;
};

let player: ReturnType<typeof AsciinemaPlayer.create> | null = null;

const playRec = async (rec: MachineTermOp & { playRecLoding?: boolean }) => {
    try {
        // 销毁旧播放器
        if (player) {
            player.dispose();
            player = null;
        }

        rec.playRecLoding = true;
        // 关闭列表 dialog，打开全屏播放器
        visible.value = false;
        state.playerVisible = true;

        // 等待 Teleport + Transition 完成 DOM 插入
        await nextTick();
        await nextTick();

        if (playerRef.value) {
            player = AsciinemaPlayer.create(getFileUrl(rec.fileKey), playerRef.value, {
                autoPlay: true,
                speed: 1.0,
                idleTimeLimit: 2,
                // fit:"both" 同时适配宽高，确保终端内容完整显示不截断
                fit: 'both',
                terminalFontSize: 14,
            });
        }
    } finally {
        rec.playRecLoding = false;
    }
};

const closePlayer = () => {
    // 销毁播放器
    if (player) {
        player.dispose();
        player = null;
    }
    state.playerVisible = false;
    // 重新打开列表 dialog
    visible.value = true;
};

/**
 * 关闭取消按钮触发的事件
 */
const handleClose = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss">
#terminalRecDialog {
    overflow: hidden;
}

/* 全屏终端回放容器 */
.rec-player-fullscreen {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background: #1e1e1e;
    display: flex;
    flex-direction: column;
}

.rec-player__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 16px;
    background: #2d2d2d;
    color: #e0e0e0;
    flex-shrink: 0;
    border-bottom: 1px solid #3d3d3d;
}

.rec-player__title {
    font-size: 14px;
    font-weight: 500;
}

.rec-player__body {
    flex: 1;
    overflow: hidden;
    padding: 0;

    /* asciinema-player 占满容器 */
    > div {
        height: 100%;
    }
}

/* 淡入淡出过渡 */
.rec-fade-enter-active,
.rec-fade-leave-active {
    transition: opacity 0.2s ease;
}

.rec-fade-enter-from,
.rec-fade-leave-to {
    opacity: 0;
}
</style>
