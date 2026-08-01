<template>
    <el-tabs v-model="state.tabName" type="card" class="es-dashboard-tabs">
        <el-tab-pane name="idxManage" :label="t('es.opIndex')">
            <div class="idx-manage-content">
                <EsIndexManage :instId="props.instId" @viewData="onViewIndexData" />
            </div>
        </el-tab-pane>

        <el-tab-pane name="dataManage" :label="t('es.opDataManage')">
            <div class="idx-manage-content">
                <EsIndexData ref="esIndexDataRef" :instId="props.instId" />
            </div>
        </el-tab-pane>

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
                        <el-select v-model="state.analyze.idxName" filterable clearable :teleported="false" @change="onSelectIdxField">
                            <el-option v-for="idx in state.idxFields" :key="idx.name" :value="idx.name" :label="idx.name" />
                        </el-select>
                    </el-form-item>
                    <el-form-item :label="t('es.dashboard.field')" required prop="field">
                        <el-select v-model="state.analyze.field" filterable clearable :teleported="false">
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
import { formatByteSize } from '@/common/utils/format';
import { esApi } from '@/views/ops/es/api';
import type { EsAnalyzeRes, EsAnalyzeToken, EsClusterStateRes, EsIdxField, EsNameValue, EsNodeStats, EsNodesStatsRes } from '@/views/ops/es/types';
import dayjs from 'dayjs';
import { defineAsyncComponent, nextTick, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

const EsIndexData = defineAsyncComponent(() => import('./EsIndexData.vue'));
const EsIndexManage = defineAsyncComponent(() => import('./EsIndexManage.vue'));

const { t } = useI18n();

interface Props {
    instId: number;
}
const props = defineProps<Props>();

const analyzeFormRef = ref();
const esIndexDataRef = ref();

const onViewIndexData = async (idxName: string) => {
    state.tabName = 'dataManage';
    await nextTick();
    esIndexDataRef.value?.selectIndex(idxName);
};

const state = reactive({
    tabName: 'idxManage',
    instInfo: [] as EsNameValue[],
    clusterHealth: [] as EsNameValue[],
    nodesStats: { _nodes: { total: 0, successful: 0, failed: 0 }, nodes: [] as EsNodeStats[] },
    idxFields: [] as EsIdxField[],
    nodesStatsLoading: false,
    instInfoLoading: false,
    clusterHealthLoading: false,
    clusterStateLoading: false,
    analyze: {
        loading: false,
        idxName: '',
        fields: [] as string[],
        field: '',
        text: '',
        tokens: [] as EsAnalyzeToken[],
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

function flattenObject(obj: Record<string, unknown>, parentKey = '', result: Record<string, unknown> = {}): Record<string, unknown> {
    for (const key in obj) {
        if (Object.prototype.hasOwnProperty.call(obj, key)) {
            const newKey = parentKey ? `${parentKey}.${key}` : key;
            if (typeof obj[key] === 'object' && obj[key] !== null && !Array.isArray(obj[key])) {
                flattenObject(obj[key] as Record<string, unknown>, newKey, result);
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
    const res = await esApi.proxyReq<EsNodesStatsRes>('get', props.instId, '/_nodes/stats/os,jvm,indices,transport,fs');
    state.nodesStats._nodes = res._nodes;
    const nodes: EsNodeStats[] = [];
    for (const k in res.nodes) {
        nodes.push({ ...res.nodes[k], key: k });
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
    const res = await esApi.proxyReq<EsClusterStateRes>('get', props.instId, '/_cluster/state');

    const idxFields: EsIdxField[] = [];

    for (const k in res.metadata.indices) {
        // 过滤系统索引
        if (k.indexOf('.') >= 0) {
            continue;
        }
        const properties = res.metadata.indices[k]?.mappings?._doc?.properties || {};
        const fields: string[] = [];
        for (const fk in properties) {
            const f = properties[fk];
            // long字段类型不支持分析
            if (f.type === 'long' || f.type === 'date') {
                continue;
            }

            // 添加字段
            fields.push(fk);

            // 如果有子字段，则添加子字段
            if (f.fields) {
                for (const sfk in f.fields) {
                    fields.push(`${fk}.${sfk}`);
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
    state.analyze.fields = state.idxFields.find((item) => item.name === state.analyze.idxName)?.fields || [];
    state.analyze.field = '';
};

const onAnalyze = async () => {
    await analyzeFormRef.value?.validate();
    state.analyze.loading = true;

    setTimeout(() => {
        state.analyze.loading = false;
    }, 2000);

    const res = await esApi.proxyReq<EsAnalyzeRes>('post', props.instId, `/${state.analyze.idxName}/_analyze`, {
        field: state.analyze.field,
        text: state.analyze.text,
    });
    state.analyze.tokens = res.tokens || [];
    state.analyze.loading = false;
};
</script>

<style scoped lang="scss">
.es-dashboard-tabs {
    height: 100%;
    display: flex;
    flex-direction: column;

    :deep(.el-tabs__content) {
        flex: 1;
        min-height: 0;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    :deep(.el-tab-pane) {
        flex: 1;
        min-height: 0;
    }
}

.idx-manage-content {
    height: 100%;
}

.nodes-num {
    font-size: 20px;
}
</style>
