<template>
    <div>
        <el-card>
            <div style="float: left">
                <el-row type="flex" justify="space-between">
                    <el-col :span="24">
                        <project-env-select @changeProjectEnv="changeProjectEnv" @clear="clearRedis">
                            <template #default>
                                <el-form-item label="redis" label-width="40px">
                                    <el-select v-model="scanParam.id" placeholder="请选择redis" @change="changeRedis" @clear="clearRedis" clearable>
                                        <el-option v-for="item in redisList" :key="item.id" :label="item.host" :value="item.id">
                                            <span style="float: left">{{ item.host }}</span>
                                            <span style="float: right; color: #8492a6; margin-left: 6px; font-size: 13px">{{
                                                `库: [${item.db}]`
                                            }}</span>
                                        </el-option>
                                    </el-select>
                                </el-form-item>
                            </template>
                        </project-env-select>
                    </el-col>
                    <el-col class="mt10">
                        <el-form class="search-form" label-position="right" :inline="true" label-width="60px">
                            <el-form-item label="key" label-width="40px">
                                <el-input
                                    placeholder="match 支持*模糊key"
                                    style="width: 240px"
                                    v-model="scanParam.match"
                                    @clear="clear()"
                                    clearable
                                ></el-input>
                            </el-form-item>
                            <el-form-item label="count" label-width="60px">
                                <el-input placeholder="count" style="width: 62px" v-model.number="scanParam.count"></el-input>
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
                <el-table-column prop="ttl" label="ttl(过期时间)" width="130">
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
            @cancel="onCancelDataEdit"
            @valChange="searchKey"
        />

        <string-value
            v-model:visible="stringValueDialog.visible"
            :operationType="dataEdit.operationType"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
            @cancel="onCancelDataEdit"
            @valChange="searchKey"
        />

        <set-value
            v-model:visible="setValueDialog.visible"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
            :operationType="dataEdit.operationType"
            @valChange="searchKey"
            @cancel="onCancelDataEdit"
        />

        <list-value
            v-model:visible="listValueDialog.visible"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
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
import ProjectEnvSelect from '../component/ProjectEnvSelect.vue';
import HashValue from './HashValue.vue';
import StringValue from './StringValue.vue';
import SetValue from './SetValue.vue';
import ListValue from './ListValue.vue';
import { isTrue, notBlank, notNull } from '@/common/assert';

export default defineComponent({
    name: 'DataOperation',
    components: {
        StringValue,
        HashValue,
        SetValue,
        ListValue,
        ProjectEnvSelect,
    },
    setup() {
        const state = reactive({
            loading: false,
            redisList: [],
            query: {
                envId: 0,
            },
            scanParam: {
                id: null,
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
            notNull(state.query.envId, '请先选择项目环境');
            const res = await redisApi.redisList.request(state.query);
            state.redisList = res.list;
        };

        const changeProjectEnv = (projectId: any, envId: any) => {
            clearRedis();
            if (envId != null) {
                state.query.envId = envId;
                searchRedis();
            }
        };

        const changeRedis = (id: number) => {
            resetScanParam(id);
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
                    state.scanParam.count = 5;
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
            changeProjectEnv,
            changeRedis,
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
