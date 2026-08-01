<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="channelApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.saveChannel" type="primary" icon="plus" @click="editChannel(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delChannel" :disabled="state.selectionData.length < 1" @click="deleteChannel()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #action="{ data }">
                <el-button link v-auth="perms.saveChannel" @click="editChannel(data)" type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <ChannelEdit v-model:visible="editDialog.visible" :form="editDialog.data" :title="editDialog.title" @success="search" />
    </div>
</template>

<script lang="ts" setup>
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { channelApi } from '../api';
import { ChannelStatusEnum, ChannelTypeEnum } from '../enums';
import ChannelEdit from './ChannelEdit.vue';
import type { MsgChannel } from '@/views/system/msg/types';

const perms = {
    saveChannel: 'msg:channel:save',
    delChannel: 'msg:channel:del',
};

const searchItems = [SearchItem.input('name', 'msg.name')];
const columns = [
    TableColumn.new('code', 'common.code'),
    TableColumn.new('name', 'msg.name'),
    TableColumn.new('status', 'common.status').typeTag(ChannelStatusEnum),
    TableColumn.new('type', 'common.type').typeTag(ChannelTypeEnum).setAddWidth(15),
    TableColumn.new('url', 'URL'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.saveChannel, perms.delChannel]);
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter();

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const state = reactive({
    /**
     * 选中的数据
     */
    selectionData: [] as MsgChannel[],
    /**
     * 查询条件
     */
    query: {
        name: '',
        code: '',
        type: '',
        pageNum: 1,
        pageSize: 0,
    },
    editDialog: {
        title: '',
        visible: false,
        data: null as MsgChannel | null,
    },
});

const { selectionData, query, editDialog } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
});

const search = async () => {
    pageTableRef.value?.search();
};

const editChannel = (data: MsgChannel | false) => {
    if (!data) {
        state.editDialog.title = useI18nCreateTitle('msg.msgChannel');
        state.editDialog.data = null;
    } else {
        state.editDialog.title = useI18nEditTitle('msg.msgChannel');
        state.editDialog.data = data;
    }
    state.editDialog.visible = true;
};

const deleteChannel = async () => {
    await useI18nDeleteConfirm(state.selectionData.map((x) => x.code).join('、'));
    await channelApi.del.request({ id: state.selectionData.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};
</script>
<style lang="scss"></style>
