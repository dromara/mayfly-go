<template>
    <div class="system-info-container">
        <!-- 核心指标卡片 -->
        <el-row :gutter="16" class="metrics-row">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
                <el-card shadow="hover" class="metric-card">
                    <div class="metric-icon version">
                        <SvgIcon name="InfoFilled" :size="14" />
                    </div>
                    <div class="metric-content">
                        <div class="metric-label">{{ $t('milvus.versionInfo') }}</div>
                        <div class="metric-value" :title="version || $t('milvus.notFetched')">
                            {{ version || $t('milvus.notFetched') }}
                        </div>
                    </div>
                </el-card>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
                <el-card shadow="hover" class="metric-card" :class="{ 'is-healthy': healthStatus, 'is-unhealthy': healthStatus === false }">
                    <div class="metric-icon" :class="healthStatus ? 'healthy' : 'unhealthy'">
                        <SvgIcon name="FirstAidKit" :size="14" />
                    </div>
                    <div class="metric-content">
                        <div class="metric-label">{{ $t('milvus.healthStatus') }}</div>
                        <div class="metric-value">
                            <el-tag v-if="healthStatus !== null" :type="healthStatus ? 'success' : 'danger'" size="small">
                                {{ healthStatus ? $t('milvus.healthy') : $t('milvus.unhealthy') }}
                            </el-tag>
                            <span v-else class="text-gray">{{ $t('milvus.notChecked') }}</span>
                        </div>
                    </div>
                </el-card>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
                <el-card shadow="hover" class="metric-card">
                    <div class="metric-icon database">
                        <SvgIcon name="Collection" :size="14" />
                    </div>
                    <div class="metric-content">
                        <div class="metric-label">{{ $t('milvus.databaseCount') }}</div>
                        <div class="metric-value">{{ stats.databaseCount }}</div>
                    </div>
                </el-card>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
                <el-card shadow="hover" class="metric-card">
                    <div class="metric-icon collection">
                        <SvgIcon name="DocumentCopy" :size="14" />
                    </div>
                    <div class="metric-content">
                        <div class="metric-label">{{ $t('milvus.collectionCount') }}</div>
                        <div class="metric-value">{{ stats.collectionCount }}</div>
                    </div>
                </el-card>
            </el-col>
        </el-row>

        <!-- 健康状态详情 -->
        <el-card v-if="healthDetails.length > 0" shadow="hover" class="health-detail-card">
            <template #header>
                <div class="card-header">
                    <span>
                        <SvgIcon name="Warning" :size="14" class="mr-1" />
                        {{ $t('milvus.healthDetail') }}
                    </span>
                    <el-tag v-if="healthStatus" type="success" size="small">{{ $t('milvus.allNormal') }}</el-tag>
                    <el-tag v-else type="danger" size="small">{{ healthDetails.length }} {{ $t('milvus.issuesFound') }}</el-tag>
                </div>
            </template>
            <el-timeline>
                <el-timeline-item
                    v-for="(item, index) in healthDetails"
                    :key="index"
                    :type="item.healthy ? 'success' : 'danger'"
                    :icon="item.healthy ? 'Check' : 'Close'"
                >
                    <div class="health-item">
                        <div class="health-item-name">{{ item.name || item }}</div>
                        <div v-if="item.message" class="health-item-message">{{ item.message }}</div>
                    </div>
                </el-timeline-item>
            </el-timeline>
        </el-card>

        <!-- 资源组信息 -->
        <el-card v-if="resourceGroups.length > 0" shadow="hover" class="info-card">
            <template #header>
                <div class="card-header">
                    <span>
                        <SvgIcon name="Box" :size="14" class="mr-1" />
                        {{ $t('milvus.resourceGroup') }}
                    </span>
                    <el-tag size="small">{{ resourceGroups.length }} {{ $t('milvus.total') }}</el-tag>
                </div>
            </template>
            <el-collapse v-model="activeResourceGroups">
                <el-collapse-item v-for="rg in resourceGroups" :key="rg.Name || rg.name" :name="rg.Name || rg.name">
                    <template #title>
                        <div class="rg-collapse-title">
                            <span class="rg-name">{{ rg.Name || rg.name }}</span>
                            <el-tag size="small" type="info" class="rg-node-tag">
                                {{ $t('milvus.nodeCount') }}: {{ (rg.Nodes ?? rg.nodes)?.length || 0 }}
                            </el-tag>
                            <el-tag size="small" :type="getLoadedReplicaCount(rg) > 0 ? 'success' : 'info'" class="rg-replica-tag">
                                {{ $t('milvus.loadedCollections') }}: {{ getLoadedReplicaCount(rg) }}
                            </el-tag>
                        </div>
                    </template>

                    <el-descriptions :column="2" border size="small">
                        <el-descriptions-item :label="$t('milvus.capacity')">
                            <el-progress :percentage="getCapacityPercentage(rg)" :color="getCapacityColor(rg)" :stroke-width="8" style="width: 150px" />
                            <span class="ml-2">{{ getCapacityPercentage(rg) }}%</span>
                        </el-descriptions-item>
                        <el-descriptions-item :label="$t('milvus.availableNodes')">
                            {{ rg.NumAvailableNode ?? rg.numAvailableNode ?? 0 }} / {{ rg.Capacity ?? rg.capacity ?? 0 }}
                        </el-descriptions-item>
                        <el-descriptions-item :label="$t('milvus.nodeConfig')" :span="2" v-if="rg.Config || rg.config">
                            <div class="config-info">
                                <el-tag size="small" class="mr-1">
                                    {{ $t('milvus.requestNodeNum') }}: {{ rg.Config?.Requests?.NodeNum ?? rg.config?.requests?.nodeNum ?? 0 }}
                                </el-tag>
                                <el-tag size="small">
                                    {{ $t('milvus.limitNodeNum') }}: {{ rg.Config?.Limits?.NodeNum ?? rg.config?.limits?.nodeNum ?? 1000000 }}
                                </el-tag>
                            </div>
                        </el-descriptions-item>
                    </el-descriptions>

                    <!-- 节点详情 -->
                    <div v-if="((rg.Nodes ?? rg.nodes)?.length ?? 0) > 0" class="rg-section">
                        <div class="rg-section-title">{{ $t('milvus.nodeDetails') }}</div>
                        <el-table :data="rg.Nodes ?? rg.nodes" size="small" border>
                            <el-table-column prop="NodeID" :label="$t('milvus.nodeId')" min-width="80" />
                            <el-table-column prop="Address" :label="$t('milvus.nodeAddress')" min-width="150" show-overflow-tooltip />
                            <el-table-column prop="HostName" :label="$t('milvus.nodeHostname')" min-width="120" show-overflow-tooltip />
                        </el-table>
                    </div>

                    <!-- 已加载 Collection -->
                    <div v-if="getLoadedReplicaCount(rg) > 0" class="rg-section">
                        <div class="rg-section-title">{{ $t('milvus.loadedCollections') }} ({{ getLoadedReplicaCount(rg) }})</div>
                        <div class="collection-tags">
                            <el-tag
                                v-for="(count, name) in rg.NumLoadedReplica ?? rg.numLoadedReplica ?? {}"
                                :key="name"
                                size="small"
                                type="success"
                                class="collection-tag"
                            >
                                {{ name }}: {{ count }} {{ $t('milvus.replica') }}
                            </el-tag>
                        </div>
                    </div>
                </el-collapse-item>
            </el-collapse>
        </el-card>

        <!-- 数据库列表 -->
        <el-card v-if="databases.length > 0" shadow="hover" class="info-card">
            <template #header>
                <div class="card-header">
                    <span>
                        <el-icon class="mr-1"><Collection /></el-icon>
                        {{ $t('milvus.databaseManagement') }}
                    </span>
                    <el-tag size="small">{{ databases.length }} {{ $t('milvus.total') }}</el-tag>
                </div>
            </template>
            <el-table :data="databases" stripe size="small" border>
                <el-table-column prop="name" :label="$t('milvus.dbName')" min-width="150" show-overflow-tooltip />
                <el-table-column prop="id" :label="$t('milvus.dbId')" min-width="100" align="center" />
                <el-table-column prop="create_time" :label="$t('milvus.createTime')" min-width="180" align="center" />
                <el-table-column :label="$t('milvus.properties')" min-width="200">
                    <template #default="{ row }">
                        <el-tag v-for="(value, key) in row.properties" :key="key" size="small" class="mr-1 mb-1"> {{ key }}: {{ value }} </el-tag>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <!-- 系统配置信息 -->
        <el-card v-if="systemConfig.length > 0" shadow="hover" class="info-card">
            <template #header>
                <div class="card-header">
                    <span>
                        <SvgIcon name="Setting" :size="14" class="mr-1" />
                        {{ $t('milvus.systemConfig') }}
                    </span>
                </div>
            </template>
            <el-descriptions :column="2" border size="small">
                <el-descriptions-item v-for="(config, index) in systemConfig" :key="index" :label="config.name" label-align="left" align="right">
                    <el-tag v-if="isBoolean(config.value)" :type="config.value === 'true' ? 'success' : 'info'" size="small">
                        {{ config.value }}
                    </el-tag>
                    <span v-else>{{ config.value }}</span>
                </el-descriptions-item>
            </el-descriptions>
        </el-card>
    </div>
</template>

<script setup lang="ts">
import { Msg } from '@/hooks/useI18n';
import SvgIcon from '@/components/svg-icon/index.vue';
import { onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { milvusApi } from '../api';
import type { IDatabase, IResourceGroup, IMilvusHealthStatus, MilvusHealthDetail } from '../types';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';

const { t } = useI18n();

const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');

const loading = ref(false);
const versionLoading = ref(false);
const healthLoading = ref(false);
const version = ref('');
const healthStatus = ref<boolean | null>(null);
const healthDetails = ref<MilvusHealthDetail[]>([]);
const databases = ref<IDatabase[]>([]);
const resourceGroups = ref<IResourceGroup[]>([]);
const activeResourceGroups = ref<string[]>([]);
const systemConfig = ref<Record<string, unknown>[]>([]);
const stats = ref({
    databaseCount: 0,
    collectionCount: 0,
    userCount: 0,
    roleCount: 0,
});

const loadVersion = async () => {
    versionLoading.value = true;
    try {
        const res = await milvusApi.getVersion(props.milvusId);
        version.value = res || t('milvus.unknown');
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message || 'milvus.getVersionFailed' : 'milvus.getVersionFailed');
    } finally {
        versionLoading.value = false;
    }
};

const checkHealth = async () => {
    healthLoading.value = true;
    try {
        const res = await milvusApi.checkHealth(props.milvusId);
        let data: IMilvusHealthStatus;
        try {
            data = typeof res === 'string' ? JSON.parse(res) : res;
        } catch (e) {
            data = { IsHealthy: false, Reasons: [{ name: 'Parse Error', message: String(res) }] };
        }

        healthStatus.value = data.isHealthy ?? data.IsHealthy ?? false;

        // 处理健康详情
        const reasons = data.reasons || data.Reasons || [];
        if (Array.isArray(reasons)) {
            healthDetails.value = reasons.map((reason) => {
                if (typeof reason === 'string') {
                    return { name: reason, healthy: false };
                }
                return {
                    name: reason.name || reason.Name || String(reason),
                    message: reason.message || reason.Message || '',
                    healthy: reason.healthy !== undefined ? reason.healthy : !reason.message,
                };
            });
        } else {
            healthDetails.value = [];
        }
    } catch (error: unknown) {
        const errMsg = error instanceof Error ? error.message : 'checkHealthFailed';
        healthStatus.value = false;
        healthDetails.value = [{ name: 'Connection Error', message: errMsg, healthy: false }];
        Msg.error(errMsg || 'milvus.checkHealthFailed');
    } finally {
        healthLoading.value = false;
    }
};

const loadDatabases = async () => {
    try {
        const res = await milvusApi.listDatabases(props.milvusId);
        databases.value = res || [];
        stats.value.databaseCount = databases.value.length;
    } catch (error: unknown) {
        console.error('Failed to load databases:', error);
    }
};

const loadResourceGroups = async () => {
    try {
        const res = await milvusApi.listResourceGroups(props.milvusId);
        if (res && Array.isArray(res)) {
            // 获取每个资源组的详细信息
            const groups: IResourceGroup[] = [];
            for (const name of res) {
                try {
                    const detail = await milvusApi.describeResourceGroup(props.milvusId, name);
                    groups.push({
                        ...detail,
                        name,
                    });
                } catch (e) {
                    groups.push({ name });
                }
            }
            resourceGroups.value = groups;
        }
    } catch (error: unknown) {
        console.error('Failed to load resource groups:', error);
    }
};

const loadCollections = async () => {
    try {
        // 如果没有加载数据库列表，先加载
        if (databases.value.length === 0) {
            await loadDatabases();
        }

        // 遍历所有数据库统计 Collection 总数
        let totalCount = 0;
        const dbNames = databases.value.map((db: Record<string, unknown>) => (db.name as string) || (db.Name as string) || 'default');

        // 添加默认数据库
        if (!dbNames.includes('default')) {
            dbNames.unshift('default');
        }

        for (const dbName of dbNames) {
            try {
                const res = await milvusApi.listCollections(props.milvusId, dbName);
                totalCount += (res || []).length;
            } catch (e) {
                console.error(`Failed to load collections from ${dbName}:`, e);
            }
        }

        stats.value.collectionCount = totalCount;
    } catch (error: unknown) {
        console.error('Failed to load collections:', error);
    }
};

const loadSystemConfig = async () => {
    // 尝试从数据库属性或其他配置源获取系统配置
    try {
        const configs = [];
        // 添加一些常见的 Milvus 配置项
        if (version.value) {
            configs.push({ name: 'Server Version', value: version.value });
        }
        if (healthStatus.value !== null) {
            configs.push({ name: 'Health Status', value: healthStatus.value ? 'Healthy' : 'Unhealthy' });
        }
        configs.push({ name: 'Database Count', value: String(stats.value.databaseCount) });
        configs.push({ name: 'Collection Count', value: String(stats.value.collectionCount) });
        configs.push({ name: 'Resource Groups', value: String(resourceGroups.value.length) });
        systemConfig.value = configs;
    } catch (error: unknown) {
        console.error('Failed to load system config:', error);
    }
};

const loadAll = async () => {
    loading.value = true;
    try {
        await Promise.all([loadVersion(), checkHealth(), loadDatabases(), loadResourceGroups(), loadCollections()]);
        await loadSystemConfig();
    } finally {
        loading.value = false;
    }
};

const getCapacityPercentage = (row: IResourceGroup) => {
    // 适配 Go SDK 返回的大驼峰字段名
    const capacity = Number(row.Capacity ?? row.capacity ?? 0);
    const numAvailableNode = Number(row.NumAvailableNode ?? row.numAvailableNode ?? row.availableNodes ?? 0);
    if (!capacity) return 0;
    return Math.round(((capacity - numAvailableNode) / capacity) * 100);
};

const getCapacityColor = (row: IResourceGroup) => {
    const percentage = getCapacityPercentage(row);
    if (percentage < 50) return '#67C23A';
    if (percentage < 80) return '#E6A23C';
    return '#F56C6C';
};

const getLoadedReplicaCount = (rg: IResourceGroup) => {
    const replicas = rg.NumLoadedReplica ?? rg.numLoadedReplica ?? {};
    return Object.keys(replicas).length;
};

const isBoolean = (value: unknown) => {
    return value === 'true' || value === 'true' || value === true || value === false;
};

onMounted(() => {
    loadAll();
});

watch([() => props.milvusId, () => milvusStore.authCertName], () => {
    loadAll();
});
</script>

<style scoped>
.system-info-container {
    height: 100%;
    overflow: auto;
    padding: 16px;
}

.operation-bar {
    margin-bottom: 16px;
    display: flex;
    gap: 8px;
}

.metrics-row {
    margin-bottom: 16px;
}

.metric-card {
    display: flex;
    align-items: center;
    padding: 16px;
    transition: all 0.3s;
}

.metric-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.metric-card.is-healthy {
    border-left: 4px solid #67c23a;
}

.metric-card.is-unhealthy {
    border-left: 4px solid #f56c6c;
}

.metric-icon {
    width: 48px;
    height: 48px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    margin-right: 16px;
    background-color: #f5f7fa;
    color: #909399;
}

.metric-icon.version {
    background-color: #ecf5ff;
    color: #409eff;
}

.metric-icon.healthy {
    background-color: #f0f9eb;
    color: #67c23a;
}

.metric-icon.unhealthy {
    background-color: #fef0f0;
    color: #f56c6c;
}

.metric-icon.database {
    background-color: #f5f0ff;
    color: #9254de;
}

.metric-icon.collection {
    background-color: #fff7e6;
    color: #fa8c16;
}

.metric-content {
    flex: 1;
    min-width: 0;
}

.metric-label {
    font-size: 12px;
    color: #909399;
    margin-bottom: 4px;
}

.metric-value {
    font-size: 18px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.text-gray {
    color: #909399;
}

.info-card,
.health-detail-card {
    margin-bottom: 16px;
}

.card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-weight: 600;
}

.health-item {
    padding: 4px 0;
}

.health-item-name {
    font-weight: 500;
    color: #303133;
}

.health-item-message {
    font-size: 12px;
    color: #909399;
    margin-top: 4px;
}

/* 资源组样式 */
.rg-collapse-title {
    display: flex;
    align-items: center;
    flex: 1;
    padding-right: 16px;
}

.rg-name {
    font-weight: 600;
    font-size: 14px;
    margin-right: 16px;
    min-width: 150px;
}

.rg-node-tag,
.rg-replica-tag {
    margin-left: 8px;
}

.rg-section {
    margin-top: 16px;
}

.rg-section-title {
    font-size: 13px;
    font-weight: 600;
    color: #606266;
    margin-bottom: 8px;
    padding-left: 8px;
    border-left: 3px solid #409eff;
}

.collection-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.collection-tag {
    margin: 0;
}

.config-info {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
}

.ml-2 {
    margin-left: 8px;
}

.mr-1 {
    margin-right: 4px;
}

.mb-1 {
    margin-bottom: 4px;
}

:deep(.el-card__header) {
    padding: 12px 16px;
    border-bottom: 1px solid #ebeef5;
}

:deep(.el-card__body) {
    padding: 16px;
}

:deep(.el-timeline-item__node) {
    display: flex;
    align-items: center;
    justify-content: center;
}
</style>
