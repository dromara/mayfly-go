<template>
    <div>
        <el-card>
            <el-button type="primary" icon="plus" @click="editRedis(true)" plain>添加</el-button>
            <el-button type="primary" icon="edit" :disabled="currentId == null" @click="editRedis(false)" plain>编辑
            </el-button>
            <el-button type="danger" icon="delete" :disabled="currentId == null" @click="deleteRedis" plain>删除
            </el-button>
            <div style="float: right">
                <el-select @focus="getTags" v-model="query.tagPath" placeholder="请选择标签" filterable clearable>
                    <el-option v-for="item in tags" :key="item" :label="item" :value="item"> </el-option>
                </el-select>
                <el-button class="ml5" @click="search" type="success" icon="search"></el-button>
            </div>
            <el-table :data="redisTable" @current-change="choose" stripe>
                <el-table-column label="选择" width="60px">
                    <template #default="scope">
                        <el-radio v-model="currentId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="tagPath" label="标签路径" min-width="150" show-overflow-tooltip>
                    <template #default="scope">
                        <tag-info :tag-path="scope.row.tagPath" />
                        <span class="ml5">
                            {{ scope.row.tagPath }}
                        </span>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="名称" min-width="100"></el-table-column>
                <el-table-column prop="host" label="host:port" min-width="150" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="mode" label="mode" min-width="100"></el-table-column>
                <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip></el-table-column>

                <el-table-column label="更多" min-width="155" fixed="right">
                    <template #default="scope">
                        <el-link @click="showDetail(scope.row)" :underline="false">详情</el-link>
                        <el-divider direction="vertical" border-style="dashed" />

                        <el-link v-if="scope.row.mode === 'standalone' || scope.row.mode === 'sentinel'" type="primary"
                            @click="showInfoDialog(scope.row)" :underline="false">单机信息</el-link>
                        <el-link @click="onShowClusterInfo(scope.row)" v-if="scope.row.mode === 'cluster'"
                            type="primary" :underline="false">集群信息</el-link>
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination style="text-align: right" @current-change="handlePageChange" :total="total"
                    layout="prev, pager, next, total, jumper" v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"></el-pagination>
            </el-row>
        </el-card>

        <info v-model:visible="infoDialog.visible" :title="infoDialog.title" :info="infoDialog.info"></info>

        <el-dialog width="1000px" title="集群信息" v-model="clusterInfoDialog.visible">
            <el-input type="textarea" :autosize="{ minRows: 12, maxRows: 12 }" v-model="clusterInfoDialog.info">
            </el-input>

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
                        <el-tooltip class="box-item" effect="dark"
                            content="ip:port1@port2：port1指redis服务器与客户端通信的端口，port2则是集群内部节点间通信的端口" placement="top">
                            <el-icon>
                                <question-filled />
                            </el-icon>
                        </el-tooltip>
                    </template>
                    <template #default="scope">
                        <el-tag @click="showInfoDialog({ id: clusterInfoDialog.redisId, ip: scope.row.ip })"
                            effect="plain" type="success" size="small" style="cursor: pointer">{{ scope.row.ip }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="flags" label="flags" min-width="110"></el-table-column>
                <el-table-column prop="masterSlaveRelation" label="masterSlaveRelation" min-width="300">
                    <template #header>
                        masterSlaveRelation
                        <el-tooltip class="box-item" effect="dark"
                            content="如果节点是slave，并且已知master节点，则为master节点ID；否则为符号'-'" placement="top">
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
                        <el-tooltip class="box-item" effect="dark"
                            content="节点的epoch值（如果该节点是从节点，则为其主节点的epoch值）。每当节点发生失败切换时，都会创建一个新的，独特的，递增的epoch。"
                            placement="top">
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

                <el-descriptions-item :span="3" label="SSH隧道">{{ detailDialog.data.sshTunnelMachineId > 0 ? '是' : '否' }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" label="创建时间">{{ dateFormat(detailDialog.data.createTime) }}
                </el-descriptions-item>
                <el-descriptions-item :span="1" label="创建者">{{ detailDialog.data.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="更新时间">{{ dateFormat(detailDialog.data.updateTime) }}
                </el-descriptions-item>
                <el-descriptions-item :span="1" label="修改者">{{ detailDialog.data.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <redis-edit @val-change="valChange" :tags="tags" :title="redisEditDialog.title"
            v-model:visible="redisEditDialog.visible" v-model:redis="redisEditDialog.data"></redis-edit>
    </div>
</template>

<script lang="ts" setup>
import Info from './Info.vue';
import { redisApi } from './api';
import { toRefs, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { tagApi } from '../tag/api.ts';
import RedisEdit from './RedisEdit.vue';
import { dateFormat } from '@/common/utils/date';
import TagInfo from '../component/TagInfo.vue';

const state = reactive({
    tags: [],
    redisTable: [],
    total: 0,
    currentId: null,
    currentData: null,
    query: {
        tagPath: null,
        pageNum: 1,
        pageSize: 10,
        clusterId: null,
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

const {
    tags,
    redisTable,
    total,
    currentId,
    query,
    detailDialog,
    clusterInfoDialog,
    infoDialog,
    redisEditDialog,
} = toRefs(state)

onMounted(async () => {
    search();
});

const handlePageChange = (curPage: number) => {
    state.query.pageNum = curPage;
    search();
};

const showDetail = (detail: any) => {
    state.detailDialog.data = detail;
    state.detailDialog.visible = true;
}

const choose = (item: any) => {
    if (!item) {
        return;
    }
    state.currentId = item.id;
    state.currentData = item;
};

const deleteRedis = async () => {
    try {
        await ElMessageBox.confirm(`确定删除该redis?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await redisApi.delRedis.request({ id: state.currentId });
        ElMessage.success('删除成功');
        state.currentData = null;
        state.currentId = null;
        search();
    } catch (err) { }
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

const search = async () => {
    const res = await redisApi.redisList.request(state.query);
    state.redisTable = res.list;
    state.total = res.total;
};

const getTags = async () => {
    state.tags = await tagApi.getAccountTags.request(null);
};

const editRedis = async (isAdd = false) => {
    if (isAdd) {
        state.redisEditDialog.data = null;
        state.redisEditDialog.title = '新增redis';
    } else {
        state.redisEditDialog.data = state.currentData;
        state.redisEditDialog.title = '修改redis';
    }
    state.redisEditDialog.visible = true;
};

const valChange = () => {
    state.currentId = null;
    state.currentData = null;
    search();
};
</script>

<style>

</style>
