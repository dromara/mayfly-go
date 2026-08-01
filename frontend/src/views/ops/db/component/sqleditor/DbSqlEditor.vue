<template>
    <div class="h-full flex flex-col">
        <SqlEditorToolbar :token="token" :upload-url="getUploadSqlFileUrl()" :upload-fn="handleSqlFileUpload" @run="onRunSql()" @format="onFormatSql()" @commit="onCommit()" @save="saveSql()" />

        <el-splitter ref="splitterRef" class="flex-1 min-h-0" layout="vertical" @resize-end="onResizeTableHeight">
            <el-splitter-panel :size="state.editorSize" max="80%">
                <MonacoEditor ref="monacoEditorRef" class="mt-1" v-model="state.sql" language="sql" height="100%" :id="'MonacoTextarea-' + getKey()" />
            </el-splitter-panel>

            <el-splitter-panel>
                <SqlExecResultTabs
                    :exec-res-tabs="state.execResTabs"
                    :active-tab="state.activeTab"
                    :db-id="dbId"
                    :db="dbName"
                    :table-data-height="tableDataHeight"
                    :table-data-empty-text="state.tableDataEmptyText"
                    @tab-remove="onRemoveTab"
                    @tab-change="active"
                    @submit-update-fields="submitUpdateFields"
                    @cancel-update-fields="cancelUpdateFields"
                    @change-updated-field="changeUpdatedField"
                    @data-delete="onDeleteData"
                />
            </el-splitter-panel>
        </el-splitter>
    </div>
</template>

<script lang="ts" setup>
import { notBlank } from '@/common/assert';
import config from '@/common/config';
import { getToken } from '@/common/utils/storage';
import { ElMessageBox } from 'element-plus';
import { format as sqlFormatter } from 'sql-formatter';
import { nextTick, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';

import { editor, KeyCode, KeyMod, type IRange } from 'monaco-editor';

import { dbApi, uploadSqlFile } from '../../api';
import { DbInst } from '../../db';

import { joinClientParams } from '@/common/request';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { Msg } from '@/hooks/useI18n';
import { useDebounceFn, useEventListener } from '@vueuse/core';
import { useI18n } from 'vue-i18n';

import SqlEditorToolbar from './SqlEditorToolbar.vue';
import SqlExecResultTabs from './SqlExecResultTabs.vue';
import { useSqlExec, type ExecResTab, type ExecResTabState } from './composables/useSqlExec';

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

/** sql-formatter 支持的方言语言类型 */
type SqlFormatterLanguage = NonNullable<Parameters<typeof sqlFormatter>[1]>['language'];

const token = getToken();
const monacoEditorRef = useTemplateRef<InstanceType<typeof MonacoEditor>>('monacoEditorRef');

let monacoEditor: editor.IStandaloneCodeEditor;

const state = reactive({
    editorSize: 50, // editor高度比例
    sql: '', // 当前编辑器的sql内容
    sqlName: '', // sql模板名称
    execResTabs: [] as ExecResTabState[],
    activeTab: 1,
    editorHeight: '500',
    tableDataHeight: '250px',
    tableDataEmptyText: t('db.tableDataEmptyTextTips'),
});

const { tableDataHeight } = toRefs(state);

const { getNowDbInst, pushNewTab, onRunSql, changeUpdatedField, onDeleteData, submitUpdateFields, cancelUpdateFields, onRemoveTab, activeTab: active } = useSqlExec({
    dbId: props.dbId,
    dbName: props.dbName,
    state,
    get monacoEditor() {
        return monacoEditor ?? null;
    },
});

onMounted(async () => {
    // 第一个pane为sql editor
    onResizeTableHeight(0, [-1]);
    useEventListener(
        'resize',
        useDebounceFn(() => onResizeTableHeight(0, [-1]), 200)
    );

    // 默认新建一个结果集tab
    pushNewTab(1);

    state.sqlName = props.sqlName ?? '';
    if (props.sqlName) {
        const res = await dbApi.getSql.request({ id: props.dbId, type: 1, db: props.dbName, name: props.sqlName });
        state.sql = res.sql;
    }
    nextTick(() => {
        setTimeout(() => initMonacoEditor(), 50);
    });
    await getNowDbInst().loadDbHints(props.dbName);
});

const splitterRef = useTemplateRef<{ $el: HTMLElement }>('splitterRef');

const onResizeTableHeight = (index: number, sizes: number[]) => {
    if (!sizes || sizes.length === 0) {
        return;
    }

    // 基于splitter容器实际高度计算，兼容全屏模式
    const splitterEl = splitterRef.value?.$el || splitterRef.value;
    const containerHeight = splitterEl ? (splitterEl as HTMLElement).getBoundingClientRect().height : window.innerHeight - 220;
    const plitpaneHeight = containerHeight - 10;

    let editorHeight = sizes[0];
    if (editorHeight < 0 || editorHeight > plitpaneHeight - 43) {
        // 默认占50%
        editorHeight = plitpaneHeight / 2;
    }

    let tableDataHeight = plitpaneHeight - editorHeight - 15;

    state.editorSize = editorHeight;
    state.tableDataHeight = tableDataHeight + 'px';
};

const getKey = () => {
    if (props.sqlName) {
        return `${props.dbId}:${props.dbName}.${props.sqlName}`;
    }
    return props.dbId + ':' + props.dbName;
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
    Msg.saveSuccess();
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

    const formatDialect = getNowDbInst().getDialect().getInfo().formatSqlDialect as SqlFormatterLanguage;

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
    Msg.success('COMMIT success');
};

/**
 * 替换选中的内容
 */
const replaceSelection = (str: string, selection: IRange) => {
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

// 自定义SQL文件上传处理
const handleSqlFileUpload = (options: { file: File }) => {
    const { file } = options;

    const { uploadId, abort } = uploadSqlFile(
        file,
        {
            dbId: props.dbId as number,
            dbName: props.dbName as string,
        },
        {
            onSuccess: () => {
                Msg.success('db.scriptFileUploadSuccess', { filename: file.name });
            },
            onError: (error) => {
                Msg.error('db.scriptFileUploadFailed', { filename: file.name, error: error.message });
            },
        }
    );

    return { abort };
};

// 获取sql文件上传执行url
const getUploadSqlFileUrl = () => {
    return `${config.baseApiUrl}/dbs/${props.dbId}/exec-sql-file?db=${props.dbName}&${joinClientParams()}`;
};

const initMonacoEditor = () => {
    const editorInstance = monacoEditorRef.value?.getEditor();
    if (!editorInstance) {
        return;
    }
    monacoEditor = editorInstance;

    // 注册快捷键：ctrl + R 运行选中的sql
    monacoEditor.addAction({
        id: 'run-sql-action' + getKey(),
        label: t('db.runSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyCode.KeyR, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.5,
        run: async function () {
            try {
                await onRunSql();
            } catch (e: unknown) {
                e instanceof Error && e.message && Msg.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + shift + R 新tab运行选中的sql
    monacoEditor.addAction({
        id: 'run-sql-action-on-newtab' + getKey(),
        label: t('db.newTabRunSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyMod.Shift | KeyCode.KeyR, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.6,
        run: async function () {
            try {
                await onRunSql(true);
            } catch (e: unknown) {
                e instanceof Error && e.message && Msg.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + shift + f 格式化sql
    monacoEditor.addAction({
        id: 'format-sql-action' + getKey(),
        label: t('db.formatSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyMod.Shift | KeyCode.KeyF, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 2,
        run: async function () {
            try {
                await onFormatSql();
            } catch (e: unknown) {
                e instanceof Error && e.message && Msg.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + s 保存sql
    monacoEditor.addAction({
        id: 'save-sql-action' + getKey(),
        label: t('db.saveSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyCode.KeyS, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 3,
        run: async function () {
            await saveSql();
        },
    });
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
