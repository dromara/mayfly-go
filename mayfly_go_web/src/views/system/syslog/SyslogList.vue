<template>
    <div class="role-list">
        <el-card>
            <div style="float: right">
                <el-select
                    remote
                    :remote-method="getAccount"
                    v-model="query.creatorId"
                    filterable
                    placeholder="请输入并选择账号"
                    clearable
                    class="mr5"
                >
                    <el-option v-for="item in accounts" :key="item.id" :label="item.username" :value="item.id"> </el-option>
                </el-select>
                <el-select v-model="query.type" filterable placeholder="请选择操作结果" clearable class="mr5">
                    <el-option label="成功" :value="1"> </el-option>
                    <el-option label="失败" :value="2"> </el-option>
                </el-select>
                <el-button @click="search" type="success" icon="search"></el-button>
            </div>
            <el-table :data="logs" style="width: 100%">
                <el-table-column prop="creator" label="操作人" min-width="100" show-overflow-tooltip></el-table-column>
                <el-table-column prop="createTime" label="操作时间" min-width="160">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="type" label="结果" min-width="65">
                    <template #default="scope">
                        <el-tag v-if="scope.row.type == 1" type="success" size="small">成功</el-tag>
                        <el-tag v-if="scope.row.type == 2" type="danger" size="small">失败</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip></el-table-column>

                <el-table-column prop="reqParam" label="请求信息" min-width="300" show-overflow-tooltip></el-table-column>
                <el-table-column prop="resp" label="响应信息" min-width="200" show-overflow-tooltip></el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination
                    style="text-align: right"
                    @current-change="handlePageChange"
                    :total="total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"
                ></el-pagination>
            </el-row>
        </el-card>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { logApi, accountApi } from '../api';
export default defineComponent({
    name: 'SyslogList',
    components: {},
    setup() {
        const state = reactive({
            query: {
                pageNum: 1,
                pageSize: 10,
                name: null,
            },
            total: 0,
            logs: [],
            accounts: [],
        });

        onMounted(() => {
            search();
        });

        const search = async () => {
            let res = await logApi.list.request(state.query);
            state.logs = res.list;
            state.total = res.total;
        };

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            search();
        };

        const getAccount = (username: any) => {
            accountApi.list.request({ username }).then((res) => {
                state.accounts = res.list;
            });
        };

        return {
            ...toRefs(state),
            search,
            handlePageChange,
            getAccount,
        };
    },
});
</script>
<style lang="scss">
</style>
