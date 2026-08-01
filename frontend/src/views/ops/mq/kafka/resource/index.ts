import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { sleep } from '@/common/utils/loading';
import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import { mqApi } from '@/views/ops/mq/api';
import type { Kafka } from '@/views/ops/mq/types';
import type { ResourceConfig } from '@/views/ops/resource/resource';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';

export const KafkaIcon = {
    name: ResourceTypeEnum.MqKafka.extra.icon,
    color: ResourceTypeEnum.MqKafka.extra.iconColor,
};

const KafkaList = defineAsyncComponent(() => import('../KafkaList.vue'));
const KafkaOp = defineAsyncComponent(() => import('./KafkaOp.vue'));

const NodeKafka = defineAsyncComponent(() => import('./NodeKafka.vue'));

const getKafkaOpTab = async (kafka: Record<string, unknown>) => {
    const tabKey = `kafka.${kafka.code}`;
    return await createResourceOpTab({
        key: tabKey,
        name: kafka.name as string,
        component: KafkaOp,
        tabComponentProps: { icon: KafkaIcon },
    });
};

const getKafkaOpTabCompInst = async (kafka: Record<string, unknown>) => {
    return (await getKafkaOpTab(kafka)).componentInstance;
};

const NodeTypeKafka = new NodeType(TagResourceTypeEnum.MqKafka.value).withNodeClickFunc(async (node: TagTreeNode) => {
    const kafka = node.params;
    const compRef = await getKafkaOpTabCompInst(kafka);
    compRef?.initKafka?.(kafka);
});

// tagpath 节点类型
const NodeTypeKafkaTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const tagPath = parentNode.params.tagPath;
    const res = await mqApi.kafkaList.request({ tagPath: tagPath as string });
    if (!res.total) {
        return [];
    }
    const kafkaInfos = res.list;
    await sleep(100);
    return kafkaInfos.map((x: Kafka) => {
        return TagTreeNode.new(parentNode, `kafka.${x.code}`, x.name, NodeTypeKafka).withIsLeaf(true).withParams(x as unknown as Record<string, unknown>).withNodeComponent(NodeKafka);
    });
});

export default {
    order: 6.1,
    resourceType: TagResourceTypeEnum.MqKafka.value,
    rootNodeType: NodeTypeKafkaTag,
    manager: {
        componentConf: {
            component: KafkaList,
            icon: KafkaIcon,
            name: 'kafka',
        },
        countKey: 'kafka',
        permCode: 'mq:kafka:base',
    },
} as ResourceConfig;
