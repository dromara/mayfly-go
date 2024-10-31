<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :show-close="true" width="1000px" @close="close()">
            <el-row :gutter="20">
                <el-col :lg="16" :md="16">
                    <el-descriptions class="redis-info info-server" title="Redis服务器信息" :column="3" size="small" border>
                        <el-descriptions-item label="版本">{{ info.Server.redis_version }}</el-descriptions-item>
                        <el-descriptions-item label="端口">{{ info.Server.tcp_port }}</el-descriptions-item>
                        <el-descriptions-item label="PID">{{ info.Server.process_id }}</el-descriptions-item>
                        <el-descriptions-item label="模式">{{ info.Server.redis_mode }}</el-descriptions-item>
                        <el-descriptions-item label="操作系统">{{ info.Server.os }}</el-descriptions-item>
                        <el-descriptions-item label="运行天数">{{ info.Server.uptime_in_days }}</el-descriptions-item>
                        <el-descriptions-item label="可执行文件路径">{{ info.Server.executable }}</el-descriptions-item>
                        <el-descriptions-item label="配置文件路径">{{ info.Server.config_file }}</el-descriptions-item>
                    </el-descriptions>
                </el-col>
                <el-col :lg="8" :md="8" class="redis-info">
                    <ECharts height="150" width="360" :option="state.memOption" />
                </el-col>
            </el-row>

            <el-row :gutter="20">
                <el-col :lg="12" :md="12">
                    <el-descriptions class="redis-info info-cluster" title="节点信息" :column="3" size="small" border>
                        <el-descriptions-item label="是否启用集群模式">{{ info.Cluster.cluster_enabled }}</el-descriptions-item>
                        <el-descriptions-item label="DB总数">{{ info.Cluster.databases }}</el-descriptions-item>
                        <el-descriptions-item label="节点总数">{{ info.Cluster.nodecount }}</el-descriptions-item>
                    </el-descriptions>
                </el-col>

                <el-col :lg="12" :md="12">
                    <el-descriptions class="redis-info info-client" title="客户端连接" :column="3" size="small" border>
                        <el-descriptions-item label="已连接客户端数">{{ info.Clients.connected_clients }}</el-descriptions-item>
                        <el-descriptions-item label="正在等待阻塞命令客户端数">{{ info.Clients.blocked_clients }}</el-descriptions-item>
                    </el-descriptions>
                </el-col>
            </el-row>

            <el-descriptions class="redis-info info-memory" title="CPU" :column="2" size="small" border>
                <el-descriptions-item label="系统CPU">{{ info.CPU.used_cpu_sys }}</el-descriptions-item>
                <el-descriptions-item label="用户CPU">{{ info.CPU.used_cpu_user }}</el-descriptions-item>
                <el-descriptions-item label="后台系统CPU">{{ info.CPU.used_cpu_sys_children }}</el-descriptions-item>
                <el-descriptions-item label="后台用户CPU">{{ info.CPU.used_cpu_user_children }}</el-descriptions-item>
            </el-descriptions>

            <el-row :gutter="20" class="redis-info">
                <el-col :lg="24" :md="24">
                    <span style="font-size: 14px; font-weight: 700">键值统计</span>
                    <el-table :data="Keyspace" stripe max-height="250" style="width: 100%" border>
                        <el-table-column prop="db" label="数据库" min-width="100" show-overflow-tooltip> </el-table-column>
                        <el-table-column prop="keys" label="keys" min-width="70" show-overflow-tooltip> </el-table-column>
                        <el-table-column prop="expires" label="expires" min-width="70" show-overflow-tooltip> </el-table-column>
                        <el-table-column prop="avg_ttl" label="avg_ttl" min-width="70" show-overflow-tooltip> </el-table-column>
                    </el-table>
                </el-col>
            </el-row>

            <el-descriptions class="redis-info info-state" title="统计信息" :column="3" size="small" border>
                <el-descriptions-item label="总处理命令数">{{ info.Stats.total_commands_processed }}</el-descriptions-item>
                <el-descriptions-item label="当前qps">{{ info.Stats.instantaneous_ops_per_sec }}</el-descriptions-item>
                <el-descriptions-item label="过期key的总数量">{{ info.Stats.expired_keys }}</el-descriptions-item>
                <el-descriptions-item label="网络入口流量字节数">{{ info.Stats.total_net_input_bytes }}</el-descriptions-item>
                <el-descriptions-item label="网络出口流量字节数">{{ info.Stats.total_net_output_bytes }}</el-descriptions-item>
            </el-descriptions>

            <el-descriptions class="redis-info info-persistence" title="持久化" :column="3" size="small" border>
                <el-descriptions-item label="是否启用aof">{{ info.Persistence?.aof_enabled || false }}</el-descriptions-item>
                <el-descriptions-item label="是否正在载入持久化文件">{{ info.Persistence?.loading || false }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, watch, toRefs, nextTick } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import ECharts from '@/components/echarts/ECharts.vue';
import { ECOption } from '@/components/echarts/config';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    title: {
        type: String,
    },
    info: {
        type: [Boolean, Object],
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
                let data = { db: k };
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
        { name: '可用内存：', value: maxMem - state.memInfo.used_memory },
        {
            name: '已用内存：',
            value: state.memInfo.used_memory,
        },
    ];
    const option: ECOption = {
        title: {
            text: '内存',
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
                name: '内存',
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
