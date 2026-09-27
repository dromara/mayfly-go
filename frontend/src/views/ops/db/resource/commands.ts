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
    DbObjectKind,
    ObjectKind,
    dbNodeParams,
    dbTableNodeParams,
    dbObjectNodeParams,
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
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.reloadNode(ctx.node.key),
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
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onRenameTable(toTableCallbackData(ctx.node)),
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
    handler: async (ctx: TreeCommandCtx) => (await getDbOpTabCompInst(dbNodeParams(ctx.node), ctx.node.key))?.onDeleteTable(toTableCallbackData(ctx.node)),
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

// 扩展对象：查看 DDL（右键菜单项，复用表 DDL 对话框，经后端 MetaNavigator.NodeDDL）
registerCommand({
    id: 'db.object.ddl',
    txt: 'DDL',
    icon: 'Document',
    handler: async (ctx: TreeCommandCtx) => {
        const o = dbObjectNodeParams(ctx.node);
        (await getDbOpTabCompInst(o, ctx.node.key))?.onGenObjectDdl({ id: o.id, db: o.db, type: o.type, schema: o.schema, kind: o.objKind, name: o.objName });
    },
});

// 视图：可查询，单击像表一样浏览其结果数据（默认看数据，DDL 走右键菜单）
// 视图一般不可更新（尤其含 JOIN/聚合），故以只读网格打开，禁用编辑/新增/删除，避免生成不可执行的 UPDATE/DELETE
registerCommand({
    id: 'db.object.data',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const o = dbObjectNodeParams(ctx.node);
        (await getDbOpTabCompInst(o, ctx.node.key))?.loadTableData({ id: o.id, nodeKey: ctx.node.key }, o.db, o.objName, true);
    },
});

// 序列：无行数据可浏览，单击查看其定义属性面板（数据类型/起始/步长/范围/缓存/当前值/循环），DDL 走右键
registerCommand({
    id: 'db.object.props',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const o = dbObjectNodeParams(ctx.node);
        (await getDbOpTabCompInst(o, ctx.node.key))?.onShowObjectProps({
            id: o.id,
            db: o.db,
            type: o.type,
            schema: o.schema,
            kind: o.objKind,
            name: o.objName,
            attrs: o.objAttrs ?? {},
        });
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
// 扩展对象：视图单击看数据、序列单击看属性；DDL 统一走右键菜单
registerMenu({ command: 'db.object.data', kinds: [DbObjectKind], trigger: 'click', when: ({ node }) => node.params.objKind === ObjectKind.View });
registerMenu({ command: 'db.object.props', kinds: [DbObjectKind], trigger: 'click', when: ({ node }) => node.params.objKind === ObjectKind.Sequence });
registerMenu({ command: 'db.object.ddl', kinds: [DbObjectKind] });
