import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode } from '@/views/ops/resource/tree';
import { mqApi } from '@/views/ops/mq/api';
import type { Kafka } from '@/views/ops/mq/types';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';

export const KafkaIcon = {
    name: ResourceTypeEnum.MqKafka.extra.icon,
    color: ResourceTypeEnum.MqKafka.extra.iconColor,
};

const KafkaList = defineAsyncComponent(() => import('../KafkaList.vue'));
const KafkaOp = defineAsyncComponent(() => import('./KafkaOp.vue'));

const NodeKafka = defineAsyncComponent(() => import('./NodeKafka.vue'));

/**
 * kafka 资源树节点 kind 常量
 */
export const KafkaKind = 'kafka';

const getKafkaOpTab = async (kafka: Record<string, unknown>, nodeKey?: string | number) => {
    const tabKey = `kafka.${kafka.code}`;
    return await createResourceOpTab({
        key: tabKey,
        nodeKey,
        name: kafka.name as string,
        component: KafkaOp,
        tabComponentProps: { icon: KafkaIcon },
    });
};

const getKafkaOpTabCompInst = async (kafka: Record<string, unknown>, nodeKey?: string | number) => {
    return (await getKafkaOpTab(kafka, nodeKey)).componentInstance;
};

// kafka 节点单击：打开 kafka 操作 tab
registerCommand({
    id: 'kafka.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const kafka = ctx.node.params ?? {};
        const compRef = await getKafkaOpTabCompInst(kafka, ctx.node.key);
        compRef?.initKafka?.(kafka);
    },
});

registerMenu({ command: 'kafka.open', kinds: [KafkaKind], trigger: 'click' });

// kafka 节点（叶子）：loadRoots 列出标签下 kafka 集群；集群本身即操作目标，选择场景可选
registerContributor({
    kind: KafkaKind,
    resourceType: TagResourceTypeEnum.MqKafka.value,
    selectable: true,
    renderer: NodeKafka,
    loadRoots: async (groupNode: TreeNode) => {
        const res = await mqApi.kafkaList.request({ tagPath: groupNode.params?.tagPath as string });
        if (!res.total) {
            return [];
        }
        return (res.list ?? []).map((x: Kafka) => ({
            key: `kafka.${x.code}`,
            kind: KafkaKind,
            label: x.name,
            params: x as unknown as Record<string, unknown>,
        }));
    },
});

export default defineResourceConfig({
    order: 6.1,
    resourceType: TagResourceTypeEnum.MqKafka.value,
    manager: {
        componentConf: {
            component: KafkaList,
            icon: KafkaIcon,
            name: 'kafka',
        },
        countKey: 'kafka',
        permCode: 'mq:kafka:base',
    },
});
