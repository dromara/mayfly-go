<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="redisApi.redisList"
            :before-query-fn="checkRouteTagPath"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="'redis:save'" type="primary" icon="plus" @click="editEntity()" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="'redis:del'" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="onDelete" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #action="{ data }">
                <el-button v-if="data.mode === 'standalone' || data.mode === 'sentinel'" type="primary" @click="showInfoDialog(data)" link>
                    {{ $t('redis.standaloneInfo') }}
                </el-button>
                <el-button @click="onShowClusterInfo(data)" v-if="data.mode === 'cluster'" type="primary" link>{{ $t('redis.clusterInfo') }}</el-button>

                <el-button @click="showDetail(data)" link>{{ $t('common.detail') }}</el-button>
                <el-button v-auth="'redis:save'" type="primary" link @click="editEntity(data)">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <info v-model:visible="infoDialog.visible" :title="infoDialog.title" :info="infoDialog.info"></info>

        <el-dialog width="1000px" :title="$t('redis.clusterInfo')" v-model="clusterInfoVisible">
            <el-input type="textarea" :autosize="{ minRows: 12, maxRows: 12 }" v-model="clusterInfo"> </el-input>

            <el-divider content-position="left">{{ $t('redis.node') }}</el-divider>
            <el-table :data="clusterNodes" stripe size="small" border>
                <el-table-column prop="nodeId" label="nodeId" min-width="300">
                    <template #header>
                        nodeId
                        <el-tooltip class="box-item" effect="dark" content="node id" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="ip" label="ip" min-width="180">
                    <template #header>
                        ip
                        <el-tooltip class="box-item" effect="dark" :content="$t('redis.clusterIpTips')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                    <template #default="scope">
                        <el-tag
                            @click="showInfoDialog({ id: clusterRedisId, ip: scope.row.ip })"
                            effect="plain"
                            type="success"
                            size="small"
                            class="cursor-pointer"
                            >{{ scope.row.ip }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="flags" label="flags" min-width="110"></el-table-column>
                <el-table-column prop="masterSlaveRelation" label="masterSlaveRelation" min-width="300">
                    <template #header>
                        masterSlaveRelation
                        <el-tooltip class="box-item" effect="dark" :content="$t('redis.masterSlaveRelationTips')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="pingSent" label="pingSent" min-width="130" show-overflow-tooltip>
                    <template #default="scope">
                        {{ scope.row.pingSent == 0 ? 0 : new Date(parseInt(scope.row.pingSent)).toLocaleString() }}
                    </template>
                </el-table-column>
                <el-table-column prop="pongRecv" label="pongRecv" min-width="130" show-overflow-tooltip>
                    <template #default="scope">
                        {{ scope.row.pongRecv == 0 ? 0 : new Date(parseInt(scope.row.pongRecv)).toLocaleString() }}
                    </template>
                </el-table-column>
                <el-table-column prop="configEpoch" label="configEpoch" min-width="130">
                    <template #header>
                        configEpoch
                        <el-tooltip class="box-item" effect="dark" :content="$t('redis.configEpochTips')" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="linkState" label="linkState" min-width="100"></el-table-column>
                <el-table-column prop="slot" label="slot" min-width="100"></el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog v-if="detailVisible" v-model="detailVisible">
            <el-descriptions v-if="detailData" :title="$t('common.detail')" :column="3" border>
                <el-descriptions-item :span="1.5" label="id">{{ detailData.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('common.name')">{{ detailData.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><TagCodePath :code="detailData.code" /></el-descriptions-item>

                <el-descriptions-item :span="3" label="Host">{{ detailData.host }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="DB">{{ detailData.db }}</el-descriptions-item>
                <el-descriptions-item :span="3" :label="$t('common.remark')">{{ detailData.remark }}</el-descriptions-item>
                <el-descriptions-item :span="3" :label="$t('machine.sshTunnel')">
                    {{ detailData.sshTunnelMachineId > 0 ? $t('common.yes') : $t('common.no') }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.createTime')">{{ formatDate(detailData.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.creator')">{{ detailData.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.updateTime')">{{ formatDate(detailData.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.modifier')">{{ detailData.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <redis-edit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog, useRouteTagPath } from '@/hooks/useResourceForm';
import { ref, useTemplateRef } from 'vue';
import TagCodePath from '../component/TagCodePath.vue';
import Info from './Info.vue';
import RedisEdit from './RedisEdit.vue';
import { redisApi } from './api';
import type { Redis } from './types';

/** Redis 集群节点行 (对应后端 cluster-info 返回的节点信息) */
interface RedisClusterNodeRow {
    nodeId: string;
    ip: string;
    flags: string;
    masterSlaveRelation: string;
    pingSent: string;
    pongRecv: string;
    configEpoch: string;
    linkState: string;
    slot: string;
}

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const checkRouteTagPath = useRouteTagPath();
const { editDialog, editEntity } = useEditDialog<Redis>('Redis');

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('redis.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(15),
    TableColumn.new('host', 'Host'),
    TableColumn.new('mode', 'Mode'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(200).fixedRight().alignCenter(),
]);

const query = ref({
    tagPath: '',
    pageNum: 1,
    pageSize: 0,
});

const selectionData = ref<Redis[]>([]);

// --- 详情弹窗 ---
const detailVisible = ref(false);
const detailData = ref<Redis | null>(null);

const showDetail = (detail: Redis) => {
    detailData.value = detail;
    detailVisible.value = true;
};

// --- 信息弹窗 ---
const infoDialog = ref({
    title: '',
    visible: false,
    info: {
        Server: {},
        Keyspace: {},
        Clients: {},
        CPU: {},
        Memory: {},
    } as Record<string, unknown>,
});

// --- 集群信息弹窗 ---
const clusterInfoVisible = ref(false);
const clusterInfo = ref('');
const clusterNodes = ref<RedisClusterNodeRow[]>([]);
const clusterRedisId = ref(0);

const showInfoDialog = async (redis: { id: number; ip?: string; host?: string; name?: string }) => {
    let host = redis.host ?? '';
    if (redis.ip) {
        host = redis.ip.split('@')[0];
    }
    const res = await redisApi.redisInfo.request({ id: redis.id, host });
    infoDialog.value.info = res;
    infoDialog.value.title = `[${redis.name || host}] redis`;
    infoDialog.value.visible = true;
};

const onShowClusterInfo = async (redis: Redis) => {
    const ci = await redisApi.clusterInfo.request({ id: redis.id });
    clusterInfo.value = ci.clusterInfo as string;
    clusterNodes.value = ci.clusterNodes as RedisClusterNodeRow[];
    clusterRedisId.value = redis.id;
    clusterInfoVisible.value = true;
};

// --- 删除 ---
const onDelete = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) return;
    try {
        await useI18nDeleteConfirm(records.map((x) => x.name).join('、'));
    } catch {
        return; // 用户取消
    }
    await redisApi.delRedis.request({ id: records.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const search = (tagPath?: string) => {
    // tagPath 为 undefined 时（如"所有资源"节点），清空过滤条件查询全部；为具体值时按标签过滤
    query.value.tagPath = tagPath ?? '';
    pageTableRef.value?.search();
};

defineExpose({ search });
</script>

<style></style>
