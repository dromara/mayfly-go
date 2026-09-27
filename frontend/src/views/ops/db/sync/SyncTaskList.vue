<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="dbSyncApi.datasyncTasks"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
            :data-handler-fn="handleData"
        >
            <template #tableHeader>
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="edit(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" @click="del()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>
            <template #taskName="{ data }">
                <!-- 双向同步只在正向任务上挂标记，单开一列几乎全是空值；反向任务缺失的异常态用颜色+tooltip 说明。
                标记放在名称前：单元格是截断展示，尾随会被 ellipsis 直接切掉 -->
                <el-tooltip v-if="data.biDirEnabled" :content="biDirTip(data)">
                    <el-tag class="mr-1" size="small" :type="data.reverseTaskId ? 'warning' : 'danger'">{{ $t('db.biDirEnabled') }}</el-tag>
                </el-tooltip>
                <span>{{ data.taskName }}</span>
            </template>
            <template #srcDb="{ data }">
                <el-tooltip :content="`${data.srcTagPath} > ${data.srcDbName}`">
                    <span class="block truncate">
                        <SvgIcon v-if="data.srcDbType" :name="getDbDialect(data.srcDbType).getInfo().icon" :size="18" />
                        {{ data.srcDbName }}
                    </span>
                </el-tooltip>
            </template>
            <template #targetTable="{ data }">
                <el-tooltip :content="`${data.targetTagPath} > ${data.targetDbName}.${data.targetTableName}`">
                    <span class="block truncate">
                        <SvgIcon v-if="data.targetDbType" :name="getDbDialect(data.targetDbType).getInfo().icon" :size="18" />
                        {{ `${data.targetDbName}.${data.targetTableName}` }}
                    </span>
                </el-tooltip>
            </template>
            <template #updFieldVal="{ data }">
                <span>{{ syncCursorText(data) }}</span>
            </template>

            <template #status="{ data }">
                <span v-if="actionBtns[perms.status]">
                    <el-switch
                        :model-value="data.status === 1 ? 1 : -1"
                        @change="(val: 1 | -1) => updStatus(data.id, val)"
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
                <el-button v-if="actionBtns[perms.run] && data.status === 1 && data.runningState !== 1" @click="run(data.id)" type="success" link>{{
                    $t('db.run')
                }}</el-button>
                <el-button v-if="actionBtns[perms.stop] && data.runningState === 1" @click="stop(data.id)" type="danger" link>{{ $t('db.stop') }}</el-button>
                <el-button v-if="actionBtns[perms.log]" type="primary" link @click="log(data)">{{ $t('db.log') }}</el-button>
            </template>
        </page-table>

        <data-sync-task-edit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />

        <data-sync-task-log v-model:visible="logsDialog.visible" v-model:taskId="logsDialog.taskId" :running="state.logsDialog.running" />
    </div>
</template>

<script lang="ts" setup>
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nConfirm, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { getDbDialect } from '@/views/ops/db/dialect';
import { dbSyncApi } from '@/views/ops/db/sync/api';
import { DbDataSyncModeEnum, DbDataSyncDuplicateStrategyEnum, DbDataSyncRecentStateEnum, DbDataSyncRunningStateEnum } from '@/views/ops/db/sync/enums';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import type { PageResult } from '@/types/common';
import type { DataSyncTaskListVO } from '../types';

const DataSyncTaskEdit = defineAsyncComponent(() => import('./SyncTaskEdit.vue'));
const DataSyncTaskLog = defineAsyncComponent(() => import('./SyncTaskLog.vue'));

const { t } = useI18n();

/** 归一 status 字段：Go int8 零值 0 映射为 -1（停用），避免 ElSwitch model-value 校验告警 */
const handleData = (res: PageResult<DataSyncTaskListVO>) => {
    for (const task of res.list) {
        if (task.status !== 1 && task.status !== -1) {
            task.status = -1;
        }
    }
    return res;
};

const perms = {
    save: 'db:sync:save',
    del: 'db:sync:del',
    status: 'db:sync:status',
    log: 'db:sync:log',
    run: 'db:sync:run',
    stop: 'db:sync:stop',
};

const searchItems = [SearchItem.input('name', 'common.name')];

/**
 * 默认收起的列：创建/修改两组审计信息只需常驻一组，分页大小与键冲突策略取值基本不变，
 * 逐列常驻会把操作列顶到屏外；需要时在右上角「表格配置」里按需勾选
 */
const optionalColumns = [
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('pageSize', 'db.pageSize').alignCenter(),
    TableColumn.new('duplicateStrategy', 'db.keyDuplicateStrategy').typeTag(DbDataSyncDuplicateStrategyEnum).alignCenter(),
];
optionalColumns.forEach((column) => (column.show = 0));

// 身份 → 同步链路（源库 → 目标库表）→ 增量水位/定时 → 当前与最近执行态 → 启停用 → 修改信息
const columns = ref([
    // 任务名是主识别信息，预留比默认更宽的宽度，过长仍可由溢出 tooltip 看全称
    TableColumn.new('taskName', 'db.taskName').setMinWidth(180).isSlot(),
    TableColumn.new('syncMode', 'db.syncMode').typeTag(DbDataSyncModeEnum).alignCenter(),
    // 插槽内已自带 el-tooltip，需关掉列的溢出提示（否则叠成两个气泡）；
    // 省略号不跟该开关走，改由插槽内的 block truncate 自己保证单行
    TableColumn.new('srcDb', 'db.srcDb').setMinWidth(140).isSlot().noShowOverflowTooltip(),
    TableColumn.new('targetTable', 'db.targetDbTable').setMinWidth(180).isSlot().noShowOverflowTooltip(),
    TableColumn.new('updFieldVal', 'db.syncCursor').alignCenter().isSlot().setMinWidth(160),
    TableColumn.new('cron', 'db.cron')
        .alignCenter()
        .setFormatFunc((data: DataSyncTaskListVO) => data.cron || t('db.manual')),
    TableColumn.new('runningState', 'db.runState').typeTag(DbDataSyncRunningStateEnum).alignCenter(),
    TableColumn.new('recentState', 'db.recentState').typeTag(DbDataSyncRecentStateEnum).alignCenter(),
    TableColumn.new('status', 'common.status').isSlot(),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
    ...optionalColumns,
]);

/** 增量水位只对「增量追加/增量合并」生效，其余模式恒为全量扫描源查询，展示不适用 */
const cursorModes: number[] = [DbDataSyncModeEnum.IncrementalAppend.value, DbDataSyncModeEnum.IncrementalMerge.value];

const syncCursorText = (data: DataSyncTaskListVO) => {
    if (!cursorModes.includes(data.syncMode)) return t('db.notApplicable');
    // 水位为空或 '0' 表示尚未推进，下一次执行会全量拉取源查询结果
    if (!data.updField || !data.updFieldVal || data.updFieldVal === '0') return t('db.syncCursorInit');
    return `${data.updField}: ${data.updFieldVal}`;
};

/** 双向同步标记：反向任务未创建（reverseTaskId 为 0）时配对不完整，需能直接看出来 */
const biDirTip = (data: DataSyncTaskListVO) => (data.reverseTaskId ? `${t('db.biDirEnabled')} → #${data.reverseTaskId}` : t('db.biDirReverseMissing'));

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del, perms.status, perms.log, perms.run, perms.stop]);
const actionWidth =
    ((actionBtns[perms.save] ? 1 : 0) + (actionBtns[perms.log] ? 1 : 0) + (actionBtns[perms.run] ? 1 : 0) + (actionBtns[perms.stop] ? 1 : 0)) * 55;
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().setMinWidth(actionWidth).fixedRight().alignCenter();
const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

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
        data: null as DataSyncTaskListVO | null,
        title: '',
    },
    logsDialog: {
        taskId: 0,
        visible: false,
        data: null as DataSyncTaskListVO | null,
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
    pageTableRef.value?.search();
};

const edit = async (data: DataSyncTaskListVO | false) => {
    if (!data) {
        state.editDialog.data = null;
        state.editDialog.title = useI18nCreateTitle('db.dbSync');
    } else {
        state.editDialog.data = data;
        state.editDialog.title = useI18nEditTitle('db.dbSync');
    }
    state.editDialog.visible = true;
};

const run = async (id: number) => {
    await useI18nConfirm('db.runConfirm');
    try {
        await dbSyncApi.runDatasyncTask.request({ taskId: id });
        Msg.operateSuccess();
        // 执行后自动弹出日志弹窗，查看实时执行日志
        state.logsDialog.taskId = id;
        state.logsDialog.visible = true;
        state.logsDialog.running = true;
    } catch (e) {
        //
    }
    setTimeout(search, 2000);
};

const stop = async (id: number) => {
    await useI18nConfirm('db.stopConfirm');
    await dbSyncApi.stopDatasyncTask.request({ taskId: id });
    Msg.operateSuccess();
    search();
};

const log = async (data: DataSyncTaskListVO) => {
    state.logsDialog.taskId = data.id;
    state.logsDialog.visible = true;
    state.logsDialog.running = data.runningState === 1;
};

const updStatus = async (id: number, status: 1 | -1) => {
    try {
        await dbSyncApi.updateDatasyncTaskStatus.request({ taskId: id, status });
        Msg.operateSuccess();
        search();
    } catch (err) {
        //
    }
};

const del = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: DataSyncTaskListVO) => x.taskName).join('、'));
        await dbSyncApi.deleteDatasyncTask.request({ taskId: state.selectionData.map((x: DataSyncTaskListVO) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
