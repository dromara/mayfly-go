<template>
    <div class="db-list">
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
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="edit(false)">添加</el-button>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" @click="del()" type="danger" icon="delete">删除</el-button>
            </template>
            <template #status="{ data }">
                <span v-if="actionBtns[perms.status]">
                    <el-switch
                        v-model="data.status"
                        @click="updStatus(data.id, data.status)"
                        inline-prompt
                        active-text="启用"
                        inactive-text="禁用"
                        :active-value="1"
                        :inactive-value="-1"
                    />
                </span>
                <span v-else>
                    <el-tag v-if="data.status == 1" class="ml-2" type="success">启用</el-tag>
                    <el-tag v-else class="ml-2" type="danger">禁用</el-tag>
                </span>
            </template>

            <template #action="{ data }">
                <!-- 删除、启停用、编辑 -->
                <el-button v-if="actionBtns[perms.save]" @click="edit(data)" type="primary" link>编辑</el-button>
                <el-button v-if="data.status === 1 && data.runningState !== 1" @click="run(data.id)" type="success" link>执行</el-button>
                <el-button v-if="data.runningState === 1" @click="stop(data.id)" type="danger" link>停止</el-button>
                <el-button v-if="actionBtns[perms.log]" type="primary" link @click="log(data)">日志</el-button>
            </template>
        </page-table>

        <data-sync-task-edit @val-change="search" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />

        <data-sync-task-log v-model:visible="logsDialog.visible" v-model:taskId="logsDialog.taskId" :running="state.logsDialog.running" />
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dbApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import { DbDataSyncRecentStateEnum, DbDataSyncRunningStateEnum } from './enums';

const DataSyncTaskEdit = defineAsyncComponent(() => import('./SyncTaskEdit.vue'));
const DataSyncTaskLog = defineAsyncComponent(() => import('./SyncTaskLog.vue'));

const perms = {
    save: 'db:sync:save',
    del: 'db:sync:del',
    status: 'db:sync:status',
    log: 'db:sync:log',
};

const searchItems = [SearchItem.input('name', '名称')];

// 任务名、修改人、修改时间、最近一次任务执行状态、状态(停用启用)、操作
const columns = ref([
    TableColumn.new('taskName', '任务名'),
    TableColumn.new('runningState', '运行状态').alignCenter().typeTag(DbDataSyncRunningStateEnum),
    TableColumn.new('recentState', '最近任务状态').alignCenter().typeTag(DbDataSyncRecentStateEnum),
    TableColumn.new('status', '状态').alignCenter().isSlot(),
    TableColumn.new('modifier', '修改人').alignCenter(),
    TableColumn.new('updateTime', '修改时间').alignCenter().isTime(),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del, perms.status, perms.log]);
const actionWidth = ((actionBtns[perms.save] ? 1 : 0) + (actionBtns[perms.log] ? 1 : 0)) * 55 + 55;
const actionColumn = TableColumn.new('action', '操作').isSlot().setMinWidth(actionWidth).fixedRight().alignCenter();
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
        title: '新增数据同步任务',
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
        state.editDialog.title = '新增数据同步任务';
    } else {
        state.editDialog.data = data;
        state.editDialog.title = '修改数据同步任务';
    }
    state.editDialog.visible = true;
};

const run = async (id: any) => {
    await ElMessageBox.confirm(`确定执行?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });
    await dbApi.runDatasyncTask.request({ taskId: id });
    ElMessage.success(`执行成功`);
    setTimeout(search, 1000);
};

const stop = async (id: any) => {
    await ElMessageBox.confirm(`确定停止?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });
    await dbApi.stopDatasyncTask.request({ taskId: id });
    ElMessage.success(`停止成功`);
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
        ElMessage.success(`${status === 1 ? '启用' : '禁用'}成功`);
        search();
    } catch (err) {
        //
    }
};

const del = async () => {
    try {
        await ElMessageBox.confirm(`确定删除数据同步任务【${state.selectionData.map((x: any) => x.taskName).join(', ')}】?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteDatasyncTask.request({ taskId: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
