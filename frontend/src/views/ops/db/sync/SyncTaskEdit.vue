<template>
    <div class="sync-task-edit">
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="dialogVisible"
            :title="title"
            v-model:active-tab="tabActiveName"
            :tabs="tabs"
            :data="editData"
            size="45%"
            :confirm-api="btnOk"
            @opened="onOpened"
            @cancel="emit('cancel')"
        >
            <!-- cron 表达式编辑器 -->
            <template #taskCron="{ form }">
                <CrontabInput v-model="form.taskCron" />
            </template>

            <!-- 源库选择 -->
            <template #srcDbId="{ form }">
                <db-select-tree
                    v-model:db-id="form.srcDbId"
                    v-model:inst-name="form.srcInstName"
                    v-model:db-name="form.srcDbName"
                    v-model:tag-path="form.srcTagPath"
                    v-model:db-type="form.srcDbType"
                    @select-db="onSelectSrcDb"
                />
            </template>

            <!-- 目标库选择 -->
            <template #targetDbId="{ form }">
                <db-select-tree
                    v-model:db-id="form.targetDbId"
                    v-model:inst-name="form.targetInstName"
                    v-model:db-name="form.targetDbName"
                    v-model:tag-path="form.targetTagPath"
                    v-model:db-type="form.targetDbType"
                    @select-db="onSelectTargetDb"
                />
            </template>

            <!-- 字段映射表格编辑 -->
            <template #fieldMap="{ form }">
                <el-table :data="form.fieldMap" :max-height="state.fieldMapTableHeight">
                    <el-table-column prop="src" :label="$t('db.srcField')" :width="200"></el-table-column>
                    <el-table-column prop="target" :label="$t('db.targetField')">
                        <template #default="scope">
                            <el-select v-model="scope.row.target" allow-create filterable>
                                <el-option
                                    v-for="item in state.targetColumnList"
                                    :key="item.columnName"
                                    :label="`${item.columnType}${item.columnComment && ' - ' + item.columnComment}`"
                                    :value="item.columnName"
                                >
                                    <div class="flex justify-between">
                                        {{ item.columnName }}
                                        <el-text size="small">
                                            {{ item.columnType }}{{ item.columnComment && ' - ' + item.columnComment }}
                                        </el-text>
                                    </div>
                                </el-option>
                            </el-select>
                        </template>
                    </el-table-column>
                    <!-- Phase 3: 转换类型列 -->
                    <el-table-column prop="transformType" :label="$t('db.transformRules')" :width="160">
                        <template #default="scope">
                            <el-select v-model="scope.row.transformType" size="small">
                                <el-option :label="$t('db.none')" value="column"></el-option>
                                <el-option :label="$t('db.transformTypeConstant')" value="constant"></el-option>
                                <el-option :label="$t('db.transformTypeExpr')" value="expr"></el-option>
                            </el-select>
                        </template>
                    </el-table-column>
                    <!-- Phase 3: 转换配置列 -->
                    <el-table-column prop="transformConfig" :label="$t('db.transformConfig')" :width="200">
                        <template #default="scope">
                            <el-input
                                v-if="scope.row.transformType === 'constant'"
                                v-model="scope.row.transformConfig"
                                size="small"
                                :placeholder="$t('db.transformConstantPlaceholder')"
                            />
                            <el-input
                                v-if="scope.row.transformType === 'expr'"
                                v-model="scope.row.transformConfig"
                                size="small"
                                :placeholder="$t('db.transformExprPlaceholder')"
                            />
                        </template>
                    </el-table-column>
                </el-table>
            </template>

            <!-- SQL 预览 -->
            <template #previewDataSql>
                <el-input type="textarea" :model-value="state.previewDataSql" readonly :rows="10" />
            </template>
            <template #previewInsertSql>
                <el-input type="textarea" :model-value="state.previewInsertSql" readonly :rows="10" />
            </template>

            <template #footer>
                <div>
                    <el-button
                        v-if="tabActiveName != basicTab"
                        @click="
                            () => {
                                switch (tabActiveName) {
                                    case fieldTab:
                                        tabActiveName = basicTab;
                                        break;
                                    case advancedTab:
                                        tabActiveName = fieldTab;
                                        break;
                                    case sqlPreviewTab:
                                        tabActiveName = advancedTab;
                                        break;
                                }
                            }
                        "
                        >{{ $t('common.previousStep') }}</el-button
                    >
                    <el-button
                        v-if="tabActiveName != sqlPreviewTab"
                        :disabled="!baseFieldCompleted"
                        @click="
                            () => {
                                switch (tabActiveName) {
                                    case basicTab:
                                        tabActiveName = fieldTab;
                                        break;
                                    case fieldTab:
                                        tabActiveName = advancedTab;
                                        break;
                                    case advancedTab:
                                        tabActiveName = sqlPreviewTab;
                                        break;
                                }
                            }
                        "
                        >{{ $t('common.nextStep') }}</el-button
                    >

                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit()">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import { AutoFormDrawer, type AutoFormData, type AutoFormTab } from '@/components/auto-form';
import { Msg } from '@/hooks/useI18n';
import { dbApi } from '@/views/ops/db/api';
import DbSelectTree from '@/views/ops/db/widgets/DbSelectTree.vue';
import { DbInst } from '@/views/ops/db/db';
import { createSqlCompletionScope } from '@/views/ops/db/completion/lazy';
import { getDbDialect, getDialectCapabilities } from '@/views/ops/db/dialect';
import { dbSyncApi } from '@/views/ops/db/sync/api';
import {
    DbDataSyncDuplicateStrategyEnum,
    DbDataSyncModeEnum,
    DbNullStrategyEnum,
    DbSchemaEvolveModeEnum,
    DbConflictStrategyEnum,
} from '@/views/ops/db/sync/enums';
import { computed, onBeforeUnmount, reactive, ref, useTemplateRef, watch, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import type { ColumnMetadata, DataSyncTask, Db, DbNodeParams, DbTableInfo } from '@/views/ops/db/types';

const { t } = useI18n();

const sqlCompletion = createSqlCompletionScope();
onBeforeUnmount(() => sqlCompletion.release());

const props = defineProps({
    data: {
        type: Object as PropType<DataSyncTask | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const emit = defineEmits<{
    cancel: [];
    'val-change': [form: AutoFormData];
}>();

const dialogVisible = defineModel<boolean>('visible', { default: false });

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown>; submitting: boolean; submit: () => Promise<void> }>('drawerRef');

const basicTab = 'basic';
const fieldTab = 'field';
const advancedTab = 'advanced';
const sqlPreviewTab = 'sqlPreview';

const targetTableOptions = () =>
    Promise.resolve(state.targetTableList.map((item) => ({ value: item.tableName, label: item.tableName + (item.tableComment && '-' + item.tableComment) })));

const tabs: AutoFormTab[] = [
    {
        name: basicTab,
        label: 'common.basic',
        items: [
            { prop: 'taskName', label: 'db.taskName', required: true, span: 12 },
            { prop: 'taskCron', label: 'cron', type: 'custom', required: true, span: 8 },
            {
                prop: 'status',
                label: 'common.status',
                type: 'switch',
                required: true,
                span: 4,
                props: { inlinePrompt: true, activeText: t('common.enable'), inactiveText: t('common.disable'), activeValue: 1, inactiveValue: -1 },
            },
            {
                prop: 'syncMode',
                label: 'db.syncMode',
                type: 'enum',
                enums: DbDataSyncModeEnum,
                required: true,
                span: 8,
                tooltip: 'db.syncModeTips',
            },
            { prop: 'srcDbId', label: 'db.srcDb', type: 'custom', required: true },
            { prop: 'targetDbId', label: 'db.targetDb', type: 'custom', required: true },
            { prop: 'dataSql', label: 'db.srcDataSql', type: 'monaco', required: true, props: { language: 'sql', height: '200px' } },
            { prop: 'targetTableName', label: 'db.targetDbTable', type: 'select', required: true, span: 12, options: targetTableOptions },
            { prop: 'pageSize', label: 'db.pageSize', type: 'number', required: true, span: 12, placeholder: 'db.pageSizePlaceholder' },
            { prop: 'updField', label: 'db.updateField', tooltip: 'db.updateFieldTips', placeholder: 'db.updateFiledPlaceholder', span: 12 },
            { prop: 'updFieldVal', label: 'db.updateFieldValue', tooltip: 'db.updateFieldValueTips', placeholder: 'db.updateFieldValuePlaceholder', span: 12 },
            { prop: 'updFieldSrc', label: 'db.fieldValueSrc', tooltip: 'db.fieldValueSrcTips', placeholder: 'db.fieldValueSrcPlaceholder', span: 12 },
            // Phase 2: 辅助增量字段
            { prop: 'updFieldSecondary', label: 'db.updFieldSecondary', tooltip: 'db.updFieldSecondaryTips', placeholder: 'db.updFieldSecondaryPlaceholder', span: 12 },
        ],
    },
    {
        name: fieldTab,
        label: 'db.fieldMap',
        disabled: () => !baseFieldCompleted.value,
        items: [{ prop: 'fieldMap', label: 'db.fieldMap', type: 'custom', required: true }],
    },
    // Phase 3/5/6: 高级配置 tab
    {
        name: advancedTab,
        label: 'db.advancedSettings',
        disabled: () => !baseFieldCompleted.value,
        items: [
            // Phase 3: 数据转换与过滤
            { prop: 'filterCondition', label: 'db.filterCondition', tooltip: 'db.filterConditionTips', placeholder: 'db.filterConditionPlaceholder', span: 12 },
            {
                prop: 'nullStrategy',
                label: 'db.nullStrategy',
                type: 'enum',
                enums: DbNullStrategyEnum,
                tooltip: 'db.nullStrategyTips',
                span: 8,
            },
            { prop: 'nullDefault', label: 'db.nullDefault', placeholder: 'db.nullDefaultPlaceholder', span: 8 },
            // Phase 5: Schema 演化
            {
                prop: 'schemaEvolveMode',
                label: 'db.schemaEvolveMode',
                type: 'enum',
                enums: DbSchemaEvolveModeEnum,
                tooltip: 'db.schemaEvolveTips',
                span: 8,
            },
            // Phase 6.3: 双向同步
            {
                prop: 'biDirEnabled',
                label: 'db.biDirEnabled',
                type: 'switch',
                tooltip: 'db.biDirEnabledTips',
                span: 6,
                props: { inlinePrompt: true, activeText: t('common.enable'), inactiveText: t('common.disable') },
            },
            {
                prop: 'conflictStrategy',
                label: 'db.conflictStrategy',
                type: 'enum',
                enums: DbConflictStrategyEnum,
                tooltip: 'db.conflictStrategyTips',
                span: 8,
                when: (f) => !!f.biDirEnabled,
            },
            {
                prop: 'biDirTimestampField',
                label: 'db.biDirTimestampField',
                placeholder: 'db.biDirTimestampFieldPlaceholder',
                span: 10,
                when: (f) => !!f.biDirEnabled,
            },
        ],
    },
    {
        name: sqlPreviewTab,
        label: 'db.sqlPreview',
        disabled: () => !baseFieldCompleted.value,
        items: [
            {
                prop: 'duplicateStrategy',
                label: 'db.keyDuplicateStrategy',
                type: 'enum',
                enums: DbDataSyncDuplicateStrategyEnum,
                when: (f) => getDialectCapabilities(getDbDialect(f.targetDbType!)).supportsDuplicateStrategy,
                onChange: () => handleDuplicateStrategy(),
            },
            { prop: 'previewDataSql', label: 'db.selectSql', type: 'custom' },
            { prop: 'previewInsertSql', label: 'db.insertSql', type: 'custom' },
        ],
    },
];

type FormData = {
    id?: number;
    taskName?: string;
    taskCron: string;
    srcDbId?: number;
    srcInstName?: string;
    srcDbName?: string;
    srcDbType?: string;
    srcTagPath?: string;
    targetDbId?: number;
    targetInstName?: string;
    targetDbName?: string;
    targetTagPath?: string;
    targetTableName?: string;
    targetDbType?: string;
    dataSql?: string;
    pageSize?: number;
    updField?: string;
    updFieldVal?: string;
    updFieldSrc?: string;
    updFieldSecondary?: string;
    fieldMap?: { src: string; target: string; transformType?: string; transformConfig?: string }[];
    status?: 1 | 2;
    syncMode?: 1 | 2 | 3 | 4 | 5 | 6;
    softDeleteField?: string;
    softDeleteValue?: string;
    filterCondition?: string;
    nullStrategy?: 0 | 1 | 2;
    nullDefault?: string;
    schemaEvolveMode?: 0 | 1 | 2;
    biDirEnabled?: boolean;
    conflictStrategy?: 1 | 2 | 3;
    biDirTimestampField?: string;
    duplicateStrategy?: -1 | 1 | 2;
};

const basicFormData = {
    srcDbId: -1,
    targetDbId: -1,
    dataSql: 'select * from',
    pageSize: 1000,
    updField: '',
    updFieldVal: '0',
    updFieldSecondary: '',
    fieldMap: [{ src: 'a', target: 'b', transformType: 'column', transformConfig: '' }],
    status: 1,
    syncMode: 1,
    softDeleteField: '',
    softDeleteValue: '',
    filterCondition: '',
    nullStrategy: 0,
    nullDefault: '',
    schemaEvolveMode: 0,
    biDirEnabled: false,
    conflictStrategy: 1,
    biDirTimestampField: '',
    duplicateStrategy: -1,
} as FormData;

const editData = { ...basicFormData, taskCron: '' } as unknown as AutoFormData;

const state = reactive({
    targetTableList: [] as { tableName: string; tableComment: string }[],
    targetColumnList: [] as ColumnMetadata[],
    srcDbInst: {} as DbInst,
    targetDbInst: {} as DbInst,
    previewRes: {} as Record<string, unknown>,
    previewDataSql: '',
    previewInsertSql: '',
    previewFieldArr: [] as string[],
    fieldMapTableHeight: window.innerHeight - 50,
});

const tabActiveName = ref('basic');
const internalForm = ref<AutoFormData>({});

const baseFieldCompleted = computed(() => {
    const form = internalForm.value;
    return form.srcDbId && form.srcDbName && form.targetDbId && form.targetDbName && form.targetTableName;
});

const { execute: saveExec } = dbSyncApi.saveDatasyncTask.useApi();

const onOpened = async (form: AutoFormData) => {
    internalForm.value = form;
    tabActiveName.value = 'basic';
    const propsData = props.data;
    if (!propsData?.id) {
        return;
    }

    let data = await dbSyncApi.getDatasyncTask.request({ taskId: propsData?.id });
    const formData = data as unknown as FormData;
    if (!formData.duplicateStrategy) {
        formData.duplicateStrategy = -1;
    }
    if (formData.nullStrategy === undefined) {
        formData.nullStrategy = 0;
    }
    if (formData.schemaEvolveMode === undefined) {
        formData.schemaEvolveMode = 0;
    }
    if (formData.conflictStrategy === undefined) {
        formData.conflictStrategy = 1;
    }
    try {
        const parsed = JSON.parse(data.fieldMap);
        // 兼容旧数据：无 transformType 字段时补默认值
        formData.fieldMap = parsed.map((fm: any) => ({
            src: fm.src,
            target: fm.target,
            transformType: fm.transformType || 'column',
            transformConfig: fm.transformConfig || '',
        }));
    } catch (e) {
        formData.fieldMap = [];
    }
    Object.assign(form, formData);
    let { srcDbId, srcDbName, targetDbId } = formData;

    if (srcDbId) {
        const dbInfoRes = await dbApi.dbs.request({ id: srcDbId });
        const db = dbInfoRes.list[0] as Db & { databases?: string[] };
        db.databases = db.database?.split(' ').sort() || [];
        state.srcDbInst = await DbInst.getOrNewInst(db);
        form.srcDbType = state.srcDbInst.type;
        form.srcInstName = db.name;
    }

    if (targetDbId) {
        const dbInfoRes = await dbApi.dbs.request({ id: targetDbId });
        const db = dbInfoRes.list[0] as Db & { databases?: string[] };
        db.databases = db.database?.split(' ').sort() || [];
        state.targetDbInst = await DbInst.getOrNewInst(db);
        form.targetDbType = state.targetDbInst.type;
        form.targetInstName = db.name;
    }

    if (targetDbId && formData.targetDbName) {
        await loadDbTables(targetDbId, formData.targetDbName);
    }

    if (srcDbId && srcDbName) {
        sqlCompletion.register(srcDbId, srcDbName, state.srcDbInst.databases, state.srcDbInst.type);
    }
};

watch(tabActiveName, async (newValue: string) => {
    switch (newValue) {
        case fieldTab:
            await handleGetSrcFields();
            await handleGetTargetFields();
            break;
        case sqlPreviewTab:
            let targetDbDialect = getDbDialect(state.targetDbInst.type);
            let updField = internalForm.value.updField!;

            let hasCondition = /where/i.test(internalForm.value.dataSql!);
            state.previewDataSql = `${internalForm.value.dataSql?.trim() || t('db.noDataSqlMsg')} \n ${hasCondition ? 'and' : 'where'} ${updField} > '${internalForm.value.updFieldVal || ''}'`;

            let fields = new Set();
            internalForm.value.fieldMap?.map((a: { src: string; target: string }) => {
                if (a.target) {
                    fields.add(a.target);
                }
            });
            if (fields.size < (internalForm.value.fieldMap?.length || 0)) {
                Msg.warning('db.fieldMapError');
                state.previewInsertSql = '';
                return;
            }

            let fieldArr = internalForm.value.fieldMap?.map((a: { src: string; target: string }) => targetDbDialect.quoteIdentifier(a.target)) || [];
            state.previewFieldArr = fieldArr;
            refreshPreviewInsertSql();
            break;
        default:
            break;
    }
});

const refreshPreviewInsertSql = () => {
    let targetDbDialect = getDbDialect(state.targetDbInst.type);
    state.previewInsertSql = targetDbDialect.getBatchInsertPreviewSql(internalForm.value.targetTableName!, state.previewFieldArr, internalForm.value.duplicateStrategy!);
};

const onSelectSrcDb = async (params: DbNodeParams) => {
    params.databases = params.dbs;
    state.srcDbInst = await DbInst.getOrNewInst(params);
    sqlCompletion.register(params.id, params.db, params.dbs, params.type ?? '');
};

const onSelectTargetDb = async (params: DbNodeParams) => {
    state.targetDbInst = await DbInst.getOrNewInst(params);
    await loadDbTables(params.id, params.db);
};

const loadDbTables = async (dbId: number, db: string) => {
    let data = await dbApi.tableInfos.request({ id: dbId, db });
    state.targetTableList = data;
    if (data && data.length > 0) {
        let names = data.map((a: DbTableInfo) => a.tableName);
        if (!names.includes(internalForm.value.targetTableName ?? '')) {
            internalForm.value.targetTableName = data[0].tableName;
        }
    }
};

const handleGetSrcFields = async () => {
    const dataSql = internalForm.value.dataSql as string | undefined;
    if (!dataSql || !dataSql.trim()) {
        Msg.warning('db.noDataSqlMsg');
        return;
    }
    if (!/^select/i.test(dataSql.trim()!)) {
        Msg.warning('db.notSelectSql');
        return;
    }
    if (/;/i.test(dataSql!)) {
        Msg.warning('db.notOneSql');
        return;
    }

    const sql = getDbDialect(internalForm.value.srcDbType!).getPreviewSql(dataSql!);
    const res = await dbApi.sqlExec.request({
        id: internalForm.value.srcDbId,
        db: internalForm.value.srcDbName,
        sql,
    });

    if (res.length && !res[0].columns) {
        Msg.warning('db.notColumnSql');
        return;
    }

    let data = res[0];

    let filedMap: Record<string, { target: string; transformType: string; transformConfig: string }> = {};
    if (internalForm.value.fieldMap && internalForm.value.fieldMap.length > 0) {
        internalForm.value.fieldMap.forEach((a: any) => {
            filedMap[a.src] = { target: a.target, transformType: a.transformType || 'column', transformConfig: a.transformConfig || '' };
        });
    }

    const srcColumns = data.columns ?? [];
    internalForm.value.fieldMap = srcColumns.map((a) => {
        const existing = filedMap[a.name];
        return {
            src: a.name,
            target: existing?.target || '',
            transformType: existing?.transformType || 'column',
            transformConfig: existing?.transformConfig || '',
        };
    });

    state.previewRes = data;
};

const handleGetTargetFields = async () => {
    if (internalForm.value.targetDbName && internalForm.value.targetTableName) {
        let columns = await state.targetDbInst.loadColumns(internalForm.value.targetDbName, internalForm.value.targetTableName);
        if (columns && Array.isArray(columns)) {
            state.targetColumnList = columns;
            let names = columns.map((a) => a.columnName?.toLowerCase());

            internalForm.value.fieldMap?.forEach((a: any) => {
                if (a.target && !names.includes(a.target)) {
                    a.target = '';
                }
                if (names.includes(a.src?.toLowerCase())) {
                    let res = columns.find((col) => col.columnName?.toLowerCase() === a.src?.toLowerCase());
                    if (res) {
                        a.target = res.columnName;
                    }
                }
            });
        }
    }
};

// 提交动作：组装 fieldMap（含转换规则）后走统一提交
const btnOk = async () => {
    const reqForm: Record<string, unknown> = { ...internalForm.value };
    // 将字段映射（含转换类型和配置）序列化为 JSON
    reqForm.fieldMap = JSON.stringify(internalForm.value.fieldMap);
    await saveExec(reqForm);
    emit('val-change', internalForm.value);
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};

const handleDuplicateStrategy = () => {
    refreshPreviewInsertSql();
};
</script>
<style lang="scss">
.sync-task-edit {
    .el-select {
        width: 100%;
    }
    .task-sql {
        width: 100%;
    }
}
</style>
