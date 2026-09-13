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
export const EsIndexKind = 'es-index';

const EsIndexIcon = { name: 'Document' };

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

// 索引节点单击：打开/激活 es 面板 tab 并切换到数据管理选中索引
registerCommand({
    id: 'es.index.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const params = ctx.node.params;
        const tabKey = `${params?.code}`;
        const tab = await createResourceOpTab({
            key: tabKey,
            nodeKey: ctx.node.key,
            name: params?.name as string,
            component: EsDashboard,
            componentProps: {
                instId: params?.id as number,
            },
            tabComponentProps: { icon: Icon },
        });
        tab.componentInstance?.onViewIndexData?.(ctx.node.label as string);
    },
});

registerMenu({ command: 'es.inst.open', kinds: [EsInstKind], trigger: 'click' });
registerMenu({ command: 'es.index.open', kinds: [EsIndexKind], trigger: 'click' });

// es 实例节点：loadRoots 列出标签下实例；loadChildren 展开实例列出索引列表
registerContributor({
    kind: EsInstKind,
    resourceType: TagResourceTypeEnum.EsInstance.value,
    hasChildren: true,
    selectable: true,
    renderer: NodeEs,
    locateCode: (node) => node.params.code as string,
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
    loadChildren: async (node: TreeNode) => {
        const instId = node.params?.id as number;
        const res = await esApi.proxyReq('get', instId, '/_cat/indices/?h=index,health,status,docs.count,store.size&format=json');
        const list = (res as unknown as { index: string; health?: string; 'docs.count'?: string }[]) || [];
        return list
            .filter((idx) => !idx.index.startsWith('.'))
            .map((idx) => ({
                key: `${node.key}.${idx.index}`,
                kind: EsIndexKind,
                label: idx.index,
                icon: EsIndexIcon,
                badge: idx['docs.count'] || undefined,
                params: { ...node.params, indexName: idx.index },
            }));
    },
});

// es 索引节点（叶子）
registerContributor({
    kind: EsIndexKind,
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
