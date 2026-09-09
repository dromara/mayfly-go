import { ResourceTypeEnum } from '@/common/commonEnum';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode } from '@/views/ops/resource/tree';
import { dockerApi } from '@/views/ops/docker/api';
import type { Container } from '@/views/ops/docker/types';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';

const ContainerConfList = defineAsyncComponent(() => import('../ContainerConfList.vue'));
const ContainerOp = defineAsyncComponent(() => import('./ContainerOp.vue'));

const Icon = {
    name: ResourceTypeEnum.Container.extra.icon,
    color: ResourceTypeEnum.Container.extra.iconColor,
};

const getContainerOpTab = async (container: Record<string, unknown>, nodeKey?: string | number) => {
    const tabKey = `${container.code}`;
    return await createResourceOpTab({
        key: tabKey,
        nodeKey,
        name: container.name as string,
        component: ContainerOp,
        tabComponentProps: { icon: Icon },
    });
};

/**
 * ContainerOp tab 组件对外方法契约（与 ContainerOp.vue 的 defineExpose 通过 satisfies 双向校验）
 */
export interface ContainerOpTabApi {
    init: (id: any) => void;
}

const getContainerOpTabCompInst = async (container: Record<string, unknown>, nodeKey?: string | number): Promise<ContainerOpTabApi | undefined> => {
    return (await getContainerOpTab(container, nodeKey)).componentInstance as ContainerOpTabApi | undefined;
};

/**
 * docker 资源树节点 kind 常量
 */
export const ContainerKind = 'container';

// 容器节点单击：打开容器操作 tab
registerCommand({
    id: 'container.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const container = ctx.node.params ?? {};
        const compRef = await getContainerOpTabCompInst(container, ctx.node.key);
        compRef?.init?.(container.id);
    },
});

registerMenu({ command: 'container.open', kinds: [ContainerKind], trigger: 'click' });

// 容器节点（叶子）：loadRoots 列出标签下容器；容器本身即操作目标，选择场景可选
registerContributor({
    kind: ContainerKind,
    resourceType: ResourceTypeEnum.Container.value,
    selectable: true,
    icon: Icon,
    loadRoots: async (groupNode: TreeNode) => {
        // 加载标签树下的容器列表
        const res = await dockerApi.page.request({ tagPath: groupNode.params?.tagPath as string });
        return (res?.list ?? [])
            .sort((a: Container, b: Container) => a.name.localeCompare(b.name))
            .map((x: Container) => ({
                key: `${x.code}`,
                kind: ContainerKind,
                label: x.name,
                icon: Icon,
                params: x as unknown as Record<string, unknown>,
            }));
    },
});

export default defineResourceConfig({
    order: 1.5,
    resourceType: ResourceTypeEnum.Container.value,
    manager: {
        componentConf: {
            component: ContainerConfList,
            icon: Icon,
            name: 'tag.container',
        },
        permCode: 'container',
        countKey: 'container',
    },
});
