<template>
    <div>
        <page-table :search-items="searchItems" v-model:query-form="query" :columns="columns" :page-api="logApi.list">
            <template #selectAccount>
                <el-select remote :remote-method="getAccount" v-model="query.creatorId" filterable placeholder="请输入并选择账号" clearable>
                    <el-option v-for="item in accounts" :key="item.id" :label="item.username" :value="item.id"> </el-option>
                </el-select>
            </template>
        </page-table>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive } from 'vue';
import { logApi, accountApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { LogTypeEnum } from '../enums';
import { SearchItem } from '@/components/SearchForm';

const searchItems = [
    SearchItem.slot('creatorId', '操作人', 'selectAccount'),
    SearchItem.select('type', '操作结果').withEnum(LogTypeEnum),
    SearchItem.input('description', '描述'),
];

const columns = [
    TableColumn.new('creator', '操作人'),
    TableColumn.new('createTime', '操作时间').isTime(),
    TableColumn.new('type', '结果').typeTag(LogTypeEnum),
    TableColumn.new('description', '描述'),
    TableColumn.new('reqParam', '操作信息').canBeautify(),
    TableColumn.new('resp', '响应信息'),
];

const state = reactive({
    query: {
        type: null,
        creatorId: null,
        description: null,
        pageNum: 1,
        pageSize: 0,
    },
    accounts: [] as any,
});

const { query, accounts } = toRefs(state);

const getAccount = (username: any) => {
    if (!username) {
        return;
    }
    accountApi.list.request({ username }).then((res) => {
        state.accounts = res.list;
    });
};
</script>
<style lang="scss"></style>
