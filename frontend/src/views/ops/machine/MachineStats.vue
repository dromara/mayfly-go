<template>
    <el-dialog :title="title" v-model="dialogVisible" :close-on-click-modal="true" :destroy-on-close="true" :before-close="cancel" width="1050px">
        <el-row :gutter="20">
            <el-col :lg="12" :md="12">
                <el-descriptions size="small" :title="$t('machine.basicInfo')" :column="2" border>
                    <template #extra>
                        <el-link @click="onRefresh" icon="refresh" underline="never" type="success"></el-link>
                    </template>
                    <el-descriptions-item :label="$t('machine.hostname')">
                        {{ stats.hostname }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('machine.runTime')">
                        {{ stats.uptime }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('machine.totalTask')">
                        {{ stats.totalProcs }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('machine.runningTask')">
                        {{ stats.runningProcs }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('machine.load')"> {{ stats.load1 }} {{ stats.load5 }} {{ stats.load10 }} </el-descriptions-item>
                </el-descriptions>
            </el-col>

            <el-col :lg="6" :md="6">
                <ECharts height="200" :option="state.memOption" />
            </el-col>

            <el-col :lg="6" :md="6">
                <ECharts height="200" :option="state.cpuOption" />
            </el-col>
        </el-row>

        <el-row :gutter="20">
            <el-col :lg="8" :md="8">
                <span style="font-size: 16px; font-weight: 700">{{ $t('machine.disk') }}</span>
                <el-table :data="stats.fSInfos" stripe max-height="250" style="width: 100%" border>
                    <el-table-column prop="mountPoint" :label="$t('machine.mountPoint')" min-width="100" show-overflow-tooltip> </el-table-column>
                    <el-table-column :label="$t('machine.available')" min-width="70" show-overflow-tooltip>
                        <template #default="scope">
                            {{ formatByteSize(scope.row.free) }}
                        </template>
                    </el-table-column>
                    <el-table-column prop="Used" :label="$t('machine.used')" min-width="70" show-overflow-tooltip>
                        <template #default="scope">
                            {{ formatByteSize(scope.row.used) }}
                        </template>
                    </el-table-column>
                </el-table>
            </el-col>

            <el-col :lg="16" :md="16">
                <span style="font-size: 16px; font-weight: 700">{{ $t('machine.networkCard') }}</span>
                <el-table :data="netInter" stripe max-height="250" style="width: 100%" border>
                    <el-table-column prop="name" :label="$t('machine.networkCard')" min-width="120" show-overflow-tooltip></el-table-column>
                    <el-table-column prop="ipv4" label="IPv4" min-width="130" show-overflow-tooltip> </el-table-column>
                    <el-table-column prop="ipv6" label="IPv6" min-width="130" show-overflow-tooltip> </el-table-column>
                    <el-table-column prop="rx" :label="`${$t('machine.receive')}(rx)`" min-width="110" show-overflow-tooltip>
                        <template #default="scope">
                            {{ formatByteSize(scope.row.rx) }}
                        </template>
                    </el-table-column>
                    <el-table-column prop="tx" :label="`${$t('machine.send')}(tx)`" min-width="110" show-overflow-tooltip>
                        <template #default="scope">
                            {{ formatByteSize(scope.row.tx) }}
                        </template>
                    </el-table-column>
                </el-table>
            </el-col>
        </el-row>
    </el-dialog>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, nextTick } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import { machineApi } from './api';
import ECharts from '@/components/echarts/ECharts.vue';
import { ECOption } from '@/components/echarts/config';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    visible: {
        type: Boolean,
    },
    machineId: {
        type: Number,
    },
    title: {
        type: String,
    },
});

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId']);

const state = reactive({
    dialogVisible: false,
    stats: {} as any,
    netInter: [] as any,
    memOption: {},
    cpuOption: {},
});

const { dialogVisible, stats, netInter } = toRefs(state);

watch(props, async (newValue: any) => {
    const visible = newValue.visible;
    if (visible) {
        await setStats();
    }
    state.dialogVisible = visible;
    if (visible) {
        initCharts();
    }
});

const setStats = async () => {
    state.stats = await machineApi.stats.request({ id: props.machineId });
};

const onRefresh = async () => {
    await setStats();
    initCharts();
};

const initMemStats = () => {
    const mem = state.stats.memInfo;
    const data = [
        { name: t('machine.available'), value: mem.available },
        {
            name: t('machine.used'),
            value: mem.total - mem.available,
        },
    ];

    const option: ECOption = {
        title: {
            text: t('machine.memory'),
            textStyle: { fontSize: 15 },
        },
        tooltip: {
            trigger: 'item',
            valueFormatter: (val: any) => formatByteSize(val),
        },
        legend: {
            top: '15%',
            orient: 'vertical',
            left: 'left',
            textStyle: { fontSize: 12 },
        },
        series: [
            {
                name: t('machine.memory'),
                type: 'pie',
                radius: ['30%', '60%'], // 饼图内圈和外圈大小
                center: ['60%', '50%'], // 饼图位置，0: 左右；1: 上下
                avoidLabelOverlap: false,
                label: {
                    show: false,
                    position: 'center',
                },
                emphasis: {
                    label: {
                        show: true,
                        fontSize: '15',
                        fontWeight: 'bold',
                    },
                },
                labelLine: {
                    show: false,
                },
                data: data,
            },
        ],
    };
    state.memOption = option;
};

const initCpuStats = () => {
    const cpu = state.stats.cpu;
    const data = [
        { name: 'Idle', value: cpu.idle },
        {
            name: 'Iowait',
            value: cpu.iowait,
        },
        {
            name: 'System',
            value: cpu.system,
        },
        {
            name: 'User',
            value: cpu.user,
        },
    ];

    const option: ECOption = {
        title: {
            text: t('machine.cpuUsageRate'),
            textStyle: { fontSize: 15 },
        },
        tooltip: {
            trigger: 'item',
            valueFormatter: (value: any) => value + '%',
        },
        legend: {
            top: '15%',
            orient: 'vertical',
            left: 'left',
            textStyle: { fontSize: 12 },
        },
        series: [
            {
                name: 'CPU',
                type: 'pie',
                radius: ['30%', '60%'], // 饼图内圈和外圈大小
                center: ['60%', '50%'], // 饼图位置，0: 左右；1: 上下
                avoidLabelOverlap: false,
                label: {
                    show: false,
                    position: 'center',
                },
                emphasis: {
                    label: {
                        show: true,
                        fontSize: '15',
                        fontWeight: 'bold',
                    },
                },
                labelLine: {
                    show: false,
                },
                data: data,
            },
        ],
    };
    state.cpuOption = option;
};

const initCharts = () => {
    nextTick(() => {
        initMemStats();
        initCpuStats();
    });
    parseNetInter();
};

const parseNetInter = () => {
    state.netInter = [];
    const netInter = state.stats.netIntf;
    const keys = Object.keys(netInter);
    const values = Object.values(netInter);
    for (let i = 0; i < values.length; i++) {
        let value: any = values[i];
        // 将网卡名称赋值新属性值name
        value.name = keys[i];
        state.netInter.push(value);
    }
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
