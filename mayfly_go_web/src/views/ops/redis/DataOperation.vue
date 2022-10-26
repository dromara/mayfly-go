<template>
    <div>
        <el-card>
            <div style="float: left">
                <el-row type="flex" justify="space-between">
                    <el-col :span="24">
                        <el-form class="search-form" label-position="right" :inline="true">
                            <el-form-item label="标签">
                                <el-select
                                    @change="changeTag"
                                    @focus="getTags"
                                    v-model="query.tagPath"
                                    placeholder="请选择标签"
                                    filterable
                                    style="width: 250px"
                                >
                                    <el-option v-for="item in tags" :key="item" :label="item" :value="item"> </el-option>
                                </el-select>
                            </el-form-item>
                            <el-form-item label="redis" label-width="40px">
                                <el-select
                                    v-model="scanParam.id"
                                    placeholder="请选择redis"
                                    @change="changeRedis"
                                    @clear="clearRedis"
                                    clearable
                                    style="width: 250px"
                                >
                                    <el-option
                                        v-for="item in redisList"
                                        :key="item.id"
                                        :label="`${item.name ? item.name : ''} [${item.host}]`"
                                        :value="item.id"
                                    >
                                    </el-option>
                                </el-select>
                            </el-form-item>
                            <el-form-item label="库" label-width="20px">
                                <el-select v-model="scanParam.db" @change="changeDb" placeholder="库" style="width: 85px">
                                    <el-option v-for="db in dbList" :key="db" :label="db" :value="db"> </el-option>
                                </el-select>
                            </el-form-item>
                        </el-form>
                    </el-col>
                    <el-col class="mt10">
                        <el-form class="search-form" label-position="right" :inline="true" label-width="60px">
                            <el-form-item label="key" label-width="40px">
                                <el-input
                                    placeholder="match 支持*模糊key"
                                    style="width: 250px"
                                    v-model="scanParam.match"
                                    @clear="clear()"
                                    clearable
                                ></el-input>
                            </el-form-item>
                            <el-form-item label="count" label-width="40px">
                                <el-input placeholder="count" style="width: 70px" v-model.number="scanParam.count"></el-input>
                            </el-form-item>
                            <el-form-item>
                                <el-button @click="searchKey()" type="success" icon="search" plain></el-button>
                                <el-button @click="scan()" icon="bottom" plain>scan</el-button>
                                <el-popover placement="right" :width="200" trigger="click">
                                    <template #reference>
                                        <el-button type="primary" icon="plus" plain></el-button>
                                    </template>
                                    <el-tag @click="onAddData('string')" :color="getTypeColor('string')" style="cursor: pointer">string</el-tag>
                                    <el-tag @click="onAddData('hash')" :color="getTypeColor('hash')" class="ml5" style="cursor: pointer">hash</el-tag>
                                    <el-tag @click="onAddData('set')" :color="getTypeColor('set')" class="ml5" style="cursor: pointer">set</el-tag>
                                    <!-- <el-tag @click="onAddData('list')" :color="getTypeColor('list')" class="ml5" style="cursor: pointer">list</el-tag> -->
                                </el-popover>
                            </el-form-item>
                            <div style="float: right">
                                <span>keys: {{ dbsize }}</span>
                            </div>
                        </el-form>
                    </el-col>
                </el-row>
            </div>

            <el-table v-loading="loading" :data="keys" stripe :highlight-current-row="true" style="cursor: pointer">
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
                        <el-button @click="getValue(scope.row)" type="success" icon="search" plain size="small">查看</el-button>
                        <el-button @click="del(scope.row.key)" type="danger" icon="delete" plain size="small">删除</el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <div style="text-align: center; margin-top: 10px"></div>

        <hash-value
            v-model:visible="hashValueDialog.visible"
            :operationType="dataEdit.operationType"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
            :db="scanParam.db"
            @cancel="onCancelDataEdit"
            @valChange="searchKey"
        />

        <string-value
            v-model:visible="stringValueDialog.visible"
            :operationType="dataEdit.operationType"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
            :db="scanParam.db"
            @cancel="onCancelDataEdit"
            @valChange="searchKey"
        />

        <set-value
            v-model:visible="setValueDialog.visible"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
            :db="scanParam.db"
            :operationType="dataEdit.operationType"
            @valChange="searchKey"
            @cancel="onCancelDataEdit"
        />

        <list-value
            v-model:visible="listValueDialog.visible"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
            :db="scanParam.db"
            :operationType="dataEdit.operationType"
            @valChange="searchKey"
            @cancel="onCancelDataEdit"
        />
    </div>
</template>

<script lang="ts">
import { redisApi } from './api';
import { toRefs, reactive, defineComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import HashValue from './HashValue.vue';
import StringValue from './StringValue.vue';
import SetValue from './SetValue.vue';
import ListValue from './ListValue.vue';
import { isTrue, notBlank, notNull } from '@/common/assert';
import { tagApi } from '../tag/api.ts';

export default defineComponent({
    name: 'DataOperation',
    components: {
        StringValue,
        HashValue,
        SetValue,
        ListValue,
    },
    setup() {
        const state = reactive({
            loading: false,
            tags: [],
            redisList: [] as any,
            dbList: [],
            query: {
                tagPath: null,
            },
            scanParam: {
                id: null,
                db: null,
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
        });

        const searchRedis = async () => {
            notBlank(state.query.tagPath, '请先选择标签');
            const res = await redisApi.redisList.request(state.query);
            state.redisList = res.list;
        };

        const changeTag = (tagPath: string) => {
            clearRedis();
            if (tagPath != null) {
                searchRedis();
            }
        };

        const getTags = async () => {
            state.tags = await tagApi.getAccountTags.request(null);
        };

        const changeRedis = (id: number) => {
            resetScanParam(id);
            state.scanParam.db = null;
            state.dbList = (state.redisList.find((x: any) => x.id == id) as any).db.split(',');
            state.keys = [];
            state.dbsize = 0;
        };

        const changeDb = () => {
            resetScanParam(state.scanParam.id as any);
            state.keys = [];
            state.dbsize = 0;
            searchKey();
        };

        const scan = async () => {
            isTrue(state.scanParam.id != null, '请先选择redis');
            notBlank(state.scanParam.count, 'count不能为空');

            const match = state.scanParam.match;
            if (!match || (match as string).length < 4) {
                isTrue(state.scanParam.count <= 200, 'key为空或小于4字符时, count不能超过200');
            } else {
                isTrue(state.scanParam.count <= 20000, 'count不能超过20000');
            }

            state.loading = true;
            try {
                const res = await redisApi.scan.request(state.scanParam);
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

        const clearRedis = () => {
            state.redisList = [];
            state.scanParam.id = null;
            resetScanParam();
            state.scanParam.db = null;
            state.keys = [];
            state.dbsize = 0;
        };

        const clear = () => {
            resetScanParam();
            if (state.scanParam.id) {
                scan();
            }
        };

        const resetScanParam = (id: number = 0) => {
            state.scanParam.count = 10;
            if (id != 0) {
                const redis: any = state.redisList.find((x: any) => x.id == id);
                // 集群模式count设小点，因为后端会从所有master节点scan一遍然后合并结果
                if (redis && redis.mode == 'cluster') {
                    state.scanParam.count = 4;
                }
            }
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
            })
                .then(() => {
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
                })
                .catch(() => {});
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

        return {
            ...toRefs(state),
            getTags,
            changeTag,
            changeRedis,
            changeDb,
            clearRedis,
            searchKey,
            scan,
            clear,
            getValue,
            del,
            ttlConveter,
            getTypeColor,
            onAddData,
            onCancelDataEdit,
        };
    },
});
</script>

<style>
</style>
