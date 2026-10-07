import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode, type TreeNodeData } from '@/views/ops/resource/tree';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';
import { mongoApi } from '../api';

const Icon = {
    name: ResourceTypeEnum.Mongo.extra.icon,
    color: ResourceTypeEnum.Mongo.extra.iconColor,
};

const DbIcon = { name: 'Coin', color: '#67c23a' };
const CollIcon = { name: 'Document' };

const MongoList = defineAsyncComponent(() => import('../MongoList.vue'));
const MongoDataOp = defineAsyncComponent(() => import('./MongoDataOp.vue'));

const NodeMongo = defineAsyncComponent(() => import('./NodeMongo.vue'));
const NodeMongoDb = defineAsyncComponent(() => import('./NodeMongoDb.vue'));

/**
 * mongo 资源树节点 kind 常量
 */
export const MongoKind = 'mongo';
export const MongoDbKind = 'mongo-db';
export const MongoCollMenuKind = 'mongo-coll-menu';
export const MongoCollKind = 'mongo-coll';

/**
 * 打开 mongo 操作 tab 所需的实例定位信息。
 *
 * tab 的键**只有实例 code 一个来源**：树上有「点实例行」与「点集合行」两条入口，
 * 若一条用 code、另一条用 id，同一实例会被挂成两个面板（一个带着已打开的集合、
 * 另一个是空的并被显示出来），用户看到的就是「点了没反应」。
 * 因此 code 是必填，由树节点参数一路带下来。
 */
interface MongoTabTarget {
    code: string;
    name?: string;
    instName?: string;
}

const getMongoOpTab = async (target: MongoTabTarget, nodeKey?: string | number) => {
    return await createResourceOpTab({
        key: target.code,
        nodeKey,
        name: target.instName || target.name || '',
        component: MongoDataOp,
        tabComponentProps: { icon: Icon },
    });
};

/**
 * 从树节点参数取实例定位。
 *
 * 参数由 contributor 构造，形状固定；逐字段收窄而不是整对象断言，
 * 是因为 `Record<string, unknown>` 与必填字段的类型并不重叠，整对象断言要靠 `as unknown as` 才能过。
 */
function tabTargetOf(params: Record<string, unknown>): MongoTabTarget {
    return { code: params.code as string, instName: (params.instName ?? params.name) as string };
}

/** 从树节点参数取集合定位 */
function collectionTargetOf(params: Record<string, unknown>): MongoCollectionTarget {
    return {
        code: params.code as string,
        id: params.id as number,
        instName: params.instName as string,
        database: params.database as string,
        collection: params.collection as string,
        readOnly: params.readOnly as boolean | undefined,
    };
}

/** 集合节点的定位参数（由 contributor 构造，形状固定） */
export interface MongoCollectionTarget {
    code: string;
    id: number;
    instName: string;
    database: string;
    collection: string;
    readOnly?: boolean;
}

export interface MongoOpTabApi {
    changeCollection: (id: number, database: string, collection: string, readOnly?: boolean) => Promise<void>;
    onRefresh: () => void;
}

const getMongoOpTabCompInst = async (target: MongoTabTarget, nodeKey?: string | number): Promise<MongoOpTabApi | undefined> => {
    return (await getMongoOpTab(target, nodeKey)).componentInstance as MongoOpTabApi | undefined;
};

/**
 * 打开某个集合的数据视图（供资源树以外的入口复用）。
 *
 * 实例管理页的库集合弹窗以前自带一套 stats/删除弹窗，与树上的操作视图是两份实现；
 * 归位后那些入口只需调这里，不再各写一遍「打开 tab + 切集合」。
 */
export async function openMongoCollection(target: MongoCollectionTarget) {
    const comp = await getMongoOpTabCompInst({ code: target.code, instName: target.instName });
    await comp?.changeCollection?.(target.id, target.database, target.collection, Boolean(target.readOnly));
}

// 实例节点单击：打开 mongo 操作 tab
registerCommand({
    id: 'mongo.inst.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        await getMongoOpTabCompInst(tabTargetOf(ctx.node.params), ctx.node.key);
    },
});

// 集合节点单击：打开集合并切换
registerCommand({
    id: 'mongo.coll.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const params = collectionTargetOf(ctx.node.params);
        const compRef = await getMongoOpTabCompInst({ code: params.code, instName: params.instName }, ctx.node.key);
        compRef?.changeCollection?.(params.id, params.database, params.collection, Boolean(params.readOnly));
    },
});

registerMenu({ command: 'mongo.inst.open', kinds: [MongoKind], trigger: 'click' });
registerMenu({ command: 'mongo.coll.open', kinds: [MongoCollKind], trigger: 'click' });

// mongo 实例节点：loadRoots 列出标签下实例，loadChildren 展开实例列出数据库
registerContributor({
    kind: MongoKind,
    resourceType: TagResourceTypeEnum.Mongo.value,
    hasChildren: true,
    renderer: NodeMongo,
    locateCode: (node) => node.params.code as string,
    loadRoots: async (groupNode) => {
        const res = await mongoApi.mongoList.request({ tagPath: groupNode.params?.tagPath as string });
        if (!res.total) {
            return [];
        }
        return (res.list ?? []).map((x: any) => ({
            key: `${x.code}`,
            kind: MongoKind,
            label: x.name,
            params: { ...x, tagPath: String(groupNode.key) },
        }));
    },
    loadChildren: async (node) => {
        const inst = node.params as { id: number; name: string; code: string };
        // 点击mongo -> 加载mongo数据库列表
        const databases = await mongoApi.databases.request({ id: inst.id });
        return (databases ?? []).map((database): TreeNodeData => ({
            key: `${node.key}.${database.name}`,
            kind: MongoDbKind,
            label: database.name,
            icon: DbIcon,
            params: {
                // code 一路透传：所有层级的入口最终都要落到同一个操作面板
                code: inst.code,
                id: inst.id,
                instName: inst.name,
                database: database.name,
                size: database.sizeOnDisk,
            },
        }));
    },
});

// 数据库节点：展开列出集合菜单；选择场景为有效选择目标（选库即可操作）
registerContributor({
    kind: MongoDbKind,
    hasChildren: true,
    selectable: true,
    renderer: NodeMongoDb,
    loadChildren: async (node) => [
        {
            key: `${node.key}.mongo-coll`,
            kind: MongoCollMenuKind,
            label: 'mongo.coll',
            icon: CollIcon,
            params: { ...node.params },
        },
    ],
});

// 集合菜单节点：展开加载集合列表（集合数可能上千，折叠释放子树回收内存，再展开重新拉取）
registerContributor({
    kind: MongoCollMenuKind,
    hasChildren: true,
    icon: CollIcon,
    releaseOnCollapse: true,
    loadChildren: async (node) => {
        const { id, code, database, instName } = node.params as { id: number; code: string; database: string; instName: string };
        // 点击数据库集合节点 -> 加载集合列表
        const colls = await mongoApi.collections.request({ id, database });
        return (colls ?? []).map((coll): TreeNodeData => ({
            key: `${node.key}.${coll.name}`,
            kind: MongoCollKind,
            // 视图与集合在树上必须能看出来：视图不可写，只读图标提示比点开报错便宜
            label: coll.type && coll.type !== 'collection' ? `${coll.name} (${coll.type})` : coll.name,
            icon: CollIcon,
            params: {
                code,
                id,
                instName,
                database,
                collection: coll.name,
                // 视图不可写：把判据带到操作视图，写入口就地收敛
                readOnly: coll.readOnly,
            },
        }));
    },
});

// 集合节点（叶子，单击打开集合数据）：选择场景的终级粒度
registerContributor({
    kind: MongoCollKind,
    selectable: true,
    icon: CollIcon,
});

export default defineResourceConfig({
    order: 4,
    resourceType: TagResourceTypeEnum.Mongo.value,
    manager: {
        componentConf: {
            component: MongoList,
            icon: Icon,
            name: 'tag.mongo',
        },
        countKey: 'mongo',
        permCode: 'mongo:manage:base',
    },
});
