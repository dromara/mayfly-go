import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode, ResourceComponentConfig, ResourceConfig } from '../../component/tag';
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { sleep } from '@/common/utils/loading';
import { mongoApi } from '../api';

const Icon = {
    name: ResourceTypeEnum.Mongo.extra.icon,
    color: ResourceTypeEnum.Mongo.extra.iconColor,
};

const MongoList = defineAsyncComponent(() => import('../MongoList.vue'));
const MongoDataOp = defineAsyncComponent(() => import('./MongoDataOp.vue'));

const NodeMongo = defineAsyncComponent(() => import('./NodeMongo.vue'));
const NodeMongoDb = defineAsyncComponent(() => import('./NodeMongoDb.vue'));

export const MongoOpComp: ResourceComponentConfig = {
    name: 'tag.mongoDataOp',
    component: MongoDataOp,
    icon: Icon,
};

// tagpath 节点类型
const NodeTypeMongoTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    parentNode.ctx?.addResourceComponent(MongoOpComp);

    const res = await mongoApi.mongoList.request({ tagPath: parentNode.params.tagPath });
    if (!res.total) {
        return [];
    }

    const mongoInfos = res.list;
    await sleep(100);
    return mongoInfos?.map((x: any) => {
        x.tagPath = parentNode.key;
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeMongo).withParams(x).withNodeComponent(NodeMongo);
    });
});

const NodeTypeMongo = new NodeType(1).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const inst = parentNode.params;
    // 点击mongo -> 加载mongo数据库列表
    const res = await mongoApi.databases.request({ id: inst.id });
    return res.Databases.map((x: any) => {
        const database = x.Name;
        return TagTreeNode.new(parentNode, `${inst.id}.${database}`, database, NodeTypeDbs)
            .withParams({
                id: inst.id,
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
    return [
        TagTreeNode.new(parentNode, `${params.id}.${params.database}.mongo-coll`, 'mongo.coll', NodeTypeCollMenu)
            .withIcon({ name: 'Document' })
            .withParams(params),
    ];
});

const NodeTypeCollMenu = new NodeType(3).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const { id, database } = parentNode.params;
    // 点击数据库集合节点 -> 加载集合列表
    const colls = await mongoApi.collections.request({ id, database });
    return colls.map((x: any) => {
        return TagTreeNode.new(parentNode, `${id}.${database}.${x}`, x, NodeTypeColl)
            .withIsLeaf(true)
            .withParams({
                id,
                database,
                collection: x,
            })
            .withIcon({ name: 'Document' });
    });
});

const NodeTypeColl = new NodeType(4).withNodeClickFunc(async (nodeData: TagTreeNode) => {
    const compRef = await nodeData.ctx?.addResourceComponent(MongoOpComp);
    const { id, database, collection } = nodeData.params;
    compRef.changeCollection(id, database, collection);
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
