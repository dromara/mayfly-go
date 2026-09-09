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

const getMongoOpTab = async (inst: any, nodeKey?: string | number) => {
    const tabKey = `${inst.code}`;
    return await createResourceOpTab({
        key: tabKey,
        nodeKey,
        name: inst.instName || inst.name,
        component: MongoDataOp,
        tabComponentProps: { icon: Icon },
    });
};

/**
 * MongoDataOp tab 组件对外方法契约（与 MongoDataOp.vue 的 defineExpose 通过 satisfies 双向校验）
 */
export interface MongoOpTabApi {
    changeCollection: (id: any, schema: any, collection: any) => Promise<void>;
    onRefresh: () => void;
}

const getMongoOpTabCompInst = async (inst: any, nodeKey?: string | number): Promise<MongoOpTabApi | undefined> => {
    return (await getMongoOpTab(inst, nodeKey)).componentInstance as MongoOpTabApi | undefined;
};

// 实例节点单击：打开 mongo 操作 tab
registerCommand({
    id: 'mongo.inst.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        await getMongoOpTabCompInst(ctx.node.params, ctx.node.key);
    },
});

// 集合节点单击：打开集合并切换
registerCommand({
    id: 'mongo.coll.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const { id, database, collection, instName } = ctx.node.params as Record<string, any>;
        const inst = { id, instName };
        const compRef = await getMongoOpTabCompInst(inst, ctx.node.key);
        compRef?.changeCollection?.(id, database, collection);
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
    loadRoots: async (groupNode) => {
        const res = await mongoApi.mongoList.request({ tagPath: groupNode.params?.tagPath } as any);
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
        const inst = node.params;
        // 点击mongo -> 加载mongo数据库列表
        const res = (await mongoApi.databases.request({ id: inst.id })) as any;
        return res.Databases.map(
            ({ Name: database, SizeOnDisk: size }: { Name: string; SizeOnDisk: number }): TreeNodeData => ({
                key: `${node.key}.${database}`,
                kind: MongoDbKind,
                label: database,
                icon: DbIcon,
                params: {
                    id: inst.id,
                    instName: inst.name,
                    database,
                    size,
                },
            })
        );
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
        const { id, database } = node.params as Record<string, any>;
        // 点击数据库集合节点 -> 加载集合列表
        const colls = (await mongoApi.collections.request({ id, database })) as any;
        return colls.map(
            (x: string): TreeNodeData => ({
                key: `${node.key}.${x}`,
                kind: MongoCollKind,
                label: x,
                icon: CollIcon,
                params: {
                    id,
                    instName: (node.params as Record<string, any>).instName,
                    database,
                    collection: x,
                },
            })
        );
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
