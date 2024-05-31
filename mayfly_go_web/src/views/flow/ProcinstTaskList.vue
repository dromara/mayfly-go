<template>
    <div>
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
                <el-button link @click="showProcinst(data, false)" type="primary">查看</el-button>
                <el-button v-if="data.status == ProcinstTaskStatus.Process.value" link @click="showProcinst(data, true)" type="primary">审核</el-button>
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

const searchItems = [SearchItem.select('status', '任务状态').withEnum(ProcinstTaskStatus), SearchItem.select('bizType', '业务类型').withEnum(FlowBizType)];
const columns = [
    TableColumn.new('procinst.bizType', '业务').typeTag(FlowBizType),
    TableColumn.new('procinst.remark', '备注'),
    TableColumn.new('procinst.creator', '发起人'),
    TableColumn.new('procinst.status', '流程状态').typeTag(ProcinstStatus),
    TableColumn.new('status', '任务状态').typeTag(ProcinstTaskStatus),
    TableColumn.new('procinst.bizKey', '业务key'),
    TableColumn.new('procinst.procdefName', '流程名'),
    TableColumn.new('taskName', '当前节点'),
    TableColumn.new('procinst.createTime', '发起时间').isTime(),
    TableColumn.new('createTime', '开始时间').isTime(),
    TableColumn.new('endTime', '结束时间').isTime(),
    TableColumn.new('duration', '持续时间').setFormatFunc((data: any, prop: string) => {
        const duration = data[prop];
        if (!duration) {
            return '';
        }
        return formatTime(duration);
    }),
    TableColumn.new('action', '操作').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter(),
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
        title: '查看流程',
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
        state.procinstDetail.title = '流程查看';
    } else {
        state.procinstDetail.instTaskId = data.id;
        state.procinstDetail.title = '流程审批';
    }
    state.procinstDetail.visible = true;
};

const valChange = () => {
    state.procinstDetail.visible = false;
    search();
};
</script>
<style lang="scss"></style>
