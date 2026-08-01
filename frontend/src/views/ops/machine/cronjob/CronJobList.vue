<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="cronJobApi.list"
            :query="searchItems"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.saveCronJob" type="primary" icon="plus" @click="openFormDialog(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delCronJob" :disabled="selectionData.length < 1" @click="deleteCronJob()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #running="{ data }">
                <el-tag v-if="data.running" type="success" effect="plain">{{ $t('machine.cronjobRunning') }}</el-tag>
                <el-tag v-else type="danger" effect="plain">{{ $t('machine.cronjobNoRun') }}</el-tag>
            </template>

            <template #codePaths="{ data }">
                <TagCodePath :path="data.tags" />
            </template>

            <template #action="{ data }">
                <el-button :disabled="data.status == CronJobStatusEnum.Disable.value" v-auth="perms.saveCronJob" type="primary" @click="runCronJob(data)" link
                    >{{ $t('machine.cronjobRun') }}
                </el-button>
                <el-button v-auth="perms.saveCronJob" type="primary" @click="openFormDialog(data)" link>{{ $t('common.edit') }}</el-button>
                <el-button type="primary" @click="showExec(data)" link>{{ $t('machine.cronjobExecRecord') }}</el-button>
            </template>
        </page-table>

        <CronJobEdit v-model:visible="cronJobEdit.visible" v-model:data="cronJobEdit.data" :title="cronJobEdit.title" @submit-success="search" />
        <CronJobExecList v-model:visible="execDialog.visible" :data="execDialog.data" />
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import TagCodePath from '../../component/TagCodePath.vue';
import { cronJobApi } from '../api';
import { CronJobSaveExecResTypeEnum, CronJobStatusEnum } from '../enums';
import type { MachineCronJob } from '../types';

const CronJobEdit = defineAsyncComponent(() => import('./CronJobEdit.vue'));
const CronJobExecList = defineAsyncComponent(() => import('./CronJobExecList.vue'));

const perms = {
    saveCronJob: 'machine:cronjob:save',
    delCronJob: 'machine:cronjob:del',
};

const searchItems = [SearchItem.input('name', 'common.name'), SearchItem.select('status', 'common.status').withEnum(CronJobStatusEnum)];

const columns = ref([
    TableColumn.new('key', 'key'),
    TableColumn.new('name', 'common.name'),
    TableColumn.new('cron', 'cron'),
    TableColumn.new('script', 'machine.script').canBeautify(),
    TableColumn.new('status', 'common.status').typeTag(CronJobStatusEnum),
    TableColumn.new('running', 'machine.cronjobRunState').isSlot(),
    TableColumn.new('saveExecResType', 'machine.execResRecordType').typeTag(CronJobSaveExecResTypeEnum),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('codePaths', 'machine.relateMachine').isSlot().setMinWidth('250px'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(180).fixedRight().noShowOverflowTooltip().alignCenter(),
]);

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const state = reactive({
    params: {
        pageNum: 1,
        pageSize: 0,
        ip: null,
        name: null,
    },
    selectionData: [],
    execDialog: {
        visible: false,
        total: 0,
        data: null as MachineCronJob | null,
    },
    cronJobEdit: {
        visible: false,
        data: null as MachineCronJob | null,
        title: '',
    },
});

const { selectionData, params, execDialog, cronJobEdit } = toRefs(state);

onMounted(async () => {});

const openFormDialog = async (data: MachineCronJob | false) => {
    let dialogTitle;
    if (data) {
        state.cronJobEdit.data = data;
        dialogTitle = useI18nEditTitle('machine.cronjob');
    } else {
        state.cronJobEdit.data = null;
        dialogTitle = useI18nCreateTitle('machine.cronjob');
    }

    state.cronJobEdit.title = dialogTitle;
    state.cronJobEdit.visible = true;
};

const runCronJob = async (data: MachineCronJob) => {
    await cronJobApi.run.request({ key: data.key });
    Msg.success('machine.runSuccess');
};

const deleteCronJob = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: MachineCronJob) => x.name).join('、'));
        await cronJobApi.delete.request({ id: state.selectionData.map((x: MachineCronJob) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};

/**
 * 显示计划任务执行记录
 */
const showExec = async (data: MachineCronJob) => {
    state.execDialog.data = data;
    state.execDialog.visible = true;
};

const search = async () => {
    pageTableRef.value?.search();
};
</script>

<style>
.el-dialog__body {
    padding: 2px 2px;
}
</style>
