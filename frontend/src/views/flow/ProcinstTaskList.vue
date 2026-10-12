<template>
    <div class="h-full card p-2!">
        <el-tabs v-model="activeTabName" @tab-change="onTaskTabChange" class="h-full">
            <el-tab-pane :label="$t('flow.todoTask')" :name="todoTabName" class="h-full">
                <div class="h-full">
                    <data-table
                        ref="todoDataTableRef"
                        :page-api="procinstTaskApi.tasks"
                        :search-items="todoSearchItems"
                        v-model:query-form="todoQuery"
                        v-model:selection-data="selectionData"
                        :columns="todoColumns"
                    >
                        <template #tableHeader>
                            <!-- <el-button v-auth="perms.addAccount" type="primary" icon="plus" @click="editFlowDef(false)">添加</el-button> -->
                        </template>

                        <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
                    </data-table>
                </div>
            </el-tab-pane>

            <el-tab-pane :label="$t('flow.doneTask')" :name="doneTabName" class="h-full">
                <div class="h-full">
                    <data-table
                        ref="doneDataTableRef"
                        :page-api="procinstTaskApi.tasks"
                        :search-items="searchItems"
                        v-model:query-form="query"
                        v-model:selection-data="selectionData"
                        :columns="columns"
                    >
                        <template #tableHeader>
                            <!-- <el-button v-auth="perms.addAccount" type="primary" icon="plus" @click="editFlowDef(false)">添加</el-button> -->
                        </template>

                        <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
                    </data-table>
                </div>
            </el-tab-pane>
        </el-tabs>

        <ProcinstDetail
            v-model:visible="procinstDetail.visible"
            :title="procinstDetail.title"
            :procinst-id="procinstDetail.procinstId"
            :inst-task-id="procinstDetail.instTaskId"
            @val-change="onValChange()"
            @cancel="procinstDetail.procinstId = 0"
        />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, Ref, useTemplateRef } from 'vue';
import { procinstTaskApi } from './api';
import DataTable from '@/components/data-table/DataTable.vue';
import { TableColumn } from '@/components/data-table';
import { SearchItem } from '@/components/data-table/SearchForm';
import ProcinstDetail from './ProcinstDetail.vue';
import { FlowBizType, ProcinstStatus, ProcinstTaskStatus } from './enums';
import type { ProcinstTask } from './types';
import { formatTime } from '@/common/utils/format';
import { useI18nDetailTitle } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const todoSearchItems = [SearchItem.input('bizKey', 'flow.bizKey'), SearchItem.select('bizType', 'flow.bizType').withEnum(FlowBizType)];

const todoColumns = [
    TableColumn.new('procinst.bizType', 'flow.bizType').typeTag(FlowBizType),
    TableColumn.new('procinst.remark', 'common.remark'),
    TableColumn.new('procinst.creator', 'flow.initiator'),
    TableColumn.new('procinst.status', 'flow.procinstStatus').typeTag(ProcinstStatus),
    TableColumn.new('status', 'flow.taskStatus').typeTag(ProcinstTaskStatus),
    TableColumn.new('procinst.bizKey', 'flow.bizKey'),
    TableColumn.new('procinst.procdefName', 'flow.procdefName'),
    TableColumn.new('procinst.createTime', 'flow.startingTime').isTime(),
    TableColumn.new('nodeName', 'flow.taskName'),
    TableColumn.new('createTime', 'flow.taskBeginTime').isTime(),
    TableColumn.new('action', 'common.operation')
        .setActions<ProcinstTask>([
            { key: 'detail', label: 'common.detail', type: 'primary', onClick: (row) => onShowProcinst(row, false) },
            {
                key: 'audit',
                label: 'flow.audit',
                type: 'primary',
                show: (row) => row.status == ProcinstTaskStatus.Process.value,
                onClick: (row) => onShowProcinst(row, true),
            },
        ])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
];

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
    TableColumn.new('procinst.createTime', 'flow.startingTime').isTime(),
    TableColumn.new('nodeName', 'flow.taskName'),
    TableColumn.new('createTime', 'flow.taskBeginTime').isTime(),
    TableColumn.new('endTime', 'flow.endTime').isTime(),
    TableColumn.new('duration', 'flow.duration').setFormatFunc((data: any, prop: string) => {
        const duration = data[prop];
        if (!duration) {
            return '';
        }
        return formatTime(duration);
    }),
    TableColumn.new('remark', 'flow.approvalRemark'),
    TableColumn.new('action', 'common.operation')
        .setActions<ProcinstTask>([{ key: 'detail', label: 'common.detail', type: 'primary', onClick: (row) => onShowProcinst(row, false) }])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
];

const todoTabName = 'todo';
const doneTabName = 'done';

const activeTabName = ref(todoTabName);

const todoDataTableRef: Ref<any> = useTemplateRef('todoDataTableRef');
const doneDataTableRef: Ref<any> = useTemplateRef('doneDataTableRef');

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
    todoQuery: {
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

const { selectionData, query, todoQuery, procinstDetail } = toRefs(state);

const todoSearch = async () => {
    todoDataTableRef.value?.search();
};

const onTaskTabChange = (activeName: string) => {
    if (activeName === todoTabName) {
        todoDataTableRef.value?.search();
    } else {
        doneDataTableRef.value?.search();
    }
};

const onShowProcinst = (data: any, audit: boolean) => {
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

const onValChange = () => {
    state.procinstDetail.visible = false;
    todoSearch();
};
</script>
<style lang="scss"></style>
