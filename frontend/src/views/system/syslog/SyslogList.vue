<template>
    <div class="h-full">
        <page-table :page-api="logApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns">
            <template #creator="{ data }">
                <account-info :username="data.creator || ''" />
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
import { OptionsApi, SearchItem } from '@/components/SearchForm';
import AccountInfo from '../account/components/AccountInfo.vue';

const searchItems = [
    SearchItem.select('creatorId', 'system.syslog.operator')
        .withPlaceholder('system.syslog.operatorPlaceholder')
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
    SearchItem.select('type', 'system.syslog.result').withEnum(LogTypeEnum),
    SearchItem.input('description', 'system.syslog.description'),
];

const columns = [
    TableColumn.new('creator', 'system.syslog.operator').isSlot().noShowOverflowTooltip(),
    TableColumn.new('createTime', 'system.syslog.operatingTime').isTime(),
    TableColumn.new('description', 'system.syslog.description'),
    TableColumn.new('type', 'system.syslog.result').typeTag(LogTypeEnum),
    TableColumn.new('reqParam', 'system.syslog.operatingInfo').canBeautify(),
    TableColumn.new('resp', 'system.syslog.response').canBeautify(),
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
