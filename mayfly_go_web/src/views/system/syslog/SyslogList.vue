<template>
    <div>
        <page-table ref="pageTableRef" :query="state.queryConfig" v-model:query-form="query" :data="logs"
            :columns="state.columns" :total="total" v-model:page-size="query.pageSize" v-model:page-num="query.pageNum"
            @pageChange="search()">

            <template #selectAccount>
                <el-select remote :remote-method="getAccount" v-model="query.creatorId" filterable placeholder="请输入并选择账号"
                    clearable class="mr5" style="width: 200px">
                    <el-option v-for="item in accounts" :key="item.id" :label="item.username" :value="item.id">
                    </el-option>
                </el-select>
            </template>

            <template #type="{ data }">
                <el-tag v-if="data.type == 1" type="success" size="small">成功</el-tag>
                <el-tag v-if="data.type == 2" type="danger" size="small">失败</el-tag>
            </template>
        </page-table>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted } from 'vue';
import { logApi, accountApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue'
import { TableColumn, TableQuery } from '@/components/pagetable';

const pageTableRef: any = ref(null);

const state = reactive({
    query: {
        type: null,
        creatorId: null,
        pageNum: 1,
        pageSize: 10,
    },
    queryConfig: [
        TableQuery.slot("creatorId", "操作人", "selectAccount"),
        TableQuery.select("type", "操作结果").setOptions([
            { label: "成功", value: 1 },
            { label: "失败", value: 2 },
        ]),
    ],
    columns: [
        TableColumn.new("creator", "操作人"),
        TableColumn.new("createTime", "操作时间").isTime(),
        TableColumn.new("type", "结果").isSlot(),
        TableColumn.new("description", "描述"),
        TableColumn.new("reqParam", "操作信息"),
        TableColumn.new("resp", "响应信息"),
    ],
    total: 0,
    logs: [],
    accounts: [] as any,
});

const {
    query,
    total,
    logs,
    accounts,
} = toRefs(state)

onMounted(() => {
    search();
});

const search = async () => {
    try {
        pageTableRef.value.loading(true);
        let res = await logApi.list.request(state.query);
        state.logs = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false);
    }
};

const getAccount = (username: any) => {
    accountApi.list.request({ username }).then((res) => {
        state.accounts = res.list;
    });
};

</script>
<style lang="scss"></style>
