<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :show-close="true" width="35%" @close="close()">
            <el-collapse>
                <el-collapse-item title="Server(Redis服务器的一般信息)" name="server">
                    <div class="row">
                        <span class="title">redis_version(版本):</span>
                        <span class="value">{{ info.Server.redis_version }}</span>
                    </div>
                    <div class="row">
                        <span class="title">tcp_port(端口):</span>
                        <span class="value">{{ info.Server.tcp_port }}</span>
                    </div>
                    <div class="row">
                        <span class="title">redis_mode(模式):</span>
                        <span class="value">{{ info.Server.redis_mode }}</span>
                    </div>
                    <div class="row">
                        <span class="title">os(宿主操作系统):</span>
                        <span class="value">{{ info.Server.os }}</span>
                    </div>
                    <div class="row">
                        <span class="title">uptime_in_days(运行天数):</span>
                        <span class="value">{{ info.Server.uptime_in_days }}</span>
                    </div>
                    <div class="row">
                        <span class="title">executable(可执行文件路径):</span>
                        <span class="value">{{ info.Server.executable }}</span>
                    </div>
                    <div class="row">
                        <span class="title">config_file(配置文件路径):</span>
                        <span class="value">{{ info.Server.config_file }}</span>
                    </div>
                </el-collapse-item>

                <el-collapse-item title="Clients(客户端连接)" name="client">
                    <div class="row">
                        <span class="title">connected_clients(已连接客户端数):</span>
                        <span class="value">{{ info.Clients.connected_clients }}</span>
                    </div>
                    <div class="row">
                        <span class="title">blocked_clients(正在等待阻塞命令客户端数):</span>
                        <span class="value">{{ info.Clients.blocked_clients }}</span>
                    </div>
                </el-collapse-item>
                <el-collapse-item title="Keyspace(key信息)" name="keyspace">
                    <div class="row" v-for="(value, key) in info.Keyspace" :key="key">
                        <span class="title">{{ key }}: </span>
                        <span class="value">{{ value }}</span>
                    </div>
                </el-collapse-item>

                <el-collapse-item title="Stats(统计)" name="state">
                    <div class="row">
                        <span class="title">total_commands_processed(总处理命令数):</span>
                        <span class="value">{{ info.Stats.total_commands_processed }}</span>
                    </div>
                    <div class="row">
                        <span class="title">instantaneous_ops_per_sec(当前qps):</span>
                        <span class="value">{{ info.Stats.instantaneous_ops_per_sec }}</span>
                    </div>
                    <div class="row">
                        <span class="title">total_net_input_bytes(网络入口流量字节数):</span>
                        <span class="value">{{ info.Stats.total_net_input_bytes }}</span>
                    </div>
                    <div class="row">
                        <span class="title">total_net_output_bytes(网络出口流量字节数):</span>
                        <span class="value">{{ info.Stats.total_net_output_bytes }}</span>
                    </div>
                    <div class="row">
                        <span class="title">expired_keys(过期key的总数量):</span>
                        <span class="value">{{ info.Stats.expired_keys }}</span>
                    </div>
                    <div class="row">
                        <span class="title">instantaneous_ops_per_sec(当前qps):</span>
                        <span class="value">{{ info.Stats.instantaneous_ops_per_sec }}</span>
                    </div>
                </el-collapse-item>

                <el-collapse-item title="Persistence(持久化)" name="persistence">
                    <div class="row">
                        <span class="title">aof_enabled(是否启用aof):</span>
                        <span class="value">{{ info.Persistence.aof_enabled }}</span>
                    </div>
                    <div class="row">
                        <span class="title">loading(是否正在载入持久化文件):</span>
                        <span class="value">{{ info.Persistence.loading }}</span>
                    </div>
                </el-collapse-item>

                <el-collapse-item title="Cluster(集群)" name="cluster">
                    <div class="row">
                        <span class="title">cluster_enabled(是否启用集群模式):</span>
                        <span class="value">{{ info.Cluster.cluster_enabled }}</span>
                    </div>
                </el-collapse-item>

                <el-collapse-item title="Memory(内存消耗相关信息)" name="memory">
                    <div class="row">
                        <span class="title">used_memory(分配内存总量):</span>
                        <span class="value">{{ info.Memory.used_memory_human }}</span>
                    </div>
                    <div class="row">
                        <span class="title">maxmemory(最大内存配置):</span>
                        <span class="value">{{ info.Memory.maxmemory }}</span>
                    </div>
                    <div class="row">
                        <span class="title">used_memory_rss(已分配的内存总量，操作系统角度):</span>
                        <span class="value">{{ info.Memory.used_memory_rss_human }}</span>
                    </div>
                    <div class="row">
                        <span class="title">mem_fragmentation_ratio(used_memory_rss和used_memory 之间的比率):</span>
                        <span class="value">{{ info.Memory.mem_fragmentation_ratio }}</span>
                    </div>
                    <div class="row">
                        <span class="title">used_memory_peak(内存消耗峰值):</span>
                        <span class="value">{{ info.Memory.used_memory_peak_human }}</span>
                    </div>
                    <div class="row">
                        <span class="title">total_system_memory(主机总内存):</span>
                        <span class="value">{{ info.Memory.total_system_memory_human }}</span>
                    </div>
                </el-collapse-item>

                <el-collapse-item title="CPU" name="cpu">
                    <div class="row">
                        <span class="title">used_cpu_sys(由Redis服务器消耗的系统CPU):</span>
                        <span class="value">{{ info.CPU.used_cpu_sys }}</span>
                    </div>
                    <div class="row">
                        <span class="title">used_cpu_user(由Redis服务器消耗的用户CPU):</span>
                        <span class="value">{{ info.CPU.used_cpu_user }}</span>
                    </div>
                    <div class="row">
                        <span class="title">used_cpu_sys_children(由后台进程消耗的系统CPU):</span>
                        <span class="value">{{ info.CPU.used_cpu_sys_children }}</span>
                    </div>
                    <div class="row">
                        <span class="title">used_cpu_user_children(由后台进程消耗的用户CPU):</span>
                        <span class="value">{{ info.CPU.used_cpu_user_children }}</span>
                    </div>
                </el-collapse-item>
            </el-collapse>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { defineComponent, reactive, watch, toRefs } from 'vue';

export default defineComponent({
    name: 'Info',
    props: {
        visible: {
            type: Boolean,
        },
        title: {
            type: String,
        },
        info: {
            type: [Boolean, Object],
        },
    },
    setup(props: any, { emit }) {
        const state = reactive({
            dialogVisible: false,
        });

        watch(
            () => props.visible,
            (val) => {
                state.dialogVisible = val;
            }
        );
        
        const close = () => {
            emit('update:visible', false);
            emit('close');
        };

        return {
            ...toRefs(state),
            close,
        };
    },
});
</script>

<style>
.row .title {
    font-size: 12px;
    color: #8492a6;
    margin-right: 6px;
}

.row .value {
    font-size: 12px;
    color: black;
}
</style>
