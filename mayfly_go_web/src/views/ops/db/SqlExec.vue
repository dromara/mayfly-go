<template>
    <div class="db-sql-exec">
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
                <tag-tree ref="tagTreeRef" :loadTags="loadTags" @current-contextmenu-click="onCurrentContextmenuClick" :height="state.tagTreeHeight">
                    <template #prefix="{ data }">
                        <span v-if="data.type.value == SqlExecNodeType.DbInst">
                            <el-popover :show-after="500" placement="right-start" title="数据库实例信息" trigger="hover" :width="210">
                                <template #reference>
                                    <SvgIcon v-if="data.params.type === 'mysql'" name="iconfont icon-op-mysql" :size="18" />
                                    <SvgIcon v-if="data.params.type === 'postgres'" name="iconfont icon-op-postgres" :size="18" />

                                    <SvgIcon name="InfoFilled" v-else />
                                </template>
                                <template #default>
                                    <el-form class="instances-pop-form" label-width="auto" :size="'small'">
                                        <el-form-item label="类型:">{{ data.params.type }}</el-form-item>
                                        <el-form-item label="host:">{{ `${data.params.host}:${data.params.port}` }}</el-form-item>
                                        <el-form-item label="user:">{{ data.params.username }}</el-form-item>
                                        <el-form-item label="名称:">{{ data.params.name }}</el-form-item>
                                        <el-form-item v-if="data.params.remark" label="备注:">{{ data.params.remark }}</el-form-item>
                                    </el-form>
                                </template>
                            </el-popover>
                        </span>

                        <SvgIcon v-if="data.type.value == SqlExecNodeType.Db" name="Coin" color="#67c23a" />

                        <SvgIcon name="Calendar" v-if="data.type.value == SqlExecNodeType.TableMenu" color="#409eff" />

                        <el-tooltip
                            :show-after="500"
                            v-if="data.type.value == SqlExecNodeType.Table"
                            effect="customized"
                            :content="data.params.tableComment"
                            placement="top-end"
                        >
                            <SvgIcon name="Calendar" color="#409eff" />
                        </el-tooltip>

                        <SvgIcon name="Files" v-if="data.type.value == SqlExecNodeType.SqlMenu || data.type.value == SqlExecNodeType.Sql" color="#f56c6c" />
                    </template>

                    <template #suffix="{ data }">
                        <span class="db-table-size" v-if="data.type.value == SqlExecNodeType.Table && data.params.size">{{ ` ${data.params.size}` }}</span>
                        <span class="db-table-size" v-if="data.type.value == SqlExecNodeType.TableMenu && data.params.dbTableSize">{{
                            ` ${data.params.dbTableSize}`
                        }}</span>
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
import { formatByteSize } from '@/common/utils/format';
import { DbInst, TabInfo, TabType, registerDbCompletionItemProvider } from './db';
import { TagTreeNode, NodeType } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { dbApi } from './api';
import { dispposeCompletionItemProvider } from '../../../components/monaco/completionItemProvider';

const Query = defineAsyncComponent(() => import('./component/tab/Query.vue'));
const TableData = defineAsyncComponent(() => import('./component/tab/TableData.vue'));
/**
 * 树节点类型
 */
class SqlExecNodeType {
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

// node节点点击时，触发改变db事件
const changeDb = (nodeData: TagTreeNode) => {
    const params = nodeData.params;
    changeSchema({ id: params.id, name: params.name, type: params.type, tagPath: params.tagPath, databases: params.database }, params.db);
};

// tagpath 节点类型
const NodeTypeTagPath = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const dbInfos = instMap.get(parentNode.key);
    if (!dbInfos) {
        return [];
    }
    return dbInfos?.map((x: any) => {
        return new TagTreeNode(`${parentNode.key}.${x.id}`, x.name, NodeTypeDbInst).withParams(x);
    });
});

// 数据库实例节点类型
const NodeTypeDbInst = new NodeType(SqlExecNodeType.DbInst)
    .withLoadNodesFunc((parentNode: TagTreeNode) => {
        const params = parentNode.params;
        const dbs = params.database.split(' ')?.sort();
        return dbs.map((x: any) => {
            return new TagTreeNode(`${parentNode.key}.${x}`, x, NodeTypeDb).withParams({
                tagPath: params.tagPath,
                id: params.id,
                name: params.name,
                type: params.type,
                dbs: dbs,
                db: x,
            });
        });
    })
    .withNodeClickFunc(changeDb);

// 数据库节点
const NodeTypeDb = new NodeType(SqlExecNodeType.Db)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        return [
            new TagTreeNode(`${params.id}.${params.db}.table-menu`, '表', NodeTypeTableMenu).withParams(params),
            new TagTreeNode(getSqlMenuNodeKey(params.id, params.db), 'SQL', NodeTypeSqlMenu).withParams(params),
        ];
    })
    .withNodeClickFunc(changeDb);

// 数据库表菜单节点
const NodeTypeTableMenu = new NodeType(SqlExecNodeType.TableMenu)
    .withContextMenuItems([{ contextMenuClickId: ContextmenuClickId.ReloadTable, txt: '刷新', icon: 'RefreshRight' }] as any)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        const { id, db } = params;
        // 获取当前库的所有表信息
        let tables = await DbInst.getInst(id).loadTables(db, state.reloadStatus);
        state.reloadStatus = false;
        let dbTableSize = 0;
        const tablesNode = tables.map((x: any) => {
            dbTableSize += x.dataLength + x.indexLength;
            return new TagTreeNode(`${id}.${db}.${x.tableName}`, x.tableName, NodeTypeTable).withIsLeaf(true).withParams({
                id,
                db,
                tableName: x.tableName,
                tableComment: x.tableComment,
                size: formatByteSize(x.dataLength + x.indexLength, 1),
            });
        });
        // 设置父节点参数的表大小
        parentNode.params.dbTableSize = formatByteSize(dbTableSize);
        return tablesNode;
    })
    .withNodeClickFunc(changeDb);

// 数据库sql模板菜单节点
const NodeTypeSqlMenu = new NodeType(SqlExecNodeType.SqlMenu)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        const id = params.id;
        const db = params.db;
        const dbs = params.dbs;
        // 加载用户保存的sql脚本
        const sqls = await dbApi.getSqlNames.request({ id: id, db: db });
        return sqls.map((x: any) => {
            return new TagTreeNode(`${id}.${db}.${x.name}`, x.name, NodeTypeSql).withIsLeaf(true).withParams({
                id,
                db,
                dbs,
                sqlName: x.name,
            });
        });
    })
    .withNodeClickFunc(changeDb);

// 表节点类型
const NodeTypeTable = new NodeType(SqlExecNodeType.Table).withNodeClickFunc((nodeData: TagTreeNode) => {
    const params = nodeData.params;
    loadTableData({ id: params.id, nodeKey: nodeData.key }, params.db, params.tableName);
});

// sql模板节点类型
const NodeTypeSql = new NodeType(SqlExecNodeType.Sql).withNodeClickFunc((nodeData: TagTreeNode) => {
    const params = nodeData.params;
    addQueryTab({ id: params.id, nodeKey: nodeData.key, dbs: params.dbs }, params.db, params.sqlName);
});

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
 * 加载标签树节点
 */
const loadTags = async () => {
    await getInsts();
    const tagPaths = instMap.keys();
    const tagNodes = [];
    for (let tagPath of tagPaths) {
        tagNodes.push(new TagTreeNode(tagPath, tagPath, NodeTypeTagPath));
    }
    return tagNodes;
};

// 当前右击菜单点击事件
const onCurrentContextmenuClick = (clickData: any) => {
    const clickId = clickData.id;
    if (clickId == ContextmenuClickId.ReloadTable) {
        reloadTables(clickData.item.key);
    }
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
    changeSchema(inst, db);

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

    // 注册当前sql编辑框提示词
    registerDbCompletionItemProvider('sql', tab.dbId, tab.db, tab.params.dbs);
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
.db-sql-exec {
    .db-table-size {
        color: #c4c9c4;
        font-size: 9px;
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
}
</style>
