<template>
    <div>
        <page-table
            ref="pageTableRef"
            :page-api="redisApi.redisList"
            :before-query-fn="checkRouteTagPath"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="editRedis(false)" plain>添加</el-button>
                <el-button type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteRedis" plain>删除 </el-button>
            </template>

            <template #tagPath="{ data }">
                <resource-tag :resource-code="data.code" :resource-type="TagResourceTypeEnum.Redis.value" />
            </template>

            <template #action="{ data }">
                <el-button v-if="data.mode === 'standalone' || data.mode === 'sentinel'" type="primary" @click="showInfoDialog(data)" link>单机信息</el-button>
                <el-button @click="onShowClusterInfo(data)" v-if="data.mode === 'cluster'" type="primary" link>集群信息</el-button>

                <el-button @click="showDetail(data)" link>详情</el-button>
                <el-button type="primary" link @click="editRedis(data)">编辑</el-button>
            </template>
        </page-table>

        <info v-model:visible="infoDialog.visible" :title="infoDialog.title" :info="infoDialog.info"></info>

        <el-dialog width="1000px" title="集群信息" v-model="clusterInfoDialog.visible">
            <el-input type="textarea" :autosize="{ minRows: 12, maxRows: 12 }" v-model="clusterInfoDialog.info"> </el-input>

            <el-divider content-position="left">节点信息</el-divider>
            <el-table :data="clusterInfoDialog.nodes" stripe size="small" border>
                <el-table-column prop="nodeId" label="nodeId" min-width="300">
                    <template #header>
                        nodeId
                        <el-tooltip class="box-item" effect="dark" content="节点id" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="ip" label="ip" min-width="180">
                    <template #header>
                        ip
                        <el-tooltip
                            class="box-item"
                            effect="dark"
                            content="ip:port1@port2：port1指redis服务器与客户端通信的端口，port2则是集群内部节点间通信的端口"
                            placement="top"
                        >
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
                            style="cursor: pointer"
                            >{{ scope.row.ip }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="flags" label="flags" min-width="110"></el-table-column>
                <el-table-column prop="masterSlaveRelation" label="masterSlaveRelation" min-width="300">
                    <template #header>
                        masterSlaveRelation
                        <el-tooltip
                            class="box-item"
                            effect="dark"
                            content="如果节点是slave，并且已知master节点，则为master节点ID；否则为符号'-'"
                            placement="top"
                        >
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
                        <el-tooltip
                            class="box-item"
                            effect="dark"
                            content="节点的epoch值（如果该节点是从节点，则为其主节点的epoch值）。每当节点发生失败切换时，都会创建一个新的，独特的，递增的epoch。"
                            placement="top"
                        >
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

        <el-dialog v-model="detailDialog.visible">
            <el-descriptions title="详情" :column="3" border>
                <el-descriptions-item :span="1.5" label="id">{{ detailDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" label="名称">{{ detailDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="标签路径">{{ detailDialog.data.tagPath }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="主机">{{ detailDialog.data.host }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="库">{{ detailDialog.data.db }}</el-descriptions-item>
                <el-descriptions-item :span="3" label="备注">{{ detailDialog.data.remark }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="SSH隧道">{{ detailDialog.data.sshTunnelMachineId > 0 ? '是' : '否' }} </el-descriptions-item>

                <el-descriptions-item :span="2" label="创建时间">{{ dateFormat(detailDialog.data.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="创建者">{{ detailDialog.data.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="更新时间">{{ dateFormat(detailDialog.data.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="修改者">{{ detailDialog.data.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <redis-edit
            @val-change="search"
            :title="redisEditDialog.title"
            v-model:visible="redisEditDialog.visible"
            v-model:redis="redisEditDialog.data"
        ></redis-edit>
    </div>
</template>

<script lang="ts" setup>
import Info from './Info.vue';
import { redisApi } from './api';
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import RedisEdit from './RedisEdit.vue';
import { dateFormat } from '@/common/utils/date';
import ResourceTag from '../component/ResourceTag.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { useRoute } from 'vue-router';
import { getTagPathSearchItem } from '../component/tag';

const route = useRoute();
const pageTableRef: Ref<any> = ref(null);

const searchItems = [getTagPathSearchItem(TagResourceTypeEnum.Redis.value)];

const columns = ref([
    TableColumn.new('name', '名称'),
    TableColumn.new('host', 'host:port'),
    TableColumn.new('mode', 'mode'),
    TableColumn.new('tagPath', '关联标签').isSlot().setAddWidth(10).alignCenter(),
    TableColumn.new('remark', '备注'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(200).fixedRight().alignCenter(),
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
        title: '新增redis',
    },
});

const { selectionData, query, detailDialog, clusterInfoDialog, infoDialog, redisEditDialog } = toRefs(state);

onMounted(async () => {});

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
        await ElMessageBox.confirm(`确定删除该【${state.selectionData.map((x: any) => x.name).join(', ')}】redis信息?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await redisApi.delRedis.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
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
    state.infoDialog.title = `[${redis.name || host}] redis信息`;
    state.infoDialog.visible = true;
};

const onShowClusterInfo = async (redis: any) => {
    const ci = await redisApi.clusterInfo.request({ id: redis.id });
    state.clusterInfoDialog.info = ci.clusterInfo;
    state.clusterInfoDialog.nodes = ci.clusterNodes;
    state.clusterInfoDialog.redisId = redis.id;
    state.clusterInfoDialog.visible = true;
};

const search = () => {
    pageTableRef.value.search();
};

const editRedis = async (data: any) => {
    if (!data) {
        state.redisEditDialog.data = null;
        state.redisEditDialog.title = '新增redis';
    } else {
        state.redisEditDialog.data = data;
        state.redisEditDialog.title = '修改redis';
    }
    state.redisEditDialog.visible = true;
};
</script>

<style></style>
