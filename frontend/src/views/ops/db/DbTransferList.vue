<template>
    <div class="h-full">
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
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="edit(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" @click="del()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #taskName="{ data }">
                <span :style="`${data.taskName ? '' : 'color:red'}`">
                    {{ data.taskName || $t('db.pleaseSetting') }}
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
                    <el-tag v-else class="ml-2" type="danger">{{ $t('common.disable') }}</el-tag>
                </span>
            </template>

            <template #action="{ data }">
                <!-- 删除、启停用、编辑 -->
                <el-button v-if="actionBtns[perms.save]" @click="edit(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
                <el-button v-if="actionBtns[perms.log]" type="warning" link @click="log(data)">{{ $t('db.log') }}</el-button>
                <el-button v-if="data.runningState === 1" @click="stop(data.id)" type="danger" link>{{ $t('db.stop') }}</el-button>
                <el-button v-if="actionBtns[perms.run] && data.runningState !== 1 && data.status === 1" type="success" link @click="reRun(data)">
                    {{ $t('db.run') }}
                </el-button>
                <el-button v-if="actionBtns[perms.files] && data.mode === 2" type="success" link @click="openFiles(data)">{{ $t('db.file') }}</el-button>
            </template>
        </page-table>

        <db-transfer-edit @val-change="search" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
        <db-transfer-file :title="filesDialog.title" v-model:visible="filesDialog.visible" v-model:data="filesDialog.data" />

        <TerminalLog v-model:log-id="logsDialog.logId" v-model:visible="logsDialog.visible" :title="logsDialog.title" />
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import { dbApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import { getDbDialect } from '@/views/ops/db/dialect';
import { DbTransferRunningStateEnum } from './enums';
import TerminalLog from '@/components/terminal/TerminalLog.vue';
import DbTransferFile from './DbTransferFile.vue';
import { useI18nConfirm, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const DbTransferEdit = defineAsyncComponent(() => import('./DbTransferEdit.vue'));

const { t } = useI18n();

const perms = {
    save: 'db:transfer:save',
    del: 'db:transfer:del',
    status: 'db:transfer:status',
    log: 'db:transfer:log',
    run: 'db:transfer:run',
    files: 'db:transfer:files',
};

const searchItems = [SearchItem.input('name', 'common.name')];

const columns = ref([
    TableColumn.new('taskName', 'db.taskName').setMinWidth(150).isSlot(),
    TableColumn.new('srcDb', 'db.srcDb').setMinWidth(150).isSlot(),
    // TableColumn.new('targetDb', '目标库').setMinWidth(150).isSlot(),
    TableColumn.new('runningState', 'db.runState').typeTag(DbTransferRunningStateEnum),
    TableColumn.new('status', 'common.status').isSlot(),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del, perms.status, perms.log, perms.run, perms.files]);
const actionWidth =
    ((actionBtns[perms.save] ? 1 : 0) + (actionBtns[perms.log] ? 1 : 0) + (actionBtns[perms.run] ? 1 : 0) + (actionBtns[perms.files] ? 1 : 0)) * 55;
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
        logId: 0,
        title: '',
        visible: false,
        data: null as any,
        running: false,
    },
    filesDialog: {
        taskId: 0,
        title: '',
        visible: false,
        data: null as any,
    },
});

const { selectionData, query, editDialog, logsDialog, filesDialog } = toRefs(state);

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
        state.editDialog.title = t('db.createDbTransferDialogTitle');
    } else {
        state.editDialog.data = data;
        state.editDialog.title = t('db.editDbTransferDialogTitle');
    }
    state.editDialog.visible = true;
};

const stop = async (id: any) => {
    await useI18nConfirm('db.stopConfirm');
    await dbApi.stopDbTransferTask.request({ taskId: id });
    useI18nOperateSuccessMsg();
    search();
};

const log = (data: any) => {
    state.logsDialog.logId = data.logId;
    state.logsDialog.visible = true;
    state.logsDialog.title = t('db.log');
    state.logsDialog.running = data.state === 1;
};

const reRun = async (data: any) => {
    await useI18nConfirm('db.runConfirm');
    try {
        let res = await dbApi.runDbTransferTask.request({ taskId: data.id });
        console.log(res);
        useI18nOperateSuccessMsg();
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

const openFiles = async (data: any) => {
    state.filesDialog.visible = true;
    state.filesDialog.title = t('db.transferFileManage');
    state.filesDialog.taskId = data.id;
    state.filesDialog.data = data;
};
const updStatus = async (id: any, status: 1 | -1) => {
    try {
        await dbApi.updateDbTransferTaskStatus.request({ taskId: id, status });
        useI18nOperateSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const del = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.taskName).join('、'));
        await dbApi.deleteDbTransferTask.request({ taskId: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
