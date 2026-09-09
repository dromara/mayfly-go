import { defineAsyncComponent } from 'vue';

import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { formatByteSize } from '@/common/utils/format';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode, type TreeNodeData } from '@/views/ops/resource/tree';
import { createResourceOpTab } from '../../resource/resourceOp';
import { dbApi } from '../api';
import { DbInst } from '../db';
import { getDbDialect, schemaDbTypes } from '../dialect/index';
import type { DbInstance, Db, DbTableInfo, DbSql, DbNamesParam } from '../types';

const DbInstList = defineAsyncComponent(() => import('../InstanceList.vue'));
const DbDataOp = defineAsyncComponent(() => import('./DbDataOp.vue'));
const NodeDbInst = defineAsyncComponent(() => import('./NodeDbInst.vue'));
const NodeDb = defineAsyncComponent(() => import('./NodeDb.vue'));
const NodeDbTable = defineAsyncComponent(() => import('./NodeDbTable.vue'));

export const DbIcon = {
    name: ResourceTypeEnum.Db.extra.icon,
    color: ResourceTypeEnum.Db.extra.iconColor,
};

// pgsql schema icon
export const SchemaIcon = {
    name: 'List',
    color: '#67c23a',
};

export const TableIcon = {
    name: 'icon db/table',
    color: '#409eff',
};

const SqlIcon = {
    name: 'icon db/sql',
    color: '#f56c6c',
};

/**
 * db 资源树节点 kind 常量（贡献者协议字符串，跨模块复用如选择树/ai 引用面板）
 */
export const DbInstKind = 'db-inst';
export const DbDbsKind = 'db-dbs';
export const DbKind = 'db';
export const DbSchemaKind = 'db-schema';
export const DbTableMenuKind = 'db-table-menu';
export const DbSqlMenuKind = 'db-sql-menu';
export const DbTableKind = 'db-table';
export const DbSqlKind = 'db-sql';

const getDbOpTab = async (params: Record<string, unknown>, nodeKey?: string | number) => {
    const tabKey = `${params.instCode}.${params.dbCode}.${params.db}`;
    return await createResourceOpTab({
        key: tabKey,
        name: `${params.name}/${params.db}`,
        // 登记触发本次操作的树节点，激活 tab 时用于定位左侧资源树
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

/**
 * DbDataOp tab 组件对外方法契约（与 DbDataOp.vue 的 defineExpose 通过 satisfies 双向校验）：
 * - 方法名/参数个数/返回值受编译期检查，拼写错误不再被索引签名掩盖；
 * - 入参为 resource/index.ts 中动态构造的节点 params（跨模块动态传递，统一 any）；
 * - 结构化入参（如 targetName/nodeKey）保留具体类型。
 */
export interface DbOpTabApi {
    onRefresh: () => void;
    onChangeDb: (db: any, dbName: any) => void;
    loadTables: (dbInfo: any) => Promise<DbTableInfo[] | undefined>;
    loadTableData: (db: any, dbName: any, tableName: any) => Promise<void>;
    onCopyTable: (data: any) => Promise<void>;
    onEditTable: (data: any) => Promise<void>;
    onDeleteTable: (data: any) => Promise<void>;
    onGenDdl: (data: any) => Promise<void>;
    onRenameTable: (data: any) => Promise<void>;
    onRemoveTab: (targetName: string) => void;
    addQueryTab: (db: any, dbName: any, sqlName?: any) => Promise<void>;
    addTablesOpTab: (db: any) => Promise<void>;
    reloadSqls: (dbId: any, db: any) => void;
    deleteSql: (dbId: any, db: any, sqlName: any) => Promise<void>;
    reloadNode: (nodeKey: string) => void;
}

const getDbOpTabCompInst = async (params: Record<string, unknown>, nodeKey?: string | number): Promise<DbOpTabApi | undefined> => {
    return (await getDbOpTab(params, nodeKey)).componentInstance as DbOpTabApi | undefined;
};

/** 命令回调数据：DbDataOp 等组件按 { params, key, label } 形状读取节点信息 */
const toCallbackData = (node: TreeNode) => ({ params: nodeParams(node), key: node.key, label: node.label });

// node.params 为可选字段，命令/加载器统一经此取参（db 节点均携带 params）
const nodeParams = (node: TreeNode): Record<string, unknown> => node.params ?? {};

// ---------------------------------- 命令注册 ----------------------------------

const CmdRefresh = 'db.node.refresh';
registerCommand({
    id: CmdRefresh,
    txt: 'common.refresh',
    icon: 'RefreshRight',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(nodeParams(ctx.node), ctx.node.key))?.reloadNode(ctx.node.key as string),
});

registerCommand({
    id: 'db.table.create',
    txt: 'db.createTable',
    icon: 'Plus',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(nodeParams(ctx.node), ctx.node.key))?.onEditTable(toCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.op',
    txt: 'db.tableOp',
    icon: 'Setting',
    handler: async (ctx: TreeCommandCtx) => {
        const params = nodeParams(ctx.node);
        (await getDbOpTabCompInst(params, ctx.node.key))?.addTablesOpTab({
            id: params.id,
            db: params.db,
            type: params.type,
            nodeKey: ctx.node.key,
        });
    },
});

registerCommand({
    id: 'db.table.copy',
    txt: 'db.copyTable',
    icon: 'copyDocument',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(nodeParams(ctx.node), ctx.node.key))?.onCopyTable(toCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.rename',
    txt: 'db.renameTable',
    icon: 'edit',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(nodeParams(ctx.node), ctx.node.key))?.onRenameTable(toCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.edit',
    txt: 'db.editTable',
    icon: 'edit',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(nodeParams(ctx.node), ctx.node.key))?.onEditTable(toCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.delete',
    txt: 'db.delTable',
    icon: 'Delete',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(nodeParams(ctx.node), ctx.node.key))?.onDeleteTable(toCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.ddl',
    txt: 'DDL',
    icon: 'Document',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(nodeParams(ctx.node), ctx.node.key))?.onGenDdl(toCallbackData(ctx.node)),
});

// 表节点单击：打开表数据 tab
registerCommand({
    id: 'db.table.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const params = nodeParams(ctx.node);
        (await getDbOpTabCompInst(params, ctx.node.key))?.loadTableData({ id: params.id, nodeKey: ctx.node.key }, params.db, params.tableName);
    },
});

// sql 模板节点单击：打开查询 tab
registerCommand({
    id: 'db.sql.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const params = nodeParams(ctx.node);
        (await getDbOpTabCompInst(params, ctx.node.key))?.addQueryTab({ id: params.id, nodeKey: ctx.node.key, dbs: params.dbs }, params.db, params.sqlName);
    },
});

registerCommand({
    id: 'db.sql.delete',
    txt: 'common.delete',
    icon: 'delete',
    handler: async (ctx: TreeCommandCtx) => {
        const params = nodeParams(ctx.node);
        (await getDbOpTabCompInst(params, ctx.node.key))?.deleteSql(params.id, params.db, params.sqlName);
    },
});

// ---------------------------------- 菜单挂载 ----------------------------------

registerMenu({ command: CmdRefresh, kinds: [DbKind, DbSchemaKind, DbTableMenuKind] });
registerMenu({ command: 'db.table.create', kinds: [DbTableMenuKind], order: 2 });
registerMenu({ command: 'db.table.op', kinds: [DbTableMenuKind], order: 3 });

registerMenu({ command: 'db.table.copy', kinds: [DbTableKind], order: 1 });
registerMenu({ command: 'db.table.rename', kinds: [DbTableKind], order: 2 });
registerMenu({ command: 'db.table.edit', kinds: [DbTableKind], order: 3 });
registerMenu({ command: 'db.table.delete', kinds: [DbTableKind], order: 4 });
registerMenu({ command: 'db.table.ddl', kinds: [DbTableKind], order: 5 });
registerMenu({ command: 'db.table.open', kinds: [DbTableKind], trigger: 'click' });

registerMenu({ command: 'db.sql.open', kinds: [DbSqlKind], trigger: 'click' });
registerMenu({ command: 'db.sql.delete', kinds: [DbSqlKind] });

// ---------------------------------- 贡献者注册 ----------------------------------

// 数据库实例节点：loadRoots 供类型分组节点展开（列出标签下实例），loadChildren 展开实例列出库列表
registerContributor({
    kind: DbInstKind,
    resourceType: ResourceTypeEnum.Db.value,
    hasChildren: true,
    renderer: NodeDbInst,
    loadRoots: async (groupNode) => {
        const tagPath = groupNode.params?.tagPath as string;
        const dbInstancesRes = await dbApi.instances.request({ tagPath, pageSize: 500 });
        const list: TreeNodeData[] = (dbInstancesRes.list ?? []).map((x: DbInstance) => ({
            key: `${x.code}`,
            kind: DbInstKind,
            label: x.name,
            params: { ...x, tagPath, instCode: x.code },
        }));
        // 分页截断保护：超出部分静默丢弃等于资产"消失"，追加禁用提示节点显式告知
        if ((dbInstancesRes.total ?? list.length) > list.length) {
            list.push({ key: `${groupNode.key}__truncated`, kind: 'db-truncated', label: 'common.treeTruncated', disabled: true });
        }
        return list;
    },
    loadChildren: async (node) => {
        const params = node.params;
        const tagPath = params.tagPath as string;
        const authCerts = {} as Record<string, Record<string, unknown>>;
        for (const authCert of params.authCerts as Record<string, unknown>[]) {
            authCerts[authCert.name as string] = authCert;
        }

        const dbInfoRes = await dbApi.dbs.request({
            tagPath: `${tagPath}${TagResourceTypeEnum.DbInstance.value}|${params.code}`,
        });
        return (dbInfoRes.list ?? []).map((x: Db) => ({
            key: `${node.key}.${x.code}`,
            kind: DbDbsKind,
            label: x.name,
            icon: DbIcon,
            params: {
                ...x,
                tagPath,
                username: (authCerts[x.authCertName || ''] as Record<string, unknown>)?.username,
                instCode: params.instCode,
                dbCode: x.code,
            },
        }));
    },
});

// 数据库列表名节点：展开列出实例下的物理 database
registerContributor({
    kind: DbDbsKind,
    hasChildren: true,
    icon: DbIcon,
    loadChildren: async (node) => {
        const params = node.params;
        const dbs = (await DbInst.getDbNames(params as unknown as DbNamesParam))?.sort();
        if (!dbs?.length) {
            return [];
        }
        // 查询数据库版本信息
        const version = await dbApi.getCompatibleDbVersion.request({ id: params.id, db: dbs[0] });
        return dbs.map((x: string) => ({
            key: `${node.key}.${x}`,
            kind: DbKind,
            label: x,
            icon: DbIcon,
            params: {
                tagPath: params.tagPath,
                id: params.id,
                code: params.code,
                instCode: params.instCode,
                dbCode: params.dbCode,
                name: params.name,
                type: params.type,
                version: version || 'unset',
                host: `${params.host}:${params.port}`,
                dbs: dbs,
                db: x,
            },
        }));
    },
});

// 物理 database 节点：pg 类库展开多一层 schema，其余直接展开表/SQL 菜单；选择/引用场景为有效选择目标
registerContributor({
    kind: DbKind,
    hasChildren: true,
    selectable: true,
    icon: DbIcon,
    loadChildren: async (node) => {
        const params = node.params;
        // pg类数据库会多一层schema
        if (schemaDbTypes.includes(params.type as string)) {
            const { id, db } = params;
            const schemaNames = await dbApi.pgSchemas.request({ id, db });
            return schemaNames.map((sn: string) => ({
                // 将db变更为 db/schema
                key: `${node.key}/${sn}`,
                kind: DbSchemaKind,
                label: sn,
                icon: SchemaIcon,
                params: { ...params, schema: sn, db: `${db}/${sn}` },
            }));
        }
        return tablesMenuChildren(node);
    },
});

// postgres schema 节点（选择场景的终级粒度）
registerContributor({
    kind: DbSchemaKind,
    hasChildren: true,
    selectable: true,
    icon: SchemaIcon,
    loadChildren: async (node) => tablesMenuChildren(node),
});

// 数据库表菜单节点（折叠释放子树回收内存）
registerContributor({
    kind: DbTableMenuKind,
    hasChildren: true,
    icon: TableIcon,
    releaseOnCollapse: true,
    loadChildren: async (node) => {
        const params = node.params;
        const compRef = await getDbOpTabCompInst(params, node.key);
        if (!compRef) {
            return [];
        }
        // 获取当前库的所有表信息
        const tables = (await compRef.loadTables(params)) ?? [];
        return tables.map((x: DbTableInfo) => {
            const tableSize = x.dataLength + x.indexLength;
            return {
                key: `${node.key}.${x.tableName}`,
                kind: DbTableKind,
                label: x.tableName,
                labelRemark: `${x.tableName} ${x.tableComment ? '| ' + x.tableComment : ''}`,
                icon: TableIcon,
                params: {
                    ...params,
                    tableName: x.tableName,
                    tableComment: x.tableComment,
                    size: tableSize == 0 ? '' : formatByteSize(tableSize, 1),
                },
            };
        });
    },
});

// 数据库 sql 模板菜单节点
registerContributor({
    kind: DbSqlMenuKind,
    hasChildren: true,
    icon: SqlIcon,
    loadChildren: async (node) => {
        const params = node.params;
        // 加载用户保存的sql脚本
        const sqls = await dbApi.getSqlNames.request({ id: params.id, db: params.db });
        return sqls.map((x: DbSql) => ({
            key: `${node.key}.${x.name}`,
            kind: DbSqlKind,
            label: x.name,
            icon: SqlIcon,
            params: { ...params, sqlName: x.name },
        }));
    },
});

// 表节点（叶子，单击打开表数据）
registerContributor({
    kind: DbTableKind,
    renderer: NodeDbTable,
});

// sql 模板节点（叶子，单击打开查询窗口）
registerContributor({
    kind: DbSqlKind,
    icon: SqlIcon,
});

/** 库/schema 节点展开后的表菜单 + SQL 菜单两个子节点 */
const tablesMenuChildren = (node: TreeNode): TreeNodeData[] => {
    const params = { ...node.params, parentKey: node.key };
    return [
        {
            key: `${node.key}.table-menu`,
            kind: DbTableMenuKind,
            label: 'db.table',
            icon: TableIcon,
            params: { ...params, key: `${node.key}.table-menu` },
        },
        {
            key: `${node.key}.sql-menu`,
            kind: DbSqlMenuKind,
            label: 'SQL',
            icon: SqlIcon,
            params: { ...params, key: `${node.key}.sql-menu` },
        },
    ];
};

export default defineResourceConfig({
    order: 2,
    resourceType: ResourceTypeEnum.Db.value,
    manager: {
        componentConf: {
            component: DbInstList,
            icon: DbIcon,
            name: 'tag.db',
        },
        countKey: 'db',
        permCode: 'db:instance',
    },
});
