<template>
    <div>
        <page-table
            ref="pageTableRef"
            :query="state.queryConfig"
            v-model:query-form="query"
            :data="logs"
            :columns="state.columns"
            :total="total"
            v-model:page-size="query.pageSize"
            v-model:page-num="query.pageNum"
            @pageChange="search()"
        >
            <template #selectAccount>
                <el-select
                    style="width: 200px"
                    remote
                    :remote-method="getAccount"
                    v-model="query.creatorId"
                    filterable
                    placeholder="请输入并选择账号"
                    clearable
                >
                    <el-option v-for="item in accounts" :key="item.id" :label="item.username" :value="item.id"> </el-option>
                </el-select>
            </template>
        </page-table>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted } from 'vue';
import { logApi, accountApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn, TableQuery } from '@/components/pagetable';
import { LogTypeEnum } from '../enums';

const pageTableRef: any = ref(null);

const state = reactive({
    query: {
        type: null,
        creatorId: null,
        description: null,
        pageNum: 1,
        pageSize: 0,
    },
    queryConfig: [
        TableQuery.slot('creatorId', '操作人', 'selectAccount'),
        TableQuery.select('type', '操作结果').setOptions(Object.values(LogTypeEnum)),
        TableQuery.text('description', '描述'),
    ],
    columns: [
        TableColumn.new('creator', '操作人'),
        TableColumn.new('createTime', '操作时间').isTime(),
        TableColumn.new('type', '结果').typeTag(LogTypeEnum),
        TableColumn.new('description', '描述'),
        TableColumn.new('reqParam', '操作信息').canBeautify(),
        TableColumn.new('resp', '响应信息'),
    ],
    total: 0,
    logs: [],
    accounts: [] as any,
});

const { query, total, logs, accounts } = toRefs(state);

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
