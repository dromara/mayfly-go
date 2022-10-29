<template>
    <div id="terminalRecDialog">
        <el-dialog :title="title" v-model="dialogVisible" :before-close="handleClose" :close-on-click-modal="false"
            :destroy-on-close="true" width="70%">
            <div class="toolbar">
                <el-select @change="getUsers" v-model="operateDate" placeholder="操作日期" filterable>
                    <el-option v-for="item in operateDates" :key="item" :label="item" :value="item"> </el-option>
                </el-select>
                <el-select class="ml10" @change="getRecs" filterable v-model="user" placeholder="请选择操作人">
                    <el-option v-for="item in users" :key="item" :label="item" :value="item"> </el-option>
                </el-select>
                <el-select class="ml10" @change="playRec" filterable v-model="rec" placeholder="请选择操作记录">
                    <el-option v-for="item in recs" :key="item" :label="item" :value="item"> </el-option>
                </el-select>
                <el-divider direction="vertical" border-style="dashed" />
                快捷键-> space[空格键]: 暂停/播放
            </div>
            <div ref="playerRef" id="rc-player"></div>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, watch, ref, reactive } from 'vue';
import { machineApi } from './api';
import * as AsciinemaPlayer from 'asciinema-player';
import 'asciinema-player/dist/bundle/asciinema-player.css';

const props = defineProps({
    visible: { type: Boolean },
    machineId: { type: Number },
    title: { type: String },
})

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId'])

const playerRef = ref(null);
const state = reactive({
    dialogVisible: false,
    title: '',
    machineId: 0,
    operateDates: [],
    users: [],
    recs: [],
    operateDate: '',
    user: '',
    rec: '',
});

const {
    dialogVisible,
    title,
    operateDates,
    operateDate,
    users,
    recs,
    user,
    rec,
} = toRefs(state)

watch(props, async (newValue: any) => {
    const visible = newValue.visible;
    if (visible) {
        state.machineId = newValue.machineId;
        state.title = newValue.title;
        await getOperateDate();
    }
    state.dialogVisible = visible;
});

const getOperateDate = async () => {
    const res = await machineApi.recDirNames.request({ path: state.machineId });
    state.operateDates = res as any;
};

const getUsers = async (operateDate: string) => {
    state.users = [];
    state.user = '';
    state.recs = [];
    state.rec = '';
    const res = await machineApi.recDirNames.request({ path: `${state.machineId}/${operateDate}` });
    state.users = res as any;
};

const getRecs = async (user: string) => {
    state.recs = [];
    state.rec = '';
    const res = await machineApi.recDirNames.request({ path: `${state.machineId}/${state.operateDate}/${user}` });
    state.recs = res as any;
};

let player: any = null;

const playRec = async (rec: string) => {
    if (player) {
        player.dispose();
    }
    const content = await machineApi.recDirNames.request({
        isFile: '1',
        path: `${state.machineId}/${state.operateDate}/${state.user}/${rec}`,
    });
    player = AsciinemaPlayer.create(`data:text/plain;base64,${content}`, playerRef.value, {
        autoPlay: true,
        speed: 1.0,
        idleTimeLimit: 2,
    });
};

/**
 * 关闭取消按钮触发的事件
 */
const handleClose = () => {
    emit('update:visible', false);
    emit('update:machineId', null);
    emit('cancel');
    state.operateDates = [];
    state.users = [];
    state.recs = [];
    state.operateDate = '';
    state.user = '';
    state.rec = '';
};
</script>
<style lang="scss">
#terminalRecDialog {
    .el-overlay .el-overlay-dialog .el-dialog .el-dialog__body {
        padding: 0px !important;
    }
}
</style>
