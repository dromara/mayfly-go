<template>
    <div>
        <el-row class="mb5">
            <el-col :span="4">
                <el-button
                    :disabled="!state.db || !nowDbInst.id"
                    type="primary"
                    icon="plus"
                    @click="addQueryTab({ id: nowDbInst.id, dbs: nowDbInst.databases?.split(' ') }, state.db)"
                    size="small"
                    >新建查询</el-button
                >
            </el-col>
            <el-col :span="20" v-if="state.db">
                <el-descriptions :column="4" size="small" border style="height: 10px" class="ml5">
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
            <el-col :span="4">
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
                                        <el-form-item label="名称:">{{ data.params.name }}</el-form-item>
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
                <el-container id="data-exec" class="mt5 ml5">
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
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, onBeforeUnmount } from 'vue';
import { ElMessage } from 'element-plus';

import { DbInst, TabInfo, TabType, registerDbCompletionItemProvider } from './db';
import { TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { dbApi } from './api';
import { dispposeCompletionItemProvider } from '../../../components/monaco/completionItemProvider';

const Query = defineAsyncComponent(() => import('./component/tab/Query.vue'));
const TableData = defineAsyncComponent(() => import('./component/tab/TableData.vue'));
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
    setHeight();
    // 监听浏览器窗口大小变化,更新对应组件高度
    window.onresize = () => setHeight();
});

onBeforeUnmount(() => {
    dispposeCompletionItemProvider('sql');
});

/**
 * 设置editor高度和数据表高度
 */
const setHeight = () => {
    state.editorHeight = window.innerHeight - 518 + 'px';
    state.dataTabsTableHeight = window.innerHeight - 256 + 'px';
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

    if (nowTab?.type == TabType.Query) {
        // 注册sql提示
        registerDbCompletionItemProvider('sql', nowTab.dbId, nowTab.db, nowTab.params.dbs);
    }
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
