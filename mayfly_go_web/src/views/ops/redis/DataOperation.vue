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
                                    placeholder="支持*模糊key"
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
                                <el-button type="primary" icon="plus" @click="onAddData(false)" plain></el-button>
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

        <!-- <value-dialog v-model:visible="valueDialog.visible" :keyValue="valueDialog.value" /> -->

        <data-edit
            v-model:visible="dataEdit.visible"
            :title="dataEdit.title"
            :keyInfo="dataEdit.keyInfo"
            :redisId="scanParam.id"
            :operationType="dataEdit.operationType"
            :stringValue="dataEdit.stringValue"
            :setValue="dataEdit.setValue"
            :hashValue="dataEdit.hashValue"
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
import DataEdit from './DataEdit.vue';
import { isTrue, notBlank, notNull } from '@/common/assert';

export default defineComponent({
    name: 'DataOperation',
    components: {
        DataEdit,
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
            valueDialog: {
                visible: false,
                value: {},
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
                stringValue: '',
                hashValue: [{ key: '', value: '' }],
                setValue: [{ value: '' }],
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
            isTrue(state.scanParam.count < 20001, 'count不能超过20000');

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
            const key = row.key;

            let res: any;
            const reqParam = {
                key: row.key,
                id: state.scanParam.id,
            };
            switch (type) {
                case 'string':
                    res = await redisApi.getStringValue.request(reqParam);
                    break;
                case 'hash':
                    res = await redisApi.getHashValue.request(reqParam);
                    break;
                case 'set':
                    res = await redisApi.getSetValue.request(reqParam);
                    break;
                default:
                    res = null;
                    break;
            }
            notNull(res, '暂不支持该类型数据查看');

            if (type == 'string') {
                state.dataEdit.stringValue = res;
            }
            if (type == 'set') {
                state.dataEdit.setValue = res.map((x: any) => {
                    return {
                        value: x,
                    };
                });
            }
            if (type == 'hash') {
                const hash = [];
                //遍历key和value
                const keys = Object.keys(res);
                for (let i = 0; i < keys.length; i++) {
                    const key = keys[i];
                    const value = res[key];
                    hash.push({
                        key,
                        value,
                    });
                }
                state.dataEdit.hashValue = hash;
            }

            state.dataEdit.keyInfo.type = type;
            state.dataEdit.keyInfo.timed = row.ttl;
            state.dataEdit.keyInfo.key = key;
            state.dataEdit.operationType = 2;
            state.dataEdit.title = '修改数据';
            state.dataEdit.visible = true;
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

        const onAddData = () => {
            notNull(state.scanParam.id, '请先选择redis');
            state.dataEdit.operationType = 1;
            state.dataEdit.title = '新增数据';
            state.dataEdit.visible = true;
        };

        const onCancelDataEdit = () => {
            state.dataEdit.keyInfo = {} as any;
            state.dataEdit.stringValue = '';
            state.dataEdit.setValue = [];
            state.dataEdit.hashValue = [];
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
