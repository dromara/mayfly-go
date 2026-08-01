import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { sleep } from '@/common/utils/loading';
import type { ResourceConfig } from '@/views/ops/resource/resource';
import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode } from '../../component/tag';
import { createResourceOpTab } from '../../resource/resourceOp';
import { redisApi } from '../api';
import type { Redis } from '../types';

export const RedisIcon = {
    name: ResourceTypeEnum.Redis.extra.icon,
    color: ResourceTypeEnum.Redis.extra.iconColor,
};

const RedisList = defineAsyncComponent(() => import('../RedisList.vue'));
const RedisDataOp = defineAsyncComponent(() => import('./RedisDataOp.vue'));

const NodeRedis = defineAsyncComponent(() => import('./NodeRedis.vue'));
const NodeRedisDb = defineAsyncComponent(() => import('./NodeRedisDb.vue'));

// tagpath 节点类型
const NodeTypeRedisTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const res = await redisApi.redisList.request({ tagPath: parentNode.params.tagPath as string });
    if (!res.total) {
        return [];
    }

    const redisInfos = res.list;
    await sleep(100);
    return redisInfos.map((x: Redis & { tagPath?: string }) => {
        x.tagPath = String(parentNode.key);
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeRedis).withParams(x as unknown as Record<string, unknown>).withNodeComponent(NodeRedis);
    });
});

// redis实例节点类型
const NodeTypeRedis = new NodeType(2).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const redisInfo = parentNode.params;

    let dbs: TagTreeNode[] = (redisInfo.db as string).split(',').map((x: string) => {
        return TagTreeNode.new(parentNode, `${parentNode.key}.${x}`, `db${x}`, NodeTypeDb)
            .withIsLeaf(true)
            .withParams({
                tagPath: redisInfo.tagPath,
                id: redisInfo.id,
                redisName: redisInfo.name,
                code: redisInfo.code,
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
    const keyspace = res.Keyspace as unknown as Record<string, string>;
    for (let db in keyspace) {
        for (let d of dbs) {
            if (db == d.params.name) {
                d.params.keys = keyspace[db]?.split(',')[0]?.split('=')[1] || 0;
            }
        }
    }
    // 替换label
    dbs.forEach((e: import('@/views/ops/component/tag').TagTreeNode) => {
        e.label = `${e.params.name}`;
    });
    return dbs;
});

// 库节点类型
const NodeTypeDb = new NodeType(21).withNodeClickFunc(async (node: TagTreeNode) => {
    const params = node.params;

    const key = `${params.code}`;
    const resourceOpTab = await createResourceOpTab({
        key,
        name: `${params.redisName}`,

        component: RedisDataOp,
        componentProps: {
            redisInfo: params,
        },

        tabComponentProps: {
            icon: RedisIcon,
        },
    });
    resourceOpTab.componentInstance?.onDbClick?.(params);
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
