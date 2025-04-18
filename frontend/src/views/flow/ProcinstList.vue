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

                <el-popconfirm
                    v-if="data.status == ProcinstStatus.Active.value || data.status == ProcinstStatus.Suspended.value"
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

        <ProcInstEdit v-model:visible="procinstEdit.visible" :title="procinstEdit.title" @val-change="search" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, Ref } from 'vue';
import { procinstApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import ProcinstDetail from './ProcinstDetail.vue';
import { FlowBizType, ProcinstBizStatus, ProcinstStatus } from './enums';
import { formatTime } from '@/common/utils/format';
import ProcInstEdit from './ProcInstEdit.vue';
import { useI18nDetailTitle, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

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
    TableColumn.new('duration', 'flow.duration').setFormatFunc((data: any, prop: string) => {
        const duration = data[prop];
        if (!duration) {
            return '';
        }
        return formatTime(duration);
    }),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter(),
];

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
    },
});

const { selectionData, query, procinstDetail, procinstEdit } = toRefs(state);

const search = async () => {
    pageTableRef.value.search();
};

const procinstCancel = async (data: any) => {
    await procinstApi.cancel.request({ id: data.id });
    useI18nOperateSuccessMsg();
    search();
};

const showProcinst = (data: any) => {
    state.procinstDetail.procinstId = data.id;
    state.procinstDetail.title = useI18nDetailTitle('flow.proc');
    state.procinstDetail.visible = true;
};

const startProcInst = () => {
    state.procinstEdit.title = t('flow.startProcess');
    state.procinstEdit.visible = true;
};

const valChange = () => {
    state.procinstDetail.visible = false;
    search();
};
</script>
<style lang="scss"></style>
