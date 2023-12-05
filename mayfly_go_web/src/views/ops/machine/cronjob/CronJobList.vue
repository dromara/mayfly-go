<template>
    <div>
        <page-table
            ref="pageTableRef"
            :query="queryConfig"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :data="data.list"
            :columns="columns"
            :total="data.total"
            v-model:page-size="params.pageSize"
            v-model:page-num="params.pageNum"
            @pageChange="search()"
        >
            <template #queryRight>
                <el-button v-auth="perms.saveCronJob" type="primary" icon="plus" @click="openFormDialog(false)" plain>添加 </el-button>
                <el-button v-auth="perms.delCronJob" :disabled="selectionData.length < 1" @click="deleteCronJob()" type="danger" icon="delete">删除</el-button>
            </template>

            <template #running="{ data }">
                <el-tag v-if="data.running" type="success" effect="plain">运行中</el-tag>
                <el-tag v-else type="danger" effect="plain">未运行</el-tag>
            </template>

            <template #action="{ data }">
                <el-button :disabled="data.status == CronJobStatusEnum.Disable.value" v-auth="perms.saveCronJob" type="primary" @click="runCronJob(data)" link
                    >执行</el-button
                >
                <el-button v-auth="perms.saveCronJob" type="primary" @click="openFormDialog(data)" link>编辑</el-button>
                <el-button type="primary" @click="showExec(data)" link>执行记录</el-button>
            </template>
        </page-table>

        <CronJobEdit v-model:visible="cronJobEdit.visible" v-model:data="cronJobEdit.data" :title="cronJobEdit.title" @submit-success="search" />
        <CronJobExecList v-model:visible="execDialog.visible" :data="execDialog.data" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, defineAsyncComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { cronJobApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn, TableQuery } from '@/components/pagetable';
import { CronJobStatusEnum, CronJobSaveExecResTypeEnum } from '../enums';

const CronJobEdit = defineAsyncComponent(() => import('./CronJobEdit.vue'));
const CronJobExecList = defineAsyncComponent(() => import('./CronJobExecList.vue'));

const pageTableRef: any = ref(null);

const perms = {
    saveCronJob: 'machine:cronjob:save',
    delCronJob: 'machine:cronjob:del',
};

const queryConfig = [TableQuery.text('name', '名称'), TableQuery.select('status', '状态').setOptions(Object.values(CronJobStatusEnum))];

const columns = ref([
    TableColumn.new('key', 'key'),
    TableColumn.new('name', '名称'),
    TableColumn.new('cron', 'cron'),
    TableColumn.new('script', '脚本').canBeautify(),
    TableColumn.new('status', '状态').typeTag(CronJobStatusEnum),
    TableColumn.new('running', '运行状态').isSlot(),
    TableColumn.new('saveExecResType', '记录类型').typeTag(CronJobSaveExecResTypeEnum),
    TableColumn.new('remark', '备注'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(180).fixedRight().alignCenter(),
]);

const state = reactive({
    params: {
        pageNum: 1,
        pageSize: 0,
        ip: null,
        name: null,
    },
    // 列表数据
    data: {
        list: [],
        total: 10,
    },
    selectionData: [],
    execDialog: {
        visible: false,
        total: 0,
        data: [] as any,
    },
    cronJobEdit: {
        visible: false,
        data: null as any,
        title: '新增机器',
    },
});

const { selectionData, params, data, execDialog, cronJobEdit } = toRefs(state);

onMounted(async () => {
    search();
});

const openFormDialog = async (data: any) => {
    let dialogTitle;
    if (data) {
        state.cronJobEdit.data = data;
        dialogTitle = '编辑计划任务';
    } else {
        state.cronJobEdit.data = null;
        dialogTitle = '添加计划任务';
    }

    state.cronJobEdit.title = dialogTitle;
    state.cronJobEdit.visible = true;
};

const runCronJob = async (data: any) => {
    await cronJobApi.run.request({ key: data.key });
    ElMessage.success('执行成功');
};

const deleteCronJob = async () => {
    try {
        await ElMessageBox.confirm(`确定删除【${state.selectionData.map((x: any) => x.name).join(', ')}】计划任务信息? 该操作将同时删除执行记录`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await cronJobApi.delete.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('操作成功');
        search();
    } catch (err) {
        //
    }
};

/**
 * 显示计划任务执行记录
 */
const showExec = async (data: any) => {
    state.execDialog.data = data;
    state.execDialog.visible = true;
};

const search = async () => {
    try {
        pageTableRef.value.loading(true);
        const res = await cronJobApi.list.request(state.params);
        state.data = res;
    } finally {
        pageTableRef.value.loading(false);
    }
};
</script>

<style>
.el-dialog__body {
    padding: 2px 2px;
}
</style>
