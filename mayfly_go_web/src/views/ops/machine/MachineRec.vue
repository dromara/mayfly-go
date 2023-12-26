<template>
    <div id="terminalRecDialog">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :before-close="handleClose"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            width="800"
            @open="getTermOps()"
        >
            <page-table ref="pageTableRef" :page-api="machineApi.termOpRecs" :lazy="true" height="100%" v-model:query-form="query" :columns="columns">
                <template #action="{ data }">
                    <el-button @click="playRec(data)" loading-icon="loading" :loading="data.playRecLoding" type="primary" link>回放</el-button>
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
    </div>
</template>

<script lang="ts" setup>
import { toRefs, watch, ref, reactive, nextTick, Ref } from 'vue';
import { machineApi } from './api';
import * as AsciinemaPlayer from 'asciinema-player';
import 'asciinema-player/dist/bundle/asciinema-player.css';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';

const props = defineProps({
    visible: { type: Boolean },
    machineId: { type: Number },
    title: { type: String },
});

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId']);

const columns = [
    TableColumn.new('creator', '操作者').setMinWidth(120),
    TableColumn.new('createTime', '开始时间').isTime().setMinWidth(150),
    TableColumn.new('endTime', '结束时间').isTime().setMinWidth(150),
    TableColumn.new('recordFilePath', '文件路径').setMinWidth(200),
    TableColumn.new('action', '操作').isSlot().setMinWidth(60).fixedRight().alignCenter(),
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
});

const { dialogVisible, query, playerDialogVisible } = toRefs(state);

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

let player: any = null;

const playRec = async (rec: any) => {
    try {
        if (player) {
            player.dispose();
        }
        rec.playRecLoding = true;
        const content = await machineApi.termOpRec.request({
            recId: rec.id,
            id: rec.machineId,
        });

        state.playerDialogVisible = true;
        nextTick(() => {
            player = AsciinemaPlayer.create(`data:text/plain;base64,${content}`, playerRef.value, {
                autoPlay: true,
                speed: 1.0,
                idleTimeLimit: 2,
                // fit: false,
                // terminalFontSize: 'small',
                // cols: 100,
                // rows: 33,
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
    .el-overlay .el-overlay-dialog .el-dialog .el-dialog__body {
        padding: 0px !important;
    }
}
</style>
