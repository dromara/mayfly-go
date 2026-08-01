<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="procinstApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="startProcInst()">{{ $t('flow.startProcess') }}</el-button>
            </template>

            <template #action="{ data }">
                <el-button link @click="showProcinst(data)" type="primary">{{ $t('common.detail') }}</el-button>

                <el-button
                    v-if="data.status == ProcinstStatus.Back.value && data.creator == useUserInfo().userInfo.username"
                    link
                    @click="startProcInst(data)"
                    type="primary"
                    >{{ $t('common.edit') }}
                </el-button>

                <el-popconfirm
                    v-if="
                        data.status == ProcinstStatus.Active.value || data.status == ProcinstStatus.Suspended.value || data.status == ProcinstStatus.Back.value
                    "
                    :title="$t('flow.cancelProcessConfirm')"
                    width="160"
                    @confirm="procinstCancel(data)"
                >
                    <template #reference>
                        <el-button link type="warning">{{ $t('common.cancel') }}</el-button>
                    </template>
                </el-popconfirm>
            </template>
        </page-table>

        <ProcinstDetail
            v-model:visible="procinstDetail.visible"
            :title="procinstDetail.title"
            :procinst-id="procinstDetail.procinstId"
            :inst-task-id="procinstDetail.instTaskId"
            @val-change="valChange()"
            @cancel="procinstDetail.procinstId = 0"
        />

        <ProcinstEdit v-model="procinstEdit.procinst" v-model:visible="procinstEdit.visible" :title="procinstEdit.title" @val-change="search" />
    </div>
</template>

<script lang="ts" setup>
import { formatTime } from '@/common/utils/format';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nDetailTitle } from '@/hooks/useI18n';
import { useUserInfo } from '@/store/userInfo';
import { defineAsyncComponent, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import ProcinstDetail from './ProcinstDetail.vue';
import { procinstApi } from './api';
import { FlowBizType, ProcinstBizStatus, ProcinstStatus } from './enums';
import type { Procinst, FlowBizForm } from './types';

const { t } = useI18n();

const ProcinstEdit = defineAsyncComponent(() => import('@/views/flow/ProcInstEdit.vue'));

const searchItems = [
    SearchItem.select('status', 'common.status').withEnum(ProcinstStatus),
    SearchItem.select('bizType', 'flow.bizType').withEnum(FlowBizType),
    SearchItem.input('bizKey', 'flow.bizKey'),
];

const columns = [
    TableColumn.new('bizType', 'flow.bizType').typeTag(FlowBizType),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('creator', 'flow.initiator'),
    TableColumn.new('bizKey', 'flow.bizKey'),
    TableColumn.new('procdefName', 'flow.procdefName'),
    TableColumn.new('status', 'common.status').setAddWidth(8).typeTag(ProcinstStatus),
    TableColumn.new('bizStatus', 'flow.bizStatus').typeTag(ProcinstBizStatus),
    TableColumn.new('createTime', 'flow.startingTime').isTime(),
    TableColumn.new('endTime', 'flow.endTime').isTime(),
    TableColumn.new('duration', 'flow.duration').setFormatFunc((data: Procinst) => {
        const duration = data.duration;
        if (!duration) {
            return '';
        }
        return formatTime(duration);
    }),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter(),
];

const pageTableRef = useTemplateRef<{ search: () => void }>('pageTableRef');
const state = reactive({
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        status: null,
        bizType: '',
        pageNum: 1,
        pageSize: 0,
    },
    procinstDetail: {
        title: '',
        visible: false,
        procinstId: 0,
        instTaskId: 0,
    },
    procinstEdit: {
        title: '',
        visible: false,
        procinst: {},
    },
});

const { selectionData, query, procinstDetail, procinstEdit } = toRefs(state);

const search = async () => {
    pageTableRef.value?.search();
};

const procinstCancel = async (data: Procinst) => {
    await procinstApi.cancel.request({ id: data.id });
    Msg.operateSuccess();
    search();
};

const showProcinst = (data: Procinst) => {
    state.procinstDetail.procinstId = data.id;
    state.procinstDetail.title = useI18nDetailTitle('flow.proc');
    state.procinstDetail.visible = true;
};

const startProcInst = (procinst: Procinst | null = null) => {
    state.procinstEdit.title = t('flow.startProcess');
    if (procinst) {
        const data = { ...procinst, bizForm: JSON.parse(procinst.bizForm || '{}') as FlowBizForm };
        state.procinstEdit.procinst = data as unknown as Procinst;
    } else {
        state.procinstEdit.procinst = {
            bizType: FlowBizType.DbSqlExec.value,
            bizForm: {},
        } as unknown as Procinst;
    }

    state.procinstEdit.visible = true;
};

const valChange = () => {
    state.procinstDetail.visible = false;
    search();
};
</script>
<style lang="scss"></style>
