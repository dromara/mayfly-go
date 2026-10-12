<template>
    <el-dialog
        :title="`${$t('machine.process')}${title ? ': ' + title : ''}`"
        v-model="visible"
        :destroy-on-close="true"
        :show-close="true"
        :before-close="handleClose"
        width="65%"
    >
        <!-- 弹窗内嵌列表：DataTable 走裸容器（:card="false"）+ 不分页（:pageable="false"）+ 静态 :data，
             复用与分页页同一套列配置（列显隐、自适应列宽、统一行内操作），只是不套卡片、不接分页 -->
        <div class="h-[60vh]">
            <data-table class="h-full" :columns="columns" :data="processList" :pageable="false" :card="false" size="small">
                <template #tableHeader>
                    <div class="flex flex-wrap items-center gap-2">
                        <!-- 宽度类带 `!`：EP 的 .el-input/.el-select 宽度写在未分层样式里，优先级高于 Tailwind @layer 工具类，需 important 才能压过（否则控件退回 width:100% 换行堆叠） -->
                        <el-input
                            size="small"
                            class="w-[150px]!"
                            :placeholder="$t('machine.processName')"
                            v-model="params.name"
                            clearable
                            @keyup.enter="getProcess"
                        />
                        <el-select size="small" class="w-[120px]!" v-model="params.sortType" :placeholder="$t('machine.selectSortType')" @change="getProcess">
                            <el-option :label="$t('machine.cpuDesc')" value="1" />
                            <el-option :label="$t('machine.memDesc')" value="2" />
                        </el-select>
                        <el-select size="small" class="w-[90px]!" v-model="params.count" :placeholder="$t('machine.selectProcessNum')" @change="getProcess">
                            <el-option v-for="n in [10, 15, 20, 25]" :key="n" :label="`${n}`" :value="`${n}`" />
                        </el-select>
                        <el-button size="small" plain icon="refresh" @click="getProcess">{{ $t('common.refresh') }}</el-button>
                    </div>
                </template>
            </data-table>
        </div>
    </el-dialog>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { reactive, toRefs, watch } from 'vue';
import { machineApi } from './api';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import type { MachineProcess } from './types';

const props = defineProps({
    title: { type: String },
});

const emit = defineEmits(['cancel']);

const visible = defineModel<boolean>('visible', { default: false });
const machineId = defineModel<number | null>('machineId');

const state = reactive({
    params: {
        name: '',
        sortType: '1',
        count: '10',
        id: 0,
    },
    processList: [] as MachineProcess[],
});

const { params, processList } = toRefs(state);

const getProcess = async () => {
    const res = await machineApi.process.request(state.params);
    // 解析字符串
    // USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
    // root         1  0.0  0.0 125632  3352 ?        Ss    2019 154:04 /usr/lib/systemd/systemd --system --deserialize 22
    const psStrings = res.split('\n');
    const ps: MachineProcess[] = [];
    // 如果有根据名称查进程，则第一行没有表头
    const index = state.params.name == '' ? 1 : 0;
    for (let i = index; i < psStrings.length; i++) {
        const psStr = psStrings[i];
        const process = psStr.split(/\s+/);
        if (process.length < 2) {
            continue;
        }
        let command = process[10];
        // 搜索进程时由于使用grep命令，可能会多个bash或grep进程
        if (state.params.name) {
            if (command == 'bash' || command == 'grep') {
                continue;
            }
        }
        // 获取command，由于command中也有可能存在空格被切割，故重新拼接
        for (let j = 10; j < process.length - 1; j++) {
            command += ' ' + process[j + 1];
        }
        ps.push({
            user: process[0],
            pid: process[1],
            cpu: process[2],
            mem: process[3],
            vsz: kb2Mb(process[4]),
            rss: kb2Mb(process[5]),
            stat: process[7],
            start: process[8],
            time: process[9],
            command,
        });
    }
    state.processList = ps;
};

const confirmKillProcess = async (pid: string | number) => {
    await machineApi.killProcess.request({
        pid,
        id: state.params.id,
    });
    Msg.success('kill success');
    state.params.name = '';
    getProcess();
};

const kb2Mb = (kb: string) => {
    return (parseInt(kb) / 1024).toFixed(2) + 'M';
};

/**
 * 列声明复用 DataTable 机制：
 * - 列宽全部交给自适应（按实际内容估宽 + 渲染后实测回填），不写死 minWidth —— 手写定宽既脆弱（PID 位数、用户名长度因机器而异，写死就会截断）又与机制重复；
 * - jargon 列表头（VSZ/RSS/STAT/START/TIME）用 setHeaderTip 挂问号提示，与分页页同一渲染路径；
 * - command 设为弹性列吸收容器富余并靠溢出 tooltip 兜底截断；
 * - 操作列用 setActions：权限、就地确认、列宽从同一份声明推出。
 */
const columns: TableColumn[] = [
    new TableColumn('user', 'USER'),
    new TableColumn('pid', 'PID'),
    new TableColumn('cpu', '%CPU'),
    new TableColumn('mem', '%MEM'),
    new TableColumn('vsz', 'VSZ').setHeaderTip('machine.virtualMemory'),
    new TableColumn('rss', 'RSS').setHeaderTip('machine.fixedMemory'),
    new TableColumn('stat', 'STAT').setHeaderTip('machine.procState'),
    new TableColumn('start', 'START').setHeaderTip('machine.startTime'),
    new TableColumn('time', 'TIME').setHeaderTip('machine.procCpuRunTime'),
    new TableColumn('command', 'command').setFlexible(),
    new TableColumn('action', 'common.operation').setActions<MachineProcess>(
        [
            {
                key: 'kill',
                label: 'machine.kill',
                type: 'danger',
                icon: 'delete',
                perm: 'machine:killprocess',
                confirm: { title: 'machine.killProcConfirm', pop: true },
                onClick: (row: MachineProcess) => confirmKillProcess(row.pid),
            },
        ],
        { size: 'small', appearance: 'plain' }
    ),
];

watch(
    machineId,
    (val) => {
        if (val) {
            state.params.id = val;
            getProcess();
        }
    },
    { immediate: true }
);

/**
 * 关闭取消按钮触发的事件
 */
const handleClose = () => {
    visible.value = false;
    machineId.value = null;
    emit('cancel');
    state.params = {
        name: '',
        sortType: '1',
        count: '10',
        id: 0,
    };
    state.processList = [];
};
</script>
