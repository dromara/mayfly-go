<template>
    <div>
        <page-table :page-api="logApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns"> </page-table>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive } from 'vue';
import { logApi, accountApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { LogTypeEnum } from '../enums';
import { OptionsApi, SearchItem } from '@/components/SearchForm';

const searchItems = [
    SearchItem.select('creatorId', '操作人')
        .withPlaceholder('请输入并选择账号')
        .withOptionsApi(
            OptionsApi.new(accountApi.list, { username: null })
                .withConvertFn((res: any) => {
                    const accounts = res.list;
                    return accounts.map((x: any) => {
                        return {
                            label: x.username,
                            value: x.id,
                        };
                    });
                })
                .isRemote('username')
        ),
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
});

const { query } = toRefs(state);
</script>
<style lang="scss"></style>
