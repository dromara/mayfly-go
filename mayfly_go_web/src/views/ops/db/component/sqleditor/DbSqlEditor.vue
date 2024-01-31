<template>
    <div>
        <div>
            <div class="card pd5 flex-justify-between">
                <div>
                    <el-link @click="onRunSql()" :underline="false" class="ml15" icon="VideoPlay"> </el-link>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="1000" class="box-item" effect="dark" content="format sql" placement="top">
                        <el-link @click="formatSql()" type="primary" :underline="false" icon="MagicStick"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="1000" class="box-item" effect="dark" content="commit" placement="top">
                        <el-link @click="onCommit()" type="success" :underline="false" icon="CircleCheck"> </el-link>
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
                        <el-tooltip :show-after="1000" class="box-item" effect="dark" content="SQL脚本执行" placement="top">
                            <el-link type="success" :underline="false" icon="Document"></el-link>
                        </el-tooltip>
                    </el-upload>
                </div>

                <div>
                    <el-button @click="saveSql()" type="primary" icon="document-add" plain size="small">保存SQL</el-button>
                </div>
            </div>
        </div>

        <Splitpanes
            @pane-maximize="resizeTableHeight([{ size: 0 }])"
            @resize="resizeTableHeight"
            horizontal
            class="default-theme"
            style="height: calc(100vh - 233px)"
        >
            <Pane :size="state.editorSize" max-size="80">
                <MonacoEditor ref="monacoEditorRef" class="mt5" v-model="state.sql" language="sql" height="100%" :id="'MonacoTextarea-' + getKey()" />
            </Pane>

            <Pane :size="100 - state.editorSize">
                <div class="mt5 sql-exec-res h100">
                    <el-tabs class="h100 w100" v-if="state.execResTabs.length > 0" @tab-remove="onRemoveTab" v-model="state.activeTab">
                        <el-tab-pane class="h100" closable v-for="dt in state.execResTabs" :label="dt.id" :name="dt.id" :key="dt.id">
                            <template #label>
                                <el-popover :show-after="1000" placement="top-start" title="执行信息" trigger="hover" :width="300">
                                    <template #reference>
                                        <div>
                                            <span>
                                                <span v-if="dt.loading">
                                                    <SvgIcon class="mb2 is-loading" name="Loading" color="var(--el-color-primary)" />
                                                </span>
                                                <span v-else>
                                                    <SvgIcon class="mb2" v-if="!dt.errorMsg" name="CircleCheck" color="var(--el-color-success)" />
                                                    <SvgIcon class="mb2" v-if="dt.errorMsg" name="CircleClose" color="var(--el-color-error)" />
                                                </span>
                                            </span>

                                            <span> 结果{{ dt.id }} </span>
                                        </div>
                                    </template>
                                    <template #default>
                                        <el-descriptions v-if="dt.sql" :column="1" size="small">
                                            <el-descriptions-item>
                                                <div style="width: 280px">
                                                    <el-text size="small" truncated :title="dt.sql"> {{ dt.sql }} </el-text>
                                                </div>
                                            </el-descriptions-item>
                                            <el-descriptions-item label="耗时 :"> {{ dt.execTime }}ms </el-descriptions-item>
                                            <el-descriptions-item label="结果集 :">
                                                {{ dt.data?.length }}
                                            </el-descriptions-item>
                                        </el-descriptions>
                                    </template>
                                </el-popover>
                            </template>

                            <el-row>
                                <span v-if="dt.hasUpdatedFileds" class="mt5">
                                    <span>
                                        <el-link type="success" :underline="false" @click="submitUpdateFields(dt)"
                                            ><span style="font-size: 12px">提交</span></el-link
                                        >
                                    </span>
                                    <span>
                                        <el-divider direction="vertical" border-style="dashed" />
                                        <el-link type="warning" :underline="false" @click="cancelUpdateFields(dt)"
                                            ><span style="font-size: 12px">取消</span></el-link
                                        >
                                    </span>
                                </span>
                            </el-row>
                            <db-table-data
                                v-if="!dt.errorMsg"
                                :ref="(el) => (dt.dbTableRef = el)"
                                :db-id="dbId"
                                :db="dbName"
                                :data="dt.data"
                                :table="dt.table"
                                :columns="dt.tableColumn"
                                :loading="dt.loading"
                                :abort-fn="dt.abortFn"
                                :height="tableDataHeight"
                                empty-text="tips: select *开头的单表查询或点击表名默认查询的数据,可双击数据在线修改"
                                @change-updated-field="changeUpdatedField($event, dt)"
                                @data-delete="onDeleteData($event, dt)"
                            ></db-table-data>

                            <el-result v-else icon="error" title="执行失败" :sub-title="dt.errorMsg"> </el-result>
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </Pane>
        </Splitpanes>
    </div>
</template>

<script lang="ts" setup>
import { h, nextTick, onMounted, reactive, ref, toRefs, unref } from 'vue';
import { getToken } from '@/common/utils/storage';
import { notBlank } from '@/common/assert';
import { format as sqlFormatter } from 'sql-formatter';
import config from '@/common/config';
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus';

import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import { editor } from 'monaco-editor';

import DbTableData from '@/views/ops/db/component/table/DbTableData.vue';
import { DbInst } from '../../db';
import { dbApi } from '../../api';

import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { joinClientParams } from '@/common/request';
import { buildProgressProps } from '@/components/progress-notify/progress-notify';
import ProgressNotify from '@/components/progress-notify/progress-notify.vue';
import syssocket from '@/common/syssocket';
import SvgIcon from '@/components/svgIcon/index.vue';
import { Pane, Splitpanes } from 'splitpanes';

const emits = defineEmits(['saveSqlSuccess']);

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
});

const { tableDataHeight } = toRefs(state);

const getNowDbInst = () => {
    return DbInst.getInst(props.dbId);
};

onMounted(async () => {
    console.log('in query mounted');

    // 第一个pane为sql editor
    resizeTableHeight([{ size: state.editorSize }]);
    window.onresize = () => {
        resizeTableHeight([{ size: state.editorSize }]);
    };

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

const resizeTableHeight = (e: any) => {
    const vh = window.innerHeight;
    state.editorSize = e[0].size;
    const plitpaneHeight = vh - 233;
    const editorHeight = plitpaneHeight * (state.editorSize / 100);
    state.tableDataHeight = plitpaneHeight - editorHeight - 40 + 'px';
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
    notBlank(sql && sql.trim(), '请选中需要执行的sql');
    // 去除字符串前的空格、换行等
    sql = sql.replace(/(^\s*)/g, '');
    let execRemark = '';
    let canRun = true;
    if (
        sql.startsWith('update') ||
        sql.startsWith('UPDATE') ||
        sql.startsWith('INSERT') ||
        sql.startsWith('insert') ||
        sql.startsWith('DELETE') ||
        sql.startsWith('delete')
    ) {
        const res: any = await ElMessageBox.prompt('请输入备注', 'Tip', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            inputPattern: /^[\s\S]*.*[^\s][\s\S]*$/,
            inputErrorMessage: '请输入执行该sql的备注信息',
        });
        execRemark = res.value;
        if (!execRemark) {
            canRun = false;
        }
    }
    if (!canRun) {
        return;
    }

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
            ElMessage.error('当前结果集tab正在执行, 请使用新标签执行');
            return;
        }
        id = execRes.id;
    }

    state.activeTab = id;
    const startTime = new Date().getTime();
    try {
        execRes.errorMsg = '';
        execRes.sql = '';

        const { data, execute, isFetching, abort } = getNowDbInst().execSql(props.dbName, sql, execRemark);
        execRes.loading = isFetching;
        execRes.abortFn = abort;

        await execute();
        const colAndData: any = data.value;
        if (!colAndData.res || colAndData.res.length === 0) {
            ElMessage.warning('未查询到结果集');
            return;
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
        execRes.errorMsg = e.msg;
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
    notBlank(sql, 'sql内容不能为空');

    let sqlName = state.sqlName;
    if (!sqlName) {
        try {
            const input = await ElMessageBox.prompt('请输入SQL脚本名', 'SQL名', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                inputPattern: /.+/,
                inputErrorMessage: '请输入SQL脚本名',
            });
            sqlName = input.value;
            state.sqlName = sqlName;
        } catch (e) {
            return;
        }
    }

    await dbApi.saveSql.request({ id: props.dbId, db: props.dbName, sql: sql, type: 1, name: sqlName });
    ElMessage.success('保存成功');
    // 保存sql脚本成功事件
    emits('saveSqlSuccess', props.dbId, props.dbName);
};

/**
 * 格式化sql
 */
const formatSql = () => {
    let selection = monacoEditor.getSelection();
    if (!selection) {
        return;
    }

    const formatDialect = getNowDbInst().getDialect().getInfo().formatSqlDialect;

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

/**
 * sql文件执行进度通知缓存
 */
const sqlExecNotifyMap: Map<string, any> = new Map();
const beforeUpload = (file: File) => {
    ElMessage.success(`'${file.name}' 正在上传执行, 请关注结果通知`);
    syssocket.registerMsgHandler('execSqlFileProgress', function (message: any) {
        const content = JSON.parse(message.msg);
        const id = content.id;
        let progress = sqlExecNotifyMap.get(id);
        if (content.terminated) {
            if (progress != undefined) {
                progress.notification?.close();
                sqlExecNotifyMap.delete(id);
                progress = undefined;
            }
            return;
        }

        if (progress == undefined) {
            progress = {
                props: reactive(buildProgressProps()),
                notification: undefined,
            };
        }
        progress.props.progress.title = content.title;
        progress.props.progress.executedStatements = content.executedStatements;
        if (!sqlExecNotifyMap.has(id)) {
            progress.notification = ElNotification({
                duration: 0,
                title: message.title,
                message: h(ProgressNotify, progress.props),
                type: syssocket.getMsgType(message.type),
                showClose: false,
            });
            sqlExecNotifyMap.set(id, progress);
        }
    });
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
        label: '执行SQL',
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
        label: '新标签执行SQL',
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
        label: '格式化SQL',
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
                await formatSql();
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
        label: '保存SQL',
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
