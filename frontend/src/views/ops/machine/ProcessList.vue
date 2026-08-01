<template>
    <div class="file-manage">
        <el-dialog
            :title="$t('machine.process') + `: ${title}`"
            v-model="visible"
            :destroy-on-close="true"
            :show-close="true"
            :before-close="handleClose"
            width="65%"
        >
            <div class="card p-1!">
                <el-row>
                    <el-col :span="4">
                        <el-input size="small" :placeholder="$t('machine.processName')" v-model="params.name" plain clearable></el-input>
                    </el-col>
                    <el-col :span="4" class="ml-1">
                        <el-select @change="getProcess" size="small" v-model="params.sortType" :placeholder="$t('machine.selectSortType')">
                            <el-option key="cpu" :label="$t('machine.cpuDesc')" value="1"> </el-option>
                            <el-option key="cpu" :label="$t('machine.memDesc')" value="2"> </el-option>
                        </el-select>
                    </el-col>
                    <el-col :span="4" class="ml-1">
                        <el-select @change="getProcess" size="small" v-model="params.count" :placeholder="$t('machine.selectProcessNum')">
                            <el-option key="10" label="10" value="10"> </el-option>
                            <el-option key="15" label="15" value="15"> </el-option>
                            <el-option key="20" label="20" value="20"> </el-option>
                            <el-option key="25" label="25" value="25"> </el-option>
                        </el-select>
                    </el-col>
                    <el-col :span="6">
                        <el-button class="ml-1" @click="getProcess" type="primary" icon="refresh" size="small" plain>{{ $t('common.refresh') }}</el-button>
                    </el-col>
                </el-row>
            </div>

            <el-table :data="processList" size="small" :height="500">
                <el-table-column prop="user" label="USER" :min-width="50"> </el-table-column>
                <el-table-column prop="pid" label="PID" :min-width="50" show-overflow-tooltip></el-table-column>
                <el-table-column prop="cpu" label="%CPU" :min-width="40"> </el-table-column>
                <el-table-column prop="mem" label="%MEM" :min-width="42"> </el-table-column>
                <el-table-column prop="vsz" label="vsz" :min-width="55">
                    <template #header>
                        VSZ
                        <el-tooltip class="box-item" effect="dark" :content="$t('machine.virtualMemory')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="rss" :min-width="52">
                    <template #header>
                        RSS
                        <el-tooltip class="box-item" effect="dark" :content="$t('machine.fixedMemory')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="stat" :min-width="50">
                    <template #header>
                        STAT
                        <el-tooltip class="box-item" effect="dark" :content="$t('machine.procState')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="start" :min-width="50">
                    <template #header>
                        START
                        <el-tooltip class="box-item" effect="dark" :content="$t('machine.startTime')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="time" :min-width="50">
                    <template #header>
                        TIME
                        <el-tooltip class="box-item" effect="dark" :content="$t('machine.procCpuRunTime')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="command" label="command" :min-width="120" show-overflow-tooltip> </el-table-column>

                <el-table-column :label="$t('common.operation')">
                    <template #default="scope">
                        <el-popconfirm :title="$t('machine.killProcConfirm')" @confirm="confirmKillProcess(scope.row.pid)" width="160">
                            <template #reference>
                                <el-button v-auth="'machine:killprocess'" type="danger" icon="delete" size="small" plain>{{ $t('machine.kill') }}</el-button>
                            </template>
                        </el-popconfirm>
                    </template>
                </el-table-column>
            </el-table>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { reactive, toRefs, watch } from 'vue';
import { machineApi } from './api';
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
    const ps = [];
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
    state.processList = ps as unknown as MachineProcess[];
};

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

watch(visible, (val) => {
    if (!val) {
        // sync close state if needed
    }
});

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
