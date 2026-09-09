import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { milvusApi, perms } from '@/views/ops/milvus/api';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode, type TreeNodeData } from '@/views/ops/resource/tree';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';
import type { Milvus } from '../types';
import type { MachineAuthCert } from '@/views/ops/machine/types';

export interface MilvusNodeParams {
    id: number;
    code: string;
    name: string;
    host: string;
    selectAuthCert?: MachineAuthCert;
    authCerts?: MachineAuthCert[];
    [key: string]: unknown;
}

export const MilvusIcon = {
    name: ResourceTypeEnum.Milvus.extra.icon,
    color: ResourceTypeEnum.Milvus.extra.iconColor,
};

const AuthCertIcon = { name: 'Ticket', color: '#409eff' };

const MilvusList = defineAsyncComponent(() => import('../MilvusList.vue'));
const MilvusOp = defineAsyncComponent(() => import('./MilvusOp.vue'));

const NodeMilvus = defineAsyncComponent(() => import('./NodeMilvus.vue'));
const NodeMilvusAc = defineAsyncComponent(() => import('./NodeMilvusAc.vue'));

/**
 * milvus 资源树节点 kind 常量
 */
export const MilvusKind = 'milvus';
export const MilvusAcKind = 'milvus-ac';

const getMilvusOpTab = async (milvus: MilvusNodeParams, acName: string, nodeKey?: string | number) => {
    const tabKey = `milvus.${milvus.id}.${acName}`;
    return await createResourceOpTab({
        key: tabKey,
        nodeKey,
        name: milvus.acUsername ? `${milvus.name} (${milvus.acUsername})` : milvus.name,
        component: MilvusOp,
        componentProps: {
            milvusId: milvus.id,
            acName,
            tabKey,
        },
        tabComponentProps: { icon: MilvusIcon },
    });
};

/**
 * MilvusOp tab 组件对外方法契约（与 MilvusOp.vue 的 defineExpose 通过 satisfies 双向校验）
 */
export interface MilvusOpTabApi {
    initMilvus: (milvus: any) => void;
    onActivate: () => void;
}

const getMilvusOpTabCompInst = async (milvus: MilvusNodeParams, acName: string, nodeKey?: string | number): Promise<MilvusOpTabApi | undefined> => {
    return (await getMilvusOpTab(milvus, acName, nodeKey)).componentInstance as MilvusOpTabApi | undefined;
};

// 授权凭证节点单击：打开独立标签页
registerCommand({
    id: 'milvus.ac.open',
    txt: '',
    handler: async (ctx: TreeCommandCtx) => {
        const milvus = (ctx.node.params ?? {}) as unknown as MilvusNodeParams;
        const acName = milvus.selectAuthCert?.name || '';
        // 仅在首次创建时初始化（已存在的标签页只是激活，不重置状态）
        const compRef = await getMilvusOpTabCompInst(milvus, acName, ctx.node.key);
        compRef?.initMilvus?.(milvus);
    },
});

registerMenu({ command: 'milvus.ac.open', kinds: [MilvusAcKind], trigger: 'click' });

// milvus 实例节点：loadRoots 列出标签下实例，loadChildren 展开实例列出授权凭证
registerContributor({
    kind: MilvusKind,
    resourceType: TagResourceTypeEnum.Milvus.value,
    hasChildren: true,
    renderer: NodeMilvus,
    loadRoots: async (groupNode) => {
        const res = await milvusApi.list.request({ tagPath: groupNode.params?.tagPath as string });
        if (!res.total) {
            return [];
        }
        return (res.list ?? []).map((x: Milvus) => ({
            key: `milvus.${x.id}`,
            kind: MilvusKind,
            label: x.name,
            params: x as unknown as Record<string, unknown>,
        }));
    },
    loadChildren: async (node) => {
        const milvus = node.params as unknown as MilvusNodeParams;
        const authCerts = milvus.authCerts || [];
        return authCerts.map(
            (x: MachineAuthCert): TreeNodeData => ({
                key: `milvus.${milvus.id}.${x.name}`,
                kind: MilvusAcKind,
                label: x.username,
                icon: AuthCertIcon,
                params: { ...milvus, selectAuthCert: x },
            })
        );
    },
});

// 授权凭证节点（叶子，单击打开实例操作 tab）：选择场景的终级粒度
registerContributor({
    kind: MilvusAcKind,
    selectable: true,
    renderer: NodeMilvusAc,
});

export default defineResourceConfig({
    order: 7,
    resourceType: TagResourceTypeEnum.Milvus.value,
    manager: {
        componentConf: {
            component: MilvusList,
            icon: MilvusIcon,
            name: 'tag.milvus',
        },
        countKey: 'milvus',
        permCode: perms.base,
    },
});
