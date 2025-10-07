import { ContextmenuItem } from '@/components/contextmenu';

import { NodeType, TagTreeNode, ResourceConfig } from '../../component/tag';
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { defineAsyncComponent } from 'vue';
import { dbApi } from '../api';
import { sleep } from '@/common/utils/loading';
import { DbInst } from '../db';
import { schemaDbTypes } from '../dialect/index';
import { i18n } from '@/i18n';
import { formatByteSize } from '@/common/utils/format';

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

export const DbDataOpComp = {
    name: 'tag.dbDataOp',
    component: DbDataOp,
    icon: DbIcon,
};

// node节点点击时，触发改变db事件
const nodeClickChangeDb = async (nodeData: TagTreeNode) => {
    const params = nodeData.params;
    if (params.db) {
        const compRef = await nodeData.ctx?.addResourceComponent(DbDataOpComp);
        compRef.onChangeDb(
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

const ContextmenuItemRefresh = new ContextmenuItem('refresh', 'common.refresh')
    .withIcon('RefreshRight')
    .withOnClick(async (node: TagTreeNode) => (await node.ctx?.addResourceComponent(DbDataOpComp)).reloadNode(node.key));

// 数据库实例节点类型
const NodeTypeDbInst = new NodeType(TagResourceTypeEnum.DbInstance.value).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    parentNode.ctx?.addResourceComponent(DbDataOpComp);
    const tagPath = parentNode.params.tagPath;

    const dbInstancesRes = await dbApi.instances.request({ tagPath, pageSize: 100 });
    const dbInstances = dbInstancesRes.list;
    if (!dbInstances) {
        return [];
    }

    // 防止过快加载会出现一闪而过，对眼睛不好
    await sleep(100);
    return dbInstances?.map((x: any) => {
        x.tagPath = tagPath;
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeDbConf).withParams(x).withNodeComponent(NodeDbInst);
    });
});

// 数据库配置节点类型
const NodeTypeDbConf = new NodeType(TagResourceTypeEnum.Db.value)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;

        const tagPath = params.tagPath;
        const authCerts = {} as any;
        for (let authCert of params.authCerts) {
            authCerts[authCert.name] = authCert;
        }

        const dbInfoRes = await dbApi.dbs.request({
            tagPath: `${tagPath}${TagResourceTypeEnum.DbInstance.value}|${params.code}`,
        });
        const dbInfos = dbInfoRes.list;
        if (!dbInfos) {
            return [];
        }

        return dbInfos?.map((x: any) => {
            x.tagPath = tagPath;
            x.username = authCerts[x.authCertName]?.username;
            return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeDbs).withParams(x).withIcon(DbIcon).withNodeComponent(NodeDb);
        });
    })
    .withContextMenuItems([ContextmenuItemRefresh]);

// 数据库列表名类型
const NodeTypeDbs = new NodeType(222).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    const dbs = (await DbInst.getDbNames(params))?.sort();
    // 查询数据库版本信息
    const version = await dbApi.getCompatibleDbVersion.request({ id: params.id, db: dbs[0] });
    return dbs.map((x: any) => {
        return TagTreeNode.new(parentNode, `${parentNode.key}.${x}`, x, NodeTypeDb)
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
const NodeTypeDb = new NodeType(2)
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
                return TagTreeNode.new(parentNode, `${params.id}.${params.db}.schema.${sn}`, sn, NodeTypePostgresSchema)
                    .withParams(nParams)
                    .withIcon(SchemaIcon);
            });
        }

        return getNodeTypeTables(parentNode);
    })
    .withNodeClickFunc(nodeClickChangeDb);

const getNodeTypeTables = (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    let tableKey = `${params.id}.${params.db}.table-menu`;
    let sqlKey = getSqlMenuNodeKey(params.id, params.db);
    return [
        TagTreeNode.new(parentNode, `${params.id}.${params.db}.table-menu`, i18n.global.t('db.table'), NodeTypeTableMenu)
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
const NodeTypePostgresSchema = new NodeType(3)
    .withContextMenuItems([ContextmenuItemRefresh])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        params.parentKey = parentNode.key;
        return getNodeTypeTables(parentNode);
    })
    .withNodeClickFunc(nodeClickChangeDb);

// 数据库表菜单节点
const NodeTypeTableMenu = new NodeType(4)
    .withContextMenuItems([
        ContextmenuItemRefresh,
        new ContextmenuItem('createTable', 'db.createTable').withIcon('Plus').withOnClick(async (parentNode: TagTreeNode) => {
            (await parentNode.ctx?.addResourceComponent(DbDataOpComp))?.onEditTable(parentNode);
        }),
        new ContextmenuItem('tablesOp', 'db.tableOp').withIcon('Setting').withOnClick(async (parentNode: TagTreeNode) => {
            const params = parentNode.params;
            (await parentNode.ctx?.addResourceComponent(DbDataOpComp)).addTablesOpTab({
                id: params.id,
                db: params.db,
                type: params.type,
                nodeKey: parentNode.key,
            });
        }),
    ])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const compRef = await parentNode.ctx?.addResourceComponent(DbDataOpComp);
        const params = parentNode.params;
        // // 获取当前库的所有表信息
        const tables = await compRef.loadTables(params);
        let { id, db, type, schema, version } = params;
        let dbTableSize = 0;
        const tablesNode = tables.map((x: any) => {
            const tableSize = x.dataLength + x.indexLength;
            dbTableSize += tableSize;
            const key = `${id}.${db}.${x.tableName}`;
            return TagTreeNode.new(parentNode, key, x.tableName, NodeTypeTable)
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
                .withNodeComponent(NodeDbTable)
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
const NodeTypeSqlMenu = new NodeType(5)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        const id = params.id;
        const db = params.db;
        const dbs = params.dbs;
        // 加载用户保存的sql脚本
        const sqls = await dbApi.getSqlNames.request({ id: id, db: db });
        return sqls.map((x: any) => {
            return TagTreeNode.new(parentNode, `${id}.${db}.${x.name}`, x.name, NodeTypeSql)
                .withIsLeaf(true)
                .withParams({ id, db, dbs, sqlName: x.name })
                .withIcon(SqlIcon);
        });
    })
    .withNodeClickFunc(nodeClickChangeDb);

// 表节点类型
const NodeTypeTable = new NodeType(6)
    .withContextMenuItems([
        new ContextmenuItem('copyTable', 'db.copyTable')
            .withIcon('copyDocument')
            .withOnClick(async (node: TagTreeNode) => (await node.ctx?.addResourceComponent(DbDataOpComp)).onCopyTable(node)),
        new ContextmenuItem('renameTable', 'db.renameTable')
            .withIcon('edit')
            .withOnClick(async (node: TagTreeNode) => (await node.ctx?.addResourceComponent(DbDataOpComp)).onRenameTable(node)),
        new ContextmenuItem('editTable', 'db.editTable')
            .withIcon('edit')
            .withOnClick(async (node: TagTreeNode) => (await node.ctx?.addResourceComponent(DbDataOpComp)).onEditTable(node)),
        new ContextmenuItem('delTable', 'db.delTable')
            .withIcon('Delete')
            .withOnClick(async (node: TagTreeNode) => (await node.ctx?.addResourceComponent(DbDataOpComp)).onDeleteTable(node)),
        new ContextmenuItem('ddl', 'DDL')
            .withIcon('Document')
            .withOnClick(async (node: TagTreeNode) => (await node.ctx?.addResourceComponent(DbDataOpComp)).onGenDdl(node)),
    ])
    .withNodeClickFunc(async (node: TagTreeNode) => {
        const params = node.params;
        (await node.ctx?.addResourceComponent(DbDataOpComp)).loadTableData({ id: params.id, nodeKey: node.key }, params.db, params.tableName);
    });

// sql模板节点类型
const NodeTypeSql = new NodeType(7)
    .withNodeClickFunc(async (parentNode: TagTreeNode) => {
        const compRef = await parentNode.ctx?.addResourceComponent(DbDataOpComp);
        const params = parentNode.params;
        compRef.addQueryTab({ id: params.id, nodeKey: parentNode.key, dbs: params.dbs }, params.db, params.sqlName);
    })
    .withContextMenuItems([
        new ContextmenuItem('delSql', 'common.delete')
            .withIcon('delete')
            .withOnClick(async (node: TagTreeNode) =>
                (await node.ctx?.addResourceComponent(DbDataOpComp)).deleteSql(node.params.id, node.params.db, node.params.sqlName)
            ),
    ]);

const getSqlMenuNodeKey = (dbId: number, db: string) => {
    return `${dbId}.${db}.sql-menu`;
};

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
