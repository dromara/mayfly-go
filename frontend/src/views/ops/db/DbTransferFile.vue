<template>
    <div class="db-transfer-file">
        <el-dialog :title="title" v-model="dialogVisible" :close-on-click-modal="false" :destroy-on-close="true" width="1000px">
            <page-table
                ref="pageTableRef"
                :data="state.tableData"
                v-model:query-form="state.query"
                :show-selection="true"
                v-model:selection-data="state.selectionData"
                :columns="columns"
                @page-num-change="
                    (args) => {
                        state.query.pageNum = args.pageNum;
                        search();
                    }
                "
                @page-size-change="
                    (args) => {
                        state.query.pageSize = args.pageNum;
                        search();
                    }
                "
            >
                <template #tableHeader>
                    <el-button v-auth="perms.del" :disabled="state.selectionData.length < 1" @click="del()" type="danger" icon="delete">
                        {{ $t('common.delete') }}
                    </el-button>
                </template>

                <template #fileKey="{ data }">
                    <FileInfo :fileKey="data.fileKey" :canDownload="actionBtns[perms.down] && data.status === 2" />
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
                        @click="openRun(data)"
                        type="primary"
                        link
                        >{{ $t('db.run') }}</el-button
                    >
                    <el-button v-if="data.logId" @click="openLog(data)" type="success" link>{{ $t('db.log') }}</el-button>
                </template>
            </page-table>
            <TerminalLog v-model:log-id="state.logsDialog.logId" v-model:visible="state.logsDialog.visible" :title="state.logsDialog.title" />
        </el-dialog>

        <el-dialog :title="state.runDialog.title" v-model="state.runDialog.visible" :destroy-on-close="true" width="600px">
            <el-form :model="state.runDialog.runForm" ref="runFormRef" label-width="auto" :rules="state.runDialog.formRules">
                <el-form-item :label="$t('db.dbFileType')" prop="dbType">
                    <SvgIcon :name="getDbDialect(state.runDialog.runForm.dbType).getInfo().icon" :size="18" /> {{ state.runDialog.runForm.dbType }}
                </el-form-item>
                <el-form-item :label="$t('db.targetDb')" prop="targetDbId" required>
                    <db-select-tree
                        v-model:db-id="state.runDialog.runForm.targetDbId"
                        v-model:inst-name="state.runDialog.runForm.targetInstName"
                        v-model:db-name="state.runDialog.runForm.targetDbName"
                        v-model:tag-path="state.runDialog.runForm.targetTagPath"
                        v-model:db-type="state.runDialog.runForm.targetDbType"
                        @select-db="state.runDialog.onSelectRunTargetDb"
                    />
                </el-form-item>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="state.runDialog.cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="state.runDialog.loading" @click="state.runDialog.btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, Ref, ref, watch } from 'vue';
import { dbApi } from '@/views/ops/db/api';
import { getDbDialect } from '@/views/ops/db/dialect';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { ElMessage, ElMessageBox } from 'element-plus';
import { hasPerms } from '@/components/auth/auth';
import TerminalLog from '@/components/terminal/TerminalLog.vue';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import { getClientId } from '@/common/utils/storage';
import FileInfo from '@/components/file/FileInfo.vue';
import { DbTransferFileStatusEnum } from './enums';
import { useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nFormValidate, useI18nPleaseSelect, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: [Object],
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

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

const runFormRef: any = ref(null);

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
        data: null as any,
        running: false,
    },
    runDialog: {
        title: t('db.transferFileRunDialogTitle'),
        visible: false,
        data: null as any,
        formRules: {
            targetDbId: [
                {
                    required: true,
                    message: useI18nPleaseSelect('db.targetDb'),
                    trigger: ['change', 'blur'],
                },
            ],
        },
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
        cancel: function () {
            state.runDialog.visible = false;
            state.runDialog.runForm = {} as any;
        },
        btnOk: async function () {
            await useI18nFormValidate(runFormRef);
            console.log(state.runDialog.runForm);
            if (state.runDialog.runForm.targetDbType !== state.runDialog.runForm.dbType) {
                ElMessage.warning(t('db.targetDbTypeSelectError', { dbType: state.runDialog.runForm.dbType }));
                return false;
            }
            state.runDialog.runForm.clientId = getClientId();
            await dbApi.dbTransferFileRun.request(state.runDialog.runForm);
            useI18nSaveSuccessMsg();
            state.runDialog.cancel();
            await search();
        },
        onSelectRunTargetDb: function (param: any) {
            if (param.type !== state.runDialog.runForm.dbType) {
                ElMessage.warning(t('db.targetDbTypeSelectError', { dbType: state.runDialog.runForm.dbType }));
            }
        },
    },
    selectionData: [], // 选中的数据
    tableData: [],
});

const search = async () => {
    const { total, list } = await dbApi.dbTransferFileList.request(state.query);
    state.tableData = list;
    pageTableRef.value.total = total;
};

const pageTableRef: Ref<any> = ref(null);

const del = async function () {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.fileKey).join('、'));
        await dbApi.dbTransferFileDel.request({ fileId: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        await search();
    } catch (err) {
        //
    }
};

const openLog = function (data: any) {
    state.logsDialog.logId = data.logId;
    state.logsDialog.visible = true;
    state.logsDialog.title = t('db.log');
    state.logsDialog.running = data.state === 1;
};

// 运行sql，弹出选择需要运行的库，默认运行当前数据库，需要保证数据库类型与sql文件一致
const openRun = function (data: any) {
    console.log(data);
    state.runDialog.runForm = { id: data.id, dbType: data.fileDbType } as any;
    console.log(state.runDialog.runForm);
    state.runDialog.visible = true;
};

watch(dialogVisible, async (newValue: boolean) => {
    if (!newValue) {
        return;
    }
    state.query.taskId = props.data?.id;
    state.query.pageNum = 1;
    state.query.pageSize = 10;

    await search();
});
</script>
<style lang="scss"></style>
