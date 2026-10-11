/**
 * DB 资源树 - 共享常量与辅助函数
 * 打破 index.ts ↔ commands.ts/contributors.ts 循环依赖
 */
import { defineAsyncComponent } from 'vue';

import { ResourceTypeEnum } from '@/common/commonEnum';
import { createResourceOpTab, allResourceOpTabs, activeResourceOpTabKey, getResourceOpTab } from '../../resource/resourceOp';
import type { TreeNode } from '../../resource/tree';
import { getDbDialect } from '../dialect/index';
import type { DbTreeNodeData, TreeNodeCallbackData, DbNodeParams, DbTableNodeParams, DbObjectNodeParams } from '../types';

// ---------------------------------- 异步组件 ----------------------------------

const DbDataOp = defineAsyncComponent(() => import('./DbDataOp.vue'));

// ---------------------------------- 图标 ----------------------------------

export const DbIcon = {
    name: ResourceTypeEnum.Db.extra.icon,
    color: ResourceTypeEnum.Db.extra.iconColor,
};

export const SchemaIcon = {
    name: 'List',
    color: '#67c23a',
};

export const TableIcon = {
    name: 'icon db/table',
    color: '#409eff',
};

export const SqlIcon = {
    name: 'icon db/sql',
    color: '#f56c6c',
};

// ---------------------------------- Kind 常量 ----------------------------------

export const DbInstKind = 'db-inst';
export const DbDbsKind = 'db-dbs';
export const DbKind = 'db';
export const DbSchemaKind = 'db-schema';
export const DbTableMenuKind = 'db-table-menu';
export const DbSqlMenuKind = 'db-sql-menu';
export const DbTableKind = 'db-table';
export const DbSqlKind = 'db-sql';
// 扩展元数据对象：分类菜单节点（视图/序列/存储过程…，可展开）与其下的对象叶子节点
export const DbObjectMenuKind = 'db-object-menu';
export const DbObjectKind = 'db-object';

// 表节点搜索框（表数量超阈值时作为表菜单首个子节点出现，走服务端 LIKE 下推）
export const DbTableSearchKind = 'db-table-search';
// 搜索结果容器节点：与搜索框并列，输入只刷新本节点的子级（搜索框因此不被重挂载，保留焦点/输入）
export const DbTableResultsKind = 'db-table-results';
// 「加载更多」伪节点：分页续载入口，单击取下一页 offset 增量追加到结果容器尾部
export const DbTableLoadMoreKind = 'db-table-load-more';
// 超过该表数量则不直接渲染全量表节点，改为搜索框 + 首屏分页（探测用 limit=阈值+1，不全量拉取）
export const DB_TABLE_SEARCH_THRESHOLD = 500;
// 单页表数量上限：首屏/续载/搜索共用同一页宽（满页即认为还有下一页，短页即到底）
export const DB_TABLE_PAGE_SIZE = 200;
// 表搜索结果容器节点 key（由表菜单 key 派生）：contributors 建节点与搜索框组件定位结果节点共用，避免 key 后缀字面量散落两处
export const tableResultsKey = (menuKey: string | number): string => `${menuKey}.table-results`;

export const ObjectIcon = { name: 'List', color: '#e6a23c' };

// 扩展对象类别 kind（与后端 dbi.ObjectKind 对齐）。DB_OBJECT_KINDS 与命令 when 判定共用，避免字面量散落。
export const ObjectKind = {
    View: 'view',
    Sequence: 'sequence',
    Procedure: 'procedure',
    Function: 'function',
    Trigger: 'trigger',
} as const;

// 后端能力位（SupportedFeatures 输出的对象类 feature 名，与 dbi.ObjectKindFeatures 命名对齐）→ 资源树分类节点。
// 新增对象类别只需后端加 feature + 此处加一行，树渲染零改动（数据驱动、开闭原则）。
export const DB_OBJECT_KINDS: { feature: string; kind: string; label: string }[] = [
    { feature: 'view', kind: ObjectKind.View, label: 'db.view' },
    { feature: 'sequence', kind: ObjectKind.Sequence, label: 'db.sequence' },
    { feature: 'procedure', kind: ObjectKind.Procedure, label: 'db.procedure' },
    { feature: 'function', kind: ObjectKind.Function, label: 'db.function' },
    { feature: 'trigger', kind: ObjectKind.Trigger, label: 'db.trigger' },
];

// ---------------------------------- 树节点 params 收窄 ----------------------------------

/**
 * 树框架把节点 params 声明为 `Record<string, unknown>`：其形状随 kind 变化，框架无从知晓。
 * 但 db 模块的节点 params 全部由本模块 contributors.ts 生产，形状是已知的，
 * 故在此**单点**收窄——`as` 只出现在下面两个访问器里，命令、贡献者与 widgets 内部即可享受
 * 完整类型检查，不必像其他资源模块那样在每个调用点重复 `as Record<string, any>`。
 *
 * 入参只要求「带 params 的节点」这一最小结构：容器内的 TreeNode（params 已由水合层兜底非空）
 * 与贡献者产出的 TreeNodeData（params 可选）都能直接传入，故访问器内保留 `?? {}` 兜底。
 *
 * 泛型参数用于叶子节点的更强不变式（如 DbTableKind 必有 tableName），
 * 由调用方按 registerMenu 挂载的 kind 指定，无需在此穷举。
 */

/** 携带 params 的节点（TreeNode / TreeNodeData 的最小公共结构） */
export interface NodeParamsBearer {
    params?: Record<string, unknown>;
}

/** 读取库粒度节点 params（DbKind / DbSchemaKind / DbTableMenuKind / DbSqlMenuKind / DbSqlKind） */
export const dbNodeParams = <T extends DbNodeParams = DbNodeParams>(node: NodeParamsBearer): T => (node.params ?? {}) as T;

/** 读取表粒度节点 params（DbTableKind 及其菜单节点） */
export const dbTableNodeParams = <T extends DbTableNodeParams = DbTableNodeParams>(node: NodeParamsBearer): T => (node.params ?? {}) as T;

/** 读取扩展元数据对象叶子节点 params（DbObjectKind：视图/序列…） */
export const dbObjectNodeParams = <T extends DbObjectNodeParams = DbObjectNodeParams>(node: NodeParamsBearer): T => (node.params ?? {}) as T;

/**
 * 构造表级操作回调载荷（onEditTable / onDeleteTable / onCopyTable / onRenameTable / onGenDdl 的入参）。
 *
 * 注意 DbTableMenuKind 上的「新建表」也复用同一载荷，此时 params 无 tableName，
 * 这正是 DbTableNodeParams.tableName 为可选的原因。
 */
export const toTableCallbackData = (node: TreeNode): TreeNodeCallbackData => ({
    params: dbTableNodeParams(node),
    key: node.key,
    label: node.label,
});

// ---------------------------------- Tab 操作 ----------------------------------

/**
 * DbDataOp 对外暴露的操作接口（defineExpose 的契约）。
 *
 * 资源树的 contributors 通过 getDbOpTabCompInst 拿到组件实例后按本接口调用，
 * 故签名必须与 DbDataOp.vue 中 `defineExpose({...} satisfies DbOpTabApi)` 的实现一致：
 * 参数一律用具体类型而非 any，否则 satisfies 退化为「只查键名不查签名」，
 * 实现端改参数顺序/类型都不会报错，跨组件调用会在运行时才炸。
 */
export interface DbOpTabApi {
    /** 刷新：清空当前所有页签 */
    onRefresh: () => void;
    /** 切换当前库 */
    onChangeDb: (db: DbTreeNodeData, dbName: string) => void;
    /** 打开表数据页签 */
    loadTableData: (db: DbTreeNodeData, dbName: string, tableName: string, readonly?: boolean, title?: string) => Promise<void>;
    /** 打开活动会话页签（该库所在实例的会话列表与取消入口） */
    openSessionTab: (db: DbTreeNodeData, dbName: string) => Promise<void>;
    /** 复制表 */
    onCopyTable: (data: TreeNodeCallbackData) => Promise<void>;
    /** 编辑表结构 */
    onEditTable: (data: TreeNodeCallbackData) => Promise<void>;
    /** 删除表 */
    onDeleteTable: (data: TreeNodeCallbackData) => Promise<void>;
    /** 查看建表 DDL */
    onGenDdl: (data: TreeNodeCallbackData) => Promise<void>;
    /** 查看扩展元数据对象（视图/序列…）DDL */
    onGenObjectDdl: (args: { id: number; db: string; type: string; schema?: string; kind: string; name: string }) => Promise<void>;
    onShowObjectProps: (args: { id: number; db: string; type: string; schema?: string; kind: string; name: string; attrs: Record<string, unknown> }) => void;
    /** 重命名表 */
    onRenameTable: (data: TreeNodeCallbackData) => Promise<void>;
    /** 关闭指定页签 */
    onRemoveTab: (targetName: string) => void;
    /** 新增 SQL 查询页签 */
    addQueryTab: (db: DbTreeNodeData, dbName: string, sqlName?: string) => Promise<void>;
    /** 新增表管理页签 */
    addTablesOpTab: (db: DbTreeNodeData) => Promise<void>;
    /** 重新加载 SQL 模板列表 */
    reloadSqls: (dbId: number, db: string) => void;
    /** 删除 SQL 模板 */
    deleteSql: (dbId: number, db: string, sqlName: string) => Promise<void>;
    /** 局部刷新资源树节点 */
    reloadNode: (nodeKey: string) => void;
}

/**
 * 库工作区 tab 的 key：`实例code.库code.库名.schema`，四段均以 `.` 分隔（与 instCode/dbCode 的分段约定一致）。
 *
 * `params.db` 是连接定位 locator（有 schema 层时为 `库/schema`，供 DbDataOp 建连用，wire 不动）；
 * 这里还原出纯库名并**显式带独立 schema 段**，避免把 schema 折进 `/` 与库名里可能出现的 `/` 混淆
 * （如 mysql 库名 `foo/bar` 的库级 tab，不应被另一个库 `foo` 的库级命令误命中）。
 * 无 schema 层的方言 schema 段为空，key 以 `.` 结尾。
 */
const dbOpTabKey = (params: Record<string, unknown>) => {
    const schema = (params.schema as string) ?? '';
    const locator = params.db as string;
    // 仅当 locator 确实以 `/schema` 结尾时才剥回纯库名，防止无 schema 层方言的库名含 `/` 被误剥
    const dbName = schema && locator.endsWith(`/${schema}`) ? locator.slice(0, -(`/${schema}`.length)) : locator;
    return `${params.instCode}.${params.dbCode}.${dbName}.${schema}`;
};

/**
 * 库级操作（活动会话等）该落进哪个已有工作区。
 *
 * 库节点的 dbOpTabKey 无 schema 段（以 `.` 结尾），而带 schema 层的方言是按 `库.schema` 分支建工作区 tab 的，
 * 按库级 key 查必然查不到，于是给同一个库再挂一个 tab——用户在界面上看到「凭空多出一个操作 tab」。
 * 这里补齐对齐：库级 key（`...db.`）正是其各 schema 分支 key（`...db.public`）的前缀，按 `.` 边界归一，
 * 复用已有工作区（当前激活的分支优先，其次任意分支），一个都没有才由调用方按库级 key 新建。
 * mysql（schema 段为空）库级 key 本身即已存在 → 第一步精确命中，只往已有 tab 里加子 tab。
 */
export const findDbWorkspaceTabKey = (dbTabKey: string): string | undefined => {
    if (getResourceOpTab(dbTabKey)) {
        return dbTabKey;
    }
    // dbTabKey 以 `.` 结尾（schema 段为空），其分支 tab 键形如 `${dbTabKey}<schema>`；以 `.` 为段边界，
    // 天然不误命中同前缀的其他库（`...mydb.` 不会匹配 `...mydb2.public`）
    const branchKeys = [...allResourceOpTabs.keys()].filter((key) => key.startsWith(dbTabKey) && key.length > dbTabKey.length);
    return branchKeys.includes(activeResourceOpTabKey.value) ? activeResourceOpTabKey.value : branchKeys[0];
};

const getDbOpTab = async (params: Record<string, unknown>, nodeKey?: string | number, tabKey = dbOpTabKey(params)) => {
    return await createResourceOpTab({
        key: tabKey,
        name: `${params.name}/${params.db}`,
        nodeKey,
        component: DbDataOp,
        componentProps: {
            dbInfo: {
                id: params.id,
                host: `${params.host}`,
                name: params.name,
                type: params.type,
                tagPath: params.tagPath,
                databases: params.dbs,
                // 透传实例/库 code：DbDataOp 据此重建 sql-menu 等树节点 key（须与 contributors 生成的节点 key 完全一致）
                instCode: params.instCode as string,
                dbCode: params.dbCode as string,
            },
            db: params.db,
        },
        tabComponentProps: {
            icon: { name: getDbDialect(params.type as string)?.getInfo().icon },
        },
    });
};

/**
 * @param reuseDbWorkspace 库级操作入口置 true：同一库已有工作区（含 `库/schema` 分支）时复用之而非再开一个 tab，
 * 判据见 findDbWorkspaceTabKey
 */
export const getDbOpTabCompInst = async (
    params: Record<string, unknown>,
    nodeKey?: string | number,
    reuseDbWorkspace = false
): Promise<DbOpTabApi | undefined> => {
    const tabKey = dbOpTabKey(params);
    const targetKey = reuseDbWorkspace ? (findDbWorkspaceTabKey(tabKey) ?? tabKey) : tabKey;
    return (await getDbOpTab(params, nodeKey, targetKey)).componentInstance as DbOpTabApi | undefined;
};
