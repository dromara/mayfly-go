<template>
    <div class="db-sql-exec h-full flex flex-col">
        <el-row>
            <el-col :span="24" v-if="state.db">
                <el-descriptions :column="4" size="small" border>
                    <el-descriptions-item label-align="right" :label="$t('common.operation')">
                        <el-button
                            :disabled="!state.db || !nowDbInst.id"
                            type="primary"
                            icon="Search"
                            link
                            @click="addQueryTab({ id: nowDbInst.id, dbs: nowDbInst.databases, nodeKey: getSqlMenuNodeKey(state.db) }, state.db)"
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

        <div id="data-exec" ref="dataExecRef" class="mt-1 flex-1 min-h-0 overflow-visible flex flex-col">
            <Tabs v-model="activeTabKey" :tabs="tabList" class="flex-1 min-h-0" content-class="overflow-hidden" @close="onRemoveTab" @change="onTabChange">
                <template #default="{ tab: dt }">
                    <db-table-data-op
                        v-if="dt.type === TabType.TableData"
                        :db-id="dt.dbId"
                        :db-name="dt.db"
                        :table-name="dt.params.table ?? ''"
                        :readonly="!!dt.params.readonly"
                        :ref="(el) => setTabComponentRef(dt, el)"
                    ></db-table-data-op>

                    <db-sql-editor
                        v-if="dt.type === TabType.Query"
                        :db-id="dt.dbId"
                        :db-name="dt.db"
                        :db-code="(dt.params.dbCode as string) ?? ''"
                        :sql-name="dt.params.sqlName"
                        @save-sql-success="reloadSqls"
                        @editor-ready="(editorUri: string) => onEditorReady(dt, editorUri)"
                        :ref="(el) => setTabComponentRef(dt, el)"
                    >
                    </db-sql-editor>

                    <db-tables-op
                        v-if="dt.type == TabType.TablesOp"
                        :db-id="dt.params.id"
                        :db="dt.params.db ?? ''"
                        :db-type="dt.params.type"
                        :height="state.tablesOpHeight"
                    />
                </template>
            </Tabs>
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

        <el-dialog width="55%" :title="`'${state.chooseTableName}' DDL`" v-model="state.ddlDialog.visible" append-to-body>
            <monaco-editor height="400px" language="sql" v-model="state.ddlDialog.ddl" :options="{ readOnly: true }" />
        </el-dialog>

        <el-dialog width="42%" :title="`${state.propsDialog.name} - ${$t('db.objectProps')}`" v-model="state.propsDialog.visible" append-to-body>
            <el-descriptions :column="1" border>
                <el-descriptions-item :label="$t('db.seqDataType')">{{ state.propsDialog.attrs.dataType }}</el-descriptions-item>
                <el-descriptions-item :label="$t('db.seqStartValue')">{{ state.propsDialog.attrs.startValue }}</el-descriptions-item>
                <el-descriptions-item :label="$t('db.seqIncrementBy')">{{ state.propsDialog.attrs.incrementBy }}</el-descriptions-item>
                <el-descriptions-item :label="$t('db.seqMinValue')">{{ state.propsDialog.attrs.minValue }}</el-descriptions-item>
                <el-descriptions-item :label="$t('db.seqMaxValue')">{{ state.propsDialog.attrs.maxValue }}</el-descriptions-item>
                <el-descriptions-item :label="$t('db.seqCacheSize')">{{ state.propsDialog.attrs.cacheSize }}</el-descriptions-item>
                <el-descriptions-item :label="$t('db.seqLastValue')">{{ state.propsDialog.attrs.lastValue || '-' }}</el-descriptions-item>
                <el-descriptions-item :label="$t('db.seqCycle')">{{
                    state.propsDialog.attrs.isCycle === 'true' ? $t('common.yes') : $t('common.no')
                }}</el-descriptions-item>
            </el-descriptions>
            <template #footer>
                <el-button @click="state.propsDialog.visible = false">{{ $t('common.close') }}</el-button>
                <el-button type="primary" @click="onPropsViewDdl">{{ $t('db.viewDdl') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import { Tabs } from '@/components/tabs';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { treeEvents } from '@/views/ops/resource/tree';
import { useEventListener, useStorage } from '@vueuse/core';
import { defineAsyncComponent, onActivated, onBeforeUnmount, onMounted, reactive, toRef, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { dbApi } from '../api';
import { DbInst, DbThemeConfig } from '../db';
// SQL 联想经惰性作用域注册：静态引入 completion 会把整份编辑器（约 3.9M）拖进标签页容器的首屏
import { createSqlCompletionScope } from '../completion/lazy';
import { TabInfo, TabType, type TabComponentRef } from './TabInfo';
import type { DbTreeNodeData, TableOpData } from '../types';
import type { DbOpTabApi } from './helpers';
import { useDbTabs } from './composables/useDbTabs';
import { useTableOperations } from './composables/useTableOperations';
import { getDbDialect } from '../dialect/index';

const DbTableOp = defineAsyncComponent(() => import('../table-editor/DbTableOp.vue'));
const DbSqlEditor = defineAsyncComponent(() => import('../sql-editor/DbSqlEditor.vue'));
const DbTableDataOp = defineAsyncComponent(() => import('../data-grid/DbTableDataOp.vue'));
const DbTablesOp = defineAsyncComponent(() => import('../table-editor/DbTablesOp.vue'));
// DDL 查看是低频动作，编辑器主体随弹窗首次打开才加载
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const { t } = useI18n();

const props = defineProps<{
    dbInfo: DbTreeNodeData;
    db: string;
}>();

/** 标签页状态：增删切换的纯状态逻辑在 useDbTabs，本组件只负责切换后的副作用 */
const { tabs, activeTabKey, tabList, activeTab, activateTab, addTab, closeTab, clearTabs } = useDbTabs();

const state = reactive({
    defaultExpendKey: [] as string[],
    /**
     * 当前操作的数据库实例
     */
    nowDbInst: {} as DbInst,
    db: '', // 当前操作的数据库
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
        data: null as TableOpData | null,
        parentKey: '',
    },
    chooseTableName: '',
    ddlDialog: {
        visible: false,
        ddl: '',
    },
    // 序列等对象的属性面板（序列无行数据，展示定义属性而非原始 DDL）
    propsDialog: {
        visible: false,
        name: '',
        attrs: {} as Record<string, unknown>,
        args: null as null | { id: number; db: string; type: string; schema?: string; kind: string; name: string },
    },
});

const { nowDbInst, tableCreateDialog } = toRefs(state);

const chooseTableName = toRef(state, 'chooseTableName');
const ddlDialogRef = toRef(state, 'ddlDialog');
const propsDialogRef = toRef(state, 'propsDialog');

/**
 * 局部重载资源树节点（右击刷新与建表/改表/改名/删表/复制表的成功回调共用）。
 *
 * 重载即「以最新库结构为准」：表清单等节点数据直连接口重拉，不经过 DbInst.loadTables，
 * 若不同时失效客户端元数据缓存，会出现「树已显示新结构、SQL 补全仍给旧表清单/旧列」的不一致。
 * 页签未就绪时实例 id 与库名可能为空，由 DbInst.invalidateSchema 兜为 no-op。
 */
const reloadNode = (nodeKey: string) => {
    DbInst.invalidateSchema(state.nowDbInst?.id, state.db);
    treeEvents.emit('node:invalidate', { key: nodeKey });
};

/** 表/对象操作回调（编辑表、删除表、DDL查看、重命名、复制等） */
const { onEditTable, onDeleteTable, onGenDdl, onGenObjectDdl, onShowObjectProps, onPropsViewDdl, onRenameTable, onCopyTable, onSubmitEditTableSql } =
    useTableOperations({
        nowDbInst,
        tableCreateDialog,
        ddlDialog: ddlDialogRef,
        propsDialog: propsDialogRef,
        chooseTableName,
        reloadNode,
    });

/** 设置 tab 的组件 ref（模板 ref 回调，el 为子组件实例或 null） */
const setTabComponentRef = (dt: TabInfo, el: unknown) => {
    dt.componentRef = el as TabComponentRef | null;
};

const dbConfig = useStorage('dbConfig', DbThemeConfig);

/**
 * 本组件的 SQL 联想使用方作用域。
 *
 * 注册放在容器而非编辑器组件里，是因为补全上下文需要「当前激活 tab」的库信息：
 * 标签页容器不销毁非活跃面板，多个查询编辑器实例共存，provider 按 editorUri 路由，
 * 各 tab 的编辑器在 @ready 时上报 editorUri，切 tab 时 promote 回落目标（见 onTabChange）。
 */
const sqlCompletion = createSqlCompletionScope();

// 申领与释放成对：本页是 keep-alive 路由页，回到前台时 provider 可能已被其他页面（同步任务表单等）
// 申领走，需重新指回当前 tab；卸载时交还给上一个存活的使用方
onActivated(() => sqlCompletion.refresh());
onBeforeUnmount(() => sqlCompletion.release());

/**
 * 编辑器就绪回调：DbSqlEditor 的 monaco 编辑器初始化完成后上报 editorUri，
 * 登记到补全作用域使 provider 按编辑器路由（多编辑器共存时避免跨库串扰）。
 */
const onEditorReady = (tab: TabInfo, editorUri: string) => {
    tab.editorUri = editorUri;
    sqlCompletion.setEditorUri(editorUri);
};

onMounted(() => {
    changeDb(props.dbInfo, props.db);
    setHeight();
    // 监听浏览器窗口大小变化,更新对应组件高度
    useEventListener(window, 'resize', setHeight);
});

const dataExecRef = useTemplateRef<HTMLElement>('dataExecRef');

/**
 * 设置editor高度和数据表高度（基于容器位置计算，兼容全屏模式）
 */
const setHeight = () => {
    const el = document.getElementById('data-exec');
    if (el) {
        const rect = el.getBoundingClientRect();
        state.tablesOpHeight = window.innerHeight - rect.top - 60 + 'px';
    } else {
        state.tablesOpHeight = window.innerHeight - 225 + 'px';
    }
};

// 选择数据库,改变当前正在操作的数据库信息
const changeDb = (db: DbTreeNodeData, dbName: string) => {
    state.nowDbInst = DbInst.getOrNewInst(db);
    state.nowDbInst.databases = db.databases ?? [];
    state.db = dbName;
};

// 加载选中的表数据，即新增表数据操作tab
const loadTableData = async (db: DbTreeNodeData, dbName: string, tableName: string, readonly = false, title?: string) => {
    if (tableName == '') {
        return;
    }
    changeDb(db, dbName);

    const key = `tableData:${db.id}.${dbName}.${tableName}`;
    // 如果存在该表tab，则只激活不重复创建
    if (activateTab(key)) {
        return;
    }
    const tab = new TabInfo();
    tab.label = tableName;
    // 悬浮提示带上表备注（与资源树节点提示同源），无备注时回落到表名
    tab.title = title?.trim() || undefined;
    tab.key = key;
    tab.treeNodeKey = db.nodeKey ?? '';
    tab.dbId = db.id;
    tab.db = dbName;
    tab.type = TabType.TableData;
    tab.params = {
        ...getNowDbInfo(),
        table: tableName,
        readonly,
    };
    addTab(tab);
};

// 新建查询tab
const addQueryTab = async (db: DbTreeNodeData, dbName: string, sqlName: string = '') => {
    if (!dbName || !db.id) {
        Msg.warning('db.noDbInstMsg');
        return;
    }
    changeDb(db, dbName);

    const dbId = db.id;
    let label;
    let key;
    // 存在sql模板名，则该模板名只允许一个tab
    if (sqlName) {
        label = `${t('db.query')}-${sqlName}`;
        key = `query:${dbId}.${dbName}.${sqlName}`;
    } else {
        let count = 1;
        tabs.forEach((v) => {
            if (v.type == TabType.Query && !v.params.sqlName) {
                count++;
            }
        });
        label = `${t('db.nQuery')}-${count}`;
        key = `query:${count}.${dbId}.${dbName}`;
    }
    if (activateTab(key)) {
        return;
    }
    const tab = new TabInfo();
    tab.key = key;
    tab.label = label;
    tab.treeNodeKey = db.nodeKey ?? '';
    tab.dbId = dbId;
    tab.db = dbName;
    tab.type = TabType.Query;
    tab.params = {
        ...getNowDbInfo(),
        // 库的资源编码：getNowDbInfo 取的是实例级信息、调用点传的内联节点对象也不带 code，
        // 只能从本容器对应的库节点参数取（与 getSqlMenuNodeKey 同源）；
        // 被触发策略拦下时提单要靠它才能解析出审批流程
        dbCode: db.dbCode ?? props.dbInfo.dbCode ?? '',
        sqlName: sqlName,
        dbs: db.dbs,
    };
    addTab(tab);
    // 注册当前sql编辑框提示词
    sqlCompletion.register(tab.dbId, tab.db, tab.params.dbs, nowDbInst.value.type);
};

/**
 * 添加数据操作tab
 * @param inst
 */
const addTablesOpTab = async (db: DbTreeNodeData) => {
    const dbName = db.db ?? '';
    if (!db || !db.id) {
        Msg.warning('db.noDbInstMsg');
        return;
    }
    changeDb(db, dbName);

    const dbId = db.id;
    const key = `tablesOp:${dbId}.${dbName}`;
    if (activateTab(key)) {
        return;
    }
    const tab = new TabInfo();
    tab.key = key;
    tab.label = `${t('db.tableOp')}-${dbName}`;
    tab.treeNodeKey = db.nodeKey ?? '';
    tab.dbId = dbId;
    tab.db = dbName;
    tab.type = TabType.TablesOp;
    tab.params = {
        ...getNowDbInfo(),
        id: db.id,
        db: dbName,
        type: db.type ?? '',
    };
    addTab(tab);
};

const onRemoveTab = (targetName: string) => {
    // 只有关闭的是当前激活tab时才会切换激活项，此时才需要补做切换副作用
    if (closeTab(targetName)) {
        onTabChange();
    }
};

const onTabChange = () => {
    const nowTab = activeTab.value;
    if (!nowTab) {
        state.nowDbInst = {} as DbInst;
        state.db = '';
        return;
    }

    state.nowDbInst = nowTab.getNowDbInst();
    state.db = nowTab.db;

    if (nowTab.type == TabType.Query) {
        // 注册sql提示（携带 editorUri 使 provider 按编辑器路由，避免多编辑器共存时跨库串扰）
        sqlCompletion.register(nowTab.dbId, nowTab.db, nowTab.params.dbs, nowDbInst.value.type, nowTab.editorUri);
    }

    // 激活当前tab（需要调用DbTableData组件的active，否则表头与数据会出现错位，暂不知为啥，先这样处理）
    nowTab.componentRef?.active?.();

    if (dbConfig.value.locationTreeNode) {
        locationNowTreeNode(nowTab);
    }
};

// 标签条（含右键关闭当前/其它/右侧/全部）与内容面板的常驻挂载保活均由 Tabs 托管，本页只按 tab 提供内容

/**
 * 定位至当前树节点
 */
const locationNowTreeNode = (nowTab: TabInfo | null = null) => {
    if (!nowTab) {
        nowTab = activeTab.value ?? null;
    }
    // 定位事件：容器负责展开祖先并滚动选中（目标未水合时由容器水合后重试）
    const key = nowTab?.treeNodeKey ?? '';
    if (key) {
        treeEvents.emit('node:locate', { key });
    }
};

const reloadSqls = (dbId: number, db: string) => {
    treeEvents.emit('node:invalidate', { key: getSqlMenuNodeKey(db) });
};

const deleteSql = async (dbId: number, db: string, sqlName: string) => {
    try {
        if (!(await useI18nDeleteConfirm(sqlName))) {
            return;
        }
        await dbApi.deleteDbSql.request({ id: dbId, db: db, name: sqlName });
        Msg.deleteSuccess();
        reloadSqls(dbId, db);
    } catch (err) {
        //
    }
};

// sql-menu 树节点 key 必须与 contributors.buildMenuChildren 生成的完全一致：`${instCode}.${dbCode}.${db}.sql-menu`。
// 早期用 `${dbId}.${db}` 拼（数字实例 id + 库名），与真实 key（实例 code + 库 code + 库名）对不上，
// 导致保存/删除 SQL 后 invalidate 打到不存在的节点、SQL 菜单不刷新。instCode/dbCode 由 getDbOpTab 透传。
const getSqlMenuNodeKey = (db: string) => {
    return `${props.dbInfo.instCode}.${props.dbInfo.dbCode}.${db}.sql-menu`;
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

const onRefresh = () => {
    clearTabs();
};

defineExpose({
    onRefresh,
    onChangeDb: changeDb,
    loadTableData,
    onCopyTable,
    onEditTable,
    onDeleteTable,
    onGenDdl,
    onGenObjectDdl,
    onShowObjectProps,
    onRenameTable,
    onRemoveTab,
    addQueryTab,
    addTablesOpTab,
    reloadSqls,
    deleteSql,
    reloadNode,
} satisfies DbOpTabApi);
</script>

<style lang="scss" scoped>
.db-sql-exec {
    .update_field_active {
        background-color: var(--el-color-success);
    }
}
</style>
