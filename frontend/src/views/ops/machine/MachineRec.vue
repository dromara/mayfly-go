<template>
    <div id="terminalRecDialog">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
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

        <el-dialog
            :title="title"
            v-model="playerDialogVisible"
            :before-close="handleClosePlayer"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            width="70%"
        >
            <div ref="playerRef" id="rc-player"></div>
        </el-dialog>

        <el-dialog :title="$t('machine.execCmdRecord')" v-model="execCmdsDialogVisible" :destroy-on-close="true" width="500">
            <el-table :data="state.execCmds" max-height="480" stripe size="small">
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
import { toRefs, watch, ref, reactive, nextTick, Ref } from 'vue';
import { machineApi } from './api';
import * as AsciinemaPlayer from 'asciinema-player';
import 'asciinema-player/dist/bundle/asciinema-player.css';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { formatDate } from '@/common/utils/format';
import { getFileUrl } from '@/common/request';
import FileInfo from '@/components/file/FileInfo.vue';

const props = defineProps({
    visible: { type: Boolean },
    machineId: { type: Number },
    title: { type: String },
});

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId']);

const columns = [
    TableColumn.new('creator', 'machine.operator').setMinWidth(120),
    TableColumn.new('createTime', 'machine.beginTime').isTime().setMinWidth(150),
    TableColumn.new('endTime', 'machine.endTime').isTime().setMinWidth(150),
    TableColumn.new('fileKey', 'machine.file').isSlot(),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(120).fixedRight().alignCenter(),
];

const playerRef = ref(null);
const pageTableRef: Ref<any> = ref(null);
const state = reactive({
    dialogVisible: false,
    title: '',
    query: {
        pageNum: 1,
        pageSize: 10,
        machineId: 0,
    },
    playerDialogVisible: false,
    execCmdsDialogVisible: false,
    execCmds: [],
});

const { dialogVisible, query, playerDialogVisible, execCmdsDialogVisible } = toRefs(state);

watch(props, async (newValue: any) => {
    const visible = newValue.visible;
    state.dialogVisible = visible;
    if (visible) {
        state.query.machineId = newValue.machineId;
        state.title = newValue.title;
    }
});

const getTermOps = async () => {
    pageTableRef.value.search();
};

const showExecCmds = (data: any) => {
    state.execCmds = JSON.parse(data.execCmds);
    state.execCmdsDialogVisible = true;
};

let player: any = null;

const playRec = async (rec: any) => {
    try {
        if (player) {
            player.dispose();
        }
        rec.playRecLoding = true;
        state.playerDialogVisible = true;
        nextTick(() => {
            player = AsciinemaPlayer.create(getFileUrl(rec.fileKey), playerRef.value, {
                autoPlay: true,
                speed: 1.0,
                idleTimeLimit: 2,
                // fit: false,
                // terminalFontSize: 'small',
                // cols: 144,
                // rows: 32,
            });
        });
    } finally {
        rec.playRecLoding = false;
    }
};

const handleClosePlayer = () => {
    state.playerDialogVisible = false;
};

/**
 * 关闭取消按钮触发的事件
 */
const handleClose = () => {
    emit('update:visible', false);
    emit('update:machineId', null);
    emit('cancel');
};
</script>
<style lang="scss">
#terminalRecDialog {
    overflow: hidden;

    #rc-player {
        overflow: hidden;
    }

    .el-overlay .el-overlay-dialog .el-dialog .el-dialog__body {
        padding: 0px !important;
    }
}
</style>
