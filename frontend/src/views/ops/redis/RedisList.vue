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
                <el-button type="primary" icon="plus" @click="editRedis(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteRedis" plain>{{ $t('common.delete') }}</el-button>
            </template>

            <template #tagPath="{ data }">
                <resource-tags :tags="data.tags" />
            </template>

            <template #action="{ data }">
                <el-button v-if="data.mode === 'standalone' || data.mode === 'sentinel'" type="primary" @click="showInfoDialog(data)" link>
                    {{ $t('redis.standaloneInfo') }}
                </el-button>
                <el-button @click="onShowClusterInfo(data)" v-if="data.mode === 'cluster'" type="primary" link>{{ $t('redis.clusterInfo') }}</el-button>

                <el-button @click="showDetail(data)" link>{{ $t('common.detail') }}</el-button>
                <el-button type="primary" link @click="editRedis(data)">{{ $t('common.edit') }}</el-button>
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
            <el-descriptions :title="$t('common.detail')" :column="3" border>
                <el-descriptions-item :span="1.5" label="id">{{ detailDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('common.name')">{{ detailDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><ResourceTags :tags="detailDialog.data.tags" /></el-descriptions-item>

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
import Info from './Info.vue';
import { redisApi } from './api';
import { onMounted, reactive, ref, Ref, toRefs } from 'vue';
import RedisEdit from './RedisEdit.vue';
import { formatDate } from '@/common/utils/format';
import ResourceTags from '../component/ResourceTags.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { useRoute } from 'vue-router';
import { getTagPathSearchItem } from '../component/tag';
import { SearchItem } from '@/components/SearchForm';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle } from '@/hooks/useI18n';

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const route = useRoute();
const pageTableRef: Ref<any> = ref(null);

const searchItems = [
    SearchItem.input('keyword', 'common.keyword').withPlaceholder('redis.keywordPlaceholder'),
    getTagPathSearchItem(TagResourceTypeEnum.Redis.value),
];

const columns = ref([
    TableColumn.new('tags[0].tagPath', 'tag.relateTag').isSlot('tagPath').setAddWidth(20),
    TableColumn.new('name', 'common.name'),
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
        data: null as any,
    },
    clusterInfoDialog: {
        visible: false,
        redisId: 0,
        info: '',
        nodes: [],
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
        },
    },
    redisEditDialog: {
        visible: false,
        data: null as any,
        title: '',
    },
});

const { selectionData, query, detailDialog, clusterInfoDialog, infoDialog, redisEditDialog } = toRefs(state);

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const checkRouteTagPath = (query: any) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const showDetail = (detail: any) => {
    state.detailDialog.data = detail;
    state.detailDialog.visible = true;
};

const deleteRedis = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.name).join('、'));
        await redisApi.delRedis.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const showInfoDialog = async (redis: any) => {
    var host = redis.host;
    if (redis.ip) {
        host = redis.ip.split('@')[0];
    }
    const res = await redisApi.redisInfo.request({ id: redis.id, host });
    state.infoDialog.info = res;
    state.infoDialog.title = `[${redis.name || host}] redis`;
    state.infoDialog.visible = true;
};

const onShowClusterInfo = async (redis: any) => {
    const ci = await redisApi.clusterInfo.request({ id: redis.id });
    state.clusterInfoDialog.info = ci.clusterInfo;
    state.clusterInfoDialog.nodes = ci.clusterNodes;
    state.clusterInfoDialog.redisId = redis.id;
    state.clusterInfoDialog.visible = true;
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value.search();
};

const editRedis = async (data: any) => {
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
