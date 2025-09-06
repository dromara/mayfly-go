import { ResourceTypeEnum } from '@/common/commonEnum';
import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode, ResourceComponentConfig, ResourceConfig } from '@/views/ops/component/tag';
import { dockerApi } from '@/views/ops/docker/api';

const ContainerConfList = defineAsyncComponent(() => import('../ContainerConfList.vue'));
const ContainerOp = defineAsyncComponent(() => import('./ContainerOp.vue'));

const Icon = {
    name: ResourceTypeEnum.Container.extra.icon,
    color: ResourceTypeEnum.Container.extra.iconColor,
};

export const ContainerOpComp: ResourceComponentConfig = {
    name: 'tag.containerOp',
    component: ContainerOp,
    icon: Icon,
};

export const NodeTypeContainerTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (node: TagTreeNode) => {
    // 加载标签树下的容器列表
    const res = await dockerApi.page.request({ tagPath: node.params.tagPath });
    // 把list 根据name字段排序
    return res?.list
        .sort((a: any, b: any) => a.name.localeCompare(b.name))
        .map((x: any) => TagTreeNode.new(node, x.code, x.name, NodeTypeContainer).withIsLeaf(true).withParams(x).withIcon(Icon));
});

const NodeTypeContainer = new NodeType(11).withNodeClickFunc(async (node: TagTreeNode) => {
    (await node.ctx?.addResourceComponent(ContainerOpComp)).init(node.params.id);
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
