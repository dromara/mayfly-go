<template>
    <div class="db-transfer-file">
        <el-dialog
            @open="search()"
            :title="title"
            v-model="dialogVisible"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            body-class="h-[65vh]"
            width="1000px"
        >
            <page-table
                ref="pageTableRef"
                v-model:query-form="state.query"
                :page-api="dbTransferApi.dbTransferFileList"
                :lazy="true"
                :show-selection="true"
                v-model:selection-data="state.selectionData"
                :columns="columns"
            >
                <template #tableHeader>
                    <el-button v-auth="perms.del" :disabled="state.selectionData.length < 1" @click="onDel()" type="danger" icon="delete">
                        {{ $t('common.delete') }}
                    </el-button>
                </template>

                <template #fileKey="{ data }">
                    <FileInfo :fileKey="data.fileKey" show-file-size :canDownload="actionBtns[perms.down] && data.status === 2" />
                </template>

                <template #fileDbType="{ data }">
                    <span>
                        <SvgIcon :name="getDbDialect(data.fileDbType).getInfo().icon" :size="18" />
                        {{ data.fileDbType }}
                    </span>
                </template>

                <template #action="{ data }">
                    <el-button
                        v-if="actionBtns[perms.run] && data.status === DbTransferFileStatusEnum.Success.value"
                        @click="onOpenRun(data)"
                        type="primary"
                        link
                    >
                        {{ $t('db.run') }}
                    </el-button>

                    <el-button v-if="data.logId" @click="onOpenLog(data)" type="success" link>{{ $t('db.log') }}</el-button>
                </template>
            </page-table>
        </el-dialog>

        <TerminalLog v-model:log-id="state.logsDialog.logId" v-model:visible="state.logsDialog.visible" :title="state.logsDialog.title" />

        <el-dialog :title="state.runDialog.title" v-model="state.runDialog.visible" :destroy-on-close="true" width="600px">
            <auto-form v-model="state.runDialog.runForm" :items="runFormItems" label-width="auto">
                <template #dbType>
                    <span>
                        <SvgIcon :name="getDbDialect(state.runDialog.runForm.dbType).getInfo().icon" :size="18" />
                        {{ state.runDialog.runForm.dbType }}
                    </span>
                </template>
                <template #targetDbId>
                    <db-select-tree
                        v-model:db-id="state.runDialog.runForm.targetDbId"
                        v-model:inst-name="state.runDialog.runForm.targetInstName"
                        v-model:db-name="state.runDialog.runForm.targetDbName"
                        v-model:tag-path="state.runDialog.runForm.targetTagPath"
                        v-model:db-type="state.runDialog.runForm.targetDbType"
                        @select-db="state.runDialog.onSelectRunTargetDb"
                    />
                </template>
            </auto-form>

            <template #footer>
                <el-button @click="state.runDialog.onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="state.runDialog.loading" @click="state.runDialog.onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { getClientId } from '@/common/utils/storage';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { hasPerms } from '@/components/auth/auth';
import FileInfo from '@/components/file/FileInfo.vue';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import TerminalLog from '@/components/terminal/TerminalLog.vue';
import { Msg, useI18nDeleteConfirm, useI18nFormValidate } from '@/hooks/useI18n';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import { getDbDialect } from '@/views/ops/db/dialect';
import { dbTransferApi } from '@/views/ops/db/transfer/api';
import { DbTransferFileStatusEnum } from '@/views/ops/db/transfer/enums';
import { onMounted, reactive, ref, useTemplateRef, watch, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DbTransferTask } from '../types';

interface DbTransferFile {
    id: number;
    fileKey: string;
    fileDbType: string;
    status: number;
    logId: number;
    state: number;
    createTime: string;
}

interface RunForm {
    id: number;
    dbType: string;
    clientId: string;
    targetDbId: number;
    targetDbName: string;
    targetTagPath: string;
    targetInstName: string;
    targetDbType: string;
}

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<DbTransferTask | null>,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const columns = ref([
    TableColumn.new('fileKey', 'db.file').setMinWidth(280).isSlot(),
    TableColumn.new('createTime', 'db.execTime').setMinWidth(180).isTime(),
    TableColumn.new('fileDbType', 'db.fileDbType').setMinWidth(90).isSlot(),
    TableColumn.new('status', 'common.status').typeTag(DbTransferFileStatusEnum),
]);

const perms = {
    del: 'db:transfer:files:del',
    down: 'db:transfer:files:down',
    run: 'db:transfer:files:run',
};

const actionBtns = hasPerms([perms.del, perms.down, perms.run]);

const actionWidth = ((actionBtns[perms.run] ? 1 : 0) + 1) * 55;

const actionColumn = TableColumn.new('action', 'common.operation').isSlot().setMinWidth(actionWidth).fixedRight().alignCenter();

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const runFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('runFormRef');

/** 运行弹窗表单声明（dbType 仅展示文件库类型，targetDbId 走 custom 插槽选择目标库） */
const runFormItems: AutoFormItem[] = [
    { prop: 'dbType', label: 'db.dbFileType', type: 'custom' },
    { prop: 'targetDbId', label: 'db.targetDb', type: 'custom', rules: [Rules.requiredSelect('db.targetDb')] },
];

const state = reactive({
    query: {
        taskId: props.data?.id,
        name: null,
        pageNum: 1,
        pageSize: 10,
    },
    logsDialog: {
        logId: 0,
        title: '数据库迁移日志',
        visible: false,
        data: null as DbTransferFile | null,
        running: false,
    },
    runDialog: {
        title: t('db.transferFileRunDialogTitle'),
        visible: false,
        data: null as DbTransferFile | null,
        runForm: {
            id: 0,
            dbType: '',
            clientId: '',
            targetDbId: 0,
            targetDbName: '',
            targetTagPath: '',
            targetInstName: '',
            targetDbType: '',
        },
        loading: false,
        onCancel: function () {
            state.runDialog.visible = false;
            state.runDialog.runForm = {} as RunForm;
        },
        onConfirm: async function () {
            await useI18nFormValidate(runFormRef);
            if (state.runDialog.runForm.targetDbType !== state.runDialog.runForm.dbType) {
                Msg.warning('db.targetDbTypeSelectError', { dbType: state.runDialog.runForm.dbType });
                return false;
            }
            state.runDialog.runForm.clientId = getClientId();
            await dbTransferApi.dbTransferFileRun.request(state.runDialog.runForm);
            Msg.operateSuccess();
            state.runDialog.onCancel();
            await search();
        },
        onSelectRunTargetDb: function (param: { type: string }) {
            if (param.type !== state.runDialog.runForm.dbType) {
                Msg.warning('db.targetDbTypeSelectError', { dbType: state.runDialog.runForm.dbType });
            }
        },
    },
    selectionData: [] as DbTransferFile[], // 选中的数据
    tableData: [],
});

const search = async () => {
    pageTableRef.value?.search();
    // const { total, list } = await dbTransferApi.dbTransferFileList.request(state.query);
    // state.tableData = list;
    // pageTableRef.value?.total = total;
};

const onDel = async function () {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: DbTransferFile) => x.fileKey).join('、'));
        await dbTransferApi.dbTransferFileDel.request({ fileId: state.selectionData.map((x: DbTransferFile) => x.id).join(',') });
        Msg.deleteSuccess();
        await search();
    } catch (err) {
        //
    }
};

const onOpenLog = function (data: { logId: number; state: number }) {
    state.logsDialog.logId = data.logId;
    state.logsDialog.visible = true;
    state.logsDialog.title = t('db.log');
    state.logsDialog.running = data.state === 1;
};

// 运行sql，弹出选择需要运行的库，默认运行当前数据库，需要保证数据库类型与sql文件一致
const onOpenRun = function (data: DbTransferFile) {
    state.runDialog.runForm = { id: data.id, dbType: data.fileDbType } as RunForm;
    state.runDialog.visible = true;
};

watch(dialogVisible, async (newValue: boolean) => {
    if (!newValue) {
        return;
    }
    state.query.taskId = props.data?.id || 0;
    state.query.pageNum = 1;
    state.query.pageSize = 10;

    await search();
});
</script>
<style lang="scss"></style>
