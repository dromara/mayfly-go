<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="procinstApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #empty>
                <div class="table-empty">
                    <p>{{ $t('flow.procinstEmpty') }}</p>
                    <p class="empty-tip">{{ $t('flow.procinstEmptyTip') }}</p>
                </div>
            </template>

            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="startProcInst()">{{ $t('flow.startProcess') }}</el-button>
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <ProcinstDetail
            v-model:visible="procinstDetail.visible"
            :title="procinstDetail.title"
            :procinst-id="procinstDetail.procinstId"
            :inst-task-id="procinstDetail.instTaskId"
            @val-change="valChange()"
            @cancel="procinstDetail.procinstId = 0"
        />

        <ProcinstEdit v-model="procinstEdit.procinst" v-model:visible="procinstEdit.visible" :title="procinstEdit.title" @val-change="search()" />

        <!-- 改期：只改已批准未执行工单的计划执行时间（列表与工单详情共用同一组件） -->
        <ProcinstRescheduleDialog v-model:visible="reschedule.visible" :procinst-id="reschedule.id" :current="reschedule.current" @submitted="search()" />
    </div>
</template>

<script lang="ts" setup>
import { formatTime } from '@/common/utils/format';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nDetailTitle } from '@/hooks/useI18n';
import { useUserInfo } from '@/store/userInfo';
import { defineAsyncComponent, onActivated, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import ProcinstDetail from './ProcinstDetail.vue';
import ProcinstRescheduleDialog from './components/ProcinstRescheduleDialog.vue';
import { procinstApi } from './api';
import { FlowBizType, ProcinstBizStatus, ProcinstStatus } from './enums';
import type { Procinst, ProcInstStartForm } from './types';

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
    // 主识别列设为弹性列：内容长度方差最大，宽屏富余由它吸收；其余定长列（时间/状态/时长）贴内容、不再被均摊撑宽
    TableColumn.new('procdefName', 'flow.procdefName').setFlexible(),
    TableColumn.new('status', 'common.status').typeTag(ProcinstStatus),
    TableColumn.new('bizStatus', 'flow.bizStatus').typeTag(ProcinstBizStatus),
    TableColumn.new('plannedExecTime', 'flow.plannedExecTime').isTime(),
    TableColumn.new('createTime', 'flow.startingTime').isTime(),
    TableColumn.new('endTime', 'flow.endTime').isTime(),
    TableColumn.new('duration', 'flow.duration').setFormatFunc((data: Procinst) => {
        const duration = data.duration;
        if (!duration) {
            return '';
        }
        return formatTime(duration);
    }),
    // 退回态可「编辑 + 取消」，定时待执行态可「改期 + 取消执行」，两组各自互斥（同组按最宽一格计宽）
    TableColumn.new('action', 'common.operation')
        .setActions<Procinst>([
            { key: 'detail', label: 'common.detail', type: 'primary', onClick: (row) => showProcinst(row) },
            {
                key: 'edit',
                label: 'common.edit',
                type: 'primary',
                group: 'second',
                show: (row) => row.status == ProcinstStatus.Back.value && row.creator == useUserInfo().userInfo.username,
                onClick: (row) => startProcInst(row),
            },
            {
                key: 'cancel',
                label: 'common.cancel',
                type: 'warning',
                group: 'third',
                confirm: { title: 'flow.cancelProcessConfirm', pop: true },
                show: (row) =>
                    row.status == ProcinstStatus.Active.value || row.status == ProcinstStatus.Suspended.value || row.status == ProcinstStatus.Back.value,
                onClick: (row) => procinstCancel(row),
            },
            {
                key: 'reschedule',
                label: 'flow.reschedule',
                type: 'primary',
                group: 'second',
                show: (row) => row.status == ProcinstStatus.Completed.value && row.bizStatus == ProcinstBizStatus.Wait.value,
                onClick: (row) => openReschedule(row),
            },
            {
                key: 'cancelExec',
                label: 'flow.cancelExec',
                type: 'warning',
                group: 'third',
                confirm: { title: 'flow.cancelExecConfirm', pop: true },
                show: (row) => row.status == ProcinstStatus.Completed.value && row.bizStatus == ProcinstBizStatus.Wait.value,
                onClick: (row) => procinstCancelExec(row),
            },
        ])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
];

const dataTableRef = useTemplateRef<{ search: () => void }>('dataTableRef');

/** 启动表单空值：作为抽屉打开前的占位，startProcInst 会按新建/重跑场景覆盖 */
const emptyProcInstForm: ProcInstStartForm = { bizType: '', procdefId: 0, status: null, remark: '', bizKey: '', bizForm: {} };

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
        // 传给 ProcinstEdit 的是启动表单（bizForm 为对象），非实例记录本身；并于 startProcInst 按场景覆盖
        procinst: { ...emptyProcInstForm },
    },
});

const { selectionData, query, procinstDetail, procinstEdit } = toRefs(state);

const search = async () => {
    dataTableRef.value?.search();
};

const firstActivation = ref(true);
// 提单与审批常在别的页签里完成，回到本页必须看到最新状态；
// keep-alive 首次挂载时 DataTable 已经查过一次，跳过首触发避免重复请求
onActivated(() => {
    if (firstActivation.value) {
        firstActivation.value = false;
        return;
    }
    search();
});

const procinstCancel = async (data: Procinst) => {
    await procinstApi.cancel.request({ id: data.id });
    Msg.operateSuccess();
    search();
};

// 定时执行改期/取消：仅对「已批准、待执行」工单开放（按钮已按状态隐藏）；改期表单收敛到共享组件
const reschedule = reactive({ visible: false, id: 0, current: '' });

const openReschedule = (data: Procinst) => {
    reschedule.id = data.id;
    reschedule.current = data.plannedExecTime ?? '';
    reschedule.visible = true;
};

const procinstCancelExec = async (data: Procinst) => {
    await procinstApi.cancelExec.request({ id: data.id });
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
        // 表单态：bizForm 由实例存储的 JSON 串解析为对象，其余字段直接沿用
        state.procinstEdit.procinst = { ...procinst, bizForm: JSON.parse(procinst.bizForm || '{}') };
    } else {
        state.procinstEdit.procinst = {
            bizType: FlowBizType.DbSqlExec.value,
            procdefId: 0,
            status: null,
            remark: '',
            bizKey: '',
            bizForm: {},
        };
    }

    state.procinstEdit.visible = true;
};

const valChange = () => {
    state.procinstDetail.visible = false;
    search();
};
</script>
<style lang="scss"></style>
