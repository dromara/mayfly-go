<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="procinstApi.tasks"
            :search-items="searchItems"
            v-model:query-form="query"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <!-- <el-button v-auth="perms.addAccount" type="primary" icon="plus" @click="editFlowDef(false)">添加</el-button> -->
            </template>

            <template #action="{ data }">
                <el-button link @click="showProcinst(data, false)" type="primary">{{ $t('common.detail') }}</el-button>
                <el-button v-if="data.status == ProcinstTaskStatus.Process.value" link @click="showProcinst(data, true)" type="primary">
                    {{ $t('flow.audit') }}
                </el-button>
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
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, Ref } from 'vue';
import { procinstApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import ProcinstDetail from './ProcinstDetail.vue';
import { FlowBizType, ProcinstStatus, ProcinstTaskStatus } from './enums';
import { formatTime } from '@/common/utils/format';
import { useI18nDetailTitle } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const searchItems = [
    SearchItem.select('status', 'common.status').withEnum(ProcinstTaskStatus),
    SearchItem.input('bizKey', 'flow.bizKey'),
    SearchItem.select('bizType', 'flow.bizType').withEnum(FlowBizType),
];
const columns = [
    TableColumn.new('procinst.bizType', 'flow.bizType').typeTag(FlowBizType),
    TableColumn.new('procinst.remark', 'common.remark'),
    TableColumn.new('procinst.creator', 'flow.initiator'),
    TableColumn.new('procinst.status', 'flow.procinstStatus').typeTag(ProcinstStatus),
    TableColumn.new('status', 'flow.taskStatus').typeTag(ProcinstTaskStatus),
    TableColumn.new('procinst.bizKey', 'flow.bizKey'),
    TableColumn.new('procinst.procdefName', 'flow.procdefName'),
    TableColumn.new('taskName', 'flow.taskName'),
    TableColumn.new('procinst.createTime', 'flow.startingTime').isTime(),
    TableColumn.new('createTime', 'flow.taskBeginTime').isTime(),
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
        status: ProcinstTaskStatus.Process.value,
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
});

const { selectionData, query, procinstDetail } = toRefs(state);

const search = async () => {
    pageTableRef.value.search();
};

const showProcinst = (data: any, audit: boolean) => {
    state.procinstDetail.procinstId = data.procinstId;
    if (!audit) {
        state.procinstDetail.instTaskId = 0;
        state.procinstDetail.title = useI18nDetailTitle('flow.proc');
    } else {
        state.procinstDetail.instTaskId = data.id;
        state.procinstDetail.title = t('flow.flowAudit');
    }
    state.procinstDetail.visible = true;
};

const valChange = () => {
    state.procinstDetail.visible = false;
    search();
};
</script>
<style lang="scss"></style>
