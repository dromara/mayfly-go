/**
 * DB 资源树 - 共享常量与辅助函数
 * 打破 index.ts ↔ commands.ts/contributors.ts 循环依赖
 */
import { defineAsyncComponent } from 'vue';

import { ResourceTypeEnum } from '@/common/commonEnum';
import { createResourceOpTab } from '../../resource/resourceOp';
import type { TreeNode } from '../../resource/tree';
import { getDbDialect } from '../dialect/index';
import type { DbTableInfo, DbTreeNodeData, TreeNodeCallbackData, DbNodeParams, DbTableNodeParams } from '../types';

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

// ---------------------------------- Kind 常量 ----------------------------------

export const DbInstKind = 'db-inst';
export const DbDbsKind = 'db-dbs';
export const DbKind = 'db';
export const DbSchemaKind = 'db-schema';
export const DbTableMenuKind = 'db-table-menu';
export const DbSqlMenuKind = 'db-sql-menu';
export const DbTableKind = 'db-table';
export const DbSqlKind = 'db-sql';

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
    /**
     * 加载指定库的表清单（未选中实例时提示并返回 undefined）。
     *
     * 入参只声明实现真正消费的字段（实例 id + 库名），而非整个 DbInstInfo：
     * 调用方是资源树的表菜单节点，其 params 由 contributors.ts 以 DbNodeParams 形态传入，
     * 若在此要求完整的 DbInstInfo，会因 DbNodeParams 的索引签名把 host/database 等
     * 未显式声明的可选字段推成 unknown 而报不可赋值。契约表达「最小所需」即可。
     */
    loadTables: (dbInfo: { id: number; db?: string }) => Promise<DbTableInfo[] | undefined>;
    /** 打开表数据页签 */
    loadTableData: (db: DbTreeNodeData, dbName: string, tableName: string) => Promise<void>;
    /** 复制表 */
    onCopyTable: (data: TreeNodeCallbackData) => Promise<void>;
    /** 编辑表结构 */
    onEditTable: (data: TreeNodeCallbackData) => Promise<void>;
    /** 删除表 */
    onDeleteTable: (data: TreeNodeCallbackData) => Promise<void>;
    /** 查看建表 DDL */
    onGenDdl: (data: TreeNodeCallbackData) => Promise<void>;
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

const getDbOpTab = async (params: Record<string, unknown>, nodeKey?: string | number) => {
    const tabKey = `${params.instCode}.${params.dbCode}.${params.db}`;
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
            },
            db: params.db,
        },
        tabComponentProps: {
            icon: { name: getDbDialect(params.type as string)?.getInfo().icon },
        },
    });
};

export const getDbOpTabCompInst = async (params: Record<string, unknown>, nodeKey?: string | number): Promise<DbOpTabApi | undefined> => {
    return (await getDbOpTab(params, nodeKey)).componentInstance as DbOpTabApi | undefined;
};
