<template>
    <div class="db-sql-exec h-full">
        <el-row>
            <el-col :span="24" v-if="state.db">
                <el-descriptions :column="4" size="small" border>
                    <el-descriptions-item label-align="right" :label="$t('common.operation')">
                        <el-button
                            :disabled="!state.db || !nowDbInst.id"
                            type="primary"
                            icon="Search"
                            link
                            @click="addQueryTab({ id: nowDbInst.id, dbs: nowDbInst.databases, nodeKey: getSqlMenuNodeKey(nowDbInst.id, state.db) }, state.db)"
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
                                <el-checkbox v-model="dbConfig.cacheTable" :label="$t('db.cacheTableInfo')" :true-value="1" :false-value="0" size="small" />
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
import { defineAsyncComponent, getCurrentInstance, h, inject, onBeforeUnmount, onMounted, reactive, ref, toRefs, useTemplateRef, watch } from 'vue';
import { ElCheckbox, ElMessage, ElMessageBox } from 'element-plus';
import { DbInst, DbThemeConfig, registerDbCompletionItemProvider, TabInfo, TabType } from '../db';
import { ResourceOpCtx } from '@/views/ops/component/tag';
import { dbApi } from '../api';
import { dispposeCompletionItemProvider } from '@/components/monaco/completionItemProvider';
import SvgIcon from '@/components/svgIcon/index.vue';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { getDbDialect } from '../dialect/index';
import { useEventListener, useStorage } from '@vueuse/core';
import SqlExecBox from '@/views/ops/db/component/sqleditor/SqlExecBox';
import { format as sqlFormatter } from 'sql-formatter';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { useI18n } from 'vue-i18n';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import { ResourceOpCtxKey } from '@/views/ops/resource/resource';
import { DbDataOpComp } from '@/views/ops/db/resource';

const DbTableOp = defineAsyncComponent(() => import('../component/table/DbTableOp.vue'));
const DbSqlEditor = defineAsyncComponent(() => import('../component/sqleditor/DbSqlEditor.vue'));
const DbTableDataOp = defineAsyncComponent(() => import('../component/table/DbTableDataOp.vue'));
const DbTablesOp = defineAsyncComponent(() => import('../component/table/DbTablesOp.vue'));

const { t } = useI18n();

const resourceOpCtx: ResourceOpCtx | undefined = inject(ResourceOpCtxKey);

const emits = defineEmits(['init']);

const tabContextmenuRef: any = useTemplateRef('tabContextmenuRef');

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

onMounted(() => {
    state.reloadStatus = !dbConfig.value.cacheTable;
    emits('init', { name: DbDataOpComp.name, ref: getCurrentInstance()?.exposed });
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
    setTimeout(() => resourceOpCtx?.setCurrentTreeKey(nowTab?.treeNodeKey), 500);
};

const reloadSqls = (dbId: number, db: string) => {
    resourceOpCtx?.reloadTreeNode(getSqlMenuNodeKey(dbId, db));
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
    resourceOpCtx?.reloadTreeNode(nodeKey);
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

const loadTables = async (dbInfo: any) => {
    if (!dbInfo || !dbInfo.id) {
        ElMessage.warning(t('db.noDbInstMsg'));
        return;
    }
    let { id, db } = dbInfo;
    // 获取当前库的所有表信息
    let tables = await DbInst.getInst(id).loadTables(db, state.reloadStatus);
    state.reloadStatus = !dbConfig.value.cacheTable;
    return tables;
};

defineExpose({
    onChangeDb: changeDb,
    loadTables,
    loadTableData,
    onCopyTable,
    onEditTable,
    onDeleteTable,
    onGenDdl,
    onRenameTable,
    onRemoveTab,
    addQueryTab,
    addTablesOpTab,
    reloadSqls,
    deleteSql,
    reloadNode,
});
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
