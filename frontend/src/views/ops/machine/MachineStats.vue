<template>
    <el-dialog :title="title" v-model="visible" :close-on-click-modal="true" :destroy-on-close="true" :before-close="cancel" width="1050px">
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

        <el-divider />
        <div class="mb-2 flex items-center justify-between">
            <span style="font-size: 16px; font-weight: 700">{{ $t('machine.metricTrend') }}</span>
            <el-radio-group v-model="state.trendRange" size="small" @change="loadTrend">
                <el-radio-button value="1h">{{ $t('machine.last1h') }}</el-radio-button>
                <el-radio-button value="24h">{{ $t('machine.last24h') }}</el-radio-button>
                <el-radio-button value="7d">{{ $t('machine.last7d') }}</el-radio-button>
            </el-radio-group>
        </div>
        <el-row :gutter="20">
            <el-col :lg="12" :md="12">
                <ECharts height="240" :option="state.trendOption" />
            </el-col>
            <el-col :lg="12" :md="12">
                <ECharts height="240" :option="state.netTrendOption" />
            </el-col>
        </el-row>
    </el-dialog>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, nextTick } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import { machineApi, metricApi } from './api';
import ECharts from '@/components/echarts/ECharts.vue';
import { ECOption } from '@/components/echarts/config';
import { useI18n } from 'vue-i18n';
import type { MachineStats, MachineNetIntfInfo, MachineMetric } from './types';

const { t } = useI18n();

const props = defineProps({
    title: {
        type: String,
    },
});

const emit = defineEmits(['cancel']);

const visible = defineModel<boolean>('visible', { default: false });
const machineId = defineModel<number | null>('machineId');

const state = reactive({
    stats: {} as MachineStats,
    netInter: [] as (MachineNetIntfInfo & { name: string })[],
    memOption: {} as ECOption,
    cpuOption: {} as ECOption,
    trendRange: '24h',
    trendOption: {} as ECOption,
    netTrendOption: {} as ECOption,
});

const { stats, netInter } = toRefs(state);

const setStats = async () => {
    state.stats = await machineApi.stats.request({ id: machineId.value });
};

watch(
    visible,
    async (val) => {
        if (val) {
            await setStats();
            initCharts();
            loadTrend();
        }
    },
    { immediate: true }
);

const onRefresh = async () => {
    await setStats();
    initCharts();
    loadTrend();
};

// 趋势时间范围（毫秒）
const RANGE_MS: Record<string, number> = { '1h': 3600_000, '24h': 86_400_000, '7d': 604_800_000 };

// 加载指标历史并渲染趋势图
const loadTrend = async () => {
    const end = Date.now();
    const start = end - (RANGE_MS[state.trendRange] ?? RANGE_MS['24h']);
    const list = (await metricApi.range.request({ id: machineId.value as number, start, end })) || [];
    renderTrend(list);
};

const renderTrend = (list: MachineMetric[]) => {
    const times = list.map((m) => new Date(m.collectTime).toLocaleTimeString());

    // 响应经 json-bigint(storeAsString) 解析后，长小数会回传成字符串（TS 的 number 声明不可信），
    // 直接调 .toFixed 会抛错并让整张趋势图静默空白，故所有数值先收敛为 number
    const num = (v: unknown) => {
        const n = Number(v);
        return Number.isFinite(n) ? n : 0;
    };

    state.trendOption = {
        title: { text: t('machine.usageTrend'), textStyle: { fontSize: 15 } },
        tooltip: { trigger: 'axis', valueFormatter: (v) => `${Number(v).toFixed(1)}%` },
        // 标题默认居中置顶，图例若用百分比 top 会压到标题行；改用像素把图例放到标题下方并同步下移 grid，
        // 且与标题/绘图区各留足间距，避免三者贴在一起
        legend: { top: 44 },
        grid: { left: 40, right: 20, bottom: 30, top: 74 },
        xAxis: { type: 'category', data: times },
        yAxis: { type: 'value', max: 100 },
        series: [
            { name: 'CPU', type: 'line', smooth: true, showSymbol: false, data: list.map((m) => Number(num(m.cpuUsage).toFixed(1))) },
            { name: t('machine.memory'), type: 'line', smooth: true, showSymbol: false, data: list.map((m) => Number(num(m.memUsage).toFixed(1))) },
            { name: t('machine.disk'), type: 'line', smooth: true, showSymbol: false, data: list.map((m) => Number(num(m.diskUsage).toFixed(1))) },
        ],
    };

    // 网络速率：相邻点累计字节差分 / 间隔秒 -> KB/s
    const rx: number[] = [];
    const tx: number[] = [];
    for (let i = 0; i < list.length; i++) {
        if (i === 0) {
            rx.push(0);
            tx.push(0);
            continue;
        }
        const dt = (new Date(list[i].collectTime).getTime() - new Date(list[i - 1].collectTime).getTime()) / 1000;
        if (dt <= 0) {
            rx.push(0);
            tx.push(0);
            continue;
        }
        rx.push(Number((Math.max(0, num(list[i].netRx) - num(list[i - 1].netRx)) / dt / 1024).toFixed(1)));
        tx.push(Number((Math.max(0, num(list[i].netTx) - num(list[i - 1].netTx)) / dt / 1024).toFixed(1)));
    }

    state.netTrendOption = {
        title: { text: t('machine.netRate'), textStyle: { fontSize: 15 } },
        tooltip: { trigger: 'axis', valueFormatter: (v) => `${Number(v).toFixed(1)} KB/s` },
        // 同 trendOption：图例像素定位避开居中标题行并留足间距
        legend: { top: 44 },
        grid: { left: 55, right: 20, bottom: 30, top: 74 },
        xAxis: { type: 'category', data: times },
        yAxis: { type: 'value' },
        series: [
            { name: t('machine.receive'), type: 'line', smooth: true, showSymbol: false, data: rx },
            { name: t('machine.send'), type: 'line', smooth: true, showSymbol: false, data: tx },
        ],
    };
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
            valueFormatter: (val) => formatByteSize(Number(val)),
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
            valueFormatter: (value) => `${value}%`,
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
        let value: MachineNetIntfInfo & { name?: string } = values[i];
        // 将网卡名称赋值新属性值name
        value.name = keys[i];
        state.netInter.push(value as MachineNetIntfInfo & { name: string });
    }
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
