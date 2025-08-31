<template>
    <el-tabs v-model="state.tabName" type="card">
        <el-tab-pane name="nodesStats" v-loading="state.nodesStatsLoading" style="height: calc(100vh - 200px); overflow-y: auto">
            <template #label>
                {{ t('es.dashboard.nodes') }}
                <el-button v-if="state.tabName === 'nodesStats'" icon="refresh" @click="fetchNodesStats" link type="primary" />
            </template>
            <el-descriptions class="nodes-num" :column="3" border>
                <el-descriptions-item label="total">
                    {{ state.nodesStats._nodes?.total }}
                </el-descriptions-item>
                <el-descriptions-item label="successful">
                    {{ state.nodesStats._nodes?.successful }}
                </el-descriptions-item>
                <el-descriptions-item label="failed">
                    {{ state.nodesStats._nodes?.failed }}
                </el-descriptions-item>
            </el-descriptions>

            <el-tabs>
                <el-tab-pane :label="node.name" v-for="node in state.nodesStats.nodes" :key="node.key">
                    <el-card class="mt-1">
                        <el-form label-width="100">
                            <el-form-item label="ID">
                                <el-tag size="small" type="primary">{{ node.key }}</el-tag>
                            </el-form-item>

                            <el-form-item label="IP">
                                <el-tag size="small" type="primary">{{ node.ip }}</el-tag>
                            </el-form-item>

                            <el-form-item label="TIME">
                                <el-tag size="small" type="primary">{{ dayjs(node.timestamp).format('YYYY-MM-DD HH:mm:ss') }}</el-tag>
                            </el-form-item>

                            <el-form-item label="Roles">
                                <el-space wrap>
                                    <el-tag v-for="r in node.roles" :key="r" type="success">{{ r }}</el-tag>
                                </el-space>
                            </el-form-item>

                            <el-form-item label="Docs">
                                <el-space>
                                    <el-tag type="warning">count: {{ node.indices.docs.count }}</el-tag>
                                    <el-tag type="info">deleted: {{ node.indices.docs.deleted }}</el-tag>
                                    <el-tag type="primary">{{ formatByteSize(node.indices.store.size_in_bytes) }}</el-tag>
                                </el-space>
                            </el-form-item>

                            <el-form-item :label="t('es.dashboard.sysMem')">
                                {{ formatByteSize(node.os.mem.used_in_bytes) }} / {{ formatByteSize(node.os.mem.total_in_bytes) }}
                                <el-progress
                                    striped
                                    striped-flow
                                    :duration="50"
                                    class="w-full"
                                    :percentage="node.os.mem.used_percent"
                                    :color="getPercentColor(node.os.mem.used_percent)"
                                />
                            </el-form-item>

                            <el-form-item :label="t('es.dashboard.jvmMem')">
                                {{ formatByteSize(node.jvm.mem.heap_used_in_bytes) }} / {{ formatByteSize(node.jvm.mem.heap_max_in_bytes) }}
                                <el-progress
                                    striped
                                    striped-flow
                                    :duration="50"
                                    class="w-full"
                                    :percentage="node.jvm.mem.heap_used_percent"
                                    :color="getPercentColor(node.jvm.mem.heap_used_percent)"
                                />
                            </el-form-item>

                            <el-form-item label="CPU">
                                <el-progress
                                    striped
                                    striped-flow
                                    :duration="50"
                                    class="w-full"
                                    :percentage="node.os.cpu.percent"
                                    :color="getPercentColor(node.os.cpu.percent)"
                                />
                            </el-form-item>

                            <el-form-item :label="t('es.dashboard.fileSystem')">
                                {{ formatByteSize(node.fs.total.total_in_bytes - node.fs.total.free_in_bytes) }} /
                                {{ formatByteSize(node.fs.total.total_in_bytes) }}
                                <el-progress
                                    striped
                                    striped-flow
                                    :duration="50"
                                    class="w-full"
                                    :percentage="
                                        Math.round(((node.fs.total.total_in_bytes - node.fs.total.free_in_bytes) * 100) / node.fs.total.total_in_bytes)
                                    "
                                    :color="
                                        getPercentColor(((node.fs.total.total_in_bytes - node.fs.total.free_in_bytes) * 100) / node.fs.total.total_in_bytes)
                                    "
                                />
                            </el-form-item>
                        </el-form>
                    </el-card>
                </el-tab-pane>
            </el-tabs>
        </el-tab-pane>

        <el-tab-pane
            name="instInfo"
            v-loading="state.instInfoLoading"
            :label="t('es.dashboard.instInfo')"
            style="height: calc(100vh - 200px); overflow-y: auto"
        >
            <el-card shadow="hover">
                <el-descriptions :column="1" border>
                    <el-descriptions-item label-align="left" align="right" :label="item.name" v-for="item in state.instInfo" :key="item.name">
                        {{ item.value }}
                    </el-descriptions-item>
                </el-descriptions>
            </el-card>
        </el-tab-pane>
        <el-tab-pane
            name="clusterHealth"
            v-loading="state.clusterHealthLoading"
            :label="t('es.dashboard.clusterHealth')"
            style="height: calc(100vh - 200px); overflow-y: auto"
        >
            <el-card shadow="always">
                <el-descriptions :column="1" border>
                    <el-descriptions-item label-align="left" align="right" :label="item.name" v-for="item in state.clusterHealth" :key="item.name">
                        {{ item.value }}
                    </el-descriptions-item>
                </el-descriptions>
            </el-card>
        </el-tab-pane>

        <el-tab-pane
            name="analyze"
            v-loading="state.clusterStateLoading"
            :label="t('es.dashboard.analyze')"
            style="height: calc(100vh - 200px); overflow-y: auto"
        >
            <el-card class="h-full">
                <el-form :model="state.analyze" ref="analyzeFormRef" label-position="right" label-width="100">
                    <el-form-item :label="t('es.dashboard.idxName')" required prop="idxName">
                        <el-select v-model="state.analyze.idxName" filterable clearable @change="onSelectIdxField">
                            <el-option v-for="idx in state.idxFields" :key="idx.name" :value="idx.name" :label="idx.name" />
                        </el-select>
                    </el-form-item>
                    <el-form-item :label="t('es.dashboard.field')" required prop="field">
                        <el-select v-model="state.analyze.field" filterable clearable>
                            <el-option v-for="field in state.analyze.fields" :key="field" :value="field" :label="field" />
                        </el-select>
                    </el-form-item>
                    <el-form-item :label="t('es.dashboard.text')" required prop="text">
                        <el-input type="textarea" :rows="5" v-model="state.analyze.text" />
                    </el-form-item>
                </el-form>
                <el-button @click="onAnalyze" :loading="state.analyze.loading">{{ t('es.dashboard.startAnalyze') }}</el-button>
                <el-table :data="state.analyze.tokens" style="height: calc(100vh - 500px)" stripe size="small" :v-loading="true">
                    <el-table-column label="token" prop="token" />
                    <el-table-column label="position" prop="position" />
                    <el-table-column label="start_offset" prop="start_offset" />
                    <el-table-column label="end_offset" prop="end_offset" />
                    <el-table-column label="type" prop="type" />
                </el-table>
            </el-card>
        </el-tab-pane>
    </el-tabs>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { nextTick, onMounted, reactive, ref, watch } from 'vue';
import { esApi } from '@/views/ops/es/api';
import { formatByteSize } from '@/common/utils/format';
import dayjs from 'dayjs';

const { t } = useI18n();

interface Props {
    instId: any;
}
const props = defineProps<Props>();

const analyzeFormRef = ref();

const state = reactive({
    tabName: 'nodesStats',
    instInfo: [] as any[],
    clusterHealth: [] as any[],
    nodesStats: { _nodes: {} as any, nodes: [] as any[] } as any,
    idxFields: [] as any[],
    nodesStatsLoading: false,
    instInfoLoading: false,
    clusterHealthLoading: false,
    clusterStateLoading: false,
    analyze: {
        loading: false,
        idxName: '',
        fields: [],
        field: '',
        text: '',
        tokens: [],
    },
});

onMounted(async () => {
    await nextTick(async () => {
        await fetchNodesStats();
    });
});

watch(
    () => state.tabName,
    async (val) => {
        switch (val) {
            case 'instInfo':
                return await fetchInstInfo();
            case 'clusterHealth':
                return await fetchClusterHealth();
            case 'nodesStats':
                return await fetchNodesStats();
            case 'analyze':
                await fetchClusterState();
                return;
        }
    }
);

const fetchInstInfo = async () => {
    state.instInfoLoading = true;
    state.instInfo = [];
    let res = await esApi.proxyReq('get', props.instId, '/');
    let fo = flattenObject(res);
    for (const it in fo) {
        state.instInfo.push({
            name: it,
            value: fo[it],
        });
    }
    state.instInfoLoading = false;

    // key 排序
    state.instInfo = state.instInfo.sort((a, b) => a.name.localeCompare(b.name));
};

function flattenObject(obj: Record<string, any>, parentKey = '', result: Record<string, any> = {}): Record<string, any> {
    for (const key in obj) {
        if (obj.hasOwnProperty(key)) {
            const newKey = parentKey ? `${parentKey}.${key}` : key;
            if (typeof obj[key] === 'object' && obj[key] !== null && !Array.isArray(obj[key])) {
                flattenObject(obj[key], newKey, result);
            } else {
                result[newKey] = obj[key];
            }
        }
    }
    return result;
}

const fetchClusterHealth = async () => {
    state.clusterHealthLoading = true;
    state.clusterHealth = [];
    let res = await esApi.proxyReq('get', props.instId, '/_cluster/health');
    let fo = flattenObject(res);
    for (const it in fo) {
        state.clusterHealth.push({
            name: it,
            value: fo[it],
        });
    }
    state.clusterHealthLoading = false;

    // key 排序
    state.clusterHealth = state.clusterHealth.sort((a, b) => a.name.localeCompare(b.name));
};

const fetchNodesStats = async () => {
    state.nodesStatsLoading = true;
    let res = await esApi.proxyReq('get', props.instId, '/_nodes/stats/os,jvm,indices,transport,fs');
    state.nodesStats._nodes = res._nodes;
    let nodes = [] as any[];
    for (let k in res.nodes) {
        let node = res.nodes[k];
        node.key = k;
        nodes.push(node);
    }
    state.nodesStats.nodes = nodes.sort((a, b) => a.name.localeCompare(b.name));

    // 以node名排序
    state.nodesStatsLoading = false;
    // id
    // ip
    // name
    // roles
    // 系统内存  饼图  os.mem.total_in_bytes os.mem.used_in_bytes  os.mem.used_percent
    // 系统cpu使用率  饼图 os.cpu.percent
    // jvm内存 饼图 jvm.mem.heap_max_in_bytes jvm.mem.heap_used_in_bytes  jvm.mem.heap_used_percent
    // 存储空间占用信息 饼图 fs.total.total_in_bytes fs.total.free_in_bytes
    // 索引文档数 indices.docs.count
    // 索引占用 indices.store.size_in_bytes
    // 总分片数量 indices.shard_stats.total_count
    // 网络流量   transport.rx_size_in_bytes   transport.tx_size_in_bytes
};

const fetchClusterState = async () => {
    state.clusterStateLoading = true;
    const res = await esApi.proxyReq('get', props.instId, '/_cluster/state');

    const idxFields = [];

    for (let k in res.metadata.indices) {
        // 过滤系统索引
        if (k.indexOf('.') >= 0) {
            continue;
        }
        let properties = res.metadata.indices[k]?.mappings?._doc?.properties || {};
        let fields = [];
        for (let k in properties) {
            let f = properties[k];
            // long字段类型不支持分析
            if (f.type === 'long' || f.type === 'date') {
                continue;
            }

            // 添加字段
            fields.push(k);

            // 如果有子字段，则添加子字段
            if (f.fields) {
                for (let fk in f.fields) {
                    fields.push(`${k}.${fk}`);
                }
            }
        }

        idxFields.push({
            name: k,
            fields: fields.sort(),
        });
    }

    // 索引字段信息
    state.idxFields = idxFields.sort((a, b) => a.name.localeCompare(b.name));

    state.clusterStateLoading = false;
};

const getPercentColor = (percent: number) => {
    if (percent < 60) {
        return '#67c23a';
    } else if (percent < 80) {
        return '#e6a23c';
    } else {
        return '#f56c6c';
    }
};

const onSelectIdxField = () => {
    state.analyze.fields = state.idxFields.find((item: any) => item.name === state.analyze.idxName)?.fields || [];
    state.analyze.field = '';
};

const onAnalyze = async () => {
    await analyzeFormRef.value.validate();
    state.analyze.loading = true;

    setTimeout(() => {
        state.analyze.loading = false;
    }, 2000);

    let res = await esApi.proxyReq('post', props.instId, `/${state.analyze.idxName}/_analyze`, {
        field: state.analyze.field,
        text: state.analyze.text,
    });
    state.analyze.tokens = res.tokens;
    state.analyze.loading = false;
};
</script>

<style scoped lang="scss">
.nodes-num {
    font-size: 20px;
}
</style>
