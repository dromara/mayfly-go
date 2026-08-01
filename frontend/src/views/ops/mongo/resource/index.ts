import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { sleep } from '@/common/utils/loading';
import type { ResourceConfig } from '@/views/ops/resource/resource';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode } from '../../component/tag';
import { mongoApi } from '../api';
import type { Mongo } from '../types';

const Icon = {
    name: ResourceTypeEnum.Mongo.extra.icon,
    color: ResourceTypeEnum.Mongo.extra.iconColor,
};

const MongoList = defineAsyncComponent(() => import('../MongoList.vue'));
const MongoDataOp = defineAsyncComponent(() => import('./MongoDataOp.vue'));

const NodeMongo = defineAsyncComponent(() => import('./NodeMongo.vue'));
const NodeMongoDb = defineAsyncComponent(() => import('./NodeMongoDb.vue'));

const getMongoOpTab = async (inst: any) => {
    const tabKey = `${inst.code}`;
    return await createResourceOpTab({
        key: tabKey,
        name: inst.instName || inst.name,
        component: MongoDataOp,
        tabComponentProps: { icon: Icon },
    });
};

const getMongoOpTabCompInst = async (inst: any) => {
    return (await getMongoOpTab(inst)).componentInstance;
};

// tagpath 节点类型
const NodeTypeMongoTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const res = await mongoApi.mongoList.request({ tagPath: parentNode.params.tagPath } as any);
    if (!res.total) {
        return [];
    }

    const mongoInfos = res.list;
    await sleep(100);
    return mongoInfos?.map((x: any) => {
        x.tagPath = String(parentNode.key);
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeMongo).withParams(x as unknown as Record<string, unknown>).withNodeComponent(NodeMongo);
    });
});

const NodeTypeMongo = new NodeType(1)
    .withNodeClickFunc(async (node: TagTreeNode) => {
        const inst = node.params;
        await getMongoOpTabCompInst(inst);
    })
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const inst = parentNode.params;
        // 点击mongo -> 加载mongo数据库列表
        const res = (await mongoApi.databases.request({ id: inst.id })) as any;
        return res.Databases.map((x: { Name: string; SizeOnDisk: number }) => {
            const database = x.Name;
            return TagTreeNode.new(parentNode, `${parentNode.key}.${database}`, database, NodeTypeDbs)
                .withParams({
                    id: inst.id,
                    instName: inst.name,
                    database,
                    size: x.SizeOnDisk,
                })
                .withIcon({ name: 'Coin', color: '#67c23a' })
                .withNodeComponent(NodeMongoDb);
        });
    });

const NodeTypeDbs = new NodeType(2).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    // 点击数据库列表 -> 加载数据库下拥有的菜单列表
    return [TagTreeNode.new(parentNode, `${parentNode.key}.mongo-coll`, 'mongo.coll', NodeTypeCollMenu).withIcon({ name: 'Document' }).withParams(params)];
});

const NodeTypeCollMenu = new NodeType(3).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const { id, database, instName } = parentNode.params;
    // 点击数据库集合节点 -> 加载集合列表
    const colls = (await mongoApi.collections.request({ id, database })) as any;
    return colls.map((x: string) => {
        return TagTreeNode.new(parentNode, `${parentNode.key}.${x}`, x, NodeTypeColl)
            .withIsLeaf(true)
            .withParams({
                id,
                instName: instName,
                database,
                collection: x,
            })
            .withIcon({ name: 'Document' });
    });
});

const NodeTypeColl = new NodeType(4).withNodeClickFunc(async (nodeData: TagTreeNode) => {
    const { id, database, collection, instName } = nodeData.params;
    const inst = { id, instName };
    const compRef = await getMongoOpTabCompInst(inst);
    compRef?.changeCollection?.(id, database, collection);
});

export default {
    order: 4,
    resourceType: TagResourceTypeEnum.Mongo.value,
    rootNodeType: NodeTypeMongoTag,
    manager: {
        componentConf: {
            component: MongoList,
            icon: Icon,
            name: 'mongo',
        },
        countKey: 'mongo',
        permCode: 'mongo:manage:base',
    },
} as ResourceConfig;
