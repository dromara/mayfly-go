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
        >
            <page-table
                ref="pageTableRef"
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
            </page-table>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, Ref } from 'vue';
import { cronJobApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { CronJobExecStatusEnum } from '../enums';
import { SearchItem } from '@/components/SearchForm';
import MachineDetail from '../component/MachineDetail.vue';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: Object,
    },
    title: {
        type: String,
    },
});

const searchItems = [SearchItem.input('machineCode', '机器编号'), SearchItem.select('status', '状态').withEnum(CronJobExecStatusEnum)];

const columns = ref([
    TableColumn.new('machineCode', '机器编号').isSlot(),
    TableColumn.new('status', '状态').typeTag(CronJobExecStatusEnum).setMinWidth(70),
    TableColumn.new('res', '执行结果').setMinWidth(250).canBeautify(),
    TableColumn.new('execTime', '执行时间').isTime().setMinWidth(150),
]);

const pageTableRef: Ref<any> = ref(null);

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
    state.params.cronJobId = props.data?.id;
    pageTableRef.value.search();
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
