<template>
    <div class="db-sql-exec h-full">
        <ResourceOpPanel>
            <template #left>
                <tag-tree
                    :default-expanded-keys="state.defaultExpendKey"
                    :resource-type="TagResourceTypePath.Db"
                    :tag-path-node-type="NodeTypeTagPath"
                    ref="tagTreeRef"
                >
                    <template #prefix="{ data }">
                        <span v-if="data.type.value == SqlExecNodeType.DbInst">
                            <el-popover
                                @show="showDbInfo(data.params)"
                                :show-after="500"
                                placement="right-start"
                                :title="$t('db.dbInstInfo')"
                                trigger="hover"
                                :width="250"
                            >
                                <template #reference>
                                    <SvgIcon :name="getDbDialect(data.params.type).getInfo().icon" :size="18" />
                                </template>
                                <template #default>
                                    <el-descriptions :column="1" size="small">
                                        <el-descriptions-item :label="$t('common.name')">
                                            {{ data.params.name }}
                                        </el-descriptions-item>
                                        <el-descriptions-item label="Host">
                                            {{ `${data.params.host}:${data.params.port}` }}
                                        </el-descriptions-item>
                                        <el-descriptions-item label="version">
                                            <span v-loading="loadingServerInfo"> {{ `${dbServerInfo?.version}` }}</span>
                                        </el-descriptions-item>
                                        <el-descriptions-item :label="$t('db.acName')">
                                            {{ data.params.authCertName }}
                                        </el-descriptions-item>
                                        <el-descriptions-item :label="$t('common.remark')">
                                            {{ data.params.remark }}
                                        </el-descriptions-item>
                                    </el-descriptions>
                                </template>
                            </el-popover>
                        </span>

                        <SvgIcon v-if="data.icon" :name="data.icon.name" :color="data.icon.color" />
                    </template>

                    <template #suffix="{ data }">
                        <span v-if="data.type.value == SqlExecNodeType.Table && data.params.size">{{ ` ${data.params.size}` }}</span>
                        <span v-if="data.type.value == SqlExecNodeType.TableMenu && data.params.dbTableSize">{{ ` ${data.params.dbTableSize}` }}</span>
                    </template>
                </tag-tree>
            </template>

            <template #right>
                <el-card class="h-full" body-class="h-full !p-1 flex flex-col flex-1">
                    <el-row>
                        <el-col :span="24" v-if="state.db">
                            <el-descriptions :column="4" size="small" border>
                                <el-descriptions-item label-align="right" :label="$t('common.operation')">
                                    <el-button
                                        :disabled="!state.db || !nowDbInst.id"
                                        type="primary"
                                        icon="Search"
                                        link
                                        @click="
                                            addQueryTab(
                                                { id: nowDbInst.id, dbs: nowDbInst.databases, nodeKey: getSqlMenuNodeKey(nowDbInst.id, state.db) },
                                                state.db
                                            )
                                        "
                                        :title="$t('db.newQuery')"
                                    >
                                    </el-button>

                                    <template v-if="!dbConfig.locationTreeNode">
                                        <el-divider direction="vertical" border-style="dashed" />
                                        <el-button @click="locationNowTreeNode(null)" :title="$t('db.locationTagTree')" icon="Location" link></el-button>
                                    </template>

                                    <el-divider direction="vertical" border-style="dashed" />
                                    <!-- 数据库展示配置 -->
                                    <el-popover
                                        popper-style="max-height: 550px; overflow: auto; max-width: 450px"
                                        placement="bottom"
                                        width="auto"
                                        :title="$t('db.dbShowSetting')"
                                        trigger="click"
                                    >
                                        <el-row>
                                            <el-checkbox
                                                v-model="dbConfig.showColumnComment"
                                                :label="$t('db.showFieldComments')"
                                                :true-value="1"
                                                :false-value="0"
                                                size="small"
                                            />
                                        </el-row>

                                        <el-row>
                                            <el-checkbox
                                                v-model="dbConfig.locationTreeNode"
                                                :label="$t('db.autoLocationTagTree')"
                                                :true-value="1"
                                                :false-value="0"
                                                size="small"
                                            />
                                        </el-row>

                                        <el-row>
                                            <el-checkbox
                                                v-model="dbConfig.cacheTable"
                                                :label="$t('db.cacheTableInfo')"
                                                :true-value="1"
                                                :false-value="0"
                                                size="small"
                                            />
                                        </el-row>

                                        <template #reference>
                                            <el-link type="primary" icon="setting" underline="never"></el-link>
                                        </template>
                                    </el-popover>
                                </el-descriptions-item>

                                <el-descriptions-item label-align="right" label="tag">{{ nowDbInst.tagPath }}</el-descriptions-item>

                                <el-descriptions-item label-align="right">
                                    <template #label>
                                        <div>
                                            <SvgIcon :name="nowDbInst.getDialect().getInfo().icon" :size="18" />
                                            {{ $t('db.dbInst') }}
                                        </div>
                                    </template>
                                    {{ nowDbInst.id }}
                                    <el-divider direction="vertical" border-style="dashed" />
                                    {{ nowDbInst.name }}
                                    <el-divider direction="vertical" border-style="dashed" />
                                    {{ nowDbInst.host }}
                                </el-descriptions-item>

                                <el-descriptions-item :label="$t('db.dbName')" label-align="right">{{ state.db }}</el-descriptions-item>
                            </el-descriptions>
                        </el-col>
                    </el-row>

                    <div id="data-exec" class="mt-1">
                        <el-tabs
                            v-if="state.tabs.size > 0"
                            type="card"
                            @tab-remove="onRemoveTab"
                            @tab-change="onTabChange"
                            v-model="state.activeName"
                            class="!h-full w-full"
                        >
                            <el-tab-pane class="!h-full" closable v-for="dt in state.tabs.values()" :label="dt.label" :name="dt.key" :key="dt.key">
                                <template #label>
                                    <el-popover :show-after="1000" placement="bottom-start" trigger="hover" :width="250">
                                        <template #reference>
                                            <span @contextmenu.prevent="onTabContextmenu(dt, $event)" class="!text-[12px]">{{ dt.label }}</span>
                                        </template>
                                        <template #default>
                                            <el-descriptions :column="1" size="small">
                                                <el-descriptions-item label="tagPath">
                                                    {{ dt.params.tagPath }}
                                                </el-descriptions-item>
                                                <el-descriptions-item :label="$t('common.name')">
                                                    {{ dt.params.name }}
                                                </el-descriptions-item>
                                                <el-descriptions-item label="Host">
                                                    <SvgIcon :name="getDbDialect(dt.params.type).getInfo().icon" :size="18" />
                                                    {{ dt.params.host }}
                                                </el-descriptions-item>
                                                <el-descriptions-item :label="$t('db.dbName')">
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
                                    :ref="(el: any) => (dt.componentRef = el)"
                                ></db-table-data-op>

                                <db-sql-editor
                                    v-if="dt.type === TabType.Query"
                                    :db-id="dt.dbId"
                                    :db-name="dt.db"
                                    :sql-name="dt.params.sqlName"
                                    @save-sql-success="reloadSqls"
                                    :ref="(el: any) => (dt.componentRef = el)"
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
                </el-card>
            </template>
        </ResourceOpPanel>

        <db-table-op
            :title="tableCreateDialog.title"
            :active-name="tableCreateDialog.activeName"
            :dbId="tableCreateDialog.dbId"
            :db="tableCreateDialog.db"
            :dbType="tableCreateDialog.dbType"
            :version="tableCreateDialog.version"
            :data="tableCreateDialog.data"
            v-model:visible="tableCreateDialog.visible"
            @submit-sql="onSubmitEditTableSql"
        />

        <el-dialog width="55%" :title="`'${state.chooseTableName}' DDL`" v-model="state.ddlDialog.visible">
            <monaco-editor height="400px" language="sql" v-model="state.ddlDialog.ddl" :options="{ readOnly: true }" />
        </el-dialog>

        <contextmenu ref="tabContextmenuRef" :dropdown="state.tabContextmenu.dropdown" :items="state.tabContextmenu.items" />
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, h, onBeforeUnmount, onMounted, reactive, ref, toRefs, useTemplateRef, watch } from 'vue';
import { ElCheckbox, ElMessage, ElMessageBox } from 'element-plus';
import { formatByteSize } from '@/common/utils/format';
import { DbInst, DbThemeConfig, registerDbCompletionItemProvider, TabInfo, TabType } from './db';
import { getTagTypeCodeByPath, NodeType, TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { dbApi } from './api';
import { dispposeCompletionItemProvider } from '@/components/monaco/completionItemProvider';
import SvgIcon from '@/components/svgIcon/index.vue';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { getDbDialect, schemaDbTypes } from './dialect/index';
import { sleep } from '@/common/utils/loading';
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import { useEventListener, useStorage } from '@vueuse/core';
import SqlExecBox from '@/views/ops/db/component/sqleditor/SqlExecBox';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { storeToRefs } from 'pinia';
import { format as sqlFormatter } from 'sql-formatter';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { useI18n } from 'vue-i18n';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import ResourceOpPanel from '../component/ResourceOpPanel.vue';

const DbTableOp = defineAsyncComponent(() => import('./component/table/DbTableOp.vue'));
const DbSqlEditor = defineAsyncComponent(() => import('./component/sqleditor/DbSqlEditor.vue'));
const DbTableDataOp = defineAsyncComponent(() => import('./component/table/DbTableDataOp.vue'));
const DbTablesOp = defineAsyncComponent(() => import('./component/table/DbTablesOp.vue'));

const { t } = useI18n();

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
    name: 'icon db/sql',
    color: '#f56c6c',
};

// node节点点击时，触发改变db事件
const nodeClickChangeDb = async (nodeData: TagTreeNode) => {
    const params = nodeData.params;
    if (params.db) {
        await changeDb(
            {
                id: params.id,
                host: `${params.host}`,
                name: params.name,
                type: params.type,
                tagPath: params.tagPath,
                databases: params.dbs,
            },
            params.db
        );
    }
};

const ContextmenuItemRefresh = new ContextmenuItem('refresh', 'common.refresh').withIcon('RefreshRight').withOnClick((data: any) => reloadNode(data.key));

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
            return new TagTreeNode(`${x.code}`, x.name, NodeTypeDbInst).withParams(x);
        });
    })
    .withContextMenuItems([ContextmenuItemRefresh]);

// 数据库实例节点类型
const NodeTypeDbInst = new NodeType(SqlExecNodeType.DbInst).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    const dbs = (await DbInst.getDbNames(params))?.sort();

    // 查询数据库版本信息
    const version = await dbApi.getCompatibleDbVersion.request({ id: params.id, db: dbs[0] });

    return dbs.map((x: any) => {
        return new TagTreeNode(`${parentNode.key}.${x}`, x, NodeTypeDb)
            .withParams({
                tagPath: params.tagPath,
                id: params.id,
                name: params.name,
                type: params.type,
                version: version || 'unset',
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

        return getNodeTypeTables(params);
    })
    .withNodeClickFunc(nodeClickChangeDb);

const getNodeTypeTables = (params: any) => {
    let tableKey = `${params.id}.${params.db}.table-menu`;
    let sqlKey = getSqlMenuNodeKey(params.id, params.db);
    return [
        new TagTreeNode(`${params.id}.${params.db}.table-menu`, t('db.table'), NodeTypeTableMenu)
            .withParams({
                ...params,
                key: tableKey,
            })
            .withIcon(TableIcon),
        new TagTreeNode(sqlKey, 'SQL', NodeTypeSqlMenu).withParams({ ...params, key: sqlKey }).withIcon(SqlIcon),
    ];
};

// postgres schema模式
const NodeTypePostgresSchema = new NodeType(SqlExecNodeType.PgSchema)
    .withContextMenuItems([ContextmenuItemRefresh])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        params.parentKey = parentNode.key;
        return getNodeTypeTables(params);
    })
    .withNodeClickFunc(nodeClickChangeDb);

// 数据库表菜单节点
const NodeTypeTableMenu = new NodeType(SqlExecNodeType.TableMenu)
    .withContextMenuItems([
        ContextmenuItemRefresh,
        new ContextmenuItem('createTable', 'db.createTable').withIcon('Plus').withOnClick((data: any) => onEditTable(data)),
        new ContextmenuItem('tablesOp', 'db.tableOp').withIcon('Setting').withOnClick((data: any) => {
            const params = data.params;
            addTablesOpTab({ id: params.id, db: params.db, type: params.type, nodeKey: data.key });
        }),
    ])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        let { id, db, type, schema, version } = params;
        // 获取当前库的所有表信息
        let tables = await DbInst.getInst(id).loadTables(db, state.reloadStatus);
        state.reloadStatus = !dbConfig.value.cacheTable;
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
                    schema,
                    version,
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
    });
// .withNodeDblclickFunc((node: TagTreeNode) => {
//     const params = node.params;
//     addTablesOpTab({ id: params.id, db: params.db, type: params.type, version: params.version, nodeKey: node.key });
// });

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
        new ContextmenuItem('copyTable', 'db.copyTable').withIcon('copyDocument').withOnClick((data: any) => onCopyTable(data)),
        new ContextmenuItem('renameTable', 'db.renameTable').withIcon('edit').withOnClick((data: any) => onRenameTable(data)),
        new ContextmenuItem('editTable', 'db.editTable').withIcon('edit').withOnClick((data: any) => onEditTable(data)),
        new ContextmenuItem('delTable', 'db.delTable').withIcon('Delete').withOnClick((data: any) => onDeleteTable(data)),
        new ContextmenuItem('ddl', 'DDL').withIcon('Document').withOnClick((data: any) => onGenDdl(data)),
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
        new ContextmenuItem('delSql', 'common.delete')
            .withIcon('delete')
            .withOnClick((data: any) => deleteSql(data.params.id, data.params.db, data.params.sqlName)),
    ]);

const tabContextmenuItems = [
    new ContextmenuItem(1, 'db.close').withIcon('Close').withOnClick((data: any) => {
        onRemoveTab(data.key);
    }),

    new ContextmenuItem(2, 'db.closeOther').withIcon('CircleClose').withOnClick((data: any) => {
        const tabName = data.key;
        const tabNames = [...state.tabs.keys()];
        for (let tab of tabNames) {
            if (tab !== tabName) {
                onRemoveTab(tab);
            }
        }
    }),
];

const tagTreeRef: any = ref(null);
const tabContextmenuRef: any = useTemplateRef('tabContextmenuRef');

const tabs: Map<string, TabInfo> = new Map();
const state = reactive({
    defaultExpendKey: [] as any,
    /**
     * 当前操作的数据库实例
     */
    nowDbInst: {} as DbInst,
    db: '', // 当前操作的数据库
    activeName: '',
    reloadStatus: false,
    tabs,
    tabContextmenu: {
        dropdown: { x: 0, y: 0 },
        items: tabContextmenuItems,
    },
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
        version: '',
        db: '',
        dbType: '',
        data: {},
        parentKey: '',
    },
    chooseTableName: '',
    ddlDialog: {
        visible: false,
        ddl: '',
    },
});

const { nowDbInst, tableCreateDialog } = toRefs(state);

const dbConfig = useStorage('dbConfig', DbThemeConfig);

const serverInfoReqParam = ref({
    instanceId: 0,
});
const { execute: getDbServerInfo, isFetching: loadingServerInfo, data: dbServerInfo } = dbApi.getInstanceServerInfo.useApi<any>(serverInfoReqParam);

const autoOpenResourceStore = useAutoOpenResource();
const { autoOpenResource } = storeToRefs(autoOpenResourceStore);

onMounted(() => {
    state.reloadStatus = !dbConfig.value.cacheTable;
    autoOpenDb(autoOpenResource.value.dbCodePath);
    setHeight();
    // 监听浏览器窗口大小变化,更新对应组件高度
    useEventListener(window, 'resize', setHeight);
});

onBeforeUnmount(() => {
    dispposeCompletionItemProvider('sql');
});

watch(
    () => autoOpenResource.value.dbCodePath,
    (codePath: any) => {
        autoOpenDb(codePath);
    }
);

const autoOpenDb = (codePath: string) => {
    if (!codePath) {
        return;
    }

    const typeAndCodes: any = getTagTypeCodeByPath(codePath);
    const tagPath = typeAndCodes[TagResourceTypeEnum.Tag.value].join('/') + '/';

    const dbCode = typeAndCodes[TagResourceTypeEnum.Db.value][0];
    state.defaultExpendKey = [tagPath, dbCode];

    setTimeout(() => {
        // 置空
        autoOpenResourceStore.setDbCodePath('');
        tagTreeRef.value.setCurrentKey(dbCode);
    }, 1000);
};

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
const changeDb = async (db: any, dbName: string) => {
    state.nowDbInst = await DbInst.getOrNewInst(db);
    state.nowDbInst.databases = db.databases;
    state.db = dbName;
};

// 加载选中的表数据，即新增表数据操作tab
const loadTableData = async (db: any, dbName: string, tableName: string) => {
    if (tableName == '') {
        return;
    }
    await changeDb(db, dbName);

    const key = `tableData:${db.id}.${dbName}.${tableName}`;
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
        ElMessage.warning(t('db.noDbInstMsg'));
        return;
    }
    await changeDb(db, dbName);

    const dbId = db.id;
    let label;
    let key;
    // 存在sql模板名，则该模板名只允许一个tab
    if (sqlName) {
        label = `${t('db.query')}-${sqlName}`;
        key = `query:${dbId}.${dbName}.${sqlName}`;
    } else {
        let count = 1;
        state.tabs.forEach((v) => {
            if (v.type == TabType.Query && !v.params.sqlName) {
                count++;
            }
        });
        label = `${t('db.nQuery')}-${count}`;
        key = `query:${count}.${dbId}.${dbName}`;
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
        ElMessage.warning(t('db.noDbInstMsg'));
        return;
    }
    await changeDb(db, dbName);

    const dbId = db.id;
    let key = `tablesOp:${dbId}.${dbName}`;
    state.activeName = key;

    let tab = state.tabs.get(key);
    if (tab) {
        return;
    }
    tab = new TabInfo();
    tab.key = key;
    tab.label = `${t('db.tableOp')}-${dbName}`;
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

        state.tabs.delete(targetName);
        if (activeName != targetName) {
            break;
        }

        // 如果删除的tab是当前激活的tab，则切换到前一个或后一个tab
        const nextTab = tabNames[i + 1] || tabNames[i - 1];
        if (nextTab) {
            activeName = nextTab;
        } else {
            activeName = '';
        }
        state.activeName = activeName;
        onTabChange();
        break;
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

    // 激活当前tab（需要调用DbTableData组件的active，否则表头与数据会出现错位，暂不知为啥，先这样处理）
    nowTab?.componentRef?.active();

    if (dbConfig.value.locationTreeNode) {
        locationNowTreeNode(nowTab);
    }
};

// 右键点击时：传 x,y 坐标值到子组件中（props）
const onTabContextmenu = (v: any, e: any) => {
    const { clientX, clientY } = e;
    state.tabContextmenu.dropdown.x = clientX;
    state.tabContextmenu.dropdown.y = clientY;
    tabContextmenuRef.value.openContextmenu(v);
};

/**
 * 定位至当前树节点
 */
const locationNowTreeNode = (nowTab: any = null) => {
    if (!nowTab) {
        nowTab = state.tabs.get(state.activeName);
    }
    tagTreeRef.value.setCurrentKey(nowTab?.treeNodeKey);
};

const reloadSqls = (dbId: number, db: string) => {
    tagTreeRef.value.reloadNode(getSqlMenuNodeKey(dbId, db));
};

const deleteSql = async (dbId: any, db: string, sqlName: string) => {
    try {
        await useI18nDeleteConfirm(sqlName);
        await dbApi.deleteDbSql.request({ id: dbId, db: db, name: sqlName });
        useI18nDeleteSuccessMsg();
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
    let { db, id, tableName, tableComment, type, parentKey, key, version } = data.params;
    // data.label就是表名
    if (tableName) {
        state.tableCreateDialog.title = useI18nEditTitle('db.table');
        let indexs = await dbApi.tableIndex.request({ id, db, tableName });
        let columns = await dbApi.columnMetadata.request({ id, db, tableName });
        let row = { tableName, tableComment };
        state.tableCreateDialog.data = { edit: true, row, indexs, columns };
        state.tableCreateDialog.parentKey = parentKey;
    } else {
        state.tableCreateDialog.title = useI18nCreateTitle('db.table');
        state.tableCreateDialog.data = { edit: false, row: {} };
        state.tableCreateDialog.parentKey = key;
    }

    state.tableCreateDialog.activeName = '1';
    state.tableCreateDialog.dbId = id;
    state.tableCreateDialog.version = version;
    state.tableCreateDialog.db = db;
    state.tableCreateDialog.dbType = type;
    state.tableCreateDialog.visible = true;
};

const onDeleteTable = async (data: any) => {
    let { db, id, tableName, parentKey, schema } = data.params;
    await useI18nDeleteConfirm(tableName);

    // 执行sql
    let dialect = getDbDialect(state.nowDbInst.type);
    let schemaStr = schema ? `${dialect.quoteIdentifier(schema)}.` : '';

    dbApi.sqlExec.request({ id, db, sql: `drop table ${schemaStr + dialect.quoteIdentifier(tableName)}` }).then((res) => {
        let success = true;
        for (let re of res) {
            if (re.errorMsg) {
                success = false;
                ElMessage.error(`${re.sql} -> ${re.errorMsg}`);
            }
        }
        if (success) {
            useI18nDeleteSuccessMsg();
            setTimeout(() => {
                parentKey && reloadNode(parentKey);
            }, 1000);
        }
    });
};

const onGenDdl = async (data: any) => {
    let { db, id, tableName, type } = data.params;
    state.chooseTableName = tableName;
    let res = await dbApi.tableDdl.request({ id, db, tableName });
    state.ddlDialog.ddl = sqlFormatter(res, { language: getDbDialect(type).getInfo().formatSqlDialect as any });
    state.ddlDialog.visible = true;
};

const onRenameTable = async (data: any) => {
    let { db, id, tableName, parentKey } = data.params;
    let tableData = { db, oldTableName: tableName, tableName };

    let value = ref(tableName);
    // 弹出确认框
    const promptValue = await ElMessageBox.prompt('', t('db.renamePrompt', { db, tableName }), {
        inputValue: value.value,
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
    });

    tableData.tableName = promptValue.value;
    let sql = nowDbInst.value.getDialect().getModifyTableInfoSql(tableData);
    if (!sql) {
        ElMessage.warning(t('db.noChange'));
        return;
    }

    SqlExecBox({
        sql: sql,
        dbId: id as any,
        db: db as any,
        dbType: nowDbInst.value.getDialect().getInfo().formatSqlDialect,
        runSuccessCallback: () => {
            setTimeout(() => {
                parentKey && reloadNode(parentKey);
            }, 1000);
        },
    });
};

const onCopyTable = async (data: any) => {
    let { db, id, tableName, parentKey } = data.params;

    let checked = ref(false);

    // 弹出确认框，并选择是否复制数据
    await ElMessageBox({
        title: `${t('db.copyTable')}【${tableName}】`,
        type: 'warning',
        //  icon: markRaw(Delete),
        message: () =>
            h(ElCheckbox, {
                label: t('db.isCopyTableData'),
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
                    useI18nOperateSuccessMsg();
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

<style lang="scss" scoped>
.db-sql-exec {
    #data-exec {
        ::v-deep(.el-tabs) {
            --el-tabs-header-height: 30px;
        }

        ::v-deep(.el-tabs__header) {
            margin: 0 0 5px;

            .el-tabs__item {
                padding: 0 5px;
            }
        }

        ::v-deep(.el-tabs__nav-next) {
            line-height: 30px;
        }
        ::v-deep(.el-tabs__nav-prev) {
            line-height: 30px;
        }
    }

    .update_field_active {
        background-color: var(--el-color-success);
    }
}
</style>
