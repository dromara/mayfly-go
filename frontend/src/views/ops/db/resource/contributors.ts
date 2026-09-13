/**
 * DB 资源树 - 贡献者注册
 */
import { defineAsyncComponent } from 'vue';
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { formatByteSize } from '@/common/utils/format';
import { registerContributor, type TreeNode, type TreeNodeData } from '@/views/ops/resource/tree';
import { dbApi } from '../api';
import { DbInst } from '../db';
import { getDbDialect, getDialectCapabilities } from '../dialect/index';
import type { DbInstance, Db, DbTableInfo, DbSql, DbNamesParam } from '../types';
import {
    DbInstKind,
    DbDbsKind,
    DbKind,
    DbSchemaKind,
    DbTableMenuKind,
    DbSqlMenuKind,
    DbTableKind,
    DbSqlKind,
    DbIcon,
    SchemaIcon,
    TableIcon,
    dbNodeParams,
    getDbOpTabCompInst,
} from './helpers';

const NodeDbInst = defineAsyncComponent(() => import('./NodeDbInst.vue'));
const NodeDbTable = defineAsyncComponent(() => import('./NodeDbTable.vue'));

const SqlIcon = {
    name: 'icon db/sql',
    color: '#f56c6c',
};

// 数据库实例节点
registerContributor({
    kind: DbInstKind,
    resourceType: ResourceTypeEnum.Db.value,
    hasChildren: true,
    renderer: NodeDbInst,
    locateCode: (node) => node.params.code as string,
    loadRoots: async (groupNode) => {
        const tagPath = groupNode.params?.tagPath as string;
        const dbInstancesRes = await dbApi.instances.request({ tagPath, pageSize: 500 });
        const list: TreeNodeData[] = (dbInstancesRes.list ?? []).map((x: DbInstance) => ({
            key: `${x.code}`,
            kind: DbInstKind,
            label: x.name,
            params: { ...x, tagPath, instCode: x.code },
        }));
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

// 数据库列表名节点
registerContributor({
    kind: DbDbsKind,
    hasChildren: true,
    icon: DbIcon,
    // 对应 codePath 最深段 22|dbCode（TagTypeDb 资源 code=Db.code）；账号段(5|)无对应树层级会被定位解析跳过
    locateCode: (node) => node.params.dbCode as string,
    loadChildren: async (node) => {
        const params = node.params;
        const dbs = (await DbInst.getDbNames(params as unknown as DbNamesParam))?.sort();
        if (!dbs?.length) {
            return [];
        }
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

// 物理 database 节点
registerContributor({
    kind: DbKind,
    hasChildren: true,
    selectable: true,
    icon: DbIcon,
    loadChildren: async (node) => {
        const params = node.params;
        // 是否展开 schema 层级由方言能力自描述，新增方言无需改动本文件
        if (getDialectCapabilities(getDbDialect(params.type as string)).supportsSchema) {
            const { id, db } = params;
            const schemaNames = await dbApi.dbSchemas.request({ id, db });
            return schemaNames.map((sn: string) => ({
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

// postgres schema 节点
registerContributor({
    kind: DbSchemaKind,
    hasChildren: true,
    selectable: true,
    icon: SchemaIcon,
    loadChildren: async (node) => tablesMenuChildren(node),
});

// 数据库表菜单节点
registerContributor({
    kind: DbTableMenuKind,
    hasChildren: true,
    icon: TableIcon,
    releaseOnCollapse: true,
    loadChildren: async (node) => {
        // 表菜单节点是库粒度：params 形状由上面的 DbKind/DbSchemaKind 生产，此处经访问器单点收窄
        const params = dbNodeParams(node);
        const compRef = await getDbOpTabCompInst(params, node.key);
        if (!compRef) {
            return [];
        }
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
        const params = dbNodeParams(node);
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

// 表节点（叶子）
registerContributor({
    kind: DbTableKind,
    renderer: NodeDbTable,
});

// sql 模板节点（叶子）
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
