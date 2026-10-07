/**
 * DB 资源树 - 贡献者注册
 */
import { defineAsyncComponent } from 'vue';
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { formatByteSize } from '@/common/utils/format';
import { registerContributor, type TreeNode, type TreeNodeData } from '@/views/ops/resource/tree';
import { dbApi } from '../api';
import { DbInst } from '../db';
import type { DbInstance, Db, DbTableInfo, DbSql, DbNamesParam, DbMetadataObject } from '../types';
import {
    DbInstKind,
    DbDbsKind,
    DbKind,
    DbSchemaKind,
    DbTableMenuKind,
    DbSqlMenuKind,
    DbTableKind,
    DbSqlKind,
    DbObjectMenuKind,
    DbObjectKind,
    DbTableSearchKind,
    DbTableResultsKind,
    DbTableLoadMoreKind,
    DB_TABLE_SEARCH_THRESHOLD,
    DB_TABLE_PAGE_SIZE,
    tableResultsKey,
    DB_OBJECT_KINDS,
    ObjectIcon,
    DbIcon,
    SchemaIcon,
    TableIcon,
    SqlIcon,
    dbNodeParams,
    getDbOpTabCompInst,
} from './helpers';
import { getDbCapabilities } from './composables/useCapabilities';

const NodeDbInst = defineAsyncComponent(() => import('./NodeDbInst.vue'));
const NodeDbTable = defineAsyncComponent(() => import('./NodeDbTable.vue'));
const NodeDbTableSearch = defineAsyncComponent(() => import('./NodeDbTableSearch.vue'));

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
                username: authCerts[x.authCertName || '']?.username,
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
        const dbs = (await DbInst.getDbNames(params))?.sort();
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
        // schema 层是否展开、支持哪些扩展对象，均由后端 /capabilities 单一事实源自描述（新增方言/能力前端零改动）
        const caps = await getDbCapabilities(params.id as number, params.db as string);
        if (caps.namespace.HasSchema) {
            const { id, db } = params;
            const schemaNames = await dbApi.dbSchemas.request({ id, db });
            return schemaNames.map((sn: string) => ({
                key: `${node.key}/${sn}`,
                kind: DbSchemaKind,
                label: sn,
                icon: SchemaIcon,
                // 把已协商到的 backendFeatures 向下传递，schema 子节点展开时无需再请求 /capabilities
                params: { ...params, schema: sn, db: `${db}/${sn}`, capsFeatures: caps.backendFeatures },
            }));
        }
        return buildMenuChildren(node, caps.backendFeatures);
    },
});

// postgres schema 节点
registerContributor({
    kind: DbSchemaKind,
    hasChildren: true,
    selectable: true,
    icon: SchemaIcon,
    loadChildren: async (node) => {
        // 优先复用父 db 节点向下传递的 features；缺失（如直接定位到 schema 节点）才回源协商
        const inherited = node.params.capsFeatures as string[] | undefined;
        const features = inherited ?? (await getDbCapabilities(dbNodeParams(node).id, dbNodeParams(node).db)).backendFeatures;
        return buildMenuChildren(node, features);
    },
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
        // 以 limit=阈值+1 做「限量探测」判断表是否过多，不必然全量拉取
        const probe = (await dbApi.tableInfos.request({ id: params.id, db: params.db, limit: DB_TABLE_SEARCH_THRESHOLD + 1 })) ?? [];
        if (probe.length <= DB_TABLE_SEARCH_THRESHOLD) {
            // 少量：直接全量渲染（探测结果即全量），无需搜索框
            return tableLeafNodes(node, params, probe);
        }
        // 超阈值：稳定的搜索框节点 + 并列结果容器节点。输入只刷新结果容器，
        // 搜索框不处于被刷新子树内 → 不重挂载，光标/焦点与输入值得以保留。
        return [tableSearchNode(node, params), tableResultsNode(node, params)];
    },
});

// 表名搜索框节点（仅表过多或处于搜索态时出现）：输入即回写父表菜单节点 params.tableFilter 并刷新之
registerContributor({
    kind: DbTableSearchKind,
    renderer: NodeDbTableSearch,
    hasChildren: false,
    selectable: false,
});

// 搜索结果容器节点：展开时按已提交的 tableFilter（可为空）走服务端 LIKE + 分页下推，首屏固定加载第一页
registerContributor({
    kind: DbTableResultsKind,
    hasChildren: true,
    icon: TableIcon,
    releaseOnCollapse: true,
    loadChildren: async (node) => buildTablePage(node, (node.params.tableFilter as string | undefined) ?? '', 0),
});

// 「加载更多」伪节点：单击由 commands.ts 的 db.table.loadMore 取下一页并增量追加到结果容器尾部
registerContributor({
    kind: DbTableLoadMoreKind,
    hasChildren: false,
    selectable: false,
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
            // parentKey 指向 sql 叶子真正的父（SQL 菜单节点），而非继承来的库节点 key
            params: { ...params, parentKey: node.key, sqlName: x.name },
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

// 扩展对象分类菜单节点（视图/序列/存储过程…）：展开时按 kind 懒加载后端 MetaNavigator 节点
registerContributor({
    kind: DbObjectMenuKind,
    hasChildren: true,
    icon: ObjectIcon,
    loadChildren: async (node) => {
        const p = dbNodeParams(node);
        const objKind = node.params.objKind as string;
        const nodes = await dbApi.metaObjects.request({ id: p.id, db: p.db, kind: objKind, schema: p.schema as string | undefined });
        return (nodes ?? []).map((n: DbMetadataObject) => ({
            key: `${node.key}.${n.name}`,
            kind: DbObjectKind,
            label: n.name,
            labelRemark: n.comment ? `${n.name} | ${n.comment}` : n.name,
            icon: ObjectIcon,
            // parentKey 指向对象叶子真正的父（视图/序列等对象菜单节点），而非继承来的库节点 key
            params: { ...p, parentKey: node.key, objName: n.name, objKind, objAttrs: n.attrs },
        }));
    },
});

// 扩展对象叶子节点（具体视图/序列等）
registerContributor({
    kind: DbObjectKind,
    icon: ObjectIcon,
});

/** 表叶子节点集合（表菜单在搜索态/普通态复用同一映射） */
const tableLeafNodes = (node: TreeNode, params: Record<string, unknown>, tables: DbTableInfo[]): TreeNodeData[] =>
    tables.map((x: DbTableInfo) => {
        const tableSize = x.dataLength + x.indexLength;
        return {
            key: `${node.key}.${x.tableName}`,
            kind: DbTableKind,
            label: x.tableName,
            labelRemark: `${x.tableName} ${x.tableComment ? '| ' + x.tableComment : ''}`,
            icon: TableIcon,
            params: {
                ...params,
                // parentKey 必须指向表叶子的实际树父节点（表菜单 / 搜索结果容器），
                // 而非从上层继承来的库节点 key：否则改表后 reloadNode(parentKey) 会去刷新库节点，
                // 只重建菜单子节点（不发 t-infos），令表菜单节点停在「加载中」永不水合。
                parentKey: node.key,
                tableName: x.tableName,
                tableComment: x.tableComment,
                size: tableSize == 0 ? '' : formatByteSize(tableSize, 1),
            },
        };
    });

/** 表名搜索框节点：携带库粒度 params 与父表菜单 key（渲染器据此定位并列结果节点） */
const tableSearchNode = (node: TreeNode, params: Record<string, unknown>): TreeNodeData => ({
    key: `${node.key}.table-search`,
    kind: DbTableSearchKind,
    label: '',
    params: { ...params, menuKey: node.key },
});

/** 搜索结果容器节点：稳定 key；输入刷新它而非表菜单，保证搜索框不被重挂载 */
const tableResultsNode = (node: TreeNode, params: Record<string, unknown>): TreeNodeData => ({
    key: tableResultsKey(node.key),
    kind: DbTableResultsKind,
    label: 'db.tableSearchResults',
    icon: TableIcon,
    params: { ...params, menuKey: node.key, tableFilter: '' },
});

/** 「加载更多」伪节点：携带结果容器 key + 当前过滤词 + 已累计条数，供续载命令定位父容器与计算 offset */
const tableLoadMoreNode = (resultsNode: TreeNode, params: Record<string, unknown>, like: string, loaded: number): TreeNodeData => ({
    key: `${resultsNode.key}__more`,
    kind: DbTableLoadMoreKind,
    label: 'db.loadMoreTables',
    icon: TableIcon,
    params: { ...params, resultsKey: resultsNode.key, like, loaded },
});

/**
 * 加载一页表节点（首屏/续载/搜索共用同一映射，保证三处产出节点形状一致）：
 * 服务端按 like 过滤 + offset/limit 取页；满页即认为还有下一页，尾部追加「加载更多」；
 * 搜索态零命中给就地提示，浏览态（无 like）不会为空（超阈值即有表）。
 */
export const buildTablePage = async (resultsNode: TreeNode, like: string, offset: number): Promise<TreeNodeData[]> => {
    const params = dbNodeParams(resultsNode);
    const rows = (await dbApi.tableInfos.request({ id: params.id, db: params.db, like, limit: DB_TABLE_PAGE_SIZE, offset })) ?? [];
    if (!rows.length) {
        // 续载到底（offset>0 的空页）：不追加任何节点，仅由调用方 replaceKey 摘掉「加载更多」占位；
        // 只有首屏零命中且处于搜索态才给「未匹配到表」提示，避免在已展示命中结果尾部误插自相矛盾的提示行
        if (offset > 0) {
            return [];
        }
        return like ? [{ key: `${resultsNode.key}__empty`, kind: 'db-truncated', label: 'db.noTableMatch', disabled: true }] : [];
    }
    const nodes = tableLeafNodes(resultsNode, params, rows);
    if (rows.length >= DB_TABLE_PAGE_SIZE) {
        nodes.push(tableLoadMoreNode(resultsNode, params, like, offset + rows.length));
    }
    return nodes;
};

/** 库/schema 展开后子节点：表菜单 + 后端声明支持的扩展对象分类菜单（视图/序列…，能力位驱动显隐）+ SQL 菜单置最后 */
const buildMenuChildren = (node: TreeNode, features: string[]): TreeNodeData[] => {
    const params = { ...node.params, parentKey: node.key };
    const children: TreeNodeData[] = [
        {
            key: `${node.key}.table-menu`,
            kind: DbTableMenuKind,
            label: 'db.table',
            icon: TableIcon,
            params: { ...params, key: `${node.key}.table-menu` },
        },
    ];
    for (const obj of DB_OBJECT_KINDS) {
        if (!features.includes(obj.feature)) continue;
        children.push({
            key: `${node.key}.obj-${obj.kind}`,
            kind: DbObjectMenuKind,
            label: obj.label,
            icon: ObjectIcon,
            params: { ...params, key: `${node.key}.obj-${obj.kind}`, objKind: obj.kind },
        });
    }
    // SQL 菜单置于最后（表/视图/序列等对象之后），与用户浏览优先级一致
    children.push({
        key: `${node.key}.sql-menu`,
        kind: DbSqlMenuKind,
        label: 'SQL',
        icon: SqlIcon,
        params: { ...params, key: `${node.key}.sql-menu` },
    });
    return children;
};
