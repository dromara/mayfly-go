<template>
    <div>
        <page-table
            ref="pageTableRef"
            :page-api="procinstApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="startProcInst()">发起流程</el-button>
            </template>

            <template #action="{ data }">
                <el-button link @click="showProcinst(data)" type="primary">查看</el-button>

                <el-popconfirm
                    v-if="data.status == ProcinstStatus.Active.value || data.status == ProcinstStatus.Suspended.value"
                    title="确认取消该流程?"
                    width="160"
                    @confirm="procinstCancel(data)"
                >
                    <template #reference>
                        <el-button link type="warning">取消</el-button>
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
import { ElMessage } from 'element-plus';
import { formatTime } from '@/common/utils/format';
import ProcInstEdit from './ProcInstEdit.vue';

const searchItems = [
    SearchItem.select('status', '流程状态').withEnum(ProcinstStatus),
    SearchItem.select('bizType', '业务类型').withEnum(FlowBizType),
    SearchItem.input('bizKey', '业务key'),
];

const columns = [
    TableColumn.new('bizType', '业务').typeTag(FlowBizType),
    TableColumn.new('remark', '备注'),
    TableColumn.new('creator', '发起人'),
    TableColumn.new('bizKey', '业务key'),
    TableColumn.new('procdefName', '流程名'),
    TableColumn.new('status', '流程状态').typeTag(ProcinstStatus),
    TableColumn.new('bizStatus', '业务状态').typeTag(ProcinstBizStatus),
    TableColumn.new('createTime', '发起时间').isTime(),
    TableColumn.new('endTime', '结束时间').isTime(),
    TableColumn.new('duration', '持续时间').setFormatFunc((data: any, prop: string) => {
        const duration = data[prop];
        if (!duration) {
            return '';
        }
        return formatTime(duration);
    }),
    // TableColumn.new('bizHandleRes', '业务处理结果'),
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
        status: null,
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
    procinstEdit: {
        title: '发起流程',
        visible: false,
    },
});

const { selectionData, query, procinstDetail, procinstEdit } = toRefs(state);

const search = async () => {
    pageTableRef.value.search();
};

const procinstCancel = async (data: any) => {
    await procinstApi.cancel.request({ id: data.id });
    ElMessage.success('操作成功');
    search();
};

const showProcinst = (data: any) => {
    state.procinstDetail.procinstId = data.id;
    state.procinstDetail.title = '流程查看';
    state.procinstDetail.visible = true;
};

const startProcInst = () => {
    state.procinstEdit.visible = true;
};

const valChange = () => {
    state.procinstDetail.visible = false;
    search();
};
</script>
<style lang="scss"></style>
