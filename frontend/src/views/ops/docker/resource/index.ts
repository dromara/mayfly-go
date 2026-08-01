import { ResourceTypeEnum } from '@/common/commonEnum';
import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import { dockerApi } from '@/views/ops/docker/api';
import type { Container } from '@/views/ops/docker/types';
import type { ResourceConfig } from '@/views/ops/resource/resource';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';

const ContainerConfList = defineAsyncComponent(() => import('../ContainerConfList.vue'));
const ContainerOp = defineAsyncComponent(() => import('./ContainerOp.vue'));

const Icon = {
    name: ResourceTypeEnum.Container.extra.icon,
    color: ResourceTypeEnum.Container.extra.iconColor,
};

const getContainerOpTab = async (container: Record<string, unknown>) => {
    const tabKey = `${container.code}`;
    return await createResourceOpTab({
        key: tabKey,
        name: container.name as string,
        component: ContainerOp,
        tabComponentProps: { icon: Icon },
    });
};

const getContainerOpTabCompInst = async (container: Record<string, unknown>) => {
    return (await getContainerOpTab(container)).componentInstance;
};

export const NodeTypeContainerTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (node: TagTreeNode) => {
    // 加载标签树下的容器列表
    const res = await dockerApi.page.request({ tagPath: node.params.tagPath as string });
    // 把list 根据name字段排序
    return res?.list
        .sort((a: Container, b: Container) => a.name.localeCompare(b.name))
        .map((x: Container) => TagTreeNode.new(node, `${x.code}`, x.name, NodeTypeContainer).withIsLeaf(true).withParams(x as unknown as Record<string, unknown>).withIcon(Icon));
});

const NodeTypeContainer = new NodeType(11).withNodeClickFunc(async (node: TagTreeNode) => {
    const container = node.params;
    const compRef = await getContainerOpTabCompInst(container);
    compRef?.init?.(container.id);
});

export default {
    order: 1.5,
    resourceType: ResourceTypeEnum.Container.value,
    rootNodeType: NodeTypeContainerTag,
    manager: {
        componentConf: {
            component: ContainerConfList,
            icon: Icon,
            name: 'tag.container',
        },
        permCode: 'container',
        countKey: 'container',
    },
} as ResourceConfig;
