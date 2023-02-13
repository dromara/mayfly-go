<template>
    <div>
        <el-row>
            <el-col :span="4">
                <redis-instance-tree @init-load-instances="initLoadInstances" @change-instance="changeInstance"
                    @change-schema="loadInitSchema" :instances="state.instances" />
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
                                <el-button @click="searchKey()" type="success" icon="search" plain></el-button>
                                <el-button @click="scan()" icon="bottom" plain>scan</el-button>
                                <el-popover placement="right" :width="200" trigger="click">
                                    <template #reference>
                                        <el-button type="primary" icon="plus" plain></el-button>
                                    </template>
                                    <el-tag @click="onAddData('string')" :color="getTypeColor('string')"
                                        style="cursor: pointer">string</el-tag>
                                    <el-tag @click="onAddData('hash')" :color="getTypeColor('hash')" class="ml5"
                                        style="cursor: pointer">hash</el-tag>
                                    <el-tag @click="onAddData('set')" :color="getTypeColor('set')" class="ml5"
                                        style="cursor: pointer">set</el-tag>
                                    <!-- <el-tag @click="onAddData('list')" :color="getTypeColor('list')" class="ml5" style="cursor: pointer">list</el-tag> -->
                                </el-popover>
                            </el-form-item>
                            <div style="float: right">
                                <span>keys: {{ state.dbsize }}</span>
                            </div>
                        </el-form>
                    </el-col>
                    <el-table v-loading="state.loading" :data="state.keys" stripe :highlight-current-row="true"
                        style="cursor: pointer">
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
                                <el-button @click="getValue(scope.row)" type="success" icon="search" plain
                                    size="small">查看
                                </el-button>
                                <el-button @click="del(scope.row.key)" type="danger" icon="delete" plain size="small">删除
                                </el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                </div>
            </el-col>
        </el-row>

        <div style="text-align: center; margin-top: 10px"></div>

        <hash-value v-model:visible="hashValueDialog.visible" :operationType="dataEdit.operationType"
            :title="dataEdit.title" :keyInfo="dataEdit.keyInfo" :redisId="scanParam.id" :db="scanParam.db"
            @cancel="onCancelDataEdit" @valChange="searchKey" />

        <string-value v-model:visible="stringValueDialog.visible" :operationType="dataEdit.operationType"
            :title="dataEdit.title" :keyInfo="dataEdit.keyInfo" :redisId="scanParam.id" :db="scanParam.db"
            @cancel="onCancelDataEdit" @valChange="searchKey" />

        <set-value v-model:visible="setValueDialog.visible" :title="dataEdit.title" :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id" :db="scanParam.db" :operationType="dataEdit.operationType" @valChange="searchKey"
            @cancel="onCancelDataEdit" />

        <list-value v-model:visible="listValueDialog.visible" :title="dataEdit.title" :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id" :db="scanParam.db" :operationType="dataEdit.operationType" @valChange="searchKey"
            @cancel="onCancelDataEdit" />
    </div>
</template>

<script lang="ts" setup>
import { redisApi } from './api';
import { toRefs, reactive } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import HashValue from './HashValue.vue';
import StringValue from './StringValue.vue';
import SetValue from './SetValue.vue';
import ListValue from './ListValue.vue';
import { isTrue, notBlank, notNull } from '@/common/assert';

import RedisInstanceTree from '@/views/ops/redis/RedisInstanceTree.vue';

const state = reactive({
    loading: false,
    tags: [],
    redisList: [] as any,
    dbList: [],
    query: {
        tagPath: null,
    },
    scanParam: {
        id: null as any,
        mode: '',
        db: '',
        match: null,
        count: 10,
        cursor: {},
    },
    dataEdit: {
        visible: false,
        title: '新增数据',
        operationType: 1,
        keyInfo: {
            type: 'string',
            timed: -1,
            key: '',
        },
    },
    hashValueDialog: {
        visible: false,
    },
    stringValueDialog: {
        visible: false,
    },
    setValueDialog: {
        visible: false,
    },
    listValueDialog: {
        visible: false,
    },
    keys: [],
    dbsize: 0,
    instances: { tags: {}, tree: {}, dbs: {}, tables: {} }
});

const {
    scanParam,
    dataEdit,
    hashValueDialog,
    stringValueDialog,
    setValueDialog,
    listValueDialog,
} = toRefs(state)

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

const getValue = async (row: any) => {
    const type = row.type;

    state.dataEdit.keyInfo.type = type;
    state.dataEdit.keyInfo.timed = row.ttl;
    state.dataEdit.keyInfo.key = row.key;
    state.dataEdit.operationType = 2;
    state.dataEdit.title = '查看数据';

    if (type == 'hash') {
        state.hashValueDialog.visible = true;
    } else if (type == 'string') {
        state.stringValueDialog.visible = true;
    } else if (type == 'set') {
        state.setValueDialog.visible = true;
    } else if (type == 'list') {
        state.listValueDialog.visible = true;
    } else {
        ElMessage.warning('暂不支持该类型');
    }
};

const onAddData = (type: string) => {
    notNull(state.scanParam.id, '请先选择redis');
    state.dataEdit.operationType = 1;
    state.dataEdit.title = '新增数据';
    state.dataEdit.keyInfo.type = type;
    state.dataEdit.keyInfo.timed = -1;
    if (type == 'hash') {
        state.hashValueDialog.visible = true;
    } else if (type == 'string') {
        state.stringValueDialog.visible = true;
    } else if (type == 'set') {
        state.setValueDialog.visible = true;
    } else if (type == 'list') {
        state.listValueDialog.visible = true;
    } else {
        ElMessage.warning('暂不支持该类型');
    }
};

const onCancelDataEdit = () => {
    state.dataEdit.keyInfo = {} as any;
};

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


const initLoadInstances = async () => {
    const res = await redisApi.redisList.request({});
    if (!res.total) return
    state.instances = { tags: {}, tree: {}, dbs: {}, tables: {} }; // 初始化变量
    for (const db of res.list) {
        let arr = state.instances.tree[db.tagId] || []
        const { tagId, tagPath } = db
        // tags
        state.instances.tags[db.tagId] = { tagId, tagPath }
        // 实例
        arr.push(db)
        state.instances.tree[db.tagId] = arr;
    }
}

const changeInstance = async (inst: any, fn: Function) => {
    let dbs = inst.db.split(',').map((x: string) => {
        return { name: `db${x}`, keys: 0 }
    })
    const res = await redisApi.redisInfo.request({ id: inst.id, host: inst.host, section: "Keyspace" });
    for (let db in res.Keyspace) {
        for (let d of dbs) {
            if (db == d.name) {
                d.keys = res.Keyspace[db]?.split(',')[0]?.split('=')[1] || 0
            }
        }
    }

    state.instances.dbs[inst.id] = dbs
    fn && fn(dbs)
}

/** 初始化加载db数据 */
const loadInitSchema = (inst: any, schema: string) => {
    state.scanParam.id = inst.id
    state.scanParam.db = schema.replace('db', '')
    scan()
}

</script>

<style>

</style>
