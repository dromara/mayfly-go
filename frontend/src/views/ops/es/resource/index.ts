import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode } from '@/views/ops/resource/tree';
import { esApi } from '@/views/ops/es/api';
import type { EsInstance } from '../types';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';

const Icon = {
    name: ResourceTypeEnum.Es.extra.icon,
    color: ResourceTypeEnum.Es.extra.iconColor,
};

const EsInstanceList = defineAsyncComponent(() => import('../EsInstanceList.vue'));
const EsDashboard = defineAsyncComponent(() => import('../component/EsDashboard.vue'));

const NodeEs = defineAsyncComponent(() => import('./NodeEs.vue'));

/**
 * es 资源树节点 kind 常量
 */
export const EsInstKind = 'es-inst';

// 实例节点单击：打开 es 面板 tab
registerCommand({
    id: 'es.inst.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const inst = ctx.node.params;
        const tabKey = `${inst?.code}`;
        createResourceOpTab({
            key: tabKey,
            nodeKey: ctx.node.key,
            name: inst?.name as string,
            component: EsDashboard,
            componentProps: {
                instId: inst?.id,
            },
            tabComponentProps: { icon: Icon },
        });
    },
});

registerMenu({ command: 'es.inst.open', kinds: [EsInstKind], trigger: 'click' });

// es 实例节点（叶子）：loadRoots 列出标签下实例；实例本身即操作目标，选择场景可选
registerContributor({
    kind: EsInstKind,
    resourceType: TagResourceTypeEnum.EsInstance.value,
    selectable: true,
    renderer: NodeEs,
    loadRoots: async (groupNode: TreeNode) => {
        // 加载es实例列表
        const res = await esApi.instances.request({ tagPath: groupNode.params?.tagPath as string });
        if (!res.total) {
            return [];
        }
        return (res.list ?? []).map((x: EsInstance & { tagPath?: string }) => ({
            key: `${x.code}`,
            kind: EsInstKind,
            label: x.name,
            params: { ...x, tagPath: String(groupNode.key) },
        }));
    },
});

export default defineResourceConfig({
    order: 5,
    resourceType: TagResourceTypeEnum.EsInstance.value,
    manager: {
        componentConf: {
            component: EsInstanceList,
            icon: Icon,
            name: 'tag.es',
        },
        countKey: 'es',
        permCode: 'es:instance:save',
    },
});
