<template>
    <div class="db-sql-exec">
        <Splitpanes class="default-theme">
            <Pane size="20" max-size="30">
                <tag-tree :resource-type="TagResourceTypeEnum.Db.value" :tag-path-node-type="NodeTypeTagPath" ref="tagTreeRef">
                    <template #prefix="{ data }">
                        <span v-if="data.type.value == SqlExecNodeType.DbInst">
                            <el-popover
                                @show="showDbInfo(data.params)"
                                :show-after="500"
                                placement="right-start"
                                title="数据库实例信息"
                                trigger="hover"
                                :width="250"
                            >
                                <template #reference>
                                    <SvgIcon :name="getDbDialect(data.params.type).getInfo().icon" :size="18" />
                                </template>
                                <template #default>
                                    <el-descriptions :column="1" size="small">
                                        <el-descriptions-item label="名称">
                                            {{ data.params.name }}
                                        </el-descriptions-item>
                                        <el-descriptions-item label="host">
                                            {{ `${data.params.host}:${data.params.port}` }}
                                        </el-descriptions-item>
                                        <el-descriptions-item label="数据库版本">
                                            <span v-loading="loadingServerInfo"> {{ `${dbServerInfo?.version}` }}</span>
                                        </el-descriptions-item>
                                        <el-descriptions-item label="user">
                                            {{ data.params.username }}
                                        </el-descriptions-item>
                                        <el-descriptions-item label="备注">
                                            {{ data.params.remark }}
                                        </el-descriptions-item>
                                    </el-descriptions>
                                </template>
                            </el-popover>
                        </span>

                        <SvgIcon v-if="data.icon" :name="data.icon.name" :color="data.icon.color" />
                    </template>

                    <template #suffix="{ data }">
                        <span class="db-table-size" v-if="data.type.value == SqlExecNodeType.Table && data.params.size">{{ ` ${data.params.size}` }}</span>
                        <span class="db-table-size" v-if="data.type.value == SqlExecNodeType.TableMenu && data.params.dbTableSize">{{
                            ` ${data.params.dbTableSize}`
                        }}</span>
                    </template>
                </tag-tree>
            </Pane>

            <Pane>
                <div class="card db-op pd5">
                    <el-row>
                        <el-col :span="24" v-if="state.db">
                            <el-descriptions :column="4" size="small" border>
                                <el-descriptions-item label-align="right" label="操作"
                                    ><el-button
                                        :disabled="!state.db || !nowDbInst.id"
                                        type="primary"
                                        icon="Search"
                                        @click="addQueryTab({ id: nowDbInst.id, dbs: nowDbInst.databases }, state.db)"
                                        size="small"
                                        >新建查询</el-button
                                    ></el-descriptions-item
                                >

                                <el-descriptions-item label-align="right" label="tag">{{ nowDbInst.tagPath }}</el-descriptions-item>

                                <el-descriptions-item label-align="right">
                                    <template #label>
                                        <div>
                                            <SvgIcon :name="nowDbInst.getDialect().getInfo().icon" :size="18" />
                                            实例
                                        </div>
                                    </template>
                                    {{ nowDbInst.id }}
                                    <el-divider direction="vertical" border-style="dashed" />
                                    {{ nowDbInst.name }}
                                    <el-divider direction="vertical" border-style="dashed" />
                                    {{ nowDbInst.host }}
                                </el-descriptions-item>

                                <el-descriptions-item label="库名" label-align="right">{{ state.db }}</el-descriptions-item>
                            </el-descriptions>
                        </el-col>
                    </el-row>

                    <div id="data-exec" class="mt5">
                        <el-tabs
                            v-if="state.tabs.size > 0"
                            type="card"
                            @tab-remove="onRemoveTab"
                            @tab-change="onTabChange"
                            style="width: 100%"
                            v-model="state.activeName"
                            class="h100"
                        >
                            <el-tab-pane class="h100" closable v-for="dt in state.tabs.values()" :label="dt.label" :name="dt.key" :key="dt.key">
                                <template #label>
                                    <el-popover :show-after="1000" placement="bottom-start" trigger="hover" :width="250">
                                        <template #reference> {{ dt.label }} </template>
                                        <template #default>
                                            <el-descriptions :column="1" size="small">
                                                <el-descriptions-item label="tagPath">
                                                    {{ dt.params.tagPath }}
                                                </el-descriptions-item>
                                                <el-descriptions-item label="名称">
                                                    {{ dt.params.name }}
                                                </el-descriptions-item>
                                                <el-descriptions-item label="host">
                                                    <SvgIcon :name="getDbDialect(dt.params.type).getInfo().icon" :size="18" />
                                                    {{ dt.params.host }}
                                                </el-descriptions-item>
                                                <el-descriptions-item label="库名">
                                                    {{ dt.params.dbName }}
                                                </el-descriptions-item>
                                            </el-descriptions>
                                        </template>
                                    </el-popover>
                                </template>

                                <db-table-data-op
                                    v-if="dt.type === TabType.TableData"
                                    :db-id="dt.dbId"
                                    :db-name="dt.db"
                                    :table-name="dt.params.table"
                                    :table-height="state.dataTabsTableHeight"
                                ></db-table-data-op>

                                <db-sql-editor
                                    v-if="dt.type === TabType.Query"
                                    :db-id="dt.dbId"
                                    :db-name="dt.db"
                                    :sql-name="dt.params.sqlName"
                                    @save-sql-success="reloadSqls"
                                >
                                </db-sql-editor>

                                <db-tables-op
                                    v-if="dt.type == TabType.TablesOp"
                                    :db-id="dt.params.id"
                                    :db="dt.params.db"
                                    :db-type="dt.params.type"
                                    :height="state.tablesOpHeight"
                                />
                            </el-tab-pane>
                        </el-tabs>
                    </div>
                </div>
            </Pane>
        </Splitpanes>
        <db-table-op
            :title="tableCreateDialog.title"
            :active-name="tableCreateDialog.activeName"
            :dbId="tableCreateDialog.dbId"
            :db="tableCreateDialog.db"
            :dbType="tableCreateDialog.dbType"
            :data="tableCreateDialog.data"
            v-model:visible="tableCreateDialog.visible"
            @submit-sql="onSubmitEditTableSql"
        />
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, h, onBeforeUnmount, onMounted, reactive, ref, toRefs } from 'vue';
import { ElCheckbox, ElMessage, ElMessageBox } from 'element-plus';
import { formatByteSize } from '@/common/utils/format';
import { DbInst, registerDbCompletionItemProvider, TabInfo, TabType } from './db';
import { NodeType, TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { dbApi } from './api';
import { dispposeCompletionItemProvider } from '@/components/monaco/completionItemProvider';
import SvgIcon from '@/components/svgIcon/index.vue';
import { ContextmenuItem } from '@/components/contextmenu';
import { getDbDialect, schemaDbTypes } from './dialect/index';
import { sleep } from '@/common/utils/loading';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Pane, Splitpanes } from 'splitpanes';
import { useEventListener } from '@vueuse/core';

const DbTableOp = defineAsyncComponent(() => import('./component/table/DbTableOp.vue'));
const DbSqlEditor = defineAsyncComponent(() => import('./component/sqleditor/DbSqlEditor.vue'));
const DbTableDataOp = defineAsyncComponent(() => import('./component/table/DbTableDataOp.vue'));
const DbTablesOp = defineAsyncComponent(() => import('./component/table/DbTablesOp.vue'));

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
    static PgSchemaMenu = 7;
    static PgSchema = 8;
}

const DbIcon = {
    name: 'Coin',
    color: '#67c23a',
};

// pgsql schema icon
const SchemaIcon = {
    name: 'List',
    color: '#67c23a',
};

const TableIcon = {
    name: 'Calendar',
    color: '#409eff',
};

const SqlIcon = {
    name: 'Files',
    color: '#f56c6c',
};

// node节点点击时，触发改变db事件
const nodeClickChangeDb = (nodeData: TagTreeNode) => {
    const params = nodeData.params;
    if (params.db) {
        changeDb({ id: params.id, host: `${params.host}`, name: params.name, type: params.type, tagPath: params.tagPath, databases: params.dbs }, params.db);
    }
};

const ContextmenuItemRefresh = new ContextmenuItem('refresh', '刷新').withIcon('RefreshRight').withOnClick((data: any) => reloadNode(data.key));

// tagpath 节点类型
const NodeTypeTagPath = new NodeType(TagTreeNode.TagPath)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const dbInfoRes = await dbApi.dbs.request({ tagPath: parentNode.key });
        const dbInfos = dbInfoRes.list;
        if (!dbInfos) {
            return [];
        }

        // 防止过快加载会出现一闪而过，对眼睛不好
        await sleep(100);
        return dbInfos?.map((x: any) => {
            x.tagPath = parentNode.key;
            return new TagTreeNode(`${parentNode.key}.${x.id}`, x.name, NodeTypeDbInst).withParams(x);
        });
    })
    .withContextMenuItems([ContextmenuItemRefresh]);

// 数据库实例节点类型
const NodeTypeDbInst = new NodeType(SqlExecNodeType.DbInst).withLoadNodesFunc((parentNode: TagTreeNode) => {
    const params = parentNode.params;
    const dbs = params.database.split(' ')?.sort();
    return dbs.map((x: any) => {
        return new TagTreeNode(`${parentNode.key}.${x}`, x, NodeTypeDb)
            .withParams({
                tagPath: params.tagPath,
                id: params.id,
                name: params.name,
                type: params.type,
                host: `${params.host}:${params.port}`,
                dbs: dbs,
                db: x,
            })
            .withIcon(DbIcon);
    });
});

// 数据库节点
const NodeTypeDb = new NodeType(SqlExecNodeType.Db)
    .withContextMenuItems([ContextmenuItemRefresh])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        params.parentKey = parentNode.key;
        // pg类数据库会多一层schema
        if (schemaDbTypes.includes(params.type)) {
            const { id, db } = params;
            const schemaNames = await dbApi.pgSchemas.request({ id, db });
            return schemaNames.map((sn: any) => {
                // 将db变更为  db/schema;
                const nParams = { ...params };
                nParams.schema = sn;
                nParams.db = nParams.db + '/' + sn;
                nParams.dbs = schemaNames;
                return new TagTreeNode(`${params.id}.${params.db}.schema.${sn}`, sn, NodeTypePostgresSchema).withParams(nParams).withIcon(SchemaIcon);
            });
        }
        return NodeTypeTables(params);
    })
    .withNodeClickFunc(nodeClickChangeDb);

const NodeTypeTables = (params: any) => {
    let tableKey = `${params.id}.${params.db}.table-menu`;
    let sqlKey = getSqlMenuNodeKey(params.id, params.db);
    return [
        new TagTreeNode(`${params.id}.${params.db}.table-menu`, '表', NodeTypeTableMenu).withParams({ ...params, key: tableKey }).withIcon(TableIcon),
        new TagTreeNode(sqlKey, 'SQL', NodeTypeSqlMenu).withParams({ ...params, key: sqlKey }).withIcon(SqlIcon),
    ];
};

// postgres schema模式
const NodeTypePostgresSchema = new NodeType(SqlExecNodeType.PgSchema)
    .withContextMenuItems([ContextmenuItemRefresh])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        params.parentKey = parentNode.key;
        return NodeTypeTables(params);
    })
    .withNodeClickFunc(nodeClickChangeDb);

// 数据库表菜单节点
const NodeTypeTableMenu = new NodeType(SqlExecNodeType.TableMenu)
    .withContextMenuItems([
        ContextmenuItemRefresh,
        new ContextmenuItem('createTable', '创建表').withIcon('Plus').withOnClick((data: any) => onEditTable(data)),
        new ContextmenuItem('tablesOp', '表操作').withIcon('Setting').withOnClick((data: any) => {
            const params = data.params;
            addTablesOpTab({ id: params.id, db: params.db, type: params.type, nodeKey: data.key });
        }),
    ])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        let { id, db, type } = params;
        // 获取当前库的所有表信息
        let tables = await DbInst.getInst(id).loadTables(db, state.reloadStatus);
        state.reloadStatus = false;
        let dbTableSize = 0;
        const tablesNode = tables.map((x: any) => {
            const tableSize = x.dataLength + x.indexLength;
            dbTableSize += tableSize;
            const key = `${id}.${db}.${x.tableName}`;
            return new TagTreeNode(key, x.tableName, NodeTypeTable)
                .withIsLeaf(true)
                .withParams({
                    id,
                    db,
                    type,
                    key: key,
                    parentKey: parentNode.key,
                    tableName: x.tableName,
                    tableComment: x.tableComment,
                    size: tableSize == 0 ? '' : formatByteSize(tableSize, 1),
                })
                .withIcon(TableIcon)
                .withLabelRemark(`${x.tableName} ${x.tableComment ? '| ' + x.tableComment : ''}`);
        });
        // 设置父节点参数的表大小
        parentNode.params.dbTableSize = dbTableSize == 0 ? '' : formatByteSize(dbTableSize);
        return tablesNode;
    })
    .withNodeClickFunc(nodeClickChangeDb);

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
            return new TagTreeNode(`${id}.${db}.${x.name}`, x.name, NodeTypeSql)
                .withIsLeaf(true)
                .withParams({ id, db, dbs, sqlName: x.name })
                .withIcon(SqlIcon);
        });
    })
    .withNodeClickFunc(nodeClickChangeDb);

// 表节点类型
const NodeTypeTable = new NodeType(SqlExecNodeType.Table)
    .withContextMenuItems([
        new ContextmenuItem('copyTable', '复制表').withIcon('copyDocument').withOnClick((data: any) => onCopyTable(data)),
        new ContextmenuItem('editTable', '编辑表').withIcon('edit').withOnClick((data: any) => onEditTable(data)),
        new ContextmenuItem('delTable', '删除表').withIcon('Delete').withOnClick((data: any) => onDeleteTable(data)),
    ])
    .withNodeClickFunc((nodeData: TagTreeNode) => {
        const params = nodeData.params;
        loadTableData({ id: params.id, nodeKey: nodeData.key }, params.db, params.tableName);
    });

// sql模板节点类型
const NodeTypeSql = new NodeType(SqlExecNodeType.Sql)
    .withNodeClickFunc((nodeData: TagTreeNode) => {
        const params = nodeData.params;
        addQueryTab({ id: params.id, nodeKey: nodeData.key, dbs: params.dbs }, params.db, params.sqlName);
    })
    .withContextMenuItems([
        new ContextmenuItem('delSql', '删除').withIcon('delete').withOnClick((data: any) => deleteSql(data.params.id, data.params.db, data.params.sqlName)),
    ]);

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
    dataTabsTableHeight: '600px',
    tablesOpHeight: '600',
    dbServerInfo: {
        loading: true,
        version: '',
    },
    tableCreateDialog: {
        visible: false,
        title: '',
        activeName: '',
        dbId: 0,
        db: '',
        dbType: '',
        data: {},
        parentKey: '',
    },
});

const { nowDbInst, tableCreateDialog } = toRefs(state);

const serverInfoReqParam = ref({
    instanceId: 0,
});
const { execute: getDbServerInfo, isFetching: loadingServerInfo, data: dbServerInfo } = dbApi.getInstanceServerInfo.useApi<any>(serverInfoReqParam);

onMounted(() => {
    setHeight();
    // 监听浏览器窗口大小变化,更新对应组件高度
    useEventListener(window, 'resize', setHeight);
});

onBeforeUnmount(() => {
    dispposeCompletionItemProvider('sql');
});

/**
 * 设置editor高度和数据表高度
 */
const setHeight = () => {
    state.dataTabsTableHeight = window.innerHeight - 253 + 'px';
    state.tablesOpHeight = window.innerHeight - 225 + 'px';
};

const showDbInfo = async (db: any) => {
    if (dbServerInfo.value) {
        dbServerInfo.value.version = '';
    }
    serverInfoReqParam.value.instanceId = db.instanceId;
    await getDbServerInfo();
};

// 选择数据库,改变当前正在操作的数据库信息
const changeDb = (db: any, dbName: string) => {
    state.nowDbInst = DbInst.getOrNewInst(db);
    state.nowDbInst.databases = db.databases;
    state.db = dbName;
};

// 加载选中的表数据，即新增表数据操作tab
const loadTableData = async (db: any, dbName: string, tableName: string) => {
    if (tableName == '') {
        return;
    }
    changeDb(db, dbName);

    const key = `${db.id}:\`${dbName}\`.${tableName}`;
    let tab = state.tabs.get(key);
    state.activeName = key;
    // 如果存在该表tab，则直接返回
    if (tab) {
        return;
    }
    tab = new TabInfo();
    tab.label = tableName;
    tab.key = key;
    tab.treeNodeKey = db.nodeKey;
    tab.dbId = db.id;
    tab.db = dbName;
    tab.type = TabType.TableData;
    tab.params = {
        ...getNowDbInfo(),
        table: tableName,
    };
    state.tabs.set(key, tab);
};

// 新建查询tab
const addQueryTab = async (db: any, dbName: string, sqlName: string = '') => {
    if (!dbName || !db.id) {
        ElMessage.warning('请选择数据库实例及对应的schema');
        return;
    }
    changeDb(db, dbName);

    const dbId = db.id;
    let label;
    let key;
    // 存在sql模板名，则该模板名只允许一个tab
    if (sqlName) {
        label = `查询-${sqlName}`;
        key = `查询:${dbId}:${dbName}.${sqlName}`;
    } else {
        let count = 1;
        state.tabs.forEach((v) => {
            if (v.type == TabType.Query && !v.params.sqlName) {
                count++;
            }
        });
        label = `新查询-${count}`;
        key = `新查询${count}:${dbId}:${dbName}`;
    }
    state.activeName = key;
    let tab = state.tabs.get(key);
    if (tab) {
        return;
    }
    tab = new TabInfo();
    tab.key = key;
    tab.label = label;
    tab.treeNodeKey = db.nodeKey;
    tab.dbId = dbId;
    tab.db = dbName;
    tab.type = TabType.Query;
    tab.params = {
        ...getNowDbInfo(),
        sqlName: sqlName,
        dbs: db.dbs,
    };
    state.tabs.set(key, tab);
    // 注册当前sql编辑框提示词
    registerDbCompletionItemProvider(tab.dbId, tab.db, tab.params.dbs, nowDbInst.value.type);
};

/**
 * 添加数据操作tab
 * @param inst
 */
const addTablesOpTab = async (db: any) => {
    const dbName = db.db;
    if (!db || !db.id) {
        ElMessage.warning('请选择数据库实例及对应的schema');
        return;
    }
    changeDb(db, dbName);

    const dbId = db.id;
    let key = `表操作:${dbId}:${dbName}.tablesOp`;
    state.activeName = key;

    let tab = state.tabs.get(key);
    if (tab) {
        return;
    }
    tab = new TabInfo();
    tab.key = key;
    tab.label = `表操作-${dbName}`;
    tab.treeNodeKey = db.nodeKey;
    tab.dbId = dbId;
    tab.db = dbName;
    tab.type = TabType.TablesOp;
    tab.params = {
        ...getNowDbInfo(),
        id: db.id,
        db: dbName,
        type: db.type,
    };
    state.tabs.set(key, tab);
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
        onTabChange();
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
        registerDbCompletionItemProvider(nowTab.dbId, nowTab.db, nowTab.params.dbs, nowDbInst.value.type);
    }
};

const reloadSqls = (dbId: number, db: string) => {
    tagTreeRef.value.reloadNode(getSqlMenuNodeKey(dbId, db));
};

const deleteSql = async (dbId: any, db: string, sqlName: string) => {
    try {
        await ElMessageBox.confirm(`确定删除【${sqlName}】该SQL内容?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteDbSql.request({ id: dbId, db: db, name: sqlName });
        ElMessage.success('删除成功');
        reloadSqls(dbId, db);
    } catch (err) {
        //
    }
};

const getSqlMenuNodeKey = (dbId: number, db: string) => {
    return `${dbId}.${db}.sql-menu`;
};

const reloadNode = (nodeKey: string) => {
    state.reloadStatus = true;
    tagTreeRef.value.reloadNode(nodeKey);
};

const onEditTable = async (data: any) => {
    let { db, id, tableName, tableComment, type, parentKey, key } = data.params;
    // data.label就是表名
    if (tableName) {
        state.tableCreateDialog.title = '修改表';
        let indexs = await dbApi.tableIndex.request({ id, db, tableName });
        let columns = await dbApi.columnMetadata.request({ id, db, tableName });
        let row = { tableName, tableComment };
        state.tableCreateDialog.data = { edit: true, row, indexs, columns };
        state.tableCreateDialog.parentKey = parentKey;
    } else {
        state.tableCreateDialog.title = '创建表';
        state.tableCreateDialog.data = { edit: false, row: {} };
        state.tableCreateDialog.parentKey = key;
    }

    state.tableCreateDialog.visible = true;
    state.tableCreateDialog.activeName = '1';
    state.tableCreateDialog.dbId = id;
    state.tableCreateDialog.db = db;
    state.tableCreateDialog.dbType = type;
};

const onDeleteTable = async (data: any) => {
    let { db, id, tableName, parentKey } = data.params;
    await ElMessageBox.confirm(`此操作是永久性且无法撤销，确定删除【${tableName}】? `, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });
    // 执行sql
    dbApi.sqlExec.request({ id, db, sql: `drop table ${tableName}` }).then(() => {
        ElMessage.success('删除成功');
        setTimeout(() => {
            parentKey && reloadNode(parentKey);
        }, 1000);
    });
};

const onCopyTable = async (data: any) => {
    let { db, id, tableName, parentKey } = data.params;

    let checked = ref(false);

    // 弹出确认框，并选择是否复制数据
    await ElMessageBox({
        title: `复制表【${tableName}】`,
        type: 'warning',
        //  icon: markRaw(Delete),
        message: () =>
            h(ElCheckbox, {
                label: '是否复制数据?',
                modelValue: checked.value,
                'onUpdate:modelValue': (val: boolean | string | number) => {
                    if (typeof val === 'boolean') {
                        checked.value = val;
                    }
                },
            }),
        callback: (action: string) => {
            if (action === 'confirm') {
                // 执行sql
                dbApi.copyTable.request({ id, db, tableName, copyData: checked.value }).then(() => {
                    ElMessage.success('复制成功');
                    setTimeout(() => {
                        parentKey && reloadNode(parentKey);
                    }, 1000);
                });
            }
        },
    });
};

const onSubmitEditTableSql = () => {
    state.tableCreateDialog.visible = false;
    state.tableCreateDialog.data = { edit: false, row: {} };
    reloadNode(state.tableCreateDialog.parentKey);
};

/**
 * 获取当前操作的数据库信息
 */
const getNowDbInfo = () => {
    const di = state.nowDbInst;
    return {
        tagPath: di.tagPath,
        id: di.id,
        name: di.name,
        type: di.type,
        host: di.host,
        dbName: state.db,
    };
};
</script>

<style lang="scss">
.db-sql-exec {
    .db-table-size {
        color: #c4c9c4;
        font-size: 9px;
    }

    .db-op {
        height: calc(100vh - 108px);
    }

    #data-exec {
        .el-tabs {
            --el-tabs-header-height: 30px;
        }

        .el-tabs__header {
            margin: 0 0 5px;

            .el-tabs__item {
                padding: 0 10px;
            }
        }

        .el-tabs__nav-next,
        .el-tabs__nav-prev {
            line-height: 30px;
        }
    }

    .update_field_active {
        background-color: var(--el-color-success);
    }
}
</style>
