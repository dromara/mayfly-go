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
                </el-table>
            </template>

            <!-- SQL 预览（只读，绑定预览状态而非表单） -->
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
                                    case sqlPreviewTab:
                                        tabActiveName = fieldTab;
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
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import { DbInst, registerDbCompletionItemProvider } from '@/views/ops/db/db';
import { compatibleDuplicateStrategy, DbType, getDbDialect } from '@/views/ops/db/dialect';
import { dbSyncApi } from '@/views/ops/db/sync/api';
import { DbDataSyncDuplicateStrategyEnum } from '@/views/ops/db/sync/enums';
import { computed, reactive, ref, useTemplateRef, watch, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import type { ColumnMetadata, DataSyncTask, Db, DbTableInfo } from '@/views/ops/db/types';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<DataSyncTask | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown>; submitting: boolean; submit: () => Promise<void> }>('drawerRef');

const basicTab = 'basic';
const fieldTab = 'field';
const sqlPreviewTab = 'sqlPreview';

/** 目标表下拉选项（从已加载的表列表生成） */
const targetTableOptions = () =>
    Promise.resolve(state.targetTableList.map((item) => ({ value: item.tableName, label: item.tableName + (item.tableComment && '-' + item.tableComment) })));

/** 表单声明（AutoFormTab[] 向导式三步：基础信息 / 字段映射 / SQL 预览；复杂控件走插槽） */
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
            { prop: 'srcDbId', label: 'db.srcDb', type: 'custom', required: true },
            { prop: 'targetDbId', label: 'db.targetDb', type: 'custom', required: true },
            { prop: 'dataSql', label: 'db.srcDataSql', type: 'monaco', required: true, props: { language: 'sql', height: '200px' } },
            { prop: 'targetTableName', label: 'db.targetDbTable', type: 'select', required: true, span: 12, options: targetTableOptions },
            { prop: 'pageSize', label: 'db.pageSize', type: 'number', required: true, span: 12, placeholder: 'db.pageSizePlaceholder' },
            { prop: 'updField', label: 'db.updateField', tooltip: 'db.updateFieldTips', placeholder: 'db.updateFiledPlaceholder', span: 12 },
            { prop: 'updFieldVal', label: 'db.updateFieldValue', tooltip: 'db.updateFieldValueTips', placeholder: 'db.updateFieldValuePlaceholder', span: 12 },
            { prop: 'updFieldSrc', label: 'db.fieldValueSrc', tooltip: 'db.fieldValueSrcTips', placeholder: 'db.fieldValueSrcPlaceholder', span: 12 },
        ],
    },
    {
        name: fieldTab,
        label: 'db.fieldMap',
        disabled: () => !baseFieldCompleted.value,
        items: [{ prop: 'fieldMap', label: 'db.fieldMap', type: 'custom', required: true }],
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
                when: (f) => compatibleDuplicateStrategy(f.targetDbType!),
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
    fieldMap?: { src: string; target: string }[];
    status?: 1 | 2;
    duplicateStrategy?: -1 | 1 | 2;
};

const basicFormData = {
    srcDbId: -1,
    targetDbId: -1,
    dataSql: 'select * from',
    pageSize: 1000,
    updField: '',
    updFieldVal: '0',
    fieldMap: [{ src: 'a', target: 'b' }],
    status: 1,
    duplicateStrategy: -1,
} as FormData;

/** 传给 AutoFormDrawer 的回填数据：新建态用默认值；编辑态任务实体经 @opened 异步加载后填充 */
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

/** 抽屉打开后暂存的内部表单引用（向导切换、SQL 预览与提交均基于它） */
const internalForm = ref<AutoFormData>({});

// 基础字段信息是否填写完整
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
    // 原始任务实体(fieldMap 为 JSON 字符串)转换为表单结构(fieldMap 随后解析为数组)
    const formData = data as unknown as FormData;
    if (!formData.duplicateStrategy) {
        formData.duplicateStrategy = -1;
    }
    try {
        formData.fieldMap = JSON.parse(data.fieldMap);
    } catch (e) {
        formData.fieldMap = [];
    }
    Object.assign(form, formData);
    let { srcDbId, srcDbName, targetDbId } = formData;

    //  初始化src数据源
    if (srcDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: srcDbId });
        const db = dbInfoRes.list[0] as Db & { databases?: string[] };
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];
        state.srcDbInst = await DbInst.getOrNewInst(db);
        form.srcDbType = state.srcDbInst.type;
        form.srcInstName = db.name;
    }

    //  初始化target数据源
    if (targetDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: targetDbId });
        const db = dbInfoRes.list[0] as Db & { databases?: string[] };
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];
        state.targetDbInst = await DbInst.getOrNewInst(db);
        form.targetDbType = state.targetDbInst.type;
        form.targetInstName = db.name;
    }

    if (targetDbId && formData.targetDbName) {
        await loadDbTables(targetDbId, formData.targetDbName);
    }

    // 注册sql代码提示
    if (srcDbId && srcDbName) {
        registerDbCompletionItemProvider(srcDbId, srcDbName, state.srcDbInst.databases, state.srcDbInst.type);
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

            // 判断sql是否以where .*结尾
            let hasCondition = /where/i.test(internalForm.value.dataSql!);
            state.previewDataSql = `${internalForm.value.dataSql?.trim() || t('db.noDataSqlMsg')} \n ${hasCondition ? 'and' : 'where'} ${updField} > '${internalForm.value.updFieldVal || ''}'`;

            // 检查字段映射中是否存在重复的目标字段
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

interface DbSelectParams {
    id: number;
    db: string;
    dbs: string[];
    type: string;
    databases?: string[];
    name?: string;
}

const onSelectSrcDb = async (params: DbSelectParams) => {
    //  初始化数据源
    params.databases = params.dbs; // 数据源里需要这个值
    state.srcDbInst = await DbInst.getOrNewInst(params);
    registerDbCompletionItemProvider(params.id, params.db, params.dbs, params.type);
};

const onSelectTargetDb = async (params: DbSelectParams) => {
    state.targetDbInst = await DbInst.getOrNewInst(params);
    await loadDbTables(params.id, params.db);
};

const loadDbTables = async (dbId: number, db: string) => {
    // 加载db下的表
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
    // 执行sql，获取字段信息
    const dataSql = internalForm.value.dataSql as string | undefined;
    if (!dataSql || !dataSql.trim()) {
        Msg.warning('db.noDataSqlMsg');
        return;
    }

    // 判断sql是否是查询语句
    if (!/^select/i.test(dataSql.trim()!)) {
        Msg.warning('db.notSelectSql');
        return;
    }

    // 判断是否有多条sql
    if (/;/i.test(dataSql!)) {
        Msg.warning('db.notOneSql');
        return;
    }

    // 执行sql
    let sql: string;

    if (internalForm.value.srcDbType === DbType.mssql) {
        // mssql的分页语法不一样
        let top1 = `select top 1`;
        sql = `${top1} * from (${dataSql}) a`;
    } else if (internalForm.value.srcDbType === DbType.oracle) {
        // oracle的分页关键字不一样
        let hasCondition = /where/i.test(dataSql!);
        sql = `${dataSql} ${hasCondition ? 'and' : 'where'} rownum <= 1`;
    } else {
        sql = `${dataSql} limit 1`;
    }

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

    let filedMap: Record<string, string> = {};
    if (internalForm.value.fieldMap && internalForm.value.fieldMap.length > 0) {
        internalForm.value.fieldMap.forEach((a: { src: string; target: string }) => {
            filedMap[a.src] = a.target;
        });
    }

    const srcColumns = data.columns ?? [];

    internalForm.value.fieldMap = srcColumns.map((a) => ({ src: a.name, target: filedMap[a.name] || '' }));

    state.previewRes = data;
};

const handleGetTargetFields = async () => {
    // 查询目标表下的字段信息
    if (internalForm.value.targetDbName && internalForm.value.targetTableName) {
        let columns = await state.targetDbInst.loadColumns(internalForm.value.targetDbName, internalForm.value.targetTableName);
        if (columns && Array.isArray(columns)) {
            state.targetColumnList = columns;
            // 过滤目标字段，不存在的字段值设置为空
            let names = columns.map((a) => a.columnName?.toLowerCase());

            internalForm.value.fieldMap?.forEach((a: { src: string; target: string }) => {
                if (a.target && !names.includes(a.target)) {
                    a.target = '';
                }
                // 优先设置字段名和src一样的值
                if (names.includes(a.src?.toLowerCase())) {
                    // 从columns中取出
                    let res = columns.find((col) => col.columnName?.toLowerCase() === a.src?.toLowerCase());
                    if (res) {
                        a.target = res.columnName;
                    }
                }
            });
        }
    }
};

// confirmApi 提交动作：组装 fieldMap 后走统一提交；成功提示与关闭抽屉由组件内置逻辑处理
const btnOk = async () => {
    const reqForm: Record<string, unknown> = { ...internalForm.value };
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
