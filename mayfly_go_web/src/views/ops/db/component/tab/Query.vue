<template>
    <div>
        <div>
            <div class="toolbar">
                <div class="fl">
                    <el-link @click="onRunSql()" :underline="false" class="ml15" icon="VideoPlay"> </el-link>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip class="box-item" effect="dark" content="format sql" placement="top">
                        <el-link @click="formatSql()" type="primary" :underline="false" icon="MagicStick"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
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
                        <el-tooltip class="box-item" effect="dark" content="SQL脚本执行" placement="top">
                            <el-link type="success" :underline="false" icon="Document"></el-link>
                        </el-tooltip>
                    </el-upload>
                    <el-divider direction="vertical" border-style="dashed" />
                    <el-tooltip class="box-item" effect="dark" content="limit" placement="top">
                        <el-link @click="onLimit()" type="success" :underline="false" icon="Operation"> </el-link>
                    </el-tooltip>
                </div>

                <div style="float: right" class="fl">
                    <el-button @click="saveSql()" type="primary" icon="document-add" plain size="small">保存SQL </el-button>
                    <el-button v-if="sqlName" @click="deleteSql()" type="danger" icon="delete" plain size="small">删除SQL </el-button>
                </div>
            </div>
        </div>

        <div class="mt5 sqlEditor">
            <div :id="'MonacoTextarea-' + ti.key" :style="{ height: editorHeight }"></div>
        </div>

        <div class="editor-move-resize" @mousedown="onDragSetHeight">
            <el-icon>
                <Minus />
            </el-icon>
        </div>

        <div class="mt5">
            <el-row>
                <el-link v-if="table" @click="onDeleteData()" class="ml5" type="danger" icon="delete" :underline="false"></el-link>

                <span v-if="execRes.data.length > 0">
                    <el-divider direction="vertical" border-style="dashed" />
                    <el-link type="success" :underline="false" @click="exportData"><span style="font-size: 12px">导出</span></el-link>
                </span>
                <span v-if="hasUpdatedFileds">
                    <el-divider direction="vertical" border-style="dashed" />
                    <el-link type="success" :underline="false" @click="submitUpdateFields()"><span style="font-size: 12px">提交</span></el-link>
                </span>
                <span v-if="hasUpdatedFileds">
                    <el-divider direction="vertical" border-style="dashed" />
                    <el-link type="warning" :underline="false" @click="cancelUpdateFields"><span style="font-size: 12px">取消</span></el-link>
                </span>
            </el-row>
            <db-table
                ref="dbTableRef"
                :db-id="state.ti.dbId"
                :db="state.ti.db"
                :data="execRes.data"
                :table="state.table"
                :columns="execRes.tableColumn"
                :loading="loading"
                :height="tableDataHeight"
                empty-text="tips: select *开头的单表查询或点击表名默认查询的数据,可双击数据在线修改"
                @selection-change="onDataSelectionChange"
                @change-updated-field="changeUpdatedField"
            ></db-table>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { nextTick, watch, onMounted, reactive, toRefs, ref, Ref } from 'vue';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { getSession } from '@/common/utils/storage';
import { isTrue, notBlank } from '@/common/assert';
import { format as sqlFormatter } from 'sql-formatter';
import config from '@/common/config';
import { ElMessage, ElMessageBox } from 'element-plus';
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker.js?worker';
import * as monaco from 'monaco-editor';
import { editor } from 'monaco-editor';

// 主题仓库 https://github.com/brijeshb42/monaco-themes
// 主题例子 https://editor.bitwiser.in/
import SolarizedLight from 'monaco-themes/themes/Solarized-light.json';
import DbTable from '../DbTable.vue';
import { TabInfo } from '../../db';
import { exportCsv } from '@/common/utils/export';
import { dateStrFormat } from '@/common/utils/date';
import { dbApi } from '../../api';

const emits = defineEmits(['saveSqlSuccess', 'deleteSqlSuccess']);

const props = defineProps({
    data: {
        type: TabInfo,
        required: true,
    },
    // sql脚本名，若有则去加载该sql内容
    sqlName: {
        type: String,
        default: '',
    },
    editorHeight: {
        type: String,
        default: '600',
    },
});

const { themeConfig } = storeToRefs(useThemeConfig());
const token = getSession('token');
let monacoEditor = {} as editor.IStandaloneCodeEditor;
const dbTableRef = ref(null) as Ref;

const state = reactive({
    token,
    ti: {} as TabInfo,
    dbs: [],
    dbId: null, // 当前选中操作的数据库实例
    table: '', // 当前单表操作sql的表信息
    sqlName: '',
    sql: '', // 当前编辑器的sql内容
    loading: false, // 是否在加载数据
    execRes: {
        data: [],
        tableColumn: [],
    },
    selectionDatas: [] as any,
    editorHeight: '500',
    tableDataHeight: 250 as any,
    hasUpdatedFileds: false,
});

const { tableDataHeight, editorHeight, ti, execRes, table, sqlName, loading, hasUpdatedFileds } = toRefs(state);

watch(
    () => props.editorHeight,
    (newValue: any) => {
        state.editorHeight = newValue;
    }
);

onMounted(async () => {
    console.log('in query mounted');
    state.ti = props.data;
    state.editorHeight = props.editorHeight;
    const params = state.ti.params;
    state.dbs = params && params.dbs;

    if (params && params.sqlName) {
        state.sqlName = params.sqlName;
        const res = await dbApi.getSql.request({ id: state.ti.dbId, type: 1, name: state.sqlName, db: state.ti.db });
        state.sql = res.sql;
    }
    nextTick(() => {
        setTimeout(() => initMonacoEditor(), 50);
    });
    await state.ti.getNowDbInst().loadDbHints(state.ti.db);
});

self.MonacoEnvironment = {
    getWorker() {
        return new EditorWorker();
    },
};

const initMonacoEditor = () => {
    let monacoTextarea = document.getElementById('MonacoTextarea-' + state.ti.key) as HTMLElement;
    // options参数参考 https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.IStandaloneEditorConstructionOptions.html#language
    // 初始化一些主题
    monaco.editor.defineTheme('SolarizedLight', SolarizedLight);
    monacoEditor = monaco.editor.create(monacoTextarea, {
        language: 'sql',
        theme: themeConfig.value.editorTheme,
        automaticLayout: true, //自适应宽高布局
        folding: false,
        roundedSelection: false, // 禁用选择文本背景的圆角
        matchBrackets: 'near',
        linkedEditing: true,
        cursorBlinking: 'smooth', // 光标闪烁样式
        mouseWheelZoom: true, // 在按住Ctrl键的同时使用鼠标滚轮时，在编辑器中缩放字体
        overviewRulerBorder: false, // 不要滚动条的边框
        tabSize: 2, // tab 缩进长度
        // fontFamily: 'JetBrainsMono', // 字体 暂时不要设置，否则光标容易错位
        fontWeight: 'bold',
        // letterSpacing: 1, 字符间距
        // quickSuggestions:false, // 禁用代码提示
        minimap: {
            enabled: false, // 不要小地图
        },
    });

    // 注册快捷键：ctrl + R 运行选中的sql
    monacoEditor.addAction({
        // An unique identifier of the contributed action.
        id: 'run-sql-action' + state.ti.key,
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

    // 注册快捷键：ctrl + shift + f 格式化sql
    monacoEditor.addAction({
        // An unique identifier of the contributed action.
        id: 'format-sql-action' + state.ti.key,
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

    // 动态设置主题
    // monaco.editor.setTheme('hc-black');

    // 如果sql有值，则默认赋值
    if (state.sql) {
        monacoEditor.getModel()?.setValue(state.sql);
    }
};

/**
 * 拖拽改变sql编辑区和查询结果区高度
 */
const onDragSetHeight = () => {
    document.onmousemove = (e) => {
        e.preventDefault();
        //得到鼠标拖动的宽高距离：取绝对值
        state.editorHeight = `${document.getElementById('MonacoTextarea-' + state.ti.key)!.offsetHeight + e.movementY}px`;
        state.tableDataHeight -= e.movementY;
    };
    document.onmouseup = () => {
        document.onmousemove = null;
    };
};

/**
 * 执行sql
 */
const onRunSql = async () => {
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

    try {
        state.loading = true;

        const colAndData: any = await state.ti.getNowDbInst().runSql(state.ti.db, sql, execRemark);
        if (!colAndData.res || colAndData.res.length === 0) {
            ElMessage.warning('未查询到结果集');
        }
        state.execRes.data = colAndData.res;
        // 兼容表格字段配置
        state.execRes.tableColumn = colAndData.colNames.map((x: any) => {
            return {
                columnName: x,
                show: true,
            };
        });
        cancelUpdateFields();
    } catch (e: any) {
        state.execRes.data = [];
        state.execRes.tableColumn = [];
        state.table = '';
        return;
    } finally {
        state.loading = false;
    }

    // 即只有以该字符串开头的sql才可修改表数据内容
    if (sql.startsWith('SELECT *') || sql.startsWith('select *') || sql.startsWith('SELECT\n  *')) {
        state.selectionDatas = [];
        const tableName = sql.split(/from/i)[1];
        if (tableName) {
            const tn = tableName.trim().split(' ')[0].split('\n')[0];
            state.table = tn;
            state.table = tn;
        } else {
            state.table = '';
        }
    } else {
        state.table = '';
    }
};

/**
 * 获取sql，如果有鼠标选中，则返回选中内容，否则返回输入框内所有内容
 */
const getSql = () => {
    let res = '' as string | undefined;
    // 编辑器还没初始化
    if (!monacoEditor?.getModel) {
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
    const newSql = !sqlName;
    if (newSql) {
        try {
            const input = await ElMessageBox.prompt('请输入SQL脚本名', 'SQL名', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                inputPattern: /\w+/,
                inputErrorMessage: '请输入SQL脚本名',
            });
            sqlName = input.value;
            state.sqlName = sqlName;
        } catch (e) {
            return;
        }
    }

    await dbApi.saveSql.request({ id: state.ti.dbId, db: state.ti.db, sql: sql, type: 1, name: sqlName });
    ElMessage.success('保存成功');
    // 保存sql脚本成功事件
    emits('saveSqlSuccess', state.ti.dbId, state.ti.db);
};

const deleteSql = async () => {
    const sqlName = state.sqlName;
    notBlank(sqlName, '该sql内容未保存');
    const { dbId, db } = state.ti;
    try {
        await ElMessageBox.confirm(`确定删除【${sqlName}】该SQL内容?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteDbSql.request({ id: dbId, db: db, name: sqlName });
        ElMessage.success('删除成功');
        emits('deleteSqlSuccess', dbId, db);
    } catch (err) {}
};

/**
 * 格式化sql
 */
const formatSql = () => {
    let selection = monacoEditor.getSelection();
    if (!selection) {
        return;
    }
    let sql = monacoEditor.getModel()?.getValueInRange(selection);
    // 有选中sql则格式化并替换选中sql, 否则格式化编辑器所有内容
    if (sql) {
        replaceSelection(sqlFormatter(sql), selection);
        return;
    }
    monacoEditor.getModel()?.setValue(sqlFormatter(monacoEditor.getValue()));
};

/**
 * 提交事务，用于没有开启自动提交事务
 */
const onCommit = () => {
    state.ti.getNowDbInst().runSql(state.ti.db, 'COMMIT;');
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

const onLimit = () => {
  let position = monacoEditor.getPosition() as monaco.Position;
  let newText = " limit 10";
  monacoEditor?.getModel().applyEdits([
    {
      range: new monaco.Range(position.lineNumber, position.column, position.lineNumber, position.column),
      text: newText
    }
  ]);
}

/**
 * 导出当前页数据
 */
const exportData = () => {
    const dataList = state.execRes.data as any;
    isTrue(dataList.length > 0, '没有数据可导出');
    exportCsv(
        `数据查询导出-${dateStrFormat('yyyyMMddHHmm', new Date().toString())}`,
        state.execRes.tableColumn.map((x: any) => x.columnName),
        dataList
    );
};

const beforeUpload = (file: File) => {
    ElMessage.success(`'${file.name}' 正在上传执行, 请关注结果通知`);
};

// 执行sql成功
const execSqlFileSuccess = (res: any) => {
    if (res.code !== 200) {
        ElMessage.error(res.msg);
    }
};

// 获取sql文件上传执行url
const getUploadSqlFileUrl = () => {
    return `${config.baseApiUrl}/dbs/${state.ti.dbId}/exec-sql-file?db=${state.ti.db}`;
};

const onDataSelectionChange = (datas: []) => {
    state.selectionDatas = datas;
};

const changeUpdatedField = (updatedFields: []) => {
    // 如果存在要更新字段，则显示提交和取消按钮
    state.hasUpdatedFileds = updatedFields && updatedFields.length > 0;
};

/**
 * 执行删除数据事件
 */
const onDeleteData = async () => {
    const deleteDatas = state.selectionDatas;
    isTrue(deleteDatas && deleteDatas.length > 0, '请先选择要删除的数据');
    const { db } = state.ti;
    const dbInst = state.ti.getNowDbInst();
    const primaryKey = await dbInst.loadTableColumn(db, state.table);
    const primaryKeyColumnName = primaryKey.columnName;
    dbInst.promptExeSql(db, dbInst.genDeleteByPrimaryKeysSql(db, state.table, deleteDatas), null, () => {
        state.execRes.data = state.execRes.data.filter(
            (d: any) => !(deleteDatas.findIndex((x: any) => x[primaryKeyColumnName] == d[primaryKeyColumnName]) != -1)
        );
        state.selectionDatas = [];
    });
};

const submitUpdateFields = () => {
    dbTableRef.value.submitUpdateFields();
};

const cancelUpdateFields = () => {
    dbTableRef.value.cancelUpdateFields();
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

.sqlEditor {
    font-size: 8pt;
    font-weight: 600;
    border: 1px solid #ccc;
}

.update_field_active {
    background-color: var(--el-color-success);
}

.editor-move-resize {
    cursor: n-resize;
    height: 3px;
    text-align: center;
}
</style>
