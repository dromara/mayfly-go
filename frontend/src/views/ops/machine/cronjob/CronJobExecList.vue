<template>
    <div>
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            @open="search()"
            :close-on-click-modal="false"
            :before-close="cancel"
            :show-close="true"
            :destroy-on-close="true"
            width="65%"
            body-class="h-[65vh]"
        >
            <data-table
                ref="dataTableRef"
                :page-api="cronJobApi.execList"
                :lazy="true"
                :search-items="searchItems"
                v-model:query-form="params"
                :data="state.data.list"
                :columns="columns"
            >
                <template #machineCode="{ data }">
                    <MachineDetail :code="data.machineCode" />
                </template>
            </data-table>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, Ref } from 'vue';
import { cronJobApi } from '../api';
import DataTable from '@/components/data-table/DataTable.vue';
import { TableColumn } from '@/components/data-table';
import { CronJobExecStatusEnum } from '../enums';
import { SearchItem } from '@/components/data-table/SearchForm';
import MachineDetail from '../component/MachineDetail.vue';
import type { MachineCronJob } from '../types';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: Object as () => MachineCronJob | null,
    },
    title: {
        type: String,
    },
});

const searchItems = [SearchItem.input('machineCode', 'machine.machineCode'), SearchItem.select('status', 'common.status').withEnum(CronJobExecStatusEnum)];

const columns = ref([
    TableColumn.new('machineCode', 'machine.machineCode').isSlot(),
    TableColumn.new('status', 'common.status').typeTag(CronJobExecStatusEnum).setMinWidth(70),
    TableColumn.new('res', 'machine.cronjobExecResult').setMinWidth(250).canBeautify(),
    TableColumn.new('execTime', 'machine.cronjobExecTime').isTime().setMinWidth(150),
]);

const dataTableRef: Ref<any> = ref(null);

const state = reactive({
    dialogVisible: false,
    tags: [] as any,
    params: {
        pageNum: 1,
        pageSize: 8,
        cronJobId: 0,
        status: null,
        machineCode: '',
    },
    // 列表数据
    data: {
        list: [],
        total: 10,
    },
    machines: [],
});

const { params } = toRefs(state);

const dialogVisible = defineModel<boolean>('visible');

const search = async () => {
    state.params.cronJobId = props.data?.id || 0;
    dataTableRef.value?.search();
};

const cancel = () => {
    dialogVisible.value = false;
    setTimeout(() => {
        initData();
    }, 500);
};

const initData = () => {
    state.data.list = [];
    state.data.total = 0;
    state.params.pageNum = 1;
    state.params.machineCode = '';
    state.params.status = null;
};
</script>

<style>
.el-dialog__body {
    padding: 2px 2px;
}
</style>
