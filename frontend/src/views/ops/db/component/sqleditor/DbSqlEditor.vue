<template>
    <div>
        <div>
            <div class="card !p-1 flex items-center justify-between">
                <div>
                    <el-link @click="onRunSql()" underline="never" class="ml-3.5" icon="VideoPlay"> </el-link>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="1000" class="box-item" effect="dark" content="format sql" placement="top">
                        <el-link @click="onFormatSql()" type="primary" underline="never" icon="MagicStick"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="1000" class="box-item" effect="dark" content="commit" placement="top">
                        <el-link @click="onCommit()" type="success" underline="never" icon="CircleCheck"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-upload
                        class="sql-file-exec"
                        :before-upload="beforeUpload"
                        :on-success="execSqlFileSuccess"
                        :headers="{ Authorization: token }"
                        :action="getUploadSqlFileUrl()"
                        :show-file-list="false"
                        name="file"
                        multiple
                        :limit="100"
                    >
                        <el-tooltip :show-after="1000" class="box-item" effect="dark" :content="$t('db.sqlScriptRun')" placement="top">
                            <el-link v-auth="'db:sqlscript:run'" type="success" underline="never" icon="Document"></el-link>
                        </el-tooltip>
                    </el-upload>
                </div>

                <div>
                    <el-button @click="saveSql()" type="primary" icon="document-add" plain size="small">{{ $t('db.saveSql') }}</el-button>
                </div>
            </div>
        </div>

        <el-splitter style="height: calc(100vh - 200px)" layout="vertical" @resize-end="onResizeTableHeight">
            <el-splitter-panel :size="state.editorSize" max="80%">
                <MonacoEditor ref="monacoEditorRef" class="mt-1" v-model="state.sql" language="sql" height="100%" :id="'MonacoTextarea-' + getKey()" />
            </el-splitter-panel>

            <el-splitter-panel>
                <div class="sql-exec-res !h-full">
                    <el-tabs
                        class="!h-full !w-full"
                        v-if="state.execResTabs.length > 0"
                        @tab-remove="onRemoveTab"
                        @tab-change="active"
                        v-model="state.activeTab"
                    >
                        <el-tab-pane class="!h-full" closable v-for="dt in state.execResTabs" :label="dt.id" :name="dt.id" :key="dt.id">
                            <template #label>
                                <el-popover :show-after="1000" placement="top-start" :title="$t('db.execInfo')" trigger="hover" :width="300">
                                    <template #reference>
                                        <div>
                                            <span>
                                                <span v-if="dt.loading">
                                                    <SvgIcon class="!mb-0.5 is-loading" name="Loading" color="var(--el-color-primary)" />
                                                </span>
                                                <span v-else>
                                                    <SvgIcon class="!mb-0.5" v-if="!dt.errorMsg" name="CircleCheck" color="var(--el-color-success)" />
                                                    <SvgIcon class="!mb-0.5" v-if="dt.errorMsg" name="CircleClose" color="var(--el-color-error)" />
                                                </span>
                                            </span>

                                            <span> {{ $t('db.result') }}-{{ dt.id }} </span>
                                        </div>
                                    </template>
                                    <template #default>
                                        <el-descriptions v-if="dt.sql" :column="1" size="small">
                                            <el-descriptions-item>
                                                <div style="width: 280px">
                                                    <el-text size="small" truncated :title="dt.sql"> {{ dt.sql }} </el-text>
                                                </div>
                                            </el-descriptions-item>
                                            <el-descriptions-item :label="`${$t('db.times')} :`"> {{ dt.execTime }}ms </el-descriptions-item>
                                            <el-descriptions-item :label="`${$t('db.resultSet')} :`">
                                                {{ dt.data?.length }}
                                            </el-descriptions-item>
                                        </el-descriptions>
                                    </template>
                                </el-popover>
                            </template>

                            <el-row>
                                <span v-if="dt.hasUpdatedFileds" class="mt-1">
                                    <span>
                                        <el-link type="success" underline="never" @click="submitUpdateFields(dt)"
                                            ><span style="font-size: 12px">{{ $t('common.submit') }}</span></el-link
                                        >
                                    </span>
                                    <span>
                                        <el-divider direction="vertical" border-style="dashed" />
                                        <el-link type="warning" underline="never" @click="cancelUpdateFields(dt)"
                                            ><span style="font-size: 12px">{{ $t('common.cancel') }}</span></el-link
                                        >
                                    </span>
                                </span>
                            </el-row>
                            <db-table-data
                                v-if="!dt.errorMsg"
                                :ref="(el: any) => (dt.dbTableRef = el)"
                                :db-id="dbId"
                                :db="dbName"
                                :data="dt.data"
                                :table="dt.table"
                                :columns="dt.tableColumn"
                                :loading="dt.loading"
                                :abort-fn="dt.abortFn"
                                :height="tableDataHeight"
                                :empty-text="state.tableDataEmptyText"
                                @change-updated-field="changeUpdatedField($event, dt)"
                                @data-delete="onDeleteData($event, dt)"
                            ></db-table-data>

                            <el-result v-else icon="error" :title="$t('db.execFail')" :sub-title="dt.errorMsg"> </el-result>
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </el-splitter-panel>
        </el-splitter>
    </div>
</template>

<script lang="ts" setup>
import { nextTick, onMounted, reactive, ref, toRefs, unref } from 'vue';
import { getToken } from '@/common/utils/storage';
import { notBlank } from '@/common/assert';
import { format as sqlFormatter } from 'sql-formatter';
import config from '@/common/config';
import { ElMessage, ElMessageBox } from 'element-plus';

import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import { editor } from 'monaco-editor';

import DbTableData from '@/views/ops/db/component/table/DbTableData.vue';
import { DbInst } from '../../db';
import { dbApi } from '../../api';

import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { joinClientParams } from '@/common/request';
import SvgIcon from '@/components/svgIcon/index.vue';
import { useI18n } from 'vue-i18n';
import { useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useDebounceFn, useEventListener } from '@vueuse/core';

const emits = defineEmits(['saveSqlSuccess']);

const { t } = useI18n();

const props = defineProps({
    dbId: {
        type: Number,
        required: true,
    },
    dbName: {
        type: String,
        required: true,
    },
    // sql脚本名，若有则去加载该sql内容
    sqlName: {
        type: String,
    },
});

class ExecResTab {
    id: number;

    /**
     * 当前结果集对应的sql
     */
    sql: string;

    /**
     * 响应式loading
     */
    loading: any;

    dbTableRef: any;

    abortFn: Function;

    tableColumn: any[] = [];

    data: any[] = [];

    execTime: number;

    /**
     * 当前单表操作sql关联的表信息
     */
    table: string;

    /**
     * 是否有更新字段
     */
    hasUpdatedFileds: boolean;

    errorMsg: string;

    constructor(id: number) {
        this.id = id;
    }
}

const token = getToken();
const monacoEditorRef: any = ref(null);

let monacoEditor: editor.IStandaloneCodeEditor;

const state = reactive({
    editorSize: 50, // editor高度比例
    token,
    sql: '', // 当前编辑器的sql内容s
    sqlName: '' as any, // sql模板名称
    execResTabs: [] as ExecResTab[],
    activeTab: 1,
    editorHeight: '500',
    tableDataHeight: '250px',
    tableDataEmptyText: t('db.tableDataEmptyTextTips'),
});

const { tableDataHeight } = toRefs(state);

const getNowDbInst = () => {
    return DbInst.getInst(props.dbId);
};

onMounted(async () => {
    console.log('in query mounted');

    // 第一个pane为sql editor
    onResizeTableHeight(0, [-1]);
    useEventListener(
        'resize',
        useDebounceFn(() => onResizeTableHeight(0, [-1]), 200)
    );

    // 默认新建一个结果集tab
    state.execResTabs.push(new ExecResTab(1));

    state.sqlName = props.sqlName;
    if (props.sqlName) {
        const res = await dbApi.getSql.request({ id: props.dbId, type: 1, db: props.dbName, name: props.sqlName });
        state.sql = res.sql;
    }
    nextTick(() => {
        setTimeout(() => initMonacoEditor(), 50);
    });
    await getNowDbInst().loadDbHints(props.dbName);
});

const onRemoveTab = (targetId: number) => {
    let activeTab = state.activeTab;
    const tabs = [...state.execResTabs];
    for (let i = 0; i < tabs.length; i++) {
        const tabId = tabs[i].id;
        if (tabId !== targetId) {
            continue;
        }
        const nextTab = tabs[i + 1] || tabs[i - 1];
        if (nextTab) {
            activeTab = nextTab.id;
        } else {
            activeTab = 0;
        }
        state.execResTabs.splice(i, 1);
        state.activeTab = activeTab;
    }
};

const onResizeTableHeight = (index: number, sizes: number[]) => {
    if (!sizes || sizes.length === 0) {
        return;
    }

    const vh = window.innerHeight;
    const plitpaneHeight = vh - 200;

    let editorHeight = sizes[0];
    if (editorHeight < 0 || editorHeight > plitpaneHeight - 43) {
        // 默认占50%
        editorHeight = plitpaneHeight / 2;
    }

    let tableDataHeight = plitpaneHeight - editorHeight - 43;

    state.editorSize = editorHeight;
    state.tableDataHeight = tableDataHeight + 'px';
};

const getKey = () => {
    if (props.sqlName) {
        return `${props.dbId}:${props.dbName}.${props.sqlName}`;
    }
    return props.dbId + ':' + props.dbName;
};

/**
 * 执行sql
 */
const onRunSql = async (newTab = false) => {
    // 没有选中的文本，则为全部文本
    let sql = getSql() as string;
    notBlank(sql && sql.trim(), t('db.noSelctRunSqlMsg'));
    // 去除字符串前的空格、换行等
    sql = sql.replace(/(^\s*)/g, '');

    const sqls = splitSql(sql);

    if (sqls.length == 1) {
        const oneSql = sqls[0];
        // 简单截取前十个字符
        const sqlPrefix = oneSql.slice(0, 10).toLowerCase();
        const nonQuery =
            sqlPrefix.startsWith('update') ||
            sqlPrefix.startsWith('insert') ||
            sqlPrefix.startsWith('delete') ||
            sqlPrefix.startsWith('alter') ||
            sqlPrefix.startsWith('drop') ||
            sqlPrefix.startsWith('create');
        let execRemark;
        if (nonQuery) {
            const res: any = await ElMessageBox.prompt(t('db.enterExecRemarkTips'), 'Tip', {
                confirmButtonText: t('common.confirm'),
                cancelButtonText: t('common.cancel'),
                inputErrorMessage: t('db.execRemarkPlaceholder'),
            });
            execRemark = res.value;
        }
        runSql(oneSql, execRemark, newTab);
    } else {
        let isFirst = true;
        for (let s of sqls) {
            if (isFirst) {
                isFirst = false;
                runSql(s, '', newTab);
            } else {
                runSql(s, '', true);
            }
        }
    }
};

/**
 * 执行单条sql
 *
 * @param sql 单条sql
 * @param newTab 是否新建tab
 */
const runSql = async (sql: string, remark = '', newTab = false) => {
    let execRes: ExecResTab;
    let i = 0;
    let id;
    // 新tab执行，或者tabs为0，则新建tab执行sql
    if (newTab || state.execResTabs.length == 0) {
        // 取最后一个tab的id + 1
        id = state.execResTabs.length == 0 ? 1 : state.execResTabs[state.execResTabs.length - 1].id + 1;
        execRes = new ExecResTab(id);
        state.execResTabs.push(execRes);
        i = state.execResTabs.length - 1;
    } else {
        // 不是新建tab执行，则在当前激活的tab上执行sql
        i = state.execResTabs.findIndex((x) => x.id == state.activeTab);
        execRes = state.execResTabs[i];
        if (unref(execRes.loading)) {
            ElMessage.error(t('db.currentSqlTabIsRunning'));
            return;
        }
        id = execRes.id;
    }

    state.activeTab = id;
    const startTime = new Date().getTime();
    try {
        execRes.errorMsg = '';
        execRes.sql = '';

        const { data, execute, isFetching, abort } = getNowDbInst().execSql(props.dbName, sql, remark);
        execRes.loading = isFetching;
        execRes.abortFn = abort;

        await execute();
        const colAndData: any = (data.value as any)[0];
        if (colAndData.errorMsg) {
            throw { msg: colAndData.errorMsg };
        }

        if (colAndData.res.length == 0) {
            state.tableDataEmptyText = 'No Data';
        }

        // 要实时响应，故需要用索引改变数据才生效
        state.execResTabs[i].data = colAndData.res;
        // 兼容表格字段配置
        state.execResTabs[i].tableColumn = colAndData.columns.map((x: any) => {
            return {
                columnName: x.name,
                columnType: x.type,
                show: true,
            };
        });
        cancelUpdateFields(execRes);
    } catch (e: any) {
        execRes.data = [];
        execRes.tableColumn = [];
        execRes.table = '';
        // 要实时响应，故需要用索引改变数据才生效
        state.execResTabs[i].errorMsg = e.msg;
        return;
    } finally {
        execRes.sql = sql;
        execRes.execTime = new Date().getTime() - startTime;
    }

    // 即只有以该字符串开头的sql才可修改表数据内容
    if (sql.startsWith('SELECT *') || sql.startsWith('select *') || sql.startsWith('SELECT\n  *')) {
        const tableName = sql.split(/from/i)[1];
        if (tableName) {
            const tn = tableName.trim().split(' ')[0].split('\n')[0];
            // 去除表名前后的字符`或者"
            execRes.table = tn.replace(/`/g, '').replace(/"/g, '');
        } else {
            execRes.table = '';
        }
    } else {
        execRes.table = '';
    }
};

function splitSql(sql: string) {
    let state = 'normal';
    let buffer = '';
    let result = [];
    let inString = null; // 用于记录当前字符串的引号类型（' 或 "）

    for (let i = 0; i < sql.length; i++) {
        const char = sql[i];
        const nextChar = sql[i + 1];

        if (state === 'normal') {
            if (char === '-' && nextChar === '-') {
                state = 'singleLineComment';
                i++; // 跳过下一个字符
            } else if (char === '/' && nextChar === '*') {
                state = 'multiLineComment';
                i++; // 跳过下一个字符
            } else if (char === "'" || char === '"') {
                state = 'string';
                inString = char;
                buffer += char;
            } else if (char === ';') {
                if (buffer.trim()) {
                    result.push(buffer.trim());
                }
                buffer = '';
            } else {
                buffer += char;
            }
        } else if (state === 'string') {
            buffer += char;
            if (char === '\\') {
                // 处理转义字符
                buffer += nextChar;
                i++;
            } else if (char === inString) {
                state = 'normal';
                inString = null;
            }
        } else if (state === 'singleLineComment') {
            if (char === '\n') {
                state = 'normal';
            }
        } else if (state === 'multiLineComment') {
            if (char === '*' && nextChar === '/') {
                state = 'normal';
                i++; // 跳过下一个字符
            }
        }
    }

    if (buffer.trim()) {
        result.push(buffer.trim());
    }

    return result;
}

/**
 * 获取sql，如果有鼠标选中，则返回选中内容，否则返回输入框内所有内容
 */
const getSql = () => {
    let res = '' as string | undefined;
    // 编辑器还没初始化
    if (!monacoEditor?.getModel()) {
        return res;
    }
    // 选择选中的sql
    let selection = monacoEditor.getSelection();
    if (selection) {
        res = monacoEditor.getModel()?.getValueInRange(selection);
    }

    // 整个编辑器的sql
    if (!res) {
        return monacoEditor.getModel()?.getValue();
    }
    return res;
};

const saveSql = async () => {
    const sql = monacoEditor.getModel()?.getValue();
    notBlank(sql, t('db.sqlCannotEmpty'));

    let sqlName = state.sqlName;
    if (!sqlName) {
        try {
            const input = await ElMessageBox.prompt(t('db.enterSqlScriptNameTips'), 'SQL Name', {
                confirmButtonText: t('common.confirm'),
                cancelButtonText: t('common.cancel'),
                inputPattern: /.+/,
                inputErrorMessage: t('db.enterSqlScriptNameTips'),
            });
            sqlName = input.value;
            state.sqlName = sqlName;
        } catch (e) {
            return;
        }
    }

    await dbApi.saveSql.request({ id: props.dbId, db: props.dbName, sql: sql, type: 1, name: sqlName });
    useI18nSaveSuccessMsg();
    // 保存sql脚本成功事件
    emits('saveSqlSuccess', props.dbId, props.dbName);
};

/**
 * 格式化sql
 */
const onFormatSql = () => {
    let selection = monacoEditor.getSelection();
    if (!selection) {
        return;
    }

    const formatDialect: any = getNowDbInst().getDialect().getInfo().formatSqlDialect;

    let sql = monacoEditor.getModel()?.getValueInRange(selection);
    // 有选中sql则格式化并替换选中sql, 否则格式化编辑器所有内容
    if (sql) {
        replaceSelection(sqlFormatter(sql, { language: formatDialect }), selection);
        return;
    }
    monacoEditor.getModel()?.setValue(sqlFormatter(monacoEditor.getValue(), { language: formatDialect }));
};

/**
 * 提交事务，用于没有开启自动提交事务
 */
const onCommit = () => {
    getNowDbInst().runSql(props.dbName, 'COMMIT;');
    ElMessage.success('COMMIT success');
};

/**
 * 替换选中的内容
 */
const replaceSelection = (str: string, selection: any) => {
    const model = monacoEditor.getModel();
    if (!model) {
        return;
    }
    if (!selection) {
        model.setValue(str);
        return;
    }
    const { startLineNumber, endLineNumber, startColumn, endColumn } = selection;

    const textBeforeSelection = model.getValueInRange({
        startLineNumber: 1,
        startColumn: 0,
        endLineNumber: startLineNumber,
        endColumn: startColumn,
    });

    const textAfterSelection = model.getValueInRange({
        startLineNumber: endLineNumber,
        startColumn: endColumn,
        endLineNumber: model.getLineCount(),
        endColumn: model.getLineMaxColumn(model.getLineCount()),
    });

    monacoEditor.setValue(textBeforeSelection + str + textAfterSelection);
    monacoEditor.focus();
    monacoEditor.setPosition({
        lineNumber: startLineNumber,
        column: 0,
    });
};

const beforeUpload = (file: File) => {
    ElMessage.success(t('db.scriptFileUploadRunning', { filename: file.name }));
};

// 执行sql成功
const execSqlFileSuccess = (res: any) => {
    if (res.code !== 200) {
        ElMessage.error(res.msg);
    }
};

// 获取sql文件上传执行url
const getUploadSqlFileUrl = () => {
    return `${config.baseApiUrl}/dbs/${props.dbId}/exec-sql-file?db=${props.dbName}&${joinClientParams()}`;
};

const changeUpdatedField = (updatedFields: any, dt: ExecResTab) => {
    // 如果存在要更新字段，则显示提交和取消按钮
    dt.hasUpdatedFileds = updatedFields && updatedFields.size > 0;
};

/**
 * 数据删除事件
 */
const onDeleteData = async (deleteDatas: any, dt: ExecResTab) => {
    const db = props.dbName;
    const dbInst = getNowDbInst();
    const primaryKey = await dbInst.loadTableColumn(db, dt.table);
    const primaryKeyColumnName = primaryKey.columnName;
    dt.data = dt.data.filter((d: any) => !(deleteDatas.findIndex((x: any) => x[primaryKeyColumnName] == d[primaryKeyColumnName]) != -1));
};

const submitUpdateFields = (dt: ExecResTab) => {
    dt?.dbTableRef?.submitUpdateFields();
};

const cancelUpdateFields = (dt: ExecResTab) => {
    dt?.dbTableRef?.cancelUpdateFields();
};

const initMonacoEditor = () => {
    monacoEditor = monacoEditorRef.value.getEditor();

    // 注册快捷键：ctrl + R 运行选中的sql
    monacoEditor.addAction({
        // An unique identifier of the contributed action.
        // id: 'run-sql-action' + state.ti.key,
        id: 'run-sql-action' + getKey(),
        // A label of the action that will be presented to the user.
        label: t('db.runSql'),
        // A precondition for this action.
        precondition: undefined,
        // A rule to evaluate on top of the precondition in order to dispatch the keybindings.
        keybindingContext: undefined,
        keybindings: [
            // chord
            monaco.KeyMod.chord(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyR, 0),
        ],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.5,
        // Method that will be executed when the action is triggered.
        // @param editor The editor instance is passed in as a convenience
        run: async function () {
            try {
                await onRunSql();
            } catch (e: any) {
                e.message && ElMessage.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + R 运行选中的sql
    monacoEditor.addAction({
        // An unique identifier of the contributed action.
        // id: 'run-sql-action' + state.ti.key,
        id: 'run-sql-action-on-newtab' + getKey(),
        // A label of the action that will be presented to the user.
        label: t('db.newTabRunSql'),
        // A precondition for this action.
        precondition: undefined,
        // A rule to evaluate on top of the precondition in order to dispatch the keybindings.
        keybindingContext: undefined,
        keybindings: [
            // chord
            monaco.KeyMod.chord(monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyR, 0),
        ],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.6,
        // Method that will be executed when the action is triggered.
        // @param editor The editor instance is passed in as a convenience
        run: async function () {
            try {
                await onRunSql(true);
            } catch (e: any) {
                e.message && ElMessage.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + shift + f 格式化sql
    monacoEditor.addAction({
        // An unique identifier of the contributed action.
        id: 'format-sql-action' + getKey(),
        // A label of the action that will be presented to the user.
        label: t('db.formatSql'),
        // A precondition for this action.
        precondition: undefined,
        // A rule to evaluate on top of the precondition in order to dispatch the keybindings.
        keybindingContext: undefined,
        keybindings: [
            // chord
            monaco.KeyMod.chord(monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyF, 0),
        ],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 2,
        // Method that will be executed when the action is triggered.
        // @param editor The editor instance is passed in as a convenience
        run: async function () {
            try {
                await onFormatSql();
            } catch (e: any) {
                e.message && ElMessage.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + shift + f 格式化sql
    monacoEditor.addAction({
        // An unique identifier of the contributed action.
        id: 'save-sql-action' + getKey(),
        // A label of the action that will be presented to the user.
        label: t('db.saveSql'),
        // A precondition for this action.
        precondition: undefined,
        // A rule to evaluate on top of the precondition in order to dispatch the keybindings.
        keybindingContext: undefined,
        keybindings: [
            // chord
            monaco.KeyMod.chord(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, 0),
        ],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 3,
        // Method that will be executed when the action is triggered.
        // @param editor The editor instance is passed in as a convenience
        run: async function () {
            await saveSql();
        },
    });
};

const active = () => {
    const resTab = state.execResTabs[state.activeTab - 1];
    if (!resTab || !resTab.dbTableRef) {
        return;
    }

    resTab.dbTableRef?.active();
};

defineExpose({
    active,
});
</script>

<style lang="scss">
.sql-file-exec {
    display: inline-flex;
    flex-direction: row;
    align-items: center;
    justify-content: center;
    vertical-align: middle;
    position: relative;
    text-decoration: none;
}

.editor-move-resize {
    cursor: n-resize;
    height: 3px;
    text-align: center;
}

.sql-exec-res {
    .el-tabs__header {
        margin: 0 0 !important;
    }

    .el-tabs__item {
        font-size: 12px;
        height: 25px;
        margin: 0px;
        padding: 0 6px !important;
    }
}
</style>
