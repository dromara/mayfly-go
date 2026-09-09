import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode, type TreeNodeData } from '@/views/ops/resource/tree';
import { defineAsyncComponent } from 'vue';
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

/**
 * redis 资源树节点 kind 常量
 */
export const RedisKind = 'redis';
export const RedisDbKind = 'redis-db';

/**
 * RedisDataOp tab 组件对外方法契约（与 RedisDataOp.vue 的 defineExpose 通过 satisfies 双向校验）
 */
export interface RedisOpTabApi {
    onDbClick: (dbInfo: any) => Promise<void>;
    onRefresh: () => void;
}

// 库节点单击：打开 redis 操作 tab 并切库
registerCommand({
    id: 'redis.db.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const params = ctx.node.params;

        const key = `${params?.code}`;
        const resourceOpTab = await createResourceOpTab({
            key,
            nodeKey: ctx.node.key,
            name: `${params?.redisName}`,

            component: RedisDataOp,
            componentProps: {
                redisInfo: params,
            },

            tabComponentProps: {
                icon: RedisIcon,
            },
        });
        (resourceOpTab.componentInstance as RedisOpTabApi | undefined)?.onDbClick(params);
    },
});

registerMenu({ command: 'redis.db.open', kinds: [RedisDbKind], trigger: 'click' });

// redis 实例节点：loadRoots 列出标签下实例，loadChildren 展开实例列出库（带 key 数角标）
registerContributor({
    kind: RedisKind,
    resourceType: TagResourceTypeEnum.Redis.value,
    hasChildren: true,
    renderer: NodeRedis,
    loadRoots: async (groupNode) => {
        const res = await redisApi.redisList.request({ tagPath: groupNode.params?.tagPath as string });
        if (!res.total) {
            return [];
        }
        return (res.list ?? []).map((x: Redis & { tagPath?: string }) => ({
            key: `${x.code}`,
            kind: RedisKind,
            label: x.name,
            params: { ...x, tagPath: String(groupNode.key) },
        }));
    },
    loadChildren: async (node) => {
        const redisInfo = node.params as Record<string, any>;

        const dbs: TreeNodeData[] = (redisInfo.db as string).split(',').map((x: string) => ({
            key: `${node.key}.${x}`,
            kind: RedisDbKind,
            label: `db${x}`,
            params: {
                tagPath: redisInfo.tagPath,
                id: redisInfo.id,
                redisName: redisInfo.name,
                code: redisInfo.code,
                db: x,
                name: `db${x}`,
                keys: 0,
            },
        }));

        if (redisInfo.mode == 'cluster') {
            return dbs;
        }

        const res = await redisApi.redisInfo.request({ id: redisInfo.id, host: redisInfo.host, section: 'Keyspace' });
        const keyspace = res.Keyspace as unknown as Record<string, string>;
        for (const db in keyspace) {
            for (const d of dbs) {
                if (db == d.params?.name) {
                    d.params!.keys = keyspace[db]?.split(',')[0]?.split('=')[1] || 0;
                }
            }
        }
        return dbs;
    },
});

// 库节点（叶子，单击打开库数据，角标展示 key 数量）：选择场景的终级粒度
registerContributor({
    kind: RedisDbKind,
    selectable: true,
    renderer: NodeRedisDb,
});

export default defineResourceConfig({
    order: 3,
    resourceType: TagResourceTypeEnum.Redis.value,
    manager: {
        componentConf: {
            component: RedisList,
            icon: RedisIcon,
            name: 'tag.redis',
        },
        countKey: 'redis',
        permCode: 'redis:manage',
    },
});
