<template>
    <div class="sync-task-edit">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :before-close="cancel"
            :close-on-click-modal="false"
            :close-on-press-escape="false"
            :destroy-on-close="true"
            width="700px"
        >
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-tabs v-model="tabActiveName" style="height: 450px">
                    <el-tab-pane label="基本信息" name="basic">
                        <el-form-item prop="taskName" label="任务名" required>
                            <el-input v-model.trim="form.taskName" placeholder="请输入数据库别名" auto-complete="off" />
                        </el-form-item>
                        <el-form-item prop="taskCron" label="cron" required>
                            <el-input v-model="form.taskCron" placeholder="只支持5位表达式,不支持秒级.如 0/2 * * * * 表示每两分钟执行" auto-complete="off" />
                        </el-form-item>
                        <el-form-item prop="pageSize" label="分页大小" required>
                            <el-input-number v-model.trim="form.pageSize" placeholder="同步数据时查询的每页数据大小" auto-complete="off" size="small" />
                        </el-form-item>
                        <el-form-item prop="updField" label="更新字段" required>
                            <el-input v-model.trim="form.updField" placeholder="查询数据源的时候会带上这个字段当前最大值" auto-complete="off" />
                        </el-form-item>
                        <el-form-item prop="updFieldVal" label="更新值">
                            <el-input v-model.trim="form.updFieldVal" placeholder="更新字段当前最大值" auto-complete="off" />
                        </el-form-item>
                        <el-form-item prop="status" label="状态" required>
                            <el-switch v-model="form.status" inline-prompt active-text="启用" inactive-text="禁用" :active-value="1" :inactive-value="-1" />
                        </el-form-item>
                    </el-tab-pane>
                    <el-tab-pane label="源数据库配置" name="srcDb">
                        <el-form-item prop="srcDbId" label="数据源" required>
                            <db-select-tree
                                v-model:db-id="form.srcDbId"
                                v-model:db-name="form.srcDbName"
                                v-model:tag-path="form.srcTagPath"
                                @select-db="onSelectSrcDb"
                            />
                        </el-form-item>
                        <el-form-item prop="dataSql" label="数据sql" required>
                            <monaco-editor height="200px" class="task-sql" language="sql" v-model="form.dataSql" />
                        </el-form-item>
                    </el-tab-pane>

                    <el-tab-pane label="目标数据库配置" name="targetDb">
                        <el-form-item prop="targetDbId" label="数据源" required>
                            <db-select-tree
                                v-model:db-id="form.targetDbId"
                                v-model:db-name="form.targetDbName"
                                v-model:tag-path="form.targetTagPath"
                                @select-db="onSelectTargetDb"
                            />
                        </el-form-item>

                        <el-form-item prop="targetTableName" label="目标表" required>
                            <el-select v-model="form.targetTableName" filterable placeholder="请选择目标数据库表">
                                <el-option
                                    v-for="item in state.targetTableList"
                                    :key="item.tableName"
                                    :label="item.tableName + (item.tableComment && '-' + item.tableComment)"
                                    :value="item.tableName"
                                />
                            </el-select>
                        </el-form-item>
                    </el-tab-pane>
                    <el-tab-pane label="字段映射" name="field">
                        <el-form-item prop="fieldMap" label="字段映射" required>
                            <el-table :data="form.fieldMap" :max-height="400" size="small">
                                <el-table-column prop="src" label="源字段" :width="200" />
                                <el-table-column prop="target" label="目标字段">
                                    <template #default="scope">
                                        <el-select v-model="scope.row.target">
                                            <el-option
                                                v-for="item in state.targetColumnList"
                                                :key="item.columnName"
                                                :label="item.columnName + ` ${item.columnType}` + (item.columnComment && ' - ' + item.columnComment)"
                                                :value="item.columnName"
                                            />
                                        </el-select>
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-form-item>
                    </el-tab-pane>
                    <el-tab-pane label="sql预览" name="sqlPreview">
                        <el-form-item prop="fieldMap" label="查询sql">
                            <el-input type="textarea" v-model="state.previewDataSql" readonly :input-style="{ height: '190px' }" />
                        </el-form-item>
                        <el-form-item prop="fieldMap" label="插入sql">
                            <el-input type="textarea" v-model="state.previewInsertSql" readonly :input-style="{ height: '190px' }" />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, watch } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { DbInst, registerDbCompletionItemProvider } from '@/views/ops/db/db';
import { getDbDialect } from '@/views/ops/db/dialect';

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
    taskName: [
        {
            required: true,
            message: '请输入任务名',
            trigger: ['change', 'blur'],
        },
    ],
    taskCron: [
        {
            required: true,
            message: '请输入任务cron表达式',
            trigger: ['change', 'blur'],
        },
    ],
};

const dbForm: any = ref(null);

type FormData = {
    id?: number;
    taskName?: string;
    taskCron?: string;
    srcDbId?: number;
    srcDbName?: string;
    srcTagPath?: string;
    targetDbId?: number;
    targetDbName?: string;
    targetTagPath?: string;
    targetTableName?: string;
    dataSql?: string;
    pageSize?: number;
    updField?: string;
    updFieldVal?: string;
    fieldMap?: { src: string; target: string }[];
    status?: 1 | 2;
};

const basicFormData = {
    srcDbId: -1,
    targetDbId: -1,
    dataSql: 'select * from',
    pageSize: 1000,
    updField: 'id',
    updFieldVal: '0',
    fieldMap: [{ src: 'a', target: 'b' }],
    status: 1,
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
});

const onSelectSrcDb = async (params: any) => {
    //  初始化数据源
    params.databases = params.dbs; // 数据源里需要这个值
    state.srcDbInst = DbInst.getOrNewInst(params);
    registerDbCompletionItemProvider(params.id, params.db, params.dbs, params.type);
};

const onSelectTargetDb = async (params: any) => {
    state.targetDbInst = DbInst.getOrNewInst(params);
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

const { tabActiveName, form, submitForm } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveExec } = dbApi.saveDatasyncTask.useApi(submitForm);

watch(dialogVisible, async (newValue: boolean) => {
    if (!newValue) {
        return;
    }
    state.tabActiveName = 'basic';
    const propsData = props.data as any;
    if (propsData?.id) {
        let data = await dbApi.getDatasyncTask.request({ taskId: propsData?.id });
        state.form = data;
        try {
            state.form.fieldMap = JSON.parse(data.fieldMap);
        } catch (e) {
            state.form.fieldMap = [];
        }
        let { srcDbId, srcTagPath, srcDbName, targetTagPath, targetDbId } = state.form;

        //  初始化src数据源
        if (srcTagPath && srcDbId) {
            // 通过tagPath查询实例列表
            const dbInfoRes = await dbApi.dbs.request({ tagPath: srcTagPath });
            dbInfoRes.list.forEach((a: any) => {
                if (a.id === srcDbId) {
                    // 初始化实例
                    a.databases = a.database?.split(' ').sort() || [];
                    state.srcDbInst = DbInst.getOrNewInst(a);
                }
            });
        }

        //  初始化target数据源
        if (targetTagPath && targetDbId) {
            // 通过tagPath查询实例列表
            const dbInfoRes = await dbApi.dbs.request({ tagPath: targetTagPath });
            dbInfoRes.list.forEach((a: any) => {
                if (a.id === targetDbId) {
                    // 初始化实例
                    a.databases = a.database?.split(' ').sort() || [];
                    state.targetDbInst = DbInst.getOrNewInst(a);
                }
            });
        }

        // 注册sql代码提示
        if (srcDbId && srcDbName) {
            registerDbCompletionItemProvider(srcDbId, srcDbName, state.srcDbInst.databases, state.srcDbInst.type);
        }
    } else {
        state.form = basicFormData;
    }
});

watch(tabActiveName, async (newValue: string) => {
    switch (newValue) {
        case 'field':
            await handleGetSrcFields();
            await handleGetTargetFields();
            break;
        case 'targetDb':
            await handleGetSrcFields();
            await handleGetTargetFields();
            if (state.form.targetDbId && state.form.targetDbName) {
                await loadDbTables(state.form.targetDbId, state.form.targetDbName);
            }
            break;
        case 'sqlPreview':
            let srcDbDialect = getDbDialect(state.srcDbInst.type);
            let targetDbDialect = getDbDialect(state.targetDbInst.type);

            let updField = srcDbDialect.wrapName(state.form.updField!);
            state.previewDataSql = `SELECT * FROM (\n ${state.form.dataSql?.trim() || '请输入数据sql'} \n ) t \n where ${updField} > '${
                state.form.updFieldVal || ''
            }'`;

            // 检查字段映射中是否存在重复的目标字段
            let fields = new Set();
            state.form.fieldMap?.map((a) => {
                if (a.target) {
                    fields.add(a.target);
                }
            });
            if (fields.size < (state.form.fieldMap?.length || 0)) {
                ElMessage.warning('字段映射中存在重复的目标字段，请检查');
                return;
            }

            let fieldArr = state.form.fieldMap?.map((a: any) => targetDbDialect.wrapName(a.target)) || [];
            let placeholder = '?'.repeat(fieldArr.length).split('').join(',');

            state.previewInsertSql = ` insert into ${targetDbDialect.wrapName(state.form.targetTableName!)}(${fieldArr.join(',')}) values (${placeholder});`;
            break;
        default:
            break;
    }
});

const handleGetSrcFields = async () => {
    // 执行sql，获取字段信息
    if (!state.form.dataSql || !state.form.dataSql.trim()) {
        ElMessage.warning('请输入数据源sql');
        return;
    }

    // 判断sql是否是查询语句
    if (!/^select/i.test(state.form.dataSql!)) {
        let msg = 'sql语句错误，请输入查询语句';
        ElMessage.warning(msg);
        return;
    }

    // 判断是否有多条sql
    if (/;/i.test(state.form.dataSql!)) {
        let msg = 'sql语句错误，请输入单条查询语句';
        ElMessage.warning(msg);
        return;
    }

    // 执行sql
    const res = await dbApi.sqlExec.request({
        id: state.form.srcDbId,
        db: state.form.srcDbName,
        sql: state.form.dataSql.trim() + ' limit 1',
    });

    if (!res.columns) {
        ElMessage.warning('没有查询到字段，请检查sql');
        return;
    }

    let filedMap = {};
    if (state.form.fieldMap && state.form.fieldMap.length > 0) {
        state.form.fieldMap.forEach((a: any) => {
            filedMap[a.src] = a.target;
        });
    }

    state.srcTableFields = res.columns.map((a: any) => a.name);

    state.form.fieldMap = res.columns.map((a: any) => ({ src: a.name, target: filedMap[a.name] || '' }));

    state.previewRes = res;
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
    dbForm.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        // 正则表达式检测corn表达式正确性

        // 处理一些数字类型
        state.submitForm = await getReqForm();
        state.submitForm.fieldMap = JSON.stringify(state.form.fieldMap);
        await saveExec();
        ElMessage.success('保存成功');
        emit('val-change', state.form);
        cancel();
    });
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss">
.sync-task-edit {
    .el-select {
        width: 360px;
    }
    .task-sql {
        width: 100%;
    }
}
</style>
