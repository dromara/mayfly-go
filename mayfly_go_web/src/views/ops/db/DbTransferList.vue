<template>
    <div class="db-list">
        <page-table
            ref="pageTableRef"
            :page-api="dbApi.dbTransferTasks"
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

            <template #taskName="{ data }">
                <span :style="`${data.taskName ? '' : 'color:red'}`">
                    {{ data.taskName || '请设置' }}
                </span>
            </template>
            <template #srcDb="{ data }">
                <el-tooltip :content="`${data.srcTagPath} > ${data.srcInstName} > ${data.srcDbName}`">
                    <span>
                        <SvgIcon :name="getDbDialect(data.srcDbType).getInfo().icon" :size="18" />
                        {{ data.srcDbName }}
                    </span>
                </el-tooltip>
            </template>
            <template #targetDb="{ data }">
                <el-tooltip :content="`${data.targetTagPath} > ${data.targetInstName} > ${data.targetDbName}`">
                    <span>
                        <SvgIcon :name="getDbDialect(data.targetDbType).getInfo().icon" :size="18" />
                        {{ data.targetDbName }}
                    </span>
                </el-tooltip>
            </template>

            <template #action="{ data }">
                <!-- 删除、启停用、编辑 -->
                <el-button v-if="actionBtns[perms.save]" @click="edit(data)" type="primary" link>编辑</el-button>
                <el-button v-if="actionBtns[perms.log]" type="primary" link @click="log(data)">日志</el-button>
                <el-button v-if="data.runningState === 1" @click="stop(data.id)" type="danger" link>停止</el-button>
                <el-button v-if="actionBtns[perms.run] && data.runningState !== 1" type="primary" link @click="reRun(data)">运行</el-button>
            </template>
        </page-table>

        <db-transfer-edit @val-change="search" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />

        <TerminalLog v-model:log-id="logsDialog.logId" v-model:visible="logsDialog.visible" :title="logsDialog.title" />
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
import { getDbDialect } from '@/views/ops/db/dialect';
import { DbTransferRunningStateEnum } from './enums';
import TerminalLog from '@/components/terminal/TerminalLog.vue';

const DbTransferEdit = defineAsyncComponent(() => import('./DbTransferEdit.vue'));

const perms = {
    save: 'db:transfer:save',
    del: 'db:transfer:del',
    status: 'db:transfer:status',
    log: 'db:transfer:log',
    run: 'db:transfer:run',
};

const searchItems = [SearchItem.input('name', '名称')];

const columns = ref([
    TableColumn.new('taskName', '任务名').setMinWidth(150).isSlot(),
    TableColumn.new('srcDb', '源库').setMinWidth(150).isSlot(),
    TableColumn.new('targetDb', '目标库').setMinWidth(150).isSlot(),
    TableColumn.new('runningState', '执行状态').typeTag(DbTransferRunningStateEnum),
    TableColumn.new('creator', '创建人'),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('modifier', '修改人'),
    TableColumn.new('updateTime', '修改时间').isTime(),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del, perms.status, perms.log, perms.run]);
const actionWidth = ((actionBtns[perms.save] ? 1 : 0) + (actionBtns[perms.log] ? 1 : 0) + (actionBtns[perms.run] ? 1 : 0)) * 55;
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
        title: '新增数据数据迁移任务',
    },
    logsDialog: {
        logId: 0,
        title: '数据库迁移日志',
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
        state.editDialog.title = '新增数据库迁移任务';
    } else {
        state.editDialog.data = data;
        state.editDialog.title = '修改数据库迁移任务';
    }
    state.editDialog.visible = true;
};

const stop = async (id: any) => {
    await ElMessageBox.confirm(`确定停止?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });
    await dbApi.stopDbTransferTask.request({ taskId: id });
    ElMessage.success(`停止成功`);
    search();
};

const log = (data: any) => {
    state.logsDialog.logId = data.logId;
    state.logsDialog.visible = true;
    state.logsDialog.title = '数据库迁移日志';
    state.logsDialog.running = data.state === 1;
};

const reRun = async (data: any) => {
    await ElMessageBox.confirm(`确定运行?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });
    try {
        let res = await dbApi.runDbTransferTask.request({ taskId: data.id });
        console.log(res);
        ElMessage.success('运行成功');
        // 拿到日志id之后，弹出日志弹窗
        log({ logId: res, state: 1 });
    } catch (e) {
        //
    }
    // 延迟2秒执行，后端异步执行
    setTimeout(() => {
        search();
    }, 2000);
};

const del = async () => {
    try {
        await ElMessageBox.confirm(`确定删除任务?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteDbTransferTask.request({ taskId: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
