<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="dbApi.datasyncTasks"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="edit(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" @click="del()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>
            <template #status="{ data }">
                <span v-if="actionBtns[perms.status]">
                    <el-switch
                        v-model="data.status"
                        @click="updStatus(data.id, data.status)"
                        inline-prompt
                        :active-text="$t('common.enable')"
                        :inactive-text="$t('common.disable')"
                        :active-value="1"
                        :inactive-value="-1"
                    />
                </span>
                <span v-else>
                    <el-tag v-if="data.status == 1" class="ml-2" type="success">{{ $t('common.enable') }}</el-tag>
                    <el-tag v-else class="ml-2" type="danger">{{ $t('common.enable') }}</el-tag>
                </span>
            </template>

            <template #action="{ data }">
                <!-- 删除、启停用、编辑 -->
                <el-button v-if="actionBtns[perms.save]" @click="edit(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
                <el-button v-if="data.status === 1 && data.runningState !== 1" @click="run(data.id)" type="success" link>{{ $t('db.run') }}</el-button>
                <el-button v-if="data.runningState === 1" @click="stop(data.id)" type="danger" link>{{ $t('db.stop') }}</el-button>
                <el-button v-if="actionBtns[perms.log]" type="primary" link @click="log(data)">{{ $t('db.log') }}</el-button>
            </template>
        </page-table>

        <data-sync-task-edit @val-change="search" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />

        <data-sync-task-log v-model:visible="logsDialog.visible" v-model:taskId="logsDialog.taskId" :running="state.logsDialog.running" />
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import { dbApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import { DbDataSyncRecentStateEnum, DbDataSyncRunningStateEnum } from './enums';
import { useI18nConfirm, useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle, useI18nOperateSuccessMsg } from '@/hooks/useI18n';

const DataSyncTaskEdit = defineAsyncComponent(() => import('./SyncTaskEdit.vue'));
const DataSyncTaskLog = defineAsyncComponent(() => import('./SyncTaskLog.vue'));

const perms = {
    save: 'db:sync:save',
    del: 'db:sync:del',
    status: 'db:sync:status',
    log: 'db:sync:log',
};

const searchItems = [SearchItem.input('name', 'common.name')];

// 任务名、修改人、修改时间、最近一次任务执行状态、状态(停用启用)、操作
const columns = ref([
    TableColumn.new('taskName', 'db.taskName'),
    TableColumn.new('cron', 'Cron'),
    TableColumn.new('runningState', 'db.runState').typeTag(DbDataSyncRunningStateEnum),
    TableColumn.new('recentState', 'db.recentState').typeTag(DbDataSyncRecentStateEnum),
    TableColumn.new('status', 'common.status').isSlot(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del, perms.status, perms.log]);
const actionWidth = ((actionBtns[perms.save] ? 1 : 0) + (actionBtns[perms.log] ? 1 : 0)) * 55 + 55;
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().setMinWidth(actionWidth).fixedRight().alignCenter();
const pageTableRef: Ref<any> = ref(null);

const state = reactive({
    row: {},
    dbId: 0,
    db: '',
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        name: null,
        pageNum: 1,
        pageSize: 0,
    },
    editDialog: {
        visible: false,
        data: null as any,
        title: '',
    },
    logsDialog: {
        taskId: 0,
        visible: false,
        data: null as any,
        running: false,
    },
});

const { selectionData, query, editDialog, logsDialog } = toRefs(state);

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const search = () => {
    pageTableRef.value.search();
};

const edit = async (data: any) => {
    if (!data) {
        state.editDialog.data = null;
        state.editDialog.title = useI18nCreateTitle('db.dbSync');
    } else {
        state.editDialog.data = data;
        state.editDialog.title = useI18nEditTitle('db.dbSync');
    }
    state.editDialog.visible = true;
};

const run = async (id: any) => {
    await useI18nConfirm('db.runConfirm');
    await dbApi.runDatasyncTask.request({ taskId: id });
    useI18nOperateSuccessMsg();
    setTimeout(search, 1000);
};

const stop = async (id: any) => {
    await useI18nConfirm('db.stopConfirm');
    await dbApi.stopDatasyncTask.request({ taskId: id });
    useI18nOperateSuccessMsg();
    search();
};

const log = async (data: any) => {
    state.logsDialog.taskId = data.id;
    state.logsDialog.visible = true;
    state.logsDialog.running = data.state === 1;
};

const updStatus = async (id: any, status: 1 | -1) => {
    try {
        await dbApi.updateDatasyncTaskStatus.request({ taskId: id, status });
        useI18nOperateSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const del = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.taskName).join('、'));
        await dbApi.deleteDatasyncTask.request({ taskId: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
