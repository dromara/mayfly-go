import { ContextmenuItem } from '@/components/contextmenu';

import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { formatByteSize } from '@/common/utils/format';
import { sleep } from '@/common/utils/loading';
import { i18n } from '@/i18n';
import type { ResourceConfig } from '@/views/ops/resource/resource';
import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode } from '../../component/tag';
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

const getDbOpTab = async (params: Record<string, unknown>) => {
    const tabKey = `${params.instCode}.${params.dbCode}.${params.db}`;
    return await createResourceOpTab({
        key: tabKey,
        name: `${params.name}/${params.db}`,
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

const getDbOpTabCompInst = async (params: Record<string, unknown>) => {
    return (await getDbOpTab(params)).componentInstance;
};

const ContextmenuItemRefresh = new ContextmenuItem('refresh', 'common.refresh')
    .withIcon('RefreshRight')
    .withOnClick(async (node: TagTreeNode) => (await getDbOpTabCompInst(node.params))?.reloadNode(node.key));

// 数据库实例节点类型
export const NodeTypeDbInst = new NodeType(TagResourceTypeEnum.DbInstance.value).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const tagPath = parentNode.params.tagPath;

    const dbInstancesRes = await dbApi.instances.request({ tagPath: tagPath as string, pageSize: 100 });
    const dbInstances = dbInstancesRes.list;
    if (!dbInstances) {
        return [];
    }

    // 防止过快加载会出现一闪而过，对眼睛不好
    await sleep(100);
    return dbInstances?.map((x: DbInstance) => {
        const xExt = { ...x, tagPath, instCode: x.code };
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeDbConf).withParams(xExt).withNodeComponent(NodeDbInst);
    });
});

// 数据库配置节点类型
export const NodeTypeDbConf = new NodeType(TagResourceTypeEnum.Db.value)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;

        const tagPath = params.tagPath;
        const authCerts = {} as Record<string, Record<string, unknown>>;
        for (let authCert of (params.authCerts as Record<string, unknown>[])) {
            authCerts[authCert.name as string] = authCert;
        }

        const dbInfoRes = await dbApi.dbs.request({
            tagPath: `${tagPath}${TagResourceTypeEnum.DbInstance.value}|${params.code}`,
        });
        const dbInfos = dbInfoRes.list;
        if (!dbInfos) {
            return [];
        }

        return dbInfos?.map((x: Db) => {
            const xExt: Record<string, unknown> = { ...x, tagPath, username: (authCerts[x.authCertName || ''] as Record<string, unknown>)?.username, instCode: params.instCode, dbCode: x.code };
            return TagTreeNode.new(parentNode, `${parentNode.key}.${x.code}`, x.name, NodeTypeDbs).withParams(xExt).withIcon(DbIcon).withNodeComponent(NodeDb);
        });
    })
    .withContextMenuItems([ContextmenuItemRefresh]);

// 数据库列表名类型
export const NodeTypeDbs = new NodeType(222).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    const dbs = (await DbInst.getDbNames(params as DbNamesParam))?.sort();
    // 查询数据库版本信息
    const version = await dbApi.getCompatibleDbVersion.request({ id: params.id, db: dbs[0] });
    return dbs.map((x: string) => {
        return TagTreeNode.new(parentNode, `${parentNode.key}.${x}`, x, NodeTypeDb)
            .withParams({
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
            })
            .withIcon(DbIcon);
    });
});

// 数据库节点
export const NodeTypeDb = new NodeType(223).withContextMenuItems([ContextmenuItemRefresh]).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    params.parentKey = parentNode.key;
    // pg类数据库会多一层schema
    if (schemaDbTypes.includes(params.type as string)) {
        const { id, db } = params;
        const schemaNames = await dbApi.pgSchemas.request({ id, db });
        return schemaNames.map((sn: string) => {
            // 将db变更为  db/schema;
            const nParams = { ...params };
            nParams.schema = sn;
            nParams.db = nParams.db + '/' + sn;
            // nParams.dbs = schemaNames;
            return TagTreeNode.new(parentNode, `${parentNode.key}/${sn}`, sn, NodeTypePostgresSchema).withParams(nParams).withIcon(SchemaIcon);
        });
    }

    return getNodeTypeTables(parentNode);
});

export const getNodeTypeTables = (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    let tableKey = `${parentNode.key}.table-menu`;
    let sqlKey = `${parentNode.key}.sql-menu`;
    return [
        TagTreeNode.new(parentNode, tableKey, i18n.global.t('db.table'), NodeTypeTableMenu)
            .withParams({
                ...params,
                key: tableKey,
            })
            .withIcon(TableIcon),

        TagTreeNode.new(parentNode, sqlKey, 'SQL', NodeTypeSqlMenu)
            .withParams({ ...params, key: sqlKey })
            .withIcon(SqlIcon),
    ];
};

// postgres schema模式
export const NodeTypePostgresSchema = new NodeType(224).withContextMenuItems([ContextmenuItemRefresh]).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    params.parentKey = parentNode.key;
    return getNodeTypeTables(parentNode);
});

// 数据库表菜单节点
const NodeTypeTableMenu = new NodeType(4)
    .withCollapseRemoveChildren()
    .withContextMenuItems([
        ContextmenuItemRefresh,
        new ContextmenuItem('createTable', 'db.createTable').withIcon('Plus').withOnClick(async (parentNode: TagTreeNode) => {
            (await getDbOpTabCompInst(parentNode.params))?.onEditTable(parentNode);
        }),
        new ContextmenuItem('tablesOp', 'db.tableOp').withIcon('Setting').withOnClick(async (parentNode: TagTreeNode) => {
            const params = parentNode.params;
            (await getDbOpTabCompInst(params))?.addTablesOpTab({
                id: params.id,
                db: params.db,
                type: params.type,
                nodeKey: parentNode.key,
            });
        }),
    ])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        const compRef = await getDbOpTabCompInst(params);
        // // 获取当前库的所有表信息
        const tables = await compRef.loadTables(params);
        let dbTableSize = 0;
        const tablesNode = tables.map((x: DbTableInfo) => {
            const tableSize = x.dataLength + x.indexLength;
            dbTableSize += tableSize;
            const key = `${parentNode.key}.${x.tableName}`;
            return TagTreeNode.new(parentNode, key, x.tableName, NodeTypeTable)
                .withIsLeaf(true)
                .withParams({
                    ...params,
                    key: key,
                    parentKey: parentNode.key,
                    tableName: x.tableName,
                    tableComment: x.tableComment,
                    size: tableSize == 0 ? '' : formatByteSize(tableSize, 1),
                })
                .withIcon(TableIcon)
                .withNodeComponent(NodeDbTable)
                .withLabelRemark(`${x.tableName} ${x.tableComment ? '| ' + x.tableComment : ''}`);
        });
        // 设置父节点参数的表大小
        parentNode.params.dbTableSize = dbTableSize == 0 ? '' : formatByteSize(dbTableSize);
        return tablesNode;
    });

// 数据库sql模板菜单节点
const NodeTypeSqlMenu = new NodeType(225).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    // 加载用户保存的sql脚本
    const sqls = await dbApi.getSqlNames.request({ id: params.id, db: params.db });
    return sqls.map((x: DbSql) => {
        return TagTreeNode.new(parentNode, `${parentNode.key}.${x.name}`, x.name, NodeTypeSql)
            .withIsLeaf(true)
            .withParams({ ...params, sqlName: x.name })
            .withIcon(SqlIcon);
    });
});

// 表节点类型
const NodeTypeTable = new NodeType(226)
    .withContextMenuItems([
        new ContextmenuItem('copyTable', 'db.copyTable')
            .withIcon('copyDocument')
            .withOnClick(async (node: TagTreeNode) => (await getDbOpTabCompInst(node.params))?.onCopyTable(node)),
        new ContextmenuItem('renameTable', 'db.renameTable')
            .withIcon('edit')
            .withOnClick(async (node: TagTreeNode) => (await getDbOpTabCompInst(node.params))?.onRenameTable(node)),
        new ContextmenuItem('editTable', 'db.editTable')
            .withIcon('edit')
            .withOnClick(async (node: TagTreeNode) => (await getDbOpTabCompInst(node.params))?.onEditTable(node)),
        new ContextmenuItem('delTable', 'db.delTable')
            .withIcon('Delete')
            .withOnClick(async (node: TagTreeNode) => (await getDbOpTabCompInst(node.params))?.onDeleteTable(node)),
        new ContextmenuItem('ddl', 'DDL')
            .withIcon('Document')
            .withOnClick(async (node: TagTreeNode) => (await getDbOpTabCompInst(node.params))?.onGenDdl(node)),
    ])
    .withNodeClickFunc(async (node: TagTreeNode) => {
        const params = node.params;
        (await getDbOpTabCompInst(node.params))?.loadTableData({ id: params.id, nodeKey: node.key }, params.db, params.tableName);
    });

// sql模板节点类型
const NodeTypeSql = new NodeType(227)
    .withNodeClickFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        (await getDbOpTabCompInst(params))?.addQueryTab({ id: params.id, nodeKey: parentNode.key, dbs: params.dbs }, params.db, params.sqlName);
    })
    .withContextMenuItems([
        new ContextmenuItem('delSql', 'common.delete')
            .withIcon('delete')
            .withOnClick(async (node: TagTreeNode) => (await getDbOpTabCompInst(node.params))?.deleteSql(node.params.id, node.params.db, node.params.sqlName)),
    ]);

export default {
    order: 2,
    resourceType: ResourceTypeEnum.Db.value,
    rootNodeType: NodeTypeDbInst,
    manager: {
        componentConf: {
            component: DbInstList,
            icon: DbIcon,
            name: 'tag.db',
        },
        countKey: 'db',
        permCode: 'db:instance',
    },
} as ResourceConfig;
