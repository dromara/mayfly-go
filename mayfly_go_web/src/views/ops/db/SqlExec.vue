<template>
    <div>
        <el-row type="flex">
            <el-col :span="24">
                <el-button type="primary" icon="plus"
                    @click="addQueryTab({ id: state.dbId, type: state.dbType }, state.db)"
                    size="small">新建查询</el-button>
            </el-col>
            <el-col :span="4" style="border-left: 1px solid #eee; margin-top: 10px">
                <InstanceTree ref="instanceTreeRef" @change-instance="changeInstance" @change-schema="changeSchema"
                    @clickSqlName="onClickSqlName" @clickSchemaTable="loadTableData" />
            </el-col>
            <el-col :span="20">
                <el-container id="data-exec" style="border-left: 1px solid #eee; margin-top: 10px">
                    <el-tabs @tab-remove="remoteTab" @tab-click="onDataTabClick" style="width: 100%"
                        v-model="state.activeName">

                        <el-tab-pane closable v-for="dt in state.tabs.values()" :key="dt.key" :label="dt.key"
                            :name="dt.key">
                            <table-data v-if="dt.type === TabType.TableData" @gen-insert-sql="onGenerateInsertSql"
                                :data="dt" :table-height="state.dataTabsTableHeight"></table-data>

                            <query v-else @save-sql-success="reloadSqls" @delete-sql-success="deleteSqlScript(dt)"
                                :data="dt" :editor-height="state.editorHeight">
                            </query>
                        </el-tab-pane>
                    </el-tabs>
                </el-container>
            </el-col>
        </el-row>

        <el-dialog @close="state.genSqlDialog.visible = false" v-model="state.genSqlDialog.visible" title="SQL"
            width="1000px">
            <el-input v-model="state.genSqlDialog.sql" type="textarea" rows="20" />
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, Ref } from 'vue';
import { ElMessage } from 'element-plus';

import { language as sqlLanguage } from 'monaco-editor/esm/vs/basic-languages/mysql/mysql.js';
import * as monaco from 'monaco-editor';
import { editor, languages, Position } from 'monaco-editor';
import InstanceTree from '@/views/ops/db/component/InstanceTree.vue';

import { DbInst, TabInfo, TabType } from './db'
import TableData from './component/tab/TableData.vue'
import Query from './component/tab/Query.vue'

const instanceTreeRef = ref(null) as Ref;

const tabs: Map<string, TabInfo> = new Map();
const state = reactive({
    dbId: null, // 当前选中操作的数据库实例
    dbType: '',
    db: '', // 当前操作的数据库
    activeName: 'Query',
    tabs,
    dataTabsTableHeight: '600',
    editorHeight: '600',
    genSqlDialog: {
        visible: false,
        sql: '',
    },
});

onMounted(() => {
    self.completionItemProvider?.dispose()
    setHeight();
    // 监听浏览器窗口大小变化,更新对应组件高度
    window.onresize = () => setHeight();
});

/**
 * 设置editor高度和数据表高度
 */
const setHeight = () => {
    // 默认300px
    // state.monacoOptions.height = window.innerHeight - 518 + 'px'
    state.editorHeight = window.innerHeight - 518 + 'px';
    state.dataTabsTableHeight = window.innerHeight - 219 - 36 + 'px';
};

// 选择数据库实例
const changeInstance = (inst: any, fn?: Function) => {
    state.dbId = inst.id
    state.dbType = inst.type
    fn && fn()
}

// 选择数据库
const changeSchema = (inst: any, schema: string) => {
    changeInstance(inst)
    state.db = schema
}

// 加载选中的表数据，即新增表数据操作tab
const loadTableData = async (inst: any, schema: string, tableName: string) => {
    changeSchema(inst, schema)
    if (tableName == '') {
        return;
    }

    const label = `${inst.id}:\`${schema}\`.${tableName}`;
    let tab = state.tabs.get(label);
    state.activeName = label;
    // 如果存在该表tab，则直接返回
    if (tab) {
        return;
    }
    tab = new TabInfo();
    tab.key = label;
    tab.dbId = inst.id;
    tab.dbType = inst.type;
    tab.db = schema;
    tab.type = TabType.TableData;
    tab.other = {
        table: tableName
    }
    state.tabs.set(label, tab)
}

// 新建查询panel
const addQueryTab = async (inst: any, db: string, sqlName: string = '') => {
    if (!db) {
        ElMessage.warning('请选择schema')
        return
    }
    const dbId = inst.id;
    let label;
    // 存在sql模板名，则该模板名只允许一个tab
    if (sqlName) {
        label = `查询:${dbId}:${db}.${sqlName}`;
    } else {
        let count = 1;
        state.tabs.forEach((v) => {
            if (v.type == TabType.Query && !v.other.sqlName) {
                count++;
            }
        })
        label = `新查询${count}:${dbId}:${db}`;
    }
    state.activeName = label;
    let tab = state.tabs.get(label);
    if (tab) {
        return;
    }
    tab = new TabInfo();
    tab.key = label;
    tab.dbId = dbId;
    tab.dbType = inst.type;
    tab.db = db;
    tab.type = TabType.Query;
    tab.other = {
        sqlName: sqlName,
        dbs: instanceTreeRef.value.getSchemas(dbId)
    }
    state.tabs.set(label, tab)
    registerSqlCompletionItemProvider();
}

const remoteTab = (targetName: string) => {
    let activeName = state.activeName;
    const tabNames = [...state.tabs.keys()]
    for (let i = 0; i < tabNames.length; i++) {
        const tabName = tabNames[i]
        if (tabName !== targetName) {
            continue;
        }
        const nextTab = tabNames[i + 1] || tabNames[i - 1];
        if (nextTab) {
            activeName = nextTab;
        }
        state.tabs.delete(targetName);
        state.activeName = activeName;
        break;
    }
};

/**
 * 数据tab点击
 */
const onDataTabClick = (tab: any) => {
    state.activeName = tab.props.name;
};

const onGenerateInsertSql = async (sql: string) => {
    state.genSqlDialog.sql = sql;
    state.genSqlDialog.visible = true;
};

const onClickSqlName = (inst: any, schema: string, sqlName: string) => {
    addQueryTab(inst, schema, sqlName);
}

const reloadSqls = (dbId: number, db: string) => {
    instanceTreeRef.value.reloadSqls({ id: dbId }, db);
}

const deleteSqlScript = (ti: TabInfo) => {
    instanceTreeRef.value.reloadSqls({ id: ti.dbId }, ti.db);
    remoteTab(ti.key);
}

const registerSqlCompletionItemProvider = () => {
    // 参考 https://microsoft.github.io/monaco-editor/playground.html#extending-language-services-completion-provider-example
    self.completionItemProvider = self.completionItemProvider || monaco.languages.registerCompletionItemProvider('sql', {
        triggerCharacters: ['.'],
        provideCompletionItems: async (model: editor.ITextModel, position: Position): Promise<languages.CompletionList | null | undefined> => {
            let word = model.getWordUntilPosition(position);
            const nowTab = state.tabs.get(state.activeName);
            if (!nowTab) {
                return;
            }
            const { db, dbId, dbType } = nowTab;
            const dbInst = DbInst.getInst(dbId, dbType);
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

            const dbs = nowTab.other && nowTab.other.dbs;
            // console.log("光标前文本：=>" + textBeforePointerMulti)

            // console.log("最后输入的：=>" + lastToken)
            if (lastToken.endsWith('.')) {
                // 如果是.触发代码提示，则进行【 库.表名联想 】 或 【 表别名.表字段联想 】
                let str = lastToken.substring(0, lastToken.lastIndexOf('.'))
                // 库.表名联想
                if (dbs.indexOf(str) > -1) {
                    let tables = await dbInst.loadTables(str)
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
                    let db = tableInfo.dbName;
                    // // 取出表名并提示
                    // let dbs = state.monacoOptions.dbTables[dbId + db]
                    // let columns = dbs ? (dbs[table] || []) : [];
                    // if ((!columns || columns.length === 0) && db) {
                    //     state.monacoOptions.dbTables[dbId + db] = await loadHintTables(dbId, db)
                    //     dbs = state.monacoOptions.dbTables[dbId + db]
                    //     columns = dbs ? (dbs[table] || []) : [];
                    // }
                    // 取出表名并提示
                    let dbHits = await dbInst.loadDbHints(db)
                    let columns = dbHits[table]
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
            dbs.forEach((a: string) => {
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

            const tables = await dbInst.loadTables(db);
            // 表名联想
            tables.forEach((tableMeta: any) => {
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
}

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
