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
                <el-button v-auth="'redis:save'" type="primary" icon="plus" @click="editRedis(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="'redis:del'" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteRedis" plain>
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
                <el-button v-auth="'redis:save'" type="primary" link @click="editRedis(data)">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <info v-model:visible="infoDialog.visible" :title="infoDialog.title" :info="infoDialog.info"></info>

        <el-dialog width="1000px" :title="$t('redis.clusterInfo')" v-model="clusterInfoDialog.visible">
            <el-input type="textarea" :autosize="{ minRows: 12, maxRows: 12 }" v-model="clusterInfoDialog.info"> </el-input>

            <el-divider content-position="left">{{ $t('redis.node') }}</el-divider>
            <el-table :data="clusterInfoDialog.nodes" stripe size="small" border>
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
                            @click="showInfoDialog({ id: clusterInfoDialog.redisId, ip: scope.row.ip })"
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

        <el-dialog v-if="detailDialog.visible" v-model="detailDialog.visible">
            <el-descriptions v-if="detailDialog.data" :title="$t('common.detail')" :column="3" border>
                <el-descriptions-item :span="1.5" label="id">{{ detailDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('common.name')">{{ detailDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><TagCodePath :code="detailDialog.data.code" /></el-descriptions-item>

                <el-descriptions-item :span="3" label="Host">{{ detailDialog.data.host }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="DB">{{ detailDialog.data.db }}</el-descriptions-item>
                <el-descriptions-item :span="3" :label="$t('common.remark')">{{ detailDialog.data.remark }}</el-descriptions-item>
                <el-descriptions-item :span="3" :label="$t('machine.sshTunnel')">
                    {{ detailDialog.data.sshTunnelMachineId > 0 ? $t('common.yes') : $t('common.no') }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.createTime')">{{ formatDate(detailDialog.data.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.creator')">{{ detailDialog.data.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.updateTime')">{{ formatDate(detailDialog.data.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.modifier')">{{ detailDialog.data.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <redis-edit
            @val-change="search()"
            :title="redisEditDialog.title"
            v-model:visible="redisEditDialog.visible"
            v-model:redis="redisEditDialog.data"
        ></redis-edit>
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useRoute } from 'vue-router';
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

const route = useRoute();
const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('redis.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(15),
    TableColumn.new('host', 'Host'),
    TableColumn.new('mode', 'Mode'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(200).fixedRight().alignCenter(),
]);

const state = reactive({
    selectionData: [],
    query: {
        tagPath: '',
        pageNum: 1,
        pageSize: 0,
    },
    detailDialog: {
        visible: false,
        data: null as Redis | null,
    },
    clusterInfoDialog: {
        visible: false,
        redisId: 0,
        info: '',
        nodes: [] as RedisClusterNodeRow[],
    },
    infoDialog: {
        title: '',
        visible: false,
        info: {
            Server: {},
            Keyspace: {},
            Clients: {},
            CPU: {},
            Memory: {},
        } as Record<string, unknown>,
    },
    redisEditDialog: {
        visible: false,
        data: null as Redis | null,
        title: '',
    },
});

const { selectionData, query, detailDialog, clusterInfoDialog, infoDialog, redisEditDialog } = toRefs(state);

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const checkRouteTagPath = (query: Record<string, unknown>) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const showDetail = (detail: Redis) => {
    state.detailDialog.data = detail;
    state.detailDialog.visible = true;
};

const deleteRedis = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: Redis) => x.name).join('、'));
        await redisApi.delRedis.request({ id: state.selectionData.map((x: Redis) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};

const showInfoDialog = async (redis: { id: number; ip?: string; host?: string; name?: string }) => {
    let host = redis.host ?? '';
    if (redis.ip) {
        host = redis.ip.split('@')[0];
    }
    const res = await redisApi.redisInfo.request({ id: redis.id, host });
    state.infoDialog.info = res;
    state.infoDialog.title = `[${redis.name || host}] redis`;
    state.infoDialog.visible = true;
};

const onShowClusterInfo = async (redis: Redis) => {
    const ci = await redisApi.clusterInfo.request({ id: redis.id });
    state.clusterInfoDialog.info = ci.clusterInfo as string;
    state.clusterInfoDialog.nodes = ci.clusterNodes as RedisClusterNodeRow[];
    state.clusterInfoDialog.redisId = redis.id;
    state.clusterInfoDialog.visible = true;
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value?.search();
};

const editRedis = async (data: Redis | false) => {
    if (!data) {
        state.redisEditDialog.data = null;
        state.redisEditDialog.title = useI18nCreateTitle('Redis');
    } else {
        state.redisEditDialog.data = data;
        state.redisEditDialog.title = useI18nEditTitle('Redis');
    }
    state.redisEditDialog.visible = true;
};

defineExpose({ search });
</script>

<style></style>
