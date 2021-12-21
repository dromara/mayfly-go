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
                                <el-form-item label="key" label-width="40px">
                                    <el-input
                                        placeholder="支持*模糊key"
                                        style="width: 180px"
                                        v-model="scanParam.match"
                                        size="mini"
                                        @clear="clear()"
                                        clearable
                                    ></el-input>
                                </el-form-item>
                                <el-form-item label-width="40px">
                                    <el-input placeholder="count" style="width: 62px" v-model="scanParam.count" size="mini"></el-input>
                                </el-form-item>
                                <el-button @click="searchKey()" type="success" icon="el-icon-search" size="mini" plain></el-button>
                                <el-button @click="scan()" icon="el-icon-bottom" size="mini" plain>scan</el-button>
                                <el-button type="primary" icon="el-icon-plus" size="mini" @click="save(false)" plain></el-button>
                            </template>
                        </project-env-select>
                    </el-col>
                </el-row>
            </div>
            <div style="float: right">
                <!-- <el-button @click="scan()" icon="el-icon-refresh" size="small" plain>刷新</el-button> -->
                <span>keys: {{ dbsize }}</span>
            </div>
            <el-table v-loading="loading" :data="keys" stripe :highlight-current-row="true" style="cursor: pointer">
                <el-table-column show-overflow-tooltip prop="key" label="key"></el-table-column>
                <el-table-column prop="type" label="type" width="80"> </el-table-column>
                <el-table-column prop="ttl" label="ttl(过期时间)" width="120">
                    <template #default="scope">
                        {{ ttlConveter(scope.row.ttl) }}
                    </template>
                </el-table-column>
                <el-table-column label="操作">
                    <template #default="scope">
                        <el-button @click="getValue(scope.row)" type="success" icon="el-icon-search" size="mini" plain>查看</el-button>
                        <el-button @click="del(scope.row.key)" type="danger" size="mini" icon="el-icon-delete" plain>删除</el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <div style="text-align: center; margin-top: 10px"></div>

        <value-dialog v-model:visible="valueDialog.visible" :keyValue="valueDialog.value" />
    </div>
</template>

<script lang="ts">
import ValueDialog from './ValueDialog.vue';
import { redisApi } from './api';
import { toRefs, reactive, defineComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import ProjectEnvSelect from '../component/ProjectEnvSelect.vue';
import { isTrue, notNull } from '@/common/assert';

export default defineComponent({
    name: 'DataOperation',
    components: {
        ValueDialog,
        ProjectEnvSelect,
    },
    setup() {
        const state = reactive({
            loading: false,
            cluster: 0,
            redisList: [],
            query: {
                envId: 0,
            },
            // redis: {
            //     id: 0,
            //     info: '',
            //     conf: '',
            // },
            scanParam: {
                id: null,
                cluster: 0,
                match: null,
                count: 10,
                cursor: 0,
                prevCursor: null,
            },
            valueDialog: {
                visible: false,
                value: {},
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

        const changeRedis = () => {
            resetScanParam();
            state.keys = [];
            state.dbsize = 0;
            searchKey();
        };

        const scan = () => {
            isTrue(state.scanParam.id != null, '请先选择redis');
            isTrue(state.scanParam.count < 2001, 'count不能超过2000');

            state.loading = true;
            state.scanParam.cluster = state.cluster == 0 ? 0 : 1;

            redisApi.scan.request(state.scanParam).then((res) => {
                state.keys = res.keys;
                state.dbsize = res.dbSize;
                state.scanParam.cursor = res.cursor;
                state.loading = false;
            });
        };

        const searchKey = () => {
            state.scanParam.cursor = 0;
            scan();
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

        const resetScanParam = () => {
            state.scanParam.match = null;
            state.scanParam.cursor = 0;
            state.scanParam.count = 10;
        };

        const getValue = async (row: any) => {
            let api: any;
            switch (row.type) {
                case 'string':
                    api = redisApi.getStringValue;
                    break;
                case 'hash':
                    api = redisApi.getHashValue;
                    break;
                case 'set':
                    api = redisApi.getSetValue;
                    break;
                default:
                    api = redisApi.getStringValue;
                    break;
            }
            const id = state.cluster == 0 ? state.scanParam.id : state.cluster;
            const res = await api.request({
                cluster: state.cluster,
                key: row.key,
                id,
            });

            let timed = row.ttl == 18446744073709552000 ? 0 : row.ttl;
            state.valueDialog.value = { id: state.scanParam.id, key: row.key, value: res, timed: timed, type: row.type };
            state.valueDialog.visible = true;
        };

        // closeValueDialog() {
        //   this.valueDialog.visible = false
        //   this.valueDialog.value = {}
        // }

        // const update = (key: string) => {};

        const del = (key: string) => {
            ElMessageBox.confirm(`此操作将删除对应的key , 是否继续?`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            })
                .then(() => {
                    let id = state.cluster == 0 ? state.scanParam.id : state.cluster;
                    redisApi.delKey
                        .request({
                            cluster: state.cluster,
                            key,
                            id,
                        })
                        .then(() => {
                            ElMessage.success('删除成功！');
                            scan();
                        });
                })
                .catch(() => {});
        };

        const ttlConveter = (ttl: any) => {
            if (ttl == 18446744073709552000) {
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
        };
    },
});
</script>

<style>
</style>
