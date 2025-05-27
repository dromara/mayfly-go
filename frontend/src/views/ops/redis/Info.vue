<template>
    <div>
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :show-close="true"
            width="1000px"
            @close="close()"
            body-class="h-[65vh] overflow-y-auto overflow-x-hidden"
        >
            <el-row :gutter="20">
                <el-col :lg="16" :md="16">
                    <el-descriptions class="redis-info info-server" :title="$t('redis.redisInfoTitle')" :column="3" size="small" border>
                        <el-descriptions-item :label="$t('redis.version')">{{ info.Server.redis_version }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.port')">{{ info.Server.tcp_port }}</el-descriptions-item>
                        <el-descriptions-item label="PID">{{ info.Server.process_id }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.mode')">{{ info.Server.redis_mode }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.os')">{{ info.Server.os }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.uptimeDays')">{{ info.Server.uptime_in_days }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.execPath')">{{ info.Server.executable }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.confFile')">{{ info.Server.config_file }}</el-descriptions-item>
                    </el-descriptions>
                </el-col>
                <el-col :lg="8" :md="8" class="redis-info">
                    <ECharts height="150" width="360" :option="state.memOption" />
                </el-col>
            </el-row>

            <el-row :gutter="20">
                <el-col :lg="12" :md="12">
                    <el-descriptions class="redis-info info-cluster" :title="$t('redis.node')" :column="3" size="small" border>
                        <el-descriptions-item :label="$t('redis.clusterEnable')">{{ info.Cluster?.cluster_enabled }}</el-descriptions-item>
                        <el-descriptions-item label="DB">{{ info.Cluster?.databases }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.nodeCount')">{{ info.Cluster?.nodecount }}</el-descriptions-item>
                    </el-descriptions>
                </el-col>

                <el-col :lg="12" :md="12">
                    <el-descriptions class="redis-info info-client" :title="$t('redis.clientConn')" :column="3" size="small" border>
                        <el-descriptions-item :label="$t('redis.connectedNum')">{{ info.Clients.connected_clients }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('redis.blockedClientNum')">{{ info.Clients.blocked_clients }}</el-descriptions-item>
                    </el-descriptions>
                </el-col>
            </el-row>

            <el-descriptions class="redis-info info-memory" title="CPU" :column="2" size="small" border>
                <el-descriptions-item :label="$t('redis.sysCpu')">{{ info.CPU.used_cpu_sys }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.userCpu')">{{ info.CPU.used_cpu_user }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.sysChildCpu')">{{ info.CPU.used_cpu_sys_children }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.userChildCpu')">{{ info.CPU.used_cpu_user_children }}</el-descriptions-item>
            </el-descriptions>

            <el-row :gutter="20" class="redis-info">
                <el-col :lg="24" :md="24">
                    <span style="font-size: 14px; font-weight: 700">{{ $t('redis.keyCount') }}</span>
                    <el-table :data="Keyspace" stripe max-height="250" style="width: 100%" border>
                        <el-table-column prop="db" label="DB" min-width="100" show-overflow-tooltip> </el-table-column>
                        <el-table-column prop="keys" label="keys" min-width="70" show-overflow-tooltip> </el-table-column>
                        <el-table-column prop="expires" label="expires" min-width="70" show-overflow-tooltip> </el-table-column>
                        <el-table-column prop="avg_ttl" label="avg_ttl" min-width="70" show-overflow-tooltip> </el-table-column>
                    </el-table>
                </el-col>
            </el-row>

            <el-descriptions class="redis-info info-state" :title="$t('redis.countInfo')" :column="3" size="small" border>
                <el-descriptions-item :label="$t('redis.totalCmdProcess')">{{ info.Stats.total_commands_processed }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.curQps')">{{ info.Stats.instantaneous_ops_per_sec }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.expiredKeys')">{{ info.Stats.expired_keys }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.netInputBytes')">{{ info.Stats.total_net_input_bytes }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.netOutputBytes')">{{ info.Stats.total_net_output_bytes }}</el-descriptions-item>
            </el-descriptions>

            <el-descriptions class="redis-info info-persistence" :title="$t('redis.persistence')" :column="3" size="small" border>
                <el-descriptions-item :label="$t('redis.aofEnable')">{{ info.Persistence?.aof_enabled || false }}</el-descriptions-item>
                <el-descriptions-item :label="$t('redis.loadingPersistence')">{{ info.Persistence?.loading || false }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, watch, toRefs, nextTick } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import ECharts from '@/components/echarts/ECharts.vue';
import { ECOption } from '@/components/echarts/config';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    visible: {
        type: Boolean,
    },
    title: {
        type: String,
    },
    info: {
        type: [Object],
        default: () => {},
    },
});

const emit = defineEmits(['update:visible', 'close']);

const state = reactive({
    dialogVisible: false,
    memInfo: {} as any,
    Keyspace: [] as any[],
    memOption: {},
});

const { dialogVisible, Keyspace } = toRefs(state);

watch(
    () => props.visible,
    (val) => {
        state.dialogVisible = val;
    }
);
watch(
    () => props.info,
    (info: any) => {
        state.memInfo = info['Memory'];
        if (state.memInfo) {
            initCharts();
        }
        if (info['Keyspace']) {
            let arr = [];
            for (let k in info['Keyspace']) {
                let data: any = { db: k };
                let d = info['Keyspace'][k].split(',');
                for (let f of d) {
                    let v = f.split('=');
                    data[v[0]] = v[1];
                }
                arr.push(data);
            }
            state.Keyspace = arr;
        }
    }
);

const initCharts = () => {
    nextTick(() => {
        initMemStats();
    });
};

const initMemStats = () => {
    let maxMem = state.memInfo.maxmemory === '0' ? state.memInfo.total_system_memory : state.memInfo.maxmemory;
    const data = [
        { name: t('redis.availableMemory'), value: maxMem - state.memInfo.used_memory },
        {
            name: t('redis.usedMemory'),
            value: state.memInfo.used_memory,
        },
    ];
    const option: ECOption = {
        title: {
            text: t('machine.memory'),
            textStyle: { fontSize: 14 },
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
                center: ['40%', '50%'], // 饼图位置，0: 左右；1: 上下
                avoidLabelOverlap: false,
                label: {
                    show: false,
                    position: 'center',
                },
                emphasis: {
                    label: {
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

const close = () => {
    emit('update:visible', false);
    emit('close');
};
</script>

<style lang="scss">
.redis-info {
    margin-top: 12px;
}

.row .title {
    font-size: 12px;
    color: #8492a6;
    margin-right: 6px;
}

.row .value {
    font-size: 12px;
    color: var(--el-color-success);
}
</style>
