<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="tmplApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.saveTmpl" type="primary" icon="plus" @click="editTmpl(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delTmpl" :disabled="state.selectionData.length < 1" @click="deleteTmpl()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #relateChannel="{ data }">
                <el-popover placement="top-start" trigger="click" width="auto">
                    <template #reference>
                        <el-link @click="getRelateChannels(data.id)" icon="view" type="primary" underline="never"></el-link>
                    </template>
                    <el-row v-for="item in state.relateChannels" :key="item.id">
                        {{ $t(EnumValue.getLabelByValue(ChannelTypeEnum, item.type)) }}
                        <el-divider direction="vertical" />
                        {{ item.code }}
                        <el-divider direction="vertical" />
                        {{ item.name }}
                    </el-row>
                </el-popover>
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <TmplEdit v-model:visible="editDialog.visible" :form="editDialog.data" :title="editDialog.title" @success="search" />

        <el-dialog width="500px" :title="$t('msg.sendMsg')" v-model="sendMsgDialog.visible">
            <el-form label-width="auto">
                <el-form-item prop="params" :label="$t('msg.params')">
                    <el-input v-model.trim="sendMsgDialog.params" type="textarea" rows="5" placeholder="JSON" clearable></el-input>
                </el-form-item>

                <AccountSelectFormItem multiple v-model="sendMsgDialog.receiverIds" />
            </el-form>

            <template #footer>
                <el-button @click="() => (sendMsgDialog.visible = false)">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="sendMsg()">{{ $t('msg.send') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import EnumValue from '@/common/Enum';
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import AccountSelectFormItem from '@/views/system/account/components/AccountSelectFormItem.vue';
import { onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { tmplApi } from '../api';
import { ChannelTypeEnum, TmplStatusEnum, TmplTypeEnum } from '../enums';
import TmplEdit from './TmplEdit.vue';
import type { MsgChannel, MsgTemplate } from '@/views/system/msg/types';

const perms = {
    saveTmpl: 'msg:tmpl:save',
    delTmpl: 'msg:tmpl:del',
    sendMsg: 'msg:tmpl:send',
};

const searchItems = [SearchItem.input('code', 'common.code')];
const columns = [
    TableColumn.new('code', 'common.code'),
    TableColumn.new('name', 'msg.name'),
    TableColumn.new('status', 'common.status').typeTag(TmplStatusEnum),
    TableColumn.new('msgType', 'common.type').typeTag(TmplTypeEnum),
    TableColumn.new('tmpl', 'msg.tmpl').canBeautify(),
    TableColumn.new('relateChannel', 'msg.relateChannel').isSlot().alignCenter(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.saveTmpl, perms.delTmpl]);
const actionColumn = TableColumn.new('action', 'common.operation')
    .setActions<MsgTemplate>([
        { key: 'edit', label: 'common.edit', type: 'primary', perm: perms.saveTmpl, onClick: (row) => editTmpl(row) },
        { key: 'send', label: 'msg.send', type: 'warning', perm: perms.sendMsg, onClick: (row) => showSendMsgDialog(row) },
    ])
    .fixedRight()
    .noShowOverflowTooltip()
    .alignCenter();

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');
const state = reactive({
    /**
     * 选中的数据
     */
    selectionData: [],
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
    relateChannelsVisible: false,
    relateChannels: [] as MsgChannel[],
    editDialog: {
        title: '',
        visible: false,
        data: null as MsgTemplate | null,
    },
    sendMsgDialog: {
        tmpl: null as MsgTemplate | null,
        title: '',
        visible: false,
        params: '',
        receiverIds: [],
    },
});

const { selectionData, query, editDialog, sendMsgDialog } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
});

const search = async () => {
    dataTableRef.value?.search();
};

const getRelateChannels = async (id: number) => {
    state.relateChannels = [];
    state.relateChannels = await tmplApi.relateChannels.request({ id });
};

const editTmpl = (data: MsgTemplate | false) => {
    if (!data) {
        state.editDialog.title = useI18nCreateTitle('msg.msgTmpl');
        state.editDialog.data = null;
    } else {
        state.editDialog.title = useI18nEditTitle('msg.msgTmpl');
        state.editDialog.data = data;
    }
    state.editDialog.visible = true;
};

const deleteTmpl = async () => {
    if (!(await useI18nDeleteConfirm(state.selectionData.map((x: MsgTemplate) => x.code).join('、')))) {
        return;
    }
    await tmplApi.del.request({ id: state.selectionData.map((x: MsgTemplate) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const showSendMsgDialog = (tmpl: MsgTemplate) => {
    state.sendMsgDialog.tmpl = tmpl;
    state.sendMsgDialog.params = '';
    state.sendMsgDialog.receiverIds = [];
    state.sendMsgDialog.visible = true;
};

const sendMsg = async () => {
    const tmpl = state.sendMsgDialog.tmpl as MsgTemplate;
    await tmplApi.sendMsg.request({
        code: tmpl.code,
        params: state.sendMsgDialog.params,
        receiverIds: state.sendMsgDialog.receiverIds,
    });
    Msg.operateSuccess();
    state.sendMsgDialog.visible = false;
};
</script>
<style lang="scss"></style>
