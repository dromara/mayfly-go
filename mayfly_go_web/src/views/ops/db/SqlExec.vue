<template>
    <div>
        <el-row type="flex">
            <el-col :span="24">
                <el-button type="primary" icon="plus" @click="addQueryTab" size="small">新建查询</el-button>
            </el-col>
            <el-col :span="4" style="border-left: 1px solid #eee; margin-top: 10px">
                <InstanceTree :instance-menu-max-height="state.instanceMenuMaxHeight" :instances="state.instances"
                    @init-load-instances="loadInstances" @change-instance="changeInstance" @change-schema="changeSchema"
                    @load-table-names="loadSchemaTables" @load-table-data="loadTableData" />
            </el-col>
            <el-col :span="20">
                <el-container id="data-exec" style="border-left: 1px solid #eee; margin-top: 10px">
                    <el-tabs @tab-remove="removeDataTab" @tab-click="onDataTabClick" style="width: 100%"
                        v-model="state.activeName">
                        <el-tab-pane closable v-for="q in state.queryTabs" :key="q.id" :label="q.label" :name="q.name">
                            <div>
                                <div>
                                    <div class="toolbar">
                                        <div class="fl">
                                            <el-link @click="onRunSql(q.dbId, q.db)" :underline="false" class="ml15"
                                                icon="VideoPlay">
                                            </el-link>
                                            <el-divider direction="vertical" border-style="dashed" />

                                            <el-tooltip class="box-item" effect="dark" content="format sql"
                                                placement="top">
                                                <el-link @click="formatSql(q.dbId, q.db)" type="primary"
                                                    :underline="false" icon="MagicStick">
                                                </el-link>
                                            </el-tooltip>
                                            <el-divider direction="vertical" border-style="dashed" />

                                            <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
                                                <el-link @click="onCommit(q.dbId, q.db)" type="success"
                                                    :underline="false" icon="CircleCheck">
                                                </el-link>
                                            </el-tooltip>
                                            <el-divider direction="vertical" border-style="dashed" />

                                            <el-upload class="sql-file-exec" :before-upload="beforeUpload"
                                                :on-success="execSqlFileSuccess" :headers="{ Authorization: token }"
                                                :data="{ dbId: q.dbId }" :action="getUploadSqlFileUrl(q.dbId, q.db)"
                                                :show-file-list="false" name="file" multiple :limit="100">
                                                <el-tooltip class="box-item" effect="dark" content="SQL脚本执行"
                                                    placement="top">
                                                    <el-link type="success" :underline="false"
                                                        icon="Document"></el-link>
                                                </el-tooltip>
                                            </el-upload>
                                        </div>

                                        <div style="float: right" class="fl">
                                            <el-select v-model="state.sqlName[q.dbId + q.db]" placeholder="选择or输入SQL模板名"
                                                @change="changeSqlTemplate(q.dbId, q.db)" filterable allow-create
                                                default-first-option size="small" class="mr10">
                                                <el-option v-for="item in state.sqlNames[q.dbId + q.db]" :key="item"
                                                    :label="item.database" :value="item">
                                                    {{ item }}
                                                </el-option>
                                            </el-select>

                                            <el-button @click="saveSql(q.dbId, q.db)" type="primary" icon="document-add"
                                                plain size="small">保存
                                            </el-button>
                                            <el-button @click="deleteSql(q.dbId, q.db)" type="danger" icon="delete"
                                                plain size="small">删除
                                            </el-button>
                                        </div>
                                    </div>
                                </div>

                                <div class="mt5 sqlEditor">
                                    <div :id="'MonacoTextarea-' + q.id" :style="{ height: state.monacoOptions.height }">
                                    </div>
                                </div>
                                <div class="mt5">
                                    <el-row>
                                        <el-link v-if="q.nowTableName"
                                            @click="onDeleteData(q.dbId, q.db, q.nowTableName)" class="ml5"
                                            type="danger" icon="delete" :underline="false"></el-link>

                                        <span v-if="q.execRes.data.length > 0">
                                            <el-divider direction="vertical" border-style="dashed" />
                                            <el-link type="success" :underline="false" @click="exportData"><span
                                                    style="font-size: 12px">导出</span></el-link>
                                        </span>
                                        <span v-if="q.updatedFields.length > 0">
                                            <el-divider direction="vertical" border-style="dashed" />
                                            <el-link type="success" :underline="false"
                                                @click="submitUpdateFields(q.dbId, q.db, q.nowTableName)"><span
                                                    style="font-size: 12px">提交</span></el-link>
                                        </span>
                                        <span v-if="q.updatedFields.length > 0">
                                            <el-divider direction="vertical" border-style="dashed" />
                                            <el-link type="warning" :underline="false" @click="cancelUpdateFields"><span
                                                    style="font-size: 12px">取消</span></el-link>
                                        </span>
                                    </el-row>
                                    <el-table
                                        @cell-dblclick="(row: any, column: any, cell: any, event: any) => cellClick(row, column, cell, event, { dbId: q.dbId, db: q.db, tableName: q.nowTableName })"
                                        @selection-change="onDataSelectionChange" size="small" :data="q.execRes.data"
                                        v-loading="q.loading" element-loading-text="查询中..."
                                        empty-text="tips: select *开头的单表查询或点击表名默认查询的数据,可双击数据在线修改" stripe border
                                        class="mt5">
                                        <el-table-column v-if="q.execRes.tableColumn.length > 0 && q.nowTableName"
                                            type="selection" width="35" />
                                        <el-table-column min-width="100" :width="flexColumnWidth(item, q.execRes.data)"
                                            align="center" v-for="item in q.execRes.tableColumn" :key="item"
                                            :prop="item" :label="item" show-overflow-tooltip>
                                        </el-table-column>
                                    </el-table>
                                    <el-row type="flex" class="mt5" justify="center">
                                        <el-pagination v-show="q.execRes.showPage" small :total="q.execRes.total"
                                            @current-change="doRunSql(q.dbId, q.db, q.sql)"
                                            layout="prev,pager,next,total,jumper"
                                            v-model:current-page="q.execRes.pageNum" :page-size="defalutLimit">
                                        </el-pagination>
                                    </el-row>
                                </div>

                            </div>
                        </el-tab-pane>

                        <el-tab-pane closable v-for="dt in state.dataTabs" :key="dt.key" :label="dt.label"
                            :name="dt.key">
                            <el-row>
                                <el-col :span="8">
                                    <el-link @click="onRefresh(dt.dbId, dt.db, dt.name)" icon="refresh"
                                        :underline="false" class="ml5">
                                    </el-link>
                                    <el-divider direction="vertical" border-style="dashed" />

                                    <el-link @click="addRow(dt.dbId, dt.db, dt.name)" type="primary" icon="plus"
                                        :underline="false"></el-link>
                                    <el-divider direction="vertical" border-style="dashed" />

                                    <el-link @click="onDeleteData(dt.dbId, dt.db, dt.name)" type="danger" icon="delete"
                                        :underline="false"></el-link>
                                    <el-divider direction="vertical" border-style="dashed" />

                                    <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
                                        <el-link @click="onCommit(dt.dbId, dt.db)" type="success" icon="CircleCheck"
                                            :underline="false">
                                        </el-link>
                                    </el-tooltip>
                                    <el-divider direction="vertical" border-style="dashed" />

                                    <el-tooltip class="box-item" effect="dark" content="生成insert sql" placement="top">
                                        <el-link @click="onGenerateInsertSql(dt.dbId, dt.db, dt.name)" type="success"
                                            :underline="false">gi</el-link>
                                    </el-tooltip>
                                    <el-divider direction="vertical" border-style="dashed" />

                                    <el-tooltip v-if="state.updatedFields[state.nowTableName]?.length > 0"
                                        class="box-item" effect="dark" content="提交修改" placement="top">
                                        <el-link @click="submitUpdateFields(dt.dbId, dt.db, dt.name)" type="success"
                                            :underline="false">提交</el-link>
                                    </el-tooltip>
                                    <el-divider v-if="state.updatedFields[state.nowTableName]?.length > 0"
                                        direction="vertical" border-style="dashed" />
                                    <el-tooltip v-if="state.updatedFields[state.nowTableName]?.length > 0"
                                        class="box-item" effect="dark" content="取消修改" placement="top">
                                        <el-link @click="cancelUpdateFields" type="warning"
                                            :underline="false">取消</el-link>
                                    </el-tooltip>
                                </el-col>
                                <el-col :span="16">
                                    <el-input v-model="dt.condition" placeholder="若需条件过滤，可选择列并点击对应的字段并输入需要过滤的内容点击查询按钮即可"
                                        clearable size="small" style="width: 100%">
                                        <template #prepend>
                                            <el-popover trigger="click" :width="320" placement="right">
                                                <template #reference>
                                                    <el-link type="success" :underline="false">选择列</el-link>
                                                </template>
                                                <el-table :data="getColumns4Map(dt.name)" max-height="500" size="small"
                                                    @row-click="
                                                        (...event: any) => {
                                                            onConditionRowClick(event, dt);
                                                        }
                                                    " style="cursor: pointer">
                                                    <el-table-column property="columnName" label="列名"
                                                        show-overflow-tooltip>
                                                    </el-table-column>
                                                    <el-table-column property="columnComment" label="备注"
                                                        show-overflow-tooltip>
                                                    </el-table-column>
                                                </el-table>
                                            </el-popover>
                                        </template>

                                        <template #append>
                                            <el-button @click="selectByCondition(dt.name, dt.condition)" icon="search"
                                                size="small"></el-button>
                                        </template>
                                    </el-input>
                                </el-col>
                            </el-row>
                            <el-table
                                @cell-dblclick="(row: any, column: any, cell: any, event: any) => cellClick(row, column, cell, event, { dbId: dt.dbId, db: dt.db, tableName: dt.name })"
                                @sort-change="(sort: any) => onTableSortChange(dt.dbId, dt.db, sort, dt.name)"
                                @selection-change="onDataSelectionChange" :data="dt.datas" size="small"
                                :max-height="state.dataTabsTableHeight" v-loading="dt.loading"
                                element-loading-text="查询中..." empty-text="暂无数据" stripe border class="mt5">
                                <el-table-column v-if="dt.datas.length > 0" type="selection" width="35" />
                                <el-table-column min-width="100" :width="flexColumnWidth(item, dt.datas)" align="center"
                                    v-for="item in dt.columnNames" :key="item" :prop="item" :label="item"
                                    show-overflow-tooltip :sortable="state.nowTableName !== '' ? 'custom' : false">
                                    <template #header>
                                        <el-tooltip raw-content placement="top" effect="customized">
                                            <template #content> {{ getColumnTip(dt.name, item) }} </template>
                                            {{ item }}
                                        </el-tooltip>
                                    </template>
                                </el-table-column>
                            </el-table>
                            <el-row type="flex" class="mt5" justify="center">
                                <el-pagination small :total="dt.count" @current-change="handlePageChange(dt)"
                                    layout="prev, pager, next, total, jumper" v-model:current-page="dt.pageNum"
                                    :page-size="defalutLimit"></el-pagination>
                            </el-row>
                            <div style=" font-size: 12px; padding: 0 10px; color: #606266"><span>{{ dt.sql }}</span></div>
                        </el-tab-pane>
                    </el-tabs>
                </el-container>
            </el-col>
        </el-row>

        <el-dialog v-model="state.conditionDialog.visible" :title="state.conditionDialog.title" width="420px">
            <el-row>
                <el-col :span="5">
                    <el-select v-model="state.conditionDialog.condition">
                        <el-option label="=" value="="> </el-option>
                        <el-option label="LIKE" value="LIKE"> </el-option>
                        <el-option label=">" value=">"> </el-option>
                        <el-option label=">=" value=">="> </el-option>
                        <el-option label="<" value="<"> </el-option>
                        <el-option label="<=" value="<="> </el-option>
                    </el-select>
                </el-col>
                <el-col :span="19">
                    <el-input v-model="state.conditionDialog.value" :placeholder="state.conditionDialog.placeholder" />
                </el-col>
            </el-row>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="onCancelCondition">取消</el-button>
                    <el-button type="primary" @click="onConfirmCondition">确定</el-button>
                </span>
            </template>
        </el-dialog>

        <el-dialog @close="state.genSqlDialog.visible = false" v-model="state.genSqlDialog.visible" title="SQL"
            width="1000px">
            <el-input v-model="state.genSqlDialog.sql" type="textarea" rows="20" />
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, onMounted, reactive, watch } from 'vue';
import { dbApi } from './api';

import { format as sqlFormatter } from 'sql-formatter';
import { isTrue, notBlank, notEmpty } from '@/common/assert';
import { ElMessage, ElMessageBox } from 'element-plus';
import config from '@/common/config';
import { getSession } from '@/common/utils/storage';
import SqlExecBox from './component/SqlExecBox';
import { dateStrFormat } from '@/common/utils/date.ts';
import { useStore } from '@/store/index.ts';

import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker.js?worker';
import { language as sqlLanguage } from 'monaco-editor/esm/vs/basic-languages/mysql/mysql.js';
import * as monaco from 'monaco-editor';
import { editor, languages, Position } from 'monaco-editor';

// 主题仓库 https://github.com/brijeshb42/monaco-themes
// 主题例子 https://editor.bitwiser.in/
import SolarizedLight from 'monaco-themes/themes/Solarized-light.json';
import InstanceTree from '@/views/ops/db/component/InstanceTree.vue';

const store = useStore();
const token = getSession('token');
const tableMap = new Map();
const defalutLimit = 20

type TableMeta = {
    // 表名
    tableName: string,
    // 表注释
    tableComment: string
}
/** 修改表字段所需数据 */
type UpdateFieldsMeta = {
    // 主键值
    primaryKey: string
    // 主键名
    primaryKeyName: string
    // 主键类型
    primaryKeyType: string
    // 新值
    fields: FieldsMeta[]
}
type FieldsMeta = {
    // 字段所在div
    div: HTMLElement
    // 字段名
    fieldName: string
    // 字段所在的表格行数据
    row: any
    // 字段类型
    fieldType: string
    // 原值
    oldValue: string
    // 新值
    newValue: string
}

// 使用全局变量缓存实例对应的
const queryTabMonacoEditors = {}
const state = reactive({
    token: token,
    tags: [],
    dbs: [] as any, // 数据库实例列表
    databaseList: {}, // 数据库实例拥有的数据库列表，1数据库实例  -> 多数据库
    db: '', // 当前操作的数据库
    dbType: '',
    tables: [] as any,
    dbId: null, // 当前选中操作的数据库实例
    sqlName: {}, // 当前sql模板名
    sqlNames: {}, // 所有sql模板名
    sqlMap: {}, // 所有sql名对应的sql
    activeName: 'Query',
    activeNameMap: {},// 缓存活跃tab对应的dbId和db
    nowTableName: '', // 当前表格数据操作的数据库表名，用于双击编辑表内容使用
    dataTabs: {} as any, // 点击表信息后执行结果数据展示tabs
    queryTabs: {} as any, // 查询tab
    //  queryTab: {
    //     id: '', // 唯一id
    //     sql: '',
    //     label: '查询',
    //     name: 'Query',
    //     // 点击执行按钮执行结果信息
    //     execRes: {
    //       total: 0,
    //       pageNum: 1,
    //       showPage: false,
    //       data: [],
    //       tableColumn: []
    //     },
    //     loading: false,
    //     nowTableName: '', //当前表格数据操作的数据库表名，用于双击编辑表内容使用
    //     selectionDatas: [],
    //     updatedFields: [] as UpdateFieldsMeta[]
    //   },
    dataTabsTableHeight: 600,
    params: {
        pageNum: 1,
        pageSize: 100,
        tagPath: null
    },
    conditionDialog: {
        title: '',
        placeholder: '',
        columnRow: null,
        dataTab: null,
        visible: false,
        condition: '=',
        value: null
    },
    genSqlDialog: {
        visible: false,
        sql: '',
    },
    monacoOptions: {
        editor: {} as editor.IStandaloneCodeEditor,
        height: '',
        dbTables: {},
    },
    updatedFields: {} as { [tableName: string]: UpdateFieldsMeta[] },// 各个tab表被修改的字段信息
    instances: {
        tags: {},
        tree: {},
        dbs: {},
        tables: {},
        sqls: {},
    },
    instanceMenuMaxHeight: '850px'
});

// 获取布局配置信息
const getThemeConfig: any = computed(() => {
    return store.state.themeConfig.themeConfig;
});

self.MonacoEnvironment = {
    getWorker() {
        return new EditorWorker();
    }
};

const initMonacoEditor = (queryTab: any) => {
    let monacoTextarea = document.getElementById(queryTab.editorId) as HTMLElement
    // options参数参考 https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.IStandaloneEditorConstructionOptions.html#language
    // 初始化一些主题
    monaco.editor.defineTheme('SolarizedLight', SolarizedLight);
    queryTab.monacoEditor = monaco.editor.create(monacoTextarea, {
        language: 'sql',
        theme: getThemeConfig.value.editorTheme,
        automaticLayout: true, //自适应宽高布局
        folding: false,
        roundedSelection: false, // 禁用选择文本背景的圆角
        matchBrackets: 'near',
        linkedEditing: true,
        cursorBlinking: 'smooth',// 光标闪烁样式
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

    queryTabMonacoEditors[queryTab.name] = queryTab.monacoEditor

    // 注册快捷键：ctrl + R 运行选中的sql
    queryTab.monacoEditor.addAction({
        // An unique identifier of the contributed action.
        id: 'run-sql-action' + queryTab.dbId + queryTab.db,
        // A label of the action that will be presented to the user.
        label: '执行SQL',
        // A precondition for this action.
        precondition: undefined,
        // A rule to evaluate on top of the precondition in order to dispatch the keybindings.
        keybindingContext: undefined,
        keybindings: [
            // chord
            monaco.KeyMod.chord(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyR, 0)
        ],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.5,
        // Method that will be executed when the action is triggered.
        // @param editor The editor instance is passed in as a convenience
        run: async function () {
            try {
                await onRunSql(queryTab.dbId, queryTab.db);
            } catch (e: any) {
                e.message && ElMessage.error(e.message)
            }
        }
    });

    // 注册快捷键：ctrl + shift + f 格式化sql
    queryTab.monacoEditor.addAction({
        // An unique identifier of the contributed action.
        id: 'format-sql-action' + queryTab.dbId + queryTab.db,
        // A label of the action that will be presented to the user.
        label: '格式化SQL',
        // A precondition for this action.
        precondition: undefined,
        // A rule to evaluate on top of the precondition in order to dispatch the keybindings.
        keybindingContext: undefined,
        keybindings: [
            // chord
            monaco.KeyMod.chord(monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyF, 0)
        ],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 2,
        // Method that will be executed when the action is triggered.
        // @param editor The editor instance is passed in as a convenience
        run: async function () {
            try {
                await formatSql(queryTab.dbId, queryTab.db);
            } catch (e: any) {
                e.message && ElMessage.error(e.message)
            }
        }
    });

    // 动态设置主题
    // monaco.editor.setTheme('hc-black');

    // 参考 https://microsoft.github.io/monaco-editor/playground.html#extending-language-services-completion-provider-example
    self.completionItemProvider = self.completionItemProvider || monaco.languages.registerCompletionItemProvider('sql', {
        triggerCharacters: ['.'],
        provideCompletionItems: async (model: editor.ITextModel, position: Position): Promise<languages.CompletionList | null | undefined> => {
            let word = model.getWordUntilPosition(position);
            let { dbId, db } = state.activeNameMap[state.activeName]
            const { lineNumber, column } = position
            const { startColumn, endColumn } = word

            // 当前行文本
            let lineContent = model.getLineContent(lineNumber);
            // 注释行不需要代码提示
            if (lineContent.startsWith('--')) {
                return { suggestions: [] }
            }

            let range = {
                startLineNumber: lineNumber,
                endLineNumber: lineNumber,
                startColumn,
                endColumn,
            };

            //  光标前文本
            const textBeforePointer = model.getValueInRange({
                startLineNumber: lineNumber,
                startColumn: 0,
                endLineNumber: lineNumber,
                endColumn: column
            })
            const textBeforePointerMulti = model.getValueInRange({
                startLineNumber: 1,
                startColumn: 0,
                endLineNumber: lineNumber,
                endColumn: column
            })
            // 光标后文本
            const textAfterPointerMulti = model.getValueInRange({
                startLineNumber: lineNumber,
                startColumn: column,
                endLineNumber: model.getLineCount(),
                endColumn: model.getLineMaxColumn(model.getLineCount())
            })
            // // const nextTokens = textAfterPointer.trim().split(/\s+/)
            // // const nextToken = nextTokens[0].toLowerCase()
            const tokens = textBeforePointer.trim().split(/\s+/)
            const lastToken = tokens[tokens.length - 1].toLowerCase()

            // console.log("光标前文本：=>" + textBeforePointerMulti)

            // console.log("最后输入的：=>" + lastToken)
            if (lastToken.endsWith('.')) {
                // 如果是.触发代码提示，则进行【 库.表名联想 】 或 【 表别名.表字段联想 】
                let str = lastToken.substring(0, lastToken.lastIndexOf('.'))
                // 库.表名联想
                if (state.instances.dbs[dbId].indexOf(str) > -1) {
                    let tables = await loadTableMetadata(dbId, str)
                    let suggestions: languages.CompletionItem[] = []
                    for (let item of tables) {
                        const { tableName, tableComment } = item
                        suggestions.push({
                            label: {
                                label: tableName + (tableComment ? ' - ' + tableComment : ''),
                                description: 'table'
                            },
                            kind: monaco.languages.CompletionItemKind.File,
                            insertText: tableName,
                            range
                        });
                    }
                    return { suggestions }
                }

                let sql = textBeforePointerMulti.split(';')[textBeforePointerMulti.split(';').length - 1] + textAfterPointerMulti.split(';')[0];
                // 表别名.表字段联想
                let tableInfo = getTableByAlias(sql, db, str)
                if (tableInfo.tableName) {
                    let table = tableInfo.tableName
                    let db = tableInfo.dbName
                    // 取出表名并提示
                    let dbs = state.monacoOptions.dbTables[dbId + db]
                    let columns = dbs ? (dbs[table] || []) : [];
                    if ((!columns || columns.length === 0) && db) {
                        state.monacoOptions.dbTables[dbId + db] = await loadHintTables(dbId, db)
                        dbs = state.monacoOptions.dbTables[dbId + db]
                        columns = dbs ? (dbs[table] || []) : [];
                    }
                    let suggestions: languages.CompletionItem[] = []
                    columns.forEach((a: string, index: any) => {
                        // 字段数据格式  字段名 字段注释，  如： create_time  [datetime][创建时间]
                        const nameAndComment = a.split("  ")
                        const fieldName = nameAndComment[0]
                        suggestions.push({
                            label: {
                                label: a,
                                description: 'column'
                            },
                            kind: monaco.languages.CompletionItemKind.Property,
                            detail: '', // 不显示detail, 否则选中时备注等会被遮挡
                            insertText: fieldName + ' ', // create_time
                            range,
                            sortText: 100 + index + '' // 使用表字段声明顺序排序,排序需为字符串类型
                        });
                    })
                    return { suggestions }
                }
                return { suggestions: [] }
            }

            // 库名联想

            let suggestions: languages.CompletionItem[] = []
            // mysql关键字
            sqlLanguage.keywords.forEach((item: any) => {
                suggestions.push({
                    label: {
                        label: item,
                        description: 'keyword'
                    },
                    kind: monaco.languages.CompletionItemKind.Keyword,
                    insertText: item,
                    range
                });
            })
            // 操作符
            sqlLanguage.operators.forEach((item: any) => {
                suggestions.push({
                    label: {
                        label: item,
                        description: 'opt'
                    },
                    kind: monaco.languages.CompletionItemKind.Operator,
                    insertText: item,
                    range
                });
            })
            // 内置函数
            sqlLanguage.builtinFunctions.forEach((item: any) => {
                suggestions.push({
                    label: {
                        label: item,
                        description: 'func'
                    },
                    kind: monaco.languages.CompletionItemKind.Function,
                    insertText: item,
                    range
                });
            })
            // 内置变量
            sqlLanguage.builtinVariables.forEach((item: string) => {
                suggestions.push({
                    label: {
                        label: item,
                        description: 'var'
                    },
                    kind: monaco.languages.CompletionItemKind.Variable,
                    insertText: item,
                    range
                });
            })

            // 库名提示
            state.instances.dbs[dbId].forEach((a: string) => {
                suggestions.push({
                    label: {
                        label: a,
                        description: 'schema'
                    },
                    kind: monaco.languages.CompletionItemKind.Folder,
                    insertText: a,
                    range
                });
            })

            // 表名联想
            state.instances.tables[dbId + db]?.forEach((tableMeta: TableMeta) => {
                const { tableName, tableComment } = tableMeta
                suggestions.push({
                    label: {
                        label: tableName + ' - ' + tableComment,
                        description: 'table'
                    },
                    kind: monaco.languages.CompletionItemKind.File,
                    detail: tableComment,
                    insertText: tableName + ' ',
                    range
                });
            })

            // 默认提示
            return {
                suggestions: suggestions
            };
        },
    });
};

/**
 * 根据别名获取sql里的表名
 * @param sql sql
 * @param db 默认数据库
 * @param alias 别名
 */
const getTableByAlias = (sql: string, db: string, alias: string): { dbName: string, tableName: string } => {

    // 表别名：表名
    let result = {};
    let defName = '';
    let defResult = {};
    // 正则匹配取出表名和表别名
    // 测试sql
    /*

    `select * from database.Outvisit l
left join patient p on l.patid=p.patientid
join patstatic c on   l.patid=c.patid inner join patphone  ph  on l.patid=ph.patid
where l.name='kevin' and exsits(select 1 from pharmacywestpas pw where p.outvisitid=l.outvisitid)
unit all
select * from invisit v where`.match(/(join|from)\s+(\w*-?\w*\.?\w+)\s*(as)?\s*(\w*)/gi)
     */
    let match = sql.match(/(join|from)\n*\s+\n*(\w*-?\w*\.?\w+)\s*(as)?\s*(\w*)\n*/gi)
    if (match && match.length > 0) {
        match.forEach(a => {
            // 去掉前缀，取出
            let t = a.substring(5, a.length)
                .replaceAll(/\s+/g, ' ')
                .replaceAll(/\s+as\s+/gi, ' ')
                .replaceAll(/\r\n/g, ' ').trim()
                .split(/\s+/);
            let withDb = t[0].split('.');
            // 表名是 db名.表名
            let tName = withDb.length > 1 ? withDb[1] : withDb[0]
            let dbName = withDb.length > 1 ? withDb[0] : (db || '')
            if (t.length == 2) {
                // 表别名：表名
                result[t[1]] = { tableName: tName, dbName }
            } else {
                // 只有表名无别名 取第一个无别名的表为默认表
                !defName && (defResult = { tableName: tName, dbName: db })
            }
        })
    }
    return result[alias] || defResult
}

onMounted(() => {
    self.completionItemProvider?.dispose()
    setHeight();
    instManage.loadSelectScheme()
    // 监听浏览器窗口大小变化,更新对应组件高度
    window.onresize = () => setHeight();
});

/**
 * 设置editor高度和数据表高度
 */
const setHeight = () => {
    // 默认300px
    state.monacoOptions.height = window.innerHeight - 550 + 'px'
    state.dataTabsTableHeight = window.innerHeight - 219 - 36;
    state.instanceMenuMaxHeight = window.innerHeight - 140 + 'px';
};

/**
 * 执行sql
 */
const onRunSql = async (dbId: any, db: string) => {
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

    await doRunSql(dbId, db, sql, execRemark)

    let key = state.activeName;
    // 即只有以该字符串开头的sql才可修改表数据内容
    if (sql.startsWith('SELECT *') || sql.startsWith('select *') || sql.startsWith('SELECT\n  *')) {
        state.queryTabs[key].selectionDatas = [];
        const tableName = sql.split(/from/i)[1];
        if (tableName) {
            const tn = tableName.trim().split(' ')[0];
            state.queryTabs[key].nowTableName = tn;
            state.nowTableName = tn;
        } else {
            state.queryTabs[key].nowTableName = '';
            state.nowTableName = '';
        }
    } else {
        state.queryTabs[key].nowTableName = '';
        state.nowTableName = '';
    }
};

const doRunSql = async (dbId: any, db: string, sql: string, execRemark?: string) => {
    let key = state.activeName;
    try {
        state.queryTabs[key].loading = true;

        // 执行新sql，还原分页信息
        if (state.queryTabs[key].sql !== sql) {
            state.queryTabs[key].execRes.total = 0
            state.queryTabs[key].execRes.pageNum = 1
            state.queryTabs[key].sql = sql;
        }

        // 干掉sql最后的分号
        if (sql.endsWith(';')) {
            sql = sql.substring(0, sql.length - 1)
        }
        // 给sql添加limit (pageNum-1)*pageSize, pageSize
        sql = sql.trim().toLowerCase();
        let countSql = `select count(*) ct from (${sql}) a`
        if (sql.startsWith('select') && sql.indexOf('limit') < 0) {
            state.queryTabs[key].execRes.showPage = true
            switch (state.dbType) {
                case 'mysql':
                    sql += ` limit  ${(state.queryTabs[key].execRes.pageNum - 1) * defalutLimit}, ${defalutLimit} `
                    break;
                case 'postgres':
                    sql += ` OFFSET ${(state.queryTabs[key].execRes.pageNum - 1) * defalutLimit} LIMIT ${defalutLimit} `
                    break;
            }
        } else {
            state.queryTabs[key].execRes.showPage = false
        }

        const colAndData: any = await runSql(dbId, db, sql, execRemark);
        if (!colAndData.res || colAndData.res.length === 0) {
            ElMessage.warning('暂无数据')
        }
        state.queryTabs[key].execRes.data = colAndData.res;
        if (colAndData.res && colAndData.res.length == defalutLimit) {
            const countRes = await runSql(dbId, db, countSql);
            state.queryTabs[key].execRes.total = countRes.res[0].ct
        } else {
            state.queryTabs[key].execRes.total = (state.queryTabs[key].execRes.pageNum - 1) * defalutLimit + colAndData.res.length
        }
        state.queryTabs[key].execRes.tableColumn = colAndData.colNames;
        state.queryTabs[key].loading = false;
        cancelUpdateFields()
    } catch (e: any) {
        state.queryTabs[key].loading = false;
    }
}


const exportData = () => {
    const dataList = state.queryTabs[state.activeName].execRes.data;
    isTrue(dataList.length > 0, '没有数据可导出');

    const tableColumn = state.queryTabs[state.activeName].execRes.tableColumn;
    // 二维数组
    const cvsData = [tableColumn];
    for (let data of dataList) {
        // 数据值组成的一维数组
        let dataValueArr: any = [];
        for (let column of tableColumn) {
            dataValueArr.push(data[column]);
        }
        cvsData.push(dataValueArr);
    }
    const csvString = cvsData.map((e) => e.join(',')).join('\n');

    // 导出
    let link = document.createElement('a');
    let exportContent = '\uFEFF';
    let blob = new Blob([exportContent + csvString], {
        type: 'text/plain;charset=utrf-8',
    });
    link.id = 'download-csv';
    link.setAttribute('href', URL.createObjectURL(blob));
    link.setAttribute('download', `查询数据导出-${dateStrFormat('yyyyMMddHHmmss', new Date())}.csv`);
    document.body.appendChild(link);
    link.click();
};

/**
 * 执行sql str
 *
 * @param dbId 数据库实例id
 * @param db 数据库schema
 * @param sql 执行的sql
 * @param remark sql备注
 */
const runSql = async (dbId: any, db: string, sql: string, remark: string = '') => {
    return await dbApi.sqlExec.request({
        id: dbId,
        db: db,
        sql: sql.trim(),
        remark,
    });
};

const removeDataTab = (targetName: string) => {
    let activeName = state.activeName;
    // 计算下一个tab
    const dataTabNames = Object.keys(state.dataTabs);
    let matched = false;
    dataTabNames.forEach((name, index) => {
        if (name === targetName) {
            const nextTab = dataTabNames[index + 1] || dataTabNames[index - 1];
            if (nextTab) {
                activeName = nextTab;
                matched = true
            }
            delete state.dataTabs[targetName];
        }
    });

    if (!matched) {
        const queryTabNames = Object.keys(state.queryTabs);
        queryTabNames.forEach((name: string, index: number) => {
            if (name === targetName) {
                const nextTab = queryTabNames[index + 1] || queryTabNames[index - 1];
                if (nextTab) {
                    activeName = nextTab;
                }
                delete state.queryTabs[targetName];
                return;
            }
        });
    }

    state.activeName = activeName;

};

/**
 * 数据tab点击
 */
const onDataTabClick = (tab: any) => {
    const name = tab.props.name;
    // 表数据tab，赋值当前表名，用于在线修改表数据
    if (!name.startsWith('查询')) {
        state.nowTableName = name;
    }
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
const getUploadSqlFileUrl = (dbId: any, db: string) => {
    return `${config.baseApiUrl}/dbs/${dbId}/exec-sql-file?db=${db}`;
};

const flexColumnWidth = (str: any, tableData: any, flag = 'equal') => {
    // str为该列的字段名(传字符串);tableData为该表格的数据源(传变量);
    // flag为可选值，可不传该参数,传参时可选'max'或'equal',默认为'max'
    // flag为'max'则设置列宽适配该列中最长的内容,flag为'equal'则设置列宽适配该列中第一行内容的长度。
    str = str + '';
    let columnContent = '';
    if (!tableData || !tableData.length || tableData.length === 0 || tableData === undefined) {
        return;
    }
    if (!str || !str.length || str.length === 0 || str === undefined) {
        return;
    }
    if (flag === 'equal') {
        // 获取该列中第一个不为空的数据(内容)
        for (let i = 0; i < tableData.length; i++) {
            // 转为字符串后比较
            if ((tableData[i][str] + '').length > 0) {
                columnContent = tableData[i][str] + '';
                break;
            }
        }
    } else {
        // 获取该列中最长的数据(内容)
        let index = 0;
        for (let i = 0; i < tableData.length; i++) {
            if (tableData[i][str] === null) {
                return;
            }
            const now_temp = tableData[i][str] + '';
            const max_temp = tableData[index][str] + '';
            if (now_temp.length > max_temp.length) {
                index = i;
            }
        }
        columnContent = tableData[index][str] + '';
    }
    const contentWidth: number = getContentWidth(columnContent);
    // 获取列名称的长度 加上排序图标长度
    const columnWidth: number = getContentWidth(str) + 43;
    const flexWidth: number = contentWidth > columnWidth ? contentWidth : columnWidth;
    return flexWidth + 'px';
};

/**
 * 获取内容所需要占用的宽度
 */
const getContentWidth = (content: any): number => {
    // 以下分配的单位长度可根据实际需求进行调整
    let flexWidth = 0;
    for (const char of content) {
        if (flexWidth > 500) {
            break;
        }
        if ((char >= '0' && char <= '9') || (char >= 'a' && char <= 'z')) {
            // 如果是小写字母、数字字符，分配8个单位宽度
            flexWidth += 8.5;
            continue;
        }
        if (char >= 'A' && char <= 'Z') {
            flexWidth += 9;
            continue;
        }
        if (char >= '\u4e00' && char <= '\u9fa5') {
            // 如果是中文字符，为字符分配16个单位宽度
            flexWidth += 16;
        } else {
            // 其他种类字符，为字符分配9个单位宽度
            flexWidth += 8;
        }
    }
    if (flexWidth > 500) {
        // 设置最大宽度
        flexWidth = 500;
    }
    return flexWidth;
};

const getColumnTip = (tableName: string, columnName: string) => {
    // 优先从 table map中获取
    let columns = getColumns4Map(tableName);
    if (!columns) {
        return '';
    }
    const column = columns.find((c: any) => c.columnName == columnName);
    const comment = column.columnComment;
    return `${column.columnType} ${comment ? ' |  ' + comment : ''}`;
};

/**
 * 获取sql，如果有鼠标选中，则返回选中内容，否则返回输入框内所有内容
 */
const getSql = () => {
    let monacoEditor = queryTabMonacoEditors[state.activeName]
    let res = '' as string | undefined;
    // 编辑器还没初始化
    if (!monacoEditor?.getModel) {
        return res;
    }
    // 选择选中的sql
    let selection = monacoEditor.getSelection()
    if (selection) {
        res = monacoEditor.getModel()?.getValueInRange(selection)
    }
    // 整个编辑器的sql
    if (!res) {
        return monacoEditor.getModel()?.getValue()
    }
    return res
};

const loadSchemaMeta = async (dbId: any, db: string) => {
    // 加载数据库下所有表
    state.instances.tables[dbId + db] = state.instances.tables[dbId + db] || await loadTableMetadata(dbId, db)

    // 加载数据库下所有表字段信息
    state.monacoOptions.dbTables[dbId + db] = state.monacoOptions.dbTables[dbId + db] || await loadHintTables(dbId, db)
}

const loadTableMetadata = async (dbId: any, db: string) => {
    return await dbApi.tableMetadata.request({ id: dbId, db })
}

const loadHintTables = async (dbId: any, db: string) => {
    return await dbApi.hintTables.request({ id: dbId, db, })
}

// 选择表事件
const changeTable = async (tableName: string, execSelectSql: boolean = true) => {
    if (tableName == '') {
        return;
    }
    if (!execSelectSql) {
        return;
    }

    // 执行sql，并新增tab
    state.activeName = state.dbId + state.db + tableName;
    state.nowTableName = state.activeName;
    let tab = state.dataTabs[state.dbId + state.db + tableName];
    // 如果存在该表tab，则直接返回
    if (tab) {
        return;
    }

    tab = {
        label: '`' + state.db + '`.' + tableName,
        key: state.dbId + state.db + tableName,
        name: tableName,
        datas: [],
        columnNames: [],
        pageNum: 1,
        count: 0,
        dbId: state.dbId, // 数据库id跟着tab走
        db: state.db,
    };
    tab.columnNames = await getColumnNames(tableName);
    state.dataTabs[state.dbId + state.db + tableName] = tab;

    await onRefresh(state.dbId, state.db, tableName);
};

/**
 * 获取表的所有列信息
 */
const getColumns = async (tableName: string) => {
    // 优先从 table map中获取
    let columns = getColumns4Map(tableName);
    if (columns) {
        return columns;
    }
    columns = await dbApi.columnMetadata.request({
        id: state.dbId,
        db: state.db,
        tableName: tableName,
    });
    tableMap.set(tableName, columns);
    return columns;
};

// 从缓存map获取列信息
const getColumns4Map = (tableName: string) => {
    return tableMap.get(tableName);
};

const getColumnNames = async (tableName: string) => {
    const columns = await getColumns(tableName);
    return columns.map((t: any) => t.columnName);
};

/**
 * 条件查询，点击列信息后显示输入对应的值
 */
const onConditionRowClick = (event: any, dataTab: any) => {
    const row = event[0];
    state.conditionDialog.title = `请输入 [${row.columnName}] 的值`;
    state.conditionDialog.placeholder = `${row.columnType}  ${row.columnComment}`;
    state.conditionDialog.columnRow = row;
    state.conditionDialog.dataTab = dataTab;
    state.conditionDialog.visible = true;
};

// 确认条件
const onConfirmCondition = () => {
    const conditionDialog = state.conditionDialog;
    const dataTab = state.conditionDialog.dataTab as any;
    let condition = dataTab.condition;
    if (condition) {
        condition += ` AND `;
    }
    const row = conditionDialog.columnRow as any;
    condition += `${row.columnName} ${conditionDialog.condition} `;
    dataTab.condition = condition + wrapColumnValue(row.columnType, conditionDialog.value);
    onCancelCondition();
};

const onCancelCondition = () => {
    state.conditionDialog.visible = false;
    state.conditionDialog.title = ``;
    state.conditionDialog.placeholder = ``;
    state.conditionDialog.value = null;
    state.conditionDialog.columnRow = null;
    state.conditionDialog.dataTab = null;
};

const onRefresh = async (dbId: any, db: string, tableName: string) => {
    const dataTab = state.dataTabs[state.dbId + state.db + tableName];
    // 查询条件置空
    dataTab.condition = '';
    dataTab.pageNum = 1;
    setDataTabDatas(dataTab).then(() => {
        cancelUpdateFields()
    });
};


/**
 * 数据tab修改页数
 */
const handlePageChange = async (dataTab: any) => {
    await setDataTabDatas(dataTab);
};

/**
 * 根据条件查询数据
 */
const selectByCondition = async (tableName: string, condition: string) => {
    notEmpty(condition, '条件不能为空');
    const dataTab = state.dataTabs[state.dbId + state.db + tableName];
    dataTab.pageNum = 1;
    await setDataTabDatas(dataTab);
};

/**
 * 设置data tab的表数据
 */
const setDataTabDatas = async (dataTab: any) => {
    dataTab.loading = true;
    try {
        dataTab.count = await getTableCount(dataTab.dbId, dataTab.db, dataTab.name, dataTab.condition);
        let sql = getDefaultSelectSql(dataTab.name, dataTab.condition, dataTab.orderBy, dataTab.pageNum)
        state.dataTabs[state.dbId + state.db + dataTab.name].sql = sql;
        if (dataTab.count > 0) {
            const colAndData: any = await runSql(dataTab.dbId, dataTab.db, sql);
            dataTab.datas = colAndData.res;
        } else {
            dataTab.datas = [];
        }
    } finally {
        dataTab.loading = false;
    }
};

/**
 * 获取表的统计数量
 */
const getTableCount = async (dbId: any, db: string, tableName: string, condition: string = '') => {
    const countRes = await runSql(dbId, db, getDefaultCountSql(tableName, condition));
    return countRes.res[0].count;
};

/**
 * 获取默认查询语句
 */
const getDefaultSelectSql = (tableName: string, where: string = '', orderBy: string = '', pageNum: number = 1) => {
    const baseSql = `SELECT * FROM ${tableName} ${where ? 'WHERE ' + where : ''} ${orderBy ? orderBy : ''}`;
    if (state.dbType == 'mysql') {
        return `${baseSql} LIMIT ${(pageNum - 1) * defalutLimit}, ${defalutLimit};`;
    }
    if (state.dbType == 'postgres') {
        return `${baseSql} OFFSET ${(pageNum - 1) * defalutLimit} LIMIT ${defalutLimit};`;
    }
    return baseSql;
};

/**
 * 获取默认查询统计语句
 */
const getDefaultCountSql = (tableName: string, where: string = '') => {
    return `SELECT COUNT(*) count FROM ${tableName} ${where ? 'WHERE ' + where : ''}`;
};

/**
 * 提交事务，用于没有开启自动提交事务
 */
const onCommit = (dbId: any, db: string) => {
    runSql(dbId, db, 'COMMIT;');
    ElMessage.success('COMMIT success');
};

/**
 * 表排序字段变更
 */
const onTableSortChange = async (dbId: any, db: string, sort: any, tableName: string) => {

    if (!state.nowTableName || !sort.prop) {
        return;
    }
    const sortType = sort.order == 'descending' ? 'DESC' : 'ASC';

    state.dataTabs[state.activeName].orderBy = `ORDER BY ${sort.prop} ${sortType}`;

    await onRefresh(dbId, db, tableName);
};

const changeSqlTemplate = (dbId: any, db: string) => {
    getUserSql(dbId, db, '');
};

/**
 * 获取用户保存的sql模板内容
 */
const getUserSql = (dbId: any, db: string, sql: string) => {
    notBlank(dbId, '请先选择数据库');
    sql && setSqlEditorValue(dbId, db, sql) ||
        dbApi.getSql.request({ id: dbId, type: 1, name: state.sqlName[dbId + db], db: db }).then((res: any) => {
            if (res) {
                setSqlEditorValue(dbId, db, res.sql);
                state.sqlMap[dbId + db + state.sqlName[dbId + db]] = res.sql
            } else {
                setSqlEditorValue(dbId, db, '');
            }
        });
};

const setSqlEditorValue = (dbId: any, db: string, value: string) => {
    let monacoEditor = queryTabMonacoEditors[state.activeName]
    monacoEditor?.getModel()?.setValue(value);
};

/**
 * 获取用户保存的sql模板名称
 */
const getSqlNames = (dbId: any, db: string) => {
    !state.sqlNames[dbId + db] &&
        dbApi.getSqlNames.request({ id: dbId, db: db, })
            .then((res: any) => {
                if (res && res.length > 0) {
                    state.sqlNames[dbId + db] = res.map((r: any) => r.name);
                    state.sqlName[dbId + db] = state.sqlNames[0];
                } else {
                    state.sqlNames[dbId + db] = ['default'] as any;
                    state.sqlName[dbId + db] = 'default';
                }
                getUserSql(dbId, db, '');
            })
}
const saveSql = async (dbId: any, db: string) => {
    let monacoEditor = queryTabMonacoEditors[state.activeName]
    const sql = monacoEditor.getModel()?.getValue();
    notBlank(sql, 'sql内容不能为空');
    notBlank(state.dbId, '请先选择数据库实例');
    await dbApi.saveSql.request({ id: dbId, db: db, sql: sql, type: 1, name: state.sqlName[dbId + db] });
    ElMessage.success('保存成功');

    dbApi.getSqlNames
        .request({
            id: state.dbId,
            db: state.db,
        })
        .then((res) => {
            if (res) {
                state.sqlNames[dbId + db] = res.map((r: any) => r.name);
            }
        });
};

const deleteSql = async (dbId: any, db: string) => {
    try {
        await ElMessageBox.confirm(`确定删除【${state.sqlName[dbId + db]}】该SQL模板?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteDbSql.request({ id: dbId, db: db, name: state.sqlName[dbId + db] });
        ElMessage.success('删除成功');
        getSqlNames(dbId, db);
    } catch (err) { }
};

const onDataSelectionChange = (datas: []) => {
    if (isQueryTab()) {
        state.queryTabs[state.activeName].selectionDatas = datas;
    } else {
        state.dataTabs[state.activeName].selectionDatas = datas;
    }
};

/**
 * 执行删除数据事件
 */
const onDeleteData = async (dbId: any, db: string, tableName: string) => {
    const isQuery = isQueryTab();
    const queryTab = state.queryTabs[state.activeName];
    const deleteDatas = isQuery ? queryTab.selectionDatas : state.dataTabs[state.activeName].selectionDatas;
    isTrue(deleteDatas && deleteDatas.length > 0, '请先选择要删除的数据');
    const primaryKey = await getColumn(tableName);
    const primaryKeyColumnName = primaryKey.columnName;
    const ids = deleteDatas.map((d: any) => `${wrapColumnValue(primaryKey.columnType, d[primaryKeyColumnName])}`).join(',');
    const sql = `DELETE FROM ${tableName} WHERE ${primaryKeyColumnName} IN (${ids})`;

    promptExeSql(dbId, db, sql, null, () => {
        if (!isQuery) {
            onRefresh(dbId, db, tableName);
        } else {
            queryTab.execRes.data = queryTab.execRes.data.filter(
                (d: any) => !(deleteDatas.findIndex((x: any) => x[primaryKeyColumnName] == d[primaryKeyColumnName]) != -1)
            );
            queryTab.selectionDatas = [];
        }
    });
};

const onGenerateInsertSql = async (dbId: any, db: string, tableName: string) => {
    const datas = isQueryTab() ? state.queryTabs[state.activeName].selectionDatas : state.dataTabs[state.activeName].selectionDatas;
    isTrue(datas && datas.length > 0, '请先选择要生成insert语句的数据');
    const columns: any = await getColumns(tableName);

    const sqls = [];
    for (let data of datas) {
        let colNames = [];
        let values = [];
        for (let column of columns) {
            const colName = column.columnName;
            colNames.push(colName);
            values.push(wrapValueByType(data[colName]));
        }
        sqls.push(`INSERT INTO ${tableName} (${colNames.join(', ')}) VALUES(${values.join(', ')})`);
    }
    state.genSqlDialog.sql = sqls.join(';\n') + ';';
    state.genSqlDialog.visible = true;
};

const wrapValueByType = (val: any) => {
    if (val == null) {
        return 'NULL';
    }
    if (typeof val == 'number') {
        return val;
    }
    return `'${val}'`;
};

/**
 * 是否为查询tab
 */
const isQueryTab = () => {
    return state.activeName.startsWith('查询');
};

// 监听单元格点击事件
const cellClick = (row: any, column: any, cell: any, event: Event, tabInfo: { dbId: number, db: string, tableName: string }) => {
    const property = column.property;
    // 如果当前操作的表名不存在 或者 当前列的property不存在(如多选框)，则不允许修改当前单元格内容
    if (!state.nowTableName || !property) {
        return;
    }
    let div: HTMLElement = cell.children[0];
    if (div && div.tagName === 'DIV') {
        // 转为字符串比较,可能存在数字等
        let text = (row[property] || row[property] == 0 ? row[property] : '') + '';
        let input = document.createElement('input');
        input.setAttribute('value', text);
        // 将表格width也赋值于输入框，避免输入框长度超过表格长度
        input.setAttribute('style', 'height:23px;text-align:center;border:none;' + div.getAttribute('style'));
        cell.replaceChildren(input);
        input.focus();
        input.addEventListener('blur', async () => {
            row[property] = input.value;
            cell.replaceChildren(div);
            if (input.value !== text) {
                let currentUpdatedFields: UpdateFieldsMeta[]
                if (isQueryTab()) {
                    currentUpdatedFields = state.queryTabs[state.activeName].updatedFields
                } else {
                    currentUpdatedFields = state.updatedFields[state.nowTableName];
                }
                // 主键
                const primaryKey = await getColumn(tabInfo.tableName);
                const primaryKeyValue = row[primaryKey.columnName];
                // 更新字段列信息
                const updateColumn = await getColumn(tabInfo.tableName, property);
                const newField = {
                    div, row,
                    fieldName: column.rawColumnKey,
                    fieldType: updateColumn.columnType,
                    oldValue: text,
                    newValue: input.value
                } as FieldsMeta;

                // 被修改的字段
                const primaryKeyFields = currentUpdatedFields.filter((meta) => meta.primaryKey === primaryKeyValue)
                let hasKey = false;
                if (primaryKeyFields.length <= 0) {
                    primaryKeyFields[0] = {
                        primaryKey: primaryKeyValue,
                        primaryKeyName: primaryKey.columnName,
                        primaryKeyType: primaryKey.columnType,
                        fields: [newField]
                    }
                } else {
                    hasKey = true
                    let hasField = primaryKeyFields[0].fields.some(a => {
                        if (a.fieldName === newField.fieldName) {
                            a.newValue = newField.newValue
                        }
                        return a.fieldName === newField.fieldName
                    })
                    if (!hasField) {
                        primaryKeyFields[0].fields.push(newField)
                    }
                }
                let fields = primaryKeyFields[0].fields

                const fieldsParam = fields.filter((a) => {
                    if (a.fieldName === column.rawColumnKey) {
                        a.newValue = input.value
                    }
                    return a.fieldName === column.rawColumnKey
                })

                const field = fieldsParam.length > 0 && fieldsParam[0] || {} as FieldsMeta
                if (field.oldValue === input.value) { // 新值=旧值
                    // 删除数据
                    div.classList.remove('update_field_active')
                    let delIndex: number[] = [];
                    currentUpdatedFields.forEach((a, i) => {
                        if (a.primaryKey === primaryKeyValue) {
                            a.fields = a.fields && a.fields.length > 0 ? a.fields.filter(f => f.fieldName !== column.rawColumnKey) : [];
                            a.fields.length <= 0 && delIndex.push(i)
                        }
                    });
                    delIndex.forEach(i => delete currentUpdatedFields[i])
                    currentUpdatedFields = currentUpdatedFields.filter(a => a)
                } else {
                    // 新增数据
                    div.classList.add('update_field_active')
                    if (hasKey) {
                        currentUpdatedFields.forEach((value, index, array) => {
                            if (value.primaryKey === primaryKeyValue) {
                                array[index].fields = fields
                            }
                        })
                    } else {
                        currentUpdatedFields.push({
                            primaryKey: primaryKeyValue,
                            primaryKeyName: primaryKey.columnName,
                            primaryKeyType: primaryKey.columnType,
                            fields
                        })
                    }
                }
                if (isQueryTab()) {
                    state.queryTabs[state.activeName].updatedFields = currentUpdatedFields
                } else {
                    state.updatedFields[state.nowTableName] = currentUpdatedFields
                }
            }
        });
    }
};

const submitUpdateFields = (dbId: any, db: string, tableName: string) => {
    let currentUpdatedFields: UpdateFieldsMeta[];
    let isQuery = false;
    if (isQueryTab()) {
        isQuery = true;
        currentUpdatedFields = state.queryTabs[state.activeName].updatedFields
    } else {
        currentUpdatedFields = state.updatedFields[state.nowTableName]
    }
    if (currentUpdatedFields.length <= 0) {
        return;
    }
    let res = '';
    let divs: HTMLElement[] = [];
    currentUpdatedFields.forEach(a => {
        let sql = `UPDATE ${tableName} SET `;
        let primaryKey = a.primaryKey;
        let primaryKeyType = a.primaryKeyType;
        let primaryKeyName = a.primaryKeyName;
        a.fields.forEach(f => {
            sql += ` ${f.fieldName} = ${wrapColumnValue(f.fieldType, f.newValue)},`
            divs.push(f.div)
        })
        sql = sql.substring(0, sql.length - 1)
        sql += ` WHERE ${primaryKeyName} = ${wrapColumnValue(primaryKeyType, primaryKey)} ;`
        res += sql;
    })

    promptExeSql(dbId, db, res, () => { }, () => {
        currentUpdatedFields = [];
        divs.forEach(a => {
            a.classList.remove('update_field_active')
        })
        if (isQuery) {
            state.queryTabs[state.activeName].updatedFields = []
            doRunSql(dbId, db, state.queryTabs[state.activeName].sql)
        } else {
            state.updatedFields[state.nowTableName] = []
            onRefresh(dbId, db, tableName)
        }
    });

}

const cancelUpdateFields = () => {
    if (isQueryTab()) {
        state.queryTabs[state.activeName].updatedFields.forEach((a: any) => {
            a.fields.forEach((b: any) => {
                b.div.classList.remove('update_field_active')
                b.row[b.fieldName] = b.oldValue
            })
        })
        state.queryTabs[state.activeName].updatedFields = []
    } else {
        state.updatedFields[state.nowTableName]?.forEach(a => {
            a.fields.forEach(b => {
                b.div.classList.remove('update_field_active')
                b.row[b.fieldName] = b.oldValue
            })
        })
        state.updatedFields[state.nowTableName] = []
    }
}

/**
 * 根据字段列名获取字段列信息。
 * 若字段列名为空，则返回主键列，若无主键列返回第一个字段列信息（用于获取主键等）
 */
const getColumn = async (tableName: string, columnName: string = '') => {
    const cols = await getColumns(tableName);
    if (!columnName) {
        const col = cols.find((c: any) => c.columnKey == 'PRI');
        return col || cols[0];
    }
    return cols.find((c: any) => c.columnName == columnName);
};

/**
 * 根据字段信息包装字段值，如为字符串等则添加‘’
 */
const wrapColumnValue = (columnType: string, value: any) => {
    if (isNumber(columnType)) {
        return value;
    }
    return `'${value}'`;
};

/**
 * 判断字段类型是否为数字类型
 */
const isNumber = (columnType: string) => {
    return columnType.match(/int|double|float|nubmer|decimal/gi);
};

/**
 * 弹框提示是否执行sql
 */
const promptExeSql = (dbId: any, db: string, sql: string, cancelFunc: any = null, successFunc: any = null) => {
    SqlExecBox({
        sql, dbId, db,
        runSuccessCallback: successFunc,
        cancelCallback: cancelFunc,
    });
};

// 添加新数据行
const addRow = async (dbId: any, db: string, tableName: string) => {
    const columns = await getColumns(tableName);
    // key: 字段名，value: 字段名提示
    let obj: any = {};
    columns.forEach((item: any) => {
        obj[`${item.columnName}`] = `'${item.columnComment || ''} ${item.columnName}[${item.columnType}]${item.nullable == 'YES' ? '' : '[not null]'}'`;
    });
    let columnNames = Object.keys(obj).join(',');
    let values = Object.values(obj).join(',');
    let sql = `INSERT INTO ${tableName} (${columnNames}) VALUES (${values});`;
    promptExeSql(dbId, db, sql, null, () => {
        onRefresh(dbId, db, tableName);
    });
};

/**
 * 格式化sql
 */
const formatSql = (dbId: any, db: string) => {
    let monacoEditor = queryTabMonacoEditors[state.activeName]
    let selection = monacoEditor.getSelection()
    if (!selection) {
        return;
    }
    let sql = monacoEditor.getModel()?.getValueInRange(selection)
    // 有选中sql则格式化并替换选中sql, 否则格式化编辑器所有内容
    if (sql) {
        replaceSelection(dbId, db, sqlFormatter(sql), selection)
        return;
    }
    monacoEditor.getModel()?.setValue(sqlFormatter(monacoEditor.getValue()));
};

/**
 * 替换选中的内容
 */
const replaceSelection = (dbId: any, db: string, str: string, selection: any) => {
    let monacoEditor = queryTabMonacoEditors[state.activeName]
    const model = monacoEditor.getModel();
    if (!model) {
        return;
    }
    if (!selection) {
        model.setValue(str);
        return;
    }
    const { startLineNumber, endLineNumber, startColumn, endColumn } = selection

    const textBeforeSelection = model.getValueInRange({
        startLineNumber: 1,
        startColumn: 0,
        endLineNumber: startLineNumber,
        endColumn: startColumn,
    })

    const textAfterSelection = model.getValueInRange({
        startLineNumber: endLineNumber,
        startColumn: endColumn,
        endLineNumber: model.getLineCount(),
        endColumn: model.getLineMaxColumn(model.getLineCount()),
    })

    monacoEditor.setValue(textBeforeSelection + str + textAfterSelection)
    monacoEditor.focus()
    monacoEditor.setPosition({
        lineNumber: startLineNumber,
        column: 0,
    })
}

// 加载实例数据
const loadInstances = async () => {
    const res = await dbApi.dbs.request({ pageNum: 1, pageSize: 1000, })
    if (!res.total) return

    state.instances = { tags: {}, tree: {}, dbs: {}, tables: {}, sqls: {} }; // 初始化变量
    for (const db of res.list) {
        let arr = state.instances.tree[db.tagId] || []
        const { tagId, tagPath } = db
        // tags
        state.instances.tags[db.tagId] = { tagId, tagPath }

        // tree
        arr.push(db)
        state.instances.tree[db.tagId] = arr;

        // dbs
        state.instances.dbs[db.id] = db.database.split(' ')

    }
}

// 加载实例对应的所有表名
const loadSchemaTables = async (inst: any, schema: string, fn: Function) => {
    changeSchema(inst, schema)
    let { id } = inst
    let tables = state.instances.tables[id + schema];
    if (!tables) {
        let tables = await dbApi.tableMetadata.request({ id, db: schema })
        tables && tables.forEach((a: any) => a.show = true)
        state.instances.tables[id + schema] = tables;
    } else {
        tables.forEach((a: any) => a.show = true)
    }
    fn()
}

// 选择数据库实例
const changeInstance = (inst: any) => {
    state.dbId = inst.id
    state.dbType = inst.type
}
// 选择数据库
const changeSchema = (inst: any, schema: string) => {
    changeInstance(inst)
    state.db = schema
}

// 加载选中的表数据
const loadTableData = async (inst: any, schema: string, tableName: string) => {
    changeSchema(inst, schema)
    await changeTable(tableName)
}

// 新建查询panel
const addQueryTab = async () => {
    let { dbId, db } = state

    if (!db) {
        ElMessage.warning('请选择schema')
        return
    }
    const index = Object.keys(state.queryTabs).length;
    const id = dbId + db + index;
    const name = '查询-' + dbId + db + index
    const queryTab = {
        id: id,
        sql: '',
        label: '查询:' + db,
        name: name,
        // 点击执行按钮执行结果信息
        execRes: {
            total: 0,
            pageNum: 1,
            showPage: false,
            data: [],
            tableColumn: []
        },
        loading: false,
        nowTableName: '', //当前表格数据操作的数据库表名，用于双击编辑表内容使用
        selectionDatas: [],
        updatedFields: [] as UpdateFieldsMeta[],
        editorId: 'MonacoTextarea-' + id,
        dbId: dbId,
        db: db
    }
    state.queryTabs[name] = queryTab

    state.activeName = name;
    state.activeNameMap[name] = { dbId, db }

    getSqlNames(dbId, db)

    await nextTick(() => {
        loadSchemaMeta(dbId, db)
        setTimeout(() => initMonacoEditor(queryTab), 500)
    })

}

const instManage = {
    loadSelectScheme: () => {
        let { dbId, db } = store.state.sqlExecInfo.dbOptInfo;
        if (dbId) {
            state.dbId = dbId;
            state.db = db;
            addQueryTab()
            // fixme 差展开菜单树至对应的db
        }

    }
}

watch(() => store.state.sqlExecInfo.dbOptInfo, () => {
    instManage.loadSelectScheme()
})

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

.editor-move-resize {
    cursor: n-resize;
    height: 3px;
    text-align: center;
}

.el-tabs__header {
    padding: 0 10px;
    background-color: #fff;
}

#data-exec {
    min-height: calc(100vh - 155px);

    .el-tabs__header {
        margin: 0 0 5px;

        .el-tabs__item {
            padding: 0 5px;
        }
    }
}

.update_field_active {
    background-color: var(--el-color-success)
}
</style>
