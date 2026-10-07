<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="dbTransferApi.dbTransferTasks"
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
                    <span class="block truncate">
                        <SvgIcon :name="getDbDialect(data.srcDbType).getInfo().icon" :size="18" />
                        {{ data.srcDbName }}
                    </span>
                </el-tooltip>
            </template>
            <template #targetDb="{ data }">
                <!-- 迁移到文件没有目标库，该列展示生成 SQL 所用的方言 -->
                <el-tooltip v-if="data.mode === DbTransferModeEnum.File.value" :content="fileTargetTip(data)">
                    <span class="block truncate">
                        <SvgIcon v-if="data.targetFileDbType" :name="getDbDialect(data.targetFileDbType).getInfo().icon" :size="18" />
                        {{ fileTargetDialectName(data.targetFileDbType) }}
                    </span>
                </el-tooltip>
                <el-tooltip v-else :content="`${data.targetTagPath} > ${data.targetInstName} > ${data.targetDbName}`">
                    <span class="block truncate">
                        <SvgIcon :name="getDbDialect(data.targetDbType).getInfo().icon" :size="18" />
                        {{ data.targetDbName }}
                    </span>
                </el-tooltip>
            </template>
            <!-- 迁移范围：全部表 / N 张表（悬停看表名清单）/ 未设置，用普通文本而非 tag，避免被当成状态标签 -->
            <template #checkedKeys="{ data }">
                <span v-if="isAllTables(data.checkedKeys)" class="block truncate">{{ $t('db.allTable') }}</span>
                <span v-else-if="!transferTables(data.checkedKeys).length" class="block truncate text-red-500">{{ $t('db.pleaseSetting') }}</span>
                <el-tooltip v-else :content="transferTablesTip(data.checkedKeys)">
                    <span class="block truncate">{{ $t('db.transferTablesCount', { count: transferTables(data.checkedKeys).length }) }}</span>
                </el-tooltip>
            </template>

            <template #status="{ data }">
                <span v-if="actionBtns[perms.status]">
                    <!-- status 可能存在历史非法值（如 0），归一化后展示，避免 [ElSwitch] model-value 警告 -->
                    <el-switch
                        :model-value="data.status === 1 ? 1 : -1"
                        @change="(val: any) => updStatus(data.id, val)"
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
                <el-button v-if="actionBtns[perms.log]" type="warning" link @click="onOpenLog(data)">{{ $t('db.log') }}</el-button>
                <el-button
                    v-if="actionBtns[perms.stop] && data.runningState === DbTransferRunningStateEnum.Running.value"
                    @click="stop(data.id)"
                    type="danger"
                    link
                >
                    {{ $t('db.stop') }}
                </el-button>
                <el-button
                    v-if="actionBtns[perms.run] && data.runningState !== DbTransferRunningStateEnum.Running.value && data.status === 1"
                    type="success"
                    link
                    @click="onReRun(data)"
                >
                    {{ $t('db.run') }}
                </el-button>
                <el-button
                    v-if="actionBtns[perms.verify] && data.mode === 1 && data.runningState !== DbTransferRunningStateEnum.Running.value"
                    type="info"
                    link
                    @click="onVerify(data)"
                >
                    {{ $t('db.verify') }}
                </el-button>
                <el-button v-if="actionBtns[perms.files] && data.mode === 2" type="success" link @click="openFiles(data)">{{ $t('db.file') }}</el-button>
            </template>
        </page-table>

        <db-transfer-edit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
        <db-transfer-file :title="filesDialog.title" v-model:visible="filesDialog.visible" v-model:data="filesDialog.data" />

        <!-- 迁移日志历史列表 -->
        <db-transfer-log :task-id="logsDialog.taskId" :running="logsDialog.running" v-model:visible="logsDialog.visible" @cancel="search" />
    </div>
</template>

<script lang="ts" setup>
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nConfirm, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { getDbDialect } from '@/views/ops/db/dialect';
import { dbTransferApi } from '@/views/ops/db/transfer/api';
import {
    DbTransferRunningStateEnum,
    DbTransferModeEnum,
    DbTransferStrategyEnum,
    DbTransferDeleteTableEnum,
    DbTransferNameCaseEnum,
} from '@/views/ops/db/transfer/enums';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DbTransferTaskListVO } from '../types';

const DbTransferEdit = defineAsyncComponent(() => import('./DbTransferEdit.vue'));
const DbTransferFile = defineAsyncComponent(() => import('./DbTransferFile.vue'));
const DbTransferLog = defineAsyncComponent(() => import('./DbTransferLog.vue'));

const { t } = useI18n();

const perms = {
    save: 'db:transfer:save',
    del: 'db:transfer:del',
    status: 'db:transfer:status',
    log: 'db:transfer:log',
    run: 'db:transfer:run',
    stop: 'db:transfer:stop',
    verify: 'db:transfer:verify',
    files: 'db:transfer:files',
};

const searchItems = [SearchItem.input('name', 'common.name')];

/**
 * 默认收起的列：配置类列取值多为默认值（全量、不转换、并发度 4），
 * 创建/修改两组审计信息只需常驻一组，逐列常驻会把操作列顶到屏外；
 * 需要时在右上角「表格配置」里按需勾选
 */
const optionalColumns = [
    TableColumn.new('strategy', 'db.transferStrategy').typeTag(DbTransferStrategyEnum).alignCenter(),
    TableColumn.new('deleteTable', 'db.deleteTable').typeTag(DbTransferDeleteTableEnum).alignCenter(),
    TableColumn.new('nameCase', 'db.nameCase').typeTag(DbTransferNameCaseEnum).alignCenter(),
    TableColumn.new('concurrency', 'db.concurrency').alignCenter(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];
optionalColumns.forEach((column) => (column.show = 0));

const columns = ref([
    // 任务名是主识别信息，预留比默认更宽的宽度，过长仍可由溢出 tooltip 看全称
    TableColumn.new('taskName', 'db.taskName').setMinWidth(180).isSlot(),
    TableColumn.new('mode', 'db.transferMode').typeTag(DbTransferModeEnum).alignCenter(),
    // 以下三列插槽内已自带 el-tooltip，需关掉列的溢出提示（否则叠成两个气泡）；
    // 省略号不跟该开关走，改由插槽内的 block truncate 自己保证单行
    TableColumn.new('srcDb', 'db.srcDb').setMinWidth(150).isSlot().noShowOverflowTooltip(),
    TableColumn.new('targetDb', 'db.transferTarget').setMinWidth(150).isSlot().noShowOverflowTooltip(),
    TableColumn.new('checkedKeys', 'db.transferScope').alignCenter().isSlot().setMinWidth(100).noShowOverflowTooltip(),
    // 未启用定时时该列展示“手动”，不留空白占位符，避免被当成数据缺失
    TableColumn.new('cron', 'db.cronEnabled')
        .alignCenter()
        .setFormatFunc((data: DbTransferTaskListVO) => (data.cronEnabled === 1 ? data.cron : t('db.manual'))),
    TableColumn.new('runningState', 'db.runState').typeTag(DbTransferRunningStateEnum),
    TableColumn.new('status', 'common.status').isSlot(),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
    ...optionalColumns,
]);

/**
 * 迁移范围展示辅助：checkedKeys 为 'all'（源库全表）或逗号分隔的表名清单
 */
const isAllTables = (checkedKeys?: string) => checkedKeys === 'all';

const transferTables = (checkedKeys?: string) => (checkedKeys ? checkedKeys.split(',').filter(Boolean) : []);

/** 表名清单提示：全库迁移时只展示前若干个，避免提示框过长 */
const transferTablesTip = (checkedKeys?: string) => {
    const tables = transferTables(checkedKeys);
    return `${tables.slice(0, 20).join('、')}${tables.length > 20 ? ' …' : ''}`;
};

/** 文件迁移的 SQL 方言名；历史任务可能未配置，不能回退成某个方言的名字展示，否则会误导为目标方言 */
const fileTargetDialectName = (dbType?: string) => (dbType ? getDbDialect(dbType).getInfo().name : t('db.pleaseSetting'));

/** 文件迁移的目标信息：无目标库，按生成 SQL 的方言展示；保留天数非正数时后端不会自动清理文件，故不展示该项 */
const fileTargetTip = (data: DbTransferTaskListVO) => {
    const dialect = fileTargetDialectName(data.targetFileDbType);
    if (data.fileSaveDays < 1) return `${t('db.transfer2File')} · ${dialect}`;
    return `${t('db.transfer2File')} · ${dialect} · ${t('db.fileSaveDays')}: ${data.fileSaveDays}${t('db.day')}`;
};

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del, perms.status, perms.log, perms.run, perms.stop, perms.verify, perms.files]);
const actionWidth =
    ((actionBtns[perms.save] ? 1 : 0) +
        (actionBtns[perms.log] ? 1 : 0) +
        (actionBtns[perms.run] ? 1 : 0) +
        (actionBtns[perms.stop] ? 1 : 0) +
        (actionBtns[perms.verify] ? 1 : 0) +
        (actionBtns[perms.files] ? 1 : 0)) *
    55;
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
        data: null as DbTransferTaskListVO | null,
        title: '',
    },
    logsDialog: {
        taskId: 0,
        title: '',
        visible: false,
        running: false,
    },
    filesDialog: {
        taskId: 0,
        title: '',
        visible: false,
        data: null as DbTransferTaskListVO | null,
    },
});

const { selectionData, query, editDialog, logsDialog, filesDialog } = toRefs(state);

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const search = () => {
    pageTableRef.value?.search();
};

const edit = async (data: DbTransferTaskListVO | false) => {
    if (!data) {
        state.editDialog.data = null;
        state.editDialog.title = t('db.createDbTransferDialogTitle');
    } else {
        state.editDialog.data = data;
        state.editDialog.title = t('db.editDbTransferDialogTitle');
    }
    state.editDialog.visible = true;
};

const stop = async (id: number) => {
    if (!(await useI18nConfirm('db.stopConfirm'))) {
        return;
    }
    await dbTransferApi.stopDbTransferTask.request({ taskId: id });
    Msg.operateSuccess();
    search();
};

const onOpenLog = (data: DbTransferTaskListVO, running = false) => {
    // 列表行是任务 VO，主键字段为 id（此前取 data.taskId 得到 undefined，日志接口会按空 taskId 请求）
    state.logsDialog.taskId = data.id;
    state.logsDialog.visible = true;
    state.logsDialog.title = t('db.log');
    state.logsDialog.running = running || data.runningState === DbTransferRunningStateEnum.Running.value;
};

const onReRun = async (data: DbTransferTaskListVO) => {
    if (!(await useI18nConfirm('db.runConfirm'))) {
        return;
    }
    try {
        await dbTransferApi.runDbTransferTask.request({ taskId: data.id });
        Msg.operateSuccess();
        // 执行后弹出日志弹窗
        onOpenLog(data, true);
    } catch (e) {
        //
    }
    // 延迟2秒执行，后端异步执行
    setTimeout(() => {
        search();
    }, 2000);
};

const onVerify = async (data: DbTransferTaskListVO) => {
    if (!(await useI18nConfirm('db.verifyConfirm'))) {
        return;
    }
    try {
        await dbTransferApi.verifyDbTransferTask.request({ taskId: data.id });
        Msg.operateSuccess();
        // 校验为异步任务，弹出日志弹窗查看校验报告
        onOpenLog(data, true);
    } catch (e) {
        //
    }
};

const openFiles = async (data: DbTransferTaskListVO) => {
    state.filesDialog.visible = true;
    state.filesDialog.title = t('db.transferFileManage');
    state.filesDialog.taskId = data.id;
    state.filesDialog.data = data;
};
const updStatus = async (id: number, status: 1 | -1) => {
    try {
        await dbTransferApi.updateDbTransferTaskStatus.request({ taskId: id, status });
        Msg.operateSuccess();
        search();
    } catch (err) {
        //
    }
};

const del = async () => {
    try {
        if (!(await useI18nDeleteConfirm(state.selectionData.map((x: DbTransferTaskListVO) => x.taskName).join('、')))) {
            return;
        }
        await dbTransferApi.deleteDbTransferTask.request({ taskId: state.selectionData.map((x: DbTransferTaskListVO) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
