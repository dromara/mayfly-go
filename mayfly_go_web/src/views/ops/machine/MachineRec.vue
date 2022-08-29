<template>
    <div>
        <div class="toolbar">
            <span style="dispaly: inline-block" class="ml10">{{ title }}</span>
            <el-divider direction="vertical" border-style="dashed" />
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
            快捷键-> space[空格键]: 暂停/播放  |  f: 全屏/取消全屏
        </div>
        <div ref="playerRef" id="rc-player"></div>
    </div>
</template>

<script lang="ts">
import { toRefs, onMounted, ref, reactive, defineComponent } from 'vue';
import { machineApi } from './api';
import * as AsciinemaPlayer from 'asciinema-player';
import 'asciinema-player/dist/bundle/asciinema-player.css';
import { useRoute } from 'vue-router';
export default defineComponent({
    name: 'MachineRec',
    components: {},
    props: {
        visible: { type: Boolean },
        machineId: { type: Number },
        title: { type: String },
    },
    setup(props: any, context) {
        const route = useRoute();
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

        onMounted(() => {
            state.machineId = Number.parseInt(route.query.id as string);
            state.title = route.query.name as string;
            getOperateDate();
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
            context.emit('update:visible', false);
            context.emit('update:machineId', null);
            context.emit('cancel');
            state.operateDates = [];
            state.users = [];
            state.recs = [];
            state.operateDate = '';
            state.user = '';
            state.rec = '';
        };

        return {
            ...toRefs(state),
            playerRef,
            getUsers,
            getRecs,
            playRec,
            handleClose,
        };
    },
});
</script>
