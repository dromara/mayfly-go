<template>
    <div class="file-manage">
        <el-dialog title="进程信息" v-model="dialogVisible" :destroy-on-close="true" :show-close="true" :before-close="handleClose" width="65%">
            <div class="card pd5">
                <el-row>
                    <el-col :span="4">
                        <el-input size="small" placeholder="进程名" v-model="params.name" plain clearable></el-input>
                    </el-col>
                    <el-col :span="4" class="ml5">
                        <el-select class="w100" @change="getProcess" size="small" v-model="params.sortType" placeholder="请选择排序类型">
                            <el-option key="cpu" label="cpu降序" value="1"> </el-option>
                            <el-option key="cpu" label="mem降序" value="2"> </el-option>
                        </el-select>
                    </el-col>
                    <el-col :span="4" class="ml5">
                        <el-select class="w100" @change="getProcess" size="small" v-model="params.count" placeholder="请选择进程个数">
                            <el-option key="10" label="10" value="10"> </el-option>
                            <el-option key="15" label="15" value="15"> </el-option>
                            <el-option key="20" label="20" value="20"> </el-option>
                            <el-option key="25" label="25" value="25"> </el-option>
                        </el-select>
                    </el-col>
                    <el-col :span="6">
                        <el-button class="ml5" @click="getProcess" type="primary" icon="tickets" size="small" plain>刷新 </el-button>
                    </el-col>
                </el-row>
            </div>

            <el-table :data="processList" size="small" style="width: 100%">
                <el-table-column prop="user" label="USER" :min-width="50"> </el-table-column>
                <el-table-column prop="pid" label="PID" :min-width="50" show-overflow-tooltip></el-table-column>
                <el-table-column prop="cpu" label="%CPU" :min-width="40"> </el-table-column>
                <el-table-column prop="mem" label="%MEM" :min-width="42"> </el-table-column>
                <el-table-column prop="vsz" label="vsz" :min-width="55">
                    <template #header>
                        VSZ
                        <el-tooltip class="box-item" effect="dark" content="虚拟内存" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="rss" :min-width="52">
                    <template #header>
                        RSS
                        <el-tooltip class="box-item" effect="dark" content="固定内存" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="stat" :min-width="50">
                    <template #header>
                        STAT
                        <el-tooltip class="box-item" effect="dark" content="进程状态" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="start" :min-width="50">
                    <template #header>
                        START
                        <el-tooltip class="box-item" effect="dark" content="启动时间" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="time" :min-width="50">
                    <template #header>
                        TIME
                        <el-tooltip class="box-item" effect="dark" content="该进程实际使用CPU运作的时间" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="command" label="command" :min-width="120" show-overflow-tooltip> </el-table-column>

                <el-table-column label="操作">
                    <template #default="scope">
                        <el-popconfirm title="确定终止该进程?" @confirm="confirmKillProcess(scope.row.pid)" width="160">
                            <template #reference>
                                <el-button v-auth="'machine:killprocess'" type="danger" icon="delete" size="small" plain>终止</el-button>
                            </template>
                        </el-popconfirm>
                        <!-- <el-button @click="addFiles(scope.row)" type="danger" icon="delete" size="small" plain>终止</el-button> -->
                    </template>
                </el-table-column>
            </el-table>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch } from 'vue';
import { ElMessage } from 'element-plus';
import { machineApi } from './api';

const props = defineProps({
    visible: { type: Boolean },
    machineId: { type: Number },
    title: { type: String },
});

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId']);

const state = reactive({
    dialogVisible: false,
    params: {
        name: '',
        sortType: '1',
        count: '10',
        id: 0,
    },
    processList: [],
});

const { dialogVisible, params, processList } = toRefs(state);

watch(props, (newValue) => {
    if (props.machineId) {
        state.params.id = props.machineId;
        getProcess();
    }
    state.dialogVisible = newValue.visible;
});

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
    state.processList = ps as any;
};

const confirmKillProcess = async (pid: any) => {
    await machineApi.killProcess.request({
        pid,
        id: state.params.id,
    });
    ElMessage.success('kill success');
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
    emit('update:visible', false);
    emit('update:machineId', null);
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
