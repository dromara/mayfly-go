import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode, ResourceComponentConfig, ResourceConfig } from '../../component/tag';
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { redisApi } from '../api';
import { sleep } from '@/common/utils/loading';

export const RedisIcon = {
    name: ResourceTypeEnum.Redis.extra.icon,
    color: ResourceTypeEnum.Redis.extra.iconColor,
};

const RedisList = defineAsyncComponent(() => import('../RedisList.vue'));
const RedisDataOp = defineAsyncComponent(() => import('./RedisDataOp.vue'));

const NodeRedis = defineAsyncComponent(() => import('./NodeRedis.vue'));
const NodeRedisDb = defineAsyncComponent(() => import('./NodeRedisDb.vue'));

export const RedisOpComp: ResourceComponentConfig = {
    name: 'tag.redisDataOp',
    component: RedisDataOp,
    icon: RedisIcon,
};

// tagpath 节点类型
const NodeTypeRedisTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    parentNode.ctx?.addResourceComponent(RedisOpComp);

    const res = await redisApi.redisList.request({ tagPath: parentNode.params.tagPath });
    if (!res.total) {
        return [];
    }

    const redisInfos = res.list;
    await sleep(100);
    return redisInfos.map((x: any) => {
        x.tagPath = parentNode.key;
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeRedis).withParams(x).withNodeComponent(NodeRedis);
    });
});

// redis实例节点类型
const NodeTypeRedis = new NodeType(2).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const redisInfo = parentNode.params;

    let dbs: TagTreeNode[] = redisInfo.db.split(',').map((x: string) => {
        return TagTreeNode.new(parentNode, `${parentNode.key}.${x}`, `db${x}`, NodeTypeDb)
            .withIsLeaf(true)
            .withParams({
                id: redisInfo.id,
                db: x,
                name: `db${x}`,
                keys: 0,
            })
            .withNodeComponent(NodeRedisDb);
    });

    if (redisInfo.mode == 'cluster') {
        return dbs;
    }

    const res = await redisApi.redisInfo.request({ id: redisInfo.id, host: redisInfo.host, section: 'Keyspace' });
    for (let db in res.Keyspace) {
        for (let d of dbs) {
            if (db == d.params.name) {
                d.params.keys = res.Keyspace[db]?.split(',')[0]?.split('=')[1] || 0;
            }
        }
    }
    // 替换label
    dbs.forEach((e: any) => {
        e.label = `${e.params.name}`;
    });
    return dbs;
});

// 库节点类型
const NodeTypeDb = new NodeType(21).withNodeClickFunc(async (node: TagTreeNode) => {
    (await node.ctx?.addResourceComponent(RedisOpComp)).onDbClick(node.params);
});

export default {
    order: 3,
    resourceType: TagResourceTypeEnum.Redis.value,
    rootNodeType: NodeTypeRedisTag,
    manager: {
        componentConf: {
            component: RedisList,
            icon: RedisIcon,
            name: 'redis',
        },
        countKey: 'redis',
        permCode: 'redis:manage',
    },
} as ResourceConfig;
