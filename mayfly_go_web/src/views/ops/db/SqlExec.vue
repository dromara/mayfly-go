<template>
    <div>
        <el-row>
            <el-col :span="4">
                <el-button type="primary" icon="plus" @click="addQueryTab({ id: nowDbInst.id, dbs: nowDbInst.databases }, state.db)" size="small"
                    >新建查询</el-button
                >
            </el-col>
            <el-col :span="20" v-if="state.db">
                <el-descriptions :column="4" size="small" border style="height: 10px">
                    <el-descriptions-item label-align="right" label="tag">{{ nowDbInst.tagPath }}</el-descriptions-item>

                    <el-descriptions-item label="实例" label-align="right">
                        {{ nowDbInst.id }}
                        <el-divider direction="vertical" border-style="dashed" />
                        {{ nowDbInst.type }}
                        <el-divider direction="vertical" border-style="dashed" />
                        {{ nowDbInst.name }}
                    </el-descriptions-item>

                    <el-descriptions-item label="库名" label-align="right">{{ state.db }}</el-descriptions-item>
                </el-descriptions>
            </el-col>
        </el-row>
        <el-row type="flex">
            <el-col :span="4" style="border-left: 1px solid #eee; margin-top: 10px">
                <tag-tree
                    ref="tagTreeRef"
                    @node-click="nodeClick"
                    :load="loadNode"
                    :load-contextmenu-items="getContextmenuItems"
                    @current-contextmenu-click="onCurrentContextmenuClick"
                    :height="state.tagTreeHeight"
                >
                    <template #prefix="{ data }">
                        <span v-if="data.type == NodeType.DbInst">
                            <el-popover placement="right-start" title="数据库实例信息" trigger="hover" :width="210">
                                <template #reference>
                                    <SvgIcon v-if="data.params.type === 'mysql'" name="iconfont icon-op-mysql" :size="18" />
                                    <SvgIcon v-if="data.params.type === 'postgres'" name="iconfont icon-op-postgres" :size="18" />

                                    <SvgIcon name="InfoFilled" v-else />
                                </template>
                                <template #default>
                                    <el-form class="instances-pop-form" label-width="55px" :size="'small'">
                                        <el-form-item label="类型:">{{ data.params.type }}</el-form-item>
                                        <el-form-item label="链接:">{{ data.params.host }}:{{ data.params.port }}</el-form-item>
                                        <el-form-item label="用户:">{{ data.params.username }}</el-form-item>
                                        <el-form-item v-if="data.params.remark" label="备注:">{{ data.params.remark }}</el-form-item>
                                    </el-form>
                                </template>
                            </el-popover>
                        </span>

                        <SvgIcon v-if="data.type == NodeType.Db" name="Coin" color="#67c23a" />

                        <SvgIcon name="Calendar" v-if="data.type == NodeType.TableMenu" color="#409eff" />

                        <el-tooltip v-if="data.type == NodeType.Table" effect="customized" :content="data.params.tableComment" placement="top-end">
                            <SvgIcon name="Calendar" color="#409eff" />
                        </el-tooltip>

                        <SvgIcon name="Files" v-if="data.type == NodeType.SqlMenu || data.type == NodeType.Sql" color="#f56c6c" />
                    </template>
                </tag-tree>
            </el-col>
            <el-col :span="20">
                <el-container id="data-exec" style="border-left: 1px solid #eee; margin-top: 10px">
                    <el-tabs @tab-remove="onRemoveTab" @tab-change="onTabChange" style="width: 100%" v-model="state.activeName">
                        <el-tab-pane closable v-for="dt in state.tabs.values()" :key="dt.key" :label="dt.key" :name="dt.key">
                            <table-data
                                v-if="dt.type === TabType.TableData"
                                @gen-insert-sql="onGenerateInsertSql"
                                :data="dt"
                                :table-height="state.dataTabsTableHeight"
                            ></table-data>

                            <query
                                v-else
                                @save-sql-success="reloadSqls"
                                @delete-sql-success="deleteSqlScript(dt)"
                                :data="dt"
                                :editor-height="state.editorHeight"
                            >
                            </query>
                        </el-tab-pane>
                    </el-tabs>
                </el-container>
            </el-col>
        </el-row>

        <el-dialog @close="state.genSqlDialog.visible = false" v-model="state.genSqlDialog.visible" title="SQL" width="1000px">
            <el-input v-model="state.genSqlDialog.sql" type="textarea" rows="20" />
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, onMounted, reactive, ref, toRefs } from 'vue';
import { ElMessage } from 'element-plus';

import { language as sqlLanguage } from 'monaco-editor/esm/vs/basic-languages/mysql/mysql.js';
import { language as addSqlLanguage } from './lang/mysql.js';
import * as monaco from 'monaco-editor';
import { editor, languages, Position } from 'monaco-editor';

import { DbInst, TabInfo, TabType } from './db';
import { TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { dbApi } from './api';

const Query = defineAsyncComponent(() => import('./component/tab/Query.vue'));
const TableData = defineAsyncComponent(() => import('./component/tab/TableData.vue'));

const sqlCompletionKeywords = [...sqlLanguage.keywords, ...addSqlLanguage.keywords];
const sqlCompletionOperators = [...sqlLanguage.operators, ...addSqlLanguage.operators];
const sqlCompletionBuiltinFunctions = [...sqlLanguage.builtinFunctions, ...addSqlLanguage.builtinFunctions];
const sqlCompletionBuiltinVariables = [...sqlLanguage.builtinVariables, ...addSqlLanguage.builtinVariables];
/**
 * 树节点类型
 */
class NodeType {
    static DbInst = 1;
    static Db = 2;
    static TableMenu = 3;
    static SqlMenu = 4;
    static Table = 5;
    static Sql = 6;
}
class ContextmenuClickId {
    static ReloadTable = 0;
}

const tagTreeRef: any = ref(null);

const tabs: Map<string, TabInfo> = new Map();
const state = reactive({
    /**
     * 当前操作的数据库实例
     */
    nowDbInst: {} as DbInst,
    db: '', // 当前操作的数据库
    activeName: '',
    reloadStatus: false,
    tabs,
    dataTabsTableHeight: '600',
    editorHeight: '600',
    tagTreeHeight: window.innerHeight - 178 + 'px',
    genSqlDialog: {
        visible: false,
        sql: '',
    },
});

const { nowDbInst } = toRefs(state);

onMounted(() => {
    self.completionItemProvider?.dispose();
    setHeight();
    // 监听浏览器窗口大小变化,更新对应组件高度
    window.onresize = () => setHeight();
});

/**
 * 设置editor高度和数据表高度
 */
const setHeight = () => {
    state.editorHeight = window.innerHeight - 518 + 'px';
    state.dataTabsTableHeight = window.innerHeight - 219 - 36 + 'px';
    state.tagTreeHeight = window.innerHeight - 165 + 'px';
};

/**
 * instmap; tagPaht -> info[]
 */
const instMap: Map<string, any[]> = new Map();

const getInsts = async () => {
    const res = await dbApi.dbs.request({ pageNum: 1, pageSize: 1000 });
    if (!res.total) return;
    for (const db of res.list) {
        const tagPath = db.tagPath;
        let dbInsts = instMap.get(tagPath) || [];
        dbInsts.push(db);
        instMap.set(tagPath, dbInsts?.sort());
    }
};

/**
 * 加载树节点
 * @param {Object} node
 * @param {Object} resolve
 */
const loadNode = async (node: any) => {
    // 一级为tagPath
    if (node.level === 0) {
        await getInsts();
        const tagPaths = instMap.keys();
        const tagNodes = [];
        for (let tagPath of tagPaths) {
            tagNodes.push(new TagTreeNode(tagPath, tagPath));
        }
        return tagNodes;
    }

    const data = node.data;
    const nodeType = data.type;
    const params = data.params;

    // 点击tagPath -> 加载数据库实例信息列表
    if (nodeType === TagTreeNode.TagPath) {
        const dbInfos = instMap.get(data.key);
        return dbInfos?.map((x: any) => {
            return new TagTreeNode(`${data.key}.${x.id}`, x.name, NodeType.DbInst).withParams(x);
        });
    }

    // 点击数据库实例 -> 加载库列表
    if (nodeType === NodeType.DbInst) {
        const dbs = params.database.split(' ')?.sort();
        return dbs.map((x: any) => {
            return new TagTreeNode(`${data.key}.${x}`, x, NodeType.Db).withParams({
                tagPath: params.tagPath,
                id: params.id,
                name: params.name,
                type: params.type,
                dbs: dbs,
                db: x,
            });
        });
    }

    // 点击数据库 -> 加载 表&Sql 菜单
    if (nodeType === NodeType.Db) {
        return [
            new TagTreeNode(`${params.id}.${params.db}.table-menu`, '表', NodeType.TableMenu).withParams(params),
            new TagTreeNode(getSqlMenuNodeKey(params.id, params.db), 'SQL', NodeType.SqlMenu).withParams(params),
        ];
    }

    // 点击表菜单 -> 加载表列表
    if (nodeType === NodeType.TableMenu) {
        return await getTables(params);
    }

    if (nodeType === NodeType.SqlMenu) {
        return await loadSqls(params.id, params.db, params.dbs);
    }

    return [];
};

const nodeClick = async (data: any) => {
    const params = data.params;
    const nodeKey = data.key;
    const dataType = data.type;
    // 点击数据库，修改当前数据库信息
    if (dataType === NodeType.Db || dataType === NodeType.SqlMenu || dataType === NodeType.TableMenu || dataType === NodeType.DbInst) {
        changeSchema({ id: params.id, name: params.name, type: params.type, tagPath: params.tagPath, databases: params.database }, params.db);
        return;
    }

    // 点击表加载表数据tab
    if (dataType === NodeType.Table) {
        await loadTableData({ id: params.id, nodeKey: nodeKey }, params.db, params.tableName);
        return;
    }

    // 点击表加载表数据tab
    if (dataType === NodeType.Sql) {
        await addQueryTab({ id: params.id, nodeKey: nodeKey, dbs: params.dbs }, params.db, params.sqlName);
    }
};

const getContextmenuItems = (data: any) => {
    const dataType = data.type;
    if (dataType === NodeType.TableMenu) {
        return [{ contextMenuClickId: ContextmenuClickId.ReloadTable, txt: '刷新', icon: 'RefreshRight' }];
    }
    return [];
};

// 当前右击菜单点击事件
const onCurrentContextmenuClick = (clickData: any) => {
    const clickId = clickData.id;
    if (clickId == ContextmenuClickId.ReloadTable) {
        reloadTables(clickData.item.key);
    }
};

const getTables = async (params: any) => {
    const { id, db } = params;
    let tables = await DbInst.getInst(id).loadTables(db, state.reloadStatus);
    state.reloadStatus = false;
    return tables.map((x: any) => {
        return new TagTreeNode(`${id}.${db}.${x.tableName}`, x.tableName, NodeType.Table).withIsLeaf(true).withParams({
            id,
            db,
            tableName: x.tableName,
            tableComment: x.tableComment,
        });
    });
};

/**
 * 加载用户保存的sql脚本
 *
 * @param inst
 * @param schema
 */
const loadSqls = async (id: any, db: string, dbs: any) => {
    const sqls = await dbApi.getSqlNames.request({ id: id, db: db });
    return sqls.map((x: any) => {
        return new TagTreeNode(`${id}.${db}.${x.name}`, x.name, NodeType.Sql).withIsLeaf(true).withParams({
            id,
            db,
            dbs,
            sqlName: x.name,
        });
    });
};

// 选择数据库
const changeSchema = (inst: any, schema: string) => {
    state.nowDbInst = DbInst.getOrNewInst(inst);
    state.db = schema;
};

// 加载选中的表数据，即新增表数据操作tab
const loadTableData = async (inst: any, schema: string, tableName: string) => {
    changeSchema(inst, schema);
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
    tab.treeNodeKey = inst.nodeKey;
    tab.dbId = inst.id;
    tab.db = schema;
    tab.type = TabType.TableData;
    tab.params = {
        table: tableName,
    };
    state.tabs.set(label, tab);
};

// 新建查询panel
const addQueryTab = async (inst: any, db: string, sqlName: string = '') => {
    if (!db || !inst.id) {
        ElMessage.warning('请选择数据库实例及对应的schema');
        return;
    }

    const dbId = inst.id;
    let label;
    // 存在sql模板名，则该模板名只允许一个tab
    if (sqlName) {
        label = `查询:${dbId}:${db}.${sqlName}`;
    } else {
        let count = 1;
        state.tabs.forEach((v) => {
            if (v.type == TabType.Query && !v.params.sqlName) {
                count++;
            }
        });
        label = `新查询${count}:${dbId}:${db}`;
    }
    state.activeName = label;
    let tab = state.tabs.get(label);
    if (tab) {
        return;
    }
    tab = new TabInfo();
    tab.key = label;
    tab.treeNodeKey = inst.nodeKey;
    tab.dbId = dbId;
    tab.db = db;
    tab.type = TabType.Query;
    tab.params = {
        sqlName: sqlName,
        dbs: inst.dbs,
    };
    state.tabs.set(label, tab);
    registerSqlCompletionItemProvider();
};

const onRemoveTab = (targetName: string) => {
    let activeName = state.activeName;
    const tabNames = [...state.tabs.keys()];
    for (let i = 0; i < tabNames.length; i++) {
        const tabName = tabNames[i];
        if (tabName !== targetName) {
            continue;
        }
        const nextTab = tabNames[i + 1] || tabNames[i - 1];
        if (nextTab) {
            activeName = nextTab;
        } else {
            activeName = '';
        }
        state.tabs.delete(targetName);
        state.activeName = activeName;
    }
};

const onTabChange = () => {
    if (!state.activeName) {
        state.nowDbInst = {} as DbInst;
        state.db = '';
        return;
    }
    const nowTab = state.tabs.get(state.activeName);
    state.nowDbInst = DbInst.getInst(nowTab?.dbId);
    state.db = nowTab?.db as string;
};

const onGenerateInsertSql = async (sql: string) => {
    state.genSqlDialog.sql = sql;
    state.genSqlDialog.visible = true;
};

const reloadSqls = (dbId: number, db: string) => {
    tagTreeRef.value.reloadNode(getSqlMenuNodeKey(dbId, db));
};

const deleteSqlScript = (ti: TabInfo) => {
    reloadSqls(ti.dbId, ti.db);
    onRemoveTab(ti.key);
};

const getSqlMenuNodeKey = (dbId: number, db: string) => {
    return `${dbId}.${db}.sql-menu`;
};

const reloadTables = (nodeKey: string) => {
    state.reloadStatus = true;
    tagTreeRef.value.reloadNode(nodeKey);
};

const registerSqlCompletionItemProvider = () => {
    // 参考 https://microsoft.github.io/monaco-editor/playground.html#extending-language-services-completion-provider-example
    self.completionItemProvider =
        self.completionItemProvider ||
        monaco.languages.registerCompletionItemProvider('sql', {
            triggerCharacters: ['.', ' '],
            provideCompletionItems: async (model: editor.ITextModel, position: Position): Promise<languages.CompletionList | null | undefined> => {
                let word = model.getWordUntilPosition(position);
                const nowTab = state.tabs.get(state.activeName);
                if (!nowTab) {
                    return;
                }
                const { db, dbId } = nowTab;
                const dbInst = DbInst.getInst(dbId);
                const { lineNumber, column } = position;
                const { startColumn, endColumn } = word;

                // 当前行文本
                let lineContent = model.getLineContent(lineNumber);
                // 注释行不需要代码提示
                if (lineContent.startsWith('--')) {
                    return { suggestions: [] };
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
                    endColumn: column,
                });
                const textBeforePointerMulti = model.getValueInRange({
                    startLineNumber: 1,
                    startColumn: 0,
                    endLineNumber: lineNumber,
                    endColumn: column,
                });
                // 光标后文本
                const textAfterPointerMulti = model.getValueInRange({
                    startLineNumber: lineNumber,
                    startColumn: column,
                    endLineNumber: model.getLineCount(),
                    endColumn: model.getLineMaxColumn(model.getLineCount()),
                });
                // // const nextTokens = textAfterPointer.trim().split(/\s+/)
                // // const nextToken = nextTokens[0].toLowerCase()
                const tokens = textBeforePointer.trim().split(/\s+/);
                let lastToken = tokens[tokens.length - 1].toLowerCase();
                const secondToken = (tokens.length > 2 && tokens[tokens.length - 2].toLowerCase()) || '';

                // const dbs = nowTab.params?.dbs?.split(' ') || [];
                const dbs = (nowTab.params && nowTab.params.dbs && nowTab.params.dbs.split(' ')) || [];
                // console.log("光标前文本：=>" + textBeforePointerMulti)
                // console.log("最后输入的：=>" + lastToken)

                let suggestions: languages.CompletionItem[] = [];
                const tables = await dbInst.loadTables(db);

                async function hintTableColumns(tableName: any, db: any) {
                    let dbHits = await dbInst.loadDbHints(db);
                    let columns = dbHits[tableName];
                    let suggestions: languages.CompletionItem[] = [];
                    columns?.forEach((a: string, index: any) => {
                        // 字段数据格式  字段名 字段注释，  如： create_time  [datetime][创建时间]
                        const nameAndComment = a.split('  ');
                        const fieldName = nameAndComment[0];
                        suggestions.push({
                            label: {
                                label: a,
                                description: 'column',
                            },
                            kind: monaco.languages.CompletionItemKind.Property,
                            detail: '', // 不显示detail, 否则选中时备注等会被遮挡
                            insertText: fieldName, // create_time
                            range,
                            sortText: 100 + index + '', // 使用表字段声明顺序排序,排序需为字符串类型
                        });
                    });
                    return suggestions;
                }

                if (lastToken.indexOf('.') > -1 || secondToken.indexOf('.') > -1) {
                    // 如果是.触发代码提示，则进行【 库.表名联想 】 或 【 表别名.表字段联想 】
                    let str = lastToken.substring(0, lastToken.lastIndexOf('.'));
                    if (lastToken.trim().startsWith('.')) {
                        str = secondToken;
                    }
                    // 如果字符串粘连起了如:'a.creator,a.',需要重新取出别名
                    let aliasArr = lastToken.split(',');
                    if (aliasArr.length > 1) {
                        lastToken = aliasArr[aliasArr.length - 1];
                        str = lastToken.substring(0, lastToken.lastIndexOf('.'));
                        if (lastToken.trim().startsWith('.')) {
                            str = secondToken;
                        }
                    }
                    // 库.表名联想
                    if (dbs && dbs.filter((a: any) => a === str)?.length > 0) {
                        let tables = await dbInst.loadTables(str);
                        let suggestions: languages.CompletionItem[] = [];
                        for (let item of tables) {
                            const { tableName, tableComment } = item;
                            suggestions.push({
                                label: {
                                    label: tableName + (tableComment ? ' - ' + tableComment : ''),
                                    description: 'table',
                                },
                                kind: monaco.languages.CompletionItemKind.File,
                                insertText: tableName,
                                range,
                            });
                        }
                        return { suggestions };
                    }

                    let sql = textBeforePointerMulti.split(';')[textBeforePointerMulti.split(';').length - 1] + textAfterPointerMulti.split(';')[0];
                    // 表别名.表字段联想
                    let tableInfo = getTableByAlias(sql, db, str);
                    if (tableInfo.tableName) {
                        let tableName = tableInfo.tableName;
                        let db = tableInfo.dbName;
                        // 取出表名并提示
                        let suggestions = await hintTableColumns(tableName, db);
                        if (suggestions.length > 0) {
                            return { suggestions };
                        }
                    }
                    return { suggestions: [] };
                } else {
                    // 如果sql里含有表名，则提示表字段
                    let mat = textBeforePointerMulti.match(/[from|update]\n*\s+\n*(\w+)\n*\s+\n*/i);
                    if (mat && mat.length > 1) {
                        let tableName = mat[1];
                        // 取出表名并提示
                        let addSuggestions = await hintTableColumns(tableName, db);
                        if (addSuggestions.length > 0) {
                            suggestions = suggestions.concat(addSuggestions);
                        }
                    }
                }

                // 表名联想
                tables.forEach((tableMeta: any) => {
                    const { tableName, tableComment } = tableMeta;
                    suggestions.push({
                        label: {
                            label: tableName + ' - ' + tableComment,
                            description: 'table',
                        },
                        kind: monaco.languages.CompletionItemKind.File,
                        detail: tableComment,
                        insertText: tableName + ' ',
                        range,
                    });
                });

                // mysql关键字
                sqlCompletionKeywords.forEach((item: any) => {
                    suggestions.push({
                        label: {
                            label: item,
                            description: 'keyword',
                        },
                        kind: monaco.languages.CompletionItemKind.Keyword,
                        insertText: item,
                        range,
                    });
                });

                // 操作符
                sqlCompletionOperators.forEach((item: any) => {
                    suggestions.push({
                        label: {
                            label: item,
                            description: 'opt',
                        },
                        kind: monaco.languages.CompletionItemKind.Operator,
                        insertText: item,
                        range,
                    });
                });
                
                let replacedFunctions = [] as string[]; 

                // 添加的函数
                addSqlLanguage.replaceFunctions.forEach((item: any) => {
                    replacedFunctions.push(item.label)
                    suggestions.push({
                        label: {
                            label: item.label,
                            description: item.description,
                        },
                        kind: monaco.languages.CompletionItemKind.Function,
                        insertText: item.insertText,
                        range,
                    });
                });
              
                // 内置函数
                sqlCompletionBuiltinFunctions.forEach((item: any) => {
                  replacedFunctions.indexOf(item) < 0 && suggestions.push({
                        label: {
                            label: item,
                            description: 'func',
                        },
                        kind: monaco.languages.CompletionItemKind.Function,
                        insertText: item,
                        range,
                    });
                });
                // 内置变量
                sqlCompletionBuiltinVariables.forEach((item: string) => {
                    suggestions.push({
                        label: {
                            label: item,
                            description: 'var',
                        },
                        kind: monaco.languages.CompletionItemKind.Variable,
                        insertText: item,
                        range,
                    });
                });

                // 库名提示
                if (dbs && dbs.length > 0) {
                    dbs.forEach((a: any) => {
                        suggestions.push({
                            label: {
                                label: a,
                                description: 'schema',
                            },
                            kind: monaco.languages.CompletionItemKind.Folder,
                            insertText: a,
                            range,
                        });
                    });
                }

                // 默认提示
                return {
                    suggestions: suggestions,
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
const getTableByAlias = (sql: string, db: string, alias: string): { dbName: string; tableName: string } => {
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
    let match = sql.match(/(join|from)\n*\s+\n*(\w*-?\w*\.?\w+)\s*(as)?\s*(\w*)\n*/gi);
    if (match && match.length > 0) {
        match.forEach((a) => {
            // 去掉前缀，取出
            let t = a
                .substring(5, a.length)
                .replaceAll(/\s+/g, ' ')
                .replaceAll(/\s+as\s+/gi, ' ')
                .replaceAll(/\r\n/g, ' ')
                .trim()
                .split(/\s+/);
            let withDb = t[0].split('.');
            // 表名是 db名.表名
            let tName = withDb.length > 1 ? withDb[1] : withDb[0];
            let dbName = withDb.length > 1 ? withDb[0] : db || '';
            if (t.length == 2) {
                // 表别名：表名
                result[t[1]] = { tableName: tName, dbName };
            } else {
                // 只有表名无别名 取第一个无别名的表为默认表
                !defName && (defResult = { tableName: tName, dbName: db });
            }
        });
    }
    return result[alias] || defResult;
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
    background-color: var(--el-color-success);
}

.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}
</style>
