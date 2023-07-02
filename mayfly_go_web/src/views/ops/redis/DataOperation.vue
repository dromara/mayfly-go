<template>
    <div>
        <el-row>
            <el-col :span="4">
                <el-row type="flex" justify="space-between">
                    <el-col :span="24" class="el-scrollbar flex-auto">
                        <tag-tree @node-click="nodeClick" :load="loadNode">
                            <template #prefix="{ data }">
                                <span v-if="data.type == NodeType.Redis">
                                    <el-popover placement="right-start" title="redis实例信息" trigger="hover" :width="210">
                                        <template #reference>
                                            <SvgIcon name="iconfont icon-op-redis" :size="18" />
                                        </template>
                                        <template #default>
                                            <el-form class="instances-pop-form" label-width="50px" :size="'small'">
                                                <el-form-item label="名称:">{{ data.params.name }}</el-form-item>
                                                <el-form-item label="模式:">{{ data.params.mode }}</el-form-item>
                                                <el-form-item label="链接:">{{ data.params.host }}</el-form-item>
                                                <el-form-item label="备注:">{{
                                                    data.params.remark
                                                }}</el-form-item>
                                            </el-form>
                                        </template>
                                    </el-popover>
                                </span>

                                <SvgIcon v-if="data.type == NodeType.Db" name="Coin" color="#67c23a" />
                            </template>
                        </tag-tree>
                    </el-col>
                </el-row>
            </el-col>

            <el-col :span="20" style="border-left: 1px solid var(--el-card-border-color);">
                <div class="mt10 ml5">
                    <el-col>
                        <el-form class="search-form" label-position="right" :inline="true" label-width="60px">
                            <el-form-item label="key" label-width="40px">
                                <el-input placeholder="match 支持*模糊key" style="width: 250px" v-model="scanParam.match"
                                    @clear="clear()" clearable></el-input>
                            </el-form-item>
                            <el-form-item label="count" label-width="40px">
                                <el-input placeholder="count" style="width: 70px" v-model.number="scanParam.count">
                                </el-input>
                            </el-form-item>
                            <el-form-item>
                                <el-button :disabled="!scanParam.id || !scanParam.db" @click="searchKey()" type="success"
                                    icon="search" plain></el-button>
                                <el-button :disabled="!scanParam.id || !scanParam.db" @click="scan()" icon="bottom"
                                    plain>scan</el-button>
                                <el-button :disabled="!scanParam.id || !scanParam.db" @click="showNewKeyDialog"
                                    type="primary" icon="plus" plain v-auth="'redis:data:save'"></el-button>
                                <el-button :disabled="!scanParam.id || !scanParam.db" @click="flushDb" type="danger" plain
                                    v-auth="'redis:data:save'">flush</el-button>
                            </el-form-item>
                            <div style="float: right">
                                <span>keys: {{ state.dbsize }}</span>
                            </div>
                        </el-form>
                    </el-col>
                    <el-table v-loading="state.loading" :data="state.keys" :height="tableHeight" stripe
                        :highlight-current-row="true" style="cursor: pointer">
                        <el-table-column show-overflow-tooltip prop="key" label="key"></el-table-column>
                        <el-table-column prop="type" label="type" width="80">
                            <template #default="scope">
                                <el-tag :color="getTypeColor(scope.row.type)" size="small">{{ scope.row.type }}</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column prop="ttl" label="ttl(过期时间)" width="140">
                            <template #default="scope">
                                {{ ttlConveter(scope.row.ttl) }}
                            </template>
                        </el-table-column>
                        <el-table-column label="操作">
                            <template #default="scope">
                                <el-button @click="showKeyDetail(scope.row)" type="success" icon="search" plain
                                    size="small">查看
                                </el-button>
                                <el-button v-auth="'redis:data:del'" @click="del(scope.row.key)" type="danger" icon="delete"
                                    plain size="small">删除
                                </el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                </div>
            </el-col>
        </el-row>

        <div style="text-align: center; margin-top: 10px"></div>

        <el-dialog title="Key详情" v-model="keyDetailDialog.visible" width="800px" :destroy-on-close="true"
            :close-on-click-modal="false">
            <key-detail :redisId="scanParam.id" :db="scanParam.db" :key-info="keyDetailDialog.keyInfo"
                @change-key="searchKey()" />
        </el-dialog>

        <el-dialog title="新增Key" v-model="newKeyDialog.visible" width="500px" :destroy-on-close="true"
            :close-on-click-modal="false">
            <el-form ref="keyForm" label-width="50px">
                <el-form-item prop="key" label="键名">
                    <el-input v-model.trim="keyDetailDialog.keyInfo.key" placeholder="请输入键名"></el-input>
                </el-form-item>
                <el-form-item prop="type" label="类型">
                    <el-select v-model="keyDetailDialog.keyInfo.type" default-first-option style="width: 100%"
                        placeholder="请选择类型">
                        <el-option key="string" label="string" value="string"></el-option>
                        <el-option key="hash" label="hash" value="hash"></el-option>
                        <el-option key="set" label="set" value="set"></el-option>
                        <el-option key="zset" label="zset" value="zset"></el-option>
                        <el-option key="list" label="list" value="list"></el-option>
                    </el-select>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelNewKey()">取 消</el-button>
                    <el-button v-auth="'machine:script:save'" type="primary" @click="newKey">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { redisApi } from './api';
import { defineAsyncComponent, toRefs, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { isTrue, notBlank, notNull } from '@/common/assert';
import { TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';

const KeyDetail = defineAsyncComponent(() => import('./KeyDetail.vue'));

/**
 * 树节点类型
 */
class NodeType {
    static Redis = 1
    static Db = 2
}

const state = reactive({
    loading: false,
    tableHeight: 600,
    tags: [],
    redisList: [] as any,
    dbList: [],
    query: {
        tagPath: null,
    },
    scanParam: {
        id: null as any,
        mode: '',
        db: null as any,
        match: null,
        count: 10,
        cursor: {},
    },
    keyDetailDialog: {
        visible: false,
        keyInfo: {
            type: 'string',
            timed: -1,
            key: '',
        },
    },
    newKeyDialog: {
        visible: false,
    },
    keys: [],
    dbsize: 0,
});

const {
    tableHeight,
    scanParam,
    keyDetailDialog,
    newKeyDialog,
} = toRefs(state)


onMounted(async () => {
    setHeight();
})

const setHeight = () => {
    state.tableHeight = window.innerHeight - 159;
}

/**
 * instmap;  tagPaht -> redis info[]
 */
const instMap: Map<string, any[]> = new Map();

const getInsts = async () => {
    const res = await redisApi.redisList.request({ pageNum: 1, pageSize: 1000 });
    if (!res.total) return
    for (const redisInfo of res.list) {
        const tagPath = redisInfo.tagPath;
        let redisInsts = instMap.get(tagPath) || [];
        redisInsts.push(redisInfo);
        instMap.set(tagPath, redisInsts);
    }
}

/**
 * 加载文件树节点
 * @param {Object} node
 * @param {Object} resolve
 */
const loadNode = async (node: any) => {
    // 一级为tagPath
    if (node.level === 0) {
        await getInsts();
        const tagPaths = instMap.keys();
        const tagNodes = [];
        for (let tagPath of tagPaths) {
            tagNodes.push(new TagTreeNode(tagPath, tagPath));
        }
        return tagNodes;
    }

    const data = node.data;
    // 点击tagPath -> 加载数据库信息列表
    if (data.type === TagTreeNode.TagPath) {
        const redisInfos = instMap.get(data.key)
        return redisInfos?.map((x: any) => {
            return new TagTreeNode(`${data.key}.${x.id}`, x.name, NodeType.Redis).withParams(x);
        });
    }

    // 点击redis实例 -> 加载库列表
    if (data.type === NodeType.Redis) {
        return await getDbs(data.params);
    }

    return [];
};

const nodeClick = (data: any) => {
    // 点击库事件
    if (data.type === NodeType.Db) {
        resetScanParam();
        state.scanParam.id = data.params.id;
        state.scanParam.db = data.params.db;
        scan();
    }
}

/**
 * 获取所有库信息
 * @param redisInfo redis信息
 */
const getDbs = async (redisInfo: any) => {
    let dbs: TagTreeNode[] = redisInfo.db.split(',').map((x: string) => {
        return new TagTreeNode(x, `db${x}`, NodeType.Db).withIsLeaf(true).withParams({
            id: redisInfo.id,
            db: x,
            name: `db${x}`,
            keys: 0,
        })
    })

    if (redisInfo.mode == 'cluster') {
        return dbs;
    }

    const res = await redisApi.redisInfo.request({ id: redisInfo.id, host: redisInfo.host, section: "Keyspace" });
    for (let db in res.Keyspace) {
        for (let d of dbs) {
            if (db == d.params.name) {
                d.params.keys = res.Keyspace[db]?.split(',')[0]?.split('=')[1] || 0
            }
        }
    }
    // 替换label
    dbs.forEach((e: any) => {
        e.label = `${e.params.name} [${e.params.keys}]`
    });
    return dbs;
}

const scan = async () => {
    isTrue(state.scanParam.id != null, '请先选择redis');
    notBlank(state.scanParam.count, 'count不能为空');

    const match: string = state.scanParam.match || '';
    if (!match) {
        isTrue(state.scanParam.count <= 100, "key搜索条件为空时, count不能大于100")
    } else if (match.indexOf('*') != -1) {
        const dbsize = state.dbsize;
        // 如果为模糊搜索，并且搜索的key模式大于指定字符数，则将count设大点scan
        if (match.length > 10) {
            state.scanParam.count = dbsize > 100000 ? Math.floor(dbsize / 10) : 1000;
        } else {
            state.scanParam.count = 100;
        }
    }

    const scanParam = { ...state.scanParam }
    // 集群模式count设小点，因为后端会从所有master节点scan一遍然后合并结果,默认假设redis集群有3个master
    if (scanParam.mode == 'cluster') {
        scanParam.count = Math.floor(state.scanParam.count / 3)
    }

    state.loading = true;
    try {
        const res = await redisApi.scan.request(scanParam);
        state.keys = res.keys;
        state.dbsize = res.dbSize;
        state.scanParam.cursor = res.cursor;
    } finally {
        state.loading = false;
    }
};

const searchKey = async () => {
    state.scanParam.cursor = {};
    await scan();
};

const clear = () => {
    resetScanParam();
    if (state.scanParam.id) {
        scan();
    }
};

const resetScanParam = () => {
    state.scanParam.count = 10;
    state.scanParam.match = null;
    state.scanParam.cursor = {};
};

const showKeyDetail = async (row: any) => {
    const type = row.type;

    state.keyDetailDialog.keyInfo.type = type;
    state.keyDetailDialog.keyInfo.timed = row.ttl;
    state.keyDetailDialog.keyInfo.key = row.key;
    state.keyDetailDialog.visible = true;
};

const showNewKeyDialog = () => {
    notNull(state.scanParam.id, '请先选择redis');
    notNull(state.scanParam.db, "请选择要操作的库")
    resetKeyDetailInfo();
    state.newKeyDialog.visible = true;
}

const flushDb = () => {
    ElMessageBox.confirm(`确定清空[${state.scanParam.db}]库的所有key?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    }).then(() => {
        redisApi.flushDb
            .request({
                id: state.scanParam.id,
                db: state.scanParam.db,
            })
            .then(() => {
                ElMessage.success('清除成功！');
                searchKey();
            });
    }).catch(() => { });
}

const cancelNewKey = () => {
    resetKeyDetailInfo();
    state.newKeyDialog.visible = false;
}

const newKey = async () => {
    const keyInfo = state.keyDetailDialog.keyInfo
    const keyType = keyInfo.type
    const key = keyInfo.key;
    notBlank(key, "键名不能为空");

    if (keyType == 'string') {
        await redisApi.setString.request({
            id: state.scanParam.id,
            db: state.scanParam.db,
            key: key,
            value: '',
        })
    }
    state.newKeyDialog.visible = false;
    state.keyDetailDialog.visible = true;
    searchKey();
}

const resetKeyDetailInfo = () => {
    state.keyDetailDialog.keyInfo.key = '';
    state.keyDetailDialog.keyInfo.type = 'string';
    state.keyDetailDialog.keyInfo.timed = -1;
}

const del = (key: string) => {
    ElMessageBox.confirm(`确定删除[ ${key} ] 该key?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    }).then(() => {
        redisApi.delKey
            .request({
                key,
                id: state.scanParam.id,
                db: state.scanParam.db,
            })
            .then(() => {
                ElMessage.success('删除成功！');
                searchKey();
            });
    }).catch(() => { });
};

const ttlConveter = (ttl: any) => {
    if (ttl == -1 || ttl == 0) {
        return '永久';
    }
    if (!ttl) {
        ttl = 0;
    }
    let second = parseInt(ttl); // 秒
    let min = 0; // 分
    let hour = 0; // 小时
    let day = 0;
    if (second > 60) {
        min = parseInt(second / 60 + '');
        second = second % 60;
        if (min > 60) {
            hour = parseInt(min / 60 + '');
            min = min % 60;
            if (hour > 24) {
                day = parseInt(hour / 24 + '');
                hour = hour % 24;
            }
        }
    }
    let result = '' + second + 's';
    if (min > 0) {
        result = '' + min + 'm:' + result;
    }
    if (hour > 0) {
        result = '' + hour + 'h:' + result;
    }
    if (day > 0) {
        result = '' + day + 'd:' + result;
    }
    return result;
};

const getTypeColor = (type: string) => {
    if (type == 'string') {
        return '#E4F5EB';
    }
    if (type == 'hash') {
        return '#F9E2AE';
    }
    if (type == 'set') {
        return '#A8DEE0';
    }
};
</script>

<style lang="scss">
.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}
</style>
