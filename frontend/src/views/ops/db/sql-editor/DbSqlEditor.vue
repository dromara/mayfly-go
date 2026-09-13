<template>
    <div class="h-full flex flex-col">
        <SqlEditorToolbar :token="token" :upload-url="getUploadSqlFileUrl()" :upload-fn="handleSqlFileUpload" @run="onRunSql()" @format="onFormatSql()" @commit="onCommit()" @save="saveSql()" />

        <el-splitter ref="splitterRef" class="flex-1 min-h-0" layout="vertical" @resize-end="onResizeTableHeight">
            <el-splitter-panel :size="state.editorSize" max="80%">
                <!-- 高度扣除 mt-1(4px) + 组件边框(2px)，使内容恰好填满面板，避免溢出滚动条 -->
                <MonacoEditor
                    ref="monacoEditorRef"
                    class="mt-1"
                    v-model="state.sql"
                    language="sql"
                    height="calc(100% - 6px)"
                    :id="'MonacoTextarea-' + getKey()"
                    @ready="initMonacoEditor"
                />
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
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';

// 纯类型导入（编译期擦除）：monaco 运行时经 setup 动态取用，见 initMonacoEditor 的注释
import type { editor, IRange } from 'monaco-editor';

import { dbApi, uploadSqlFile } from '../api';
import { DbInst } from '../db';

import { joinClientParams } from '@/common/request';
import type { MonacoEditorExpose } from '@/components/monaco/types';
import { Msg } from '@/hooks/useI18n';
import { useDebounceFn, useEventListener } from '@vueuse/core';
import { useI18n } from 'vue-i18n';

import SqlEditorToolbar from './SqlEditorToolbar.vue';
import SqlExecResultTabs from './SqlExecResultTabs.vue';
import { formatSql } from './utils/formatSql';
import { useSqlExec, type ExecResTab, type ExecResTabState } from './composables/useSqlExec';

// 编辑器主体约 967KB(gzip)，本组件所在 tab 只在点开 SQL 编辑器时才挂载，故保持动态引入
// （main.ts 已在首屏空闲时预取，实际不会出现等待）
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const emits = defineEmits<{
    /** 脚本保存成功，回传所属库，父级据此刷新该库下的脚本树节点 */
    saveSqlSuccess: [dbId: number, dbName: string];
}>();

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

const token = getToken();
const monacoEditorRef = useTemplateRef<MonacoEditorExpose>('monacoEditorRef');

// 编辑器实例在 ready 前不存在（编辑器为按需加载），故可空；useSqlExec 的入参契约本就是 | null
let monacoEditor: editor.IStandaloneCodeEditor | null = null;

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
    // 编辑器为按需加载，快捷键等初始化挂在其 @ready 上（见模板），此处不再用定时器猜就绪时机
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
    const sql = monacoEditor?.getModel()?.getValue();
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
const onFormatSql = async () => {
    const editorIns = monacoEditor;
    const selection = editorIns?.getSelection();
    const model = editorIns?.getModel();
    if (!selection || !model) {
        return;
    }

    const formatDialect = getNowDbInst().getDialect().getInfo().formatSqlDialect;
    // 有选中sql则格式化并替换选中sql, 否则格式化编辑器所有内容
    const read = () => (selection.isEmpty() ? model.getValue() : model.getValueInRange(selection));
    const source = read();

    const formatted = await formatSql(source, formatDialect);
    // 格式化器按需加载，首次点击存在加载窗口：期间关掉查询 tab 会销毁 model（继续读写必抛错），
    // 用户改过内容也不能再覆盖，两种情况都丢弃本次结果
    if (model.isDisposed() || read() !== source) {
        return;
    }
    if (selection.isEmpty()) {
        model.setValue(formatted);
        return;
    }
    replaceSelection(formatted, selection);
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
    const editorIns = monacoEditor;
    const model = editorIns?.getModel();
    if (!editorIns || !model) {
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

    editorIns.setValue(textBeforeSelection + str + textAfterSelection);
    editorIns.focus();
    editorIns.setPosition({
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

const initMonacoEditor = async () => {
    const editorInstance = monacoEditorRef.value?.getEditor();
    if (!editorInstance) {
        return;
    }
    monacoEditor = editorInstance;

    // 快捷键常量属于 monaco 运行时。本函数由编辑器的 ready 事件触发，此刻编辑器 chunk 已在内存，
    // 动态导入不会多下一份，却能让本组件的静态图与 monaco 脱钩（否则 SQL 编辑器 tab 要等整份编辑器下载完才出现）
    const { KeyCode, KeyMod } = await import('@/components/monaco/setup');

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

/*
 * monaco hover（如 find 挂件按钮 tooltip）会向上弹出 splitter 面板顶边，
 * 而 .el-splitter-panel 默认 overflow:auto 会把溢出部分裁剪掉（视觉上像被工具栏按钮遮挡）。
 * hover 显示期间解除裁剪，使 tooltip 能像 VSCode 一样悬浮到上层；
 * :has() 仅显示期间精确匹配，且编辑器高度已恰好填满面板（无滚动条），切换不会引起布局抖动。
 */
.el-splitter-panel:has(.monaco-hover) {
    overflow: visible;
}
</style>
