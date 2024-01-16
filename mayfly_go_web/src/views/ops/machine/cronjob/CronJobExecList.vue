<template>
    <div>
        <el-dialog
            :title="title"
            v-model="dialogVisible"
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
                :data-handler-fn="parseData"
                :search-items="searchItems"
                v-model:query-form="params"
                :data="state.data.list"
                :columns="columns"
            >
                <template #machineSelect>
                    <el-select v-model="params.machineId" filterable placeholder="选择机器查询" clearable>
                        <el-option v-for="ac in machineMap.values()" :key="ac.id" :value="ac.id" :label="ac.ip">
                            {{ ac.ip }}
                            <el-divider direction="vertical" border-style="dashed" />
                            {{ ac.tagPath }}{{ ac.name }}
                        </el-option>
                    </el-select>
                </template>
            </page-table>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { watch, ref, toRefs, reactive, Ref } from 'vue';
import { cronJobApi, machineApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { CronJobExecStatusEnum } from '../enums';
import { SearchItem } from '@/components/SearchForm';

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

const emit = defineEmits(['update:visible', 'update:data', 'cancel']);

const searchItems = [SearchItem.slot('machineId', '机器', 'machineSelect'), SearchItem.select('status', '状态').withEnum(CronJobExecStatusEnum)];

const columns = ref([
    TableColumn.new('machineIp', '机器IP').setMinWidth(120),
    TableColumn.new('machineName', '机器名称').setMinWidth(100),
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
        pageSize: 10,
        cronJobId: 0,
        status: null,
        machineId: null,
    },
    // 列表数据
    data: {
        list: [],
        total: 10,
    },
    machines: [],
});

const machineMap: Map<number, any> = new Map();

const { dialogVisible, params } = toRefs(state);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!newValue.visible) {
        return;
    }

    const machineIds = await cronJobApi.relateMachineIds.request({
        cronJobId: props.data?.id,
    });
    const res = await machineApi.list.request({
        ids: machineIds?.join(','),
    });

    res.list?.forEach((x: any) => {
        machineMap.set(x.id, x);
    });

    state.params.cronJobId = props.data?.id;
    search();
});

const search = async () => {
    pageTableRef.value.search();
};

const parseData = async (res: any) => {
    const dataList = res.list;
    // 填充机器信息
    for (let x of dataList) {
        const machineId = x.machineId;
        let machine = machineMap.get(machineId);
        // 如果未找到，则可能被移除，则调接口查询机器信息
        if (!machine) {
            const machineRes = await machineApi.list.request({ ids: machineId });
            if (!machineRes.list) {
                machine = {
                    id: machineId,
                    ip: machineId,
                    name: '该机器已被删除',
                };
            } else {
                machine = machineRes.list[0];
            }
            machineMap.set(machineId, machine);
        }

        x.machineIp = machine?.ip;
        x.machineName = machine?.name;
    }
    return res;
};

const cancel = () => {
    emit('update:visible', false);
    setTimeout(() => {
        initData();
    }, 500);
};

const initData = () => {
    state.data.list = [];
    state.data.total = 0;
    state.params.pageNum = 1;
    state.params.machineId = null;
    state.params.status = null;
};
</script>

<style>
.el-dialog__body {
    padding: 2px 2px;
}
</style>
