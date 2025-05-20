<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
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

            <template #action="{ data }">
                <el-button link v-auth="perms.saveTmpl" @click="editTmpl(data)" type="primary">{{ $t('common.edit') }}</el-button>
                <el-button link v-auth="perms.sendMsg" @click="showSendMsgDialog(data)" type="warning">{{ $t('msg.send') }}</el-button>
            </template>
        </page-table>

        <TmplEdit v-model:visible="editDialog.visible" :form="editDialog.data" :title="editDialog.title" @success="search" />

        <el-dialog width="500px" :title="$t('msg.sendMsg')" v-model="sendMsgDialog.visible">
            <el-form label-width="auto">
                <el-form-item prop="params" :label="$t('params')">
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
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import { tmplApi } from '../api';
import { TmplStatusEnum, TmplTypeEnum, ChannelTypeEnum } from '../enums';
import TmplEdit from './TmplEdit.vue';
import EnumValue from '../../../common/Enum';
import AccountSelectFormItem from '@/views/system/account/components/AccountSelectFormItem.vue';

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
    TableColumn.new('msgType', 'common.type').typeTag(TmplTypeEnum).setAddWidth(20),
    TableColumn.new('tmpl', 'msg.tmpl').canBeautify(),
    TableColumn.new('relateChannel', 'msg.relateChannel').isSlot().alignCenter(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.saveTmpl, perms.delTmpl]);
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter();

const pageTableRef: Ref<any> = ref(null);
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
    relateChannels: [] as any,
    editDialog: {
        title: '',
        visible: false,
        data: null as any,
    },
    sendMsgDialog: {
        tmpl: null,
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
    pageTableRef.value.search();
};

const getRelateChannels = async (id: number) => {
    state.relateChannels = [];
    state.relateChannels = await tmplApi.relateChannels.request({ id });
};

const editTmpl = (data: any) => {
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
    await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.code).join('、'));
    await tmplApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
    useI18nDeleteSuccessMsg();
    search();
};

const showSendMsgDialog = (tmpl: any) => {
    state.sendMsgDialog.tmpl = tmpl;
    state.sendMsgDialog.params = '';
    state.sendMsgDialog.receiverIds = [];
    state.sendMsgDialog.visible = true;
};

const sendMsg = async () => {
    const tmpl: any = state.sendMsgDialog.tmpl;
    await tmplApi.sendMsg.request({
        code: tmpl.code,
        params: state.sendMsgDialog.params,
        receiverIds: state.sendMsgDialog.receiverIds,
    });
    useI18nOperateSuccessMsg();
    state.sendMsgDialog.visible = false;
};
</script>
<style lang="scss"></style>
