<template>
    <div class="sync-task-edit">
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="45%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="dbForm" :rules="rules" label-position="top" label-width="auto">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane :label="$t('common.basic')" :name="basicTab">
                        <el-row :gutter="10">
                            <el-col :span="12">
                                <el-form-item prop="taskName" :label="$t('db.taskName')" required>
                                    <el-input v-model.trim="form.taskName" auto-complete="off" />
                                </el-form-item>
                            </el-col>

                            <el-col :span="12">
                                <el-form-item prop="taskCron" label="cron" required>
                                    <CrontabInput v-model="form.taskCron" />
                                </el-form-item>
                            </el-col>
                        </el-row>

                        <el-form-item prop="status" :label="$t('common.status')" label-position="left" label-width="60" required>
                            <el-switch
                                v-model="form.status"
                                inline-prompt
                                :active-text="$t('common.enable')"
                                :inactive-text="$t('common.disable')"
                                :active-value="1"
                                :inactive-value="-1"
                            />
                        </el-form-item>

                        <el-form-item prop="srcDbId" :label="$t('db.srcDb')" required>
                            <db-select-tree
                                v-model:db-id="form.srcDbId"
                                v-model:inst-name="form.srcInstName"
                                v-model:db-name="form.srcDbName"
                                v-model:tag-path="form.srcTagPath"
                                v-model:db-type="form.srcDbType"
                                @select-db="onSelectSrcDb"
                            />
                        </el-form-item>

                        <el-form-item prop="targetDbId" :label="$t('db.targetDb')" required>
                            <db-select-tree
                                v-model:db-id="form.targetDbId"
                                v-model:inst-name="form.targetInstName"
                                v-model:db-name="form.targetDbName"
                                v-model:tag-path="form.targetTagPath"
                                v-model:db-type="form.targetDbType"
                                @select-db="onSelectTargetDb"
                            />
                        </el-form-item>

                        <el-form-item prop="dataSql" :label="$t('db.srcDataSql')" required>
                            <monaco-editor height="200px" class="task-sql" language="sql" v-model="form.dataSql" />
                        </el-form-item>

                        <el-row :gutter="10">
                            <el-col :span="12">
                                <el-form-item prop="targetTableName" :label="$t('db.targetDbTable')" required>
                                    <el-select v-model="form.targetTableName" filterable>
                                        <el-option
                                            v-for="item in state.targetTableList"
                                            :key="item.tableName"
                                            :label="item.tableName + (item.tableComment && '-' + item.tableComment)"
                                            :value="item.tableName"
                                        />
                                    </el-select>
                                </el-form-item>
                            </el-col>

                            <el-col :span="12">
                                <el-form-item prop="pageSize" :label="$t('db.pageSize')" required>
                                    <el-input type="number" v-model.number="form.pageSize" :placeholder="$t('db.pageSizePlaceholder')" auto-complete="off" />
                                </el-form-item>
                            </el-col>
                        </el-row>

                        <el-row :gutter="10">
                            <el-col :span="12">
                                <FormItemTooltip :label="$t('db.updateField')" prop="updField" :tooltip="$t('db.updateFieldTips')">
                                    <el-input v-model.trim="form.updField" :placeholder="$t('db.updateFiledPlaceholder')" auto-complete="off" />
                                </FormItemTooltip>
                            </el-col>

                            <el-col :span="12">
                                <FormItemTooltip :label="$t('db.updateFieldValue')" prop="updFieldVal" :tooltip="$t('db.updateFieldValueTips')">
                                    <el-input v-model.trim="form.updFieldVal" :placeholder="$t('db.updateFieldValuePlaceholder')" auto-complete="off" />
                                </FormItemTooltip>
                            </el-col>
                        </el-row>

                        <el-row :gutter="10">
                            <el-col :span="12">
                                <FormItemTooltip :label="$t('db.fieldValueSrc')" prop="updFieldSrc" :tooltip="$t('db.fieldValueSrcTips')">
                                    <el-input v-model.trim="form.updFieldSrc" :placeholder="$t('db.fieldValueSrcPlaceholder')" auto-complete="off" />
                                </FormItemTooltip>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                    <el-tab-pane :label="$t('db.fieldMap')" :name="fieldTab" :disabled="!baseFieldCompleted">
                        <el-form-item prop="fieldMap" :label="$t('db.fieldMap')" required>
                            <el-table :data="form.fieldMap" :max-height="fieldMapTableHeight">
                                <el-table-column prop="src" :label="$t('db.srcField')" :width="200"></el-table-column>
                                <el-table-column prop="target" :label="$t('db.targetField')">
                                    <template #default="scope">
                                        <el-select v-model="scope.row.target" allow-create filterable>
                                            <template #label="{ label, value }">
                                                <div class="flex justify-between">
                                                    <el-text tag="b">{{ value }}</el-text>
                                                    <el-text size="small">{{ label }}</el-text>
                                                </div>
                                            </template>

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
                        </el-form-item>
                    </el-tab-pane>

                    <el-tab-pane :label="$t('db.sqlPreview')" :name="sqlPreviewTab" :disabled="!baseFieldCompleted">
                        <el-form-item prop="isReplace" v-if="compatibleDuplicateStrategy(form.targetDbType!)" :label="$t('db.keyDuplicateStrategy')">
                            <EnumSelect :enums="DbDataSyncDuplicateStrategyEnum" v-model="form.duplicateStrategy" @change="handleDuplicateStrategy" />
                        </el-form-item>
                        <el-form-item prop="fieldMap" :label="$t('db.selectSql')">
                            <el-input type="textarea" v-model="state.previewDataSql" readonly :rows="10" />
                        </el-form-item>
                        <el-form-item prop="fieldMap" :label="$t('db.insertSql')">
                            <el-input type="textarea" v-model="state.previewInsertSql" readonly :rows="10" />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
            </el-form>

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
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>

        <!-- <el-dialog
            :title="title"
            v-model="dialogVisible"
            :before-close="cancel"
            :close-on-click-modal="false"
            :close-on-press-escape="false"
            :destroy-on-close="true"
            width="850px"
        >
        </el-dialog> -->
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, toRefs, watch } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { DbInst, registerDbCompletionItemProvider } from '@/views/ops/db/db';
import { compatibleDuplicateStrategy, DbType, getDbDialect } from '@/views/ops/db/dialect';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { DbDataSyncDuplicateStrategyEnum } from './enums';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import FormItemTooltip from '@/components/form/FormItemTooltip.vue';
import { Rules } from '@/common/rule';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const rules = {
    taskName: [Rules.requiredInput('db.taskName')],
    taskCron: [Rules.requiredInput('cron')],
};

const dbForm: any = ref(null);

const basicTab = 'basic';
const fieldTab = 'field';
const sqlPreviewTab = 'sqlPreview';

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

const state = reactive({
    tabActiveName: 'basic',
    form: basicFormData,
    submitForm: {} as any,
    srcTableFields: [] as string[],
    targetTableList: [] as { tableName: string; tableComment: string }[],
    targetColumnList: [] as any[],
    srcDbInst: {} as DbInst,
    targetDbInst: {} as DbInst,
    previewRes: {} as any,
    previewDataSql: '',
    previewInsertSql: '',
    previewFieldArr: [] as string[],
    fieldMapTableHeight: window.innerHeight - 50,
});

const { tabActiveName, form, submitForm, fieldMapTableHeight } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveExec } = dbApi.saveDatasyncTask.useApi(submitForm);

// 基础字段信息是否填写完整
const baseFieldCompleted = computed(() => {
    return state.form.srcDbId && state.form.srcDbName && state.form.targetDbId && state.form.targetDbName && state.form.targetTableName;
});

watch(dialogVisible, async (newValue: boolean) => {
    if (!newValue) {
        return;
    }
    state.tabActiveName = 'basic';
    const propsData = props.data as any;
    if (!propsData?.id) {
        let d = { taskCron: '' } as FormData;
        Object.assign(d, basicFormData);
        state.form = d;
        return;
    }

    let data = await dbApi.getDatasyncTask.request({ taskId: propsData?.id });
    state.form = data;
    if (!state.form.duplicateStrategy) {
        state.form.duplicateStrategy = -1;
    }
    try {
        state.form.fieldMap = JSON.parse(data.fieldMap);
    } catch (e) {
        state.form.fieldMap = [];
    }
    let { srcDbId, srcDbName, targetDbId } = state.form;

    //  初始化src数据源
    if (srcDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: srcDbId });
        const db = dbInfoRes.list[0];
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];
        state.srcDbInst = await DbInst.getOrNewInst(db);
        state.form.srcDbType = state.srcDbInst.type;
        state.form.srcInstName = db.name;
    }

    //  初始化target数据源
    if (targetDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: targetDbId });
        const db = dbInfoRes.list[0];
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];
        state.targetDbInst = await DbInst.getOrNewInst(db);
        state.form.targetDbType = state.targetDbInst.type;
        state.form.targetInstName = db.name;
    }

    if (targetDbId && state.form.targetDbName) {
        await loadDbTables(targetDbId, state.form.targetDbName);
    }

    // 注册sql代码提示
    if (srcDbId && srcDbName) {
        registerDbCompletionItemProvider(srcDbId, srcDbName, state.srcDbInst.databases, state.srcDbInst.type);
    }
});

watch(tabActiveName, async (newValue: string) => {
    switch (newValue) {
        case fieldTab:
            await handleGetSrcFields();
            await handleGetTargetFields();
            break;
        case sqlPreviewTab:
            let targetDbDialect = getDbDialect(state.targetDbInst.type);
            let updField = state.form.updField!;

            // 判断sql是否以where .*结尾
            let hasCondition = /where/i.test(state.form.dataSql!);
            state.previewDataSql = `${state.form.dataSql?.trim() || t('db.noDataSqlMsg')} \n ${hasCondition ? 'and' : 'where'} ${updField} > '${state.form.updFieldVal || ''}'`;

            // 检查字段映射中是否存在重复的目标字段
            let fields = new Set();
            state.form.fieldMap?.map((a) => {
                if (a.target) {
                    fields.add(a.target);
                }
            });
            if (fields.size < (state.form.fieldMap?.length || 0)) {
                ElMessage.warning(t('db.fieldMapError'));
                state.previewInsertSql = '';
                return;
            }

            let fieldArr = state.form.fieldMap?.map((a: any) => targetDbDialect.quoteIdentifier(a.target)) || [];
            state.previewFieldArr = fieldArr;
            refreshPreviewInsertSql();
            break;
        default:
            break;
    }
});

const refreshPreviewInsertSql = () => {
    let targetDbDialect = getDbDialect(state.targetDbInst.type);
    state.previewInsertSql = targetDbDialect.getBatchInsertPreviewSql(state.form.targetTableName!, state.previewFieldArr, state.form.duplicateStrategy!);
};

const onSelectSrcDb = async (params: any) => {
    //  初始化数据源
    params.databases = params.dbs; // 数据源里需要这个值
    console.log(params.dbs);
    state.srcDbInst = await DbInst.getOrNewInst(params);
    registerDbCompletionItemProvider(params.id, params.db, params.dbs, params.type);
};

const onSelectTargetDb = async (params: any) => {
    state.targetDbInst = await DbInst.getOrNewInst(params);
    await loadDbTables(params.id, params.db);
};

const loadDbTables = async (dbId: number, db: string) => {
    // 加载db下的表
    let data = await dbApi.tableInfos.request({ id: dbId, db });
    state.targetTableList = data;
    if (data && data.length > 0) {
        let names = data.map((a: any) => a.tableName);
        if (!names.includes(state.form.targetTableName)) {
            state.form.targetTableName = data[0].tableName;
        }
    }
};

const handleGetSrcFields = async () => {
    // 执行sql，获取字段信息
    if (!state.form.dataSql || !state.form.dataSql.trim()) {
        ElMessage.warning(t('db.noDataSqlMsg'));
        return;
    }

    // 判断sql是否是查询语句
    if (!/^select/i.test(state.form.dataSql.trim()!)) {
        ElMessage.warning(t('db.notSelectSql'));
        return;
    }

    // 判断是否有多条sql
    if (/;/i.test(state.form.dataSql!)) {
        ElMessage.warning(t('db.notOneSql'));
        return;
    }

    // 执行sql
    let sql: string;

    if (state.form.srcDbType === DbType.mssql) {
        // mssql的分页语法不一样
        let top1 = `select top 1`;
        sql = `${top1} * from (${state.form.dataSql}) a`;
    } else if (state.form.srcDbType === DbType.oracle) {
        // oracle的分页关键字不一样
        let hasCondition = /where/i.test(state.form.dataSql!);
        sql = `${state.form.dataSql} ${hasCondition ? 'and' : 'where'} rownum <= 1`;
    } else {
        sql = `${state.form.dataSql} limit 1`;
    }

    const res = await dbApi.sqlExec.request({
        id: state.form.srcDbId,
        db: state.form.srcDbName,
        sql,
    });

    if (res.length && !res[0].columns) {
        ElMessage.warning(t('db.notColumnSql'));
        return;
    }

    let data = res[0];

    let filedMap: any = {};
    if (state.form.fieldMap && state.form.fieldMap.length > 0) {
        state.form.fieldMap.forEach((a: any) => {
            filedMap[a.src] = a.target;
        });
    }

    state.srcTableFields = data.columns.map((a: any) => a.name);

    state.form.fieldMap = data.columns.map((a: any) => ({ src: a.name, target: filedMap[a.name] || '' }));

    state.previewRes = data;
};

const handleGetTargetFields = async () => {
    // 查询目标表下的字段信息
    if (state.form.targetDbName && state.form.targetTableName) {
        let columns = await state.targetDbInst.loadColumns(state.form.targetDbName, state.form.targetTableName);
        if (columns && Array.isArray(columns)) {
            state.targetColumnList = columns;
            // 过滤目标字段，不存在的字段值设置为空
            let names = columns.map((a) => a.columnName?.toLowerCase());

            state.form.fieldMap?.forEach((a) => {
                if (a.target && !names.includes(a.target)) {
                    a.target = '';
                }
                // 优先设置字段名和src一样的值
                if (names.includes(a.src?.toLowerCase())) {
                    // 从columns中取出
                    let res = columns.find((col: any) => col.columnName?.toLowerCase() === a.src?.toLowerCase());
                    if (res) {
                        a.target = res.columnName;
                    }
                }
            });
        }
    }
};

const getReqForm = async () => {
    return { ...state.form };
};

const btnOk = async () => {
    await useI18nFormValidate(dbForm);
    // 处理一些数字类型
    state.submitForm = await getReqForm();
    state.submitForm.fieldMap = JSON.stringify(state.form.fieldMap);
    await saveExec();
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    cancel();
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
    state.form = basicFormData;
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
