/**
 * DB 资源树 - 命令与菜单注册
 */
import { registerCommand, registerMenu, type TreeCommandCtx } from '@/views/ops/resource/tree';
import type { DbNodeParams, DbTableNodeParams } from '../types';
import {
    DbKind,
    DbSchemaKind,
    DbTableMenuKind,
    DbTableKind,
    DbSqlKind,
    DbSqlMenuKind,
    dbNodeParams,
    dbTableNodeParams,
    getDbOpTabCompInst,
    toTableCallbackData,
} from './helpers';

const CmdRefresh = 'db.node.refresh';

/**
 * 叶子节点的更强不变式：这些命令经 registerMenu 只挂在对应 kind 上，
 * 而该 kind 的 params 由 contributors.ts 生产时必然带上相应字段。
 * 在此显式声明，避免调用点出现 `!` 或 `?? ''` 之类的兜底掩盖问题。
 */
type TableLeafParams = DbTableNodeParams & { tableName: string };
type SqlLeafParams = DbNodeParams & { sqlName: string };

// ---------------------------------- 命令注册 ----------------------------------

registerCommand({
    id: CmdRefresh,
    txt: 'common.refresh',
    icon: 'RefreshRight',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.reloadNode(ctx.node.key as string),
});

registerCommand({
    id: 'db.table.create',
    txt: 'db.createTable',
    icon: 'Plus',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onEditTable(toTableCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.op',
    txt: 'db.tableOp',
    icon: 'Setting',
    handler: async (ctx: TreeCommandCtx) => {
        const params = dbNodeParams(ctx.node);
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
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onCopyTable(toTableCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.rename',
    txt: 'db.renameTable',
    icon: 'edit',
    handler: async (ctx: TreeCommandCtx) =>
        (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onRenameTable(toTableCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.edit',
    txt: 'db.editTable',
    icon: 'edit',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onEditTable(toTableCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.delete',
    txt: 'db.delTable',
    icon: 'Delete',
    handler: async (ctx: TreeCommandCtx) =>
        (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onDeleteTable(toTableCallbackData(ctx.node)),
});

registerCommand({
    id: 'db.table.ddl',
    txt: 'DDL',
    icon: 'Document',
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onGenDdl(toTableCallbackData(ctx.node)),
});

// 表节点单击：打开表数据 tab
registerCommand({
    id: 'db.table.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const params = dbTableNodeParams<TableLeafParams>(ctx.node);
        (await getDbOpTabCompInst(params, ctx.node.key))?.loadTableData({ id: params.id, nodeKey: ctx.node.key }, params.db, params.tableName);
    },
});

// sql 模板节点单击：打开查询 tab
registerCommand({
    id: 'db.sql.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const params = dbNodeParams<SqlLeafParams>(ctx.node);
        (await getDbOpTabCompInst(params, ctx.node.key))?.addQueryTab({ id: params.id, nodeKey: ctx.node.key, dbs: params.dbs }, params.db, params.sqlName);
    },
});

registerCommand({
    id: 'db.sql.delete',
    txt: 'common.delete',
    icon: 'delete',
    handler: async (ctx: TreeCommandCtx) => {
        const params = dbNodeParams<SqlLeafParams>(ctx.node);
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
